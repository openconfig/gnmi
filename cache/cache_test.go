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

package cache

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/metadata"
)

// T provides a shorthand function to reference a test timestamp with an int.
func T(n int) time.Time { return time.Unix(0, int64(n)) }

func TestHasTarget(t *testing.T) {
	c := New(nil)
	for _, tt := range []string{"", "dev1", "dev2", "dev3"} {
		if c.HasTarget(tt) {
			t.Errorf("HasTarget(%q) got true for empty cache, want false", tt)
		}
	}
	if !c.HasTarget("*") {
		t.Error("HasTarget(*) got false for empty cache, want true")
	}
	c = New([]string{"dev1", "dev2"})
	for tt, want := range map[string]bool{"": false, "*": true, "dev1": true, "dev2": true, "dev3": false} {
		if got := c.HasTarget(tt); got != want {
			t.Errorf("HasTarget(%q) got %t, want %t", tt, got, want)
		}
	}
}

func TestAddTarget(t *testing.T) {
	c := New(nil)
	c.AddTarget("dev1")
	if !c.HasTarget("dev1") {
		t.Error("dev1 not added")
	}
}

func TestRemoveTarget(t *testing.T) {
	c := New([]string{"dev1"})
	c.RemoveTarget("dev1")
	if c.HasTarget("dev1") {
		t.Errorf("dev1 not deleted")
	}
}

type queryable struct {
	t client.Notification
	q bool
	v string
}

func TestQuery(t *testing.T) {
	c := New([]string{"dev1"})
	c.Query("", nil, func([]string, *ctree.Leaf, interface{}) { t.Error("querying without a target invoked callback") })
	updates := []queryable{
		// This update is inserted here, but deleted below.
		{client.Update{[]string{"dev1", "a", "e"}, "value1", T(0)}, true, ""},
		// This update is ovewritten below.
		{client.Update{[]string{"dev1", "a", "b"}, "value1", T(0)}, true, "value3"},
		// This update is inserted and not modified.
		{client.Update{[]string{"dev1", "a", "c"}, "value4", T(0)}, true, "value4"},
		// This update overwrites a previous above.
		{client.Update{[]string{"dev1", "a", "b"}, "value3", T(1)}, true, "value3"},
		// These two targets don't exist in the cache and the updates are rejected.
		{client.Update{[]string{"dev2", "a", "b"}, "value1", T(0)}, false, ""},
		{client.Update{[]string{"dev3", "a", "b"}, "value2", T(0)}, false, ""},
		// This is a delete that removes the first update, above.
		{client.Delete{Path: []string{"dev1", "a", "e"}, TS: T(1)}, true, ""},
	}
	// Add updates to cache.
	for _, tt := range updates {
		c.Update(tt.t)
	}
	// Run queries over the inserted updates.
	for x, tt := range updates {
		var l client.Leaf
		switch v := tt.t.(type) {
		case client.Update:
			l = (client.Leaf)(v)
		case client.Delete:
			l = (client.Leaf)(v)
		}
		target, path := l.Path[0], l.Path[1:]
		if r := c.HasTarget(target); r != tt.q {
			t.Errorf("#%d: got %v, want %v", x, r, tt.q)
		}
		var results []interface{}
		appendResults := func(_ []string, _ *ctree.Leaf, val interface{}) { results = append(results, val) }
		c.Query(target, path, appendResults)
		if len(results) != 1 {
			if tt.v != "" {
				t.Errorf("Query(%s, %v, ): got %d results, want 1", target, path, len(results))
			}
			continue
		}
		val := results[0].(client.Update).Val.(string)
		if val != tt.v {
			t.Errorf("#%d: got %q, want %q", x, val, tt.v)
		}
	}
}

func TestQueryAll(t *testing.T) {
	c := New([]string{"dev1", "dev2", "dev3"})
	updates := map[string]client.Update{
		"value1": client.Update{[]string{"dev1", "a", "b"}, "value1", T(0)},
		"value2": client.Update{[]string{"dev2", "a", "c"}, "value2", T(0)},
		"value3": client.Update{[]string{"dev3", "a", "d"}, "value3", T(0)},
	}
	// Add updates to cache.
	for _, u := range updates {
		c.Update(u)
	}
	target, path := "*", []string{"a"}
	if r := c.HasTarget(target); !r {
		t.Error("Query not executed against cache for target * and path a")
	}
	var results []interface{}
	appendResults := func(_ []string, _ *ctree.Leaf, val interface{}) { results = append(results, val) }
	c.Query(target, path, appendResults)
	for _, v := range results {
		val := v.(client.Update).Val.(string)
		if _, ok := updates[val]; !ok {
			t.Errorf("got unexpected update value %#v, want one of %v", v, updates)
		}
		delete(updates, val)
	}
	if len(updates) > 0 {
		t.Errorf("the following updates were not received for query of target * with path a: %v", updates)
	}
}

