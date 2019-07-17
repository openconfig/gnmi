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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/gnmi/metadata"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

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

func TestAdd(t *testing.T) {
	c := New(nil)
	c.Add("dev1")
	if !c.HasTarget("dev1") {
		t.Error("dev1 not added")
	}
}

func TestRemove(t *testing.T) {
	c := New([]string{"dev1"})
	c.Remove("dev1")
	if c.HasTarget("dev1") {
		t.Errorf("dev1 not deleted")
	}
}

func TestGNMIRemove(t *testing.T) {
	Type = GnmiNoti
	defer func() {
		Type = ClientLeaf
	}()
	tg := "dev1"
	c := New([]string{tg})
	var got interface{}
	c.SetClient(func(l *ctree.Leaf) {
		got = l.Value()
	})
	c.Remove(tg)
	if got == nil {
		t.Fatalf("no update was received")
	}
	if c.HasTarget("dev1") {
		t.Errorf("dev1 not deleted")
	}
	noti, ok := got.(*gpb.Notification)
	if !ok {
		t.Fatalf("got %T, want *gpb.Notification type", got)
	}
	if noti.Prefix.GetTarget() != tg {
		t.Errorf("got %q, want %q", noti.Prefix.GetTarget(), tg)
	}
	if len(noti.Delete) != 1 {
		t.Fatalf("got %d, want 1 delete update in notification", len(noti.Delete))
	}
	d := noti.Delete[0]
	if len(d.Elem) != 1 {
		t.Fatalf("got %d, want 1 elems in delete notificatin path", len(d.Elem))
	}
	if p := d.Elem[0].Name; p != "*" {
		t.Errorf("got %q, want %q in target delete path", p, "*")
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
		{client.Update{Path: []string{"dev1", "a", "e"}, Val: "value1", TS: T(0)}, true, ""},
		// This update is ovewritten below.
		{client.Update{Path: []string{"dev1", "a", "b"}, Val: "value1", TS: T(0)}, true, "value3"},
		// This update is inserted and not modified.
		{client.Update{Path: []string{"dev1", "a", "c"}, Val: "value4", TS: T(0)}, true, "value4"},
		// This update overwrites a previous above.
		{client.Update{Path: []string{"dev1", "a", "b"}, Val: "value3", TS: T(1)}, true, "value3"},
		// These two targets don't exist in the cache and the updates are rejected.
		{client.Update{Path: []string{"dev2", "a", "b"}, Val: "value1", TS: T(0)}, false, ""},
		{client.Update{Path: []string{"dev3", "a", "b"}, Val: "value2", TS: T(0)}, false, ""},
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
		"value1": client.Update{Path: []string{"dev1", "a", "b"}, Val: "value1", TS: T(0)},
		"value2": client.Update{Path: []string{"dev2", "a", "c"}, Val: "value2", TS: T(0)},
		"value3": client.Update{Path: []string{"dev3", "a", "d"}, Val: "value3", TS: T(0)},
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

func TestReset(t *testing.T) {
	targets := []string{"dev1", "dev2", "dev3"}
	c := New(targets)
	updates := map[string]client.Update{
		"value1":  client.Update{Path: []string{"dev1", "a", "b"}, Val: "value1", TS: T(0)},
		"value2":  client.Update{Path: []string{"dev2", "a", "c"}, Val: "value2", TS: T(0)},
		"value3":  client.Update{Path: []string{"dev3", "a", "d"}, Val: "value3", TS: T(0)},
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
		c.Reset(target)
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

func TestResetUnknown(t *testing.T) {
	c := New([]string{})
	c.Reset("dev1")
	if c.HasTarget("dev1") {
		t.Error("c.Reset created a target that didn't exist before")
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
		c.Remove("dev1")
		want := client.Delete{Path: []string{"dev1"}}
		if len(got) != 1 {
			t.Fatalf("Remove didn't produce correct client update, got: %+v, want: %+v", got, []interface{}{want})
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
		c.Update(client.Update{Path: []string{"dev1", "a", fmt.Sprint(i)}, Val: "b", TS: T(int64(i))})

		c.GetTarget("dev1").updateSize(nil)
		c.GetTarget("dev1").updateMeta(nil)

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
}

func TestMetadataStale(t *testing.T) {
	c := New([]string{"dev1"})
	for i := 0; i < 10; i++ {
		c.Update(client.Update{Path: []string{"dev1", "a"}, Val: i, TS: T(int64(10 - i))})
		c.GetTarget("dev1").updateMeta(nil)
		path := metadata.Path(metadata.StaleCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			staleCount := v.(client.Update).Val.(int64)
			if staleCount != int64(i) {
				t.Errorf("got staleCount = %d, want %d", staleCount, i)
			}
		})
		path = metadata.Path(metadata.UpdateCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			updates := v.(client.Update).Val.(int64)
			if updates != 1 {
				t.Errorf("got updates %d, want 1", updates)
			}
		})
	}
}

func TestGNMIUpdateIntermediateUpdate(t *testing.T) {
	c := New([]string{"dev1"})
	// Initialize a cache tree for next GnmiUpdate test.
	n := gnmiNotification("dev1", []string{"prefix", "path"}, []string{"update", "a", "b", "c"}, 0, "", false)
	if err := c.GnmiUpdate(n); err != nil {
		t.Fatalf("GnmiUpdate(%+v): got %v, want nil error", n, err)
	}
	// This is a negative test case for invalid path in an update.
	// For a cache tree initialized above with path "a"/"b"/"c", "a" is a non-leaf node.
	// Because non-leaf node "a" does not contain notification, error should be returned in following update.
	n = gnmiNotification("dev1", []string{"prefix", "path"}, []string{"update", "a"}, 0, "", false)
	err := c.GnmiUpdate(n)
	if diff := errdiff.Substring(err, "corrupt schema with collision"); diff != "" {
		t.Errorf("GnmiUpdate(%+v): %v", n, diff)
	}
}

func TestMetadataSuppressed(t *testing.T) {
	c := New([]string{"dev1"})
	// Unique values not suppressed.
	for i := 0; i < 10; i++ {
		c.GnmiUpdate(gnmiNotification("dev1", []string{"prefix", "path"}, []string{"update", "path"}, int64(i), strconv.Itoa(i), false))
		c.GetTarget("dev1").updateMeta(nil)
		path := metadata.Path(metadata.SuppressedCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			suppressedCount := v.(client.Update).Val.(int64)
			if suppressedCount != 0 {
				t.Errorf("got suppressedCount = %d, want 0", suppressedCount)
			}
		})
		path = metadata.Path(metadata.UpdateCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			updates := v.(client.Update).Val.(int64)
			if updates != int64(i+1) {
				t.Errorf("got updates %d, want %d", updates, i)
			}
		})
	}
	c.Reset("dev1")
	// Duplicate values suppressed.
	for i := 0; i < 10; i++ {
		c.GnmiUpdate(gnmiNotification("dev1", []string{"prefix", "path"}, []string{"update", "path"}, int64(i), "same value", false))
		c.GetTarget("dev1").updateMeta(nil)
		path := metadata.Path(metadata.SuppressedCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			suppressedCount := v.(client.Update).Val.(int64)
			if suppressedCount != int64(i) {
				t.Errorf("got suppressedCount = %d, want %d", suppressedCount, i)
			}
		})
		path = metadata.Path(metadata.UpdateCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			updates := v.(client.Update).Val.(int64)
			if updates != 1 {
				t.Errorf("got updates %d, want 1", updates)
			}
		})
	}
}

