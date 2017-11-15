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

package value

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

type scalarTest struct {
	intf interface{}
	msg  *pb.TypedValue
	err  bool
}

func TestFromScalar(t *testing.T) {
	tests := []scalarTest{
		{intf: "foo", msg: &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}}},
		{intf: "a longer multiword string", msg: &pb.TypedValue{Value: &pb.TypedValue_StringVal{"a longer multiword string"}}},
		{intf: 500, msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{500}}},
		{intf: int8(50), msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{50}}},
		{intf: int16(500), msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{500}}},
		{intf: int32(500), msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{500}}},
		{intf: int64(500), msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{500}}},
		{intf: uint(500), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{500}}},
		{intf: uint8(50), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{50}}},
		{intf: uint16(500), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{500}}},
		{intf: uint32(500), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{500}}},
		{intf: uint64(500), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{500}}},
		{intf: float32(3.5), msg: &pb.TypedValue{Value: &pb.TypedValue_FloatVal{3.5}}},
		{intf: true, msg: &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}}},
		{intf: false, msg: &pb.TypedValue{Value: &pb.TypedValue_BoolVal{false}}},
		{intf: float64(3.5), msg: &pb.TypedValue{Value: &pb.TypedValue_FloatVal{3.5}}},
		{intf: []byte("foo"), msg: &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte("foo")}}},
		{intf: "a non-utf-8 string \377", err: true},
		{
			intf: []string{"a", "b"},
			msg: &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{
				Element: []*pb.TypedValue{
					{Value: &pb.TypedValue_StringVal{"a"}},
					{Value: &pb.TypedValue_StringVal{"b"}},
				},
			}}},
		},
		{
			intf: []string{},
			msg:  &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{}}},
		},
		{
			intf: []interface{}{"a", "b"},
			msg: &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{
				Element: []*pb.TypedValue{
					{Value: &pb.TypedValue_StringVal{"a"}},
					{Value: &pb.TypedValue_StringVal{"b"}},
				},
			}}},
		},
		{
			intf: []interface{}{},
			msg:  &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{}}},
		},
		{
			intf: []interface{}{"a", 1},
			msg: &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{
				Element: []*pb.TypedValue{
					{Value: &pb.TypedValue_StringVal{"a"}},
					{Value: &pb.TypedValue_IntVal{1}},
				},
			}}},
		},
	}
	for _, tt := range tests {
		v, err := FromScalar(tt.intf)
		switch {
		case tt.err:
			if err == nil {
				t.Errorf("FromScalar(%v): got nil, wanted err", tt.intf)
			}
		case err != nil:
			t.Errorf("FromScalar(%v): got error %v, want %s", tt.intf, err, tt.msg)
		case !proto.Equal(v, tt.msg):
			t.Errorf("FromScalar(%v): got %s, want %s", tt.intf, v, tt.msg)
		}
	}
}

func TestToScalar(t *testing.T) {
	tests := []scalarTest{
		{intf: "foo", msg: &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}}},
		{intf: "a longer multiword string", msg: &pb.TypedValue{Value: &pb.TypedValue_StringVal{"a longer multiword string"}}},
		{intf: int64(500), msg: &pb.TypedValue{Value: &pb.TypedValue_IntVal{500}}},
		{intf: uint64(500), msg: &pb.TypedValue{Value: &pb.TypedValue_UintVal{500}}},
		{intf: float32(3.5), msg: &pb.TypedValue{Value: &pb.TypedValue_FloatVal{3.5}}},
		{intf: true, msg: &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}}},
		{intf: false, msg: &pb.TypedValue{Value: &pb.TypedValue_BoolVal{false}}},
		{
			msg: &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{
				Element: []*pb.TypedValue{
					{Value: &pb.TypedValue_StringVal{"a"}},
					{Value: &pb.TypedValue_StringVal{"b"}},
				},
			}}},
			intf: []interface{}{"a", "b"},
		},
		{
			msg:  &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{}}},
			intf: []interface{}{},
		},
		{
			msg:  &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{}},
			intf: []interface{}{},
		},
		{
			msg: &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{
				Element: []*pb.TypedValue{
					{Value: &pb.TypedValue_StringVal{"a"}},
					{Value: &pb.TypedValue_IntVal{1}},
					{Value: &pb.TypedValue_UintVal{1}},
				},
			}}},
			intf: []interface{}{"a", int64(1), uint64(1)},
		},
		{intf: []byte("foo"), msg: &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte("foo")}}},
		{
			msg: &pb.TypedValue{
				Value: &pb.TypedValue_DecimalVal{
					DecimalVal: &pb.Decimal64{
						Digits:    312,
						Precision: 1,
					},
				},
			},
			intf: float32(31.2),
		},
		{
			msg: &pb.TypedValue{
				Value: &pb.TypedValue_DecimalVal{
					DecimalVal: &pb.Decimal64{
						Digits: 5,
					},
				},
			},
			intf: float32(5),
		},
		{
			msg: &pb.TypedValue{
				Value: &pb.TypedValue_DecimalVal{
					DecimalVal: &pb.Decimal64{
						Digits:    5678,
						Precision: 18,
					},
				},
			},
			intf: float32(.000000000000005678),
		},
		{msg: &pb.TypedValue{Value: &pb.TypedValue_AnyVal{}}, err: true},
		{msg: &pb.TypedValue{Value: &pb.TypedValue_JsonVal{}}, err: true},
		{msg: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{}}, err: true},
		{msg: &pb.TypedValue{Value: &pb.TypedValue_AsciiVal{}}, err: true},
	}
	for _, tt := range tests {
		v, err := ToScalar(tt.msg)
		switch {
		case tt.err:
			if err == nil {
				t.Errorf("ToScalar(%v): got nil, wanted err", tt.msg)
			}
		case err != nil:
			t.Errorf("ToScalar(%v): got error %v, want %+v", tt.msg, err, tt.intf)
		case !reflect.DeepEqual(v, tt.intf):
			t.Errorf("ToScalar(%v): got %#v, want %#v", tt.msg, v, tt.intf)
		}
	}
}
