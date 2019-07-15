/*
Copyright 2019 Google LLC.
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

// Package connection manages cached client connections to gRPC servers.
package connection

import (
	"context"
	"errors"
	"sync"

	log "github.com/golang/glog"
	"google.golang.org/grpc"
)

type connection struct {
	id    string
	ref   int
	c     *grpc.ClientConn // Set during dial attempt, before signaling ready.
	err   error            // Set during dial attempt, before signaling ready.
	ready chan struct{}
}

// Manager provides functionality for creating cached client gRPC connections.
type Manager struct {
	opts []grpc.DialOption
	d    Dial

	mu    sync.Mutex
	conns map[string]*connection
}

// NewManager creates a new Manager. The opts arguments are used
// to dial new gRPC targets, with the same semantics as grpc.DialContext.
func NewManager(opts ...grpc.DialOption) (*Manager, error) {
	return NewManagerCustom(grpc.DialContext, opts...)
}

// Dial defines a function to dial the gRPC connection.
type Dial func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)

// NewManagerCustom creates a new Manager. The opts arguments are used
// to dial new gRPC targets, using the provided Dial function.
func NewManagerCustom(d Dial, opts ...grpc.DialOption) (*Manager, error) {
	if d == nil {
		return nil, errors.New("nil Dial provided")
	}
	m := &Manager{}
	m.conns = map[string]*connection{}
	m.opts = opts
	m.d = d
	return m, nil
}

// remove should be called while locking m.
func (m *Manager) remove(addr string) {
	c, ok := m.conns[addr]
	if !ok {
		log.Errorf("Connection %q missing or already removed", addr)
	}
	delete(m.conns, addr)
	if c.c != nil {
		if err := c.c.Close(); err != nil {
			log.Errorf("Error cleaning up connection %q: %v", addr, err)
		}
	}
}

func (m *Manager) dial(ctx context.Context, addr string, c *connection) {
	defer close(c.ready)
	cc, err := m.d(ctx, addr, m.opts...)
	if err != nil {
		log.Infof("Error creating gRPC connection to %q: %v", addr, err)
		m.mu.Lock()
		m.remove(addr)
		c.err = err
		m.mu.Unlock()
		return
	}

	log.Infof("Created gRPC connection to %q", addr)
	c.c = cc
}

func (c *connection) done(m *Manager) func() {
	var once sync.Once
	fn := func() {
		if c == nil {
			log.Error("Attempted to call done on nil connection")
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		c.ref--
		if c.ref <= 0 {
			m.remove(c.id)
		}
	}
	return func() {
		once.Do(fn)
	}
}

func newConnection(addr string) *connection {
	return &connection{
		id:    addr,
		ready: make(chan struct{}),
	}
}

// Connection creates a new grpc.ClientConn to the destination address or
// returns the existing connection, along with a done function.
//
// Usage is registered when a connection is retrieved using Connection. Clients
// should call the returned done function when the returned connection handle is
// unused. Subsequent calls to the same done function have no effect. If an
// error is returned, done has no effect. Connections with no usages will be
// immediately closed and removed from Manager.
//
// If there is already a pending connection attempt for the same addr,
// Connection blocks until that attempt finishes and returns a shared result.
// Note that canceling the context of a pending attempt early would propagate
// an error to blocked callers.
func (m *Manager) Connection(ctx context.Context, addr string) (conn *grpc.ClientConn, done func(), err error) {
	select {
	case <-ctx.Done():
		return nil, func() {}, ctx.Err()
	default:
		m.mu.Lock()
		c, ok := m.conns[addr]
		if !ok {
			c = newConnection(addr)
			m.conns[addr] = c
			go m.dial(ctx, addr, c)
		}
		c.ref++
		m.mu.Unlock()

		<-c.ready
		if c.err != nil {
			return nil, func() {}, c.err
		}
		log.V(2).Infof("Reusing connection %q", addr)
		return c.c, c.done(m), nil
	}
}