func TestMetadataLatency(t *testing.T) {
	c := New([]string{"dev1"})

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
	c.Update(client.Update{Path: append([]string{"dev1"}, metadata.Path(metadata.Sync)...), Val: true, TS: timestamp})
	c.Update(client.Update{Path: []string{"dev1", "a", "1"}, Val: "b", TS: timestamp})
	c.GetTarget("dev1").updateMeta(nil)
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
		{metadata.Root, metadata.StaleCount},
		{metadata.Root, metadata.SuppressedCount},
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
	c.Update(client.Update{Path: []string{"dev1", "a", "1"}, Val: make([]byte, 1000), TS: T(0)})
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
		c.Update(client.Update{Path: path, Val: value, TS: T(int64(n))})
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
		noti interface{}
		want bool
	}{
		{
			name: "Update",
			noti: client.Update{Path: client.Path{"a"}},
			want: false,
		},
		{
			name: "Leaf delete",
			noti: client.Delete{Path: client.Path{"a", "b"}},
			want: false,
		},
		{
			name: "Target delete",
			noti: client.Delete{Path: client.Path{"target"}},
			want: true,
		},
		{
			name: "GNMI Update",
			noti: &gpb.Notification{
				Prefix: &gpb.Path{Target: "d", Origin: "o"},
				Update: []*gpb.Update{
					{
						Path: &gpb.Path{
							Elem: []*gpb.PathElem{{Name: "p"}},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "GNMI Leaf Delete",
			noti: &gpb.Notification{
				Prefix: &gpb.Path{Target: "d", Origin: "o"},
				Delete: []*gpb.Path{{
					Elem: []*gpb.PathElem{{Name: "p"}},
				}},
			},
			want: false,
		},
		{
			name: "GNMI Root Delete",
			noti: &gpb.Notification{
				Prefix: &gpb.Path{Target: "d", Origin: "o"},
				Delete: []*gpb.Path{{
					Elem: []*gpb.PathElem{{Name: "*"}},
				}},
			},
			want: false,
		},
		{
			name: "GNMI Target Delete",
			noti: &gpb.Notification{
				Prefix: &gpb.Path{Target: "d"},
				Delete: []*gpb.Path{{
					Elem: []*gpb.PathElem{{Name: "*"}},
				}},
			},
			want: true,
		},
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

func gnmiNotification(dev string, prefix []string, path []string, ts int64, val string, delete bool) *gpb.Notification {
	return notificationBundle(dev, prefix, ts, []update{
		{
			delete: delete,
			path:   path,
			val:    val,
		}})
}

type update struct {
	delete bool
	path   []string
	val    string
}

func notificationBundle(dev string, prefix []string, ts int64, updates []update) *gpb.Notification {
	n := &gpb.Notification{
		Prefix:    &gpb.Path{Element: prefix, Target: dev},
		Timestamp: ts,
	}
	for _, u := range updates {
		if u.delete {
			n.Delete = append(n.Delete, &gpb.Path{Element: u.path})
		} else {
			n.Update = append(n.Update, &gpb.Update{
				Path: &gpb.Path{
					Element: u.path,
				},
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{u.val}},
			})
		}
	}
	return n
}

func TestGNMIQuery(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})
	c.Query("", nil, func([]string, *ctree.Leaf, interface{}) { t.Error("querying without a target invoked callback") })
	updates := []struct {
		t *gpb.Notification
		q bool
		v string
	}{
		// This update is inserted here, but deleted below.
		{gnmiNotification("dev1", []string{}, []string{"a", "e"}, 0, "value1", false), true, ""},
		// This update is ovewritten below.
		{gnmiNotification("dev1", []string{}, []string{"a", "b"}, 0, "value1", false), true, "value3"},
		// This update is inserted and not modified.
		{gnmiNotification("dev1", []string{}, []string{"a", "c"}, 0, "value4", false), true, "value4"},
		// This update overwrites a previous above.
		{gnmiNotification("dev1", []string{}, []string{"a", "b"}, 1, "value3", false), true, "value3"},
		// These two targets don't exist in the cache and the updates are rejected.
		{gnmiNotification("dev2", []string{}, []string{"a", "b"}, 0, "value1", false), false, ""},
		{gnmiNotification("dev3", []string{}, []string{"a", "b"}, 0, "value2", false), false, ""},
		// This is a delete that removes the first update, above.
		{gnmiNotification("dev1", []string{}, []string{"a", "e"}, 1, "", true), true, ""},
	}
	// Add updates to cache.
	for _, tt := range updates {
		c.GnmiUpdate(tt.t)
	}
	// Run queries over the inserted updates.
	for x, tt := range updates {
		target := tt.t.GetPrefix().GetTarget()
		var gp *gpb.Path
		if tt.t.Update != nil {
			gp = tt.t.Update[0].GetPath()
		} else {
			gp = tt.t.Delete[0]
		}
		p := joinPrefixAndPath(tt.t.GetPrefix(), gp)
		if r := c.HasTarget(target); r != tt.q {
			t.Errorf("#%d: got %v, want %v", x, r, tt.q)
		}
		var results []interface{}
		appendResults := func(_ []string, _ *ctree.Leaf, val interface{}) { results = append(results, val) }
		c.Query(target, p, appendResults)
		if len(results) != 1 {
			if tt.v != "" {
				t.Errorf("Query(%s, %v, ): got %d results, want 1", target, p, len(results))
			}
			continue
		}
		val := results[0].(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_StringVal).StringVal
		if val != tt.v {
			t.Errorf("#%d: got %q, want %q", x, val, tt.v)
		}
	}
}

func TestGNMIQueryAll(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1", "dev2", "dev3"})
	updates := map[string]*gpb.Notification{
		"value1": gnmiNotification("dev1", []string{}, []string{"a", "b"}, 0, "value1", false),
		"value2": gnmiNotification("dev2", []string{}, []string{"a", "c"}, 0, "value2", false),
		"value3": gnmiNotification("dev3", []string{}, []string{"a", "d"}, 0, "value3", false),
	}
	// Add updates to cache.
	for _, u := range updates {
		c.GnmiUpdate(u)
	}
	target, path := "*", []string{"a"}
	if r := c.HasTarget(target); !r {
		t.Error("Query not executed against cache for target * and path a")
	}
	var results []interface{}
	appendResults := func(_ []string, _ *ctree.Leaf, val interface{}) { results = append(results, val) }
	c.Query(target, path, appendResults)
	for _, v := range results {
		val := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_StringVal).StringVal
		if _, ok := updates[val]; !ok {
			t.Errorf("got unexpected update value %#v, want one of %v", v, updates)
		}
		delete(updates, val)
	}
	if len(updates) > 0 {
		t.Errorf("the following updates were not received for query of target * with path a: %v", updates)
	}
}

func TestGNMIAtomic(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})
	n := &gpb.Notification{
		Atomic:    true,
		Timestamp: time.Now().UnixNano(),
		Prefix: &gpb.Path{
			Target: "dev1",
			Origin: "openconfig",
			Elem:   []*gpb.PathElem{{Name: "a"}, {Name: "b", Key: map[string]string{"key": "value"}}},
		},
		Update: []*gpb.Update{
			{Path: &gpb.Path{Elem: []*gpb.PathElem{{Name: "x"}}}, Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"x val"}}},
			{Path: &gpb.Path{Elem: []*gpb.PathElem{{Name: "y"}}}, Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"y val"}}},
			{Path: &gpb.Path{Elem: []*gpb.PathElem{{Name: "z"}}}, Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"z val"}}},
		},
	}
	c.GnmiUpdate(n)
	tests := []struct {
		path   []string
		expect bool
	}{
		// Query paths that return the atomic update.
		{path: []string{"*"}, expect: true},
		{path: []string{"openconfig"}, expect: true},
		{path: []string{"openconfig", "a"}, expect: true},
		{path: []string{"openconfig", "a", "b"}, expect: true},
		{path: []string{"openconfig", "a", "b", "value"}, expect: true},
		// Query paths that do not.
		{path: []string{"foo"}},
		{path: []string{"openconfig", "a", "b", "value", "x"}},
	}
	for _, tt := range tests {
		c.Query("dev1", tt.path, func(_ []string, _ *ctree.Leaf, val interface{}) {
			if !tt.expect {
				t.Errorf("Query(%p): got notification %v, want none", tt.path, val)
			} else {
				if v, ok := val.(*gpb.Notification); !ok || !cmp.Equal(v, n) {
					t.Errorf("got:\n%s want\n%s", proto.MarshalTextString(v), proto.MarshalTextString(n))
				}
			}
		})
	}
}

