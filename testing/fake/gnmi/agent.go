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
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/openconfig/gnmi/unimplemented"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

// Agent manages a single gNMI agent implementation. Each client that connects
// via Subscribe or Get will receive a stream of updates based on the requested
// path and the provided initial configuration.
type Agent struct {
	unimplemented.Server
	mu     sync.Mutex
	s      *grpc.Server
	lis    net.Listener
	target string
	state  fpb.State
	config *fpb.Config
	// cMu protects clients.
	cMu     sync.Mutex
	clients map[string]*Client
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
		s:       s,
		state:   fpb.State_INIT,
		config:  config,
		clients: map[string]*Client{},
		target:  config.Target,
	}
	var err error
	if a.config.Port < 0 {
		a.config.Port = 0
	}
	a.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", a.config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %v", a.config.Port, err)
	}
	gnmipb.RegisterGNMIServer(a.s, a)
	log.V(1).Infof("Created Agent: %s on %s", a.target, a.Address())
	go a.Serve()
	return a, nil
}

// Serve will start the agent serving and block until closed.
func (a *Agent) Serve() error {
	a.mu.Lock()
	a.state = fpb.State_RUNNING
	s := a.s
	a.mu.Unlock()
	if s == nil {
		return fmt.Errorf("Serve() failed: not initialized")
	}
	return a.s.Serve(a.lis)
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
	addr := a.lis.Addr().String()
	return strings.Replace(addr, "[::]", "localhost", 1)
}

// Port returns the port the agent is listening to.
func (a *Agent) Port() int32 {
	return a.config.Port
}

// State returns the current state of the agent.
func (a *Agent) State() fpb.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// Close shuts down the agent and closes all clients currently connected to the
// agent.
func (a *Agent) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Clear()
	a.state = fpb.State_STOPPED
	if a.s == nil {
		return
	}
	a.s.Stop()
	a.lis.Close()
	a.s = nil
	a.lis = nil
}

// Clear closes all currently connected clients of the agent.
func (a *Agent) Clear() {
	a.cMu.Lock()
	defer a.cMu.Unlock()
	var wg sync.WaitGroup
	for k, v := range a.clients {
		log.V(1).Infof("Closing client: %s", k)
		wg.Add(1)
		go func(name string, c *Client) {
			c.Close()
			log.V(1).Infof("Client %s closed", name)
			wg.Done()
		}(k, v)
	}
	wg.Wait()
	a.clients = map[string]*Client{}
}

// Subscribe implements the gNMI Subscribe RPC.
func (a *Agent) Subscribe(stream gnmipb.GNMI_SubscribeServer) error {
	c := NewClient(a.config)

	a.cMu.Lock()
	a.clients[c.String()] = c
	a.cMu.Unlock()

	err := c.Run(stream)

	a.cMu.Lock()
	delete(a.clients, c.String())
	a.cMu.Unlock()

	return err
}
