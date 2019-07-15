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

package connection

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/unimplemented"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

func newDevice(t *testing.T) (string, func()) {
	t.Helper()

	srv := grpc.NewServer()
	s := &unimplemented.Server{}
	pb.RegisterGNMIServer(srv, s)
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	go srv.Serve(lis)

	return lis.Addr().String(), srv.Stop
}

func TestCtxCanceled(t *testing.T) {
	addr, stop := newDevice(t)
	defer stop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, err := NewManager(grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to initialize Manager: %v", err)
	}
	if _, _, err := m.Connection(ctx, addr); err == nil {
		t.Errorf("Connection returned no error, want error")
	}
}

func assertConns(t *testing.T, m *Manager, want int) {
	if l := len(m.conns); l != want {
		t.Fatalf("got %v connections, want %v", l, want)
	}
}

func assertRefs(t *testing.T, m *Manager, addr string, want int) {
	c, ok := m.conns[addr]
	if !ok {
		t.Fatalf("connection %q missing", addr)
	}
	if r := c.ref; r != want {
		t.Fatalf("got %v references, want %v", r, want)
	}
}

func TestConcurrentConnection(t *testing.T) {
	addr, stop := newDevice(t)
	defer stop()
	ctx := context.Background()
	m, err := NewManager(grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to initialize Manager: %v", err)
	}
	var wg sync.WaitGroup
	lim := 300
	wg.Add(lim)

	for i := 0; i < lim; i++ {
		go func() {
			conn, _, err := m.Connection(ctx, addr)
			if err != nil {
				t.Fatalf("got error creating connection: %v, want no error", err)
			}
			if conn == nil {
				t.Fatalf("got nil connection, expected not nil")
			}
			wg.Done()
		}()
	}
	wg.Wait()

	assertConns(t, m, 1)
	assertRefs(t, m, addr, lim)
}

func TestDone(t *testing.T) {
	addr, stop := newDevice(t)
	defer stop()
	ctx := context.Background()
	m, err := NewManager(grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to initialize Manager: %v", err)
	}

	_, done1, err := m.Connection(ctx, addr)
	if err != nil {
		t.Fatalf("got error creating connection: %v, want no error", err)
	}
	assertConns(t, m, 1)
	assertRefs(t, m, addr, 1)

	_, done2, err := m.Connection(ctx, addr)
	if err != nil {
		t.Fatalf("got error creating connection: %v, want no error", err)
	}
	assertConns(t, m, 1)
	assertRefs(t, m, addr, 2)

	done1()
	assertConns(t, m, 1)
	assertRefs(t, m, addr, 1)

	done2()
	assertConns(t, m, 0)

	done1() // No panic.
	done2() // No panic.
}

// TestConnectionDone simulates concurrently creating, reusing, and
// destroying multiple connections to different addresses.
func TestConnectionDone(t *testing.T) {
	type connTest struct {
		desc         string
		connections  int
		dones        int
		connectionID string // Populated by test.
	}

	tests := []*connTest{
		// NOTE: done counts are relatively low to avoid connection
		// thrashing that causes gRPC attempts to hang and affect this test.
		{
			desc:        "connection1",
			connections: 200,
			dones:       20,
		},
		{
			desc:        "connection2",
			connections: 200,
			dones:       25,
		},
		{
			desc:        "connection3",
			connections: 200,
			dones:       30,
		},
	}

	process := func(ctx context.Context, t *testing.T, c *connTest, m *Manager, wg *sync.WaitGroup) {
		for i := 0; i < c.connections; i++ {
			d := i
			go func() {
				conn, done, err := m.Connection(ctx, c.connectionID)
				wg.Done()
				if conn == nil {
					t.Fatalf("got nil connection")
				}
				if err != nil {
					t.Fatalf("got error creating connection: %v, want no error", err)
				}
				if d < c.dones {
					go func() {
						done()
						wg.Done()
					}()
				}
			}()
		}
	}

	var wg sync.WaitGroup
	m, err := NewManager(grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to initialize Manager: %v", err)
	}
	for _, tt := range tests {
		wg.Add(tt.connections + tt.dones)
		addr, stop := newDevice(t)
		defer stop()
		tt.connectionID = addr
	}
	for _, tt := range tests {
		go process(context.Background(), t, tt, m, &wg)
	}
	wg.Wait()

	if len(m.conns) != len(tests) {
		t.Fatalf("got %v connections, want %v", len(m.conns), len(tests))
	}
	for _, tt := range tests {
		_, ok := m.conns[tt.connectionID]
		if !ok {
			t.Fatalf("%s: missing connection", tt.desc)
		}
		assertRefs(t, m, tt.connectionID, tt.connections-tt.dones)
	}
}

func errDialWait(d time.Duration) func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
	return func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		time.Sleep(d)
		return nil, errors.New("error occurred")
	}
}

func TestConcurrentDialErr(t *testing.T) {
	ctx := context.Background()
	m, err := NewManagerCustom(errDialWait(time.Second), grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to initialize Manager: %v", err)
	}
	var wg sync.WaitGroup
	lim := 2
	wg.Add(lim)
	errs := make(chan error, lim)
	start := make(chan struct{})

	for i := 0; i < lim; i++ {
		go func() {
			<-start
			_, _, err := m.Connection(ctx, "")
			if err == nil {
				t.Fatal("got no error, want error")
			}
			errs <- err
			wg.Done()
		}()
	}
	close(start)
	wg.Wait()

	close(errs)
	var prevErr error
	for err := range errs {
		if prevErr == nil {
			prevErr = err
		}
		if err != prevErr {
			t.Fatal("got different error instance, want same")
		}
	}
}

func TestNewManagerCustom(t *testing.T) {
	tests := []struct {
		desc    string
		d       Dial
		opts    []grpc.DialOption
		wantErr bool
	}{
		{
			desc:    "missing dial",
			opts:    []grpc.DialOption{grpc.WithBlock()},
			wantErr: true,
		}, {
			desc: "missing opts",
			d:    grpc.DialContext,
		}, {
			desc: "valid dial and opts",
			opts: []grpc.DialOption{grpc.WithBlock()},
			d:    grpc.DialContext,
		},
	}

	for _, tt := range tests {
		_, err := NewManagerCustom(tt.d, tt.opts...)
		switch {
		case err == nil && !tt.wantErr:
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error.", tt.desc)
		}
	}
}