func TestGNMIUpdateMeta(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})

	var lastSize, lastCount, lastAdds, lastUpds int64
	for i := 0; i < 10; i++ {
		c.GnmiUpdate(gnmiNotification("dev1", []string{}, []string{"a", fmt.Sprint(i)}, int64(i), "b", false))

		c.GetTarget("dev1").updateSize(nil)
		c.GetTarget("dev1").updateMeta(nil)

		var path []string
		path = metadata.Path(metadata.Size)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newSize := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
			if newSize <= lastSize {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastSize = newSize
		})
		path = metadata.Path(metadata.LeafCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newCount := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
			if newCount <= lastCount {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastCount = newCount
		})
		path = metadata.Path(metadata.AddCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newAdds := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
			if newAdds <= lastAdds {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastAdds = newAdds
		})
		path = metadata.Path(metadata.UpdateCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			newUpds := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
			if newUpds <= lastUpds {
				t.Errorf("%s didn't increase after adding leaf #%d",
					strings.Join(path, "/"), i+1)
			}
			lastUpds = newUpds
		})
		path = metadata.Path(metadata.DelCount)
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			dels := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
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
			if l := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal; l != 0 {
				t.Errorf("%s exists with value %d when device not in sync",
					strings.Join(path, "/"), l)
			}
		})
	}
	pathGen := func(ph []string) *gpb.Path {
		pe := make([]*gpb.PathElem, 0, len(ph))
		for _, p := range ph {
			pe = append(pe, &gpb.PathElem{Name: p})
		}
		return &gpb.Path{Elem: pe}
	}
	timestamp := time.Now().Add(-time.Minute)

	c.Sync("dev1")

	c.GnmiUpdate(
		&gpb.Notification{
			Timestamp: timestamp.UnixNano(),
			Prefix:    &gpb.Path{Target: "dev1"},
			Update: []*gpb.Update{
				&gpb.Update{
					Path: pathGen([]string{"a", "1"}),
					Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"b"}},
				},
			},
		})
	c.GetTarget("dev1").updateMeta(nil)
	for _, path := range [][]string{
		metadata.Path(metadata.LatencyAvg),
		metadata.Path(metadata.LatencyMax),
		metadata.Path(metadata.LatencyMin),
	} {
		c.Query("dev1", path, func(_ []string, _ *ctree.Leaf, v interface{}) {
			l := v.(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_IntVal).IntVal
			if want := time.Minute.Nanoseconds(); l < want {
				t.Errorf("%s got value %d, want greater than %d",
					strings.Join(path, "/"), l, want)
			}
		})
	}
}

