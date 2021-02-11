/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package gnmi implements a gRPC gNMI agent for testing against a
// collector implementation. Each agent will generate a set of Value protocol
// buffer messages that create a queue of updates to be streamed from a
// synthetic device.
package gnmi

import (
	"golang.org/x/net/context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/openconfig/grpctunnel/tunnel"
	tw "github.com/openconfig/gnmi/tunnel"

	tunnelpb "github.com/openconfig/grpctunnel/proto/tunnel"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

// Agent manages a single gNMI agent implementation. Each client that connects
// via Subscribe or Get will receive a stream of updates based on the requested
// path and the provided initial configuration.
type Agent struct {
	gnmipb.UnimplementedGNMIServer
	mu     sync.Mutex
	s      *grpc.Server
	lis    []net.Listener
	target string
	state  fpb.State
	config *fpb.Config
	// cMu protects client.
	cMu    sync.Mutex
	client *Client
}

// New returns an initialized fake agent.
func New(config *fpb.Config, opts []grpc.ServerOption) (*Agent, error) {
	if config == nil {
		return nil, errors.New("config not provided")
	}
	s := grpc.NewServer(opts...)
	reflection.Register(s)

	return NewFromServer(s, config)
}

// NewFromServer returns a new initialized fake agent from provided server.
func NewFromServer(s *grpc.Server, config *fpb.Config) (*Agent, error) {
	a := &Agent{
		s:      s,
		state:  fpb.State_INIT,
		config: config,
		target: config.Target,
	}
	var err error
	if a.config.Port < 0 {
		a.config.Port = 0
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %v", a.config.Port, err)
	}
	a.lis = append(a.lis, lis)

	if config.TunnelAddr != "" {
		targets := map[tunnel.Target]struct{}{tunnel.Target{ID: config.Target, Type: tunnelpb.TargetType_GNMI_GNOI.String()}: struct{}{}}
		lis, err = tw.Listen(context.Background(), config.TunnelAddr, config.TunnelCrt, targets)
		if err != nil {
			return nil, fmt.Errorf("failed to open listener port %d: %v", a.config.Port, err)
		}
		a.lis = append(a.lis, lis)
	}

	gnmipb.RegisterGNMIServer(a.s, a)
	log.V(1).Infof("Created Agent: %s on %s", a.target, a.Address())
	go a.serve()
	return a, nil
}

// serve will start the agent serving and block until closed.
func (a *Agent) serve() error {
	a.mu.Lock()
	a.state = fpb.State_RUNNING
	s := a.s
	lis := a.lis
	a.mu.Unlock()
	if s == nil || lis == nil {
		return fmt.Errorf("Serve() failed: not initialized")
	}

	chErr := make(chan error, len(lis))
	for _, l := range lis {
		go func(l net.Listener) {
			log.Infof("listening: %s", l.Addr())
			chErr <- s.Serve(l)
		}(l)
	}

	for range lis {
		if err := <-chErr; err != nil {
			log.Infof("received error serving: %v", err)
			return err
		}
	}

	return nil
}

// Target returns the target name the agent is faking.
func (a *Agent) Target() string {
	return a.target
}

// Type returns the target type the agent is faking.
func (a *Agent) Type() string {
	return a.config.ClientType.String()
}

// Address returns the port the agent is listening to.
func (a *Agent) Address() string {
	var addr string
	for _, l := range a.lis {
		// Skip tunnel listeners.
		// We assume there is at most one non-tunnel listener.
		if _, ok := l.(*tw.Listener); !ok {
			addr = strings.Replace(l.Addr().String(), "[::]", "localhost", 1)
			break
		}
	}
	return addr
}

// State returns the current state of the agent.
func (a *Agent) State() fpb.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// Close shuts down the agent.
func (a *Agent) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = fpb.State_STOPPED
	if a.s == nil {
		return
	}
	a.s.Stop()
	for _, l := range a.lis {
		l.Close()
	}
	a.s = nil
	a.lis = nil
}

// Subscribe implements the gNMI Subscribe RPC.
func (a *Agent) Subscribe(stream gnmipb.GNMI_SubscribeServer) error {
	c := NewClient(a.config)
	defer c.Close()

	a.cMu.Lock()
	a.client = c
	a.cMu.Unlock()

	return c.Run(stream)
}

// Requests returns the subscribe requests received by the most recently created client.
func (a *Agent) Requests() []*gnmipb.SubscribeRequest {
	a.cMu.Lock()
	defer a.cMu.Unlock()
	return a.client.requests
}
