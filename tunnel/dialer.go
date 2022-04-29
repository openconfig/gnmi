/*
Copyright 2021 Google Inc.

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

// Package dialer implements the dialer library for tunnel connection.
package dialer

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"github.com/openconfig/grpctunnel/tunnel"

	tunnelpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// Dialer performs dialing at tunnel clients connections.
type Dialer struct {
	s *tunnel.Server
}

var serverConn = tunnel.ServerConn
var grpcDialContext = grpc.DialContext

// DialContext implements connection.Dial. It dials at the tunnel target and
// returns an error if the connection is not established.
func (d *Dialer) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	withContextDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return serverConn(ctx, d.s, &tunnel.Target{ID: target, Type: tunnelpb.TargetType_GNMI_GNOI.String()})
	})
	opts = append(opts, withContextDialer)
	return grpcDialContext(ctx, target, opts...)
}

// NewDialer creates a new dialer with an existing tunnel server.
func NewDialer(s *tunnel.Server) (*Dialer, error) {
	if s == nil {
		return nil, fmt.Errorf("tunnel server is nil")
	}

	return &Dialer{s: s}, nil
}
