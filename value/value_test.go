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
	"github.com/golang/protobuf/ptypes"
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

func TestEqual(t *testing.T) {
	anyVal, err := ptypes.MarshalAny(&pb.TypedValue{Value: &pb.TypedValue_StringVal{"any val"}})
	if err != nil {
		t.Errorf("MarshalAny: %v", err)
	}
	for _, test := range []struct {
		name string
		a, b *pb.TypedValue
		want bool
	}{
		// Equality is true.
		{
			name: "String equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}},
			want: true,
		}, {
			name: "Int equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{1234}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{1234}},
			want: true,
		}, {
			name: "Uint equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_UintVal{1234}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_UintVal{1234}},
			want: true,
		}, {
			name: "Bool equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}},
			want: true,
		}, {
			name: "Bytes equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte{1, 2, 3}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte{1, 2, 3}}},
			want: true,
		}, {
			name: "Float equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_FloatVal{1234.56789}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_FloatVal{1234.56789}},
			want: true,
		}, {
			name: "Decimal equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
			want: true,
		}, {
			name: "Leaflist equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{Element: []*pb.TypedValue{{Value: &pb.TypedValue_StringVal{"one"}}, {Value: &pb.TypedValue_StringVal{"two"}}, {Value: &pb.TypedValue_StringVal{"three"}}}}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{Element: []*pb.TypedValue{{Value: &pb.TypedValue_StringVal{"one"}}, {Value: &pb.TypedValue_StringVal{"two"}}, {Value: &pb.TypedValue_StringVal{"three"}}}}}},
			want: true,
		},
		// Equality is false.
		{
			name: "String not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_StringVal{"bar"}},
		}, {
			name: "Int not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{1234}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{123456789}},
		}, {
			name: "Uint not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_UintVal{1234}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_UintVal{123456789}},
		}, {
			name: "Bool not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_BoolVal{false}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}},
		}, {
			name: "Bytes not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte{2, 3}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_BytesVal{[]byte{1, 2, 3}}},
		}, {
			name: "Float not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_FloatVal{12340.56789}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_FloatVal{1234.56789}},
		}, {
			name: "Decimal not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 11}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Leaflist not equal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{Element: []*pb.TypedValue{{Value: &pb.TypedValue_StringVal{"one"}}, {Value: &pb.TypedValue_StringVal{"two"}}, {Value: &pb.TypedValue_StringVal{"three"}}}}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{Element: []*pb.TypedValue{{Value: &pb.TypedValue_StringVal{"one"}}, {Value: &pb.TypedValue_StringVal{"two"}}, {Value: &pb.TypedValue_StringVal{"four"}}}}}},
		}, {
			name: "Types not equal - String",
			a:    &pb.TypedValue{Value: &pb.TypedValue_StringVal{"foo"}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Types not equal - Int",
			a:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{5}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Types not equal - Uint",
			a:    &pb.TypedValue{Value: &pb.TypedValue_UintVal{5}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Types not equal - Bool",
			a:    &pb.TypedValue{Value: &pb.TypedValue_BoolVal{true}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Types not equal - Float",
			a:    &pb.TypedValue{Value: &pb.TypedValue_FloatVal{5.25}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
		}, {
			name: "Types not equal - Decimal",
			a:    &pb.TypedValue{Value: &pb.TypedValue_DecimalVal{&pb.Decimal64{Digits: 1234, Precision: 10}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{5}},
		}, {
			name: "Types not equal - Leaflist",
			a:    &pb.TypedValue{Value: &pb.TypedValue_LeaflistVal{&pb.ScalarArray{Element: []*pb.TypedValue{{Value: &pb.TypedValue_StringVal{"one"}}, {Value: &pb.TypedValue_StringVal{"two"}}, {Value: &pb.TypedValue_StringVal{"three"}}}}}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_IntVal{5}},
		},
		// Equality is not checked, expect false.
		{
			name: "Empty values not compared",
			a:    &pb.TypedValue{},
			b:    &pb.TypedValue{},
		}, {
			name: "Proto values not compared",
			a:    &pb.TypedValue{Value: &pb.TypedValue_AnyVal{anyVal}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_AnyVal{anyVal}},
		}, {
			name: "JSON values not compared",
			a:    &pb.TypedValue{Value: &pb.TypedValue_JsonVal{[]byte("5")}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_JsonVal{[]byte("5")}},
		}, {
			name: "JSON IETF values not compared",
			a:    &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{[]byte("10.5")}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{[]byte("10.5")}},
		}, {
			name: "ASCII values not compared",
			a:    &pb.TypedValue{Value: &pb.TypedValue_AsciiVal{"foo"}},
			b:    &pb.TypedValue{Value: &pb.TypedValue_AsciiVal{"foo"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Equal(test.a, test.b)
			if got != test.want {
				t.Errorf("got: %t, want: %t", got, test.want)
			}
		})
	}
}
