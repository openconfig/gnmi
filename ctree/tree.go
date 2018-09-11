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

// Package ctree implements a Tree container whose methods are all thread-safe
// allowing concurrent access for multiple goroutines. This container was
// designed to support concurrent reads using sync.RWLock and are optimized for
// situations where reads and updates to previously Added values are
// dominant.  The acquisition of write locks is avoided wherever possible.
package ctree

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type branch map[string]*Tree

// Tree is a thread-safe container.
type Tree struct {
	mu sync.RWMutex
	// Each node is either a leaf or a branch.
	leafBranch interface{}
}

// Leaf is a Tree node that represents a leaf.
//
// Leaf is safe for use from multiple goroutines and will return the latest
// value.
// This means that if value in this leaf was updated after Leaf was retrieved,
// Value will return the updated content, not the original one.
// This also means that multiple calls to Value may return different results.
type Leaf Tree

// DetachedLeaf returns a Leaf that's not attached to any tree.
func DetachedLeaf(val interface{}) *Leaf {
	return &Leaf{leafBranch: val}
}

// Value returns the latest value stored in this leaf. Value is safe to call on
// nil Leaf.
func (l *Leaf) Value() interface{} {
	if l == nil {
		return nil
	}
	defer l.mu.RUnlock()
	l.mu.RLock()
	return l.leafBranch
}

// Update sets the value of this Leaf to val.
func (l *Leaf) Update(val interface{}) {
	defer l.mu.Unlock()
	l.mu.Lock()
	l.leafBranch = val
}

func newBranch(path []string, value interface{}) *Tree {
	if len(path) == 0 {
		return &Tree{leafBranch: value}
	}
	return &Tree{leafBranch: branch{path[0]: newBranch(path[1:], value)}}
}

// isBranch assumes the calling function holds a lock on t.
func (t *Tree) isBranch() bool {
	_, ok := t.leafBranch.(branch)
	return ok
}

// IsBranch returns whether the Tree node represents a branch.
// Returns false if called on a nil node.
func (t *Tree) IsBranch() bool {
	if t == nil {
		return false
	}
	defer t.mu.RUnlock()
	t.mu.RLock()
	return t.isBranch()
}

// Children returns a mapping of child nodes if current node represents a branch, nil otherwise.
func (t *Tree) Children() map[string]*Tree {
	if t == nil {
		return nil
	}
	defer t.mu.RUnlock()
	t.mu.RLock()
	if t.isBranch() {
		ret := make(branch)
		for k, v := range t.leafBranch.(branch) {
			ret[k] = v
		}
		return ret
	}
	return nil
}

// Value returns the latest value stored in node t if it represents a leaf, nil otherwise. Value is safe to call on
// nil Tree.
func (t *Tree) Value() interface{} {
	if t == nil {
		return nil
	}
	defer t.mu.RUnlock()
	t.mu.RLock()
	if t.isBranch() {
		return nil
	}
	return t.leafBranch
}

// slowAdd will add a new branch to Tree t all the way to the leaf containing
// value, unless another routine added the branch before the write lock was
// acquired, in which case it will return to the normal Add for the remaining
// path that exists, but while still holding the write lock.  This routine is
// called as a fallback by Add when a node does not already exist.
func (t *Tree) slowAdd(path []string, value interface{}) error {
	if t.leafBranch == nil {
		t.leafBranch = branch{}
	}
	switch b := t.leafBranch.(type) {
	case branch:
		// Verify the branch was not added by another routine during the
		// reader/writer lock exchange.
		br := b[path[0]]
		if br == nil {
			br = newBranch(path[1:], value)
			b[path[0]] = br
		}
		return br.Add(path[1:], value)
	default:
		return fmt.Errorf("attempted to add value %#v at path %q which is already a leaf with value %#v", value, path, t.leafBranch)
	}
}

func (t *Tree) terminalAdd(value interface{}) error {
	defer t.mu.Unlock()
	t.mu.Lock()
	if _, ok := t.leafBranch.(branch); ok {
		return fmt.Errorf("attempted to add a leaf in place of a branch")
	}
	t.leafBranch = value
	return nil
}

