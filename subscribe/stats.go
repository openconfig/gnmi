package subscribe

import (
	"sync"
)

// TypeStats is the container of client side statistics for a particular
// subscribe mode, e.g. stream, once, or poll.
type TypeStats struct {
	ActiveSubscriptionCount  int64 // currently active subscription count
	PendingSubscriptionCount int64 // currently pending subscription count
	SubscriptionCount        int64 // total subscription count, cumulative
}

// TargetStats is the container of client side statistics for a target.
type TargetStats struct {
	ActiveSubscriptionCount int64 // currently active subscription count
	SubscriptionCount       int64 // total subscription count, cumulative
}

// ClientStats is the container of statistics for a particular client.
type ClientStats struct {
	Target        string // for which target
	CoalesceCount int64  // total coalsced updates for the client, cumulative
	QueueSize     int64  // current queue size for the client
}

type stats struct {
	mu      sync.Mutex
	types   map[string]*TypeStats
	targets map[string]*TargetStats
	clients map[string]*ClientStats
}

func newStats() *stats {
	return &stats{
		types:   map[string]*TypeStats{},
		targets: map[string]*TargetStats{},
		clients: map[string]*ClientStats{},
	}
}

func (s *stats) allTypeStats() map[string]TypeStats {
	m := map[string]TypeStats{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, st := range s.types {
		m[t] = *st
	}
	return m
}

func (s *stats) allTargetStats() map[string]TargetStats {
	m := map[string]TargetStats{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, st := range s.targets {
		m[t] = *st
	}
	return m
}

func (s *stats) allClientStats() map[string]ClientStats {
	m := map[string]ClientStats{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, st := range s.clients {
		m[t] = *st
	}
	return m
}

func (s *stats) typeStats(typ string) *TypeStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st := s.types[typ]; st != nil {
		return st
	}
	nst := &TypeStats{}
	s.types[typ] = nst
	return nst
}

func (s *stats) targetStats(target string) *TargetStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st := s.targets[target]; st != nil {
		return st
	}
	nst := &TargetStats{}
	s.targets[target] = nst
	return nst
}

func (s *stats) clientStats(client, target string) *ClientStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st := s.clients[client]; st != nil {
		return st
	}
	nst := &ClientStats{Target: target}
	s.clients[client] = nst
	return nst
}

func (s *stats) removeClientStats(client string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
}
