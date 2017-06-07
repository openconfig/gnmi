// Package value provides utility functions for working with gNMI TypedValue
// messages.
package value

import (
	"fmt"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// FromScalar will convert common scalar types to their TypedValue equivalent.
// It will return an error if the type cannot be mapped to a scalar value.
func FromScalar(i interface{}) (*pb.TypedValue, error) {
	tv := &pb.TypedValue{}
	switch v := i.(type) {
	case string:
		tv.Value = &pb.TypedValue_StringVal{v}
	case int:
		tv.Value = &pb.TypedValue_IntVal{int64(v)}
	case int8:
		tv.Value = &pb.TypedValue_IntVal{int64(v)}
	case int16:
		tv.Value = &pb.TypedValue_IntVal{int64(v)}
	case int32:
		tv.Value = &pb.TypedValue_IntVal{int64(v)}
	case int64:
		tv.Value = &pb.TypedValue_IntVal{v}
	case uint:
		tv.Value = &pb.TypedValue_UintVal{uint64(v)}
	case uint8:
		tv.Value = &pb.TypedValue_UintVal{uint64(v)}
	case uint16:
		tv.Value = &pb.TypedValue_UintVal{uint64(v)}
	case uint32:
		tv.Value = &pb.TypedValue_UintVal{uint64(v)}
	case uint64:
		tv.Value = &pb.TypedValue_UintVal{v}
	case float32:
		tv.Value = &pb.TypedValue_FloatVal{v}
	case float64:
		tv.Value = &pb.TypedValue_FloatVal{float32(v)}
	case bool:
		tv.Value = &pb.TypedValue_BoolVal{v}
	default:
		return nil, fmt.Errorf("non-scalar type %+v", i)
	}
	return tv, nil
}

// ToScalar will convert TypedValue scalar types to a Go native type. It will
// return an error if the TypedValue does not contain a scalar type.
func ToScalar(tv *pb.TypedValue) (interface{}, error) {
	var i interface{}
	switch tv.Value.(type) {
	case *pb.TypedValue_StringVal:
		i = tv.GetStringVal()
	case *pb.TypedValue_IntVal:
		i = tv.GetIntVal()
	case *pb.TypedValue_UintVal:
		i = tv.GetUintVal()
	case *pb.TypedValue_BoolVal:
		i = tv.GetBoolVal()
	case *pb.TypedValue_FloatVal:
		i = tv.GetFloatVal()
	default:
		return nil, fmt.Errorf("non-scalar type %+v", tv.Value)
	}
	return i, nil
}
