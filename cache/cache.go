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

// Package cache is a tree-based cache of timestamped state provided from
// one or more gNMI targets. It accepts updates from the target(s) to
// refresh internal values that are made available to clients via subscriptions.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/metadata"
)

type latency struct {
	mu        sync.Mutex
	totalDiff time.Duration // cumulative difference in timestamps from device
	count     int64         // number of updates in latency count
	min       time.Duration // minimum latency
	max       time.Duration // maximum latency
}

// A targetCache provides parallel Map and Tree indexes to the state of the
// corresponding target.
type targetCache struct {
	name string             // name of the target
	t    *ctree.Tree        // actual cache of target data
	sync bool               // denotes whether this cache is in sync with target
	meta *metadata.Metadata // metadata associated with target
	lat  latency            // latency measurements
	tsmu sync.Mutex         // protects latest timestamp
	ts   time.Time          // latest timestamp for an update
}

// Cache is a structure holding network target state information.
type Cache struct {
	mu      sync.RWMutex
	targets map[string]*targetCache // Map of per target caches.
	client  func(*ctree.Leaf)       // Function to pass all cache updates.
}

// New creates a new instance of Cache that receives target updates from the
// translator and provides an interface to service client queries.
func New(targets []string) *Cache {
	c := &Cache{
		targets: make(map[string]*targetCache, len(targets)),
		client:  func(*ctree.Leaf) {},
	}
	for _, t := range targets {
		c.AddTarget(t)
	}
	return c
}

// SetClient registers a callback function to receive calls for each update
// accepted by the cache.
func (c *Cache) SetClient(client func(*ctree.Leaf)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = client
}

// Metadata returns the per-target metadata structures.
func (c *Cache) Metadata() map[string]*metadata.Metadata {
	md := map[string]*metadata.Metadata{}
	defer c.mu.RUnlock()
	c.mu.RLock()
	for target, cache := range c.targets {
		md[target] = cache.meta
	}
	return md
}

// UpdateMetadata copies the current metadata for each target cache to the
// metadata path within each target cache.
func (c *Cache) UpdateMetadata() {
	c.updateCache((*targetCache).updateMeta)
}

// UpdateSize computes the size of each target cache and updates the size
// metadata reported within the each target cache.
func (c *Cache) UpdateSize() {
	c.updateCache((*targetCache).updateSize)
}

func (c *Cache) getTarget(target string) *targetCache {
	defer c.mu.RUnlock()
	c.mu.RLock()
	return c.targets[target]
}

// HasTarget reports whether the specified target exists in the cache or a glob
// (*) is passed which will match any target (even if no targets yet exist).
func (c *Cache) HasTarget(target string) bool {
	switch target {
	case "":
		return false
	case "*":
		return true
	default:
		defer c.mu.RUnlock()
		c.mu.RLock()
		return c.targets[target] != nil
	}
}

// Query calls the specified callback for all results matching the query. All
// values passed to fn are client.Notification.
func (c *Cache) Query(target string, query []string, fn ctree.VisitFunc) error {
	switch {
	case target == "":
		return errors.New("no target specified in query")
	case target == "*":
		c.mu.RLock()
		// Run the query sequentially for each target cache.
		for _, target := range c.targets {
			target.t.Query(query, fn)
		}
		c.mu.RUnlock()
	default:
		dc := c.getTarget(target)
		if dc == nil {
			return fmt.Errorf("target %q not found in cache", target)
		}
		dc.t.Query(query, fn)
	}
	return nil
}

// AddTarget reserves space in c to receive updates for the specified target.
func (c *Cache) AddTarget(target string) {
	defer c.mu.Unlock()
	c.mu.Lock()
	c.targets[target] = newTargetCache(target)
}

// ResetTarget clears the cache for a target once a connection is resumed after
// having been lost.
func (c *Cache) ResetTarget(target string) {
	if t := c.getTarget(target); t != nil {
		t.reset(c.client)
	}
}

// RemoveTarget removes the space in c corresponding to the specified target.
func (c *Cache) RemoveTarget(target string) {
	defer c.mu.Unlock()
	c.mu.Lock()
	delete(c.targets, target)
	// Notify clients that the target is removed.
	c.client(ctree.DetachedLeaf(client.Delete{Path: []string{target}, TS: time.Now()}))
}

