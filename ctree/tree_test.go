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

package ctree

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestAdd(t *testing.T) {
	tr := &Tree{}
	if err := tr.Add([]string{}, "foo"); err != nil {
		t.Error(err)
	}
	if err := tr.Add([]string{"a"}, "foo"); err == nil {
		t.Error("got nil, expected error adding a leaf to a leaf")
	}
	tr = &Tree{}
	if err := tr.Add([]string{"a"}, "foo"); err != nil {
		t.Error(err)
	}
	if err := tr.Add([]string{}, "foo"); err == nil {
		t.Error("got nil, want error adding leaf in place of a branch")
	}
	if err := tr.Add([]string{"a", "b"}, "foo"); err == nil {
		t.Error("got nil, want error adding a leaf to a leaf")
	}
	if err := tr.Add([]string{"b", "c", "d", "e"}, "foo"); err != nil {
		t.Error(err)
	}
}

func TestSlowAdd(t *testing.T) {
	for _, test := range []struct {
		tree      *Tree
		path      []string
		expectErr bool
	}{
		{
			tree:      &Tree{leafBranch: "not a branch"},
			path:      []string{"a"},
			expectErr: true,
		},
		{
			tree:      &Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			path:      []string{"a"},
			expectErr: false,
		},
	} {
		testVal := "testVal"
		err := test.tree.slowAdd(test.path, testVal)
		gotErr := err != nil
		if gotErr != test.expectErr {
			t.Errorf("slowAdd(%v, %v) = %v, want %v", test.path, testVal, gotErr, test.expectErr)
		}
	}
}

func TestTreeGetLeafValue(t *testing.T) {
	tr := &Tree{}
	for x, tt := range []struct {
		path  []string
		value string
	}{
		{[]string{"a", "b"}, "value0"},
		{[]string{"a", "c"}, "value1"},
		{[]string{"a", "d"}, "value2"},
		{[]string{"b"}, "value2"},
		{[]string{"c", "a"}, "value3"},
		{[]string{"c", "b", "a"}, "value4"},
		{[]string{"c", "d"}, "value5"},
	} {
		// Value shouldn't exist before addition.
		if value := tr.GetLeafValue(tt.path); nil != value {
			t.Errorf("#%d: got %v, expected %v", x, value, nil)
		}
		if err := tr.Add(tt.path, tt.value); err != nil {
			t.Error(err)
		}
		value := tr.GetLeafValue(tt.path)
		// Value should exist on successful addition.
		if tt.value != value {
			t.Errorf("#%d: got %v, expected %v", x, value, tt.value)
		}
	}
}

var testPaths = [][]string{
	[]string{"a", "b", "c"},
	[]string{"a", "d"},
	[]string{"b", "a", "d"},
	[]string{"b", "c", "d"},
	[]string{"c", "d", "e", "f", "g", "h", "i"},
	[]string{"d"},
}

func buildTree(t *Tree) {
	buildTreePaths(t, testPaths)
}

func buildTreePaths(t *Tree, paths [][]string) {
	for _, path := range paths {
		value := strings.Join(path, "/")
		t.Add(path, value)
	}
}

type expectedQuery struct {
	query   []string
	results map[string]interface{}
}

func TestQuery(t *testing.T) {
	tr := &Tree{}
	results := make(map[string]interface{})
	appendResults := func(p []string, _ *Leaf, v interface{}) { results[strings.Join(p, "/")] = v }

	tr.Query([]string{"*"}, appendResults)
	if len(results) > 0 {
		t.Errorf("tr.Query: got %d results, expected 0", len(results))
	}

	buildTree(tr)
	// Test a set of queries.
	for x, tt := range []expectedQuery{
		{[]string{"a", "d"}, map[string]interface{}{"a/d": "a/d"}},
		{[]string{"a"}, map[string]interface{}{"a/d": "a/d", "a/b/c": "a/b/c"}},
		// A trailing glob is equivalent to a query without it, as above.
		{[]string{"a", "*"}, map[string]interface{}{"a/d": "a/d", "a/b/c": "a/b/c"}},
		{[]string{"*"}, map[string]interface{}{"a/d": "a/d", "a/b/c": "a/b/c", "b/c/d": "b/c/d", "b/a/d": "b/a/d", "c/d/e/f/g/h/i": "c/d/e/f/g/h/i", "d": "d"}},
		{[]string{"*", "*", "d"}, map[string]interface{}{"b/c/d": "b/c/d", "b/a/d": "b/a/d"}},
		{[]string{"c", "d", "e"}, map[string]interface{}{"c/d/e/f/g/h/i": "c/d/e/f/g/h/i"}},
	} {
		results = make(map[string]interface{})
		tr.Query(tt.query, appendResults)

		if !reflect.DeepEqual(results, tt.results) {
			t.Errorf("#%d: got results: %v\nwant results: %v", x, results, tt.results)
		}
	}
}

