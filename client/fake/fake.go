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

// Package client implements a fake client implementation to be used with
// streaming telemetry collection.  It provides a simple Updates queue of data
// to send it should be used to provide an RPC free test infra for user facing
// libraries.
package client

import (
	log "github.com/golang/glog"
	"context"
	"github.com/openconfig/gnmi/client"
)

// New can be replaced for any negative testing you would like to do as well.
var New = func(ctx context.Context, q client.Query) (client.Impl, error) {
	return &Client{
		Handler: q.NotificationHandler,
	}, nil
}

// Client is the fake of a client implementation. It will provide a simple
// list of updates to send to the generic client. The Updates slice can be
// made up of either client.Notifications or errors anything else will cause
// a panic.
type Client struct {
	currUpdate int
	Updates    []interface{}
	Handler    client.NotificationHandler
	// When BlockAfterSync is set, Client will read from it in Recv after
	// sending all Updates before returning ErrStopReading.
	// BlockAfterSync is closed when Close is called.
	BlockAfterSync chan struct{}
}

// Reset will reset the client to start playing new updates.
func (c *Client) Reset(u []interface{}) {
	c.currUpdate = 0
	c.Updates = u
}

// Recv will be called for each update the generic client wants to receive.
func (c *Client) Recv() error {
	if c.currUpdate >= len(c.Updates) {
		if c.BlockAfterSync != nil {
			log.Info("No more updates, blocking on BlockAfterSync")
			<-c.BlockAfterSync
		}
		log.Infof("Recv() returning %v", client.ErrStopReading)
		return client.ErrStopReading
	}
	log.Infof("Recv() called sending update: %v", c.Updates[c.currUpdate])
	switch v := c.Updates[c.currUpdate].(type) {
	case client.Notification:
		c.Handler(v)
	case error:
		return v
	}
	c.currUpdate++
	return nil
}

// Close is a noop in the fake.
func (c *Client) Close() error {
	if c.BlockAfterSync != nil {
		close(c.BlockAfterSync)
	}
	return nil
}

// Poll is a noop in the fake.
func (c *Client) Poll() error {
	return nil
}

// Set is not supported in fake.
func (c *Client) Set(context.Context, client.SetRequest) (client.SetResponse, error) {
	return client.SetResponse{}, client.ErrUnsupported
}