// Update sends a client.Notification into the cache.
func (c *Cache) Update(n client.Notification) error {
	var l client.Leaf
	switch u := n.(type) {
	case client.Update:
		l = (client.Leaf)(u)
	case client.Delete:
		l = (client.Leaf)(u)
	default:
		return fmt.Errorf("received unsupported client.Notification: %#v", n)
	}
	if len(l.Path) == 0 {
		return errors.New("client.Update contained no Path")
	}
	name := l.Path[0]
	target := c.getTarget(name)
	if target == nil {
		return fmt.Errorf("target %q not found in cache", name)
	}
	target.meta.AddInt(metadata.UpdateCount, 1)
	target.checkTimestamp(l)
	switch u := n.(type) {
	case client.Update:
		nd, err := target.update(u)
		if err != nil {
			return err
		}
		if nd != nil {
			c.client(nd)
		}
	case client.Delete:
		for _, nd := range target.remove(u) {
			c.client(nd)
		}
	}
	return nil
}

func newTargetCache(name string) *targetCache {
	return &targetCache{t: &ctree.Tree{}, name: name, meta: metadata.New()}
}

func (t *targetCache) checkTimestamp(l client.Leaf) {
	// Locking ensures that d.ts is always increasing regardless of the order in
	// which updates are processed in parallel by multiple goroutines.
	defer t.tsmu.Unlock()
	t.tsmu.Lock()
	// Track latest timestamp for a target.
	if l.TS.After(t.ts) {
		t.ts = l.TS
	}
}

func pathEquals(path1, path2 []string) bool {
	if len(path1) != len(path2) {
		return false
	}
	for i := range path1 {
		if path1[i] != path2[i] {
			return false
		}
	}
	return true
}

func (t *targetCache) updateStatus(u client.Update) error {
	switch u.Path[2] {
	case metadata.Sync:
		var ok bool
		t.sync, ok = u.Val.(bool)
		if !ok {
			return fmt.Errorf("%v : has value %v of type %T, expected boolean", metadata.Path(metadata.Sync), u.Val, u.Val)
		}
		t.meta.SetBool(metadata.Sync, t.sync)
	case metadata.Connected:
		connected, ok := u.Val.(bool)
		if !ok {
			return fmt.Errorf("%v : has value %v of type %T, expected boolean", metadata.Path(metadata.Connected), u.Val, u.Val)
		}
		t.meta.SetBool(metadata.Connected, connected)
	}
	return nil
}

func (t *targetCache) update(u client.Update) (*ctree.Leaf, error) {
	path := u.Path[1:]
	realData := true
	switch {
	case path[0] == metadata.Root:
		if err := t.updateStatus(u); err != nil {
			return nil, err
		}
		realData = false
	case t.sync:
		// Record latency for post-sync target updates.  Exclude metadata updates.
		t.lat.compute(u)
	}
	// Update an existing leaf.
	if oldval := t.t.GetLeaf(path); oldval != nil {
		// Since we control what goes into the tree, oldval should always
		// contain client.Update and there's no need to do a safe assertion.
		old := oldval.Value().(client.Update)
		if !old.TS.Before(u.TS) {
			// Update rejected. Timestamp <= previous recorded timestamp.
			return nil, errors.New("update is stale")
		}
		oldval.Update(u)
		return oldval, nil
	}
	// Add a new leaf.
	if err := t.t.Add(path, u); err != nil {
		return nil, err
	}
	if realData {
		t.meta.AddInt(metadata.LeafCount, 1)
		t.meta.AddInt(metadata.AddCount, 1)
	}
	return t.t.GetLeaf(path), nil
}

func olderThan(t time.Time) func(interface{}) bool {
	return func(x interface{}) bool {
		u, ok := x.(client.Update)
		return ok && u.TS.Before(t)
	}
}