func TestResetTarget(t *testing.T) {
	targets := []string{"dev1", "dev2", "dev3"}
	c := New(targets)
	updates := map[string]client.Update{
		"value1":  client.Update{[]string{"dev1", "a", "b"}, "value1", T(0)},
		"value2":  client.Update{[]string{"dev2", "a", "c"}, "value2", T(0)},
		"value3":  client.Update{[]string{"dev3", "a", "d"}, "value3", T(0)},
		"invalid": client.Update{}, // Should have no effect on test.
	}
	// Add updates to cache.
	for _, u := range updates {
		c.Update(u)
	}
	var results []interface{}
	var hasMeta bool
	appendResults := func(path []string, _ *ctree.Leaf, val interface{}) {
		if path[0] == metadata.Root {
			hasMeta = true
		} else {
			results = append(results, val)
		}
	}
	for _, target := range targets {
		results = nil
		hasMeta = false
		c.Query(target, []string{"*"}, appendResults)
		if got := len(results); got != 1 {
			t.Errorf("Target %q got %d results, want 1\n\t%v", target, got, results)
		}
		if hasMeta {
			t.Errorf("Target %q got metadata, want none", target)
		}
		c.ResetTarget(target)
		results = nil
		hasMeta = false
		c.Query(target, []string{"*"}, appendResults)
		if got := len(results); got != 0 {
			t.Errorf("Target %q got %d results, want 0\n\t%v", target, got, results)
		}
		if !hasMeta {
			t.Errorf("Target %q got no metadata, want metadata", target)
		}
	}
}

func TestResetTargetUnknown(t *testing.T) {
	c := New([]string{})
	c.ResetTarget("dev1")
	if c.HasTarget("dev1") {
		t.Error("c.ResetTarget created a target that didn't exist before")
	}
}

func TestPathEquals(t *testing.T) {
	tests := []struct {
		p1, p2 []string
		want   bool
	}{
		{[]string{}, []string{}, true},
		{[]string{"a"}, []string{"a"}, true},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{[]string{}, []string{"a"}, false},
		{[]string{"b"}, []string{"a"}, false},
		{[]string{"b"}, []string{}, false},
		{[]string{"a", "b"}, []string{"a"}, false},
		{[]string{"a"}, []string{"a", "b"}, false},
	}
	for _, tt := range tests {
		if got := pathEquals(tt.p1, tt.p2); got != tt.want {
			t.Errorf("pathEquals(%q, %q) got %t, want %t", tt.p1, tt.p2, got, tt.want)
		}
	}
}

