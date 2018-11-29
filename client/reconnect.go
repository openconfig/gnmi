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

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	log "github.com/golang/glog"
)

var (
	// ReconnectBaseDelay is the minimum delay between re-Subscribe attempts in
	// Reconnect. You can change this before creating ReconnectClient instances.
	ReconnectBaseDelay = time.Second
	// ReconnectMaxDelay is the maximum delay between re-Subscribe attempts in
	// Reconnect. You can change this before creating ReconnectClient instances.
	ReconnectMaxDelay = time.Minute
)

// ReconnectClient is a wrapper around any Client that never returns from
// Subscribe (unless explicitly closed). Underlying calls to Subscribe are
// repeated indefinitely, with an exponential backoff between attempts.
//
// ReconnectClient should only be used with streaming or polling queries. Once
// queries will fail immediately in Subscribe.
type ReconnectClient struct {
	Client
	disconnect func()
	reset      func()

	mu            sync.Mutex
	subscribeDone chan struct{}
	cancel        func()
	closed        bool
}

var _ Client = &ReconnectClient{}

// Reconnect wraps c and returns a new ReconnectClient using it.
//
// disconnect callback is called each time the underlying Subscribe returns, it
// may be nil.
//
// reset callback is called each time the underlying Subscribe is retried, it
// may be nil.
//
// Closing the returned ReconnectClient will unblock Subscribe.
func Reconnect(c Client, disconnect, reset func()) *ReconnectClient {
	return &ReconnectClient{Client: c, disconnect: disconnect, reset: reset}
}

// Subscribe implements Client interface.
func (p *ReconnectClient) Subscribe(ctx context.Context, q Query, clientType ...string) error {
	switch q.Type {
	default:
		return fmt.Errorf("ReconnectClient used for %s query", q.Type)
	case Stream, Poll:
	}

	ctx, done := p.initDone(ctx)
	defer done()

	failCount := 0
	for {
		start := time.Now()
		err := p.Client.Subscribe(ctx, q, clientType...)
		if p.disconnect != nil {
			p.disconnect()
		}
		failCount++

		// Check if Subscribe returned because ctx was canceled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err == nil {
			failCount = 0
		}
		// Since Client won't tell us whether error was immediate or after
		// streaming for a while, try to "guess" if it's the latter.
		if time.Since(start) > ReconnectMaxDelay {
			failCount = 0
		}

		bo := backoff(ReconnectBaseDelay, ReconnectMaxDelay, failCount)
		log.Errorf("client.Subscribe (target %q) failed (%d times): %v; reconnecting in %s", q.Target, failCount, err, bo)
		time.Sleep(bo)

		// Signal caller right before we attempt to reconnect.
		if p.reset != nil {
			p.reset()
		}
	}
}

// initDone finishes Subscribe initialization before starting the inner
// Subscribe loop.
// If p is closed before initDone, a cancelled context is returned.
func (p *ReconnectClient) initDone(ctx context.Context) (context.Context, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribeDone = make(chan struct{})

	// ctx is cancelled in p.Close().
	ctx, p.cancel = context.WithCancel(ctx)

	// If Close was called before initDone returned, it didn't have a cancel
	// func to trigger. Trigger it here instead.
	// Since initDone and Cancel are synchronizing on p.mu, either this or
	// Close will call p.cancel(), preventing a hanging client.
	if p.closed {
		p.cancel()
	}

	return ctx, func() {
		close(p.subscribeDone)
	}
}

// Close implements Client interface.
func (p *ReconnectClient) Close() error {
	subscribeDone := func() chan struct{} {
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.cancel != nil {
			p.cancel()
		}
		p.closed = true

		return p.subscribeDone
	}()

	err := p.Client.Close()

	// Wait for Subscribe to return.
	if subscribeDone != nil {
		<-subscribeDone
	}
	return err
}

// Impl implements Client interface.
func (p *ReconnectClient) Impl() (Impl, error) {
	return p.Client.Impl()
}

// Poll implements Client interface.
// Poll may fail if Subscribe is reconnecting when it's called.
func (p *ReconnectClient) Poll() error {
	return p.Client.Poll()
}

const (
	backoffFactor = 1.3 // backoff increases by this factor on each retry
	backoffRange  = 0.4 // backoff is randomized downwards by this factor
)

// backoff a duration to wait for before retrying a query. The duration grows
// exponentially as retries increases.
func backoff(baseDelay, maxDelay time.Duration, retries int) time.Duration {
	backoff, max := float64(baseDelay), float64(maxDelay)
	for backoff < max && retries > 0 {
		backoff = backoff * backoffFactor
		retries--
	}
	if backoff > max {
		backoff = max
	}

	// Randomize backoff delays so that if a cluster of requests start at
	// the same time, they won't operate in lockstep.  We just subtract up
	// to 40% so that we obey maxDelay.
	backoff -= backoff * backoffRange * rand.Float64()
	if backoff < 0 {
		return 0
	}
	return time.Duration(backoff)
}
