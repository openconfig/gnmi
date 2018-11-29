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

package gnmi

import (
	"golang.org/x/net/context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/testing/fake/testing/grpc/config"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

type direction string

const (
	sendDirection direction = "Send"
	recvDirection direction = "Recv"
	noneDirection direction = "None"
)

type event interface {
	Direction() direction
	String() string
}

type cancelEvent struct {
	d direction
}

func (c *cancelEvent) String() string       { return fmt.Sprintf("%s: Cancel Event", c.d) }
func (c *cancelEvent) Direction() direction { return c.d }

type errorEvent struct {
	d   direction
	err error
}

func (e *errorEvent) String() string {
	if e.err == nil {
		return fmt.Sprintf("%s: no error", e.d)
	}
	return fmt.Sprintf("%s: %s", e.d, e.err.Error())
}
func (e *errorEvent) Direction() direction { return e.d }

type receiveEvent struct {
	d direction
	e *gnmipb.SubscribeRequest
}

func (r *receiveEvent) String() string       { return fmt.Sprintf("%s: Event\n%s", r.d, r.e) }
func (r *receiveEvent) Direction() direction { return r.d }

type fakeStream struct {
	grpc.ServerStream
	curr   int
	events []event
	recv   []*gnmipb.SubscribeResponse
	ctx    context.Context
	Cancel func()
	rEvent chan event
	sEvent chan event
	mu     sync.Mutex
	synced int
}

func newFakeStream(events []event) *fakeStream {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeStream{
		events: events,
		ctx:    ctx,
		recv:   []*gnmipb.SubscribeResponse{},
		Cancel: cancel,
		rEvent: make(chan event),
		sEvent: make(chan event),
	}
	go func() {
		for _, e := range events {
			switch e.Direction() {
			default:
				switch e.(type) {
				case *cancelEvent:
					f.Cancel()
					return
				}
			case sendDirection:
				f.sEvent <- e
			case recvDirection:
				f.rEvent <- e
			}
		}
		f.Cancel()
	}()
	go func() {
		<-f.ctx.Done()
		close(f.sEvent)
		close(f.rEvent)
	}()
	return f
}

func (f *fakeStream) Send(resp *gnmipb.SubscribeResponse) (err error) {
	if resp.GetSyncResponse() {
		f.mu.Lock()
		f.synced = 1
		f.mu.Unlock()
	}
	f.recv = append(f.recv, resp)
	_, ok := <-f.sEvent
	if !ok {
		return io.EOF
	}
	return nil
}

func (f *fakeStream) Recv() (resp *gnmipb.SubscribeRequest, err error) {
	e, ok := <-f.rEvent
	if !ok {
		return nil, io.EOF
	}
	switch v := e.(type) {
	default:
		return nil, io.EOF
	case *receiveEvent:
		return v.e, nil
	case *errorEvent:
		return nil, v.err
	}
}

func (f *fakeStream) Context() context.Context {
	return f.ctx
}

func TestClientCreate(t *testing.T) {
	defaultConfig := &fpb.Config{
		Target: "arista",
		Port:   -1,
		Values: []*fpb.Value{{
			Path: []string{"interfaces", "interface[name=Port-Channel1]", "state", "counters", "in-octets"},
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 0,
				Distribution: &fpb.IntValue_Range{
					&fpb.IntRange{
						Minimum:  0,
						Maximum:  10000000,
						DeltaMax: 100,
						DeltaMin: 0,
					}}}},
		}},
	}
	tests := []struct {
		config *fpb.Config
		events []event
		err    codes.Code
	}{{
		config: nil,
		err:    codes.FailedPrecondition,
	}, {
		config: defaultConfig,
		err:    codes.Aborted,
	}, {
		config: defaultConfig,
		events: []event{
			&cancelEvent{d: "None"},
		},
		err: codes.Aborted,
	}, {
		config: defaultConfig,
		events: []event{
			&receiveEvent{d: "Recv"},
		},
		err: codes.InvalidArgument,
	}, {
		config: defaultConfig,
		events: []event{
			&receiveEvent{d: "Recv", e: &gnmipb.SubscribeRequest{
				Request: &gnmipb.SubscribeRequest_Subscribe{},
			}},
			&receiveEvent{d: "Recv"},
		},
		err: codes.InvalidArgument,
	}, {
		config: defaultConfig,
		events: []event{
			&receiveEvent{d: "Recv", e: &gnmipb.SubscribeRequest{
				Request: &gnmipb.SubscribeRequest_Subscribe{
					Subscribe: &gnmipb.SubscriptionList{},
				},
			}},
			&receiveEvent{d: "Recv"},
			&errorEvent{d: "Recv", err: errors.New("cancelable error")},
		},
		err: codes.OK,
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   -1,
			Values: []*fpb.Value{{
				Path: []string{"interfaces", "interface[name=Port-Channel1]", "state", "counters", "in-octets"},
				Value: &fpb.Value_IntValue{&fpb.IntValue{
					Value: 0,
					Distribution: &fpb.IntValue_Range{
						&fpb.IntRange{
							Minimum:  0,
							Maximum:  10000000,
							DeltaMax: 100,
							DeltaMin: 0,
						}}}},
				Repeat: 2,
			}},
		},
		events: []event{
			&receiveEvent{d: "Recv", e: &gnmipb.SubscribeRequest{
				Request: &gnmipb.SubscribeRequest_Subscribe{
					Subscribe: &gnmipb.SubscriptionList{},
				},
			}},
			&receiveEvent{d: "Recv"},
			&receiveEvent{d: "Recv"},
		},
		err: codes.OK,
	}}
	pp := pretty.Config{
		IncludeUnexported: true,
		PrintStringers:    true,
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test case %d", i+1), func(t *testing.T) {
			c := NewClient(tt.config)
			s := newFakeStream(tt.events)
			gotErr := c.Run(s)
			if gotErr != nil {
				if got, want := grpc.Code(gotErr), tt.err; got != want {
					t.Errorf("Test:\n%s\nRun() unexpected error %s: got %s, want %s", pp.Sprint(tt), gotErr.Error(), got, want)
				}
				return
			}
			if tt.err != codes.OK {
				t.Errorf("Test:\n%s\nRun() expected error %s: got nil", pp.Sprint(tt), tt.err)
			}
		})
	}
}

