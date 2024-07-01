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
	"google.golang.org/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	tpb "github.com/openconfig/gnmi/proto/target"
)

const (
	// AddrSeparator delimits the chain of addresses used to connect to a target.
	AddrSeparator      = ";"
	metaReceiveTimeout = "receive_timeout"
)

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
	// ReceiveTimeout defines the optional duration to wait for receiving from
	// a target. 0 means timeout is disabled.
	ReceiveTimeout time.Duration
	// Update will be invoked in response to gNMI updates.
	Update func(string, *gpb.Notification)
	// ConnectionManager is used to create gRPC connections.
	ConnectionManager ConnectionManager
	// ConnectError records connecting errors from subscribe connections.
	ConnectError func(string, error)
	// MonitorError handles errors from subscribe connections.
	MonitorError func(string, error)
}

type target struct {
	name           string
	t              *tpb.Target
	sr             *gpb.SubscribeRequest
	cancel         func()
	finished       chan struct{}
	mu             sync.Mutex
	reconnect      func()
	receiveTimeout time.Duration
}

// Manager provides functionality for making gNMI subscriptions to targets and
// handling updates.
type Manager struct {
	connect           func(string)
	connectError      func(string, error)
	monitorError      func(string, error)
	connectionManager ConnectionManager
	cred              CredentialsClient
	reset             func(string)
	sync              func(string)
	testSync          func() // exposed for test synchronization
	timeout           time.Duration
	receiveTimeout    time.Duration
	update            func(string, *gpb.Notification)

	mu      sync.Mutex
	targets map[string]*target
}

// ConnectionManager defines a function to create a *grpc.ClientConn along with
// a done function that will be called when the returned connection handle is
// unused.
type ConnectionManager interface {
	Connection(ctx context.Context, addr, dialer string) (conn *grpc.ClientConn, done func(), err error)
}

// NewManager returns a new target Manager.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.ConnectionManager == nil {
		return nil, errors.New("nil Config.ConnectionManager supplied")
	}
	return &Manager{
		connect:           cfg.Connect,
		connectError:      cfg.ConnectError,
		monitorError:      cfg.MonitorError,
		connectionManager: cfg.ConnectionManager,
		cred:              cfg.Credentials,
		reset:             cfg.Reset,
		sync:              cfg.Sync,
		targets:           make(map[string]*target),
		testSync:          func() {},
		timeout:           cfg.Timeout,
		receiveTimeout:    cfg.ReceiveTimeout,
		update:            cfg.Update,
	}, nil
}

