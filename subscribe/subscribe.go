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

// Package subscribe implements the gnmi.proto Subscribe service API.
package subscribe

import (
	"errors"
	"io"
	"time"

	log "github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"github.com/openconfig/gnmi/cache"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/coalesce"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/match"
	"github.com/openconfig/gnmi/unimplemented"
	"github.com/openconfig/gnmi/value"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	// Value overridden in tests to simulate flow control.
	flowControlTest = func() {}
	// Timeout specifies how long a send can be pending before the RPC is closed.
	Timeout = time.Minute
	// SubscriptionLimit specifies how many queries can be processing simultaneously.
	// This number includes Once queries, Polling subscriptions, and Streaming
	// subscriptions that have not yet synced. Once a streaming subscription has
	// synced, it no longer counts against the limit. A polling subscription
	// counts against the limit during each polling cycle while it is processed.
	SubscriptionLimit = 0
	// Value overridden in tests to evaluate SubscriptionLimit enforcement.
	subscriptionLimitTest = func() {}
)

// Server is the implementation of the gNMI Subcribe API.
type Server struct {
	unimplemented.Server // Stub out all RPCs except Subscribe.

	c *cache.Cache // The cache queries are performed against.
	m *match.Match // Structure to match updates against active subscriptions.
	// subscribeSlots is a channel of size SubscriptionLimit to restrict how many
	// queries are in flight.
	subscribeSlots chan struct{}
	timeout        time.Duration
}

// NewServer instantiates server to handle client queries.  The cache should be
// already instantiated.
func NewServer(c *cache.Cache) (*Server, error) {
	s := &Server{c: c, m: match.New(), timeout: Timeout}
	if SubscriptionLimit > 0 {
		s.subscribeSlots = make(chan struct{}, SubscriptionLimit)
	}
	return s, nil
}

// Update passes a streaming update to registered clients.
func (s *Server) Update(n *ctree.Leaf) {
	s.m.Update(n)
}

// addSubscription registers all subscriptions for this client for update matching.
func addSubscription(m *match.Match, target string, s *pb.SubscriptionList, c *matchClient) (remove func()) {
	var removes []func()
	prefix := []string{target}
	if s.Prefix != nil {
		prefix = append(prefix, s.Prefix.Element...)
	}
	for _, p := range s.Subscription {
		if p.Path == nil {
			continue
		}
		path := append(prefix, p.Path.Element...)
		removes = append(removes, m.AddQuery(path, c))
	}
	return func() {
		for _, remove := range removes {
			remove()
		}
	}
}

// Subscribe is the entry point for the external RPC request of the same name
// defined in gnmi.proto.
func (s *Server) Subscribe(stream pb.GNMI_SubscribeServer) error {
	c := streamClient{stream: stream}
	var err error
	c.sr, err = stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	if c.target = c.sr.GetSubscribe().GetPrefix().GetTarget(); c.target == "" {
		return status.Error(codes.InvalidArgument, "missing target")
	}
	if !s.c.HasTarget(c.target) {
		return status.Errorf(codes.NotFound, "no such target: %q", c.target)
	}
	peer, _ := peer.FromContext(stream.Context())
	if c.sr.GetSubscribe() == nil {
		return status.Errorf(codes.InvalidArgument, "request must contain a subscription %#v", c.sr)
	}
	mode := c.sr.GetSubscribe().Mode

	log.Infof("peer: %v target: %q subscription: %s", peer.Addr, c.target, c.sr)
	defer log.Infof("peer: %v target %q subscription: end: %q", peer.Addr, c.target, c.sr)

	c.queue = coalesce.NewQueue()
	defer c.queue.Close()

	// This error channel is buffered to accept errors from all goroutines spawned
	// for this RPC.  Only the first is ever read and returned causing the RPC to
	// terminate.
	errC := make(chan error, 3)
	c.errC = errC

	switch mode {
	case pb.SubscriptionList_ONCE:
		go func() {
			s.processSubscription(&c)
			c.queue.Close()
		}()
	case pb.SubscriptionList_POLL:
		go s.processPollingSubscription(&c)
	case pb.SubscriptionList_STREAM:
		if c.sr.GetSubscribe().GetUpdatesOnly() {
			c.queue.Insert(sync{})
		}
		remove := addSubscription(s.m, c.target, c.sr.GetSubscribe(), &matchClient{q: c.queue})
		defer remove()
		if !c.sr.GetSubscribe().GetUpdatesOnly() {
			go s.processSubscription(&c)
		}
	default:
		return status.Errorf(codes.InvalidArgument, "Subscription mode %v not recognized", mode)
	}

	go s.sendStreamingResults(&c)

	return <-errC
}

type resp struct {
	stream pb.GNMI_SubscribeServer
	n      *ctree.Leaf
	dup    int
	t      *time.Timer // Timer used to timout the subscription.
}

// sendSubscribeResponse populates and sends a single response returned on
// the Subscription RPC output stream. Streaming queries send responses for the
// initial walk of the results as well as streamed updates and use a queue to
// ensure order.
func (s *Server) sendSubscribeResponse(r *resp) error {
	notification, err := MakeSubscribeResponse(r.n.Value())
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	// gNMI Update does not include a field to report coalesced duplicates.  When
	// it does, r.dup holds the duplicates for this value.
	// Start the timeout before attempting to send.
	r.t.Reset(s.timeout)
	// Clear the timeout upon sending.
	defer r.t.Stop()
	// An empty function in production, replaced in test to simulate flow control
	// by blocking before send.
	flowControlTest()
	return r.stream.Send(notification)
}

