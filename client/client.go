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

// Package client provides generic access layer for streaming telemetry.
//
// The Client interface is implemented by 3 types in this package.
//
// BaseClient simply forwards all messages from the underlying connection to
// NotificationHandler or ProtoHandler (see type Query).
//
// CacheClient wraps around BaseClient and adds a persistence layer for all
// notifications. The notifications build up an internal tree which can be
// queried and walked using CacheClient's methods.
//
// ReconnectClient wraps around any Client implementation (BaseClient,
// CacheClient or a user-provided one) and adds transparent reconnection loop
// in Subscribe. Reconnection attempts are done with exponential backoff.
//
// Underlying transport is abstracted by interface Impl. Implementations of
// this interface live outside of this package and must be registered via
// Register.
//
// Examples:
//  // Once client will run the query and once complete you can act on the
//  // returned tree
//	func ExampleClient_Once() {
//		q := client.Query{
//			Addrs:       []string{"127.0.0.1:1234"},
//			Target:     "dev",
//			Queries:    []client.Path{{"*"}},
//			Type:       client.Once,
//		}
//		c := client.New()
//		defer c.Close()
//		err := c.Subscribe(context.Background(), q, pclient.Type)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//		for _, v := range c.Leaves() {
//			fmt.Printf("%v: %v\n", v.Path, v.Val)
//		}
//	}
//
//  // Poll client is like Once client, but can be re-triggered via Poll to
//  // re-execute the query.
//	func ExampleClient_Poll() {
//		q := client.Query{
//			Addrs:       []string{"127.0.0.1:1234"},
//			Target:     "dev",
//			Queries:    []client.Path{{"*"}},
//			Type:       client.Poll,
//		}
//		c := client.New()
//		defer c.Close()
//		err := c.Subscribe(context.Background(), q)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//		for _, v := range c.Leaves() {
//			fmt.Printf("%v: %v\n", v.Path, v.Val)
//		}
//		err = c.Poll() // Poll allows the underyling Query to keep running
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//		for _, v := range c.Leaves() {
//			fmt.Printf("%v: %v\n", v.Path, v.Val)
//		}
//	}
//
//  // Stream client returns the current state for the query and keeps running
//  // until closed or the underlying connection breaks.
//	func ExampleClient_Stream() {
//		q := client.Query{
//			Addrs:       []string{"127.0.0.1:1234"},
//			Target:     "dev",
//			Queries:    []client.Path{{"*"}},
//			Type:       client.Stream,
//			NotificationHandler: func(n Notification) error {
//				switch nn := n.(type) {
//				case Connected:
//					fmt.Println("client is connected")
//				case Sync:
//					fmt.Println("client is synced")
//				case Update, Delete:
//					fmt.Println("update: %+v", nn)
//				case Error:
//					fmt.Println("error: %v", nn)
//				}
//			},
//		}
//		c := client.New()
//		defer c.Close()
//		// Note that Subscribe will block.
//		err := c.Subscribe(context.Background(), q)
//		if err != nil {
//			fmt.Println(err)
//		}
//	}
package client

import (
	"errors"
	"fmt"
	"io"
	"sync"

	log "github.com/golang/glog"
	"context"
)

// Client defines a set of methods which every client must implement.
// This package provides a few implementations: BaseClient, CacheClient,
// ReconnectClient.
//
// Do not confuse this with Impl.
type Client interface {
	// Subscribe will perform the provided query against the requested
	// clientType. clientType is the name of a specific Impl specified in
	// Register (most implementations will call Register in init()).
	//
	// It will try each clientType listed in order until one succeeds. If
	// clientType is nil, it will try each registered clientType in random
	// order.
	Subscribe(context.Context, Query, ...string) error
	// Poll will send a poll request to the server and process all
	// notifications. It is up the caller to identify the sync and realize the
	// Poll is complete.
	Poll() error
	// Close terminates the underlying Impl, which usually terminates the
	// connection right away.
	// Close must be called to release any resources that Impl could have
	// allocated.
	Close() error
	// Impl will return the underlying client implementation.
	Impl() (Impl, error)
	// Set will make updates/deletes on the given values in SetRequest.
	//
	// Note that SetResponse and inner SetResult's contain Err fields that
	// should be checked manually. Error from Set is only related to
	// transport-layer issues in the RPC.
	Set(context.Context, SetRequest) (SetResponse, error)
}

var (
	// ErrStopReading is the common error defined to have the client stop a read
	// loop.
	ErrStopReading = errors.New("stop the result reading loop")
	// ErrClientInit is the common error for when making calls before the client
	// has been started via Query.
	ErrClientInit = errors.New("Query() must be called before any operations on client")
	// ErrUnsupported is returned by Impl's methods when the underlying
	// implementation doesn't support it.
	ErrUnsupported = errors.New("operation not supported by client implementation")
)

// BaseClient is a streaming telemetry client with minimal footprint. The
// caller must call Subscribe to perform the actual query. BaseClient stores no
// state. All updates must be handled by the provided handlers inside of
// Query.
type BaseClient struct {
	mu         sync.RWMutex
	closed     bool
	clientImpl Impl

	query Query
}

var _ Client = &BaseClient{}

// Subscribe implements the Client interface.
func (c *BaseClient) Subscribe(ctx context.Context, q Query, clientType ...string) error {
	impl, err := NewImpl(ctx, q, clientType...)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.query = q
	if c.clientImpl != nil {
		c.clientImpl.Close()
	}
	c.clientImpl = impl
	c.closed = false
	c.mu.Unlock()

	return c.run(impl)
}

// Poll implements the Client interface.
func (c *BaseClient) Poll() error {
	impl, err := c.Impl()
	if err != nil {
		return ErrClientInit
	}
	if c.query.Type != Poll {
		return fmt.Errorf("Poll() can only be used on Poll query type: %v", c.query.Type)
	}
	if err := impl.Poll(); err != nil {
		return err
	}
	return c.run(impl)
}

// Close implements the Client interface.
func (c *BaseClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clientImpl == nil {
		return ErrClientInit
	}
	c.closed = true
	return c.clientImpl.Close()
}

// Impl implements the Client interface.
func (c *BaseClient) Impl() (Impl, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clientImpl == nil {
		return nil, ErrClientInit
	}
	return c.clientImpl, nil
}

// Set implements the Client interface.
func (c *BaseClient) Set(ctx context.Context, r SetRequest) (SetResponse, error) {
	c.mu.Lock()
	impl := c.clientImpl
	c.mu.Unlock()
	if impl == nil {
		return SetResponse{}, ErrClientInit
	}
	return impl.Set(ctx, r)
}

func (c *BaseClient) run(impl Impl) error {
	for {
		err := impl.Recv()
		switch err {
		default:
			log.V(1).Infof("impl.Recv() received unknown error: %v", err)
			impl.Close()
			return err
		case io.EOF, ErrStopReading:
			log.V(1).Infof("impl.Recv() stop marker: %v", err)
			return nil
		case nil:
		}

		// Close fast, so that we don't deliver any buffered updates.
		//
		// Note: this approach still allows at most 1 update through after
		// Close. A more thorough solution would be to do the check at
		// Notification/ProtoHandler or Impl level, but that would involve much
		// more work.
		c.mu.RLock()
		closed := c.closed
		c.mu.RUnlock()
		if closed {
			return nil
		}
	}
}