func (m *Manager) handleGNMIUpdate(name string, resp *gpb.SubscribeResponse) error {
	if log.V(2) {
		log.Info(resp)
	}
	if resp.Response == nil {
		return fmt.Errorf("nil Response")
	}
	switch v := resp.Response.(type) {
	case *gpb.SubscribeResponse_Update:
		if m.update != nil {
			m.update(name, v.Update)
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

func uniqueNextHops(addrs []string) map[string]struct{} {
	nhs := map[string]struct{}{}
	for _, addrLine := range addrs {
		nhs[strings.Split(addrLine, AddrSeparator)[0]] = struct{}{}
	}
	return nhs
}

func (m *Manager) createConn(ctx context.Context, name string, t *tpb.Target) (conn *grpc.ClientConn, done func(), err error) {
	nhs := uniqueNextHops(t.GetAddresses())
	if len(nhs) == 0 {
		return nil, func() {}, errors.New("target has no addresses for next hop connection")
	}
	for nh := range nhs {
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
			conn, done, err = m.connectionManager.Connection(connCtx, nh, t.GetDialer())
			if err == nil {
				return
			}
		}
	}
	return
}

func (m *Manager) handleUpdates(ctx context.Context, ta *target, sc gpb.GNMI_SubscribeClient) error {
	defer m.testSync()
	connected := false
	var recvTimer *time.Timer
	if ta.receiveTimeout.Nanoseconds() > 0 {
		recvTimer = time.NewTimer(ta.receiveTimeout)
		recvTimer.Stop()
		go func() {
			select {
			case <-ctx.Done():
			case <-recvTimer.C:
				log.Errorf("Timed out waiting to receive from %q after %v", ta.name, ta.receiveTimeout)
				m.Reconnect(ta.name)
			}
		}()
	}
	for {
		if recvTimer != nil {
			recvTimer.Reset(ta.receiveTimeout)
		}
		resp, err := sc.Recv()
		if recvTimer != nil {
			recvTimer.Stop()
		}
		if err != nil {
			if m.reset != nil {
				m.reset(ta.name)
			}
			return err
		}
		if !connected {
			if m.connect != nil {
				m.connect(ta.name)
			}
			connected = true
			log.Infof("Target %q successfully subscribed", ta.name)
		}
		if err := m.handleGNMIUpdate(ta.name, resp); err != nil {
			log.Errorf("Error processing request %v for target %q: %v", resp, ta.name, err)
		}
		m.testSync()
	}
}

// subscribeClient is exposed for testing.
var subscribeClient = func(ctx context.Context, conn *grpc.ClientConn) (gpb.GNMI_SubscribeClient, error) {
	return gpb.NewGNMIClient(conn).Subscribe(ctx)
}

func (m *Manager) subscribe(ctx context.Context, ta *target, conn *grpc.ClientConn) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Infof("Attempting to open stream to target %q", ta.name)
	sc, err := subscribeClient(ctx, conn)
	if err != nil {
		return fmt.Errorf("error opening stream to target %q: %v", ta.name, err)
	}
	cr := customizeRequest(ta.name, ta.sr)
	log.V(2).Infof("Sending subscription request to target %q: %v", ta.name, cr)
	if err := sc.Send(cr); err != nil {
		return fmt.Errorf("error sending subscription request to target %q: %v", ta.name, err)
	}
	if err = m.handleUpdates(ctx, ta, sc); err != nil {
		return fmt.Errorf("stream failed for target %q: %v", ta.name, err)
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

	e := backoff.NewExponentialBackOff()
	e.MaxElapsedTime = 0 // Retry target connections indefinitely.
	e.InitialInterval = RetryBaseDelay
	e.MaxInterval = RetryMaxDelay
	e.RandomizationFactor = RetryRandomization
	e.Reset()
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
			if err != nil && m.monitorError != nil {
				m.monitorError(ta.name, err)
			}
			if time.Since(t0) > 2*RetryMaxDelay {
				e.Reset()
			}
			delay := e.NextBackOff()
			log.Errorf("Retrying monitoring of %q in %v due to error: %v", ta.name, delay, err)
			timer.Reset(delay)
		}
	}
}

func (m *Manager) monitor(ctx context.Context, ta *target) (err error) {
	defer func() {
		if err != nil && m.connectError != nil {
			m.connectError(ta.name, err)
		}
	}()

	meta, err := gRPCMeta(ctx, ta.name, ta.t, m.cred)
	if err != nil {
		return fmt.Errorf("error creating gRPC metadata for %q: %v", ta.name, err)
	}
	sCtx := metadata.NewOutgoingContext(ctx, meta)
	log.Infof("Attempting to connect %q", ta.name)
	conn, done, err := m.createConn(sCtx, ta.name, ta.t)
	if err != nil {
		return
	}
	defer done()
	return m.subscribe(sCtx, ta, conn)
}

func (m *Manager) targetRecvTimeout(name string, t *tpb.Target) time.Duration {
	if timeout := t.GetMeta()[metaReceiveTimeout]; timeout != "" {
		recvTimeout, err := time.ParseDuration(timeout)
		if err == nil {
			return recvTimeout
		}
		log.Warningf("Wrong receive_timeout %q specified for %q: %v", timeout, name, err)
	}
	return m.receiveTimeout
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
		name:           name,
		t:              t,
		sr:             sr,
		cancel:         cancel,
		finished:       make(chan struct{}),
		receiveTimeout: m.targetRecvTimeout(name, t),
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
