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

package openconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/testing/fake/queue"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
	ocpb "github.com/openconfig/reference/rpc/openconfig"
)

// Client contains information about a client that has connected to the fake.
type Client struct {
	sendMsg   int64
	recvMsg   int64
	errors    int64
	cTime     time.Time
	cCount    int64
	polled    chan struct{}
	mu        sync.RWMutex
	config    *fpb.Config
	canceled  bool
	q         *queue.UpdateQueue
	subscribe *ocpb.SubscriptionList
}

// NewClient returns a new initialized client.
func NewClient(config *fpb.Config) *Client {
	c := &Client{
		config: config,
		polled: make(chan struct{}),
	}
	return c
}

func (c *Client) String() string {
	return c.Config().Target
}

// Run starts the client. The first message received must be a
// SubscriptionList. Once the client is started, it will run until the stream
// is closed or the schedule completes. For Poll queries the Run will block
// internally after sync until a Poll request is made to the server. This is
// important as the test may look like a deadlock since it can cause a timeout.
// Also if you Reset the client the change will not take effect until after the
// previous queue has been drained of notifications.
func (c *Client) Run(stream ocpb.OpenConfig_SubscribeServer) (err error) {
	if c.config == nil {
		return grpc.Errorf(codes.FailedPrecondition, "cannot start client config is nil")
	}
	if stream == nil {
		return grpc.Errorf(codes.FailedPrecondition, "cannot start client stream is nil.")
	}

	defer func() {
		if err != nil {
			c.errors++
		}
	}()

	query, err := stream.Recv()
	c.cTime = time.Now()
	c.cCount++
	c.recvMsg++
	if err != nil {
		if err == io.EOF {
			return grpc.Errorf(codes.Aborted, "stream EOF received before init")
		}
		return grpc.Errorf(grpc.Code(err), "received error from client")
	}
	log.V(1).Infof("Client %s recieved initial query: %v", c, query)

	c.subscribe = query.GetSubscribe()
	if c.subscribe == nil {
		c.errors++
		return grpc.Errorf(codes.InvalidArgument, "first message must be SubscriptionList: %q", query)
	}
	c.reset() // Initialize the queue used between send and recv.

	log.V(1).Infof("Client %s running", c)
	go c.recv(stream)
	c.send(stream)
	log.V(1).Infof("Client %s shutdown", c)
	return nil
}

// Close will cancel the client context and will cause the send and recv goroutines to exit.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.canceled = true
}

// Config returns the current config of the client.
func (c *Client) Config() *fpb.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

var syncResp = &ocpb.SubscribeResponse{
	Response: &ocpb.SubscribeResponse_SyncResponse{
		SyncResponse: true,
	},
}

func (c *Client) recv(stream ocpb.OpenConfig_SubscribeServer) {
	for {
		event, err := stream.Recv()
		c.recvMsg++
		switch err {
		default:
			log.V(1).Infof("Client %s received error: %v", c, err)
			c.Close()
			return
		case io.EOF:
			log.V(1).Infof("Client %s received io.EOF", c)
			return
		case nil:
		}
		if c.subscribe.Mode == ocpb.SubscriptionList_POLL {
			log.V(1).Infof("Client %s received Poll event: %v", c, event)
			if _, ok := event.Request.(*ocpb.SubscribeRequest_Poll); !ok {
				log.V(1).Infof("Client %s received invalid Poll event: %v", c, event)
				c.Close()
				return
			}
			c.reset()
			c.polled <- struct{}{}
			continue
		}
		log.V(1).Infof("Client %s received invalid event: %s", c, event)
	}
}

// processQueue makes a copy of q then will process values in the queue until
// the queue is complete or an error.  Each value is converted into a gNMI
// notification and sent on stream.
func (c *Client) processQueue(stream ocpb.OpenConfig_SubscribeServer) error {
	c.mu.RLock()
	q := c.q
	c.mu.RUnlock()
	for {
		c.mu.RLock()
		canceled := c.canceled
		c.mu.RUnlock()
		if canceled {
			return fmt.Errorf("client canceled")
		}
		event, err := q.Next()
		c.sendMsg++
		if err != nil {
			c.errors++
			return fmt.Errorf("unexpected queue Next(): %v", err)
		}
		if event == nil {
			switch {
			case c.subscribe.Mode == ocpb.SubscriptionList_POLL:
				<-c.polled
				log.V(1).Infof("Client %s received poll", c)
				return nil
			case c.config.DisableEof:
				return fmt.Errorf("send exiting due to disabled EOF")
			}
			return fmt.Errorf("end of updates")
		}
		v, err := valToResp(event)
		if err != nil {
			c.errors++
			return err
		}
		log.V(1).Infof("Client %s sending:\n%s", c, v)
		err = stream.Send(v)
		if err != nil {
			c.errors++
			return err
		}
	}
}

// send runs until process Queue returns an error. Each loop is meant to allow
// for a reset of the sending queue based on query type.
func (c *Client) send(stream ocpb.OpenConfig_SubscribeServer) {
	for {
		if err := c.processQueue(stream); err != nil {
			log.Errorf("Client %s error: %v", c, err)
			return
		}
	}
}

// SetConfig will replace the current configuration of the Client. If the client
// is running then the change will not take effect until the queue is drained
// of notifications.
func (c *Client) SetConfig(config *fpb.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
}

func (c *Client) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.V(1).Infof("Client %s using config:\n%s", c, pretty.Sprint(c.config))
	c.q = queue.New(false, c.config.Seed, c.config.Values)
	// Inject sync message after latest provided update in the config.
	if !c.config.DisableSync {
		c.q.Add(&fpb.Value{
			Timestamp: &fpb.Timestamp{Timestamp: c.q.Latest()},
			Repeat:    1,
			Value:     &fpb.Value_Sync{uint64(1)},
		})
	}
}

// valToResp converts a fake_proto Value to its corresponding gNMI proto stream
// response type.
// fake_proto sync values are converted to gNMI subscribe responses containing
// SyncResponses.
// All other fake_proto values are assumed to be gNMI subscribe responses
// containing Updates.
func valToResp(val *fpb.Value) (*ocpb.SubscribeResponse, error) {
	switch val.GetValue().(type) {
	case *fpb.Value_Delete:
		return &ocpb.SubscribeResponse{
			Response: &ocpb.SubscribeResponse_Update{
				Update: &ocpb.Notification{
					Timestamp: val.Timestamp.Timestamp,
					Delete:    []*ocpb.Path{{Element: val.Path}},
				},
			},
		}, nil
	case *fpb.Value_Sync:
		var sync bool
		if queue.ValueOf(val).(uint64) > 0 {
			sync = true
		}
		return &ocpb.SubscribeResponse{
			Response: &ocpb.SubscribeResponse_SyncResponse{
				SyncResponse: sync,
			},
		}, nil
	}
	b, err := json.Marshal(queue.ValueOf(val))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value %s: %s", val, err)
	}
	return &ocpb.SubscribeResponse{
		Response: &ocpb.SubscribeResponse_Update{
			Update: &ocpb.Notification{
				Timestamp: val.Timestamp.Timestamp,
				Update: []*ocpb.Update{{
					Path:  &ocpb.Path{Element: val.Path},
					Value: &ocpb.Value{Value: b},
				}},
			},
		},
	}, nil
}
