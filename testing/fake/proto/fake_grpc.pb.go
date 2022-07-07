// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.2
// source: github.com/openconfig/gnmi/testing/fake/proto/fake.proto

package gnmi_fake

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AgentManagerClient is the client API for AgentManager service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AgentManagerClient interface {
	// Add adds an agent to the server.
	Add(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error)
	// Remove removes an agent from the server.
	Remove(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error)
	// Status returns the current status of an agent on the server.
	Status(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error)
}

type agentManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentManagerClient(cc grpc.ClientConnInterface) AgentManagerClient {
	return &agentManagerClient{cc}
}

func (c *agentManagerClient) Add(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error) {
	out := new(Config)
	err := c.cc.Invoke(ctx, "/gnmi.fake.AgentManager/Add", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentManagerClient) Remove(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error) {
	out := new(Config)
	err := c.cc.Invoke(ctx, "/gnmi.fake.AgentManager/Remove", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentManagerClient) Status(ctx context.Context, in *Config, opts ...grpc.CallOption) (*Config, error) {
	out := new(Config)
	err := c.cc.Invoke(ctx, "/gnmi.fake.AgentManager/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AgentManagerServer is the server API for AgentManager service.
// All implementations should embed UnimplementedAgentManagerServer
// for forward compatibility
type AgentManagerServer interface {
	// Add adds an agent to the server.
	Add(context.Context, *Config) (*Config, error)
	// Remove removes an agent from the server.
	Remove(context.Context, *Config) (*Config, error)
	// Status returns the current status of an agent on the server.
	Status(context.Context, *Config) (*Config, error)
}

// UnimplementedAgentManagerServer should be embedded to have forward compatible implementations.
type UnimplementedAgentManagerServer struct {
}

func (UnimplementedAgentManagerServer) Add(context.Context, *Config) (*Config, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Add not implemented")
}
func (UnimplementedAgentManagerServer) Remove(context.Context, *Config) (*Config, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Remove not implemented")
}
func (UnimplementedAgentManagerServer) Status(context.Context, *Config) (*Config, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}

// UnsafeAgentManagerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentManagerServer will
// result in compilation errors.
type UnsafeAgentManagerServer interface {
	mustEmbedUnimplementedAgentManagerServer()
}

func RegisterAgentManagerServer(s grpc.ServiceRegistrar, srv AgentManagerServer) {
	s.RegisterService(&AgentManager_ServiceDesc, srv)
}

func _AgentManager_Add_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Config)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentManagerServer).Add(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnmi.fake.AgentManager/Add",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentManagerServer).Add(ctx, req.(*Config))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentManager_Remove_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Config)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentManagerServer).Remove(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnmi.fake.AgentManager/Remove",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentManagerServer).Remove(ctx, req.(*Config))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentManager_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Config)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentManagerServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnmi.fake.AgentManager/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentManagerServer).Status(ctx, req.(*Config))
	}
	return interceptor(ctx, in, info, handler)
}

// AgentManager_ServiceDesc is the grpc.ServiceDesc for AgentManager service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AgentManager_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gnmi.fake.AgentManager",
	HandlerType: (*AgentManagerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Add",
			Handler:    _AgentManager_Add_Handler,
		},
		{
			MethodName: "Remove",
			Handler:    _AgentManager_Remove_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _AgentManager_Status_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/openconfig/gnmi/testing/fake/proto/fake.proto",
}
