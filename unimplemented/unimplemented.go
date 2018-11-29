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

// Package unimplemented provides a convenience type to stub out unimplemented
// gNMI RPCs.
package unimplemented

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Server is a type that can be embedded anonymously in a gNMI server to stub
// out all RPCs that are not implemented with a proper return code.
type Server struct{}

// Capabilities satisfies the gNMI service definition.
func (*Server) Capabilities(context.Context, *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Get satisfies the gNMI service definition.
func (*Server) Get(context.Context, *pb.GetRequest) (*pb.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Set satisfies the gNMI service definition.
func (*Server) Set(context.Context, *pb.SetRequest) (*pb.SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Subscribe satisfies the gNMI service definition.
func (s *Server) Subscribe(stream pb.GNMI_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Client is a type that can be embedded anonymously in a gNMI client to stub
// out all RPCs that are not implemented with a proper return code.
type Client struct{}

// Capabilities satisfies the gNMI client definition.
func (*Client) Capabilities(ctx context.Context, in *pb.CapabilityRequest, opts ...grpc.CallOption) (*pb.CapabilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Get satisfies the gNMI client definition.
func (*Client) Get(ctx context.Context, in *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Set satisfies the gNMI client definition.
func (*Client) Set(ctx context.Context, in *pb.SetRequest, opts ...grpc.CallOption) (*pb.SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Subscribe satisfies the gNMI client definition.
func (*Client) Subscribe(ctx context.Context, opts ...grpc.CallOption) (pb.GNMI_SubscribeClient, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}
