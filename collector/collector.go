/*
Copyright 2020 Google Inc.

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

// Package collector provides a gRPC interface to reconnect targets in a gNMI
// collector.
package collector

import (
	"context"
	"fmt"

	log "github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/openconfig/gnmi/proto/collector"
)

// Server provides an implementation of a gRPC service to allow reconnection of
// targets.
type Server struct {
	reconnect func(string) error
}

// New constructs a Server struct with a specified handler for reconnection
// requests.
func New(reconnect func(string) error) *Server {
	return &Server{reconnect: reconnect}
}

// Reconnect provides the implementation of the Collector.Reconnect RPC allowing
// one or more gNMI targets to be forcefully reconnected.
func (s *Server) Reconnect(ctx context.Context, request *pb.ReconnectRequest) (*pb.Nil, error) {
	var failed []string
	for _, target := range request.Target {
		if err := s.reconnect(target); err != nil {
			failed = append(failed, target)
			log.Warningf("Invalid reconnect request for %q", target)
		} else {
			log.Infof("Forced reconnect of %q", target)
		}
	}
	var err error
	if len(failed) > 0 {
		err = status.Error(codes.NotFound, fmt.Sprintf("invalid targets %q", failed))
	}
	return &pb.Nil{}, err
}
