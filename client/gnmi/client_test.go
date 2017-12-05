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
	"reflect"
	"testing"
	"time"

	log "github.com/golang/glog"
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/grpc"
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

func TestConvertSetRequest(t *testing.T) {
	sr := client.SetRequest{
		Delete: []client.Path{
			{"a", "b"},
			{"c", "d"},
		},
		Update: []client.Leaf{
			{Path: client.Path{"e"}, Val: 2},
			{Path: client.Path{"f", "g"}, Val: "foo"},
		},
		Replace: []client.Leaf{
			{Path: client.Path{"h", "i"}, Val: true},
			{Path: client.Path{"j"}, Val: 5},
		},
	}
	want := &gpb.SetRequest{
		Delete: []*gpb.Path{
			{Element: []string{"a", "b"}},
			{Element: []string{"c", "d"}},
		},
		Update: []*gpb.Update{
			{
				Path:  &gpb.Path{Element: []string{"e"}},
				Val:   &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte(`2`)}},
				Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`2`)},
			},
			{
				Path:  &gpb.Path{Element: []string{"f", "g"}},
				Val:   &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte(`"foo"`)}},
				Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`"foo"`)},
			},
		},
		Replace: []*gpb.Update{
			{
				Path:  &gpb.Path{Element: []string{"h", "i"}},
				Val:   &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte(`true`)}},
				Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`true`)},
			},
			{
				Path:  &gpb.Path{Element: []string{"j"}},
				Val:   &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte(`5`)}},
				Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`5`)},
			},
		},
	}

	got, err := convertSetRequest(sr)
	if err != nil {
		log.Errorf("got error %v, want nil", err)
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("diff:\n%s", diff)
	}
}

