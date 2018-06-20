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

package target

import (
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/target"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		desc   string
		valid  bool
		config *pb.Configuration
	}{{
		desc:  "valid configuration with multiple targets",
		valid: true,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"target1": &pb.Target{
					Address: "192.168.0.10:10162",
					Request: "interfaces",
				},
				"target2": &pb.Target{
					Address: "192.168.0.11:10162",
					Request: "interfaces",
				},
			},
		},
	}, {
		desc:   "empty configuration",
		valid:  true,
		config: &pb.Configuration{},
	}, {
		desc:  "request but no targets",
		valid: true,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
		},
	}, {
		desc:  "target with empty name",
		valid: false,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"": &pb.Target{
					Address: "192.168.0.10:10162",
					Request: "interfaces",
				},
				"target2": &pb.Target{
					Address: "192.168.0.11:10162",
					Request: "interfaces",
				},
			},
		},
	}, {
		desc:  "missing target configuration",
		valid: false,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"target1": nil,
				"target2": &pb.Target{
					Address: "192.168.0.11:10162",
					Request: "interfaces",
				},
			},
		},
	}, {
		desc:  "missing target address",
		valid: false,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"target1": &pb.Target{
					Request: "interfaces",
				},
			},
		},
	}, {
		desc:  "missing target request",
		valid: false,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"target1": &pb.Target{
					Address: "192.168.0.10:10162",
				},
			},
		},
	}, {
		desc:  "invalid target request",
		valid: false,
		config: &pb.Configuration{
			Request: map[string]*gpb.SubscribeRequest{
				"interfaces": &gpb.SubscribeRequest{},
			},
			Target: map[string]*pb.Target{
				"target1": &pb.Target{
					Address: "192.168.0.10:10162",
					Request: "qos",
				},
			},
		},
	}}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := Validate(test.config)
			switch {
			case test.valid && err != nil:
				t.Errorf("got error %v, want nil for configuration: %s", err, test.config.String())
			case !test.valid && err == nil:
				t.Errorf("got nit, want error for configuration: %s", test.config.String())
			}
		})
	}
}