func (t *targetCache) remove(u client.Delete) []*ctree.Leaf {
	leaves := t.t.DeleteConditional(u.Path[1:], olderThan(u.TS))
	if len(leaves) == 0 {
		return nil
	}
	deleted := int64(len(leaves))
	t.meta.AddInt(metadata.LeafCount, -deleted)
	t.meta.AddInt(metadata.DelCount, deleted)
	var ls []*ctree.Leaf
	for _, l := range leaves {
		ls = append(ls, ctree.DetachedLeaf(client.Delete{Path: append([]string{t.name}, l...), TS: u.TS}))
	}
	return ls
}

// updateCache calls fn for each targetCache.
func (c *Cache) updateCache(fn func(*targetCache, func(*ctree.Leaf))) {
	defer c.mu.RUnlock()
	c.mu.RLock()
	for _, target := range c.targets {
		fn(target, c.client)
	}
}

// updateSize walks the entire tree of the target, sums up marshaled sizes of
// all leaves and writes the sum in metadata.
func (t *targetCache) updateSize(func(*ctree.Leaf)) {
	var s int64
	size := func(n interface{}) int64 {
		buf, err := json.Marshal(n)
		if err != nil {
			return 0
		}
		return int64(len(buf))
	}
	t.t.Query([]string{"*"},
		func(_ []string, _ *ctree.Leaf, v interface{}) {
			s += size(v)
		})
	t.meta.SetInt(metadata.Size, s)
}

// updateMeta updates the metadata values in the cache.
func (t *targetCache) updateMeta(clients func(*ctree.Leaf)) {
	t.tsmu.Lock()
	latest := t.ts
	t.tsmu.Unlock()
	t.meta.SetInt(metadata.LatestTimestamp, latest.UnixNano())

	t.lat.updateReset(t.meta)
	ts := time.Now()
	for value := range metadata.TargetBoolValues {
		v, err := t.meta.GetBool(value)
		if err != nil {
			continue
		}
		path := metadata.Path(value)
		prev := t.t.GetLeafValue(path)
		if prev == nil || prev.(client.Update).Val.(bool) != v {
			if n, _ := t.update(client.Update{
				Path: append([]string{t.name}, path...),
				Val:  v,
				TS:   ts,
			}); n != nil {
				if clients != nil {
					clients(n)
				}
			}
		}
	}

	for value := range metadata.TargetIntValues {
		v, err := t.meta.GetInt(value)
		if err != nil {
			continue
		}
		path := metadata.Path(value)
		prev := t.t.GetLeafValue(path)
		if prev == nil || prev.(client.Update).Val.(int64) != v {
			if n, _ := t.update(client.Update{
				Path: append([]string{t.name}, path...),
				Val:  v,
				TS:   ts,
			}); n != nil {
				if clients != nil {
					clients(n)
				}
			}
		}
	}
}

// reset clears the targetCache of stale data upon a reconnection and notifies
// cache client of the removal.
func (t *targetCache) reset(clients func(*ctree.Leaf)) {
	// Reset metadata to zero values (e.g. connected = false) and notify clients.
	t.meta.Clear()
	t.updateMeta(clients)
	resetTime := time.Now()
	for root := range t.t.Children() {
		if root == metadata.Root {
			continue
		}
		t.t.Delete([]string{root})
		clients(ctree.DetachedLeaf(client.Delete{Path: []string{t.name, root}, TS: resetTime}))
	}
}

func (l *latency) compute(u client.Update) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lat := time.Now().Sub(u.TS)
	l.totalDiff += lat
	l.count++
	if lat > l.max {
		l.max = lat
	}
	if lat < l.min || l.min == 0 {
		l.min = lat
	}
}

func (l *latency) updateReset(m *metadata.Metadata) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count == 0 {
		return
	}
	m.SetInt(metadata.LatencyAvg, (l.totalDiff / time.Duration(l.count)).Nanoseconds())
	m.SetInt(metadata.LatencyMax, l.max.Nanoseconds())
	m.SetInt(metadata.LatencyMin, l.min.Nanoseconds())
	l.totalDiff = 0
	l.count = 0
	l.min = 0
	l.max = 0
}

// IsTargetDelete is a convenience function that identifies a leaf as
// containing a delete notification for an entire target.
func IsTargetDelete(l *ctree.Leaf) bool {
	d, ok := l.Value().(client.Delete)
	if !ok {
		return false
	}
	return len(d.Path) == 1
}
