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

	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/errlist"
	"github.com/openconfig/gnmi/latency"
	"github.com/openconfig/gnmi/metadata"
	"github.com/openconfig/gnmi/path"
	"github.com/openconfig/gnmi/value"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// T provides a shorthand function to reference a timestamp with an
// int64 (nanoseconds since epoch).
func T(n int64) time.Time { return time.Unix(0, n) }

// Now is a function that can be overridden in tests to alter the timestamps
// applied to deletes and metadata updates.
var Now = time.Now

// A Target hosts an indexed cache of state for a single target.
type Target struct {
	name   string             // name of the target
	t      *ctree.Tree        // actual cache of target data
	client func(*ctree.Leaf)  // Function to pass all cache updates to.
	sync   bool               // denotes whether this cache is in sync with target
	meta   *metadata.Metadata // metadata associated with target
	lat    *latency.Latency   // latency measurements
	tsmu   sync.Mutex         // protects latest timestamp
	ts     time.Time          // latest timestamp for an update
}

// Name returns the name of the target.
func (t *Target) Name() string {
	return t.name
}

// options contains options for creating a Cache.
type options struct {
	// latencyWindows is a list of time windows for which latency stats will be
	// calculated and exported as metadata.
	latencyWindows      []time.Duration
	avgLatencyPrecision time.Duration
}

// Option defines the function prototype to set options for creating a Cache.
type Option func(*options)

// WithLatencyWindows returns an Option to set latency windows for which
// latency stats are calculated and exported for each target in Cache.
// metaUpdatePeriod is the period for updating target metadata. The latency
// windows need to be multiples of this period.
func WithLatencyWindows(ws []string, metaUpdatePeriod time.Duration) (Option, error) {
	if metaUpdatePeriod.Nanoseconds() == 0 {
		return nil, nil // disable latency stats if updatePeriod is 0
	}
	windows, err := latency.ParseWindows(ws, metaUpdatePeriod)
	if err != nil {
		return nil, err
	}
	return func(o *options) {
		o.latencyWindows = windows
	}, nil
}

// WithAvgLatencyPrecision returns an Option to set the precision of average
// latency stats calculated in Cache.
func WithAvgLatencyPrecision(avgLatencyPrecision time.Duration) Option {
	return func(o *options) {
		o.avgLatencyPrecision = avgLatencyPrecision
	}
}

// Cache is a structure holding state information for multiple targets.
type Cache struct {
	opts    options
	mu      sync.RWMutex
	targets map[string]*Target // Map of per target caches.
	client  func(*ctree.Leaf)  // Function to pass all cache updates to.
}

// New creates a new instance of Cache that receives target updates from the
// translator and provides an interface to service client queries.
func New(targets []string, opts ...Option) *Cache {
	c := &Cache{
		targets: make(map[string]*Target, len(targets)),
		client:  func(*ctree.Leaf) {},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&c.opts)
		}
	}
	latency.RegisterMetadata(c.opts.latencyWindows)

	for _, t := range targets {
		c.Add(t)
	}
	return c
}

// LatencyWindows returns the latency windows supported by the cache.
func (c *Cache) LatencyWindows() []time.Duration {
	return c.opts.latencyWindows
}

// SetClient registers a callback function to receive calls for each update
// accepted by the cache. This call should be made prior to sending any updates
// into the cache, just after initialization.
func (c *Cache) SetClient(client func(*ctree.Leaf)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = client
	for _, t := range c.targets {
		t.client = client
	}
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
	c.updateCache((*Target).updateMeta)
}

// ConnectError updates the target's metadata with the provided error.
func (c *Cache) ConnectError(name string, err error) {
	if target := c.GetTarget(name); target != nil {
		target.connectError(err)
	}
}

// connectError updates the ConnectError in the cached metadata.
func (t *Target) connectError(err error) {
	if err := t.GnmiUpdate(metaNotiStr(t.name, metadata.ConnectError, err.Error())); err != nil {
		log.Errorf("target %q got error during meta connect error update, %v", t.name, err)
	}
}

// UpdateSize computes the size of each target cache and updates the size
// metadata reported within the each target cache.
func (c *Cache) UpdateSize() {
	c.updateCache((*Target).updateSize)
}

