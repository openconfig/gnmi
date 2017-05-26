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
	"errors"
	"testing"
	"time"

	"context"
	"github.com/openconfig/gnmi/client"
)

func TestReconnect(t *testing.T) {
	c := breakingSubscribeClient{
		BaseClient: &client.BaseClient{},
		ch:         make(chan error),
	}

	disconnect := make(chan struct{})
	reset := make(chan struct{})
	rc := client.Reconnect(c, func() { disconnect <- struct{}{} }, func() { reset <- struct{}{} })
	client.ReconnectBaseDelay = time.Nanosecond
	client.ReconnectMaxDelay = time.Millisecond

	go rc.Subscribe(context.Background(), client.Query{Type: client.Stream})
	for i := 0; i < 100; i++ {
		c.ch <- errors.New("break")
		<-disconnect
		<-reset
	}
}

type breakingSubscribeClient struct {
	*client.BaseClient
	ch chan error
}

func (c breakingSubscribeClient) Subscribe(ctx context.Context, q client.Query, clientType ...string) error {
	return <-c.ch
}
