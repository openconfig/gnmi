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
	"sync"
)

type branch map[string]*Tree

// Tree is a thread-safe container.
type Tree struct {
	mu sync.RWMutex
	// Each node is either a leaf or a branch.
	leafBranch interface{}
}

func newBranch(path []string, value interface{}) *Tree {
	if len(path) == 0 {
		return &Tree{leafBranch: value}
	}
	return &Tree{leafBranch: branch{path[0]: newBranch(path[1:], value)}}
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

// Get returns the leaf value if path points to a leaf in t, nil otherwise. All
// nodes in path must be fully specified with no globbing (*).
func (t *Tree) Get(path []string) interface{} {
	defer t.mu.RUnlock()
	t.mu.RLock()
	if len(path) == 0 {
		if _, ok := t.leafBranch.(branch); ok {
			return nil
		}
		return t.leafBranch
	}
	if b, ok := t.leafBranch.(branch); ok {
		if br := b[path[0]]; br != nil {
			return br.Get(path[1:])
		}
	}
	return nil
}

func (t *Tree) enumerateChildren(prefix, path []string, f func(path []string, value interface{})) {
	// Caller should hold a read lock on t.
	if len(path) == 0 {
		switch b := t.leafBranch.(type) {
		case branch:
			for k, br := range b {
				br.queryInternal(append(prefix, k), path, f)
			}
		default:
			f(prefix, t.leafBranch)
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
func (t *Tree) Query(path []string, f func(path []string, value interface{})) {
	t.queryInternal(nil, path, f)
}

func (t *Tree) queryInternal(prefix, path []string, f func(path []string, value interface{})) {
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

func (t *Tree) walkInternal(path []string, f func(path []string, value interface{})) {
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
	f(path, t.leafBranch)
}

// Walk calls f for all leaves.
func (t *Tree) Walk(f func(path []string, value interface{})) {
	t.walkInternal(nil, f)
}

func (t *Tree) walkInternalSorted(path []string, f func(path []string, value interface{})) {
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
	f(path, t.leafBranch)
}

// WalkSorted calls f for all leaves in string sorted order.
func (t *Tree) WalkSorted(f func(path []string, value interface{})) {
	t.walkInternalSorted(nil, f)
}

// internalDelete removes nodes recursively that match subpath.  It returns true
// if the current node is to be removed from the parent and a slice of subpaths
// ([]string) for all leaves deleted thus far.
func (t *Tree) internalDelete(subpath []string, condition func(interface{}) bool) (bool, [][]string) {
	if len(subpath) == 0 {
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