// subscribeSync is a response indicating that a Subscribe RPC has successfully
// returned all matching nodes once for ONCE and POLLING queries and at least
// once for STREAMING queries.
var subscribeSync = &pb.SubscribeResponse{Response: &pb.SubscribeResponse_SyncResponse{true}}

type sync struct{}

// cacheClient implements match.Client interface.
type matchClient struct {
	q   *coalesce.Queue
	err error
}

// Update implements the match.Client Update interface for coalesce.Queue.
func (c matchClient) Update(n *ctree.Leaf) {
	// Stop processing updates on error.
	if c.err != nil {
		return
	}
	_, c.err = c.q.Insert(n)
}

type streamClient struct {
	target string
	sr     *pb.SubscribeRequest
	queue  *coalesce.Queue
	stream pb.GNMI_SubscribeServer
	errC   chan<- error
}

// processSubscription walks the cache tree and inserts all of the matching
// nodes into the coalesce queue followed by a subscriptionSync response.
func (s *Server) processSubscription(c *streamClient) {
	var err error
	// Close the cache client queue on error.
	defer func() {
		if err != nil {
			log.Error(err)
			c.queue.Close()
			c.errC <- err
		}
	}()
	if s.subscribeSlots != nil {
		select {
		// Register a subscription in the channel, which will block if SubscriptionLimit queries
		// are already in flight.
		case s.subscribeSlots <- struct{}{}:
		default:
			log.V(2).Infof("subscription %s delayed waiting for 1 of %d subscription slots.", c.sr, len(s.subscribeSlots))
			s.subscribeSlots <- struct{}{}
			log.V(2).Infof("subscription %s resumed", c.sr)
		}
		// Remove subscription from the channel upon completion.
		defer func() {
			// Artificially hold subscription processing in tests to synchronously test limit.
			subscriptionLimitTest()
			<-s.subscribeSlots
		}()
	}
	if !c.sr.GetSubscribe().GetUpdatesOnly() {
		var prefix []string
		if s := c.sr.GetSubscribe(); s != nil {
			if p := s.Prefix; p != nil {
				prefix = p.Element
			}
		}
		for _, subscription := range c.sr.GetSubscribe().Subscription {
			path := append(prefix, subscription.Path.Element...)
			s.c.Query(c.target, path, func(_ []string, l *ctree.Leaf, _ interface{}) {
				// Stop processing query results on error.
				if err != nil {
					return
				}
				_, err = c.queue.Insert(l)
			})
			if err != nil {
				return
			}
		}
	}

	_, err = c.queue.Insert(sync{})
}

// processPollingSubscription handles the POLL mode Subscription RPC.
func (s *Server) processPollingSubscription(c *streamClient) {
	s.processSubscription(c)
	log.Infof("polling subscription: first complete response: %q", c.sr)
	for {
		if c.queue.IsClosed() {
			return
		}
		// Subsequent receives are only triggers to poll again. The contents of the
		// request are completely ignored.
		_, err := c.stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error(err)
			c.errC <- err
			return
		}
		log.Infof("polling subscription: repoll: %q", c.sr)
		s.processSubscription(c)
		log.Infof("polling subscription: repoll complete: %q", c.sr)
	}
}

// sendStreamingResults forwards all streaming updates to a given streaming
// Subscription RPC client.
func (s *Server) sendStreamingResults(c *streamClient) {
	peer, _ := peer.FromContext(c.stream.Context())
	t := time.NewTimer(s.timeout)
	// Make sure the timer doesn't expire waiting for a value to send, only
	// waiting to send.
	t.Stop()
	done := make(chan struct{})
	defer close(done)
	// If a send doesn't occur within the timeout, close the stream.
	go func() {
		select {
		case <-t.C:
			err := errors.New("subscription timed out while sending")
			c.errC <- err
			log.Errorf("%v : %v", peer, err)
		case <-done:
		}
	}()
	for {
		item, dup, err := c.queue.Next()
		if coalesce.IsClosedQueue(err) {
			c.errC <- nil
			return
		}
		if err != nil {
			c.errC <- err
			return
		}

		// s.processSubscription will send a sync marker, handle it separately.
		if _, ok := item.(sync); ok {
			if err = c.stream.Send(subscribeSync); err != nil {
				break
			}
			continue
		}

		n, ok := item.(*ctree.Leaf)
		if !ok || n == nil {
			c.errC <- status.Errorf(codes.Internal, "invalid cache node: %#v", item)
			return
		}
		if err = s.sendSubscribeResponse(&resp{
			stream: c.stream,
			n:      n,
			dup:    dup,
			t:      t,
		}); err != nil {
			c.errC <- err
			return
		}
		// If the only target being subscribed was deleted, stop streaming.
		if cache.IsTargetDelete(n) && c.target != "*" {
			log.Infof("Target %q was deleted. Closing stream.", c.target)
			c.errC <- nil
			return
		}
	}
}

// MakeSubscribeResponse produces a gnmi_proto.SubscribeResponse from a client Notification.
func MakeSubscribeResponse(n interface{}) (*pb.SubscribeResponse, error) {
	notification := &pb.Notification{}
	response := &pb.SubscribeResponse{
		Response: &pb.SubscribeResponse_Update{
			Update: notification,
		},
	}
	switch v := n.(type) {
	case client.Delete:
		notification.Delete = []*pb.Path{{Element: v.Path}}
		notification.Timestamp = v.TS.UnixNano()
	case client.Update:
		typedVal, err := value.FromScalar(v.Val)
		if err != nil {
			return nil, err
		}
		notification.Update = []*pb.Update{{Path: &pb.Path{Element: v.Path}, Val: typedVal}}
		notification.Timestamp = v.TS.UnixNano()
	}
	return response, nil
}