func TestUpdateLeaf(t *testing.T) {
	tr := &Tree{}
	buildTree(tr)

	l := tr.GetLeaf(testPaths[0])

	nv := "new value"
	tr.Add(testPaths[0], nv)

	if got := l.Value(); got != nv {
		t.Errorf("Value on updated leaf returned %+v, want %+v", got, nv)
	}
}

func TestWalk(t *testing.T) {
	tr := &Tree{}
	buildTree(tr)
	paths := [][]string{}
	tr.Walk(func(path []string, _ *Leaf, value interface{}) {
		got, want := value.(string), strings.Join(path, "/")
		if got != want {
			t.Errorf("Walk got value %q, want %q", got, want)
		}
		paths = append(paths, path)
	})
	if got, want := len(paths), len(testPaths); got != want {
		t.Errorf("Walk got %d paths, want %d", got, want)
	}
gotpaths:
	for _, p := range paths {
		for _, tp := range testPaths {
			if reflect.DeepEqual(p, tp) {
				continue gotpaths
			}
		}
		t.Errorf("Walk got path %q, wanted one of %q", p, testPaths)
	}
wantpaths:
	for _, tp := range testPaths {
		for _, p := range paths {
			if reflect.DeepEqual(p, tp) {
				continue wantpaths
			}
		}
		t.Errorf("Walk got paths %q, want %q included", paths, tp)
	}
}

func TestWalkSorted(t *testing.T) {
	tr := &Tree{}
	buildTree(tr)
	paths := [][]string{}
	tr.WalkSorted(func(path []string, _ *Leaf, value interface{}) {
		got, want := value.(string), strings.Join(path, "/")
		if got != want {
			t.Errorf("WalkSorted got value %q, want %q", got, want)
		}
		paths = append(paths, path)
	})
	if got, want := len(paths), len(testPaths); got != want {
		t.Errorf("WalkSorted got %d paths, want %d", got, want)
	}
	if !reflect.DeepEqual(paths, testPaths) {
		t.Errorf("WalkSorted got %q, want %q", paths, testPaths)
	}
}

func TestEmptyWalk(t *testing.T) {
	tr := &Tree{}
	tr.Walk(func(_ []string, _ *Leaf, _ interface{}) {
		t.Error("Walk on empty tree should not call func.")
	})
	tr.WalkSorted(func(_ []string, _ *Leaf, _ interface{}) {
		t.Error("WalkSorted on empty tree should not call func.")
	})
}

type expectedDelete struct {
	subpath []string
	leaves  map[string]bool
}

