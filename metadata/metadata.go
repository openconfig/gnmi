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

// Package metadata describes metadata paths stored in the cache per target.
package metadata

import (
	"errors"
	"sync"
)

const (
	// Root node where metadata is cached.
	Root = "meta"

	// Per-target metadata

	// Sync is a boolean that reports whether all target state is cached.
	Sync = "sync"
	// Connected is a boolean that reports whether updates are being received.
	Connected = "connected"
	// AddCount is the total number of leaves that have been added.
	AddCount = "targetLeavesAdded"
	// DelCount is the total number of leaves that have been deleted.
	DelCount = "targetLeavesDeleted"
	// LeafCount is the current total leaves stored in the cache.
	LeafCount = "targetLeaves"
	// UpdateCount is the total number of leaf updates received.
	UpdateCount = "targetLeavesUpdated"
	// StaleCount is the total number of leaf updates that had timestamp older
	// than that cached.
	StaleCount = "targetLeavesStale"
	// SuppressedCount is the total number of leaf updates that were suppressed
	// because the update had the same value as already cached.
	SuppressedCount = "targetLeavesSuppressed"
	// LatencyAvg is the average latency between target timestamp and cache
	// reception.
	LatencyAvg = "latencyAvg"
	// LatencyMax is the maximum latency between target timestamp and cache
	// reception.
	LatencyMax = "latencyMax"
	// LatencyMin is the minimum latency between target timestamp and cache
	// reception.
	LatencyMin = "latencyMin"
	// Size is the total number of bytes used to store all values.  This count
	// excludes all indexing overhead.
	Size = "targetSize"
	// LatestTimestamp is the latest timestamp for any update received for the
	// target.
	LatestTimestamp = "latestTimestamp"
)

var (
	// TargetBoolValues is the list of all bool metadata fields.
	TargetBoolValues = map[string]bool{
		Sync:      true,
		Connected: true,
	}

	// TargetIntValues is the list of all int64 metadata fields.
	TargetIntValues = map[string]bool{
		AddCount:        true,
		DelCount:        true,
		LeafCount:       true,
		UpdateCount:     true,
		StaleCount:      true,
		SuppressedCount: true,
		LatencyAvg:      true,
		LatencyMax:      true,
		LatencyMin:      true,
		Size:            true,
		LatestTimestamp: true,
	}
)

// Metadata is the container for all target specific metadata.
type Metadata struct {
	mu         sync.Mutex
	valuesInt  map[string]int64
	valuesBool map[string]bool
}

// Path is a convenience function that will return the full metadata path for
// any valid metadata value.  Only metadata values registered above in
// TargetBoolValues and TargetIntValues will return a path.  An invalid metadata
// value will return nil.
func Path(value string) []string {
	if TargetBoolValues[value] || TargetIntValues[value] {
		return []string{Root, value}
	}
	return nil
}

// New returns an initialized Metadata structure.  Integer values are
// initialized to 0 and boolean values are initialized to false.
func New() *Metadata {
	m := Metadata{
		valuesInt:  make(map[string]int64, len(TargetIntValues)),
		valuesBool: make(map[string]bool, len(TargetBoolValues)),
	}
	m.Clear()
	return &m
}

// ErrInvalidValue is returned when a metadata operation is attempted on a value
// that does not exist.
var ErrInvalidValue = errors.New("invalid metadata value")

func validInt(value string) error {
	if valid := TargetIntValues[value]; !valid {
		return ErrInvalidValue
	}
	return nil
}

func validBool(value string) error {
	if valid := TargetBoolValues[value]; !valid {
		return ErrInvalidValue
	}
	return nil
}

// Clear sets all metadata values to zero values.
func (m *Metadata) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range TargetBoolValues {
		m.valuesBool[k] = false
	}
	for k := range TargetIntValues {
		m.valuesInt[k] = 0
	}
}

// AddInt atomically increments the metadata value specified by i.
func (m *Metadata) AddInt(value string, i int64) error {
	if err := validInt(value); err != nil {
		return err
	}
	m.mu.Lock()
	m.valuesInt[value] += i
	m.mu.Unlock()
	return nil
}

// SetInt atomically sets the metadata value specified to v.
func (m *Metadata) SetInt(value string, v int64) error {
	if err := validInt(value); err != nil {
		return err
	}
	m.mu.Lock()
	m.valuesInt[value] = v
	m.mu.Unlock()
	return nil
}

// ErrUnsetValue is returned when a metadata Get is attempted on a value that
// has not been Set (or Added).
var ErrUnsetValue = errors.New("unset value")

// GetInt atomically retrieves the metadata value specified.
func (m *Metadata) GetInt(value string) (int64, error) {
	if err := validInt(value); err != nil {
		return 0, err
	}
	m.mu.Lock()
	v, ok := m.valuesInt[value]
	m.mu.Unlock()
	if !ok {
		return 0, ErrUnsetValue
	}
	return v, nil
}

// SetBool atomically sets the metadata value specified to v.
func (m *Metadata) SetBool(value string, v bool) error {
	if err := validBool(value); err != nil {
		return err
	}
	m.mu.Lock()
	m.valuesBool[value] = v
	m.mu.Unlock()
	return nil
}

// GetBool atomically retrieves the metadata value specified.
func (m *Metadata) GetBool(value string) (bool, error) {
	if err := validBool(value); err != nil {
		return false, err
	}
	m.mu.Lock()
	v, ok := m.valuesBool[value]
	m.mu.Unlock()
	if !ok {
		return false, ErrUnsetValue
	}
	return v, nil
}