func TestGNMIQueryWithPathElem(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})
	c.Query("", nil, func([]string, *ctree.Leaf, interface{}) { t.Error("querying without a target invoked callback") })
	ns := []struct {
		n *gpb.Notification
		q []string
		e string
	}{
		{
			// add value1 by sending update notification
			n: &gpb.Notification{
				Prefix: &gpb.Path{
					Target: "dev1",
					Elem:   []*gpb.PathElem{&gpb.PathElem{Name: "a"}, &gpb.PathElem{Name: "b", Key: map[string]string{"bb": "x", "aa": "y"}}, &gpb.PathElem{Name: "c"}},
				},
				Update: []*gpb.Update{
					&gpb.Update{
						Path: &gpb.Path{Elem: []*gpb.PathElem{&gpb.PathElem{Name: "d", Key: map[string]string{"kk": "1", "ff": "2"}}, &gpb.PathElem{Name: "e"}}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"value1"}},
					},
				},
				Timestamp: 0,
			},
			q: []string{"a", "b", "y", "x", "c", "d", "2", "1", "e"},
			e: "",
		},
		{
			// add value2 by sending update notification
			n: &gpb.Notification{
				Prefix: &gpb.Path{
					Target: "dev1",
					Elem:   []*gpb.PathElem{&gpb.PathElem{Name: "a"}, &gpb.PathElem{Name: "b", Key: map[string]string{"bb": "x", "aa": "y"}}, &gpb.PathElem{Name: "c"}},
				},
				Update: []*gpb.Update{
					&gpb.Update{
						Path: &gpb.Path{Elem: []*gpb.PathElem{&gpb.PathElem{Name: "d", Key: map[string]string{"kk": "1", "ff": "3"}}, &gpb.PathElem{Name: "e"}}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"value2"}},
					},
				},
				Timestamp: 1,
			},
			q: []string{"a", "b", "y", "x", "c", "d", "3", "1", "e"},
			e: "value2",
		},
		{
			// delete the value1 by sending a delete notification
			n: &gpb.Notification{
				Prefix: &gpb.Path{
					Target: "dev1",
					Elem:   []*gpb.PathElem{&gpb.PathElem{Name: "a"}, &gpb.PathElem{Name: "b", Key: map[string]string{"bb": "x", "aa": "y"}}, &gpb.PathElem{Name: "c"}},
				},
				Delete: []*gpb.Path{
					&gpb.Path{Elem: []*gpb.PathElem{&gpb.PathElem{Name: "d", Key: map[string]string{"kk": "1", "ff": "2"}}, &gpb.PathElem{Name: "e"}}},
				},
				Timestamp: 2,
			},
			q: []string{"a", "b", "y", "x", "c", "d", "2", "1", "e"},
			e: "",
		},
	}

	for _, t := range ns {
		c.GnmiUpdate(t.n)
	}

	// Run queries over the inserted updates.
	for x, tt := range ns {
		target := tt.n.GetPrefix().GetTarget()
		var gp *gpb.Path
		if tt.n.Update != nil {
			gp = tt.n.Update[0].GetPath()
		} else {
			gp = tt.n.Delete[0]
		}
		p := joinPrefixAndPath(tt.n.GetPrefix(), gp)
		var results []interface{}
		appendResults := func(_ []string, _ *ctree.Leaf, val interface{}) { results = append(results, val) }
		c.Query(target, p, appendResults)
		if len(results) != 1 {
			if tt.e != "" {
				t.Errorf("Query(%s, %v, ): got %d results, want 1", target, p, len(results))
			}
			continue
		}
		val := results[0].(*gpb.Notification).Update[0].Val.Value.(*gpb.TypedValue_StringVal).StringVal
		if val != tt.e {
			t.Errorf("#%d: got %q, want %q", x, val, tt.e)
		}
	}
}