func TestDelete(t *testing.T) {
	tr := &Tree{}
	deleted := tr.Delete([]string{"a", "b"})
	if len(deleted) > 0 {
		t.Errorf("Delete on empty tree should return empty slice.")
	}
	for x, tt := range []expectedDelete{
		{[]string{"x"}, map[string]bool{}},
		// root delete with glob appended for a non-existing root
		{[]string{"x", "*"}, map[string]bool{}},
		{[]string{"d"}, map[string]bool{"d": true}},
		{[]string{"a"}, map[string]bool{"a/d": true, "a/b/c": true}},
		// root delete with a glob appended
		{[]string{"a", "*"}, map[string]bool{"a/d": true, "a/b/c": true}},
		{[]string{"b", "c", "d"}, map[string]bool{"b/c/d": true}},
		// delete with glob in the middle of the path
		{[]string{"b", "*", "d"}, map[string]bool{"b/a/d": true, "b/c/d": true}},
		// delete with glob in the middle of the path for a non-existing path
		{[]string{"b", "*", "x"}, map[string]bool{}},
		{[]string{"b"}, map[string]bool{"b/a/d": true, "b/c/d": true}},
		{[]string{"c", "d", "e"}, map[string]bool{"c/d/e/f/g/h/i": true}},
		{[]string{}, map[string]bool{"a/d": true, "a/b/c": true, "b/a/d": true, "b/c/d": true, "c/d/e/f/g/h/i": true, "d": true}},
		// just glob in the path to delete all the tree
		{[]string{"*"}, map[string]bool{"a/d": true, "a/b/c": true, "b/a/d": true, "b/c/d": true, "c/d/e/f/g/h/i": true, "d": true}},
	} {
		// Rebuild tree for each query.
		buildTree(tr)
		for _, leaf := range tr.Delete(tt.subpath) {
			leafpath := strings.Join(leaf, "/")
			if _, ok := tt.leaves[leafpath]; !ok {
				t.Errorf("#%d: unexpected deleted leaf %v", x, leaf)
			}
			delete(tt.leaves, leafpath)
		}
		if len(tt.leaves) > 0 {
			t.Errorf("#%d: expected leaves missing from return: %v", x, tt.leaves)
		}
	}
	if tr.leafBranch != nil {
		t.Errorf("tree should be empty, but root still has branches %#v", tr.leafBranch)
	}
}

func TestDeleteConditional(t *testing.T) {
	never := func(interface{}) bool { return false }
	tr := &Tree{}
	buildTree(tr)
	leaves := tr.DeleteConditional([]string{}, never)
	if len(leaves) > 0 {
		t.Errorf("Leaves deleted for false condition: %v", leaves)
	}
	always := func(interface{}) bool { return true }
	// This is the same test as the last case for TestDelete, above.
	leaves = tr.DeleteConditional([]string{}, always)
	if len(leaves) != 6 {
		t.Errorf("Not all leaves deleted: %v", leaves)
	}
	valEqualsD := func(v interface{}) bool { return v == "d" }
	buildTree(tr)
	leaves = tr.DeleteConditional([]string{}, valEqualsD)
	if expected := [][]string{[]string{"d"}}; !reflect.DeepEqual(expected, leaves) {
		t.Errorf("got %v, expected %v", leaves, expected)
	}
	if v := tr.GetLeafValue([]string{"d"}); nil != v {
		t.Errorf("got %v, expected %v", v, nil)
	}
}

type expectedTreeEqual struct {
	t1    *Tree
	t2    *Tree
	equal bool
}

func TestEqual(t *testing.T) {
	for x, tt := range []expectedTreeEqual{
		{nil, nil, true},
		{nil, &Tree{}, false},
		{&Tree{}, nil, false},
		{&Tree{}, &Tree{}, true},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			&Tree{},
			false},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			true},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: "b"}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			false},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: "b"}}},
			false},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: "b"}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: "b"}}},
			true},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: branch{"b/c": &Tree{leafBranch: "b/c"}}}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: "b"}}},
			false},
		{&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: branch{"b/c": &Tree{leafBranch: "b/c"}}}}},
			&Tree{leafBranch: branch{"a": &Tree{leafBranch: "a"}, "b": &Tree{leafBranch: branch{"b/c": &Tree{leafBranch: "b/c"}}}}},
			true},
	} {
		if equal := reflect.DeepEqual(tt.t1, tt.t2); tt.equal != equal {
			t.Errorf("#%d: got %t, expected %t", x, equal, tt.equal)
		}
	}
}

func generatePaths(count int) [][]string {
	paths := [][]string{}
	for c := 0; c < count; c++ {
		p := []string{}
		for d := 3; d < 16; d++ {
			p = append(p, string(c%d+65))
		}
		paths = append(paths, p)
	}
	return paths
}