// GetTarget returns the Target from the cache corresponding to the target name.
func (c *Cache) GetTarget(target string) *Target {
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
		defer c.mu.RUnlock()
		c.mu.RLock()
		// Run the query sequentially for each target cache.
		for _, target := range c.targets {
			if err := target.t.Query(query, fn); err != nil {
				return err
			}
		}
	default:
		dc := c.GetTarget(target)
		if dc == nil {
			return fmt.Errorf("target %q not found in cache", target)
		}
		return dc.t.Query(query, fn)
	}
	return nil
}

// Add reserves space in c to receive updates for the specified target.
func (c *Cache) Add(target string) *Target {
	defer c.mu.Unlock()
	c.mu.Lock()
	var latOpts *latency.Options
	if c.opts.avgLatencyPrecision.Nanoseconds() != 0 {
		latOpts = &latency.Options{AvgPrecision: c.opts.avgLatencyPrecision}
	}
	t := &Target{
		t:      &ctree.Tree{},
		name:   target,
		meta:   metadata.New(),
		client: c.client,
		lat:    latency.New(c.opts.latencyWindows, latOpts),
	}
	c.targets[target] = t
	return t
}

// Reset clears the cache for a target once a connection is resumed after
// having been lost.
func (c *Cache) Reset(target string) {
	defer c.mu.RUnlock()
	c.mu.RLock()
	if t := c.targets[target]; t != nil {
		t.Reset()
	}
}

// Remove removes the space in c corresponding to the specified target.
func (c *Cache) Remove(target string) {
	defer c.mu.Unlock()
	c.mu.Lock()
	delete(c.targets, target)
	// Notify clients that the target is removed.
	c.client(ctree.DetachedLeaf(deleteNoti(target, "", []string{"*"})))
}

// Sync creates an internal gnmi.Notification with metadata/sync path
// to set the state to true for the specified target.
func (c *Cache) Sync(name string) {
	if target := c.GetTarget(name); target != nil {
		target.Sync()
	}
}

// Sync creates an internal gnmi.Notification with metadata/sync path
// to set the state to true for the specified target.
func (t *Target) Sync() {
	if err := t.GnmiUpdate(metaNotiBool(t.name, metadata.Sync, true)); err != nil {
		log.Errorf("target %q got error during meta sync update, %v", t.name, err)
	}
}

// Connect creates an internal gnmi.Notification for metadata/connected path
// to set the state to true for the specified target.
func (c *Cache) Connect(name string) {
	if target := c.GetTarget(name); target != nil {
		target.Connect()
	}
}

// Connect creates an internal gnmi.Notification for metadata/connected path
// to set the state to true for the specified target, and clear connectErr.
func (t *Target) Connect() {
	if err := t.GnmiUpdate(metaNotiBool(t.name, metadata.Connected, true)); err != nil {
		log.Errorf("target %q got error during meta connected update, %v", t.name, err)
	}

	if err := t.GnmiUpdate(deleteNoti(t.name, "", metadata.Path(metadata.ConnectError))); err != nil {
		log.Errorf("target %q got error during meta connectError update, %v", t.name, err)
	}
}

// GnmiUpdate sends a pb.Notification into the cache.
// If the notification has multiple Updates/Deletes,
// each individual Update/Delete is sent to cache as
// a separate gnmi.Notification.
func (c *Cache) GnmiUpdate(n *pb.Notification) error {
	if n == nil {
		return errors.New("gnmi.Notification is nil")
	}
	if n.GetPrefix() == nil {
		return errors.New("gnmi.Notification prefix is nil")
	}
	target := c.GetTarget(n.GetPrefix().GetTarget())
	if target == nil {
		return fmt.Errorf("target %q not found in cache", n.GetPrefix().GetTarget())
	}
	return target.GnmiUpdate(n)
}

