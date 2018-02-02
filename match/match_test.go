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

package match

import (
	"reflect"
	"testing"

	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/ctree"
)

type testClient struct {
	notifications []client.Notification
}

type testQuery struct {
	client  testClient
	query   []string
	match   []client.Notification
	noMatch []client.Notification
	remove  func()
}

func (t *testClient) Update(n *ctree.Leaf) {
	t.notifications = append(t.notifications, n.Value().(client.Notification))
}

func TestClient(t *testing.T) {
	tests := []*testQuery{
		{
			query: []string{},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{Glob},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []client.Notification{
				client.Delete{Path: []string{"b"}},
				client.Delete{Path: []string{"c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}}, // Target delete.
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []client.Notification{
				client.Delete{Path: []string{"a", "c"}},
				client.Delete{Path: []string{"b"}},
				client.Delete{Path: []string{"c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b", Glob},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},      // Target delete.
				client.Delete{Path: []string{"a", "b"}}, // Intermediate delete.
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []client.Notification{
				client.Delete{Path: []string{"a", "c"}},
				client.Delete{Path: []string{"b"}},
				client.Delete{Path: []string{"c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{Glob, "b"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}}, // Target delete.
				client.Delete{Path: []string{"b"}}, // Target delete.
				client.Delete{Path: []string{"c"}}, // Target delete.
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
				client.Update{Path: []string{"e", "b", "y"}},
			},
			noMatch: []client.Notification{
				client.Delete{Path: []string{"a", "c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b", "*", "d", "e", "f", "g"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}}, // Target delete.
				client.Delete{Path: []string{"a", "b"}},
				client.Delete{Path: []string{"a", "b", "c"}},
				client.Delete{Path: []string{"a", "b", "c", "d"}},
				client.Delete{Path: []string{"a", "b", "c", "d", "e"}},
				client.Delete{Path: []string{"a", "b", "c", "d", "e", "f"}},
				client.Delete{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []client.Notification{
				client.Delete{Path: []string{"a", "c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
			},
		},
	}
	for _, tt := range tests {
		match := New()
		remove := match.AddQuery(tt.query, &tt.client)
		for _, m := range tt.match {
			match.Update(ctree.DetachedLeaf(m))
		}
		for _, m := range tt.noMatch {
			match.Update(ctree.DetachedLeaf(m))
		}
		if !reflect.DeepEqual(tt.client.notifications, tt.match) {
			t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.notifications, tt.match)
		}
		remove()
		tt.client.notifications = nil
		for _, m := range tt.match {
			match.Update(ctree.DetachedLeaf(m))
		}
		for _, m := range tt.noMatch {
			match.Update(ctree.DetachedLeaf(m))
		}
		if tt.client.notifications != nil {
			t.Errorf("removed query %q got updates %q, want nil", tt.query, tt.client.notifications)
		}
	}
}

func TestMultipleClients(t *testing.T) {
	updates := []client.Notification{
		client.Delete{Path: []string{"a"}},
		client.Delete{Path: []string{"a", "b"}},
		client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
		client.Delete{Path: []string{"b"}},
		client.Delete{Path: []string{"c"}},
		client.Update{Path: []string{"d", "e", "f", "g"}},
		client.Update{Path: []string{"e", "b", "y"}},
	}
	tests := []*testQuery{
		{
			query: []string{},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
				client.Delete{Path: []string{"b"}},
				client.Delete{Path: []string{"c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
				client.Update{Path: []string{"e", "b", "y"}},
			},
		}, {
			query: []string{Glob},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
				client.Delete{Path: []string{"b"}},
				client.Delete{Path: []string{"c"}},
				client.Update{Path: []string{"d", "e", "f", "g"}},
				client.Update{Path: []string{"e", "b", "y"}},
			},
		}, {
			query: []string{"a"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}},
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b"},
			match: []client.Notification{
				client.Delete{Path: []string{"a"}}, // Target delete.
				client.Delete{Path: []string{"a", "b"}},
				client.Update{Path: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"b"},
			match: []client.Notification{
				client.Delete{Path: []string{"b"}},
			},
		}, {
			query: []string{"q"},
		},
	}
	match := New()
	// Add one query at a time.
	for x, tt := range tests {
		for _, tt := range tests {
			tt.client.notifications = nil
		}
		tt.remove = match.AddQuery(tt.query, &tt.client)
		for _, u := range updates {
			match.Update(ctree.DetachedLeaf(u))
		}
		for _, tt := range tests[:x+1] {
			if !reflect.DeepEqual(tt.client.notifications, tt.match) {
				t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.notifications, tt.match)
			}
		}
	}
	// Remove one query at a time.
	for x, tt := range tests {
		for _, tt := range tests {
			tt.client.notifications = nil
		}
		tt.remove()
		for _, u := range updates {
			match.Update(ctree.DetachedLeaf(u))
		}
		for _, tt := range tests[x+1:] {
			if !reflect.DeepEqual(tt.client.notifications, tt.match) {
				t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.notifications, tt.match)
			}
		}
	}
}