func buildTreeRange(wg *sync.WaitGroup, t *Tree, paths [][]string, index, modulus int) {
	if wg != nil {
		defer wg.Done()
	}
	for i := index; i < len(paths); i += modulus {
		t.Add(paths[i], strings.Join(paths[i], "/"))
	}
}

func TestParallelAdd(t *testing.T) {
	trees := []*Tree{}
	paths := generatePaths(10000)
	wg := new(sync.WaitGroup)
	r := runtime.GOMAXPROCS(8)
	for _, tt := range []int{1, 2, 4, 8} {
		atree := &Tree{}
		trees = append(trees, atree)
		for i := 0; i < tt; i++ {
			wg.Add(1)
			go buildTreeRange(wg, atree, paths, i, tt)
		}
	}
	wg.Wait()
	runtime.GOMAXPROCS(r)
	for i := 1; i < len(trees); i++ {
		if !reflect.DeepEqual(trees[0], trees[i]) {
			t.Errorf("tree %d does not equal serially created tree.", i)
		}
	}
}

func deleteTreeRange(wg *sync.WaitGroup, t *Tree, paths [][]string, index, modulus int) {
	defer wg.Done()
	for i := index; i < len(paths); i += modulus {
		t.Delete(paths[i])
	}
}

func TestParallelDelete(t *testing.T) {
	trees := []*Tree{}
	paths := generatePaths(100000)
	wg := new(sync.WaitGroup)
	r := runtime.GOMAXPROCS(8)
	parallelSize := []int{2, 4, 8}
	for _, tt := range parallelSize {
		atree := &Tree{}
		trees = append(trees, atree)
		for i := 0; i < tt; i++ {
			wg.Add(1)
			go buildTreeRange(wg, atree, paths, i, tt)
		}
	}
	wg.Wait()
	for x, tt := range parallelSize {
		atree := trees[x]
		for i := 0; i < tt; i++ {
			wg.Add(1)
			go deleteTreeRange(wg, atree, paths, i, tt)
		}
	}
	wg.Wait()
	runtime.GOMAXPROCS(r)
	emptyTree := &Tree{}
	for i := 0; i < len(trees); i++ {
		if !reflect.DeepEqual(trees[i], emptyTree) {
			t.Errorf("tree %d does not equal empty tree. %#v != %#v", i, trees[i], emptyTree)
		}
	}
}

func query(tr *Tree, path []string) (ret []interface{}) {
	tr.Query(path, func(_ []string, _ *Leaf, val interface{}) {
		ret = append(ret, val)
	})
	return ret
}

func queryGetTreeRange(t *testing.T, wg *sync.WaitGroup, tr *Tree, paths [][]string, index, modulus int) {
	defer wg.Done()
	for i := index; i < len(paths); i += modulus {
		// Test get of paths.
		got := tr.GetLeafValue(paths[i])
		if want := strings.Join(paths[i], "/"); got != want {
			t.Errorf("Get(%v): got result %s, want %s using %d threads", paths[i], got, want, modulus)
		}
		// Test query of whole paths.
		results := query(tr, paths[i])
		if n := len(results); n != 1 {
			t.Errorf("Query(%v): got %d results, want 1", paths[i], n)
		}
		for _, got := range results {
			if want := strings.Join(paths[i], "/"); got != want {
				t.Errorf("got result %s, want %s using %d threads", got, want, modulus)
			}
		}
		// Test query of partial paths.
		results = query(tr, paths[i][:3])
		if n := len(results); n < 1 {
			t.Errorf("Query(%v): got %d results, want >= 1", paths[i], n)
		}
		for _, got := range results {
			if want := strings.Join(paths[i][0:1], "/"); !strings.HasPrefix(got.(string), want) {
				t.Errorf("got result %s, want to have prefix %s using %d threads", got, want, modulus)
			}
		}
	}
}

func TestParallelQueryGet(t *testing.T) {
	paths := generatePaths(10000)
	r := runtime.GOMAXPROCS(8)
	atree := &Tree{}
	buildTreeRange(nil, atree, paths, 0, 1)
	wg := new(sync.WaitGroup)
	for _, tt := range []int{2, 4, 8} {
		for i := 0; i < tt; i++ {
			wg.Add(1)
			go queryGetTreeRange(t, wg, atree, paths, i, tt)
		}
	}
	wg.Wait()
	runtime.GOMAXPROCS(r)
}