// GnmiUpdate sends a pb.Notification into the target cache.
// If the notification has multiple Updates/Deletes,
// each individual Update/Delete is sent to cache as
// a separate gnmi.Notification.
func (t *Target) GnmiUpdate(n *pb.Notification) error {
	t.checkTimestamp(T(n.GetTimestamp()))
	switch {
	// Store atomic notifications as a single leaf in the tree.
	case n.Atomic:
		if len(n.GetDelete()) > 0 {
			return errors.New("atomic deletes unsupported")
		}
		l := len(n.GetUpdate())
		if l == 0 {
			t.meta.AddInt(metadata.EmptyCount, 1)
			return nil
		}
		nd, err := t.gnmiUpdate(n)
		if err != nil {
			return err
		}
		if nd != nil {
			t.meta.AddInt(metadata.UpdateCount, int64(l))
			t.client(nd)
		}

		// Break non-atomic complex notifications into individual leaves per update.
	case len(n.GetUpdate())+len(n.GetDelete()) > 1:
		updates := n.GetUpdate()
		deletes := n.GetDelete()
		n.Update, n.Delete = nil, nil
		// restore back the notification updates and deletes
		defer func() {
			n.Update = updates
			n.Delete = deletes
		}()
		errs := &errlist.List{}
		for _, u := range updates {
			noti := proto.Clone(n).(*pb.Notification)
			noti.Update = []*pb.Update{u}
			nd, err := t.gnmiUpdate(noti)
			if err != nil {
				errs.Add(err)
				continue
			}
			if nd != nil {
				t.meta.AddInt(metadata.UpdateCount, 1)
				t.client(nd)
			}
		}

		for _, d := range deletes {
			noti := proto.Clone(n).(*pb.Notification)
			noti.Delete = []*pb.Path{d}
			t.meta.AddInt(metadata.UpdateCount, 1)
			for _, nd := range t.gnmiRemove(noti) {
				t.client(nd)
			}
		}
		return errs.Err()

	// Single update notification could be handled by the above code but is
	// handled separately to avoid the unnecessary proto.Clone call.
	case len(n.GetUpdate()) == 1:
		nd, err := t.gnmiUpdate(n)
		if err != nil {
			return err
		}
		if nd != nil {
			t.meta.AddInt(metadata.UpdateCount, 1)
			t.client(nd)
		}

	// Single delete notification also avoids proto.Clone above.
	case len(n.GetDelete()) == 1:
		t.meta.AddInt(metadata.UpdateCount, 1)
		for _, nd := range t.gnmiRemove(n) {
			t.client(nd)
		}

	// Empty notification.
	default:
		t.meta.AddInt(metadata.EmptyCount, 1)
	}
	return nil
}

func (t *Target) checkTimestamp(ts time.Time) {
	// Locking ensures that d.ts is always increasing regardless of the order in
	// which updates are processed in parallel by multiple goroutines.
	defer t.tsmu.Unlock()
	t.tsmu.Lock()
	// Track latest timestamp for a target.
	if ts.After(t.ts) {
		t.ts = ts
	}
}

func (t *Target) gnmiUpdate(n *pb.Notification) (*ctree.Leaf, error) {
	realData := true
	suffix := n.Update[0].Path
	// If the notification is an atomic group of updates, store them under the prefix only.
	if n.Atomic {
		suffix = nil
	}
	path := joinPrefixAndPath(n.Prefix, suffix)
	if path[0] == metadata.Root {
		realData = false
		u := n.Update[0]
		switch path[1] {
		case metadata.Sync:
			var ok bool
			tv, ok := u.Val.Value.(*pb.TypedValue_BoolVal)
			if !ok {
				return nil, fmt.Errorf("%v : has value %v of type %T, expected boolean", metadata.Path(metadata.Sync), u.Val, u.Val)
			}
			t.sync = tv.BoolVal
			t.meta.SetBool(metadata.Sync, t.sync)
		case metadata.Connected:
			tv, ok := u.Val.Value.(*pb.TypedValue_BoolVal)
			if !ok {
				return nil, fmt.Errorf("%v : has value %v of type %T, expected boolean", metadata.Path(metadata.Connected), u.Val, u.Val)
			}
			t.meta.SetBool(metadata.Connected, tv.BoolVal)
		case metadata.ConnectedAddr, metadata.ConnectError:
			tv, ok := u.Val.Value.(*pb.TypedValue_StringVal)
			if !ok {
				return nil, fmt.Errorf("%v : has value %v of type %T, expected string", metadata.Path(path[1]), u.Val, u.Val)
			}
			t.meta.SetStr(path[1], tv.StringVal)
		}
	}
	// Update an existing leaf.
	if oldval := t.t.GetLeaf(path); oldval != nil {
		// An update with corrupt data is possible to visit a node that does not
		// contain *pb.Notification. Thus, need type assertion here.
		old, ok := oldval.Value().(*pb.Notification)
		if !ok {
			return nil, fmt.Errorf("corrupt schema with collision for path %q, got %T", path, oldval.Value())
		}
		switch {
		case n.GetTimestamp() < old.GetTimestamp():
			// Update rejected. Timestamp < previous recorded timestamp.
			t.meta.AddInt(metadata.StaleCount, 1)
			return nil, errors.New("update is stale")
		case n.GetTimestamp() == old.GetTimestamp():
			if !proto.Equal(old, n) {
				if log.V(1) {
					log.Warningf("received different value at same timestamp\nfirst: %s\nsecond: %s", prototext.Format(old), prototext.Format(n))
				}
				// Allow to continue to update the cache taking the last supplied value for this timestamp.
			} else {
				t.meta.AddInt(metadata.StaleCount, 1)
				return nil, errors.New("update is stale")
			}
		}
		oldval.Update(n)
		// Simulate event-driven for all non-atomic updates.
		if !n.Atomic && value.Equal(old.Update[0].Val, n.Update[0].Val) {
			t.meta.AddInt(metadata.SuppressedCount, 1)
			return nil, nil
		}
		// Compute latency for updated leaves.
		if t.sync && realData {
			// Record latency for post-sync target updates.  Exclude metadata updates.
			t.lat.Compute(T(n.GetTimestamp()))
		}
		return oldval, nil
	}
	// Add a new leaf.
	if err := t.t.Add(path, n); err != nil {
		return nil, err
	}
	if realData {
		t.meta.AddInt(metadata.LeafCount, 1)
		t.meta.AddInt(metadata.AddCount, 1)
		// Compute latency for new leaves.
		if t.sync {
			// Record latency for post-sync target updates.  Exclude metadata updates.
			t.lat.Compute(T(n.GetTimestamp()))
		}
	}
	return t.t.GetLeaf(path), nil
}

