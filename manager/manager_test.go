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

package manager

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/golang/glog"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/connection"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	tpb "github.com/openconfig/gnmi/proto/target"
)

var (
	validSubscribeRequest = &gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			&gpb.SubscriptionList{
				Subscription: []*gpb.Subscription{
					{
						Path: &gpb.Path{
							Elem: []*gpb.PathElem{{Name: "path1"}},
						},
					},
				},
			},
		},
	}
)

type fakeCreds struct {
	failNext bool
}

func (f *fakeCreds) stageErr() {
	f.failNext = true
}

func (f *fakeCreds) Lookup(ctx context.Context, key string) (string, error) {
	if f.failNext {
		f.failNext = false
		return "", errors.New("error occurred")
	}
	return fmt.Sprintf("pass_%s", key), nil
}

type fakeServer struct{}

func (f *fakeServer) Capabilities(context.Context, *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	return nil, nil
}

func (f *fakeServer) Get(context.Context, *gpb.GetRequest) (*gpb.GetResponse, error) {
	return nil, nil
}

func (f *fakeServer) Set(context.Context, *gpb.SetRequest) (*gpb.SetResponse, error) {
	return nil, nil
}

func (f *fakeServer) Subscribe(gpb.GNMI_SubscribeServer) error {
	return nil
}

func newDevice(t *testing.T, s gpb.GNMIServer) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	if srv == nil {
		t.Fatalf("could not create test server")
	}
	gpb.RegisterGNMIServer(srv, s)
	go srv.Serve(lis)

	return lis.Addr().String(), srv.Stop
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		desc    string
		c       Config
		wantErr bool
	}{
		{
			desc: "missing creds",
			c:    Config{ConnectionManager: newFakeConnection(t)},
		}, {
			desc:    "missing connection manager",
			c:       Config{Credentials: &fakeCreds{}},
			wantErr: true,
		}, {
			desc: "valid",
			c:    Config{Credentials: &fakeCreds{}, ConnectionManager: newFakeConnection(t)},
		},
	}

	for _, tt := range tests {
		_, err := NewManager(tt.c)
		switch {
		case err == nil && !tt.wantErr:
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error.", tt.desc)
		}
	}
}

func TestRemoveErr(t *testing.T) {
	m, err := NewManager(Config{Credentials: &fakeCreds{}, ConnectionManager: newFakeConnection(t)})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}

	if err := m.Remove("not added"); err == nil {
		t.Fatalf("got no error removing target, want error")
	}
}

func assertTargets(t *testing.T, m *Manager, want int) {
	if l := len(m.targets); l != want {
		t.Fatalf("got %v targets, want %v", l, want)
	}
}

func TestAddRemove(t *testing.T) {
	addr, stop := newDevice(t, &fakeServer{})
	defer stop()
	m, err := NewManager(Config{
		Credentials:       &fakeCreds{},
		ConnectionManager: newFakeConnection(t),
		Timeout:           time.Minute,
	})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}
	name := "device1"

	err = m.Add(name, &tpb.Target{Addresses: []string{addr}}, validSubscribeRequest)
	if err != nil {
		t.Fatalf("got error adding: %v, want no error", err)
	}
	assertTargets(t, m, 1)
	if err := m.Remove(name); err != nil {
		t.Fatalf("got error removing: %v, want no error", err)
	}
	assertTargets(t, m, 0)
}

