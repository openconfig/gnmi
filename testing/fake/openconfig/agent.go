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

// Package openconfig implements a gRPC OpenConfig agent for testing against a
// collector implementation.  Each agent will generate a set of Value protocol
// buffer messages that create a queue of updates to be streamed from a
// synthetic device.
package openconfig

import (
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
	ocpb "github.com/openconfig/reference/rpc/openconfig"
)

// Agent manages a single OpenConfig agent implementation. Each client that
// connects via Subscribe or Get will receive a stream of updates based on
// the requested path and the provided initial configuration.
type Agent struct {
	mu     sync.Mutex
	s      *grpc.Server
	target string
	state  fpb.State
	config *fpb.Config
	// cMu protects clients.
	cMu     sync.Mutex
	clients map[string]*Client

	lis net.Listener
}

// New returns an initialized fake agent.
func New(config *fpb.Config, opts []grpc.ServerOption) (*Agent, error) {
	if config == nil {
		return nil, fmt.Errorf("config not provided")
	}
	a := &Agent{
		s:       grpc.NewServer(opts...),
		state:   fpb.State_INIT,
		config:  config,
		clients: map[string]*Client{},
		target:  config.Target,
	}
	reflection.Register(a.s)
	if a.config.Port < 0 {
		a.config.Port = 0
	}
	var err error
	if a.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", a.config.Port)); err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %s", a.config.Port, err)
	}
	ocpb.RegisterOpenConfigServer(a.s, a)
	go a.s.Serve(a.lis)
	a.state = fpb.State_RUNNING
	log.V(2).Infof("Created Agent: %s on port %d", a.target, a.config.Port)
	return a, nil
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
func (a *Agent) Port() int64 {
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
	a.s.Stop()
	a.s = nil
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

// Subscribe implements the Openconfig Subscribe RPC.
func (a *Agent) Subscribe(stream ocpb.OpenConfig_SubscribeServer) error {
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

// Get implements the Openconfig Get RPC.
func (a *Agent) Get(context.Context, *ocpb.GetRequest) (*ocpb.GetResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "Get() is not implemented for gRPC/OpenConfig fakes")
}

// Set implements the Openconfig Set RPC.
func (a *Agent) Set(context.Context, *ocpb.SetRequest) (*ocpb.SetResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "Set() is not implemented for gRPC/OpenConfig fakes")
}

// GetModels implements the Openconfig GetModels RPC.
func (a *Agent) GetModels(context.Context, *ocpb.GetModelsRequest) (*ocpb.GetModelsResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "GetModels() is not implemented for gRPC/OpenConfig fakes")
}