func toDeleteNotification(n *pb.Notification, timestamp int64) *pb.Notification {
	d := &pb.Notification{
		Timestamp: timestamp,
		Prefix: &pb.Path{
			Target: n.GetPrefix().GetTarget(),
			Origin: n.GetPrefix().GetOrigin(),
		},
	}
	prefix := n.GetPrefix()
	path := n.Update[0].GetPath()
	// Set origin if the origin is in the update
	if origin := path.GetOrigin(); n.GetPrefix().GetOrigin() == "" && origin != "" {
		d.Prefix.Origin = origin
	}
	switch {
	case n.GetAtomic():
		d.Delete = []*pb.Path{{Elem: prefix.GetElem(), Element: prefix.GetElement()}}
	case len(prefix.GetElem()) > 0 || len(path.GetElem()) > 0:
		d.Delete = []*pb.Path{{Elem: append(prefix.GetElem(), path.GetElem()...)}}
	default:
		d.Delete = []*pb.Path{{Element: append(prefix.GetElement(), path.GetElement()...)}}
	}
	return d
}

func (t *Target) gnmiRemove(n *pb.Notification) []*ctree.Leaf {
	path := joinPrefixAndPath(n.Prefix, n.Delete[0])
	if path[0] == metadata.Root {
		t.meta.ResetEntry(path[1])
	}
	var leaves []*ctree.Leaf
	f := func(v interface{}) {
		d := v.(*pb.Notification)
		leaves = append(leaves, ctree.DetachedLeaf(toDeleteNotification(d, n.GetTimestamp())))
	}
	t.t.WalkDeleted(path, func(v interface{}) bool { return v.(*pb.Notification).GetTimestamp() < n.GetTimestamp() }, f)
	if len(leaves) == 0 {
		return nil
	}
	deleted := int64(len(leaves))
	t.meta.AddInt(metadata.LeafCount, -deleted)
	t.meta.AddInt(metadata.DelCount, deleted)
	return leaves
}

// updateCache calls fn for each Target.
func (c *Cache) updateCache(fn func(*Target, func(*ctree.Leaf))) {
	defer c.mu.RUnlock()
	c.mu.RLock()
	for _, target := range c.targets {
		fn(target, c.client)
	}
}

// updateSize walks the entire tree of the target, sums up marshaled sizes of
// all leaves and writes the sum in metadata.
func (t *Target) updateSize(func(*ctree.Leaf)) {
	var s int64
	size := func(n interface{}) int64 {
		buf, err := json.Marshal(n)
		if err != nil {
			return 0
		}
		return int64(len(buf))
	}
	t.t.Query([]string{"*"},
		func(_ []string, _ *ctree.Leaf, v interface{}) error {
			s += size(v)
			return nil
		})
	t.meta.SetInt(metadata.Size, s)
}