func TestClient(t *testing.T) {
	c := New([]string{"dev1"})
	var got []interface{}
	c.SetClient(func(n *ctree.Leaf) {
		got = append(got, n.Value())
	})
	sortGot := func() {
		path := func(i int) client.Path {
			switch l := got[i].(type) {
			case client.Update:
				return l.Path
			case client.Delete:
				return l.Path
			}
			return nil
		}
		sort.Slice(got, func(i, j int) bool {
			return path(i).Less(path(j))
		})
	}

	tests := []struct {
		desc    string
		updates []client.Notification
		want    []interface{}
	}{
		{
			desc: "add new nodes",
			updates: []client.Notification{
				client.Update{Path: []string{"dev1", "a", "b"}, Val: true, TS: T(1)},
				client.Update{Path: []string{"dev1", "a", "c"}, Val: true, TS: T(2)},
				client.Update{Path: []string{"dev1", "a", "d"}, Val: true, TS: T(3)},
			},
			want: []interface{}{
				client.Update{Path: []string{"dev1", "a", "b"}, Val: true, TS: T(1)},
				client.Update{Path: []string{"dev1", "a", "c"}, Val: true, TS: T(2)},
				client.Update{Path: []string{"dev1", "a", "d"}, Val: true, TS: T(3)},
			},
		},
		{
			desc: "update nodes",
			updates: []client.Notification{
				client.Update{Path: []string{"dev1", "a", "b"}, Val: true, TS: T(0)},
				client.Update{Path: []string{"dev1", "a", "b"}, Val: true, TS: T(1)},
				client.Update{Path: []string{"dev1", "a", "b"}, Val: false, TS: T(2)},
			},
			want: []interface{}{
				client.Update{Path: []string{"dev1", "a", "b"}, Val: false, TS: T(2)},
			},
		},
		{
			desc: "delete nodes",
			updates: []client.Notification{
				client.Delete{Path: []string{"dev1", "a", "b"}, TS: T(1)},
				client.Delete{Path: []string{"dev1", "a", "b"}, TS: T(3)},
				client.Delete{Path: []string{"dev1", "a"}, TS: T(4)},
			},
			want: []interface{}{
				client.Delete{Path: []string{"dev1", "a", "b"}, TS: T(3)},
				client.Delete{Path: []string{"dev1", "a", "c"}, TS: T(4)},
				client.Delete{Path: []string{"dev1", "a", "d"}, TS: T(4)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got = nil
			for _, u := range tt.updates {
				c.Update(u)
			}
			sortGot()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("sent updates: %v\ndiff in received updates:\n%s", tt.updates, diff)
			}
		})
	}
	t.Run("remove target", func(t *testing.T) {
		got = nil
		c.RemoveTarget("dev1")
		want := client.Delete{Path: []string{"dev1"}}
		if len(got) != 1 {
			t.Fatalf("RemoveTarget didn't produce correct client update, got: %+v, want: %+v", got, []interface{}{want})
		}
		gotVal := got[0].(client.Delete)
		// Clear timestamp before comparison.
		gotVal.TS = time.Time{}
		if diff := cmp.Diff(want, gotVal); diff != "" {
			t.Errorf("diff in received update:\n%s", diff)
		}
	})
}

func TestMetadata(t *testing.T) {
	targets := []string{"dev1", "dev2", "dev3"}
	c := New(targets)
	md := c.Metadata()
	if got, want := len(md), len(targets); got != want {
		t.Errorf("Metadata got %d targets, want %d", got, want)
	}
}

func TestUpdateMeta(t *testing.T) {
	c := New([]string{"dev1"})

	var lastSize, lastCount, lastAdds, lastUpds int64
	for i := 0; i < 10; i++ {
		c.Update(client.Update{[]string{"dev1", "a", fmt.Sprint(i)}, "b", T(i)})

		c.getTarget("dev1").updateSize(nil)
		c.getTarget("dev1").updateMeta(nil)

		var path []string
		path = metadata.Path(metadata.Size)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newSize := v.(client.Update).Val.(int64)
			if newSize <= lastSize {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastSize = newSize
		})
		path = metadata.Path(metadata.LeafCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newCount := v.(client.Update).Val.(int64)
			if newCount <= lastCount {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastCount = newCount
		})
		path = metadata.Path(metadata.AddCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newAdds := v.(client.Update).Val.(int64)
			if newAdds <= lastAdds {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastAdds = newAdds
		})
		path = metadata.Path(metadata.UpdateCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newUpds := v.(client.Update).Val.(int64)
			if newUpds <= lastUpds {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastUpds = newUpds
		})
		path = metadata.Path(metadata.DelCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			dels := v.(client.Update).Val.(int64)
			if dels > 0 {
				t.Errorf("%s is %d after adding leaf #%d, even though no leaves were removed",
					strings.Join(path, "/"), dels, i+1)
			}
		})
	}
	for _, path := range [][]string{
		metadata.Path(metadata.LatencyAvg),
		metadata.Path(metadata.LatencyMax),
		metadata.Path(metadata.LatencyMin),
	} {
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			if l := v.(client.Update).Val.(int64); l != 0 {
				t.Errorf("%s exists with value %d when device not in sync",
					strings.Join(path, "/"), l)
			}
		})
	}
	timestamp := time.Now().Add(-time.Minute)
	c.Update(client.Update{append([]string{"dev1"}, metadata.Path(metadata.Sync)...), true, timestamp})
	c.Update(client.Update{[]string{"dev1", "a", "1"}, "b", timestamp})
	c.getTarget("dev1").updateMeta(nil)
	for _, path := range [][]string{
		metadata.Path(metadata.LatencyAvg),
		metadata.Path(metadata.LatencyMax),
		metadata.Path(metadata.LatencyMin),
	} {
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			l := v.(client.Update).Val.(int64)
			if want := time.Minute.Nanoseconds(); l < want {
				t.Errorf("%s got value %d, want greater than %d",
					strings.Join(path, "/"), l, want)
			}
		})
	}
}