func TestGNMIClient(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})
	var got []interface{}
	c.SetClient(func(n *ctree.Leaf) {
		got = append(got, n.Value())
	})
	sortGot := func() {
		path := func(i int) []string {
			switch l := got[i].(type) {
			case *gpb.Notification:
				if len(l.Update) > 0 {
					return joinPrefixAndPath(l.Prefix, l.Update[0].Path)
				}
				return joinPrefixAndPath(l.Prefix, l.Delete[0])
			default:
				return nil
			}
		}
		sort.Slice(got, func(i, j int) bool {
			return client.Path(path(i)).Less(client.Path(path(j)))
		})
	}

	tests := []struct {
		desc    string
		updates []*gpb.Notification
		want    []interface{}
	}{
		{
			desc: "add new nodes",
			updates: []*gpb.Notification{
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 1, "value1", false),
				gnmiNotification("dev1", []string{}, []string{"a", "c"}, 2, "value2", false),
				gnmiNotification("dev1", []string{}, []string{"a", "d"}, 3, "value3", false),
			},
			want: []interface{}{
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 1, "value1", false),
				gnmiNotification("dev1", []string{}, []string{"a", "c"}, 2, "value2", false),
				gnmiNotification("dev1", []string{}, []string{"a", "d"}, 3, "value3", false),
			},
		},
		{
			desc: "update nodes",
			updates: []*gpb.Notification{
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 0, "value1", false),
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 1, "value1", false),
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 2, "value11", false),
			},
			want: []interface{}{
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 2, "value11", false),
			},
		},
		{
			desc: "delete nodes",
			updates: []*gpb.Notification{
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 1, "", true),
				gnmiNotification("dev1", []string{}, []string{"a", "b"}, 3, "", true),
				gnmiNotification("dev1", []string{}, []string{"a"}, 4, "", true),
			},
			want: []interface{}{
				gnmiNotification("dev1", nil, []string{"a", "b"}, 3, "", true),
				gnmiNotification("dev1", nil, []string{"a", "c"}, 4, "", true),
				gnmiNotification("dev1", nil, []string{"a", "d"}, 4, "", true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got = nil
			for _, u := range tt.updates {
				c.GnmiUpdate(u)
			}
			sortGot()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("sent updates: %v\ndiff in received updates:\n%s", tt.updates, diff)
			}
		})
	}
	t.Run("remove target", func(t *testing.T) {
		got = nil
		want := deleteNoti("dev1", "", []string{"*"})
		want.Timestamp = 0
		c.Remove("dev1")
		if len(got) != 1 {
			t.Fatalf("Remove didn't produce correct client update, got: %+v, want: %+v", got, []interface{}{want})
		}
		gotVal := got[0].(*gpb.Notification)
		// Clear timestamp before comparison.
		gotVal.Timestamp = 0
		if diff := cmp.Diff(want, gotVal); diff != "" {
			t.Errorf("diff in received update:\n%s", diff)
		}
	})
}

