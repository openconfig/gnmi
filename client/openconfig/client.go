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

// Package client contains transport implementation for the parent client
// library using openconfig.proto.
//
// Note: this package should not be used directly. Use
// github.com/openconfig/gnmi/client instead.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/golang/glog"
	"context"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/client/grpcutil"

	ocpb "github.com/openconfig/reference/rpc/openconfig"
)

// Type defines the name resolution for this client type.
const Type = "openconfig"

// Client handles execution of the query and caching of its results.
type Client struct {
	conn      *grpc.ClientConn
	client    ocpb.OpenConfigClient
	sub       ocpb.OpenConfig_SubscribeClient
	request   *ocpb.SubscribeRequest
	query     client.Query
	recv      client.ProtoHandler
	handler   client.NotificationHandler
	connected bool
}

// New returns a new initialized client. The client implementation will have
// made a connection to the backend and has sent the initial query q.
func New(ctx context.Context, d client.Destination) (client.Impl, error) {
	if len(d.Addrs) != 1 {
		return nil, fmt.Errorf("q.Addrs must only contain one entry: %v", d.Addrs)
	}
	opts := []grpc.DialOption{
		grpc.WithTimeout(d.Timeout),
		grpc.WithBlock(),
	}
	if d.TLS != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(d.TLS)))
	}
	conn, err := grpc.Dial(d.Addrs[0], opts...)
	if err != nil {
		return nil, fmt.Errorf("Dialer(%s, %v): %v", d.Addrs[0], d.Timeout, err)
	}
	return NewFromConn(ctx, conn, d)
}

// NewFromConn creates and returns the client based on the provided transport.
func NewFromConn(ctx context.Context, conn *grpc.ClientConn, d client.Destination) (*Client, error) {
	ok, err := grpcutil.Lookup(ctx, conn, "openconfig.OpenConfig")
	if err != nil {
		log.V(1).Infof("gRPC reflection lookup on %q for service openconfig.OpenConfig failed: %v", d.Addrs, err)
		// This check is disabled for now. Reflection will become part of gNMI
		// specification in the near future, so we can't enforce it yet.
	}
	if !ok {
		// This check is disabled for now. Reflection will become part of gNMI
		// specification in the near future, so we can't enforce it yet.
	}
	oc := ocpb.NewOpenConfigClient(conn)
	c := &Client{
		conn:   conn,
		client: oc,
	}
	return c, nil
}

// Subscribe initializes the Subscribe RPC.
func (c *Client) Subscribe(ctx context.Context, q client.Query) error {
	sub, err := c.client.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("ocpb.NewGNMIClient(%v) failed to create client: %v", q, err)
	}
	qq := subscribe(q)
	if err := sub.Send(qq); err != nil {
		return fmt.Errorf("client.Send(%+v): %v", qq, err)
	}
	c.sub = sub
	c.query = q
	c.request = qq

	if q.ProtoHandler == nil {
		c.recv = c.defaultRecv
		c.handler = q.NotificationHandler
	} else {
		c.recv = q.ProtoHandler
	}
	return nil
}

// Poll will send a single gNMI poll request to the server.
func (c *Client) Poll() error {
	if err := c.sub.Send(&ocpb.SubscribeRequest{Request: &ocpb.SubscribeRequest_Poll{Poll: &ocpb.PollRequest{}}}); err != nil {
		return fmt.Errorf("client.Poll(): %v", err)
	}
	return nil
}

// Peer returns the peer of the current stream. If the client is not created or
// if the peer is not valid nil is returned.
func (c *Client) Peer() string {
	return c.query.Addrs[0]
}

// Close forcefully closes the underlying connection, terminating the query
// right away. It's safe to call Close multiple times.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Recv will receive a single message from the server and process it based on
// the provided handlers (Proto or Notification).
func (c *Client) Recv() error {
	n, err := c.sub.Recv()
	if err != nil {
		return err
	}
	return c.recv(n)
}

// Set calls the Set RPC, converting request/response to appropriate protos.
func (c *Client) Set(ctx context.Context, sr client.SetRequest) (client.SetResponse, error) {
	req, err := convertSetRequest(sr)
	if err != nil {
		return client.SetResponse{}, err
	}

	resp, err := c.client.Set(ctx, req)
	if err != nil {
		return client.SetResponse{}, err
	}

	return convertSetResponse(resp)
}

