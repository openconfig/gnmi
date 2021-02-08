/*
Copyright 2021 Google Inc.

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

// Package latency supports exporting latency stats (avg/max/min) of a set
// of time windows as metadata.
package latency

import (
	"fmt"
	"sync"
	"time"

	"github.com/openconfig/gnmi/metadata"
)

const (
	// ElemLatency is the container of all latency metadata about a target.
	ElemLatency = "latency"
	// ElemWindow contains latency metadatas (avg, max, min) of a particular
	// window size.
	ElemWindow = "window"
	// ElemAvg is the average latency during a time window.
	ElemAvg = "avg"
	// ElemMax is the maximum latency during a time window.
	ElemMax = "max"
	// ElemMin is the minimum latency during a time window.
	ElemMin  = "min"
	metaName = "LatencyWindow"
)

var now = time.Now

// StatType is the type of latency statistics supported for a time window.
type StatType int

const (
	// Avg is the average latency of a time window.
	Avg = StatType(iota)
	// Max is the maximum latency of a time window.
	Max
	// Min is the minimum latency of a time window.
	Min
)

// String returns the string representation of the StatType.
func (st StatType) String() string {
	switch st {
	case Avg:
		return ElemAvg
	case Max:
		return ElemMax
	case Min:
		return ElemMin
	}
	return "unknown"
}

// stat represents a latency statistic of a time window.
type stat struct {
	window time.Duration // Window size.
	typ    StatType      // Type of the latency stats.
}

// metaName returns the metadata name of the stat s.
func (s stat) metaName() string {
	return fmt.Sprintf("%s%s%s", s.typ, metaName, s.window)
}

// metaPath returns the metadata path corresponding to the Stat s.
func (s stat) metaPath() []string {
	return []string{metadata.Root, ElemLatency, ElemWindow, fmt.Sprint(s.window), s.typ.String()}
}

// Path returns the metadata path for the latency statistics of window w
// and type typ.
func Path(w time.Duration, typ StatType) []string {
	return stat{window: w, typ: typ}.metaPath()
}

// MetadataName returns the metadata name for the latency statistics
// of window w and type typ.
func MetadataName(w time.Duration, typ StatType) string {
	return stat{window: w, typ: typ}.metaName()
}

type slot struct {
	total time.Duration // cumulative latency of this time slot
	max   time.Duration // maximum latency of this time slot
	min   time.Duration // minimum latency of this time slot
	count int64         // number of updates
	start time.Time     // the start time of the time slot
	end   time.Time     // the end time of the time slot
}

type window struct {
	stats   map[string]func(string, *metadata.Metadata)
	size    time.Duration // window size
	total   time.Duration // cumulative latency of this time window
	count   int64         // number of updates
	slots   []*slot       // time slots of latencies
	covered bool          // have received latencies covering a full window
}

func newWindow(size time.Duration) *window {
	w := &window{
		stats: map[string]func(string, *metadata.Metadata){},
		size:  size,
	}
	for st, f := range map[StatType]func(string, *metadata.Metadata){
		Avg: w.setAvg,
		Max: w.setMax,
		Min: w.setMin} {
		stat := stat{window: size, typ: st}
		w.stats[stat.metaName()] = f
	}
	return w
}

func (w *window) add(ls *slot) {
	if ls == nil || ls.count == 0 {
		return
	}
	w.total = w.total + ls.total
	w.count = w.count + ls.count
	w.slots = append(w.slots, ls)
}

func (w *window) setAvg(name string, m *metadata.Metadata) {
	if w.count == 0 {
		return
	}
	avg := w.total / time.Duration(w.count)
	if n := avg.Nanoseconds(); n != 0 {
		m.SetInt(name, n)
	}
}

func (w *window) setMax(name string, m *metadata.Metadata) {
	var max time.Duration
	for _, slot := range w.slots {
		if slot.max > max {
			max = slot.max
		}
	}
	if n := max.Nanoseconds(); n != 0 {
		m.SetInt(name, n)
	}
}

func (w *window) setMin(name string, m *metadata.Metadata) {
	if len(w.slots) == 0 {
		return
	}
	min := w.slots[0].min
	for _, slot := range w.slots[1:] {
		if slot.min < min {
			min = slot.min
		}
	}
	if n := min.Nanoseconds(); n != 0 {
		m.SetInt(name, n)
	}
}

func (w *window) slide(ts time.Time) {
	cutoff := ts.Add(-w.size)
	start := 0
	for _, s := range w.slots {
		if !s.end.After(cutoff) {
			w.count = w.count - s.count
			w.total = w.total - s.total
			start++
		}
	}
	w.slots = w.slots[start:]
}

func (w *window) isCovered(ts time.Time) bool {
	if w.covered {
		return true
	}
	if len(w.slots) == 0 { // no updates received
		return false
	}
	if ts.Sub(w.slots[0].start) >= w.size {
		w.covered = true
		return true
	}
	return false
}

func (w *window) updateMeta(m *metadata.Metadata, ts time.Time) {
	if !w.isCovered(ts) {
		return
	}
	w.slide(ts)
	for name, f := range w.stats {
		f(name, m)
	}
}

// Latency supports calculating and exporting latency stats for a specified
// set of time windows.
type Latency struct {
	mu        sync.Mutex
	start     time.Time     // start time of the current batch of cumulated latency stats
	totalDiff time.Duration // cumulative difference in timestamps from device
	count     int64         // number of updates in latency count
	min       time.Duration // minimum latency
	max       time.Duration // maximum latency
	windows   []*window
}

// New returns a Latency object supporting latency stats for time windows
// specified in windowSizes.
func New(windowSizes []time.Duration) *Latency {
	var windows []*window
	for _, size := range windowSizes {
		windows = append(windows, newWindow(size))
	}
	return &Latency{windows: windows}
}

// Compute calculates the time difference between now and ts (the timestamp
// of an update) and updates the latency stats saved in Latency.
func (l *Latency) Compute(ts time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	nowTime := now()
	lat := nowTime.Sub(ts)
	l.totalDiff += lat
	l.count++
	if lat > l.max {
		l.max = lat
	}
	if lat < l.min || l.min == 0 {
		l.min = lat
	}
	if l.start.IsZero() {
		l.start = nowTime
	}
}

// UpdateReset use the latencies saved during the last interval to update
// the latency stats of all the supported time windows. And then it updates
// the corresponding stats in Metadata m.
// UpdateReset is expected to be called periodically at a fixed interval
// (e.g. 2s) of which the time windows should be multiples of this interval.
func (l *Latency) UpdateReset(m *metadata.Metadata) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := now()
	defer func() {
		for _, window := range l.windows {
			window.updateMeta(m, ts)
		}
		l.start = ts
	}()
	if l.count == 0 {
		return
	}
	s := &slot{
		total: l.totalDiff,
		count: l.count,
		max:   l.max,
		min:   l.min,
		start: l.start,
		end:   ts,
	}
	for _, window := range l.windows {
		window.add(s)
	}
	l.totalDiff = 0
	l.count = 0
	l.min = 0
	l.max = 0
}

// RegisterMetadata registers latency stats metadata for time windows
// specified in windowSizes. RegisterMetadata is not thread-safe and
// should be called before any metadata.Metadata is instantiated.
func RegisterMetadata(windowSizes []time.Duration) {
	for _, size := range windowSizes {
		for _, typ := range []StatType{Avg, Max, Min} {
			st := stat{window: size, typ: typ}
			metadata.RegisterIntValue(st.metaName(), &metadata.IntValue{Path: st.metaPath()})
		}
	}
}

// ParseWindows parses the time durations of latency windows and verify they
// are multiples of the metadata update period.
func ParseWindows(tds []string, metaUpdatePeriod time.Duration) ([]time.Duration, error) {
	var durs []time.Duration
	for _, td := range tds {
		dur, err := time.ParseDuration(td)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %v", td, err)
		}
		if dur.Nanoseconds()%metaUpdatePeriod.Nanoseconds() != 0 {
			return nil, fmt.Errorf("latency stats window %s is not a multiple of metadata update period %v", td, metaUpdatePeriod)
		}
		durs = append(durs, dur)
	}
	return durs, nil
}