// updateMeta updates the metadata values in the cache.
func (t *Target) updateMeta(clients func(*ctree.Leaf)) {
	t.tsmu.Lock()
	latest := t.ts
	t.tsmu.Unlock()
	t.meta.SetInt(metadata.LatestTimestamp, latest.UnixNano())

	t.lat.UpdateReset(t.meta)
	for value := range metadata.TargetBoolValues {
		v, err := t.meta.GetBool(value)
		if err != nil {
			continue
		}
		path := metadata.Path(value)
		prev := t.t.GetLeafValue(path)
		if prev == nil || prev.(*pb.Notification).Update[0].Val.Value.(*pb.TypedValue_BoolVal).BoolVal != v {
			noti := metaNotiBool(t.name, value, v)
			if n, _ := t.gnmiUpdate(noti); n != nil {
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
		if prev == nil || prev.(*pb.Notification).Update[0].Val.Value.(*pb.TypedValue_IntVal).IntVal != v {
			noti := metaNotiInt(t.name, value, v)
			if n, _ := t.gnmiUpdate(noti); n != nil {
				if clients != nil {
					clients(n)
				}
			}
		}
	}

	for value := range metadata.TargetStrValues {
		v, err := t.meta.GetStr(value)
		if err != nil {
			continue
		}
		path := metadata.Path(value)
		prev := t.t.GetLeafValue(path)
		if prev == nil || prev.(*pb.Notification).Update[0].Val.Value.(*pb.TypedValue_StringVal).StringVal != v {
			noti := metaNotiStr(t.name, value, v)
			if n, _ := t.gnmiUpdate(noti); n != nil {
				if clients != nil {
					clients(n)
				}
			}
		}
	}
}

// Reset clears the Target of stale data upon a reconnection and notifies
// cache client of the removal.
func (t *Target) Reset() {
	// Reset metadata to zero values (e.g. connected = false) and notify clients.
	t.meta.Clear()
	t.updateMeta(t.client)
	for root := range t.t.Children() {
		if root == metadata.Root {
			continue
		}
		t.t.Delete([]string{root})
		t.client(ctree.DetachedLeaf(deleteNoti(t.name, root, []string{"*"})))
	}
}

func joinPrefixAndPath(pr, ph *pb.Path) []string {
	// <target> and <origin> are only valid as prefix gnmi.Path
	// https://github.com/openconfig/reference/blob/master/rpc/gnmi-specification.md#222-paths
	p := path.ToStrings(pr, true)
	p = append(p, path.ToStrings(ph, false)...)
	// remove the prepended target name
	p = p[1:]
	return p
}

func deleteNoti(t, o string, p []string) *pb.Notification {
	pe := make([]*pb.PathElem, 0, len(p))
	for _, e := range p {
		pe = append(pe, &pb.PathElem{Name: e})
	}
	return &pb.Notification{
		Timestamp: Now().UnixNano(),
		Prefix:    &pb.Path{Target: t, Origin: o},
		Delete:    []*pb.Path{&pb.Path{Elem: pe}},
	}
}

func metaNoti(t, m string, v *pb.TypedValue) *pb.Notification {
	mp := metadata.Path(m)
	pe := make([]*pb.PathElem, 0, len(mp))
	for _, p := range mp {
		pe = append(pe, &pb.PathElem{Name: p})
	}
	return &pb.Notification{
		Timestamp: Now().UnixNano(),
		Prefix:    &pb.Path{Target: t},
		Update: []*pb.Update{
			&pb.Update{
				Path: &pb.Path{Elem: pe},
				Val:  v,
			},
		},
	}
}

func metaNotiBool(t, m string, v bool) *pb.Notification {
	return metaNoti(t, m, &pb.TypedValue{Value: &pb.TypedValue_BoolVal{v}})
}

func metaNotiInt(t, m string, v int64) *pb.Notification {
	return metaNoti(t, m, &pb.TypedValue{Value: &pb.TypedValue_IntVal{v}})
}

func metaNotiStr(t, m string, v string) *pb.Notification {
	return metaNoti(t, m, &pb.TypedValue{Value: &pb.TypedValue_StringVal{v}})
}
