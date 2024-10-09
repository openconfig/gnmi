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

package gnmi

import (
	"fmt"
	"io"
	"sync"

	log "github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"github.com/openconfig/gnmi/testing/fake/queue"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

// Client contains information about a client that has connected to the fake.
type Client struct {
	errors     int64
	subscribe  *gpb.SubscriptionList
	polled     chan struct{}
	canceledCh chan struct{}

	mu       sync.RWMutex // mutex for read/write access to config and canceled values
	config   *fpb.Config
	canceled bool

	qMu sync.Mutex // mutex for accessing the queue
	q   queue.Queue

	rMu      sync.Mutex // mutex for accessing requests
	requests []*gpb.SubscribeRequest
}

// NewClient returns a new initialized client.
func NewClient(config *fpb.Config) *Client {
	return &Client{
		config:     config,
		polled:     make(chan struct{}),
		canceledCh: make(chan struct{}, 1),
	}
}

// String returns the target the client is querying.
func (c *Client) String() string {
	return c.config.Target
}

// addRequest adds a subscribe request to the list of requests received by the client.
func (c *Client) addRequest(req *gpb.SubscribeRequest) {
	c.rMu.Lock()
	defer c.rMu.Unlock()
	c.requests = append(c.requests, req)
}

// Run starts the client. The first message received must be a
// SubscriptionList. Once the client is started, it will run until the stream
// is closed or the schedule completes. For Poll queries the Run will block
// internally after sync until a Poll request is made to the server. This is
// important as the test may look like a deadlock since it can cause a timeout.
// Also if you Reset the client the change will not take effect until after the
// previous queue has been drained of notifications.
func (c *Client) Run(stream gpb.GNMI_SubscribeServer) (err error) {
	if c.config == nil {
		return grpc.Errorf(codes.FailedPrecondition, "cannot start client: config is nil")
	}
	if stream == nil {
		return grpc.Errorf(codes.FailedPrecondition, "cannot start client: stream is nil")
	}

	defer func() {
		if err != nil {
			c.errors++
		}
	}()

	query, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return grpc.Errorf(codes.Aborted, "stream EOF received before init")
		}
		return grpc.Errorf(grpc.Code(err), "received error from client")
	}
	c.addRequest(query)
	log.V(1).Infof("Client %s received initial query: %v", c, query)

	c.subscribe = query.GetSubscribe()
	if c.subscribe == nil {
		return grpc.Errorf(codes.InvalidArgument, "first message must be SubscriptionList: %q", query)
	}
	// Initialize the queue used between send and recv.
	if err = c.reset(); err != nil {
		return grpc.Errorf(codes.Aborted, "failed to initialize the queue: %v", err)
	}

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
	if c.config != nil && c.config.DisableEof {
		// If the cancel signal has already been sent by a previous close, then we
		// do not need to resend it.
		select {
		case c.canceledCh <- struct{}{}:
		default:
		}
	}
	c.canceled = true
}

// Config returns the current config of the client.
func (c *Client) Config() *fpb.Config {
	return c.config
}

var syncResp = &gpb.SubscribeResponse{
	Response: &gpb.SubscribeResponse_SyncResponse{
		SyncResponse: true,
	},
}

func (c *Client) recv(stream gpb.GNMI_SubscribeServer) {
	for {
		event, err := stream.Recv()
		switch err {
		default:
			log.V(1).Infof("Client %s received error: %v", c, err)
			c.Close()
			return
		case io.EOF:
			log.V(1).Infof("Client %s received io.EOF", c)
			return
		case nil:
			c.addRequest(event)
		}
		if c.subscribe.Mode == gpb.SubscriptionList_POLL {
			log.V(1).Infof("Client %s received Poll event: %v", c, event)
			if _, ok := event.Request.(*gpb.SubscribeRequest_Poll); !ok {
				log.V(1).Infof("Client %s received invalid Poll event: %v", c, event)
				c.Close()
				return
			}
			if err = c.reset(); err != nil {
				c.Close()
				return
			}
			c.polled <- struct{}{}
			continue
		}
		log.V(1).Infof("Client %s received invalid event: %s", c, event)
	}
}

// Requests returns the subscribe requests received by the client.
func (c *Client) Requests() []*gpb.SubscribeRequest {
	c.rMu.Lock()
	defer c.rMu.Unlock()
	return c.requests
}

// isCanceled returned whether the client has canceled.
func (c *Client) isCanceled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.canceled
}

// setQueue sets the client's queue to the provided value.
func (c *Client) setQueue(q queue.Queue) {
	c.qMu.Lock()
	defer c.qMu.Unlock()
	c.q = q
}

