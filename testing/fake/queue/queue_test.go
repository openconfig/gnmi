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

package queue

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

var seed = int64(100)

func TestUpdateTimestamp(t *testing.T) {
	tests := []struct {
		desc string
		in   *fpb.Value
		want *fpb.Timestamp
		err  string
	}{{
		desc: "No timestamp",
		in:   &fpb.Value{},
		err:  "timestamp not set",
	}, {
		desc: "Negative timestamp",
		in:   &fpb.Value{Timestamp: &fpb.Timestamp{Timestamp: -1}},
		err:  "timestamp must be positive",
	}, {
		desc: "Invalid timestamp",
		in:   &fpb.Value{Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 2, DeltaMax: 1}},
		err:  "invalid delta_min/delta_max on timestamp",
	}, {
		desc: "Valid timestamp",
		in: &fpb.Value{
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1}},
		want: &fpb.Timestamp{Timestamp: 1235, DeltaMin: 1, DeltaMax: 1},
	}, {
		desc: "Using global seed",
		in: &fpb.Value{
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 10}},
		want: &fpb.Timestamp{Timestamp: 1243, DeltaMin: 1, DeltaMax: 10},
	}, {
		desc: "Using local seed",
		in: &fpb.Value{
			Seed:      10,
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 10}},
		want: &fpb.Timestamp{Timestamp: 1240, DeltaMin: 1, DeltaMax: 10}},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.in, rand.New(rand.NewSource(seed)))
			err := v.updateTimestamp()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).updateTimestamp() %v", tc.in, diff)
			}
			if diff := pretty.Compare(v.v.GetTimestamp(), tc.want); err == nil && diff != "" {
				t.Errorf("newValue(%q).updateTimestamp() %v", tc.in, diff)
			}
		})
	}
}

func TestUpdateIntValue(t *testing.T) {
	tests := []struct {
		desc  string
		value *fpb.Value
		want  *fpb.Value
		err   string
	}{{
		desc:  "Nil value",
		value: &fpb.Value{},
		err:   "invalid IntValue",
	}, {
		desc: "Invalid min/max in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 100, Maximum: 0}}}}},
		err: "invalid minimum/maximum in IntRange",
	}, {
		desc: "Invalid init value (value < min) in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        -100,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
		err: "value not in [minimum, maximum] in IntRange",
	}, {
		desc: "Invalid init value (value > max) in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        200,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
		err: "value not in [minimum, maximum] in IntRange",
	}, {
		desc: "Invalid delta_min/delta_max in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 5}}}}},
		err: "invalid delta_min/delta_max in IntRange",
	}, {
		desc: "Non-empty value, non-cumulative in range, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        65,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
	}, {
		desc: "Non-empty value, non-cumulative in range, using local seed",
		value: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 50, Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
		want: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        69,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
	}, {
		desc: "Non-empty value, cumulative in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        60,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 10}}}}},
	}, {
		desc: "Non-empty value, cumulative, maximum capped in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 51, DeltaMin: 10, DeltaMax: 10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        51,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 51, DeltaMin: 10, DeltaMax: 10}}}}},
	}, {
		desc: "Non-empty value, cumulative, minimum capped in range",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 45, Maximum: 60, DeltaMin: -10, DeltaMax: -10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 45,
				Distribution: &fpb.IntValue_Range{
					Range: &fpb.IntRange{Minimum: 45, Maximum: 60, DeltaMin: -10, DeltaMax: -10}}}}},
	}, {
		desc: "no options, random in list",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{Distribution: &fpb.IntValue_List{
				List: &fpb.IntList{Random: true}}}}},
		err: "missing options on IntValue_List",
	}, {
		desc: "four options, random in list",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{Distribution: &fpb.IntValue_List{
				List: &fpb.IntList{Options: []int64{100, 200, 300, 400}, Random: true}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 400,
				Distribution: &fpb.IntValue_List{
					List: &fpb.IntList{Options: []int64{100, 200, 300, 400}, Random: true}}}}},
	}, {
		desc: "four options, non-random in list",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{Distribution: &fpb.IntValue_List{
				List: &fpb.IntList{Options: []int64{100, 200, 300, 400}, Random: false}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 100,
				Distribution: &fpb.IntValue_List{
					List: &fpb.IntList{Options: []int64{200, 300, 400, 100}, Random: false}}}}},
	}, {
		desc: "constant",
		value: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{Value: 100}}},
		want: &fpb.Value{
			Value: &fpb.Value_IntValue{&fpb.IntValue{Value: 100}}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.value, rand.New(rand.NewSource(seed)))
			err := v.updateIntValue()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).updateIntValue() %v", tc.value, diff)
			}
			if diff := pretty.Compare(v.v, tc.want); err == nil && diff != "" {
				t.Errorf("newValue(%q).updatedIntValue() %v", tc.value, diff)
			}
		})
	}
}

