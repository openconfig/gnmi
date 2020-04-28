/*
Copyright 2020 Google Inc.

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

// Package manager provides functions to start or stop monitoring of a target.
package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"github.com/golang/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	tpb "github.com/openconfig/gnmi/proto/target"
)

// AddrSeparator delimits the chain of addresses used to connect to a target.
const AddrSeparator = ";"

var (
	// ErrPending indicates a pending subscription attempt for the target exists.
	ErrPending = errors.New("subscription already pending")

	// RetryBaseDelay is the initial retry interval for target connections.
	RetryBaseDelay = time.Second
	// RetryMaxDelay caps the retry interval for target connections.
	RetryMaxDelay = time.Minute
	// RetryRandomization is the randomization factor applied to the retry
	// interval.
	RetryRandomization = 0.5
)

// CredentialsClient is an interface for credentials lookup.
type CredentialsClient interface {
	Lookup(ctx context.Context, key string) (string, error)
}

// Config is used by Manager to handle updates.
type Config struct {
	// Connect will be invoked when a target connects.
	Connect func(string)
	// Credentials is an optional client used to lookup passwords for targets
	// with a specified username and password ID.
	Credentials CredentialsClient
	// Reset will be invoked when a target disconnects.
	Reset func(string)
	// Sync will be invoked in response to gNMI sync updates.
	Sync func(string)
	// Timeout defines the optional duration to wait for a gRPC dial.
	Timeout time.Duration
	// Update will be invoked in response to gNMI updates.
	Update func(*gpb.Notification)
	// ConnectionManager is used to create gRPC connections.
	ConnectionManager ConnectionManager
}

type target struct {
	name      string
	t         *tpb.Target
	sr        *gpb.SubscribeRequest
	cancel    func()
	finished  chan struct{}
	mu        sync.Mutex
	reconnect func()
}

// Manager provides functionality for making gNMI subscriptions to targets and
// handling updates.
type Manager struct {
	backoff           *backoff.ExponentialBackOff
	connect           func(string)
	connectionManager ConnectionManager
	cred              CredentialsClient
	reset             func(string)
	sync              func(string)
	testSync          func() // exposed for test synchronization
	timeout           time.Duration
	update            func(*gpb.Notification)

	mu      sync.Mutex
	targets map[string]*target
}

// ConnectionManager defines a function to create a *grpc.ClientConn along with
// a done function that will be called when the returned connection handle is
// unused.
type ConnectionManager interface {
	Connection(ctx context.Context, addr string) (conn *grpc.ClientConn, done func(), err error)
}

// NewManager returns a new target Manager.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.ConnectionManager == nil {
		return nil, errors.New("nil Config.ConnectionManager supplied")
	}
	e := backoff.NewExponentialBackOff()
	e.MaxElapsedTime = 0 // Retry target connections indefinitely.
	e.InitialInterval = RetryBaseDelay
	e.MaxInterval = RetryMaxDelay
	e.RandomizationFactor = RetryRandomization
	e.Reset()
	return &Manager{
		backoff:           e,
		connect:           cfg.Connect,
		connectionManager: cfg.ConnectionManager,
		cred:              cfg.Credentials,
		reset:             cfg.Reset,
		sync:              cfg.Sync,
		targets:           make(map[string]*target),
		testSync:          func() {},
		timeout:           cfg.Timeout,
		update:            cfg.Update,
	}, nil
}

func (m *Manager) handleGNMIUpdate(name string, resp *gpb.SubscribeResponse) error {
	log.V(2).Info(resp)
	if resp.Response == nil {
		return fmt.Errorf("nil Response")
	}
	switch v := resp.Response.(type) {
	case *gpb.SubscribeResponse_Update:
		if m.update != nil {
			m.update(v.Update)
		}
	case *gpb.SubscribeResponse_SyncResponse:
		if m.sync != nil {
			m.sync(name)
		}
	case *gpb.SubscribeResponse_Error:
		return fmt.Errorf("received error response: %s", v)
	default:
		return fmt.Errorf("received unknown response %T: %s", v, v)
	}
	return nil
}

func addrChains(addrs []string) [][]string {
	ac := make([][]string, len(addrs))
	for idx, addrLine := range addrs {
		ac[idx] = strings.Split(addrLine, AddrSeparator)
	}
	return ac
}

func (m *Manager) createConn(ctx context.Context, name string, t *tpb.Target) (conn *grpc.ClientConn, done func(), err error) {
	nhs := addrChains(t.GetAddresses())
	if len(nhs) == 0 {
		return nil, func() {}, errors.New("target has no addresses for next hop connection")
	}
	// A single next-hop dial is assumed.
	nh := nhs[0][0]
	select {
	case <-ctx.Done():
		return nil, func() {}, ctx.Err()
	default:
		connCtx := ctx
		if m.timeout > 0 {
			c, cancel := context.WithTimeout(ctx, m.timeout)
			connCtx = c
			defer cancel()
		}
		return m.connectionManager.Connection(connCtx, nh)
	}
}

func (m *Manager) handleUpdates(name string, sc gpb.GNMI_SubscribeClient) error {
	defer m.testSync()
	connected := false
	for {
		resp, err := sc.Recv()
		if err != nil {
			if m.reset != nil {
				m.reset(name)
			}
			return err
		}
		if !connected {
			if m.connect != nil {
				m.connect(name)
			}
			connected = true
		}
		if err := m.handleGNMIUpdate(name, resp); err != nil {
			log.Errorf("Error processing request %v for target %q: %v", resp, name, err)
		}
		m.testSync()
	}
}

// subscribeClient is exposed for testing.
var subscribeClient = func(ctx context.Context, conn *grpc.ClientConn) (gpb.GNMI_SubscribeClient, error) {
	return gpb.NewGNMIClient(conn).Subscribe(ctx)
}

func (m *Manager) subscribe(ctx context.Context, name string, conn *grpc.ClientConn, sr *gpb.SubscribeRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Infof("Attempting to open stream to target %q", name)
	sc, err := subscribeClient(ctx, conn)
	if err != nil {
		return fmt.Errorf("error opening stream to target %q: %v", name, err)
	}
	cr := customizeRequest(name, sr)
	log.V(2).Infof("Sending subscription request to target %q: %v", name, cr)
	if err := sc.Send(cr); err != nil {
		return fmt.Errorf("error sending subscription request to target %q: %v", name, err)
	}
	log.Infof("Target %q successfully subscribed", name)
	if err = m.handleUpdates(name, sc); err != nil {
		return fmt.Errorf("stream failed for target %q: %v", name, err)
	}
	return nil
}

// customizeRequest clones the request template and customizes target prefix.
func customizeRequest(target string, sr *gpb.SubscribeRequest) *gpb.SubscribeRequest {
	cl := proto.Clone(sr).(*gpb.SubscribeRequest)
	if s := cl.GetSubscribe(); s != nil {
		if p := s.GetPrefix(); p != nil {
			p.Target = target
		} else {
			s.Prefix = &gpb.Path{Target: target}
		}
	}
	return cl
}

func (m *Manager) retryMonitor(ctx context.Context, ta *target) {
	log.Infof("Starting monitoring of %q", ta.name)
	timer := time.NewTimer(0)

	defer func() {
		timer.Stop()
		log.Infof("Finished monitoring %q", ta.name)
		close(ta.finished)
		// Ensure the cancelFunc for the subcontext is called.
		m.Reconnect(ta.name)
	}()

	// Create a subcontext that can be independently cancelled to force reconnect.
	sCtx := m.reconnectCtx(ctx, ta)
	for {
		select {
		case <-ctx.Done():
			// Avoid blocking on backoff timer when removing target.
			return
		case <-timer.C:
			select {
			case <-sCtx.Done():
				// Create a new subcontext if the target was forcibly reconnected.
				sCtx = m.reconnectCtx(ctx, ta)
			default:
			}
			t0 := time.Now()
			err := m.monitor(sCtx, ta)
			if time.Since(t0) > 2*RetryMaxDelay {
				m.backoff.Reset()
			}
			delay := m.backoff.NextBackOff()
			log.Errorf("Retrying monitoring of %q in %v due to error: %v", ta.name, delay, err)
			timer.Reset(delay)
		}
	}
}

func (m *Manager) monitor(ctx context.Context, ta *target) error {
	meta, err := gRPCMeta(ctx, ta.name, ta.t, m.cred)
	if err != nil {
		return fmt.Errorf("error creating gRPC metadata for %q: %v", ta.name, err)
	}
	sCtx := metadata.NewOutgoingContext(ctx, meta)
	log.Infof("Attempting to connect %q", ta.name)
	conn, done, err := m.createConn(sCtx, ta.name, ta.t)
	if err != nil {
		return fmt.Errorf("could not connect to %q: %v", ta.name, err)
	}
	defer done()
	return m.subscribe(sCtx, ta.name, conn, ta.sr)
}

// Add adds the target to Manager and starts a streaming subscription that
// continues until Remove is called. Failed subscriptions are retried using
// exponential backoff.
//
// After Add returns successfully, re-adding a target returns an error,
// unless first removed.
func (m *Manager) Add(name string, t *tpb.Target, sr *gpb.SubscribeRequest) error {
	if name == "" {
		return errors.New("empty name")
	}
	if sr == nil {
		return fmt.Errorf("nil subscribe request for target %q", name)
	}

	defer m.mu.Unlock()
	m.mu.Lock()
	if _, ok := m.targets[name]; ok {
		return fmt.Errorf("target %q already added", name)
	}
	if len(t.GetAddresses()) == 0 {
		return fmt.Errorf("no addresses for target %q", name)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ta := &target{
		name:     name,
		t:        t,
		sr:       sr,
		cancel:   cancel,
		finished: make(chan struct{}),
	}
	m.targets[name] = ta
	go m.retryMonitor(ctx, ta)
	return nil
}

// Remove stops the streaming connection to the target. Removing a target not
// added returns an error. Remove does not return until the subscription is
// completed and no further updates will occur.
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.targets[name]
	if !ok {
		return fmt.Errorf("attempt to remove target %q that is not added", name)
	}
	t.cancel()
	<-t.finished
	delete(m.targets, name)
	log.Infof("Target %q removed", name)
	return nil
}

// reconnectCtx creates a cancellable subcontext for a target that can be
// forcibly reconnected by calling Reconnect.
func (m *Manager) reconnectCtx(ctx context.Context, t *target) context.Context {
	sCtx, reconnect := context.WithCancel(ctx)
	t.mu.Lock()
	t.reconnect = reconnect
	t.mu.Unlock()
	return sCtx
}

// Reconnect triggers a reconnection for the specified target.
func (m *Manager) Reconnect(name string) error {
	m.mu.Lock()
	t, ok := m.targets[name]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no such target: %q", name)
	}
	t.mu.Lock()
	if t.reconnect != nil {
		t.reconnect()
		t.reconnect = nil
	}
	t.mu.Unlock()
	return nil
}