// nextInQueue returns the next event in the client's queue, unless the client is canceled.
func (c *Client) nextInQueue() (any, error) {
	c.qMu.Lock()
	defer c.qMu.Unlock()
	if c.q == nil {
		return nil, fmt.Errorf("nil client queue: nothing to do")
	}
	if c.isCanceled() {
		return nil, fmt.Errorf("client canceled")
	}
	n, err := c.q.Next()
	if err != nil {
		c.errors++
		return nil, fmt.Errorf("unexpected queue Next(): %v", err)
	}
	return n, nil
}

// processQueue makes a copy of q then will process values in the queue until
// the queue is complete or an error.  Each value is converted into a gNMI
// notification and sent on stream.
func (c *Client) processQueue(stream gpb.GNMI_SubscribeServer) error {
	for {
		event, err := c.nextInQueue()
		if err != nil {
			return err
		}

		if event == nil {
			switch {
			case c.subscribe.Mode == gpb.SubscriptionList_POLL:
				<-c.polled
				log.V(1).Infof("Client %s received poll", c)
				return nil
			case c.config.DisableEof:
				// Not an error, we ran out of values, but DisableEof asks us to hold open
				// the subscription.
				return nil
			}
			return fmt.Errorf("end of updates")
		}
		var resp *gpb.SubscribeResponse
		switch v := event.(type) {
		case *fpb.Value:
			if resp, err = valToResp(v); err != nil {
				c.errors++
				return err
			}
		case *gpb.SubscribeResponse:
			// Copy into resp to not modify the original.
			resp = proto.Clone(v).(*gpb.SubscribeResponse)
		}
		// If the subscription request specified a target explicitly...
		if sp := c.subscribe.GetPrefix(); sp != nil {
			if target := sp.Target; target != "" {
				// and the message is an update...
				if update := resp.GetUpdate(); update != nil {
					// then set target in the prefix.
					if update.Prefix == nil {
						update.Prefix = &gpb.Path{}
					}
					update.Prefix.Target = target
				}
			}
		}
		err = stream.Send(resp)
		if err != nil {
			c.errors++
			return err
		}
	}
}

// send runs until process Queue returns an error. Each loop is meant to allow
// for a reset of the sending queue based on query type.
func (c *Client) send(stream gpb.GNMI_SubscribeServer) {
	for {
		if err := c.processQueue(stream); err != nil {
			log.Errorf("Client %s error: %v", c, err)
			return
		}
		if c.config.DisableEof {
			log.V(1).Infof("Client %s: holding open connection to be shut down", c)
			<-c.canceledCh
			log.V(1).Infof("Client %s: cancelled by channel", c)
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

func (c *Client) reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.V(1).Infof("Client %s using config:\n%s", c, prototext.Format(c.config))
	switch {
	default:
		q := queue.New(c.config.GetEnableDelay(), c.config.Seed, c.config.Values)
		// Inject sync message after latest provided update in the config.
		if !c.config.DisableSync {
			q.Add(&fpb.Value{
				Timestamp: &fpb.Timestamp{Timestamp: q.Latest()},
				Repeat:    1,
				Value:     &fpb.Value_Sync{uint64(1)},
			})
		}
		c.setQueue(q)
	case c.config.GetFixed() != nil:
		q := queue.NewFixed(c.config.GetFixed().Responses, c.config.EnableDelay)
		// Inject sync message after latest provided update in the config.
		if !c.config.DisableSync {
			q.Add(syncResp)
		}
		c.setQueue(q)
	}
	return nil
}

// valToResp converts a fake_proto Value to its corresponding gNMI proto stream
// response type.
// fake_proto sync values are converted to gNMI subscribe responses containing
// SyncResponses.
// All other fake_proto values are assumed to be gNMI subscribe responses
// containing Updates.
func valToResp(val *fpb.Value) (*gpb.SubscribeResponse, error) {
	switch val.GetValue().(type) {
	case *fpb.Value_Delete:
		return &gpb.SubscribeResponse{
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: val.Timestamp.Timestamp,
					Delete:    []*gpb.Path{{Element: val.Path}},
				},
			},
		}, nil
	case *fpb.Value_Sync:
		var sync bool
		if queue.ValueOf(val).(uint64) > 0 {
			sync = true
		}
		return &gpb.SubscribeResponse{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: sync,
			},
		}, nil
	default:
		tv := queue.TypedValueOf(val)
		if tv == nil {
			return nil, fmt.Errorf("failed to get TypedValue of %s", val)
		}
		return &gpb.SubscribeResponse{
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: val.Timestamp.Timestamp,
					Update: []*gpb.Update{
						{
							Path: &gpb.Path{Element: val.Path},
							Val:  tv,
						},
					},
				},
			},
		}, nil
	}
}
