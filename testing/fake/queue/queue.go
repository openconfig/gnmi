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

// Package queue implements an update queue for use in testing a telemetry
// stream.
package queue

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

// Queue is a generic interface for getting the next element from either a
// FixedQeueue or UpdateQueue.
type Queue interface {
	Next() (interface{}, error)
}

// UpdateQueue is a structure that orders a set of device updates by their
// timestamps and repeatedly generates new pseudo-random updates based on
// a set of device path configurations.
type UpdateQueue struct {
	mu       sync.Mutex
	delay    bool
	duration time.Duration
	latest   int64
	q        [][]*value
	r        *rand.Rand
}

// New creates a new UpdateQueue. If delay is true, a call to Next()
// will invoke a sleep based on the duration between timestamps of the last
// returned update and the update to be returned by the current call. The
// values are the configuration for the updates stored in the queue.
func New(delay bool, seed int64, values []*fpb.Value) *UpdateQueue {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	u := &UpdateQueue{delay: delay, r: rand.New(rand.NewSource(seed))}
	for _, v := range values {
		u.addValue(newValue(v, u.r))
	}
	return u
}

// Add inserts v into the queue in correct timestamp order.
func (u *UpdateQueue) Add(v *fpb.Value) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.addValue(newValue(v, u.r))
}

// Latest returns the maximum timestamp in the queue.
func (u *UpdateQueue) Latest() int64 {
	return u.latest
}

// Next returns the next update in the queue or an error.  If the queue is
// exhausted, a nil is returned for the update. The return will always be a
// *fpb.Value for proper type assertion.
func (u *UpdateQueue) Next() (interface{}, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.q) == 0 {
		return nil, nil
	}
	// Incur a real delay if requested.
	if u.duration > 0 {
		time.Sleep(u.duration)
		u.duration = 0
	}
	v := u.q[0][0]
	// Make a copy of the current proto message to return.
	val := v.v
	// Update the value.
	if err := v.nextValue(); err != nil {
		return nil, err
	}
	var checkDelay bool
	// If this is the last update in the queue move to the next update set.
	// else move to the next update in the same update set.
	if len(u.q[0]) == 1 {
		u.q = u.q[1:]
		checkDelay = true
	} else {
		u.q[0] = u.q[0][1:]
	}
	// Add the updated value to the queue if the repeats are not exhausted.
	if v.v != nil {
		u.addValue(v)
	}
	// Set up a delay for the next retrieval if realtime delays are requested.
	if u.delay && checkDelay && len(u.q) > 0 {
		u.duration = time.Duration(u.q[0][0].v.Timestamp.Timestamp-val.Timestamp.Timestamp) * time.Nanosecond
	}
	return val, nil
}

func (u *UpdateQueue) addValue(v *value) {
	// If Timestamp was not provided create a default.
	if v.v.Timestamp == nil {
		v.v.Timestamp = &fpb.Timestamp{}
	}
	t := v.v.Timestamp.Timestamp
	l, r := 0, len(u.q)
	if t > u.latest {
		u.latest = t
	}
	// Binary search for sorted placement in queue.
	for {
		if l == r {
			// Insert a new list of updates for a new timestamp.
			u.q = append(u.q[:r], append([][]*value{{v}}, u.q[r:]...)...)
			return
		}
		i := (r-l)/2 + l
		t2 := u.q[i][0].v.Timestamp.Timestamp
		switch {
		case t == t2:
			// Append update to a list for an existing timestamp.
			u.q[i] = append(u.q[i], v)
			return
		case t < t2:
			r = i
		case t > t2:
			l = i + 1
		}
	}
}

type value struct {
	v *fpb.Value // The configuration for a stream of updates for a single value.
	r *rand.Rand // The PRNG used to generate subsequent updates for this value.
}

func (v value) String() string {
	return v.v.String()
}

func newValue(v *fpb.Value, r *rand.Rand) *value {
	if v.Seed == 0 {
		return &value{v: v, r: r}
	}
	return &value{v: v, r: rand.New(rand.NewSource(v.Seed))}
}

func (v *value) updateTimestamp() error {
	if v.v.Timestamp == nil {
		return fmt.Errorf("timestamp not set for %q", v.v)
	}
	t := v.v.Timestamp.Timestamp
	if t < 0 {
		return fmt.Errorf("timestamp must be positive for %q", v.v)
	}
	min, max := v.v.Timestamp.DeltaMin, v.v.Timestamp.DeltaMax
	if min > max || min < 0 {
		return fmt.Errorf("invalid delta_min/delta_max on timestamp for %q", v.v)
	}
	v.v.Timestamp.Timestamp = t + v.r.Int63n(max-min+1) + min
	return nil
}