func TestGNMIReset(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	targets := []string{"dev1", "dev2", "dev3"}
	c := New(targets)
	updates := map[string]*gpb.Notification{
		"value1":  gnmiNotification("dev1", []string{}, []string{"a", "b"}, 0, "value1", false),
		"value2":  gnmiNotification("dev2", []string{}, []string{"a", "c"}, 0, "value2", false),
		"value3":  gnmiNotification("dev3", []string{}, []string{"a", "d"}, 0, "value3", false),
		"invalid": gnmiNotification("", []string{}, []string{}, 0, "", false), // Should have no effect on test.
	}
	// Add updates to cache.
	for _, u := range updates {
		c.GnmiUpdate(u)
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
		c.Reset(target)
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

func TestGNMISyncConnectUpdates(t *testing.T) {
	Type = GnmiNoti
	defer func() { Type = ClientLeaf }()
	c := New([]string{"dev1"})
	var got []interface{}
	c.SetClient(func(l *ctree.Leaf) {
		got = append(got, l.Value())
	})
	tests := []struct {
		metadata string
		helper   func(string)
		want     []*gpb.Notification
	}{
		{metadata: metadata.Sync, helper: c.Sync, want: []*gpb.Notification{metaNotiBool("dev1", metadata.Sync, true)}},
		{metadata: metadata.Connected, helper: c.Connect, want: []*gpb.Notification{metaNotiBool("dev1", metadata.Connected, true)}},
		{metadata: "disconnect", helper: c.Disconnect, want: []*gpb.Notification{metaNotiBool("dev1", metadata.Sync, false), metaNotiBool("dev1", metadata.Connected, false)}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Test %v", tt.metadata), func(t *testing.T) {
			tt.helper("dev1")
			if len(got) < len(tt.want) {
				t.Fatalf("got %d updates, want %d. got %v, want %v", len(got), len(tt.want), got, tt.want)
			}
			for i := 0; i < len(tt.want); i++ {
				got[i].(*gpb.Notification).Timestamp = 0
				tt.want[i].Timestamp = 0
				if diff := cmp.Diff(tt.want[i], got[i]); diff != "" {
					t.Errorf("diff in received update:\n%s", diff)
				}
			}
			got = nil
		})
	}
}

func TestGNMIUpdate(t *testing.T) {
	type state struct {
		deleted   bool
		path      []string
		timestamp int
		val       string
	}

	dev := "dev1"
	prefix := []string{"prefix1"}
	path1 := []string{"path1"}
	path2 := []string{"path2"}
	path3 := []string{"path3"}
	tests := []struct {
		desc           string
		initial        *gpb.Notification
		notification   *gpb.Notification
		want           []state
		wantUpdates    int
		wantSuppressed int
		wantStale      int
		wantErr        bool
	}{
		{
			desc: "duplicate update",
			initial: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "2",
				},
			}),
			notification: notificationBundle(dev, prefix, 1, []update{
				{
					path: path1,
					val:  "11",
				}, {
					path: path2,
					val:  "2",
				},
			}),
			want: []state{
				{
					path:      path1,
					val:       "11",
					timestamp: 1,
				}, {
					path:      path2,
					val:       "2",
					timestamp: 1,
				},
			},
			wantUpdates:    1,
			wantSuppressed: 1,
			wantErr:        true,
		}, {
			desc: "no duplicates",
			initial: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "2",
				},
			}),
			notification: notificationBundle(dev, prefix, 1, []update{
				{
					path: path1,
					val:  "11",
				}, {
					path: path2,
					val:  "22",
				},
			}),
			want: []state{
				{
					path:      path1,
					val:       "11",
					timestamp: 1,
				}, {
					path:      path2,
					val:       "22",
					timestamp: 1,
				},
			},
			wantUpdates: 2,
		}, {
			desc: "duplicate update with deletes",
			initial: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "2",
				}, {
					path: path3,
					val:  "3",
				},
			}),
			notification: notificationBundle(dev, prefix, 1, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path:   path2,
					delete: true,
				}, {
					path:   path3,
					delete: true,
				},
			}),
			want: []state{
				{
					path:      path1,
					val:       "1",
					timestamp: 1,
				}, {
					path:    path2,
					deleted: true,
				}, {
					path:    path3,
					deleted: true,
				},
			},
			wantUpdates:    2,
			wantSuppressed: 1,
			wantErr:        true,
		}, {
			desc: "stale updates - same timestamp",
			initial: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "2",
				},
			}),
			notification: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "22",
				},
			}),
			want: []state{
				{
					path:      path1,
					val:       "1",
					timestamp: 0,
				}, {
					path:      path2,
					val:       "2",
					timestamp: 0,
				},
			},
			wantStale: 2,
			wantErr:   true,
		}, {
			desc: "stale updates - later timestamp",
			initial: notificationBundle(dev, prefix, 1, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "2",
				},
			}),
			notification: notificationBundle(dev, prefix, 0, []update{
				{
					path: path1,
					val:  "1",
				}, {
					path: path2,
					val:  "22",
				},
			}),
			want: []state{
				{
					path:      path1,
					val:       "1",
					timestamp: 1,
				}, {
					path:      path2,
					val:       "2",
					timestamp: 1,
				},
			},
			wantStale: 2,
			wantErr:   true,
		},
	}

	suppressedPath := metadata.Path(metadata.SuppressedCount)
	updatePath := metadata.Path(metadata.UpdateCount)
	stalePath := metadata.Path(metadata.StaleCount)

	for _, tt := range tests {
		c := New([]string{dev})
		if err := c.GnmiUpdate(tt.initial); err != nil {
			t.Fatalf("%v: Could not initialize cache: %v ", tt.desc, err)
		}
		c.GetTarget(dev).meta.Clear()

		err := c.GnmiUpdate(tt.notification)
		c.UpdateMetadata()

		if err != nil && !tt.wantErr {
			t.Errorf("%v: GnmiUpdate(%v) = %v, want no error", tt.desc, tt.notification, err)
			continue
		}
		if err == nil && tt.wantErr {
			t.Errorf("%v: GnmiUpdate(%v) = nil, want error", tt.desc, tt.notification)
			continue
		}

		checkMeta := func(desc string, path []string, want int) {
			c.Query(dev, path, func(_ []string, _ *ctree.Leaf, v interface{}) {
				got := v.(client.Update).Val.(int64)
				if got != int64(want) {
					t.Errorf("%v: got %v = %d, want %d", desc, path, got, want)
				}
			})
		}

		checkState := func(desc string, states []state) {
			for _, s := range states {
				c.Query(dev, s.path, func(_ []string, _ *ctree.Leaf, v interface{}) {
					if s.deleted {
						t.Errorf("%v: Query(%p): got %v, want none", desc, s.path, v)
					} else {
						want := gnmiNotification(dev, prefix, s.path, int64(s.timestamp), s.val, false)
						if got, ok := v.(*gpb.Notification); !ok || !proto.Equal(got, want) {
							t.Errorf("%v: got:\n%s want\n%s", desc, proto.MarshalTextString(got), proto.MarshalTextString(want))
						}
					}
				})
			}
		}

		checkState(tt.desc, tt.want)
		checkMeta(tt.desc, suppressedPath, tt.wantSuppressed)
		checkMeta(tt.desc, updatePath, tt.wantUpdates)
		checkMeta(tt.desc, stalePath, tt.wantStale)
	}
}