func TestUpdateDoubleValue(t *testing.T) {
	tests := []struct {
		desc  string
		value *fpb.Value
		want  *fpb.Value
		err   string
	}{{
		desc:  "Nil Value",
		value: &fpb.Value{},
		err:   "invalid DoubleValue",
	}, {
		desc: "Invalid min/max in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 100, Maximum: 0}}}}},
		err: "invalid minimum/maximum on DoubleValue_Range",
	}, {
		desc: "Invalid init value (value > max) in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        200,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100}}}}},
		err: "value not in [minimum, maximum] on DoubleValue_Range",
	}, {
		desc: "Invalid delta_min/delta_max in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 5}}}}},
		err: "invalid delta_min/delta_max on DoubleValue_Range",
	}, {
		desc: "Non-empty value, non-cumulative in range, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        81.65026937796166,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100}}}}},
	}, {
		desc: "Non-empty value, non-cumulative in range, using local seed",
		value: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100}}}}},
		want: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 56.60920659323543,
				Distribution: &fpb.DoubleValue_Range{
					Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100}}}}},
	}, {
		desc: "Non-empty value, cumulative in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 60,
				Distribution: &fpb.DoubleValue_Range{
					Range: &fpb.DoubleRange{Minimum: 0, Maximum: 100, DeltaMin: 10, DeltaMax: 10}}}}},
	}, {
		desc: "Non-empty value, cumulative, maximum capped in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 0, Maximum: 51, DeltaMin: 10, DeltaMax: 10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 51,
				Distribution: &fpb.DoubleValue_Range{
					Range: &fpb.DoubleRange{Minimum: 0, Maximum: 51, DeltaMin: 10, DeltaMax: 10}}}}},
	}, {
		desc: "Non-empty value, cumulative, minimum capped in range",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value:        50,
				Distribution: &fpb.DoubleValue_Range{Range: &fpb.DoubleRange{Minimum: 45, Maximum: 60, DeltaMin: -10, DeltaMax: -10}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 45,
				Distribution: &fpb.DoubleValue_Range{
					Range: &fpb.DoubleRange{Minimum: 45, Maximum: 60, DeltaMin: -10, DeltaMax: -10}}}}},
	}, {
		desc: "no options, random in list",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Distribution: &fpb.DoubleValue_List{
				List: &fpb.DoubleList{Random: true}}}}},
		err: "missing options on DoubleValue_List",
	}, {
		desc: "four options, random in list, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Distribution: &fpb.DoubleValue_List{
				List: &fpb.DoubleList{Options: []float64{100, 200, 300, 400}, Random: true}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 400,
				Distribution: &fpb.DoubleValue_List{
					List: &fpb.DoubleList{Options: []float64{100, 200, 300, 400}, Random: true}}}}},
	}, {
		desc: "four options, non-random in list, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Distribution: &fpb.DoubleValue_List{
				List: &fpb.DoubleList{Options: []float64{100, 200, 300, 400}, Random: false}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{
				Value: 100,
				Distribution: &fpb.DoubleValue_List{
					List: &fpb.DoubleList{Options: []float64{200, 300, 400, 100}, Random: false}}}}},
	}, {
		desc: "constant",
		value: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: 100}}},
		want: &fpb.Value{
			Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: 100}}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.value, rand.New(rand.NewSource(seed)))
			err := v.updateDoubleValue()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).updateDoubleValue() %v", tc.value, diff)
			}
			if err != nil {
				return
			}
			if diff := pretty.Compare(v.v, tc.want); diff != "" {
				t.Errorf("newValue(%q).updatedDoubleValue() %v", tc.value, diff)
			}
		})
	}
}