func TestAdd(t *testing.T) {
	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	tests := []struct {
		desc    string
		name    string
		t       *tpb.Target
		sr      *gpb.SubscribeRequest
		wantErr bool
	}{
		{
			desc: "valid",
			name: "device1",
			t:    &tpb.Target{Addresses: []string{addr}},
			sr:   validSubscribeRequest,
		}, {
			desc:    "missing address",
			name:    "device1",
			t:       &tpb.Target{},
			sr:      validSubscribeRequest,
			wantErr: true,
		}, {
			desc:    "missing name",
			t:       &tpb.Target{Addresses: []string{addr}},
			sr:      validSubscribeRequest,
			wantErr: true,
		}, {
			desc:    "nil request",
			name:    "device1",
			t:       &tpb.Target{Addresses: []string{addr}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		m, err := NewManager(Config{Credentials: &fakeCreds{}, ConnectionManager: newFakeConnection(t)})
		if err != nil {
			t.Fatal("could not initialize Manager")
		}
		err = m.Add(tt.name, tt.t, tt.sr)
		if err == nil {
			m.Remove(tt.name)
		}
		switch {
		case err == nil && !tt.wantErr:
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error.", tt.desc)
		}
	}
}

func waitForAdd(m *Manager, name string) {
	m.mu.Lock()
	ta := m.targets[name]
	m.mu.Unlock()
	if ta == nil {
		return
	}
	for {
		if func() bool {
			ta.mu.Lock()
			defer ta.mu.Unlock()
			return ta.reconnect != nil
		}() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestReconnect(t *testing.T) {
	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	m, err := NewManager(Config{Credentials: &fakeCreds{}, ConnectionManager: newFakeConnection(t)})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}
	if err := m.Add("target", &tpb.Target{Addresses: []string{addr}}, validSubscribeRequest); err != nil {
		t.Fatal("could not add target")
	}
	defer m.Remove("target")
	for _, tt := range []struct {
		desc string
		name string
		err  bool
	}{
		{desc: "no such target", name: "badname", err: true},
		{desc: "reconnect once", name: "target"},
		{desc: "reconnect twice", name: "target"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			waitForAdd(m, tt.name)
			err := m.Reconnect(tt.name)
			switch {
			case err != nil && !tt.err:
				t.Errorf("got error %v, want nil", err)
			case err == nil && tt.err:
				t.Error("got nil, want error")
			}
		})
	}
}

func TestConcurrentAdd(t *testing.T) {
	addr, stop := newDevice(t, &fakeServer{})
	defer stop()
	m, err := NewManager(Config{Credentials: &fakeCreds{}, ConnectionManager: newFakeConnection(t)})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}
	name := "device1"
	lim := 10
	var wg sync.WaitGroup
	wg.Add(lim)
	var errs uint32
	for i := 0; i < lim; i++ {
		go func() {
			if err := m.Add(name, &tpb.Target{Addresses: []string{addr}}, validSubscribeRequest); err != nil {
				atomic.AddUint32(&errs, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	// Only the first Add succeeds.
	if errsFinal := atomic.LoadUint32(&errs); errsFinal != uint32(lim-1) {
		t.Errorf("got %v errors, want %v", errsFinal, lim-1)
	}
	assertTargets(t, m, 1)
	m.Remove(name)
}

// fakeSubscribeClient implements GNMI_SubscribeClient.
type fakeSubscribeClient struct {
	ch                chan *gpb.SubscribeResponse
	errCh             chan error
	failNextSend      bool
	grpc.ClientStream // Unused, satisfies interface.
}

func (f *fakeSubscribeClient) stageSendErr() {
	f.failNextSend = true
}

func newFakeSubscribeClient() *fakeSubscribeClient {
	f := &fakeSubscribeClient{
		ch:    make(chan *gpb.SubscribeResponse),
		errCh: make(chan error),
	}
	return f
}

func (f *fakeSubscribeClient) Send(_ *gpb.SubscribeRequest) error {
	if f.failNextSend {
		f.failNextSend = false
		return errors.New("error occurred")
	}
	return nil
}

func (f *fakeSubscribeClient) Recv() (*gpb.SubscribeResponse, error) {
	select {
	case s := <-f.ch:
		return s, nil
	case err := <-f.errCh:
		return nil, err
	}
}

func (f *fakeSubscribeClient) sendRecvErr() {
	f.errCh <- errors.New("Recv error")
}

func (f *fakeSubscribeClient) sendErr() {
	f.ch <- &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Error{}}
}

func (f *fakeSubscribeClient) sendSync() {
	f.ch <- &gpb.SubscribeResponse{
		Response: &gpb.SubscribeResponse_SyncResponse{
			SyncResponse: true,
		}}
}

func (f *fakeSubscribeClient) sendUpdate() {
	f.ch <- &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Update{}}
}

func (f *fakeSubscribeClient) sendCustomUpdate(u *gpb.SubscribeResponse) {
	f.ch <- u
}

type record struct {
	connects []string
	resets   []string
	syncs    []string
	updates  []*gpb.Notification
}

func (r *record) assertLast(t *testing.T, desc string, wantConnect, wantSync, wantReset []string, wantUpdate int) {
	t.Helper()
	sort.Strings(r.connects)
	sort.Strings(r.syncs)
	sort.Strings(r.resets)
	switch {
	case !reflect.DeepEqual(r.connects, wantConnect):
		t.Fatalf("%s: Mismatched connects: got %v, want %v", desc, r.connects, wantConnect)
	case !reflect.DeepEqual(r.syncs, wantSync):
		t.Fatalf("%s: Mismatched syncs: got %v, want %v", desc, r.syncs, wantSync)
	case !reflect.DeepEqual(r.resets, wantReset):
		t.Fatalf("%s: Mismatched resets: got %v, want %v", desc, r.resets, wantReset)
	case len(r.updates) != wantUpdate:
		t.Fatalf("%s: Mismatched number of updates: got %v, want %v", desc, len(r.updates), wantUpdate)
	}
	r.connects = nil
	r.resets = nil
	r.syncs = nil
	r.updates = nil
	log.Infof("Successfully checked %v", desc)
}

type fakeConnection struct {
	m        *connection.Manager
	failNext bool
}

func (f *fakeConnection) stageErr() {
	f.failNext = true
}

func (f *fakeConnection) Connection(ctx context.Context, addr, dialer string) (conn *grpc.ClientConn, done func(), err error) {
	if f.failNext {
		f.failNext = false
		return nil, func() {}, errors.New("could not connect")
	}
	return f.m.Connection(ctx, addr, dialer)
}

func newFakeConnection(t *testing.T) *fakeConnection {
	t.Helper()
	m, err := connection.NewManager(grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})))
	if err != nil {
		t.Fatalf("failed to initialize Manager: %v", err)
	}
	return &fakeConnection{
		m: m,
	}
}

