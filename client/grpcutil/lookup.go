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

// Package grpcutil provides helper functions for working with gRPC targets.
package grpcutil

import (
	"context"

	"google.golang.org/grpc"

	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// Lookup uses ServerReflection service on conn to find a named service.
// It returns an error if the remote side doesn't support ServerReflection or
// if any other error occurs.
//
// If lookup succeeds and service is found, true is returned.
func Lookup(ctx context.Context, conn *grpc.ClientConn, service string) (bool, error) {
	c, err := rpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return false, err
	}
	defer c.CloseSend()

	if err := c.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return false, err
	}

	resp, err := c.Recv()
	if err != nil {
		return false, err
	}

	lsResp := resp.GetListServicesResponse()
	for _, s := range lsResp.GetService() {
		if s.Name == service {
			return true, nil
		}
	}

	return false, nil
}