func makePath(i int64) []string {
	path := []string{}
	for depth := 0; depth < 5; depth++ {
		path = append(path, fmt.Sprintf("%d", i&15))
		i >>= 4
	}
	return path
}

func makePathValue(i int64) ([]string, *string) {
	value := fmt.Sprintf("value_%d", i)
	return makePath(i), &value
}

func BenchmarkTreeParallelAddNew(b *testing.B) {
	t := &Tree{}
	var x int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.Add(makePathValue(atomic.AddInt64(&x, 1)))
		}
	})
}

func BenchmarkTreeParallelAddUpdate(b *testing.B) {
	t := &Tree{}
	for i := 0; i < b.N; i++ {
		t.Add(makePathValue(int64(i)))
	}
	// Only time the updates to already existing keys.
	b.ResetTimer()
	var x int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.Add(makePathValue(atomic.AddInt64(&x, 1)))
		}
	})
}

func BenchmarkTreeParallelGet(b *testing.B) {
	t := &Tree{}
	for i := 0; i < b.N; i++ {
		t.Add(makePathValue(int64(i)))
	}
	// Only time the Get calls.
	b.ResetTimer()
	var x int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.GetLeafValue(makePath(atomic.AddInt64(&x, 1)))
		}
	})
}

func BenchmarkTreeParallelQuerySingle(b *testing.B) {
	t := &Tree{}
	for i := 1; i <= b.N; i++ {
		t.Add(makePathValue(int64(i)))
	}
	var count int64
	// Create a query channel.
	c := make(chan interface{}, 1000)
	// Create parallel consumers.
	var wg sync.WaitGroup
	collect := func(c <-chan interface{}) {
		for r := range c {
			if r != nil {
				atomic.AddInt64(&count, 1)
			}
		}
		wg.Done()
	}
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go collect(c)
	}
	// Only time the Query calls.
	b.ResetTimer()
	var x int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t.Query(makePath(atomic.AddInt64(&x, 1)), func(_ []string, _ *Leaf, val interface{}) { c <- val })
		}
	})
	close(c)
	wg.Wait()
	b.Logf("Query result count: %v", count)
}

func BenchmarkTreeParallelQueryMany(b *testing.B) {
	t := &Tree{}
	for i := 1; i <= b.N; i++ {
		t.Add(makePathValue(int64(i)))
	}
	var count int64
	// Create a query channel.
	c := make(chan interface{}, 1000)
	// Create parallel consumers.
	var wg sync.WaitGroup
	collect := func(c <-chan interface{}) {
		for r := range c {
			if r != nil {
				atomic.AddInt64(&count, 1)
			}
		}
		wg.Done()
	}
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go collect(c)
	}
	// Only time the Query calls.
	b.ResetTimer()
	var x int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// For each query, use only a subpath to retrieve multiple elements.
			t.Query(makePath(atomic.AddInt64(&x, 1))[0:3], func(_ []string, _ *Leaf, val interface{}) { c <- val })
		}
	})
	close(c)
	wg.Wait()
	b.Logf("Query result count: %v", count)
}

func TestIsBranch(t *testing.T) {
	tr := &Tree{}
	buildTree(tr)
	for _, test := range []struct {
		node *Tree
		want bool
	}{
		{
			node: tr,
			want: true,
		}, {
			node: tr.Get(testPaths[0]),
			want: false,
		}, {
			node: nil,
			want: false,
		},
	} {
		got := test.node.IsBranch()
		if got != test.want {
			t.Errorf("IsBranch(%v) = %v, want %v", test.node, got, test.want)
		}
	}
}

