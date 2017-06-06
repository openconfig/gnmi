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

// Package client implements a gNMI client.
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
	"github.com/openconfig/gnmi/value"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// Type defines the name resolution for this client type.
const Type = "gnmi"

// Client handles execution of the query and caching of its results.
type Client struct {
	conn      *grpc.ClientConn
	client    gpb.GNMIClient
	sub       gpb.GNMI_SubscribeClient
	request   *gpb.SubscribeRequest
	query     client.Query
	recv      client.ProtoHandler
	handler   client.NotificationHandler
	connected bool
}

// New returns a new initialized client. The client implementation will have
// made a connection to the backend and has sent the initial query q.
func New(ctx context.Context, q client.Query) (client.Impl, error) {
	err := q.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid query %+v: %v", q, err)
	}
	if len(q.Addrs) != 1 {
		return nil, fmt.Errorf("q.Addrs must only contain one entry: %v", q.Addrs)
	}
	opts := []grpc.DialOption{
		grpc.WithTimeout(q.Timeout),
		grpc.WithBlock(),
	}
	if q.TLS != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(q.TLS)))
	}
	if q.Credentials != nil {
		pc := newPassCred(q.Credentials.Username, q.Credentials.Password, true)
		opts = append(opts, grpc.WithPerRPCCredentials(pc))
	}
	conn, err := grpc.DialContext(ctx, q.Addrs[0], opts...)
	if err != nil {
		return nil, fmt.Errorf("Dialer(%s, %v): %v", q.Addrs[0], q.Timeout, err)
	}
	return NewFromConn(ctx, conn, q)
}

// NewFromConn creates and returns the client based on the provided transport.
func NewFromConn(ctx context.Context, conn *grpc.ClientConn, q client.Query) (*Client, error) {
	cl := gpb.NewGNMIClient(conn)
	sub, err := cl.Subscribe(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gpb.NewGNMIClient(%v) failed to create client: %v", q, err)
	}
	qq := subscribe(q)
	if err := sub.Send(qq); err != nil {
		conn.Close()
		return nil, fmt.Errorf("client.Send(%+v): %v", qq, err)
	}
	c := &Client{
		conn:    conn,
		client:  cl,
		sub:     sub,
		query:   q,
		request: qq,
	}
	if q.ProtoHandler == nil {
		c.recv = c.defaultRecv
		c.handler = q.NotificationHandler
	} else {
		c.recv = q.ProtoHandler
	}
	return c, nil
}

// Poll will send a single gNMI poll request to the server.
func (c *Client) Poll() error {
	if err := c.sub.Send(&gpb.SubscribeRequest{Request: &gpb.SubscribeRequest_Poll{Poll: &gpb.Poll{}}}); err != nil {
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

// Recv will recieve a single message from the server and process it based on
// the provided handlers (Proto or Notification).
func (c *Client) Recv() error {
	n, err := c.sub.Recv()
	if err != nil {
		return err
	}
	return c.recv(n)
}

// defaultRecv is the default implementation of recv provided by the client.
// This function will be replaced by the ProtoHandler member of the Query
// struct passed to New(), if it is set.
func (c *Client) defaultRecv(msg proto.Message) error {
	if !c.connected {
		c.handler(client.Connected{})
		c.connected = true
	}

	resp, ok := msg.(*gpb.SubscribeResponse)
	if !ok {
		return fmt.Errorf("failed to type assert message %#v", msg)
	}
	log.V(1).Info(resp)
	switch v := resp.Response.(type) {
	default:
		return fmt.Errorf("unknown response %T: %s", v, v)
	case *gpb.SubscribeResponse_Error:
		return fmt.Errorf("error in response: %s", v)
	case *gpb.SubscribeResponse_SyncResponse:
		c.handler(client.Sync{})
		if c.query.Type == client.Poll || c.query.Type == client.Once {
			return client.ErrStopReading
		}
	case *gpb.SubscribeResponse_Update:
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
			u, err := noti(append(p, u.Path.Element...), ts, u)
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

func convertSetRequest(sr client.SetRequest) (*gpb.SetRequest, error) {
	req := &gpb.SetRequest{}
	for _, d := range sr.Delete {
		p := gpb.Path{Element: d}
		req.Delete = append(req.Delete, &p)
	}

	genUpdate := func(v client.Leaf) (*gpb.Update, error) {
		buf, err := json.Marshal(v.Val)
		if err != nil {
			return nil, err
		}
		return &gpb.Update{
			Path: &gpb.Path{Element: v.Path},
			Val:  &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{buf}},
			// Value is deprecated, remove it at some point.
			Value: &gpb.Value{Type: gpb.Encoding_JSON, Value: buf},
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

func convertSetResponse(sr *gpb.SetResponse) (client.SetResponse, error) {
	resp := client.SetResponse{
		TS: time.Unix(0, sr.GetTimestamp()),
	}
	var errs []string
	for _, r := range sr.GetResponse() {
		if r.Message != nil {
			errs = append(errs, r.GetMessage().String())
		}
	}
	if len(errs) > 0 {
		return resp, errors.New(strings.Join(errs, "; "))
	}

	return resp, nil
}

func getType(t client.Type) gpb.SubscriptionList_Mode {
	switch t {
	case client.Once:
		return gpb.SubscriptionList_ONCE
	case client.Stream:
		return gpb.SubscriptionList_STREAM
	case client.Poll:
		return gpb.SubscriptionList_POLL
	}
	return gpb.SubscriptionList_ONCE
}

func subscribe(q client.Query) *gpb.SubscribeRequest {
	s := &gpb.SubscribeRequest_Subscribe{
		Subscribe: &gpb.SubscriptionList{
			Mode: getType(q.Type),
		},
	}
	for _, qq := range q.Queries {
		s.Subscribe.Subscription = append(s.Subscribe.Subscription, &gpb.Subscription{Path: &gpb.Path{Element: qq}})
	}
	return &gpb.SubscribeRequest{Request: s}
}

func noti(p client.Path, ts time.Time, u *gpb.Update) (client.Notification, error) {
	if u == nil {
		return client.Delete{Path: p, TS: ts}, nil
	}
	if u.Val != nil {
		val, err := value.ToScalar(u.Val)
		if err != nil {
			return nil, err
		}
		return client.Update{Path: p, TS: ts, Val: val}, nil
	}
	switch v := u.Value; v.Type {
	case gpb.Encoding_BYTES:
		return client.Update{Path: p, TS: ts, Val: v.Value}, nil
	case gpb.Encoding_JSON, gpb.Encoding_JSON_IETF:
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
