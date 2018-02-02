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

package client_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"context"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	fclient "github.com/openconfig/gnmi/client/fake"
)

const (
	defaultQuery = "*"
)

func testImpl(ctx context.Context, d client.Destination) (client.Impl, error) {
	if len(d.Addrs) > 0 && d.Addrs[0] == "error" {
		return nil, fmt.Errorf("error")
	}
	return fclient.New(ctx, d)
}

func TestRegister(t *testing.T) {
	// In case some other test forgot, clean out registered impls.
	client.ResetRegisteredImpls()
	// Verify Reset
	if got := client.RegisteredImpls(); got != nil {
		t.Fatalf("client.ResetRegisteredImpls() failed: got %v want nil", got)
	}
	// Clean out what we registered.
	defer client.ResetRegisteredImpls()

	// Registered names must not be reused unless you expect an error in duplicate
	tests := []struct {
		desc       string
		name       string
		f          client.InitImpl
		clientType []string
		rErr       bool
		nErr       bool
	}{{
		desc: "Missing Impl",
		name: "foo",
		rErr: true,
	}, {
		desc: "No registration, unspecified client",
		nErr: true,
	}, {
		desc: "Name only",
		name: "foo",
		rErr: true,
	}, {
		desc:       "Valid Client",
		name:       "bar",
		f:          testImpl,
		clientType: []string{"bar"},
		nErr:       false,
	}, {
		desc: "Unspecified client with prior registeration",
		nErr: false,
	}, {
		desc:       "Duplicate Registration",
		name:       "bar",
		f:          testImpl,
		clientType: []string{"bar"},
		rErr:       true,
	}, {
		desc:       "Unknown Registration",
		name:       "foobar",
		f:          testImpl,
		clientType: []string{"zbaz"},
		nErr:       true,
	}, {
		desc:       "Multiple clients, one valid",
		clientType: []string{"zbaz", "bar"},
		nErr:       false,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.name != "" {
				err := client.Register(tt.name, tt.f)
				switch {
				case tt.rErr && err == nil:
					t.Fatalf("Register(%q, %v) unexpected success", tt.name, tt.f)
				case !tt.rErr && err != nil:
					t.Fatalf("Register(%q, %v) failed: %v", tt.name, tt.f, err)
				case tt.rErr && err != nil:
					return
				}
			}
			err := client.New().Subscribe(context.Background(), client.Query{
				Type:                client.Once,
				Addrs:               []string{"fake"},
				Queries:             []client.Path{{"*"}},
				NotificationHandler: func(client.Notification) error { return nil },
			}, tt.clientType...)
			switch {
			case tt.nErr && err == nil:
				t.Fatalf("Subscribe() unexpected success")
			case !tt.nErr && err != nil:
				t.Fatalf("Subscribe() failed: %v", err)
			case tt.nErr && err != nil:
				return
			}
		})
	}
}