func TestUpdateBoolValue(t *testing.T) {
	tests := []struct {
		desc  string
		value *fpb.Value
		want  *fpb.Value
		err   string
	}{{
		desc:  "Nil Value",
		value: &fpb.Value{},
		err:   "invalid BoolValue",
	}, {
		desc: "no options, random in list",
		value: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Distribution: &fpb.BoolValue_List{
				List: &fpb.BoolList{Random: true}}}}},
		err: "missing options on BoolValue_List",
	}, {
		desc: "Four options, random in list, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Distribution: &fpb.BoolValue_List{
				List: &fpb.BoolList{Options: []bool{true, true, false, false}, Random: true}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{
				Value: false,
				Distribution: &fpb.BoolValue_List{
					List: &fpb.BoolList{
						Options: []bool{true, true, false, false}, Random: true}}}}},
	}, {
		desc: "Four options, random in list, using local seed",
		value: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Distribution: &fpb.BoolValue_List{
				List: &fpb.BoolList{Random: true, Options: []bool{true, true, false, false}}}}}},
		want: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{
				Value: false,
				Distribution: &fpb.BoolValue_List{
					List: &fpb.BoolList{Random: true, Options: []bool{true, true, false, false}}}}}},
	}, {
		desc: "Four options, non-random in list",
		value: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Distribution: &fpb.BoolValue_List{
				List: &fpb.BoolList{Random: false, Options: []bool{true, true, false, false}}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{
				Value: true,
				Distribution: &fpb.BoolValue_List{
					List: &fpb.BoolList{Random: false, Options: []bool{true, false, false, true}}}}}},
	}, {
		desc: "constant",
		value: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Value: true}}},
		want: &fpb.Value{
			Value: &fpb.Value_BoolValue{&fpb.BoolValue{Value: true}}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.value, rand.New(rand.NewSource(seed)))
			err := v.updateBoolValue()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).updateBoolValue() %v", tc.value, diff)
			}
			if diff := pretty.Compare(v.v, tc.want); err == nil && diff != "" {
				t.Errorf("newValue(%q).updatedBoolValue() %v", tc.value, diff)
			}
		})
	}
}

