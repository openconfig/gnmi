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
	"fmt"
	"sync"
	"time"

	"github.com/openconfig/gnmi/latency"
)

const (
	// Root node where metadata is cached.
	Root = "meta"

	// Per-target metadata

	// Sync is a boolean that reports whether all target state is cached.
	Sync = "sync"
	// Connected is a boolean that reports whether updates are being received.
	Connected = "connected"
	// ConnectedAddr is a string denoting the last-hop IP address of a connected
	// target.
	ConnectedAddr = "connectedAddress"
	// AddCount is the total number of leaves that have been added.
	AddCount = "targetLeavesAdded"
	// DelCount is the total number of leaves that have been deleted.
	DelCount = "targetLeavesDeleted"
	// EmptyCount is the total number of notifications delivered that contain no
	// updates or deletes.
	EmptyCount = "targetLeavesEmpty"
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
	// Size is the total number of bytes used to store all values.  This count
	// excludes all indexing overhead.
	Size = "targetSize"
	// LatestTimestamp is the latest timestamp for any update received for the
	// target.
	LatestTimestamp = "latestTimestamp"
	// ConnectError is the error related to connection failure.
	ConnectError = "connectError"
	// ServerName is an optional metadata used to identify the server to clients.
	ServerName = "serverName"
)

// IntValue contains the path and other options for an int64 metadata.
type IntValue struct {
	Path     []string // Path of the int64 metadata
	InitZero bool     // Whether to initiate to 0 (for counters starting from 0).
}

// RegisterIntValue registers an int64 type metadata whose path and options
// are in val.
func RegisterIntValue(name string, val *IntValue) {
	TargetIntValues[name] = val
}

// UnregisterIntValue unregisters an int64 type metadata name.
func UnregisterIntValue(name string) {
	delete(TargetIntValues, name)
}

// StrValue contains the valid and the option to reset to emptry string.
type StrValue struct {
	InitEmptyStr bool // Whether to initiate to "".
}

// RegisterStrValue registers a string type metadata.
func RegisterStrValue(name string, val *StrValue) {
	TargetStrValues[name] = val
}

var (
	// TargetBoolValues is the list of all bool metadata fields.
	TargetBoolValues = map[string]bool{
		Sync:      true,
		Connected: true,
	}

	// TargetIntValues is the list of all int64 metadata fields.
	TargetIntValues = map[string]*IntValue{
		AddCount:        {[]string{Root, AddCount}, true},
		DelCount:        {[]string{Root, DelCount}, true},
		EmptyCount:      {[]string{Root, EmptyCount}, true},
		LeafCount:       {[]string{Root, LeafCount}, true},
		UpdateCount:     {[]string{Root, UpdateCount}, true},
		StaleCount:      {[]string{Root, StaleCount}, true},
		SuppressedCount: {[]string{Root, SuppressedCount}, true},
		Size:            {[]string{Root, Size}, true},
		LatestTimestamp: {[]string{Root, LatestTimestamp}, true},
	}

	// TargetStrValues is the list of all string metadata fields.
	TargetStrValues = map[string]*StrValue{
		ConnectedAddr: {InitEmptyStr: true},
		ConnectError:  {InitEmptyStr: false},
	}
)

// Metadata is the container for all target specific metadata.
type Metadata struct {
	mu         sync.Mutex
	valuesInt  map[string]int64
	valuesBool map[string]bool
	valuesStr  map[string]string
}

// Path is a convenience function that will return the full metadata path for
// any valid metadata value.  Only metadata values registered above in
// TargetBoolValues, TargetIntValues, and TargetStrValues will return a path.
// An invalid metadata value will return nil.
func Path(value string) []string {
	if TargetBoolValues[value] {
		return []string{Root, value}
	}
	if val := TargetStrValues[value]; val != nil {
		return []string{Root, value}
	}

	if val := TargetIntValues[value]; val != nil {
		return val.Path
	}
	return nil
}

// New returns an initialized Metadata structure.  Integer values are
// initialized to 0. Boolean values are initialized to false. String values are
// initialized to empty string.
func New() *Metadata {
	m := Metadata{
		valuesInt:  make(map[string]int64, len(TargetIntValues)),
		valuesBool: make(map[string]bool, len(TargetBoolValues)),
		valuesStr:  make(map[string]string, len(TargetStrValues)),
	}
	m.Clear()
	return &m
}

// ErrInvalidValue is returned when a metadata operation is attempted on a value
// that does not exist.
var ErrInvalidValue = errors.New("invalid metadata value")

func validInt(value string) error {
	if val := TargetIntValues[value]; val == nil {
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

func validStr(value string) error {
	if val := TargetStrValues[value]; val == nil {
		return ErrInvalidValue
	}
	return nil
}

// ResetEntry resets metadata entry to zero value. It will be deleted
// if it is Int with InitZero as false, or Str with InitEmptyStr as false.
func (m *Metadata) ResetEntry(entry string) error {
	if validBool(entry) == nil {
		m.SetBool(entry, false)
		return nil
	}

	if validInt(entry) == nil {
		val := TargetIntValues[entry]
		if val.InitZero {
			m.SetInt(entry, 0)
		} else {
			m.mu.Lock()
			delete(m.valuesInt, entry)
			m.mu.Unlock()
		}
		return nil
	}

	if validStr(entry) == nil {
		val := TargetStrValues[entry]
		if val.InitEmptyStr {
			m.SetStr(entry, "")
		} else {
			m.mu.Lock()
			delete(m.valuesStr, entry)
			m.mu.Unlock()
		}
		return nil
	}

	return fmt.Errorf("unsupported entry %q", entry)
}

// Clear sets all metadata values to zero values, except that ConnectError is set to EmptyError.
func (m *Metadata) Clear() {
	for k := range TargetBoolValues {
		m.ResetEntry(k)
	}
	for k := range TargetIntValues {
		m.ResetEntry(k)
	}
	for k := range TargetStrValues {
		m.ResetEntry(k)
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

// SetStr atomically sets the metadata value specified to v.
func (m *Metadata) SetStr(value, v string) error {
	if err := validStr(value); err != nil {
		return err
	}
	m.mu.Lock()
	m.valuesStr[value] = v
	m.mu.Unlock()
	return nil
}

// GetStr atomically retrieves the metadata value specified.
func (m *Metadata) GetStr(value string) (string, error) {
	if err := validStr(value); err != nil {
		return "", err
	}
	m.mu.Lock()
	v, ok := m.valuesStr[value]
	m.mu.Unlock()
	if !ok {
		return "", ErrUnsetValue
	}
	return v, nil
}

// LatencyPath returns the metadata path for the latency statistics of
// window w and type typ.
func LatencyPath(w time.Duration, typ latency.StatType) []string {
	return latency.Path(w, typ, []string{Root})
}

// RegisterLatencyMetadata registers latency stats metadata for time windows
// specified in windowSizes. RegisterLatencyMetadata is not thread-safe and
// should be called before any metadata.Metadata is instantiated.
func RegisterLatencyMetadata(windowSizes []time.Duration) {
	for _, size := range windowSizes {
		for _, typ := range []latency.StatType{latency.Avg, latency.Max, latency.Min} {
			RegisterIntValue(latency.MetadataName(size, typ), &IntValue{Path: LatencyPath(size, typ)})
		}
	}
}

// RegisterServerNameMetadata registers the serverName metadata.
func RegisterServerNameMetadata() {
	RegisterStrValue(ServerName, &StrValue{InitEmptyStr: false})
}
