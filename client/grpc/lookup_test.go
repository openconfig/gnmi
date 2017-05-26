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

package grpc

import (
	"net"
	"testing"

	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestLookup(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	defer srv.Stop()

	gpb.RegisterGNMIServer(srv, unimplementedGNMIServer{})
	reflection.Register(srv)

	go srv.Serve(l)

	c, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	t.Run("valid service", func(t *testing.T) {
		ok, err := Lookup(ctx, c, "gnmi.gNMI")
		if err != nil {
			log.Error(err)
		}
		if !ok {
			log.Error("got false, want true")
		}
	})
	t.Run("unknown service", func(t *testing.T) {
		ok, err := Lookup(ctx, c, "unknown.Unknown")
		if err != nil {
			log.Error(err)
		}
		if ok {
			log.Error("got true, want false")
		}
	})

}

type unimplementedGNMIServer struct{}

func (unimplementedGNMIServer) Capabilities(context.Context, *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	return nil, nil
}
func (unimplementedGNMIServer) Get(context.Context, *gpb.GetRequest) (*gpb.GetResponse, error) {
	return nil, nil
}
func (unimplementedGNMIServer) Set(context.Context, *gpb.SetRequest) (*gpb.SetResponse, error) {
	return nil, nil
}
func (unimplementedGNMIServer) Subscribe(gpb.GNMI_SubscribeServer) error { return nil }
