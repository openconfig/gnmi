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
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"context"
	"github.com/openconfig/gnmi/client"
	fake "github.com/openconfig/gnmi/client/fake"
)

const (
	cacheTest = "cacheTest"
	cacheFail = "cacheFail"
)

func TestPollCache(t *testing.T) {
	tests := []struct {
		desc string
		q    client.Query
		u    [][]interface{}
		want []client.Leaves
		err  bool
	}{{
		desc: "invalid query type",
		q: client.Query{
			Type:                client.Once,
			Addrs:               []string{"fake"},
			Queries:             []client.Path{{"*"}},
			NotificationHandler: func(client.Notification) error { return nil },
		},
		u: [][]interface{}{{
			client.Update{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Sync{},
		}, {}},
		want: []client.Leaves{{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
		}, {}},
		err: true,
	}, {
		desc: "Poll With Entry Test",
		q: client.Query{
			Type:                client.Poll,
			Addrs:               []string{"fake"},
			Queries:             []client.Path{{"*"}},
			NotificationHandler: func(client.Notification) error { return nil },
		},
		u: [][]interface{}{{
			client.Update{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Sync{},
		}, {
			client.Update{TS: time.Unix(3, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Sync{},
		}},
		want: []client.Leaves{{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
		}, {
			{TS: time.Unix(3, 0), Path: client.Path{"a", "b"}, Val: 1},
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fake.Mock(cacheTest, tt.u[0])
			c := client.New()
			defer c.Close()
			if err := c.Subscribe(context.Background(), tt.q, cacheTest); err != nil {
				t.Errorf("Subscribe() failed: %v", err)
			}
			l := c.Leaves()
			if !reflect.DeepEqual(l, tt.want[0]) {
				t.Fatalf("Unexpected updates: got:\n%v\nwant:\n%v", l, tt.want[0])
			}
			impl, err := c.Impl()
			if err != nil {
				t.Fatalf("c.Impl: %v", err)
			}
			impl.(*fake.Client).Reset(tt.u[1])
			err = c.Poll()
			switch {
			case err != nil && tt.err:
				return
			case err != nil && !tt.err:
				t.Errorf("Poll() failed: %v", err)
				return
			case err == nil && tt.err:
				t.Errorf("Poll() expected error.")
				return
			}
			l = c.Leaves()
			if !reflect.DeepEqual(l, tt.want[1]) {
				t.Fatalf("Unexpected updates: got:\n%v\nwant:\n%v", l, tt.want[1])
			}
		})
	}
}

func TestCache(t *testing.T) {
	fake.Mock(cacheFail, []interface{}{errors.New("client failed")})

	nTest := false
	tests := []struct {
		desc       string
		q          client.Query
		u          []interface{}
		clientType []string
		want       client.Leaves
		err        bool
	}{{
		desc:       "Error New",
		clientType: []string{cacheFail},
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
		},
		want: nil,
		err:  true,
	}, {
		desc: "Once Test",
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
		},
		u: []interface{}{
			client.Update{TS: time.Unix(1, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Delete{TS: time.Unix(2, 0), Path: client.Path{"a", "b"}},
			client.Sync{},
		},
		want: nil,
	}, {
		desc: "Once With Entry Test",
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
		},
		u: []interface{}{
			client.Update{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Sync{},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
		},
	}, {
		desc: "Custom handler with Sync test",
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
			NotificationHandler: func(n client.Notification) error {
				if _, ok := n.(client.Sync); ok {
					nTest = true
				}
				return nil
			},
		},
		u: []interface{}{
			client.Update{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
			client.Sync{},
		},
		want: client.Leaves{
			{TS: time.Unix(0, 0), Path: client.Path{"a", "b"}, Val: 1},
		},
	}, {
		desc: "Error on notification",
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
		},
		u: []interface{}{
			client.Error{},
		},
		err: true,
	}, {
		desc: "Error on Recv",
		q: client.Query{
			Type:    client.Once,
			Addrs:   []string{"fake"},
			Queries: []client.Path{{"*"}},
		},
		u: []interface{}{
			fmt.Errorf("Recv() error"),
		},
		err: true,
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fake.Mock(cacheTest, tt.u)

			c := client.New()
			defer c.Close()
			if tt.q.NotificationHandler != nil {
				go func() {
					<-c.Synced()
					if !nTest {
						t.Errorf("Synced() failed: got %v, want true", nTest)
					}
				}()
			} else {
				tt.q.NotificationHandler = func(client.Notification) error { return nil }
			}
			clientType := []string{cacheTest}
			if tt.clientType != nil {
				clientType = tt.clientType
			}
			err := c.Subscribe(context.Background(), tt.q, clientType...)
			switch {
			case err != nil && tt.err:
				return
			case err != nil && !tt.err:
				t.Errorf("Subscribe() failed: %v", err)
				return
			case err == nil && tt.err:
				t.Errorf("Subscribe() expected error.")
				return
			}
			l := c.Leaves()
			if !reflect.DeepEqual(l, tt.want) {
				t.Fatalf("Unexpected updates: got:\n%v\nwant:\n%v", l, tt.want)
			}
		})
	}
}
