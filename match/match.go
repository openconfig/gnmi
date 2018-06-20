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

// Package match builds a tree of active subscriptions that is matched against
// all incoming updates.
package match

import (
	"sync"
)

// Glob is a special string used to indicate a match of any path node.
const Glob = "*"

// Client is a interface for a callback invoked for all matching updates.
type Client interface {
	Update(interface{})
}

type branch struct {
	clients  map[Client]struct{}
	children map[string]*branch
}

// Match is a structure that will invoke registered query callbacks for each
// matching update.
type Match struct {
	mu   sync.RWMutex
	tree *branch
}

// New creates a new Match structure.
func New() *Match {
	return &Match{tree: &branch{}}
}

// AddQuery registers a client callback for all matching nodes. It returns a
// callback to remove the query.  The remove function is idempotent.
func (m *Match) AddQuery(query []string, client Client) (remove func()) {
	defer m.mu.Unlock()
	m.mu.Lock()
	m.tree.addQuery(query, client)
	return func() {
		defer m.mu.Unlock()
		m.mu.Lock()
		m.tree.removeQuery(query, client)
	}
}

func (b *branch) addQuery(query []string, client Client) {
	if len(query) == 0 {
		if b.clients == nil {
			b.clients = map[Client]struct{}{}
		}
		b.clients[client] = struct{}{}
		return
	}
	if b.children == nil {
		b.children = map[string]*branch{}
	}
	sb, ok := b.children[query[0]]
	if !ok {
		sb = &branch{}
		b.children[query[0]] = sb
	}
	sb.addQuery(query[1:], client)
}

func (b *branch) removeQuery(query []string, client Client) (empty bool) {
	defer func() {
		empty = (len(b.clients) == 0 && len(b.children) == 0)
	}()
	if len(query) == 0 {
		if b.clients != nil {
			delete(b.clients, client)
		}
		return
	}
	sb, ok := b.children[query[0]]
	if !ok {
		return
	}
	if sb.removeQuery(query[1:], client) {
		delete(b.children, query[0])
	}
	return
}

// Update invokes all client callbacks for queries that match the supplied node.
func (m *Match) Update(n interface{}, p []string) {
	defer m.mu.RUnlock()
	m.mu.RLock()
	m.tree.update(n, p)
}

func (b *branch) update(n interface{}, path []string) {
	// Update all clients at this level.
	for client := range b.clients {
		client.Update(n)
	}
	// Terminate recursion.
	if len(b.children) == 0 {
		return
	}
	// Implicit recursion for intermediate deletes.
	if len(path) == 0 {
		for _, c := range b.children {
			c.update(n, nil)
		}
		return
	}
	// This captures only target delete gnmi.Notification as it is going
	// to have Glob in the path elem in addition to device name.
	// For target delete client.Notification, device name is sufficient,
	// so it will not satisfy this case.
	if path[0] == Glob {
		for _, c := range b.children {
			c.update(n, path[1:])
		}
	} else {
		// Update all glob clients.
		if sb, ok := b.children[Glob]; ok {
			sb.update(n, path[1:])
		}
		// Update all explicit clients.
		if sb, ok := b.children[path[0]]; ok {
			sb.update(n, path[1:])
		}
	}
}