func TestRetrySubscribe(t *testing.T) {
	origBaseDelay, origMaxDelay := RetryBaseDelay, RetryMaxDelay
	defer func() { RetryBaseDelay, RetryMaxDelay = origBaseDelay, origMaxDelay }()
	RetryBaseDelay, RetryMaxDelay = 0, 10*time.Millisecond

	f := newFakeConnection(t)
	f.stageErr() // Initial failure causing retry.

	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	origSubscribeClient := subscribeClient
	defer func() { subscribeClient = origSubscribeClient }()
	fc := newFakeSubscribeClient()
	fc.stageSendErr() // Initial failure causing retry.
	subscribeClient = func(ctx context.Context, conn *grpc.ClientConn) (gpb.GNMI_SubscribeClient, error) {
		go func() {
			select {
			case <-ctx.Done():
				fc.sendRecvErr() // Simulate stream closure.
			}
		}()
		return fc, nil
	}

	r := record{}
	creds := &fakeCreds{}
	creds.stageErr() // Initial failure causing retry.
	m, err := NewManager(Config{
		Credentials: creds,
		Timeout:     time.Minute,
		Connect: func(s string) {
			r.connects = append(r.connects, s)
		},
		ConnectionManager: f,
		Sync: func(s string) {
			r.syncs = append(r.syncs, s)
		},
		Reset: func(s string) {
			r.resets = append(r.resets, s)
		},
		Update: func(_ string, g *gpb.Notification) {
			r.updates = append(r.updates, g)
		},
	})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}
	chHandleUpdate := make(chan struct{}, 1)
	m.testSync = func() {
		chHandleUpdate <- struct{}{}
	}

	name := "device1"
	ta := &tpb.Target{
		Addresses: []string{addr},
		Credentials: &tpb.Credentials{
			Username:   "user1",
			PasswordId: "pass1",
		},
	}
	err = m.Add(name, ta, validSubscribeRequest)
	if err != nil {
		t.Fatalf("Add returned error, want no error")
	}
	defer m.Remove(name)
	mTarget, ok := m.targets[name]
	if !ok {
		t.Fatalf("missing internal target")
	}

	assertTargets(t, m, 1)
	r.assertLast(t, "initial state", nil, nil, nil, 0)

	fc.sendSync()
	<-chHandleUpdate
	r.assertLast(t, "initial sync", []string{name}, []string{name}, nil, 0)

	lim := 10
	for i := 0; i < lim; i++ {
		fc.sendUpdate()
		<-chHandleUpdate
	}
	r.assertLast(t, "updates sent", nil, nil, nil, 10)

	fc.sendErr()
	<-chHandleUpdate
	r.assertLast(t, "error update", nil, nil, nil, 0)

	fc.sendCustomUpdate(&gpb.SubscribeResponse{})
	<-chHandleUpdate
	r.assertLast(t, "nil update", nil, nil, nil, 0)

	fc.sendUpdate()
	<-chHandleUpdate
	r.assertLast(t, "update after errors", nil, nil, nil, 1)

	fc.sendRecvErr()
	<-chHandleUpdate
	r.assertLast(t, "recv error", nil, nil, []string{name}, 0)

	fc.sendSync()
	<-chHandleUpdate
	r.assertLast(t, "stream recovers", []string{name}, []string{name}, nil, 0)

	fc.sendUpdate()
	<-chHandleUpdate
	r.assertLast(t, "update after recovery", nil, nil, nil, 1)

	m.Remove(name)
	<-chHandleUpdate
	r.assertLast(t, "recv error after remove", nil, nil, []string{name}, 0)

	assertTargets(t, m, 0)

	go fc.sendUpdate()
	<-mTarget.finished
	r.assertLast(t, "no recovery after remove", nil, nil, nil, 0)
}

