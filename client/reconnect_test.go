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

package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openconfig/gnmi/client"
	fclient "github.com/openconfig/gnmi/client/fake"
)

func TestReconnect(t *testing.T) {
	c := breakingSubscribeClient{
		BaseClient: &client.BaseClient{},
		ch:         make(chan error),
	}

	disconnect := make(chan struct{}, 1)
	reset := make(chan struct{})
	rc := client.Reconnect(c, func() { disconnect <- struct{}{} }, func() { reset <- struct{}{} })
	client.ReconnectBaseDelay = time.Nanosecond
	client.ReconnectMaxDelay = time.Millisecond

	subscribeDone := make(chan struct{})
	go func() {
		defer close(subscribeDone)
		rc.Subscribe(context.Background(), client.Query{Type: client.Stream})
	}()
	for i := 0; i < 100; i++ {
		c.ch <- errors.New("break")
		<-disconnect
		<-reset
	}

	// Try to trigger a race between rc.Close and rc.Subscribe.
	// rc.Close should return only after rc.Subscribe is done.
	closeDone := make(chan struct{})
	go func() {
		defer close(closeDone)
		rc.Close()
	}()
	select {
	case <-subscribeDone:
	case <-closeDone:
		t.Error("rc.Close returned before rc.Subscribe")
	}
}

type breakingSubscribeClient struct {
	*client.BaseClient
	ch chan error
}

func (c breakingSubscribeClient) Subscribe(ctx context.Context, q client.Query, clientType ...string) error {
	return <-c.ch
}

func (c breakingSubscribeClient) Close() error {
	c.ch <- errors.New("closed")
	return nil
}

func TestReconnectEarlyClose(t *testing.T) {
	block := make(fclient.Block)
	defer close(block)
	fclient.Mock("fake", []interface{}{block})

	rc := client.Reconnect(&client.BaseClient{}, nil, nil)
	rc.Close()

	done := make(chan struct{})
	go func() {
		rc.Subscribe(context.Background(), client.Query{
			Type:                client.Stream,
			NotificationHandler: func(client.Notification) error { return nil },
		}, "fake")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Subscribe on a closed ReconnectClient didn't return after 1s")
	}
}