func (t *Tree) intermediateAdd(path []string, value interface{}) error {
	readerLocked := true
	defer func() {
		if readerLocked {
			t.mu.RUnlock()
		}
	}()
	t.mu.RLock()
	var br *Tree
	switch b := t.leafBranch.(type) {
	case nil:
	case branch:
		br = b[path[0]]
	default:
		return fmt.Errorf("attempted to add value %#v at path %q which is already a leaf with value %#v", value, path, t.leafBranch)
	}
	if br == nil {
		// Exchange the reader lock on t for a writer lock to add new node(s) to the
		// Tree.
		t.mu.RUnlock()
		readerLocked = false
		defer t.mu.Unlock()
		t.mu.Lock()
		return t.slowAdd(path, value)
	}
	return br.Add(path[1:], value)
}

// Add adds value to the Tree at the specified path and returns true on
// success.
func (t *Tree) Add(path []string, value interface{}) error {
	if len(path) == 0 {
		return t.terminalAdd(value)
	}
	return t.intermediateAdd(path, value)
}

// Get returns the Tree node if path points to it, nil otherwise.
// All nodes in path must be fully specified with no globbing (*).
func (t *Tree) Get(path []string) *Tree {
	defer t.mu.RUnlock()
	t.mu.RLock()
	if len(path) == 0 {
		return t
	}
	if b, ok := t.leafBranch.(branch); ok {
		if br := b[path[0]]; br != nil {
			return br.Get(path[1:])
		}
	}
	return nil
}

// GetLeafValue returns the leaf value if path points to a leaf in t, nil otherwise. All
// nodes in path must be fully specified with no globbing (*).
func (t *Tree) GetLeafValue(path []string) interface{} {
	return t.Get(path).Value()
}

// GetLeaf returns the leaf node if path points to a leaf in t, nil otherwise. All
// nodes in path must be fully specified with no globbing (*).
func (t *Tree) GetLeaf(path []string) *Leaf {
	return (*Leaf)(t.Get(path))
}

// VisitFunc is a callback func triggered on leaf values by Query and Walk.
//
// The provided Leaf is the leaf node of the tree, val is the value stored
// inside of it. l can be retained after VisitFunc returns and l.Value() can be
// called to get the latest value for that leaf.
//
// Note that l.Value can *not* be called inside VisitFunc, because the node is
// already locked by Query/Walk.
type VisitFunc func(path []string, l *Leaf, val interface{})

func (t *Tree) enumerateChildren(prefix, path []string, f VisitFunc) {
	// Caller should hold a read lock on t.
	if len(path) == 0 {
		switch b := t.leafBranch.(type) {
		case branch:
			for k, br := range b {
				br.queryInternal(append(prefix, k), path, f)
			}
		default:
			f(prefix, (*Leaf)(t), t.leafBranch)
		}
		return
	}
	if b, ok := t.leafBranch.(branch); ok {
		for k, br := range b {
			br.queryInternal(append(prefix, k), path[1:], f)
		}
	}
}

// Query calls f for all leaves that match a given query where zero or more
// nodes in path may be specified by globs (*). Results and their full paths
// are passed to f as they are found in the Tree. No ordering of paths is
// guaranteed.
func (t *Tree) Query(path []string, f VisitFunc) {
	t.queryInternal(nil, path, f)
}

func (t *Tree) queryInternal(prefix, path []string, f VisitFunc) {
	defer t.mu.RUnlock()
	t.mu.RLock()
	if len(path) == 0 || path[0] == "*" {
		t.enumerateChildren(prefix, path, f)
		return
	}
	if b, ok := t.leafBranch.(branch); ok {
		if br := b[path[0]]; br != nil {
			br.queryInternal(append(prefix, path[0]), path[1:], f)
		}
	}
}

func (t *Tree) walkInternal(path []string, f VisitFunc) {
	defer t.mu.RUnlock()
	t.mu.RLock()
	if b, ok := t.leafBranch.(branch); ok {
		l := len(path)
		for name, br := range b {
			p := make([]string, l, l+1)
			copy(p, path)
			br.walkInternal(append(p, name), f)
		}
		return
	}
	// If this is the root node and it has no children and the value is nil,
	// most likely it's just the zero value of Tree not a valid leaf.
	if len(path) == 0 && t.leafBranch == nil {
		return
	}
	f(path, (*Leaf)(t), t.leafBranch)
}

// Walk calls f for all leaves.
func (t *Tree) Walk(f VisitFunc) {
	t.walkInternal(nil, f)
}