func TestGet(t *testing.T) {
	tr := &Tree{}
	testPaths = [][]string{
		[]string{"a", "b", "c"},
		[]string{"a", "d"},
		[]string{"d"},
	}
	buildTreePaths(tr, testPaths)
	for _, test := range []struct {
		path []string
		want *Tree
	}{
		{
			path: []string{"a"},
			want: &Tree{leafBranch: branch{
				"b": &Tree{leafBranch: branch{
					"c": &Tree{leafBranch: "a/b/c"}}},
				"d": &Tree{leafBranch: "a/d"}}},
		}, {
			path: []string{"a", "d"},
			want: &Tree{leafBranch: "a/d"},
		}, {
			path: []string{},
			want: &Tree{leafBranch: branch{
				"a": &Tree{leafBranch: branch{
					"b": &Tree{leafBranch: branch{
						"c": &Tree{leafBranch: "a/b/c"}}},
					"d": &Tree{leafBranch: "a/d"}}},
				"d": &Tree{leafBranch: "d"}}},
		}, {
			path: []string{"non existent path"},
			want: nil,
		},
	} {
		got := tr.Get(test.path)
		if diff := pretty.Compare(got, test.want); diff != "" {
			t.Errorf("Get(%v) returned diff (-got +want):\n%s", test.path, diff)
		}
	}
}

func TestGetChildren(t *testing.T) {
	tr := &Tree{}
	testPaths = [][]string{
		[]string{"a", "b", "c"},
		[]string{"a", "d"},
		[]string{"d"},
	}
	buildTreePaths(tr, testPaths)
	for _, test := range []struct {
		node *Tree
		want branch
	}{
		{
			node: tr,
			want: branch{
				"a": &Tree{leafBranch: branch{
					"b": &Tree{leafBranch: branch{
						"c": &Tree{leafBranch: "a/b/c"}}},
					"d": &Tree{leafBranch: "a/d"}}},
				"d": &Tree{leafBranch: "d"},
			},
		}, {
			node: tr.Get(testPaths[0]), // Leaf.
			want: nil,
		}, {
			node: nil,
			want: nil,
		},
	} {
		got := test.node.Children()
		if diff := pretty.Compare(got, test.want); diff != "" {
			t.Errorf("Children(%s) returned diff (-got +want):\n%s", test.node, diff)
		}
	}
}

func TestTreeValue(t *testing.T) {
	for _, test := range []struct {
		node *Tree
		want interface{}
	}{
		{
			node: &Tree{leafBranch: branch{}},
			want: nil,
		}, {
			node: &Tree{leafBranch: "val1"},
			want: "val1",
		}, {
			node: nil,
			want: nil,
		},
	} {
		got := test.node.Value()
		if got != test.want {
			t.Errorf("Value(%v) = %v, want %v", test.node, got, test.want)
		}
	}
}

func TestLeafValue(t *testing.T) {
	for _, test := range []struct {
		node *Leaf
		want interface{}
	}{
		{
			node: &Leaf{leafBranch: "val1"},
			want: "val1",
		}, {
			node: nil,
			want: nil,
		},
	} {
		got := test.node.Value()
		if got != test.want {
			t.Errorf("Value(%v) = %v, want %v", test.node, got, test.want)
		}
	}
}

func TestString(t *testing.T) {
	tr := &Tree{}
	testPaths = [][]string{
		[]string{"a", "b", "c"},
		[]string{"a", "d"},
		[]string{"d"},
	}
	buildTreePaths(tr, testPaths)
	for _, test := range []struct {
		node *Tree
		want string
	}{
		{
			node: tr,
			want: `{ "a": { "b": { "c": "a/b/c" }, "d": "a/d" }, "d": "d" }`,
		}, {
			node: tr.Get([]string{"a"}),
			want: `{ "b": { "c": "a/b/c" }, "d": "a/d" }`,
		}, {
			node: tr.Get([]string{"d"}),
			want: `"d"`,
		}, {
			node: nil,
			want: "",
		},
	} {
		// Test explicitly.
		got := test.node.String()
		if got != test.want {
			t.Errorf("String\n\tgot:  %q\n\twant: %q", got, test.want)
		}
		// Test via string format specifier.
		got = fmt.Sprintf("%s", test.node)
		if got != test.want {
			t.Errorf("string format specifier\n\tgot:  %q\n\twant: %q", got, test.want)
		}
	}
}