func festClientSend(t *testing.T) {
	defaultConfig := &fpb.Config{
		Target: "arista",
		Port:   -1,
		Values: []*fpb.Value{},
	}
	tests := []struct {
		config  *fpb.Config
		wantErr string
		events  []event
	}{{
		config:  defaultConfig,
		wantErr: "invalid configuration",
		events: []event{
			&receiveEvent{d: "Send"},
		},
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   -1,
			Values: []*fpb.Value{{
				Path: []string{"interfaces", "interface[name=Port-Channel1]", "state", "counters", "in-octets"},
				Value: &fpb.Value_IntValue{&fpb.IntValue{
					Value: 0,
					Distribution: &fpb.IntValue_Range{
						&fpb.IntRange{
							Minimum:  0,
							Maximum:  10000000,
							DeltaMax: 100,
							DeltaMin: 0,
						}}}},
				Repeat: 2,
			}},
		},
		events: []event{
			&receiveEvent{d: "Send"},
			&receiveEvent{d: "Send"},
			&receiveEvent{d: "Send"},
		},
	}}
	pp := pretty.Config{
		IncludeUnexported: true,
		PrintStringers:    true,
	}
	for _, tt := range tests {
		c := NewClient(tt.config)
		s := newFakeStream(tt.events)
		defer s.Cancel()
		c.subscribe = &gnmipb.SubscriptionList{Mode: gnmipb.SubscriptionList_ONCE}
		err := c.reset()
		switch {
		case err == nil && tt.wantErr != "":
			t.Fatalf("reset() failed: got %v, want %v", err, tt.wantErr)
		case err != nil && !strings.HasPrefix(err.Error(), tt.wantErr):
			t.Fatalf("reset() failed: got %q, want %q", err, tt.wantErr)
		}
		c.send(s)
		t.Logf("Received Events:\n%s\n", pp.Sprint(s.recv))
		if c.errors != 0 {
			t.Fatalf("send(%s) errored", pp.Sprint(tt.events))
		}
		s.mu.Lock()
		synced := s.synced
		s.mu.Unlock()
		if synced == 0 {
			t.Fatalf("send(%s) failed to sync stream", pp.Sprint(tt.events))
		}
	}
}