func TestConvertSetResponse(t *testing.T) {
	tests := []struct {
		desc    string
		sr      *gpb.SetResponse
		want    client.SetResponse
		wantErr string
	}{
		{
			desc: "one update",
			sr: &gpb.SetResponse{
				Timestamp: 1,
				Response: []*gpb.UpdateResult{
					{},
				},
			},
			want: client.SetResponse{TS: time.Unix(0, 1)},
		},
		{
			desc: "one update with error",
			sr: &gpb.SetResponse{
				Timestamp: 1,
				Response: []*gpb.UpdateResult{
					{Message: &gpb.Error{Message: "foo"}},
				},
			},
			wantErr: `message:"foo" `,
		},
		{
			desc: "two updates with one error",
			sr: &gpb.SetResponse{
				Timestamp: 1,
				Response: []*gpb.UpdateResult{
					{},
					{Message: &gpb.Error{Message: "foo"}},
				},
			},
			wantErr: `message:"foo" `,
		},
		{
			desc: "two updates with two errors",
			sr: &gpb.SetResponse{
				Timestamp: 1,
				Response: []*gpb.UpdateResult{
					{Message: &gpb.Error{Message: "foo"}},
					{Message: &gpb.Error{Message: "bar"}},
				},
			},
			wantErr: `message:"foo" ; message:"bar" `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := convertSetResponse(tt.sr)
			switch {
			case err != nil && tt.wantErr == "":
				t.Errorf("got error %q, want nil", err)
			case err == nil && tt.wantErr != "":
				t.Errorf("got nil error, want %q", tt.wantErr)
			case err != nil && tt.wantErr != "":
				if err.Error() != tt.wantErr {
					t.Errorf("got error %q, want %q", err, tt.wantErr)
				}
			default:
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("got %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestNoti(t *testing.T) {
	tests := []struct {
		desc     string
		path     client.Path
		ts       time.Time
		u        *gpb.Update
		wantNoti client.Notification
		wantErr  bool
	}{
		{
			desc:     "nil update means delete",
			path:     []string{"dev", "a", "b"},
			ts:       time.Unix(0, 200),
			wantNoti: client.Delete{Path: []string{"dev", "a", "b"}, TS: time.Unix(0, 200)},
		}, {
			desc:     "update with TypedValue",
			path:     []string{"dev", "a"},
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{5}}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
		}, {
			desc:    "update with non-scalar TypedValue",
			path:    []string{"dev", "a"},
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{[]byte("5")}}},
			wantErr: true,
		}, {
			desc:     "update with JSON value",
			path:     []string{"dev", "a"},
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte("5")}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: 5},
		}, {
			desc:     "update with bytes value",
			path:     []string{"dev", "a"},
			ts:       time.Unix(0, 100),
			u:        &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_BYTES, Value: []byte("5")}},
			wantNoti: client.Update{Path: []string{"dev", "a"}, TS: time.Unix(0, 100), Val: []byte("5")},
		}, {
			desc:    "update with un-unmarshalable JSON value",
			path:    []string{"dev", "a"},
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: []byte(`"5`)}},
			wantErr: true,
		}, {
			desc:    "update with unsupported value",
			path:    []string{"dev", "a"},
			ts:      time.Unix(0, 100),
			u:       &gpb.Update{Value: &gpb.Value{Type: gpb.Encoding_PROTO}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		got, err := noti(tt.path, tt.ts, tt.u)
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

func TestProtoResponse(t *testing.T) {
	tests := []struct {
		desc    string
		notifs  []client.Notification
		want    *gpb.SubscribeResponse
		wantErr bool
	}{
		{
			desc: "empty response",
			want: &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Update{Update: new(gpb.Notification)}},
		},
		{
			desc: "updates and deletes",
			notifs: []client.Notification{
				client.Update{Path: client.Path{"a", "b"}, Val: 1, TS: time.Unix(0, 2)},
				client.Delete{Path: client.Path{"d", "e"}, TS: time.Unix(0, 4)},
				client.Update{Path: client.Path{"a", "c"}, Val: 2, TS: time.Unix(0, 3)},
			},
			want: &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Update{Update: &gpb.Notification{
				Timestamp: 2,
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{
							Element: []string{"a", "b"},
							Elem:    []*gpb.PathElem{{Name: "a"}, {Name: "b"}},
						},
						Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{1}},
					},
					{
						Path: &gpb.Path{
							Element: []string{"a", "c"},
							Elem:    []*gpb.PathElem{{Name: "a"}, {Name: "c"}},
						},
						Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{2}},
					},
				},
				Delete: []*gpb.Path{
					{
						Element: []string{"d", "e"},
						Elem:    []*gpb.PathElem{{Name: "d"}, {Name: "e"}},
					},
				},
			}}},
		},
		{
			desc: "nil updates",
			notifs: []client.Notification{
				client.Delete{Path: client.Path{"d", "e"}, TS: time.Unix(0, 4)},
			},
			want: &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Update{Update: &gpb.Notification{
				Timestamp: 4,
				Delete: []*gpb.Path{
					{
						Element: []string{"d", "e"},
						Elem:    []*gpb.PathElem{{Name: "d"}, {Name: "e"}},
					},
				},
			}}},
		},
		{
			desc: "nil deletes",
			notifs: []client.Notification{
				client.Update{Path: client.Path{"a", "b"}, Val: 1, TS: time.Unix(0, 2)},
				client.Update{Path: client.Path{"a", "c"}, Val: 2, TS: time.Unix(0, 3)},
			},
			want: &gpb.SubscribeResponse{Response: &gpb.SubscribeResponse_Update{Update: &gpb.Notification{
				Timestamp: 2,
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{
							Element: []string{"a", "b"},
							Elem:    []*gpb.PathElem{{Name: "a"}, {Name: "b"}},
						},
						Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{1}},
					},
					{
						Path: &gpb.Path{
							Element: []string{"a", "c"},
							Elem:    []*gpb.PathElem{{Name: "a"}, {Name: "c"}},
						},
						Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{2}},
					},
				},
			}}},
		},
		{
			desc: "bad path",
			notifs: []client.Notification{
				client.Update{Path: client.Path{"a[b=]"}, Val: 1, TS: time.Unix(0, 2)},
			},
			wantErr: true,
		},
		{
			desc: "bad notification type",
			notifs: []client.Notification{
				client.Sync{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ProtoResponse(tt.notifs...)
			if err != nil && !tt.wantErr {
				t.Fatalf("got error %v, want nil", err)
			}
			if err == nil && tt.wantErr {
				t.Fatal("got nil error, want non-nil")
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(got, tt.want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("got/want diff:\n%s", diff)
			}
		})
	}
}
