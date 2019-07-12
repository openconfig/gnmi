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
	"context"
	"errors"
	"io"
	"time"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"github.com/openconfig/gnmi/cache"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/coalesce"
	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/gnmi/match"
	"github.com/openconfig/gnmi/path"
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

type aclStub struct{}

func (a *aclStub) Check(string) bool {
	return true
}

// RPCACL is per RPC ACL interface
type RPCACL interface {
	Check(string) bool
}

// ACL is server ACL interface
type ACL interface {
	NewRPCACL(context.Context) (RPCACL, error)
	Check(string, string) bool
}

// Server is the implementation of the gNMI Subcribe API.
type Server struct {
	unimplemented.Server // Stub out all RPCs except Subscribe.

	c *cache.Cache // The cache queries are performed against.
	m *match.Match // Structure to match updates against active subscriptions.
	a ACL          // server ACL.
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

// SetACL sets server ACL. This method is called before server starts to run.
func (s *Server) SetACL(a ACL) {
	s.a = a
}

// Update passes a streaming update to registered clients.
func (s *Server) Update(n *ctree.Leaf) {
	switch v := n.Value().(type) {
	case client.Delete:
		s.m.Update(n, v.Path)
	case client.Update:
		s.m.Update(n, v.Path)
	case *pb.Notification:
		p := path.ToStrings(v.Prefix, true)
		if len(v.Update) > 0 {
			p = append(p, path.ToStrings(v.Update[0].Path, false)...)
		} else if len(v.Delete) > 0 {
			p = append(p, path.ToStrings(v.Delete[0], false)...)
		}
		// If neither update nor delete notification exists,
		// just go with the path in the prefix
		s.m.Update(n, p)
	default:
		log.Errorf("update is not a known type; type is %T", v)
	}
}

// addSubscription registers all subscriptions for this client for update matching.
func addSubscription(m *match.Match, s *pb.SubscriptionList, c *matchClient) (remove func()) {
	var removes []func()
	prefix := path.ToStrings(s.Prefix, true)
	for _, p := range s.Subscription {
		if p.Path == nil {
			continue
		}
		// TODO(yusufsn) : Origin field in the Path may need to be included
		path := append(prefix, path.ToStrings(p.Path, false)...)
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
	c := streamClient{stream: stream, acl: &aclStub{}}
	var err error
	if s.a != nil {
		a, err := s.a.NewRPCACL(stream.Context())
		if err != nil {
			log.Errorf("NewRPCACL fails due to %v", err)
			return status.Error(codes.Unauthenticated, "no authentication/authorization for requested operation")
		}
		c.acl = a
	}
	c.sr, err = stream.Recv()

	switch {
	case err == io.EOF:
		return nil
	case err != nil:
		return err
	case c.sr.GetSubscribe() == nil:
		return status.Errorf(codes.InvalidArgument, "request must contain a subscription %#v", c.sr)
	case c.sr.GetSubscribe().GetPrefix() == nil:
		return status.Errorf(codes.InvalidArgument, "request must contain a prefix %#v", c.sr)
	case c.sr.GetSubscribe().GetPrefix().GetTarget() == "":
		return status.Error(codes.InvalidArgument, "missing target")
	}

	c.target = c.sr.GetSubscribe().GetPrefix().GetTarget()
	if !s.c.HasTarget(c.target) {
		return status.Errorf(codes.NotFound, "no such target: %q", c.target)
	}
	peer, _ := peer.FromContext(stream.Context())
	mode := c.sr.GetSubscribe().Mode

	log.Infof("peer: %v target: %q subscription: %s", peer.Addr, c.target, c.sr)
	defer log.Infof("peer: %v target %q subscription: end: %q", peer.Addr, c.target, c.sr)

	c.queue = coalesce.NewQueue()
	defer c.queue.Close()

	// reject single device subscription if not allowed by ACL
	if c.target != "*" && !c.acl.Check(c.target) {
		return status.Errorf(codes.PermissionDenied, "not authorized for target %q", c.target)
	}
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
			c.queue.Insert(syncMarker{})
		}
		remove := addSubscription(s.m, c.sr.GetSubscribe(),
			&matchClient{acl: c.acl, q: c.queue})
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
	dup    uint32
	t      *time.Timer // Timer used to timout the subscription.
}

// sendSubscribeResponse populates and sends a single response returned on
// the Subscription RPC output stream. Streaming queries send responses for the
// initial walk of the results as well as streamed updates and use a queue to
// ensure order.
func (s *Server) sendSubscribeResponse(r *resp, c *streamClient) error {
	notification, err := MakeSubscribeResponse(r.n.Value(), r.dup)
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}

	if pre := notification.GetUpdate().GetPrefix(); pre != nil {
		if !c.acl.Check(pre.GetTarget()) {
			// reaching here means notification is denied for sending.
			// return with no error. function caller can continue for next one.
			return nil
		}
	}

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

type syncMarker struct{}

// cacheClient implements match.Client interface.
type matchClient struct {
	acl RPCACL
	q   *coalesce.Queue
	err error
}

// Update implements the match.Client Update interface for coalesce.Queue.
func (c matchClient) Update(n interface{}) {
	// Stop processing updates on error.
	if c.err != nil {
		return
	}
	_, c.err = c.q.Insert(n)
}

type streamClient struct {
	acl    RPCACL
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
	log.V(2).Infof("start processSubscription for %p", c)
	// Close the cache client queue on error.
	defer func() {
		if err != nil {
			log.Error(err)
			c.queue.Close()
			c.errC <- err
		}
		log.V(2).Infof("end processSubscription for %p", c)
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
		for _, subscription := range c.sr.GetSubscribe().Subscription {
			var fullPath []string
			fullPath, err = path.CompletePath(c.sr.GetSubscribe().GetPrefix(), subscription.GetPath())
			if err != nil {
				return
			}
			// Note that fullPath doesn't contain target name as the first element.
			s.c.Query(c.target, fullPath, func(_ []string, l *ctree.Leaf, _ interface{}) {
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

	_, err = c.queue.Insert(syncMarker{})
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
	ctx := c.stream.Context()
	peer, _ := peer.FromContext(ctx)
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
		item, dup, err := c.queue.Next(ctx)
		if coalesce.IsClosedQueue(err) {
			c.errC <- nil
			return
		}
		if err != nil {
			c.errC <- err
			return
		}

		// s.processSubscription will send a sync marker, handle it separately.
		if _, ok := item.(syncMarker); ok {
			if err = c.stream.Send(subscribeSync); err != nil {
				c.errC <- err
				return
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
		}, c); err != nil {
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

// MakeSubscribeResponse produces a gnmi_proto.SubscribeResponse from either
// client.Notification or gnmi_proto.Notification
//
// This function modifies the message to set the duplicate count if it is
// greater than 0. The function clones the gnmi notification if the duplicate count needs to be set.
// You have to be working on a cloned message if you need to modify the message in any way.
func MakeSubscribeResponse(n interface{}, dup uint32) (*pb.SubscribeResponse, error) {
	var notification *pb.Notification
	switch cache.Type {
	case cache.GnmiNoti:
		var ok bool
		notification, ok = n.(*pb.Notification)
		if !ok {
			return nil, status.Errorf(codes.Internal, "invalid notification type: %#v", n)
		}

		// There may be multiple updates in a notification. Since duplicate count is just
		// an indicator that coalescion is happening, not a critical data, just the first
		// update is set with duplicate count to be on the side of efficiency.
		// Only attempt to set the duplicate count if it is greater than 0. The default
		// value in the message is already 0.
		if dup > 0 && len(notification.Update) > 0 {
			// We need a copy of the cached notification before writing a client specific
			// duplicate count as the notification is shared across all clients.
			notification = proto.Clone(notification).(*pb.Notification)
			notification.Update[0].Duplicates = dup
		}
	case cache.ClientLeaf:
		notification = &pb.Notification{}
		switch v := n.(type) {
		case client.Delete:
			notification.Delete = []*pb.Path{{Element: v.Path}}
			notification.Timestamp = v.TS.UnixNano()
		case client.Update:
			typedVal, err := value.FromScalar(v.Val)
			if err != nil {
				return nil, err
			}
			notification.Update = []*pb.Update{{Path: &pb.Path{Element: v.Path}, Val: typedVal, Duplicates: dup}}
			notification.Timestamp = v.TS.UnixNano()
		}
	}
	response := &pb.SubscribeResponse{
		Response: &pb.SubscribeResponse_Update{
			Update: notification,
		},
	}

	return response, nil
}