func TestRegisterHangingImpl(t *testing.T) {
	// This test makes sure that a hanging client.NewImpl (due to blocked
	// InitImpl, e.g. waiting for timeout) doesn't prevent other client.NewImpl
	// calls from blocking too. This may happen due to a global mutex in
	// register.go

	// In case some other test forgot, clean out registered impls.
	client.ResetRegisteredImpls()
	// Clean out what we registered.
	defer client.ResetRegisteredImpls()

	blocked := make(chan struct{})
	client.Register("blocking", func(ctx context.Context, _ client.Destination) (client.Impl, error) {
		close(blocked)
		// Block until test returns.
		<-ctx.Done()
		return nil, nil
	})
	client.Register("regular", func(context.Context, client.Destination) (client.Impl, error) {
		return nil, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connected := make(chan string, 1)
	go func() {
		client.NewImpl(ctx, client.Destination{}, "blocking")
		connected <- "blocking"
	}()
	go func() {
		// Wait for blocking Impl to start blocking.
		<-blocked
		client.NewImpl(ctx, client.Destination{}, "regular")
		connected <- "regular"
	}()

	select {
	case got := <-connected:
		if got != "regular" {
			t.Errorf(`connected Impl %q, want "regular"`, got)
		}
	case <-time.After(5 * time.Second):
		t.Error("blocking InitImpl prevents regular InitImpl from connecting (waiter 5s)")
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		desc     string
		in       client.Query
		wantPath []client.Path
		err      bool
		client   []string
	}{{
		desc:     "Empty Query",
		in:       client.Query{},
		err:      true,
		wantPath: []client.Path{{defaultQuery}},
	}, {
		desc:     "No Addr",
		wantPath: []client.Path{{defaultQuery}},
		in: client.Query{
			Queries: []client.Path{{"foo", "bar"}, {"a", "b"}},
			Type:    client.Once,
		},
		err: true,
	}, {
		desc:     "No Target",
		wantPath: []client.Path{{defaultQuery}},
		in: client.Query{
			Addrs:   []string{"fake addr"},
			Queries: []client.Path{{"foo", "bar"}, {"a", "b"}},
			Type:    client.Once,
		},
		err: true,
	}, {
		desc:     "No Type",
		wantPath: []client.Path{{defaultQuery}},
		in: client.Query{
			Addrs:   []string{"fake addr"},
			Target:  "",
			Queries: []client.Path{{"foo", "bar"}, {"a", "b"}},
		},
		err: true,
	}, {
		desc:     "No Queries",
		wantPath: []client.Path{{defaultQuery}},
		in: client.Query{
			Addrs:  []string{"fake addr"},
			Target: "",
			Type:   client.Once,
		},
		err: true,
	}, {
		desc:     "Both handlers set",
		wantPath: []client.Path{{"foo", "bar"}, {"a", "b"}},
		in: client.Query{
			Addrs:               []string{"fake addr"},
			Target:              "",
			Queries:             []client.Path{{"foo", "bar"}, {"a", "b"}},
			Type:                client.Once,
			NotificationHandler: func(_ client.Notification) error { return nil },
			ProtoHandler:        func(_ proto.Message) error { return nil },
		},
		err: true,
	}, {
		desc:     "Valid Query",
		wantPath: []client.Path{{"foo", "bar"}, {"a", "b"}},
		in: client.Query{
			Addrs:               []string{"fake addr"},
			Target:              "",
			Queries:             []client.Path{{"foo", "bar"}, {"a", "b"}},
			Type:                client.Once,
			NotificationHandler: func(_ client.Notification) error { return nil },
		},
	}, {
		desc: "Password contains forbidden characters",
		in: client.Query{
			Credentials: &client.Credentials{Password: "\n"},
		},
		err: true,
	}, {
		desc: "Username contains forbidden characters",
		in: client.Query{
			Credentials: &client.Credentials{Username: "\n"},
		},
		err: true,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.in.Validate()
			switch {
			case err != nil && tt.err:
				return
			case err != nil && !tt.err:
				t.Errorf("Validate() failed: %v", err)
				return
			case err == nil && tt.err:
				t.Errorf("Validate() expected error.")
				return
			}
			if !reflect.DeepEqual(tt.wantPath, tt.in.Queries) {
				t.Errorf("Validate() failed: got %v, want %v", tt.in.Queries, tt.wantPath)
			}
		})
	}
}

func TestLeaves(t *testing.T) {
	tests := []struct {
		desc string
		in   client.Leaves
		want client.Leaves
	}{{
		desc: "sorted",
		in: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b", "c"}, Val: 1},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b", "c"}, Val: 1},
		},
	}, {
		desc: "unsorted",
		in: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 1},
		},
	}, {
		desc: "stable",
		in: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 3},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 3},
		},
	}, {
		desc: "nil path",
		in: client.Leaves{
			{TS: time.Unix(0, 0), Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 3},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"b", "b", "c"}, Val: 1},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 2},
			{TS: time.Unix(0, 0), Path: client.Path{"c"}, Val: 3},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := make(client.Leaves, len(tt.in))
			copy(got, tt.in)
			sort.Sort(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("sort.Sort(%v) failed: got %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPath(t *testing.T) {
	tests := []struct {
		desc string
		in   client.Path
		cmp  client.Path
		want bool
	}{{
		desc: "same",
		in:   client.Path{"a", "b", "c"},
		cmp:  client.Path{"a", "b", "c"},
		want: true,
	}, {
		desc: "different length",
		in:   client.Path{"a", "b", "c"},
		cmp:  client.Path{"a", "b"},
		want: false,
	}, {
		desc: "different",
		in:   client.Path{"a", "b", "c"},
		cmp:  client.Path{"a", "b", "d"},
		want: false,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.in.Equal(tt.cmp); got != tt.want {
				t.Fatalf("%+v.Equal(%+v) failed: got %v, want %v", tt.in, tt.cmp, got, tt.want)
			}
		})
	}
}

