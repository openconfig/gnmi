package client

import (
	"errors"
	"fmt"
	"time"

	"context"
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
