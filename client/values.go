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

package client

import "time"

// Leaf represents a leaf value in the tree. It includes the path to the node in
// the tree. This is returned via Leaves. It is also the basis for all
// Notification types that are used by the NotificationHandler callbacks see
// "notification.go".
type Leaf struct {
	Path Path
	Val  interface{}
	TS   time.Time // TS is the timestamp of last update to this leaf.
	Dups uint32    // Dups is the number of coalesced duplicates for this leaf.
}

// Leaves implements sort.Interface over []Leaf based on paths.
type Leaves []Leaf

func (pv Leaves) Len() int      { return len(pv) }
func (pv Leaves) Swap(i, j int) { pv[i], pv[j] = pv[j], pv[i] }
func (pv Leaves) Less(i, j int) bool {
	return pv[i].Path.Less(pv[j].Path)
}

// TreeVal contains the current branch's value and the timestamp when the
// node was last updated.
type TreeVal struct {
	Val interface{} `json:"value"`
	TS  time.Time   `json:"timestamp"`
}

// Path is a standard type for describing a path inside of streaming telemetry
// tree.
type Path []string

// Less returns true if p sorts before p2.
func (p Path) Less(p2 Path) bool {
	for x := 0; x < len(p) && x < len(p2); x++ {
		if p[x] < p2[x] {
			return true
		}
		if p[x] > p2[x] {
			return false
		}
	}
	return len(p) < len(p2)
}

// Equal returns true if p is equivalent to p2.
func (p Path) Equal(p2 Path) bool {
	if len(p) != len(p2) {
		return false
	}
	for x := 0; x < len(p); x++ {
		if p[x] != p2[x] {
			return false
		}
	}
	return true
}
