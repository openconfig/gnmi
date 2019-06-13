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
	"reflect"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/target"
)

type record struct {
	adds    []Update
	updates []Update
	deletes []string
}

func (r *record) assertLast(t *testing.T, desc string, wantAdd, wantUpdate []Update, wantDelete []string) {
	sort.Slice(r.adds, func(i int, j int) bool { return r.adds[i].Name < r.adds[j].Name })
	sort.Slice(r.updates, func(i int, j int) bool { return r.updates[i].Name < r.updates[j].Name })
	sort.Strings(r.deletes)
	switch {
	case !reflect.DeepEqual(r.adds, wantAdd):
		t.Errorf("%v: Mismatched adds: got %v, want %+v", desc, r.adds, wantAdd)
	case !reflect.DeepEqual(r.updates, wantUpdate):
		t.Errorf("%v: Mismatched updates: got %v, want %+v", desc, r.updates, wantUpdate)
	case !reflect.DeepEqual(r.deletes, wantDelete):
		t.Errorf("%v: Mismatched deletes: got %v, want %v", desc, r.deletes, wantDelete)
	}
	r.adds = nil
	r.updates = nil
	r.deletes = nil
}

func TestLoad(t *testing.T) {
	for _, tt := range []struct {
		desc       string
		initial    *pb.Configuration
		toLoad     *pb.Configuration
		wantAdd    []string
		wantUpdate []string
		wantDelete []string
		err        bool
	}{
		{
			desc:   "nil Config",
			toLoad: nil,
			err:    true,
		}, {
			desc: "bad config",
			toLoad: &pb.Configuration{
				Target: map[string]*pb.Target{
					"dev1": {},
				},
				Revision: 1,
			},
			err: true,
		}, {
			desc:    "old Config",
			initial: &pb.Configuration{},
			toLoad: &pb.Configuration{
				Revision: -1,
			},
			err: true,
		}, {
			desc: "add targets",
			toLoad: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{
						Request: &gpb.SubscribeRequest_Subscribe{},
					},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
				},
				Revision: 1,
			},
			wantAdd: []string{"dev1", "dev2"},
		}, {
			desc: "modify targets",
			initial: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev3": {
						Request:   "sub1",
						Addresses: []string{"33.333.333.33:33333"},
					},
				},
				Revision: 0,
			},
			toLoad: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111", "12.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub1",
						Addresses: []string{"22.222.222.22:22222"},
					},
				},
				Revision: 1,
			},
			wantAdd:    []string{"dev2"},
			wantUpdate: []string{"dev1"},
			wantDelete: []string{"dev3"},
		}, {
			desc: "modify subscription",
			initial: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{},
					"sub3": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub1",
						Addresses: []string{"22.222.222.22:22222"},
					},
					"dev3": {
						Request:   "sub3",
						Addresses: []string{"33.333.333.33:33333"},
					},
				},
				Revision: 0,
			},
			toLoad: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{
						Request: &gpb.SubscribeRequest_Subscribe{
							&gpb.SubscriptionList{
								Subscription: []*gpb.Subscription{
									{
										Path: &gpb.Path{
											Elem: []*gpb.PathElem{{Name: "path2"}},
										},
									},
								},
							},
						},
					},
					"sub3": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub1",
						Addresses: []string{"22.222.222.22:22222"},
					},
					"dev3": {
						Request:   "sub3",
						Addresses: []string{"33.333.333.33:33333"},
					},
				},
				Revision: 1,
			},
			wantUpdate: []string{"dev1", "dev2"},
		}, {
			desc: "modify subscription key",
			initial: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{
						Request: &gpb.SubscribeRequest_Subscribe{
							&gpb.SubscriptionList{
								Subscription: []*gpb.Subscription{
									{
										Path: &gpb.Path{
											Elem: []*gpb.PathElem{{Name: "path2"}},
										},
									},
								},
							},
						},
					},
					"sub3": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub1",
						Addresses: []string{"22.222.222.22:22222"},
					},
					"dev3": {
						Request:   "sub3",
						Addresses: []string{"33.333.333.33:33333"},
					},
				},
				Revision: 0,
			},
			toLoad: &pb.Configuration{
				Request: map[string]*gpb.SubscribeRequest{
					"sub1": &gpb.SubscribeRequest{
						Request: &gpb.SubscribeRequest_Subscribe{
							&gpb.SubscriptionList{
								Subscription: []*gpb.Subscription{
									{
										Path: &gpb.Path{
											Elem: []*gpb.PathElem{{Name: "path2"}},
										},
									},
								},
							},
						},
					},
					"sub3": &gpb.SubscribeRequest{},
				},
				Target: map[string]*pb.Target{
					"dev1": {
						Request:   "sub1",
						Addresses: []string{"11.111.111.11:11111"},
					},
					"dev2": {
						Request:   "sub3",
						Addresses: []string{"22.222.222.22:22222"},
					},
					"dev3": {
						Request:   "sub3",
						Addresses: []string{"33.333.333.33:33333"},
					},
				},
				Revision: 1,
			},
			wantUpdate: []string{"dev2"},
		},
	} {
		r := &record{}
		h := Handler{
			Add: func(c Update) {
				r.adds = append(r.adds, Update{c.Name, c.Request, c.Target})
			},
			Update: func(c Update) {
				r.updates = append(r.updates, Update{c.Name, c.Request, c.Target})
			},
			Delete: func(s string) {
				r.deletes = append(r.deletes, s)
			},
		}
		c := NewConfig(h)
		c.configuration = tt.initial
		err := c.Load(tt.toLoad)
		switch {
		case err == nil && !tt.err:
			if diff := cmp.Diff(tt.toLoad, c.Current(), cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("%v: wrong state: (-want +got)\n%s", tt.desc, diff)
			}
			updates := func(want []string, c *pb.Configuration) []Update {
				if want == nil {
					return nil
				}
				u := []Update{}
				for _, n := range want {
					ta := c.GetTarget()[n]
					u = append(u, Update{
						Name:    n,
						Request: c.GetRequest()[ta.GetRequest()],
						Target:  ta,
					})
				}
				return u
			}
			r.assertLast(t, tt.desc, updates(tt.wantAdd, tt.toLoad), updates(tt.wantUpdate, tt.toLoad), tt.wantDelete)
		case err == nil:
			t.Errorf("%v: did not get expected error", tt.desc)
		case !tt.err:
			t.Errorf("%v: unexpected error %v", tt.desc, err)
		}
	}
}

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
					Addresses: []string{"192.168.0.10:10162"},
					Request:   "interfaces",
				},
				"target2": &pb.Target{
					Addresses: []string{"192.168.0.11:10162"},
					Request:   "interfaces",
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
					Addresses: []string{"192.168.0.10:10162"},
					Request:   "interfaces",
				},
				"target2": &pb.Target{
					Addresses: []string{"192.168.0.11:10162"},
					Request:   "interfaces",
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
					Addresses: []string{"192.168.0.11:10162"},
					Request:   "interfaces",
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
					Addresses: []string{"192.168.0.10:10162"},
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
					Addresses: []string{"192.168.0.10:10162"},
					Request:   "qos",
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