func (v *value) updateIntValue() error {
	val := v.v.GetIntValue()
	if val == nil {
		return fmt.Errorf("invalid IntValue for %q", v.v)
	}
	var newval int64
	switch val.Distribution.(type) {
	case *fpb.IntValue_Range:
		rng := val.GetRange()
		if rng.Minimum > rng.Maximum {
			return fmt.Errorf("invalid minimum/maximum in IntRange for %q", v.v)
		}
		if val.Value < rng.Minimum || val.Value > rng.Maximum {
			return fmt.Errorf("value not in [minimum, maximum] in IntRange for %q", v.v)
		}

		left, right := rng.GetMinimum(), rng.GetMaximum()
		if rng.DeltaMin != 0 || rng.DeltaMax != 0 {
			if rng.DeltaMin > rng.DeltaMax {
				return fmt.Errorf("invalid delta_min/delta_max in IntRange for %q", v.v)
			}
			left, right = rng.GetDeltaMin(), rng.GetDeltaMax()
			newval = val.Value
		}

		newval += v.r.Int63n(right-left+1) + left
		if newval > rng.Maximum {
			newval = rng.Maximum
		}
		if newval < rng.Minimum {
			newval = rng.Minimum
		}
	case *fpb.IntValue_List:
		list := val.GetList()
		options := list.GetOptions()
		if len(options) == 0 {
			return fmt.Errorf("missing options on IntValue_List for %q", v.v)
		}
		if list.GetRandom() {
			newval = options[v.r.Intn(len(options))]
		} else {
			newval = options[0]
			list.Options = append(options[1:], options[0])
		}
	default:
		newval = val.Value
	}
	val.Value = newval
	return nil
}

func (v *value) updateDoubleValue() error {
	val := v.v.GetDoubleValue()
	if val == nil {
		return fmt.Errorf("invalid DoubleValue for %q", v.v)
	}
	var newval float64
	switch val.Distribution.(type) {
	case *fpb.DoubleValue_Range:
		rng := val.GetRange()
		if rng.Minimum > rng.Maximum {
			return fmt.Errorf("invalid minimum/maximum on DoubleValue_Range for %q", v.v)
		}
		if val.Value < rng.Minimum || val.Value > rng.Maximum {
			return fmt.Errorf("value not in [minimum, maximum] on DoubleValue_Range for %q", v.v)
		}

		left, right := rng.GetMinimum(), rng.GetMaximum()
		if rng.DeltaMin != 0 || rng.DeltaMax != 0 {
			if rng.DeltaMin > rng.DeltaMax {
				return fmt.Errorf("invalid delta_min/delta_max on DoubleValue_Range for %q", v.v)
			}
			left, right = rng.GetDeltaMin(), rng.GetDeltaMax()
			newval = val.Value
		}

		newval += v.r.Float64()*(right-left) + left
		if newval > rng.Maximum {
			newval = rng.Maximum
		}
		if newval < rng.Minimum {
			newval = rng.Minimum
		}
	case *fpb.DoubleValue_List:
		list := val.GetList()
		options := list.GetOptions()
		if len(options) == 0 {
			return fmt.Errorf("missing options on DoubleValue_List for %q", v.v)
		}
		if list.GetRandom() {
			newval = options[v.r.Intn(len(options))]
		} else {
			newval = options[0]
			list.Options = append(options[1:], options[0])
		}
	default:
		newval = val.Value
	}
	val.Value = newval
	return nil
}

func (v *value) updateStringValue() error {
	val := v.v.GetStringValue()
	if val == nil {
		return fmt.Errorf("invalid StringValue for %q", v.v)
	}
	var newval string
	switch val.Distribution.(type) {
	case *fpb.StringValue_List:
		list := val.GetList()
		options := list.Options
		if len(options) == 0 {
			return fmt.Errorf("missing options on StringValue_List for %q", v.v)
		}
		if list.Random {
			newval = options[v.r.Intn(len(options))]
		} else {
			newval = options[0]
			list.Options = append(options[1:], options[0])
		}
	default:
		newval = val.Value
	}
	val.Value = newval
	return nil
}

func (v *value) nextValue() error {
	if v.v.Repeat == 1 {
		// This value has exhausted all of its updates, drop it.
		v.v = nil
		return nil
	}
	// Make a new proto message for the next value.
	v.v = proto.Clone(v.v).(*fpb.Value)
	if v.v.Repeat > 1 {
		v.v.Repeat--
	}
	if err := v.updateTimestamp(); err != nil {
		return err
	}
	switch v.v.GetValue().(type) {
	case *fpb.Value_IntValue:
		return v.updateIntValue()
	case *fpb.Value_DoubleValue:
		return v.updateDoubleValue()
	case *fpb.Value_StringValue:
		return v.updateStringValue()
	case *fpb.Value_Sync:
		return nil
	case *fpb.Value_Delete:
		return nil
	default:
		return fmt.Errorf("value type not found in %q", v.v)
	}
}

// ValueOf returns the concrete value of v.
func ValueOf(v *fpb.Value) interface{} {
	switch val := v.GetValue().(type) {
	case *fpb.Value_IntValue:
		return val.IntValue.Value
	case *fpb.Value_DoubleValue:
		return val.DoubleValue.Value
	case *fpb.Value_StringValue:
		return val.StringValue.Value
	case *fpb.Value_Sync:
		return val.Sync
	case *fpb.Value_Delete:
		return val.Delete
	default:
		return nil
	}
}

// TypedValueOf returns the gNMI TypedValue of v. If v is a Sync or Delete,
// TypedValueOf returns nil.
func TypedValueOf(v *fpb.Value) *gpb.TypedValue {
	tv := &gpb.TypedValue{}
	switch val := v.GetValue().(type) {
	case *fpb.Value_IntValue:
		tv.Value = &gpb.TypedValue_IntVal{val.IntValue.Value}
	case *fpb.Value_DoubleValue:
		tv.Value = &gpb.TypedValue_FloatVal{float32(val.DoubleValue.Value)}
	case *fpb.Value_StringValue:
		tv.Value = &gpb.TypedValue_StringVal{val.StringValue.Value}
	default:
		return nil
	}
	return tv
}