func TestNewAgent(t *testing.T) {
	tests := []struct {
		config *fpb.Config
		err    error
	}{{
		config: nil,
		err:    fmt.Errorf("config not provided"),
	}, {
		config: &fpb.Config{},
		err:    fmt.Errorf("config not provided"),
	}, {
		config: &fpb.Config{
			Target: "arista",
		},
		err: nil,
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   0,
		},
		err: nil,
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   -1,
		},
		err: nil,
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   -1,
		},
		err: nil,
	}, {
		config: &fpb.Config{
			Target: "arista",
			Port:   -1,
			Values: []*fpb.Value{{
				Path: []string{"interfaces", "interface[name=Port-Channel1]", "state", "counters", "in-octets"},
				Value: &fpb.Value_IntValue{&fpb.IntValue{
					Value: 0,
					Distribution: &fpb.IntValue_Range{
						&fpb.IntRange{
							Minimum:  0,
							Maximum:  10000000,
							DeltaMax: 100,
							DeltaMin: 0,
						}}}},
			}},
		},
		err: nil,
	}, {
		config: &fpb.Config{
			Target:      "arista",
			Port:        -1,
			DisableSync: true,
			Values: []*fpb.Value{{
				Path: []string{"interfaces", "interface[name=Port-Channel1]", "state", "counters", "in-octets"},
				Value: &fpb.Value_IntValue{&fpb.IntValue{
					Value: 0,
					Distribution: &fpb.IntValue_Range{
						&fpb.IntRange{
							Minimum:  0,
							Maximum:  10000000,
							DeltaMax: 100,
							DeltaMin: 0,
						}}}},
			}, {
				Value: &fpb.Value_Sync{uint64(1)},
			}},
		},
		err: nil,
	}}
	certOpt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("Test case %+v", tc.config), func(t *testing.T) {
			a, err := New(tc.config, []grpc.ServerOption{certOpt})
			if err != nil {
				if tc.err == nil || tc.err.Error() != err.Error() {
					t.Fatalf("New(%q) error return: got %v, want %v", tc.config, err, tc.err)
				}
				return
			}
			conn, err := grpc.Dial(a.Address(), grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})))
			if err != nil {
				t.Fatalf("New(%q) failed to dial server: %s", tc.config, err)
			}
			c := gnmipb.NewGNMIClient(conn)
			s, err := c.Subscribe(context.Background())
			if err != nil {
				t.Fatalf("New(%q).Subscribe() failed: %v", tc.config, err)
			}
			sub := &gnmipb.SubscribeRequest{
				Request: &gnmipb.SubscribeRequest_Subscribe{
					Subscribe: &gnmipb.SubscriptionList{},
				},
			}
			if err := s.Send(sub); err != nil {
				t.Fatalf("New(%q).Send(%q) failed: got %s", tc.config, sub, err)
			}
			if _, err = s.Recv(); err != nil {
				t.Fatalf("New(%q).Recv() failed: got %s", tc.config, err)
			}
			if got, want := a.State(), fpb.State_RUNNING; got != want {
				t.Errorf("New(%q).State() failed: got %q, want %q", tc.config, got, want)
			}
			a.Close()
			if got, want := a.State(), fpb.State_STOPPED; got != want {
				t.Errorf("New(%q).Close() failed: got %q, want %q", tc.config, got, want)
			}
		})
	}
}