func TestNewType(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want client.Type
	}{{
		desc: "Unknown",
		in:   "foo",
		want: client.Unknown,
	}, {
		desc: "Once",
		in:   "once",
		want: client.Once,
	}, {
		desc: "Stream",
		in:   "stream",
		want: client.Stream,
	}, {
		desc: "Poll",
		in:   "poll",
		want: client.Poll,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := client.NewType(tt.in); got != tt.want {
				t.Fatalf("client.NewType(%+v) failed: got %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestError(t *testing.T) {
	want := "foo"
	e := client.NewError(want)
	if got := e.Error(); got != want {
		t.Errorf("client.NewError(%q) failed: got %v, want %v", want, got, want)
	}
}

func TestClientUpdatesAfterClose(t *testing.T) {
	client.ResetRegisteredImpls()

	fake := fakeStreamingClient{ch: make(chan struct{})}
	client.Register("fake", func(context.Context, client.Destination) (client.Impl, error) {
		return fake, nil
	})

	c := &client.BaseClient{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := c.Subscribe(context.Background(), client.Query{
			Addrs:               []string{"fake"},
			Type:                client.Stream,
			Queries:             []client.Path{{"*"}},
			NotificationHandler: func(client.Notification) error { return nil },
		}, "fake")
		if err != nil {
			t.Errorf("Subscribe(): %v", err)
		}
	}()

	for i := 0; i < 10; i++ {
		select {
		case fake.ch <- struct{}{}:
		case <-done:
			t.Fatal("Subscribe returned before close")
		}
	}
	c.Close()

	var updatesAfterClose int
loop:
	for {
		select {
		case fake.ch <- struct{}{}:
			updatesAfterClose++
		case <-done:
			break loop
		}
	}
	if updatesAfterClose > 1 {
		t.Errorf("got %d updates after Close, expect at most 1", updatesAfterClose)
	}
}

type fakeStreamingClient struct {
	client.Impl
	ch chan struct{}
}

func (f fakeStreamingClient) Subscribe(context.Context, client.Query) error {
	return nil
}

func (f fakeStreamingClient) Recv() error {
	<-f.ch
	return nil
}

func (f fakeStreamingClient) Close() error { return nil }

// Once client will run the query and once complete you can act on the
// returned tree.
func ExampleClient_Once() {
	q := client.Query{
		Addrs:   []string{"127.0.0.1:1234"},
		Target:  "dev",
		Queries: []client.Path{{"*"}},
		Type:    client.Once,
	}
	c := client.New()
	defer c.Close()
	err := c.Subscribe(context.Background(), q)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range c.Leaves() {
		fmt.Printf("%v: %v\n", v.Path, v.Val)
	}
}

// Poll client is like Once client, but can be re-triggered via Poll to
// re-execute the query.
func ExampleClient_Poll() {
	q := client.Query{
		Addrs:   []string{"127.0.0.1:1234"},
		Target:  "dev",
		Queries: []client.Path{{"*"}},
		Type:    client.Poll,
	}
	c := client.New()
	defer c.Close()
	err := c.Subscribe(context.Background(), q)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range c.Leaves() {
		fmt.Printf("%v: %v\n", v.Path, v.Val)
	}
	err = c.Poll() // Poll allows the underyling Query to keep running
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range c.Leaves() {
		fmt.Printf("%v: %v\n", v.Path, v.Val)
	}
}

// Stream client returns the current state for the query and keeps running
// until closed or the underlying connection breaks.
func ExampleClient_Stream() {
	q := client.Query{
		Addrs:   []string{"127.0.0.1:1234"},
		Target:  "dev",
		Queries: []client.Path{{"*"}},
		Type:    client.Stream,
		NotificationHandler: func(n client.Notification) error {
			switch nn := n.(type) {
			case client.Connected:
				fmt.Println("client is connected")
			case client.Sync:
				fmt.Println("client is synced")
			case client.Update, client.Delete:
				fmt.Printf("update: %+v\n", nn)
			case client.Error:
				fmt.Printf("error: %v\n", nn)
			}
			return nil
		},
	}
	c := client.New()
	defer c.Close()
	// Note that Subscribe will block.
	err := c.Subscribe(context.Background(), q)
	if err != nil {
		fmt.Println(err)
	}
}
