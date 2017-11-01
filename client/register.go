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
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	log "github.com/golang/glog"
	"context"
	"github.com/openconfig/gnmi/errlist"
)

var (
	mu         sync.Mutex
	clientImpl = map[string]InitImpl{}
)

// Default timeout for all queries.
const defaultTimeout = time.Minute

// Impl is the protocol/RPC specific implementation of the streaming
// Client.
type Impl interface {
	// Subscribe sends a Subscribe request to the server.
	Subscribe(context.Context, Query) error
	// Recv processes a single message from the server. This method is exposed to
	// allow the generic client control the state of message processing.
	Recv() error
	// Close will close the underlying rpc connections.
	Close() error
	// Poll will send an implementation specific Poll request to the server.
	Poll() error
	// Set will make updates/deletes on the given values in SetRequest.
	Set(context.Context, SetRequest) (SetResponse, error)
}

// InitImpl provides a prototype for all client specific implementations of New.
type InitImpl func(context.Context, Destination) (Impl, error)

// Register will register the client specific implementation.
func Register(t string, f InitImpl) error {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := clientImpl[t]; ok {
		return fmt.Errorf("Duplicate registration of type %q", t)
	}
	if f == nil {
		return errors.New("RegisterFunc cannot be nil")
	}
	clientImpl[t] = f
	log.V(1).Infof("client.Register(%q, func) successful.", t)
	return nil
}

// RegisterTest allows tests to override client implementation for any client
// type. It's identical to Register, except t uniqueness is not enforced.
//
// RegisterTest is similar to ResetRegisteredImpls + Register.
// Commonly used with the fake client (./fake directory).
func RegisterTest(t string, f InitImpl) error {
	mu.Lock()
	defer mu.Unlock()
	if f == nil {
		return errors.New("RegisterFunc cannot be nil")
	}
	clientImpl[t] = f
	log.V(1).Infof("client.Register(%q, func) successful.", t)
	return nil
}

// NewImpl returns a client implementation based on the registered types.
// It will try all clientTypes listed in parallel until one succeeds. If
// clientType is nil, it will try all registered clientTypes.
func NewImpl(ctx context.Context, d Destination, clientType ...string) (Impl, error) {
	mu.Lock()
	registeredCount := len(clientImpl)
	if clientType == nil {
		for t := range clientImpl {
			clientType = append(clientType, t)
		}
	}
	mu.Unlock()
	if registeredCount == 0 {
		return nil, errors.New("no registered client types")
	}

	// If Timeout is not set, use a default one. There is pretty much never a
	// case where clients will want to wait for initial connection
	// indefinitely. Reconnect client helps with retries.
	if d.Timeout == 0 {
		d.Timeout = defaultTimeout
	}

	errC := make(chan error, len(clientType))
	implC := make(chan Impl)
	done := make(chan struct{})
	defer close(done)
	log.V(1).Infof("Attempting client types: %v", clientType)
	for _, t := range clientType {
		// Launch each clientType in parallel where each sends either an error or
		// an implementation over a channel.
		go func(t string) {
			mu.Lock()
			f, ok := clientImpl[t]
			mu.Unlock()
			if !ok {
				errC <- fmt.Errorf("no registered client %q", t)
				return
			}
			impl, err := f(ctx, d)
			if err != nil {
				errC <- fmt.Errorf("client %q : %v", t, err)
				return
			}
			log.V(1).Infof("client %q create with type %T", t, impl)
			select {
			case implC <- impl:
			case <-done:
				impl.Close()
			}
		}(t)
	}
	errs := errlist.Error{List: errlist.List{Separator: "\n\t"}}
	// Look for the first non-error client implementation or return an error if
	// all client types fail.
	for {
		select {
		case err := <-errC:
			errs.Add(err)
			if len(errs.Errors()) == len(clientType) {
				return nil, errs.Err()
			}
		case impl := <-implC:
			return impl, nil
		}
	}
}

// ResetRegisteredImpls removes and Impls registered with Register. This should
// only be used in tests to clear out their mock Impls, so that they don't
// affect other tests.
func ResetRegisteredImpls() {
	mu.Lock()
	defer mu.Unlock()
	clientImpl = make(map[string]InitImpl)
}

// RegisteredImpls returns a slice of currently registered client types.
func RegisteredImpls() []string {
	mu.Lock()
	defer mu.Unlock()
	var impls []string
	for k := range clientImpl {
		impls = append(impls, k)
	}
	sort.Strings(impls)
	return impls
}
