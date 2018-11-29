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

// Package coalesce provides an in-order queue for collecting outputs from a
// producer to provide inputs to a consumer.  Coalescing happens when an output
// has been produced more than once by the producer before being consumed.  In
// this case the outputs are coalesced and delivered only once.
package coalesce

import (
	"context"
	"errors"
	"sync"
)

// Queue is a structure that implements in-order delivery of coalesced inputs.
// Operations on Queue are threadsafe and any number of producers may contribute
// to the queue and any number of consumers may receive, however, no fairness
// among consumers is guaranteed and it is possible for starvation to occur.
type Queue struct {
	sync.Mutex
	// inserted is used to signal blocked consumers that a value has been added.
	inserted chan struct{}
	// closed is a signal that no Inserts can occur and Next should return error
	// for an empty queue.
	closed chan struct{}
	// coalesced tracks the number of values per interface that have been
	// coalesced.
	coalesced map[interface{}]uint32
	// queue is the in-order values transferred from producer to consumer.
	queue []interface{}
}

// NewQueue returns an initialized Queue ready for use.
func NewQueue() *Queue {
	return &Queue{
		inserted:  make(chan struct{}, 1),
		closed:    make(chan struct{}),
		coalesced: make(map[interface{}]uint32),
	}
}

var errClosedQueue = errors.New("closed queue")

// IsClosedQueue returns whether a given error is coalesce.errClosedQueue.
func IsClosedQueue(err error) bool {
	return err == errClosedQueue
}

// Insert adds i to the queue and returns true if i does not already exist.
// Insert returns an error if the Queue has been closed.
func (q *Queue) Insert(i interface{}) (bool, error) {
	select {
	case <-q.closed:
		return false, errClosedQueue
	default:
	}

	ok := q.insert(i)

	if ok {
		select {
		case q.inserted <- struct{}{}:
		default:
		}
	}

	return ok, nil
}

func (q *Queue) insert(i interface{}) bool {
	defer q.Unlock()
	q.Lock()

	if _, ok := q.coalesced[i]; ok {
		q.coalesced[i]++
		return false
	}
	q.queue = append(q.queue, i)
	q.coalesced[i] = 0
	return true
}

// Next returns the next item in the queue and the number of duplicates that
// were dropped, or an error if the Queue is empty and has been closed. Calls to
// Next are blocking.
func (q *Queue) Next(ctx context.Context) (interface{}, uint32, error) {
	for {
		i, coalesced, valid := q.next()
		if valid {
			return i, coalesced, nil
		}
		// Wait for an insert or a close.
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-q.inserted:
		case <-q.closed:
			// Make sure all values are delivered.
			if q.Len() == 0 {
				return nil, 0, errClosedQueue
			}
		}
	}
}

// next returns the next item and the number of coalesced duplicates if there is
// an item to consume as indicated by the bool.  If the bool is false, the
// caller has to option to block and try again or return an error.
func (q *Queue) next() (interface{}, uint32, bool) {
	defer q.Unlock()
	q.Lock()
	if len(q.queue) == 0 {
		return nil, 0, false
	}
	var i interface{}
	i, q.queue = q.queue[0], q.queue[1:]
	coalesced := q.coalesced[i]
	delete(q.coalesced, i)
	return i, coalesced, true
}

// Len returns the current length of the queue, useful for reporting statistics.
// Since the queue is designed to be operated on by one or more producers and
// one or more consumers, this return value only indicates what the length was
// at the instant it was evaluated which may change before the caller gets the
// return value.
func (q *Queue) Len() int {
	defer q.Unlock()
	q.Lock()
	return len(q.queue)
}

// Close closes the queue for inserts.
func (q *Queue) Close() {
	defer q.Unlock()
	q.Lock()
	// Avoid panic on closing the closed channel.
	select {
	case <-q.closed:
		return
	default:
		close(q.closed)
	}
}

// IsClosed reports if the queue is in a closed state.
func (q *Queue) IsClosed() bool {
	select {
	case <-q.closed:
		return true
	default:
	}
	return false
}