func TestUpdateStringValue(t *testing.T) {
	tests := []struct {
		desc  string
		value *fpb.Value
		want  *fpb.Value
		err   string
	}{{
		desc:  "Nil Value",
		value: &fpb.Value{},
		err:   "invalid StringValue",
	}, {
		desc: "no options, random in list",
		value: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{Distribution: &fpb.StringValue_List{
				List: &fpb.StringList{Random: true}}}}},
		err: "missing options on StringValue_List",
	}, {
		desc: "Four options, random in list, using global seed",
		value: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{Distribution: &fpb.StringValue_List{
				List: &fpb.StringList{Options: []string{"a", "b", "c", "d"}, Random: true}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{
				Value: "d",
				Distribution: &fpb.StringValue_List{
					List: &fpb.StringList{
						Options: []string{"a", "b", "c", "d"}, Random: true}}}}},
	}, {
		desc: "Four options, random in list, using local seed",
		value: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_StringValue{&fpb.StringValue{Distribution: &fpb.StringValue_List{
				List: &fpb.StringList{Random: true, Options: []string{"a", "b", "c", "d"}}}}}},
		want: &fpb.Value{
			Seed: 10,
			Value: &fpb.Value_StringValue{&fpb.StringValue{
				Value: "c",
				Distribution: &fpb.StringValue_List{
					List: &fpb.StringList{Random: true, Options: []string{"a", "b", "c", "d"}}}}}},
	}, {
		desc: "Four options, non-random in list",
		value: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{Distribution: &fpb.StringValue_List{
				List: &fpb.StringList{Random: false, Options: []string{"a", "b", "c", "d"}}}}}},
		want: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{
				Value: "a",
				Distribution: &fpb.StringValue_List{
					List: &fpb.StringList{Random: false, Options: []string{"b", "c", "d", "a"}}}}}},
	}, {
		desc: "constant",
		value: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{Value: "a"}}},
		want: &fpb.Value{
			Value: &fpb.Value_StringValue{&fpb.StringValue{Value: "a"}}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.value, rand.New(rand.NewSource(seed)))
			err := v.updateStringValue()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).updateStringValue() %v", tc.value, diff)
			}
			if diff := pretty.Compare(v.v, tc.want); err == nil && diff != "" {
				t.Errorf("newValue(%q).updatedStringValue() %v", tc.value, diff)
			}
		})
	}
}

func TestNextValue(t *testing.T) {
	tests := []struct {
		desc string
		in   *fpb.Value
		want *fpb.Value
		err  string
	}{{
		desc: "Empty value",
		in:   &fpb.Value{},
		want: &fpb.Value{},
		err:  "timestamp not set",
	}, {
		desc: "Just timestamp",
		in:   &fpb.Value{Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1}},
		want: &fpb.Value{Timestamp: &fpb.Timestamp{Timestamp: 1235, DeltaMin: 1, DeltaMax: 1}},
		err:  "value type not found",
	}, {
		desc: "Indefinite updates",
		in: &fpb.Value{
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1},
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
		want: &fpb.Value{
			Timestamp: &fpb.Timestamp{Timestamp: 1235, DeltaMin: 1, DeltaMax: 1},
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        80,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{Minimum: 0, Maximum: 100}}}}},
	}, {
		desc: "Repeat",
		in: &fpb.Value{
			Repeat:    5,
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1},
			Value:     &fpb.Value_IntValue{&fpb.IntValue{Distribution: &fpb.IntValue_List{List: &fpb.IntList{Options: []int64{10, 20, 30}, Random: false}}}}},
		want: &fpb.Value{
			Repeat:    4,
			Timestamp: &fpb.Timestamp{Timestamp: 1235, DeltaMin: 1, DeltaMax: 1},
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value:        10,
				Distribution: &fpb.IntValue_List{List: &fpb.IntList{Options: []int64{20, 30, 10}, Random: false}}}}},
	}, {
		desc: "Repeat with constant double value",
		in: &fpb.Value{
			Repeat:    5,
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1},
			Value:     &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: 50.1}},
		},
		want: &fpb.Value{
			Repeat:    4,
			Timestamp: &fpb.Timestamp{Timestamp: 1235, DeltaMin: 1, DeltaMax: 1},
			Value:     &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: 50.1}},
		},
	}, {
		desc: "Last repeat",
		in: &fpb.Value{
			Repeat:    1,
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1},
			Value:     &fpb.Value_IntValue{&fpb.IntValue{Value: 50}},
		},
	}, {
		desc: "Last repeat with constant string value",
		in: &fpb.Value{
			Repeat:    1,
			Timestamp: &fpb.Timestamp{Timestamp: 1234, DeltaMin: 1, DeltaMax: 1},
			Value:     &fpb.Value_StringValue{&fpb.StringValue{Value: "a"}},
		},
	}, {
		desc: "String value",
		in: &fpb.Value{
			Repeat:    2,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_StringValue{&fpb.StringValue{Value: "a"}},
		},
		want: &fpb.Value{
			Repeat:    1,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_StringValue{&fpb.StringValue{Value: "a"}},
		},
	}, {
		desc: "Sync value",
		in: &fpb.Value{
			Repeat:    2,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_Sync{uint64(1)},
		},
		want: &fpb.Value{
			Repeat:    1,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_Sync{uint64(1)},
		},
	}, {
		desc: "Delete value",
		in: &fpb.Value{
			Repeat:    2,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_Delete{&fpb.DeleteValue{}},
		},
		want: &fpb.Value{
			Repeat:    1,
			Timestamp: &fpb.Timestamp{Timestamp: 1234},
			Value:     &fpb.Value_Delete{&fpb.DeleteValue{}},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			v := newValue(tc.in, rand.New(rand.NewSource(seed)))
			err := v.nextValue()
			if diff := errdiff.Substring(err, tc.err); diff != "" {
				t.Errorf("newValue(%q).nextValue() %v", tc.in, diff)
			}
			if diff := pretty.Compare(v.v, tc.want); err == nil && diff != "" {
				t.Errorf("value of newValue(%q).nextValue() %v", tc.in, diff)
			}
		})
	}
}

