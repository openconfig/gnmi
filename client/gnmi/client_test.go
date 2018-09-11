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

package client

import (
	"crypto/tls"
	"testing"
	"time"

	"context"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/grpc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/testing/fake/gnmi"
	"github.com/openconfig/gnmi/testing/fake/testing/grpc/config"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

func TestClient(t *testing.T) {
	tests := []struct {
		desc       string
		q          client.Query
		updates    []*fpb.Value
		disableEOF bool
		wantErr    bool
		wantNoti   []client.Notification

		poll        int
		wantPollErr string
	}{{
		desc:    "empty query",
		q:       client.Query{},
		wantErr: true,
	}, {
		desc: "once query with one update",
		q: client.Query{
			Target:  "dev",
			Type:    client.Once,
			Queries: []client.Path{{"a"}},
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		updates: []*fpb.Value{{
			Path:      []string{"dev", "a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		wantNoti: []client.Notification{
			client.Connected{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Sync{},
		},
	}, {
		desc: "poll query with x3 by Poll()",
		poll: 3,
		q: client.Query{
			Target:  "dev",
			Type:    client.Poll,
			Queries: []client.Path{{"a"}},
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		updates: []*fpb.Value{{
			Path:      []string{"dev", "a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		wantNoti: []client.Notification{
			client.Connected{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Sync{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Sync{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Sync{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Sync{},
		},
	}, {
		desc: "once query with updates and deletes",
		q: client.Query{
			Target:  "dev",
			Type:    client.Once,
			Queries: []client.Path{{"a"}},
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		updates: []*fpb.Value{{
			Path:      []string{"dev", "a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"dev", "a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"dev", "a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_Delete{Delete: &fpb.DeleteValue{}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 300},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		wantNoti: []client.Notification{
			client.Connected{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
			client.Update{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 100), Val: 5},
			client.Delete{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 200)},
			client.Sync{},
		},
	}, {
		desc:       "stream query with updates and deletes",
		disableEOF: true,
		q: client.Query{
			Target:  "dev",
			Type:    client.Stream,
			Queries: []client.Path{{"a"}},
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		updates: []*fpb.Value{{
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}, {
			Path:      []string{"dev", "a"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"dev", "a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"dev", "a", "c"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"dev", "a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 300},
			Repeat:    1,
			Value:     &fpb.Value_Delete{Delete: &fpb.DeleteValue{}},
		}, {
			Path:      []string{"dev", "a", "c"},
			Timestamp: &fpb.Timestamp{Timestamp: 400},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 50}},
		}},
		wantNoti: []client.Notification{
			client.Connected{},
			client.Sync{},
			client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 200), Val: 5},
			client.Update{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 200), Val: 5},
			client.Update{Path: []string{"dev", "a", "c"}, TS: time.Unix(0, 200), Val: 5},
			client.Delete{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 300)},
			client.Update{Path: []string{"dev", "a", "c"}, TS: time.Unix(0, 400), Val: 50},
		},
	}}
	opt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := gnmi.New(
				&fpb.Config{
					Target:      "dev1",
					DisableSync: true,
					Values:      tt.updates,
					DisableEof:  tt.disableEOF,
				},
				[]grpc.ServerOption{opt},
			)
			go s.Serve()
			if err != nil {
				t.Fatal("failed to start test server")
			}
			defer s.Close()

			q := tt.q
			q.Addrs = []string{s.Address()}
			c := client.New()
			defer c.Close()
			var gotNoti []client.Notification
			q.NotificationHandler = func(n client.Notification) error {
				gotNoti = append(gotNoti, n)
				return nil
			}
			err = c.Subscribe(context.Background(), q)
			switch {
			case tt.wantErr && err != nil:
				return
			case tt.wantErr && err == nil:
				t.Fatalf("c.Subscribe(): got nil error, expected non-nil")
			case !tt.wantErr && err != nil:
				t.Fatalf("c.Subscribe(): got error %v, expected nil", err)
			}
			for i := 0; i < tt.poll; i++ {
				err := c.Poll()
				switch {
				case err == nil && tt.wantPollErr != "":
					t.Errorf("c.Poll(): got nil error, expected non-nil %v", tt.wantPollErr)
				case err != nil && tt.wantPollErr == "":
					t.Errorf("c.Poll(): got error %v, expected nil", err)
				case err != nil && err.Error() != tt.wantPollErr:
					t.Errorf("c.Poll(): got error %v, expected error %v", err, tt.wantPollErr)
				}
			}
			if diff := pretty.Compare(tt.wantNoti, gotNoti); diff != "" {
				t.Errorf("unexpected updates:\n%s", diff)
			}
			impl, err := c.Impl()
			if err != nil {
				t.Fatalf("c.Impl() failed: %v", err)
			}
			if got, want := impl.(*Client).Peer(), s.Address(); got != want {
				t.Errorf("Peer() failed: got %v, want %v", got, want)
			}
		})
	}
}

func TestGNMIMessageUpdates(t *testing.T) {
	var gotNoti []client.Notification
	q := client.Query{
		Target:  "dev",
		Type:    client.Stream,
		Queries: []client.Path{{"a"}},
		TLS:     &tls.Config{InsecureSkipVerify: true},
		NotificationHandler: func(n client.Notification) error {
			gotNoti = append(gotNoti, n)
			return nil
		},
	}
	updates := []*gpb.SubscribeResponse{
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 200,
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"dev", "a"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 300,
				Prefix:    &gpb.Path{Target: "dev", Element: []string{"a"}},
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 400,
				Prefix:    &gpb.Path{Target: "dev", Origin: "oc", Element: []string{"a"}},
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
	}
	wantNoti := []client.Notification{
		client.Connected{},
		client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 200), Val: 5},
		client.Update{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 300), Val: 5},
		client.Update{Path: []string{"dev", "oc", "a", "b"}, TS: time.Unix(0, 400), Val: 5},
		client.Sync{},
	}
	opt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}
	s, err := gnmi.New(
		&fpb.Config{
			Generator: &fpb.Config_Fixed{Fixed: &fpb.FixedGenerator{Responses: updates}},
		},
		[]grpc.ServerOption{opt},
	)
	go s.Serve()
	if err != nil {
		t.Fatal("failed to start test server")
	}
	defer s.Close()
	q.Addrs = []string{s.Address()}
	c := client.New()
	defer c.Close()
	err = c.Subscribe(context.Background(), q)
	if diff := pretty.Compare(wantNoti, gotNoti); diff != "" {
		t.Errorf("unexpected updates:\n%s", diff)
	}
}

func TestGNMIWithSubscribeRequest(t *testing.T) {
	q, err := client.NewQuery(&gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			Subscribe: &gpb.SubscriptionList{
				Mode:   gpb.SubscriptionList_STREAM,
				Prefix: &gpb.Path{Target: "dev"},
				Subscription: []*gpb.Subscription{
					{Path: &gpb.Path{Element: []string{"a"}}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create Query from gnmi SubscribeRequest: %v", err)
	}
	q.TLS = &tls.Config{InsecureSkipVerify: true}
	var gotNoti []client.Notification
	q.NotificationHandler = func(n client.Notification) error {
		gotNoti = append(gotNoti, n)
		return nil
	}
	updates := []*gpb.SubscribeResponse{
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 200,
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"dev", "a"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 300,
				Prefix:    &gpb.Path{Target: "dev", Element: []string{"a"}},
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
		{Response: &gpb.SubscribeResponse_Update{
			Update: &gpb.Notification{
				Timestamp: 400,
				Prefix:    &gpb.Path{Target: "dev", Origin: "oc", Element: []string{"a"}},
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{Element: []string{"b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}},
					},
				},
			},
		}},
	}
	wantNoti := []client.Notification{
		client.Connected{},
		client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 200), Val: 5},
		client.Update{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 300), Val: 5},
		client.Update{Path: []string{"dev", "oc", "a", "b"}, TS: time.Unix(0, 400), Val: 5},
		client.Sync{},
	}
	opt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}
	s, err := gnmi.New(
		&fpb.Config{
			Generator: &fpb.Config_Fixed{Fixed: &fpb.FixedGenerator{Responses: updates}},
		},
		[]grpc.ServerOption{opt},
	)
	go s.Serve()
	if err != nil {
		t.Fatal("failed to start test server")
	}
	defer s.Close()
	q.Addrs = []string{s.Address()}
	c := client.New()
	defer c.Close()
	err = c.Subscribe(context.Background(), q)
	if diff := pretty.Compare(wantNoti, gotNoti); diff != "" {
		t.Errorf("unexpected updates:\n%s\nwantnoti:%v\ngotnoti:%v\n", diff, wantNoti, gotNoti)
	}
}

func stringToPath(p string) *gpb.Path {
	pp, err := ygot.StringToPath(p, ygot.StructuredPath, ygot.StringSlicePath)
	if err != nil {
		panic(err)
	}
	return pp
}

func TestNoti(t *testing.T) {
	tests := []struct {
		desc     string
		prefix   []string
		path     *gpb.Path
		ts       time.Time
		u        *gpb.Update
		wantNoti client.Notification
		wantErr  bool
	}{
		{
			desc:     "nil update means delete",
			path:     stringToPath("dev/a/b"),
			ts:       time.Unix(0, 200),
			wantNoti: client.Delete{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 200)},
		}, {
			desc:     "update with TypedValue",
			path:     stringToPath("dev/a"),
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
		}, {
			desc:    "update with non-scalar TypedValue",
			path:    stringToPath("dev/a"),
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte("5")}}},
			wantErr: true,
		}, {
			desc:     "update with JSON value",
			path:     stringToPath("dev/a"),
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte("5")}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
		}, {
			desc:     "update with bytes value",
			path:     stringToPath("dev/a"),
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_BYTES, Value: []byte("5")}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: []byte("5")},
		}, {
			desc:    "update with un-unmarshalable JSON value",
			path:    stringToPath("dev/a"),
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`"5`)}},
			wantErr: true,
		}, {
			desc:    "update with unsupported value",
			path:    stringToPath("dev/a"),
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_PROTO}},
			wantErr: true,
		}, {
			desc:     "with prefix",
			prefix:   []string{"dev"},
			path:     stringToPath("a"),
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
		},
	}
	for _, tt := range tests {
		got, err := noti(tt.prefix, tt.path, tt.ts, tt.u)
		switch {
		case err != nil && !tt.wantErr:
			t.Errorf("%s: got error %v, want nil", tt.desc, err)
		case err == nil && tt.wantErr:
			t.Errorf("%s: got nil error, want non-nil", tt.desc)
		}
		if diff := pretty.Compare(tt.wantNoti, got); diff != "" {
			t.Errorf("%s: notification diff:\n%s", tt.desc, diff)
		}
	}
}

func TestPathToString(t *testing.T) {
	tests := []struct {
		desc string
		in   client.Path
		want string
	}{
		{"simple path", client.Path{"a", "b", "c"}, "a/b/c"},
		{"path with attributes", client.Path{"a", "b[k=v]", "c"}, "a/b[k=v]/c"},
		{"path with slashes", client.Path{"a", "b/0/1", "c"}, "a/b\\/0\\/1/c"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := pathToString(tt.in)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
