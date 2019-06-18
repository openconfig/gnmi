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

package subscribe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"github.com/openconfig/gnmi/cache"
	"github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/path"
	"github.com/openconfig/gnmi/testing/fake/testing/grpc/config"
	"github.com/openconfig/gnmi/value"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

func startServer(targets []string) (string, *cache.Cache, func(), error) {
	c := cache.New(targets)
	p, err := NewServer(c)
	if err != nil {
		return "", nil, nil, fmt.Errorf("NewServer: %v", err)
	}
	c.SetClient(p.Update)
	lis, err := net.Listen("tcp", "")
	if err != nil {
		return "", nil, nil, fmt.Errorf("net.Listen: %v", err)
	}
	opt, err := config.WithSelfTLSCert()
	if err != nil {
		return "", nil, nil, fmt.Errorf("config.WithSelfCert: %v", err)
	}
	srv := grpc.NewServer(opt)
	pb.RegisterGNMIServer(srv, p)
	go srv.Serve(lis)
	return lis.Addr().String(), p.c, func() {
		lis.Close()
	}, nil
}

func TestOnce(t *testing.T) {
	addr, cache, teardown, err := startServer(client.Path{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev1", "f", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	testCases := []struct {
		dev   string
		query client.Path
		count int
		err   bool
	}{
		// These cases will be found.
		{"dev1", client.Path{"a", "b"}, 1, false},
		{"dev1", client.Path{"a"}, 2, false},
		{"dev1", client.Path{"e"}, 1, false},
		{"dev1", client.Path{"*", "b"}, 2, false},
		// This case is not found.
		{"dev1", client.Path{"b"}, 0, false},
		// This target doesn't even exist, and will return an error.
		{"dev2", client.Path{"a"}, 0, true},
	}
	for _, tt := range testCases {
		t.Run(fmt.Sprintf("target: %q query: %q", tt.dev, tt.query), func(t *testing.T) {
			sync := 0
			count := 0
			q := client.Query{
				Addrs:   []string{addr},
				Target:  tt.dev,
				Queries: []client.Path{tt.query},
				Type:    client.Once,
				NotificationHandler: func(n client.Notification) error {
					switch n.(type) {
					case client.Update:
						count++
					case client.Sync:
						sync++
					case client.Connected:
					default:
						t.Fatalf("unexpected notification %#v", n)
					}
					return nil
				},
				TLS: &tls.Config{InsecureSkipVerify: true},
			}
			c := client.BaseClient{}
			err := c.Subscribe(context.Background(), q, gnmiclient.Type)
			defer c.Close()
			if err != nil && !tt.err {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tt.err {
				t.Fatal("didn't get expected error")
			}
			if tt.err {
				return
			}
			if sync != 1 {
				t.Errorf("got %d sync messages, want 1", sync)
			}
			if count != tt.count {
				t.Errorf("got %d updates, want %d", count, tt.count)
			}
		})
	}
}

func TestOriginInSubscribeRequest(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer(client.Path{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b", "c", "d"},
		{"dev1", "a", "b", "d", "e"},
		{"dev1", "a", "c", "d", "e"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	tests := []struct {
		desc      string
		inPrefix  *pb.Path
		inPath    *pb.Path
		wantCount int
		wantErr   bool
	}{
		{
			desc:      "no origin set",
			inPrefix:  &pb.Path{Target: "dev1", Elem: []*pb.PathElem{{Name: "a"}}},
			inPath:    &pb.Path{Elem: []*pb.PathElem{}},
			wantCount: 3,
		},
		{
			desc:      "origin set in prefix",
			inPrefix:  &pb.Path{Target: "dev1", Origin: "a", Elem: []*pb.PathElem{{Name: "b"}}},
			inPath:    &pb.Path{Elem: []*pb.PathElem{}},
			wantCount: 2,
		},
		{
			desc:     "origin set in path with path elements in prefix",
			inPrefix: &pb.Path{Target: "dev1", Elem: []*pb.PathElem{{Name: "a"}}},
			inPath:   &pb.Path{Origin: "b"},
			wantErr:  true,
		},
		{
			desc:      "origin set in path",
			inPrefix:  &pb.Path{Target: "dev1"},
			inPath:    &pb.Path{Origin: "a", Elem: []*pb.PathElem{{Name: "b"}}},
			wantCount: 2,
		},
		{
			desc:     "origin set in path and prefix",
			inPrefix: &pb.Path{Target: "dev1", Origin: "a", Elem: []*pb.PathElem{{Name: "b"}}},
			inPath:   &pb.Path{Origin: "c"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s: prefix %v and path %v", tt.desc, tt.inPrefix, tt.inPath), func(t *testing.T) {
			sync := 0
			count := 0
			q, err := client.NewQuery(&pb.SubscribeRequest{
				Request: &pb.SubscribeRequest_Subscribe{
					Subscribe: &pb.SubscriptionList{
						Prefix:       tt.inPrefix,
						Subscription: []*pb.Subscription{{Path: tt.inPath}},
						Mode:         pb.SubscriptionList_ONCE,
					},
				},
			})
			if err != nil {
				t.Fatalf("failed to initialize a client.Query: %v", err)
			}
			q.ProtoHandler = func(msg proto.Message) error {
				resp, ok := msg.(*pb.SubscribeResponse)
				if !ok {
					return fmt.Errorf("failed to type assert message %#v", msg)
				}
				switch v := resp.Response.(type) {
				case *pb.SubscribeResponse_Update:
					count++
				case *pb.SubscribeResponse_Error:
					return fmt.Errorf("error in response: %s", v)
				case *pb.SubscribeResponse_SyncResponse:
					sync++
				default:
					return fmt.Errorf("unknown response %T: %s", v, v)
				}

				return nil
			}
			q.TLS = &tls.Config{InsecureSkipVerify: true}
			q.Addrs = []string{addr}

			c := client.BaseClient{}
			err = c.Subscribe(context.Background(), q, gnmiclient.Type)
			defer c.Close()
			if tt.wantErr {
				if err == nil {
					t.Fatal("got nil, want err")
				}
				return
			}
			if err != nil {
				t.Fatalf("got %v, want no error", err)
			}
			if sync != 1 {
				t.Errorf("got %d sync messages, want 1", sync)
			}
			if count != tt.wantCount {
				t.Errorf("got %d updates, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestGNMIOnce(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer(client.Path{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev1", "f", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	testCases := []struct {
		dev   string
		query client.Path
		count int
		err   bool
	}{
		// These cases will be found.
		{"dev1", client.Path{"a", "b"}, 1, false},
		{"dev1", client.Path{"a"}, 2, false},
		{"dev1", client.Path{"e"}, 1, false},
		{"dev1", client.Path{"*", "b"}, 2, false},
		// This case is not found.
		{"dev1", client.Path{"b"}, 0, false},
		// This target doesn't even exist, and will return an error.
		{"dev2", client.Path{"a"}, 0, true},
	}
	for _, tt := range testCases {
		t.Run(fmt.Sprintf("target: %q query: %q", tt.dev, tt.query), func(t *testing.T) {
			sync := 0
			count := 0
			q := client.Query{
				Addrs:   []string{addr},
				Target:  tt.dev,
				Queries: []client.Path{tt.query},
				Type:    client.Once,
				ProtoHandler: func(msg proto.Message) error {
					resp, ok := msg.(*pb.SubscribeResponse)
					if !ok {
						return fmt.Errorf("failed to type assert message %#v", msg)
					}
					switch v := resp.Response.(type) {
					case *pb.SubscribeResponse_Update:
						count++
					case *pb.SubscribeResponse_Error:
						return fmt.Errorf("error in response: %s", v)
					case *pb.SubscribeResponse_SyncResponse:
						sync++
					default:
						return fmt.Errorf("unknown response %T: %s", v, v)
					}

					return nil
				},
				TLS: &tls.Config{InsecureSkipVerify: true},
			}
			c := client.BaseClient{}
			err := c.Subscribe(context.Background(), q, gnmiclient.Type)
			defer c.Close()
			if err != nil && !tt.err {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tt.err {
				t.Fatal("didn't get expected error")
			}
			if tt.err {
				return
			}
			if sync != 1 {
				t.Errorf("got %d sync messages, want 1", sync)
			}
			if count != tt.count {
				t.Errorf("got %d updates, want %d", count, tt.count)
			}
		})
	}
}

// sendUpdates generates an update for each supplied path incrementing the
// timestamp and value for each.
func sendUpdates(t *testing.T, c *cache.Cache, paths []client.Path, timestamp *time.Time) {
	t.Helper()
	switch cache.Type {
	case cache.ClientLeaf:
		for _, path := range paths {
			*timestamp = timestamp.Add(time.Nanosecond)
			if err := c.Update(client.Update{Path: path, Val: timestamp.UnixNano(), TS: *timestamp}); err != nil {
				t.Errorf("streamUpdate: %v", err)
			}
		}
	case cache.GnmiNoti:
		for _, path := range paths {
			*timestamp = timestamp.Add(time.Nanosecond)
			sv, err := value.FromScalar(timestamp.UnixNano())
			if err != nil {
				t.Errorf("Scalar value err %v", err)
				continue
			}
			noti := &pb.Notification{
				Prefix:    &pb.Path{Target: path[0]},
				Timestamp: timestamp.UnixNano(),
				Update: []*pb.Update{
					{
						Path: &pb.Path{Element: path[1:]},
						Val:  sv,
					},
				},
			}
			if err := c.GnmiUpdate(noti); err != nil {
				t.Errorf("streamUpdate: %v", err)
			}
		}
	}
}

func TestPoll(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	m := map[string]time.Time{}
	sync := 0
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Poll,
		NotificationHandler: func(n client.Notification) error {
			switch u := n.(type) {
			case client.Update:
				count++
				v, ts := strings.Join(u.Path, "/"), u.TS
				want1, want2 := "dev1/a/b", "dev1/a/c"
				if v != want1 && v != want2 {
					t.Fatalf("#%d: got %q, want one of (%q, %q)", count, v, want1, want2)
				}
				if ts.Before(m[v]) {
					t.Fatalf("#%d: got timestamp %s, want >= %s for value %q", count, ts, m[v], v)
				}
				m[v] = ts
			case client.Sync:
				if count != 2 {
					t.Fatalf("did not receive initial updates before sync, got %d, want 2", count)
				}
				count = 0
				sync++
				if sync == 3 {
					c.Close()
				} else {
					sendUpdates(t, cache, paths, &timestamp)
					c.Poll()
				}
			case client.Connected:
			default:
				t.Fatalf("#%d: unexpected notification %#v", count, n)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
}

func TestGNMIPoll(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	// The streaming Updates change only the timestamp, so the value is used as
	// a key.
	m := map[string]time.Time{}
	sync := 0
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Poll,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				count++
				ts := time.Unix(0, r.Update.GetTimestamp())
				v := strings.Join(path.ToStrings(r.Update.Update[0].Path, false), "/")
				if ts.Before(m[v]) {
					t.Fatalf("#%d: got timestamp %s, want >= %s for value %q", count, ts, m[v], v)
				}
				m[v] = ts
			case *pb.SubscribeResponse_Error:
				return fmt.Errorf("error in response: %s", r)
			case *pb.SubscribeResponse_SyncResponse:
				if count != 2 {
					t.Fatalf("did not receive initial updates before sync, got %d, want 2", count)
				}
				count = 0
				sync++
				if sync == 3 {
					c.Close()
				} else {
					sendUpdates(t, cache, paths, &timestamp)
					c.Poll()
				}
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
}

func TestStream(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	m := map[string]time.Time{}
	sync := false
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch u := n.(type) {
			case client.Update:
				count++
				// The total updates received should be 4, 2 before sync, 2 after.
				if count == 4 {
					c.Close()
				}
				v, ts := strings.Join(u.Path, "/"), u.TS
				want1, want2 := "dev1/a/b", "dev1/a/c"
				if v != want1 && v != want2 {
					t.Fatalf("#%d: got %q, want one of (%q, %q)", count, v, want1, want2)
				}
				if ts.Before(m[v]) {
					t.Fatalf("#%d: got timestamp %s, want >= %s for value %q", count, ts, m[v], v)
				}
				m[v] = ts
			case client.Sync:
				if sync {
					t.Fatal("received more than one sync message")
				}
				if count < 2 {
					t.Fatalf("did not receive initial updates before sync, got %d, want > 2", count)
				}
				sync = true
				// Send some updates after the sync occurred.
				sendUpdates(t, cache, paths, &timestamp)
			case client.Connected:
			default:
				t.Fatalf("#%d: unexpected notification %#v", count, n)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
	if !sync {
		t.Error("streaming query did not send sync message")
	}
}

func TestGNMIStream(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	m := map[string]time.Time{}
	sync := false
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				count++
				// The total updates received should be 4, 2 before sync, 2 after.
				if count == 4 {
					c.Close()
				}
				v := strings.Join(path.ToStrings(r.Update.Update[0].Path, false), "/")
				ts := time.Unix(0, r.Update.GetTimestamp())
				if ts.Before(m[v]) {
					t.Fatalf("#%d: got timestamp %s, want >= %s for value %q", count, ts, m[v], v)
				}
				m[v] = ts
			case *pb.SubscribeResponse_Error:
				return fmt.Errorf("error in response: %s", r)
			case *pb.SubscribeResponse_SyncResponse:
				if sync {
					t.Fatal("received more than one sync message")
				}
				if count < 2 {
					t.Fatalf("did not receive initial updates before sync, got %d, want > 2", count)
				}
				sync = true
				// Send some updates after the sync occurred.
				sendUpdates(t, cache, paths, &timestamp)
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
	if !sync {
		t.Error("streaming query did not send sync message")
	}
}

func TestStreamNewUpdates(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	newpaths := []client.Path{
		{"dev1", "b", "d"},
		{"dev2", "a", "x"},
		{"dev1", "a", "x"}, // The update we want to see.
	}

	sync := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch u := n.(type) {
			case client.Update:
				v, want := strings.Join(u.Path, "/"), "dev1/a/x"
				if v != want {
					t.Fatalf("got update %q, want only %q", v, want)
				}
				c.Close()
			case client.Sync:
				if sync {
					t.Fatal("received more than one sync message")
				}
				sync = true
				// Stream new updates only after sync which should have had 0
				// updates.
				sendUpdates(t, cache, newpaths, &timestamp)
			case client.Connected:
			default:
				t.Fatalf("unexpected notification %#v", n)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
	if !sync {
		t.Error("streaming query did not send sync message")
	}
}

func TestGNMIStreamNewUpdates(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	newpaths := []client.Path{
		{"dev1", "b", "d"},
		{"dev2", "a", "x"},
		{"dev1", "a", "x"}, // The update we want to see.
	}

	sync := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				v, want := strings.Join(path.ToStrings(r.Update.Update[0].Path, false), "/"), "a/x"
				if v != want {
					t.Fatalf("got update %q, want only %q", v, want)
				}
				c.Close()
			case *pb.SubscribeResponse_SyncResponse:
				if sync {
					t.Fatal("received more than one sync message")
				}
				sync = true
				// Stream new updates only after sync which should have had 0
				// updates.
				sendUpdates(t, cache, newpaths, &timestamp)
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
	if !sync {
		t.Error("streaming query did not send sync message")
	}
}

func TestUpdatesOnly(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	sync := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch u := n.(type) {
			case client.Update:
				if !sync {
					t.Errorf("got update %v before sync", u)
				}
				c.Close()
			case client.Sync:
				sync = true
				sendUpdates(t, cache, paths, &timestamp)
			case client.Connected:
			default:
				t.Fatalf("unexpected notification %#v", n)
			}
			return nil
		},
		TLS:         &tls.Config{InsecureSkipVerify: true},
		UpdatesOnly: true,
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
}

func TestGNMIUpdatesOnly(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths, &timestamp)

	sync := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				if !sync {
					t.Errorf("got update %v before sync", r)
				}
				c.Close()
			case *pb.SubscribeResponse_SyncResponse:
				sync = true
				sendUpdates(t, cache, paths, &timestamp)
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS:         &tls.Config{InsecureSkipVerify: true},
		UpdatesOnly: true,
	}
	err = c.Subscribe(context.Background(), q, gnmiclient.Type)
	if err != nil {
		t.Error(err)
	}
}

// If a client doesn't read any of the responses, it should not affect other
// clients querying the same target.
func TestSubscribeUnresponsiveClient(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
	}
	sendUpdates(t, cache, paths, &time.Time{})

	// Start the first client and do *not* read any responses.
	started := make(chan struct{})
	stall := make(chan struct{})
	defer close(stall)
	client1 := client.BaseClient{}
	defer client1.Close()
	q1 := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			select {
			case <-started:
			default:
				close(started)
			}
			<-stall
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	ctx := context.Background()
	go client1.Subscribe(ctx, q1, gnmiclient.Type)
	// Wait for client1 to start.
	<-started

	// Start the second client for the same target and actually accept
	// responses.
	count := 0
	client2 := client.BaseClient{}
	q2 := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch n.(type) {
			case client.Update:
				count++
			case client.Sync:
				client2.Close()
			case client.Connected:
			default:
				t.Fatalf("unexpected notification %#v", n)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := client2.Subscribe(ctx, q2, gnmiclient.Type); err != nil {
		t.Errorf("client2.Subscribe: %v", err)
	}
	if count != 2 {
		t.Errorf("client2.Subscribe got %d updates, want 2", count)
	}
}

// If a client doesn't read any of the responses, it should not affect other
// clients querying the same target.
func TestGNMISubscribeUnresponsiveClient(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, cache, teardown, err := startServer([]string{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
	}
	sendUpdates(t, cache, paths, &time.Time{})

	// Start the first client and do *not* read any responses.
	started := make(chan struct{})
	stall := make(chan struct{})
	defer close(stall)
	client1 := client.BaseClient{}
	defer client1.Close()
	q1 := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			select {
			case <-started:
			default:
				close(started)
			}
			<-stall
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	ctx := context.Background()
	go client1.Subscribe(ctx, q1, gnmiclient.Type)
	// Wait for client1 to start.
	<-started

	// Start the second client for the same target and actually accept
	// responses.
	count := 0
	client2 := client.BaseClient{}
	q2 := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				count++
			case *pb.SubscribeResponse_SyncResponse:
				client2.Close()
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := client2.Subscribe(ctx, q2, gnmiclient.Type); err != nil {
		t.Errorf("client2.Subscribe: %v", err)
	}
	if count != 2 {
		t.Errorf("client2.Subscribe got %d updates, want 2", count)
	}
}

func TestDeletedTargetMessage(t *testing.T) {
	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	sendUpdates(t, cache, paths, &time.Time{})

	deleted := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch v := n.(type) {
			case client.Update:
			case client.Sync:
				cache.Remove("dev1")
			case client.Delete:
				// Want to see a target delete message.  No need to call c.Close()
				// because the server should close the connection if the target is
				// removed.
				if len(v.Path) == 1 && v.Path[0] == "dev1" {
					deleted = true
				}
			case client.Connected:
			default:
				t.Fatalf("unexpected notification %#v", n)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := c.Subscribe(context.Background(), q, gnmiclient.Type); err != nil {
		t.Errorf("c.Subscribe: %v", err)
	}
	if !deleted {
		t.Error("Target delete not sent.")
	}
}

func TestGNMIDeletedTargetMessage(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	addr, ch, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	sendUpdates(t, ch, paths, &time.Time{})

	deleted := false
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				if len(r.Update.Delete) > 0 {
					// Want to see a target delete message.  No need to call c.Close()
					// because the server should close the connection if the target is
					// removed.
					if r.Update.Prefix.GetTarget() == "dev1" {
						deleted = true
					}
				}
			case *pb.SubscribeResponse_SyncResponse:
				ch.Remove("dev1")
			default:
				return fmt.Errorf("unknown response %T: %s", r, r)
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := c.Subscribe(context.Background(), q, gnmiclient.Type); err != nil {
		t.Errorf("c.Subscribe: %v", err)
	}
	if !deleted {
		t.Error("Target delete not sent.")
	}
}

func TestCoalescedDupCount(t *testing.T) {
	// Inject a simulated flow control to block sends and induce coalescing.
	flowControlTest = func() { time.Sleep(100 * time.Microsecond) }
	addr, cache, teardown, err := startServer([]string{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	stall := make(chan struct{})
	done := make(chan struct{})
	coalesced := uint32(0)
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch u := n.(type) {
			case client.Update:
				count++
				if u.Dups > 0 {
					coalesced = u.Dups
				}
				switch count {
				case 1:
					close(stall)
				case 2:
					close(done)
				}
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Subscribe(ctx, q, gnmiclient.Type)

	paths := []client.Path{
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths[0:1], &timestamp)
	<-stall
	sendUpdates(t, cache, paths, &timestamp)
	<-done

	if want := uint32(len(paths) - 1); coalesced != want {
		t.Errorf("got coalesced count %d, want %d", coalesced, want)
	}
}

func TestGNMICoalescedDupCount(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	// Inject a simulated flow control to block sends and induce coalescing.
	flowControlTest = func() { time.Sleep(100 * time.Microsecond) }
	addr, cache, teardown, err := startServer([]string{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	stall := make(chan struct{})
	done := make(chan struct{})
	coalesced := uint32(0)
	count := 0
	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch r := resp.Response.(type) {
			case *pb.SubscribeResponse_Update:
				count++
				if r.Update.Update[0].GetDuplicates() > 0 {
					coalesced = r.Update.Update[0].GetDuplicates()
				}
				switch count {
				case 1:
					close(stall)
				case 2:
					close(done)
				}
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Subscribe(ctx, q, gnmiclient.Type)

	paths := []client.Path{
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
		{"dev1", "a"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths[0:1], &timestamp)
	<-stall
	sendUpdates(t, cache, paths, &timestamp)
	<-done

	if want := uint32(len(paths) - 1); coalesced != want {
		t.Errorf("got coalesced count %d, want %d", coalesced, want)
	}
}

func TestSubscribeTimeout(t *testing.T) {
	// Set a low timeout that is below the induced flowControl delay.
	Timeout = 100 * time.Millisecond
	// Cause query to hang indefinitely to induce timeout.
	flowControlTest = func() { select {} }
	// Reset the global variables so as not to interfere with other tests.
	defer func() {
		Timeout = time.Minute
		flowControlTest = func() {}
	}()

	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	sendUpdates(t, cache, paths, &time.Time{})

	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := c.Subscribe(context.Background(), q, gnmiclient.Type); err == nil {
		t.Error("c.Subscribe got nil, wanted a timeout err")
	}
}

func TestGNMISubscribeTimeout(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	// Set a low timeout that is below the induced flowControl delay.
	Timeout = 100 * time.Millisecond
	// Cause query to hang indefinitely to induce timeout.
	flowControlTest = func() { select {} }
	// Reset the global variables so as not to interfere with other tests.
	defer func() {
		Timeout = time.Minute
		flowControlTest = func() {}
	}()

	addr, cache, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	paths := []client.Path{
		{"dev1", "a", "b"},
		{"dev1", "a", "c"},
		{"dev1", "e", "f"},
		{"dev2", "a", "b"},
	}
	sendUpdates(t, cache, paths, &time.Time{})

	c := client.BaseClient{}
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Stream,
		ProtoHandler: func(msg proto.Message) error {
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}
	if err := c.Subscribe(context.Background(), q, gnmiclient.Type); err == nil {
		t.Error("c.Subscribe got nil, wanted a timeout err")
	}
}

func TestSubscriptionLimit(t *testing.T) {
	totalQueries := 20
	SubscriptionLimit = 7
	causeLimit := make(chan struct{})
	subscriptionLimitTest = func() {
		<-causeLimit
	}
	// Clear the global variables so as not to interfere with other tests.
	defer func() {
		SubscriptionLimit = 0
		subscriptionLimitTest = func() {}
	}()

	addr, _, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	fc := make(chan struct{})
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Once,
		NotificationHandler: func(n client.Notification) error {
			switch n.(type) {
			case client.Sync:
				fc <- struct{}{}
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}

	// Launch parallel queries.
	for i := 0; i < totalQueries; i++ {
		c := client.BaseClient{}
		go c.Subscribe(context.Background(), q, gnmiclient.Type)
	}

	timeout := time.After(500 * time.Millisecond)
	finished := 0
firstQueries:
	for {
		select {
		case <-fc:
			finished++
		case <-timeout:
			break firstQueries
		}
	}
	if finished != SubscriptionLimit {
		t.Fatalf("got %d finished queries, want %d", finished, SubscriptionLimit)
	}

	close(causeLimit)
	timeout = time.After(time.Second)
remainingQueries:
	for {
		select {
		case <-fc:
			if finished++; finished == totalQueries {
				break remainingQueries
			}
		case <-timeout:
			t.Errorf("Remaining queries did not proceed after limit removed. got %d, want %d", finished, totalQueries)
		}
	}
}

func TestGNMISubscriptionLimit(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	totalQueries := 20
	SubscriptionLimit = 7
	causeLimit := make(chan struct{})
	subscriptionLimitTest = func() {
		<-causeLimit
	}
	// Clear the global variables so as not to interfere with other tests.
	defer func() {
		SubscriptionLimit = 0
		subscriptionLimitTest = func() {}
	}()

	addr, _, teardown, err := startServer([]string{"dev1", "dev2"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	fc := make(chan struct{})
	q := client.Query{
		Addrs:   []string{addr},
		Target:  "dev1",
		Queries: []client.Path{{"a"}},
		Type:    client.Once,
		ProtoHandler: func(msg proto.Message) error {
			resp, ok := msg.(*pb.SubscribeResponse)
			if !ok {
				return fmt.Errorf("failed to type assert message %#v", msg)
			}
			switch resp.Response.(type) {
			case *pb.SubscribeResponse_SyncResponse:
				fc <- struct{}{}
			}
			return nil
		},
		TLS: &tls.Config{InsecureSkipVerify: true},
	}

	// Launch parallel queries.
	for i := 0; i < totalQueries; i++ {
		c := client.BaseClient{}
		go c.Subscribe(context.Background(), q, gnmiclient.Type)
	}

	timeout := time.After(500 * time.Millisecond)
	finished := 0
firstQueries:
	for {
		select {
		case <-fc:
			finished++
		case <-timeout:
			break firstQueries
		}
	}
	if finished != SubscriptionLimit {
		t.Fatalf("got %d finished queries, want %d", finished, SubscriptionLimit)
	}

	close(causeLimit)
	timeout = time.After(time.Second)
remainingQueries:
	for {
		select {
		case <-fc:
			if finished++; finished == totalQueries {
				break remainingQueries
			}
		case <-timeout:
			t.Errorf("Remaining queries did not proceed after limit removed. got %d, want %d", finished, totalQueries)
		}
	}
}

func TestGNMIMultipleSubscriberCoalescion(t *testing.T) {
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	// Inject a simulated flow control to block sends and induce coalescing.
	flowControlTest = func() { time.Sleep(time.Second) }
	addr, cache, teardown, err := startServer([]string{"dev1"})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	var wg sync.WaitGroup
	sc := 5
	wg.Add(sc)
	cr := make([]uint32, 0, sc)
	var mux sync.Mutex
	for i := 0; i < sc; i++ {
		c := client.BaseClient{}
		q := client.Query{
			Addrs:   []string{addr},
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Stream,
			ProtoHandler: func(msg proto.Message) error {
				resp, ok := msg.(*pb.SubscribeResponse)
				if !ok {
					return fmt.Errorf("failed to type assert message %#v", msg)
				}
				switch r := resp.Response.(type) {
				case *pb.SubscribeResponse_Update:
					mux.Lock()
					if r.Update.Update[0].GetDuplicates() > 0 {
						cr = append(cr, r.Update.Update[0].GetDuplicates())
					}
					mux.Unlock()
					wg.Done()
				}
				return nil
			},
			TLS: &tls.Config{InsecureSkipVerify: true},
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go c.Subscribe(ctx, q, gnmiclient.Type)
	}

	paths := []client.Path{
		{"dev1", "a"},
		{"dev1", "a"},
	}
	var timestamp time.Time
	sendUpdates(t, cache, paths[0:1], &timestamp)
	wg.Wait()
	wg.Add(sc)
	sendUpdates(t, cache, paths, &timestamp)
	wg.Wait()
	for i, d := range cr {
		if d != uint32(len(paths)-1) {
			t.Errorf("#%d got %d, expect %d duplicate count", i, d, uint32(len(paths)-1))
		}
	}
}

type fakeRPCACL struct {
	user string
	acl  ACL
}

func (r *fakeRPCACL) Check(dev string) bool {
	return r.acl.Check(r.user, dev)
}

type fakeACL struct {
	allow int
	deny  int
	check int
}
type fakeNet struct{}

func (n *fakeNet) Network() string {
	return "fake network"
}
func (n *fakeNet) String() string {
	return "127.0.0.1"
}

type userKey int

var uk userKey

func (a *fakeACL) NewRPCACL(ctx context.Context) (RPCACL, error) {
	u, ok := ctx.Value(uk).(string)
	if !ok {
		return nil, errors.New("no user field in ctx")
	}
	r := &fakeRPCACL{user: u, acl: a}
	return r, nil
}

func (a *fakeACL) Check(user string, dev string) bool {
	a.check++
	m := map[string]map[string]bool{
		"dev-pii":    {"user1": true, "user2": false},
		"dev-no-pii": {"user1": true, "user2": true},
	}
	if v, ok := m[dev]; ok {
		if v2, ok := v[user]; ok {
			if v2 {
				a.allow++
			} else {
				a.deny++
			}
			return v2
		}
	}
	a.deny++
	return false
}

type fakeSubServer struct {
	user string
	ctx  context.Context
	req  chan *pb.SubscribeRequest
	rsp  chan *pb.SubscribeResponse
	grpc.ServerStream
}

func (s *fakeSubServer) Send(rsp *pb.SubscribeResponse) error {
	select {
	case <-s.ctx.Done():
		return io.ErrClosedPipe
	default:
	}
	s.rsp <- rsp
	return nil
}

func (s *fakeSubServer) Recv() (*pb.SubscribeRequest, error) {
	select {
	case <-s.ctx.Done():
		return <-s.req, io.EOF
	default:
	}
	return <-s.req, nil
}

func (s *fakeSubServer) Context() context.Context {
	return s.ctx
}

func TestGNMIACL(t *testing.T) {
	targets := []string{"dev-pii", "dev-no-pii"}
	cache.Type = cache.GnmiNoti
	defer func() {
		cache.Type = cache.ClientLeaf
	}()
	c := cache.New(targets)
	p, err := NewServer(c)
	if err != nil {
		t.Errorf("NewServer: %v", err)
		return
	}
	paths := []client.Path{
		{"dev-pii", "a", "b"},
		{"dev-pii", "a", "c"},
		{"dev-no-pii", "e", "f"},
		{"dev-no-pii", "a", "b"},
	}
	var timestamp time.Time
	sendUpdates(t, c, paths, &timestamp)
	tests := []struct {
		name    string
		user    string
		dev     string
		mode    pb.SubscriptionList_Mode
		wantErr string
		wantCnt *fakeACL
	}{
		{
			name:    "user1 once with pii allow",
			user:    "user1",
			dev:     "dev-pii",
			mode:    pb.SubscriptionList_ONCE,
			wantErr: "<nil>",
			wantCnt: &fakeACL{allow: 3, deny: 0, check: 3},
		},
		{
			name:    "user2 once with pii deny",
			user:    "user2",
			dev:     "dev-pii",
			mode:    pb.SubscriptionList_ONCE,
			wantErr: "rpc error: code = PermissionDenied desc = not authorized for target \"dev-pii\"",
			wantCnt: &fakeACL{allow: 0, deny: 1, check: 1},
		},
		{
			name:    "user1 once all devices with pii allow",
			user:    "user1",
			dev:     "*",
			mode:    pb.SubscriptionList_ONCE,
			wantErr: "<nil>",
			wantCnt: &fakeACL{allow: 3, deny: 0, check: 3},
		},
		{
			name:    "user2 once all devices with pii deny",
			user:    "user2",
			dev:     "*",
			mode:    pb.SubscriptionList_ONCE,
			wantErr: "<nil>",
			wantCnt: &fakeACL{allow: 1, deny: 2, check: 3},
		},
		{
			name:    "user2 poll with pii deny",
			user:    "user2",
			dev:     "dev-pii",
			mode:    pb.SubscriptionList_POLL,
			wantErr: "rpc error: code = PermissionDenied desc = not authorized for target \"dev-pii\"",
			wantCnt: &fakeACL{allow: 0, deny: 1, check: 1},
		},
		{
			name:    "user2 stream with pii deny",
			user:    "user2",
			dev:     "dev-pii",
			mode:    pb.SubscriptionList_STREAM,
			wantErr: "rpc error: code = PermissionDenied desc = not authorized for target \"dev-pii\"",
			wantCnt: &fakeACL{allow: 0, deny: 1, check: 1},
		},
	}

	for _, test := range tests {
		facl := &fakeACL{}
		p.SetACL(facl)
		var cancel context.CancelFunc
		subSvr := &fakeSubServer{user: test.user,
			req: make(chan *pb.SubscribeRequest, 2),
			rsp: make(chan *pb.SubscribeResponse, len(paths)+1)}
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &fakeNet{}})
		subSvr.ctx = context.WithValue(ctx, uk, test.user)
		subSvr.ctx, cancel = context.WithCancel(subSvr.ctx)
		subSvr.req <- &pb.SubscribeRequest{
			Request: &pb.SubscribeRequest_Subscribe{
				Subscribe: &pb.SubscriptionList{
					Prefix: &pb.Path{
						Target: test.dev,
					},
					Subscription: []*pb.Subscription{
						&pb.Subscription{
							Path: &pb.Path{
								Element: []string{"a"},
								Elem: []*pb.PathElem{
									&pb.PathElem{
										Name: "a",
									},
								},
							},
						},
					},
					Mode: test.mode,
				},
			},
		}

		errCh := make(chan error)
		go func() {
			errCh <- p.Subscribe(subSvr)
		}()

		select {
		case err := <-errCh:
			got := fmt.Sprint(err)
			if diff := cmp.Diff(got, test.wantErr, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("%v returned unexpected result:\n got %v\n want %v\n diff %v", test.name, got, test.wantErr, diff)
			}
		case <-time.After(5 * time.Second):
		}
		cancel()

		if diff := cmp.Diff(facl, test.wantCnt, cmpopts.EquateEmpty(), cmp.AllowUnexported(fakeACL{})); diff != "" {
			t.Errorf("%v returned unexpected result:\n got %v\n want %v\n diff %v", test.name, facl, test.wantCnt, diff)
		}
	}
}