func TestEmptyQueue(t *testing.T) {
	q := New(false, seed, nil)
	u, err := q.Next()
	if err != nil {
		t.Fatalf("New(false, nil).Next() unexpected error: got %q, want nil", err)
	}
	if u != nil {
		t.Errorf("New(false, nil).Next() got %v, want nil", u)
	}
}

func TestQueueFiniteUpdates(t *testing.T) {
	for _, x := range []int{1, 5, 100} {
		in := &fpb.Value{
			Repeat: int32(x),
			Timestamp: &fpb.Timestamp{
				Timestamp: 1234,
				DeltaMin:  1,
				DeltaMax:  1,
			},
			Value: &fpb.Value_IntValue{&fpb.IntValue{
				Value: 50,
				Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
					Minimum: 0,
					Maximum: 100,
				}}}},
		}
		q := New(false, seed, []*fpb.Value{in})
		for i := 0; i < x; i++ {
			u, err := q.Next()
			if err != nil {
				continue
			}
			if u == nil {
				t.Errorf("New(false, %q).Next() got nil, expected an update %d of %d", in, i, x)
			}
		}
		// Try one more time to valid nil next value.
		u, err := q.Next()
		if err != nil {
			t.Errorf("New(false, %q).Next() unexpected error: got %q for %d updates", in, err, x)
		}
		if u != nil {
			t.Errorf("New(false, %q).Next() got update %v, want nil for %d updates", in, u, x)
		}
	}
}

