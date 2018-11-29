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
	"errors"
	"fmt"
	"time"

	"github.com/openconfig/gnmi/client"
)

func ExampleClient() {
	block := make(Block)
	Mock("fake", []interface{}{
		client.Update{Path: []string{"target", "a", "b"}, Val: 1, TS: time.Now()},
		client.Update{Path: []string{"target", "a", "c"}, Val: 2, TS: time.Now()},
		block,
		client.Delete{Path: []string{"target", "a", "b"}, TS: time.Now()},
		errors.New("unexpected error"),
	})

	// Unblock the stream after a second.
	go func() {
		time.Sleep(time.Second)
		block.Unblock()
	}()

	err := client.New().Subscribe(context.Background(), client.Query{
		Addrs:   []string{""},
		Queries: []client.Path{{"*"}},
		Type:    client.Once,
		NotificationHandler: func(n client.Notification) error {
			switch nn := n.(type) {
			case client.Connected:
				fmt.Println("connected")
			case client.Sync:
				fmt.Println("sync")
			case client.Update:
				fmt.Printf("%q: %v\n", nn.Path, nn.Val)
			case client.Delete:
				fmt.Printf("%q deleted\n", nn.Path)
			}
			return nil
		},
	})
	fmt.Println("got error:", err)
	// Output:
	// connected
	// ["target" "a" "b"]: 1
	// ["target" "a" "c"]: 2
	// ["target" "a" "b"] deleted
	// got error: unexpected error
}