func TestUpdateMetadata(t *testing.T) {
	c := New([]string{"dev1"})
	c.UpdateMetadata()
	want := []client.Path{
		{metadata.Root, metadata.LatestTimestamp},
		{metadata.Root, metadata.LeafCount},
		{metadata.Root, metadata.AddCount},
		{metadata.Root, metadata.UpdateCount},
		{metadata.Root, metadata.DelCount},
		{metadata.Root, metadata.Connected},
		{metadata.Root, metadata.Sync},
		{metadata.Root, metadata.Size},
		{metadata.Root, metadata.LatencyAvg},
		{metadata.Root, metadata.LatencyMin},
		{metadata.Root, metadata.LatencyMax},
	}
	var got []client.Path
	c.Query("dev1", []string{metadata.Root}, func(path []string, _ *ctree.Leaf, _ interface{}) {
		got = append(got, path)
	})
	sort.Slice(got, func(i, j int) bool { return got[i].Less(got[j]) })
	sort.Slice(want, func(i, j int) bool { return want[i].Less(want[j]) })
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got update paths: %q\n want: %q", got, want)
	}
}

func TestUpdateSize(t *testing.T) {
	c := New([]string{"dev1"})
	c.Update(client.Update{[]string{"dev1", "a", "1"}, make([]byte, 1000), T(0)})
	c.UpdateSize()
	c.UpdateMetadata()
	var val int64
	c.Query("dev1", []string{metadata.Root, metadata.Size}, func(_ []string, _ *ctree.Leaf, v interface{}) {
		t.Logf("%v", v)
		val = v.(client.Update).Val.(int64)
	})
	if val <= 1000 {
		t.Errorf("got size of %d want > 1000", val)
	}
}

type updateQueryData struct {
	targets []string
	paths   [][]string
	values  []*string
}

func sendUpdates(u updateQueryData, c *Cache, n int, wg *sync.WaitGroup) {
	r := rand.New(rand.NewSource(5))
	for i := 0; i < n; i++ {
		target := u.targets[r.Intn(len(u.targets))]
		path := append([]string{target}, u.paths[r.Intn(len(u.paths))]...)
		value := u.values[r.Intn(len(u.values))]
		c.Update(client.Update{path, value, T(n)})
	}
}

func makeQueries(u updateQueryData, c *Cache, n int, wg *sync.WaitGroup) {
	r := rand.New(rand.NewSource(30))
	for i := 0; i < n; i++ {
		target := u.targets[r.Intn(len(u.targets))]
		path := u.paths[r.Intn(len(u.paths))]
		c.Query(target, path, func([]string, *ctree.Leaf, interface{}) {})
	}
	wg.Done()
}

func createPaths(l, n int) [][]string {
	var paths [][]string
	if l == 0 {
		for i := 0; i < n; i++ {
			paths = append(paths, []string{fmt.Sprintf("level_%d_node_%d", l, i)})
		}
		return paths
	}
	subpaths := createPaths(l-1, n)
	for i := 0; i < n; i++ {
		for _, p := range subpaths {
			paths = append(paths, append([]string{fmt.Sprintf("level_%d_node_%d", l, i)}, p...))
		}
	}
	return paths
}

func BenchmarkParallelUpdateQuery(b *testing.B) {
	u := updateQueryData{
		targets: []string{"dev1", "dev2", "dev3"},
		paths:   createPaths(5, 10),
		values:  []*string{nil},
	}
	for v := 0; v < 100; v++ {
		s := fmt.Sprintf("value_%d", v)
		u.values = append(u.values, &s)
	}
	procs := runtime.GOMAXPROCS(0)
	c := New(u.targets)
	var wg sync.WaitGroup
	b.ResetTimer()
	for p := 0; p < procs; p++ {
		wg.Add(2)
		go sendUpdates(u, c, b.N, &wg)
		go makeQueries(u, c, b.N, &wg)
	}
	wg.Wait()
}

func TestIsDeleteTarget(t *testing.T) {
	testCases := []struct {
		name string
		noti client.Notification
		want bool
	}{
		{"Update", client.Update{Path: client.Path{"a"}}, false},
		{"Leaf delete", client.Delete{Path: client.Path{"a", "b"}}, false},
		{"Target delete", client.Delete{Path: client.Path{"target"}}, true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := ctree.DetachedLeaf(tc.noti)
			if got := IsTargetDelete(l); got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}