func TestQueueInfiniteUpdates(t *testing.T) {
	in := &fpb.Value{
		Timestamp: &fpb.Timestamp{
			Timestamp: 1234,
			DeltaMin:  1,
			DeltaMax:  1,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 50,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}
	q := New(false, seed, []*fpb.Value{in})
	// Not really infinite testing, but without repeat set as above, should
	// continue indefinitely.  We check a large definite number as a proxy.
	for i := 0; i < 10000; i++ {
		u, err := q.Next()
		if err != nil {
			t.Errorf("New(false, %q).Next() unexpected error: got %q trying to receive update %d from an infinite queue", in, err, i)
			continue
		}
		if u == nil {
			t.Errorf("New(false, %q).Next() got nil, want update %d from an infinite queue", in, i)
		}
	}
}

func TestQueueDelay(t *testing.T) {
	in := &fpb.Value{
		Timestamp: &fpb.Timestamp{
			Timestamp: 1234,
			DeltaMin:  250 * time.Millisecond.Nanoseconds(),
			DeltaMax:  250 * time.Millisecond.Nanoseconds(),
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 50,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}
	q := New(true, seed, []*fpb.Value{in})
	// No delay to get the first value.
	if _, err := q.Next(); err != nil {
		t.Errorf("New(true, %q).Next() unexpected error: got %q receiving a value from an infinite queue", in, err)
	}
	b := time.Now()
	// Second value should be delayed 250ms.
	if _, err := q.Next(); err != nil {
		t.Errorf("New(true, %q).Next() unexpected error: got %q receiving a value from an infinite queue", in, err)
	}
	if e := time.Since(b); e < 250*time.Millisecond {
		t.Errorf("New(true, %q).Next() got delayed %dms, want delay >= 250ms", in, e/time.Millisecond)
	}
}

func TestQueueAddValue(t *testing.T) {
	in := []*fpb.Value{{
		Repeat: 1,
		Timestamp: &fpb.Timestamp{
			Timestamp: 2,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 50,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}, {
		Repeat: 1,
		Timestamp: &fpb.Timestamp{
			Timestamp: 1,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 50,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}, {
		Repeat: 1,
		Timestamp: &fpb.Timestamp{
			Timestamp: 3,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 50,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}, {
		Repeat: 1,
		Timestamp: &fpb.Timestamp{
			Timestamp: 3,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 60,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}, {
		Repeat: 1,
		Timestamp: &fpb.Timestamp{
			Timestamp: 1,
		},
		Value: &fpb.Value_IntValue{&fpb.IntValue{
			Value: 60,
			Distribution: &fpb.IntValue_Range{Range: &fpb.IntRange{
				Minimum: 0,
				Maximum: 100,
			}}}},
	}}
	q := New(true, seed, in)
	if got, want := q.Latest(), int64(3); got != want {
		t.Errorf("New(true, %q) unexpected Latest(): got %q, want %q", in, got, want)
	}
	if len(q.q) != 3 {
		t.Errorf("New(true, %q) unexpected value set: got %q, want %q", in, q.q, in)
	}
	for i, q := range q.q {
		t.Logf("queue:%d %q", i, q[0].v)
		if q[0].v.Timestamp.Timestamp != int64(i+1) {
			t.Errorf("New(true, %q) unexpected value set: got %q, want Timestamp=%d", in, q[0].v, i+1)
		}
	}
	for i := 0; i < len(in); i++ {
		if _, err := q.Next(); err != nil {
			t.Errorf("New(true, %q).Next() unexpected error: got %q, want nil", in, err)
		}
	}
	u, err := q.Next()
	if err != nil {
		t.Errorf("New(true, %q).Next() unexpected error: got %q, want nil", in, u)
	}
	if u != nil {
		t.Errorf("New(true, %q).Next() unexpected update: got %q, want nil", in, u)
	}
}

func TestValueOf(t *testing.T) {
	tests := []struct {
		desc string
		in   *fpb.Value
		want interface{}
	}{{
		desc: "string value",
		in:   &fpb.Value{Value: &fpb.Value_StringValue{&fpb.StringValue{Value: "UP"}}},
		want: "UP",
	}, {
		desc: "int value",
		in:   &fpb.Value{Value: &fpb.Value_IntValue{&fpb.IntValue{Value: 100}}},
		want: int64(100),
	}, {
		desc: "double value",
		in:   &fpb.Value{Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: float64(101)}}},
		want: float64(101),
	}, {
		desc: "Delete value",
		in:   &fpb.Value{Value: &fpb.Value_Delete{&fpb.DeleteValue{}}},
		want: &fpb.DeleteValue{},
	}, {
		desc: "Sync value",
		in:   &fpb.Value{Value: &fpb.Value_Sync{uint64(1)}},
		want: uint64(1),
	}, {
		desc: "Bool value",
		in:   &fpb.Value{Value: &fpb.Value_BoolValue{&fpb.BoolValue{Value: true}}},
		want: true,
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got, want := ValueOf(tc.in), tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("ValueOf(%q) failed: got %q, want %q", tc.in, got, want)
			}
		})
	}
}

func TestFixedQueue(t *testing.T) {
	tests := []struct {
		desc    string
		in      []*gpb.SubscribeResponse
		delay   bool
		want    *gpb.TypedValue
		updates []*gpb.SubscribeResponse
	}{{
		desc: "empty notifications",
	}, {
		desc: "single sync",
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}},
	}, {
		desc: "sync then updates",
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 1,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 1}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 2}},
					}},
				},
			},
		}},
	}, {
		desc: "sync then updates with add",
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 1,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 1}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 2}},
					}},
				},
			},
		}},
		updates: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 100,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 10}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 20}},
					}},
				},
			},
		}},
	}, {
		desc: "multi update sync",
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 1,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 1}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 2}},
					}},
				},
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 2,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 10}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 20}},
					}},
				},
			},
		}},
	}, {
		desc:  "multi update sync with delay",
		delay: true,
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 1,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 1}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 2}},
					}},
				},
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 2,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 10}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 20}},
					}},
				},
			},
		}},
	}, {
		desc:  "multi update sync with delay <negative>",
		delay: true,
		in: []*gpb.SubscribeResponse{{
			Response: &gpb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 2,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 1}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 2}},
					}},
				},
			},
		}, {
			Response: &gpb.SubscribeResponse_Update{
				Update: &gpb.Notification{
					Timestamp: 1,
					Update: []*gpb.Update{{
						Path: &gpb.Path{Element: []string{"a", "b"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 10}},
					}, {
						Path: &gpb.Path{Element: []string{"a", "c"}},
						Val:  &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 20}},
					}},
				},
			},
		}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			want := make([]*gpb.SubscribeResponse, len(tc.in))
			copy(want, tc.in)
			q := NewFixed(want, tc.delay)
			for _, u := range tc.updates {
				q.Add(u)
				want = append(want, u)
			}
			var got []*gpb.SubscribeResponse
		Loop:
			for {
				v, err := q.Next()
				switch {
				case err != nil:
					t.Fatalf("NewFixed(%q, %v).Next() unexpected error: got %q, want nil", want, tc.delay, err)
				case v == nil:
					break Loop
				}
				got = append(got, v.(*gpb.SubscribeResponse))
			}
			if gotL, wantL := len(got), len(want); gotL != wantL {
				t.Fatalf("q.Next() failed: got length(%d), want length(%d): %v", gotL, wantL, got)
			}
		})
	}
}