func TestRemoveDuringBackoff(t *testing.T) {
	origBaseDelay, origMaxDelay := RetryBaseDelay, RetryMaxDelay
	defer func() { RetryBaseDelay, RetryMaxDelay = origBaseDelay, origMaxDelay }()
	RetryBaseDelay, RetryMaxDelay = time.Hour, time.Hour // High delay exceeding test timeout.

	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	origSubscribeClient := subscribeClient
	defer func() { subscribeClient = origSubscribeClient }()
	fc := newFakeSubscribeClient()
	subscribeClient = func(ctx context.Context, conn *grpc.ClientConn) (gpb.GNMI_SubscribeClient, error) {
		go func() {
			select {
			case <-ctx.Done():
				fc.sendRecvErr() // Simulate stream closure.
			}
		}()
		return fc, nil
	}

	m, err := NewManager(Config{
		Credentials:       &fakeCreds{},
		ConnectionManager: newFakeConnection(t),
	})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}
	chHandleUpdate := make(chan struct{}, 1)
	m.testSync = func() {
		chHandleUpdate <- struct{}{}
	}

	name := "device1"
	err = m.Add(name, &tpb.Target{Addresses: []string{addr}}, validSubscribeRequest)
	if err != nil {
		t.Fatalf("got error adding: %v, want no error", err)
	}
	assertTargets(t, m, 1)

	// Induce failure and backoff.
	fc.sendRecvErr()
	<-chHandleUpdate

	if err := m.Remove(name); err != nil {
		t.Fatalf("got error removing: %v, want no error", err)
	}
	assertTargets(t, m, 0)
}

func TestCancelSubscribe(t *testing.T) {
	origSubscribeClient := subscribeClient
	defer func() { subscribeClient = origSubscribeClient }()
	fc := newFakeSubscribeClient()
	subscribeClient = func(ctx context.Context, conn *grpc.ClientConn) (gpb.GNMI_SubscribeClient, error) {
		return fc, nil
	}

	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	m, err := NewManager(Config{
		Credentials:       &fakeCreds{},
		ConnectionManager: newFakeConnection(t),
	})
	if err != nil {
		t.Fatal("could not initialize Manager")
	}

	cCtx, cancel := context.WithCancel(context.Background())
	cancel()
	conn, done, err := m.connectionManager.Connection(context.Background(), addr, "")
	if err != nil {
		t.Fatalf("error creating connection: %v", err)
	}
	defer done()
	if err := m.subscribe(cCtx, "device", conn, &gpb.SubscribeRequest{}); err == nil {
		t.Fatalf("got no error, want error")
	}
}

func TestCustomizeRequest(t *testing.T) {
	dev := "device"
	tests := []struct {
		desc string
		sr   *gpb.SubscribeRequest
		want *gpb.SubscribeRequest
	}{
		{
			desc: "no existing prefix",
			sr: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
			want: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Prefix: &gpb.Path{Target: dev},
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "existing prefix",
			sr: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Prefix: &gpb.Path{Origin: "openconfig"},
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
			want: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Prefix: &gpb.Path{Origin: "openconfig", Target: dev},
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "existing prefix with target",
			sr: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Prefix: &gpb.Path{Origin: "openconfig", Target: "unknown target"},
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
			want: &gpb.SubscribeRequest{
				Request: &gpb.SubscribeRequest_Subscribe{
					&gpb.SubscriptionList{
						Prefix: &gpb.Path{Origin: "openconfig", Target: dev},
						Subscription: []*gpb.Subscription{
							{
								Path: &gpb.Path{
									Elem: []*gpb.PathElem{{Name: "path1"}},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got := customizeRequest(dev, tt.sr)
		if !proto.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestCreateConn(t *testing.T) {
	addr, stop := newDevice(t, &fakeServer{})
	defer stop()

	tests := []struct {
		desc    string
		ta      *tpb.Target
		makeCtx func() context.Context
		wantErr bool
	}{
		{
			desc: "missing address",
			ta:   &tpb.Target{},
			makeCtx: func() context.Context {
				return context.Background()
			},
			wantErr: true,
		},
		{
			desc: "canceled context",
			ta: &tpb.Target{
				Addresses: []string{addr},
			},
			makeCtx: func() context.Context {
				cCtx, cancel := context.WithCancel(context.Background())
				cancel()
				return cCtx
			},
			wantErr: true,
		},
		{
			desc: "valid",
			ta: &tpb.Target{
				Addresses: []string{addr},
			},
			makeCtx: func() context.Context {
				return context.Background()
			},
		},
	}

	for _, tt := range tests {
		m, err := NewManager(Config{
			Credentials:       &fakeCreds{},
			ConnectionManager: newFakeConnection(t),
		})
		if err != nil {
			t.Fatal("could not initialize Manager")
		}
		_, _, err = m.createConn(tt.makeCtx(), "device", tt.ta)
		switch {
		case err == nil && !tt.wantErr:
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error.", tt.desc)
		}
	}
}
