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
)

type stored struct {
	ph []string
}

type testClient struct {
	updates []stored
}

type testQuery struct {
	client  testClient
	query   []string
	match   []stored
	noMatch []stored
	remove  func()
}

func (t *testClient) Update(n interface{}) {
	t.updates = append(t.updates, n.(stored))
}

func TestClient(t *testing.T) {
	tests := []*testQuery{
		{
			query: []string{},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{Glob},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a"},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []stored{
				{ph: []string{"b"}},
				{ph: []string{"c"}},
				{ph: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b"},
			match: []stored{
				{ph: []string{"a"}}, // Target delete.
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []stored{
				{ph: []string{"a", "c"}},
				{ph: []string{"b"}},
				{ph: []string{"c"}},
				{ph: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b", Glob},
			match: []stored{
				{ph: []string{"a"}},      // Target delete.
				{ph: []string{"a", "b"}}, // Intermediate delete.
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []stored{
				{ph: []string{"a", "c"}},
				{ph: []string{"b"}},
				{ph: []string{"c"}},
				{ph: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{Glob, "b"},
			match: []stored{
				{ph: []string{"a"}}, // Target delete.
				{ph: []string{"b"}}, // Target delete.
				{ph: []string{"c"}}, // Target delete.
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
				{ph: []string{"e", "b", "y"}},
			},
			noMatch: []stored{
				{ph: []string{"a", "c"}},
				{ph: []string{"d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b", "*", "d", "e", "f", "g"},
			match: []stored{
				{ph: []string{"a"}}, // Target delete.
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c"}},
				{ph: []string{"a", "b", "c", "d"}},
				{ph: []string{"a", "b", "c", "d", "e"}},
				{ph: []string{"a", "b", "c", "d", "e", "f"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
			noMatch: []stored{
				{ph: []string{"a", "c"}},
				{ph: []string{"d", "e", "f", "g"}},
			},
		},
	}
	for _, tt := range tests {
		match := New()
		remove := match.AddQuery(tt.query, &tt.client)
		for _, m := range tt.match {
			match.Update(m, m.ph)
		}
		for _, m := range tt.noMatch {
			match.Update(m, m.ph)
		}
		if !reflect.DeepEqual(tt.client.updates, tt.match) {
			t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.updates, tt.match)
		}
		remove()
		tt.client.updates = nil
		for _, m := range tt.match {
			match.Update(m, m.ph)
		}
		for _, m := range tt.noMatch {
			match.Update(m, m.ph)
		}
		if tt.client.updates != nil {
			t.Errorf("removed query %q got updates %q, want nil", tt.query, tt.client.updates)
		}
	}
}

func TestMultipleClients(t *testing.T) {
	updates := []stored{
		{ph: []string{"a"}},
		{ph: []string{"a", "b"}},
		{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
		{ph: []string{"b"}},
		{ph: []string{"c"}},
		{ph: []string{"d", "e", "f", "g"}},
		{ph: []string{"e", "b", "y"}},
	}
	tests := []*testQuery{
		{
			query: []string{},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
				{ph: []string{"b"}},
				{ph: []string{"c"}},
				{ph: []string{"d", "e", "f", "g"}},
				{ph: []string{"e", "b", "y"}},
			},
		}, {
			query: []string{Glob},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
				{ph: []string{"b"}},
				{ph: []string{"c"}},
				{ph: []string{"d", "e", "f", "g"}},
				{ph: []string{"e", "b", "y"}},
			},
		}, {
			query: []string{"a"},
			match: []stored{
				{ph: []string{"a"}},
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"a", "b"},
			match: []stored{
				{ph: []string{"a"}}, // Target delete.
				{ph: []string{"a", "b"}},
				{ph: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		}, {
			query: []string{"b"},
			match: []stored{
				{ph: []string{"b"}},
			},
		}, {
			query: []string{"q"},
		},
	}
	match := New()
	// Add one query at a time.
	for x, tt := range tests {
		for _, tt := range tests {
			tt.client.updates = nil
		}
		tt.remove = match.AddQuery(tt.query, &tt.client)
		for _, u := range updates {
			match.Update(u, u.ph)
		}
		for _, tt := range tests[:x+1] {
			if !reflect.DeepEqual(tt.client.updates, tt.match) {
				t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.updates, tt.match)
			}
		}
	}
	// Remove one query at a time.
	for x, tt := range tests {
		for _, tt := range tests {
			tt.client.updates = nil
		}
		tt.remove()
		for _, u := range updates {
			match.Update(u, u.ph)
		}
		for _, tt := range tests[x+1:] {
			if !reflect.DeepEqual(tt.client.updates, tt.match) {
				t.Errorf("query %q got updates %q, want %q", tt.query, tt.client.updates, tt.match)
			}
		}
	}
}

type globTestClient struct {
	match bool
}

type globQuery struct {
	client globTestClient
	query  []string
	expect bool
	remove func()
}

func (t *globTestClient) Update(n interface{}) {
	t.match = true
}

func TestGlobUpdates(t *testing.T) {
	tests := []struct {
		update  stored
		queries []*globQuery
	}{
		// test updates with Glob at the end of the path
		{
			update: stored{ph: []string{"a", Glob}},
			queries: []*globQuery{
				&globQuery{query: []string{"a"}, expect: true},
				&globQuery{query: []string{"a", Glob}, expect: true},
				&globQuery{query: []string{"a", "b"}, expect: true},
				&globQuery{query: []string{"b"}, expect: false},
			},
		},
		// test updates with Glob in the middle of the path
		{
			update: stored{ph: []string{"a", Glob, "b"}},
			queries: []*globQuery{
				&globQuery{query: []string{"a"}, expect: true},
				&globQuery{query: []string{"a", Glob}, expect: true},
				&globQuery{query: []string{"a", "b"}, expect: true},
				&globQuery{query: []string{"a", "c", "b"}, expect: true},
				&globQuery{query: []string{"a", "c", "c"}, expect: false},
				&globQuery{query: []string{"a", Glob, "b"}, expect: true},
				&globQuery{query: []string{"a", Glob, "c"}, expect: false},
				&globQuery{query: []string{"b"}, expect: false},
			},
		},
	}

	for i, tt := range tests {
		match := New()
		// register all the queries
		for _, q := range tt.queries {
			q.remove = match.AddQuery(q.query, &q.client)
		}
		match.Update(tt.update, tt.update.ph)
		for j, q := range tt.queries {
			if q.client.match != q.expect {
				t.Errorf("#%d update message %v; #%d query path %v; got %v, want %v", i, tt.update.ph, j, q.query, q.client.match, q.expect)
			}
		}
		for _, q := range tt.queries {
			q.remove()
		}
	}
}