func TestTypedValueOf(t *testing.T) {
	tests := []struct {
		desc string
		in   *fpb.Value
		want *gpb.TypedValue
	}{{
		desc: "string value",
		in:   &fpb.Value{Value: &fpb.Value_StringValue{&fpb.StringValue{Value: "UP"}}},
		want: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{"UP"}},
	}, {
		desc: "int value",
		in:   &fpb.Value{Value: &fpb.Value_IntValue{&fpb.IntValue{Value: 100}}},
		want: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{int64(100)}},
	}, {
		desc: "double value",
		in:   &fpb.Value{Value: &fpb.Value_DoubleValue{&fpb.DoubleValue{Value: float64(101)}}},
		want: &gpb.TypedValue{Value: &gpb.TypedValue_FloatVal{float32(101)}},
	}, {
		desc: "delete value",
		in:   &fpb.Value{Value: &fpb.Value_Delete{&fpb.DeleteValue{}}},
		want: nil,
	}, {
		desc: "sync value",
		in:   &fpb.Value{Value: &fpb.Value_Sync{uint64(1)}},
		want: nil,
	}, {
		desc: "bool value",
		in:   &fpb.Value{Value: &fpb.Value_BoolValue{&fpb.BoolValue{Value: true}}},
		want: &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{true}},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got, want := TypedValueOf(tc.in), tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("TypedValueOf(%q) failed: got %q, want %q", tc.in, got, want)
			}
		})
	}
}