func convertSetRequest(sr client.SetRequest) (*ocpb.SetRequest, error) {
	req := &ocpb.SetRequest{}
	for _, d := range sr.Delete {
		p := ocpb.Path{Element: d}
		req.Delete = append(req.Delete, &p)
	}

	genUpdate := func(v client.Leaf) (*ocpb.Update, error) {
		buf, err := json.Marshal(v.Val)
		if err != nil {
			return nil, err
		}
		return &ocpb.Update{
			Path:  &ocpb.Path{Element: v.Path},
			Value: &ocpb.Value{Type: ocpb.Type_JSON, Value: buf},
		}, nil
	}

	for _, u := range sr.Update {
		uu, err := genUpdate(u)
		if err != nil {
			return nil, err
		}
		req.Update = append(req.Update, uu)
	}
	for _, u := range sr.Replace {
		uu, err := genUpdate(u)
		if err != nil {
			return nil, err
		}
		req.Replace = append(req.Replace, uu)
	}

	return req, nil
}

func convertSetResponse(sr *ocpb.SetResponse) (client.SetResponse, error) {
	var resp client.SetResponse
	var errs []string
	for i, r := range sr.GetResponse() {
		if i == 0 {
			resp.TS = time.Unix(0, r.Timestamp)
		}
		if r.Message != nil {
			errs = append(errs, r.GetMessage().String())
		}
	}
	if len(errs) > 0 {
		return resp, errors.New(strings.Join(errs, "; "))
	}

	return resp, nil
}

// defaultRecv is the default implementation of recv provided by the client.
// This function will be replaced by the ProtoHandler member of the Query
// struct passed to New(), if it is set.
func (c *Client) defaultRecv(msg proto.Message) error {
	if !c.connected {
		c.handler(client.Connected{})
		c.connected = true
	}

	resp, ok := msg.(*ocpb.SubscribeResponse)
	if !ok {
		return fmt.Errorf("failed to type assert message %#v", msg)
	}
	log.V(1).Info(resp)
	switch v := resp.Response.(type) {
	default:
		return fmt.Errorf("unknown response %T: %s", v, v)
	case *ocpb.SubscribeResponse_Heartbeat:
		log.V(1).Info("hearbeat from server: %v", v)
		return nil
	case *ocpb.SubscribeResponse_SyncResponse:
		c.handler(client.Sync{})
		if c.query.Type == client.Poll || c.query.Type == client.Once {
			return client.ErrStopReading
		}
	case *ocpb.SubscribeResponse_Update:
		n := v.Update
		var p []string
		if n.Prefix != nil {
			p = append(p, n.Prefix.Element...)
		}
		ts := time.Unix(0, n.Timestamp)
		for _, u := range n.Update {
			if u.Path == nil {
				return fmt.Errorf("invalid nil path in update: %v", u)
			}
			u, err := noti(append(p, u.Path.Element...), ts, u.Value)
			if err != nil {
				return err
			}
			c.handler(u)
		}
		for _, d := range n.Delete {
			u, err := noti(append(p, d.Element...), ts, nil)
			if err != nil {
				return err
			}
			c.handler(u)
		}
	}
	return nil
}

func getType(t client.Type) ocpb.SubscriptionList_Mode {
	switch t {
	case client.Once:
		return ocpb.SubscriptionList_ONCE
	case client.Stream:
		return ocpb.SubscriptionList_STREAM
	case client.Poll:
		return ocpb.SubscriptionList_POLL
	}
	return ocpb.SubscriptionList_ONCE
}

func subscribe(q client.Query) *ocpb.SubscribeRequest {
	s := &ocpb.SubscribeRequest_Subscribe{
		Subscribe: &ocpb.SubscriptionList{
			Mode: getType(q.Type),
		},
	}
	for _, qq := range q.Queries {
		s.Subscribe.Subscription = append(s.Subscribe.Subscription, &ocpb.Subscription{Path: &ocpb.Path{Element: qq}})
	}
	return &ocpb.SubscribeRequest{Request: s}
}

func noti(p client.Path, ts time.Time, v *ocpb.Value) (client.Notification, error) {
	if v == nil {
		return client.Delete{Path: p, TS: ts}, nil
	}
	switch v.Type {
	case ocpb.Type_BYTES:
		return client.Update{Path: p, TS: ts, Val: v.Value}, nil
	case ocpb.Type_JSON:
		var val interface{}
		if err := json.Unmarshal(v.Value, &val); err != nil {
			return nil, fmt.Errorf("json.Unmarshal(%q, val): %v", v, err)
		}
		return client.Update{Path: p, TS: ts, Val: val}, nil
	default:
		return nil, fmt.Errorf("Unsupported value type: %v", v.Type)
	}
}

func init() {
	client.Register(Type, New)
}