func (t *Tree) walkInternalSorted(path []string, f VisitFunc) {
	defer t.mu.RUnlock()
	t.mu.RLock()
	if b, ok := t.leafBranch.(branch); ok {
		names := make([]string, 0, len(b))
		for name := range b {
			names = append(names, name)
		}
		sort.Strings(names)
		l := len(path)
		for _, name := range names {
			p := make([]string, l, l+1)
			copy(p, path)
			b[name].walkInternalSorted(append(p, name), f)
		}
		return
	}
	// If this is the root node and it has no children and the value is nil,
	// most likely it's just the zero value of Tree not a valid leaf.
	if len(path) == 0 && t.leafBranch == nil {
		return
	}
	f(path, (*Leaf)(t), t.leafBranch)
}

// WalkSorted calls f for all leaves in string sorted order.
func (t *Tree) WalkSorted(f VisitFunc) {
	t.walkInternalSorted(nil, f)
}

// internalDelete removes nodes recursively that match subpath.  It returns true
// if the current node is to be removed from the parent and a slice of subpaths
// ([]string) for all leaves deleted thus far.
func (t *Tree) internalDelete(subpath []string, condition func(interface{}) bool) (bool, [][]string) {
	if len(subpath) == 0 || subpath[0] == "*" {
		if len(subpath) != 0 {
			subpath = subpath[1:]
		}
		// The subpath is a full path to a leaf.
		switch b := t.leafBranch.(type) {
		case branch:
			// The subpath terminates in a branch node and will recursively delete any
			// progeny leaves.
			allLeaves := [][]string{}
			for k, v := range b {
				del, leaves := v.internalDelete(subpath, condition)
				leaf := []string{k}
				for _, l := range leaves {
					allLeaves = append(allLeaves, append(leaf, l...))
				}
				if del {
					delete(b, k)
				}
			}
			return len(t.leafBranch.(branch)) == 0, allLeaves
		default:
			if condition(t.leafBranch) {
				// The second parameter is an empty path that will be filled as recursion
				// unwinds for this leaf that will be deleted in its parent.
				return true, [][]string{[]string{}}
			}
			return false, [][]string{}
		}
	}
	if b, ok := t.leafBranch.(branch); ok {
		// Continue to recurse on subpath while it matches nodes in the Tree.
		if br := b[subpath[0]]; br != nil {
			delBr, allLeaves := br.internalDelete(subpath[1:], condition)
			leaf := []string{subpath[0]}
			// Prepend branch node name to all progeny leaves of branch.
			for i := range allLeaves {
				allLeaves[i] = append(leaf, allLeaves[i]...)
			}
			// Remove branch if requested.
			if delBr {
				delete(b, subpath[0])
			}
			// Node contains no branches so remove this branch node as well.
			if len(b) == 0 {
				return true, allLeaves
			}
			// This node still contains branches, don't delete it.
			return false, allLeaves
		}
	}
	// The subpath doesn't match any Tree branch, return empty list of leaves.
	return false, [][]string{}
}

// DeleteConditional removes all leaves at or below subpath as well as any
// ancestors with no children for those leaves which the given conditional
// function returns true, returning the list of all leaves removed.
// DeleteConditional prevents all other concurrent access.
func (t *Tree) DeleteConditional(subpath []string, condition func(interface{}) bool) [][]string {
	// It is possible that a single delete operation will remove the whole Tree,
	// so only the top level write lock is obtained to prevent concurrent accesses
	// to the entire Tree.
	defer t.mu.Unlock()
	t.mu.Lock()
	delBr, leaves := t.internalDelete(subpath, condition)
	if delBr {
		t.leafBranch = nil
	}
	return leaves
}

// Delete removes all leaves at or below subpath as well as any ancestors with
// no children, returning the list of all leaves removed.  Deletes prevent all
// other concurrent access.
func (t *Tree) Delete(subpath []string) [][]string {
	always := func(interface{}) bool { return true }
	return t.DeleteConditional(subpath, always)
}

// String implements the string interface for Tree returning a stable output
// sorting keys at each level.
func (t *Tree) String() string {
	if t == nil {
		return ""
	}
	defer t.mu.RUnlock()
	t.mu.RLock()
	if t.isBranch() {
		b := t.leafBranch.(branch)
		keys := make([]string, 0, len(b))
		for k := range b {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		children := make([]string, 0, len(b))
		for _, k := range keys {
			children = append(children, fmt.Sprintf("%q: %s", k, b[k]))
		}
		return fmt.Sprintf("{ %s }", strings.Join(children, ", "))
	}
	return fmt.Sprintf("%#v", t.leafBranch)
}
