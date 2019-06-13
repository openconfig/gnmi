/*
Copyright 2018 Google Inc.

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

// Package target has utility functions for working with target configuration
// proto messages in target.proto.
package target

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/target"
)

// Update describes a single target configuration.
type Update struct {
	Name    string
	Request *gpb.SubscribeRequest
	Target  *pb.Target
}

// Handler defines callbacks to be synchronously invoked in response to
// configuration changes.
type Handler struct {
	// Add handles addition of a new target.
	Add func(Update)
	// Update handles target modification, including subscription request changes.
	Update func(Update)
	// Delete handles a target being removed.
	Delete func(name string)
}

// Config handles configuration file changes and contains configuration state.
type Config struct {
	h             Handler
	mu            sync.Mutex
	configuration *pb.Configuration
}

// NewConfig creates a new Config that can process configuration changes.
func NewConfig(h Handler) *Config {
	return &Config{
		h: h,
	}
}

// Current returns a copy of the current configuration.
func (c *Config) Current() *pb.Configuration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return proto.Clone(c.configuration).(*pb.Configuration)
}

// Load updates the current configuration and invokes Handler callbacks for
// detected changes. An error is returned when loading fails, or the new revision
// is not strictly increasing.
func (c *Config) Load(config *pb.Configuration) error {
	if config == nil {
		return fmt.Errorf("attempted to load nil configuration")
	}
	if err := Validate(config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkRevision(config); err != nil {
		return err
	}
	// Diff before setting new state.
	c.handleDiffs(config)
	c.configuration = config

	return nil
}

func (c *Config) checkRevision(cf *pb.Configuration) error {
	switch {
	case c.configuration == nil:
		return nil
	case cf.Revision <= c.configuration.GetRevision():
		return fmt.Errorf("revision %v is not strictly greater than %v", cf.Revision, c.configuration.GetRevision())
	}
	return nil
}

// handleDiffs should be called while locking c. It performs a read-only diff on
// state in c against the new configuration.
func (c *Config) handleDiffs(config *pb.Configuration) {
	requestChanged := map[string]bool{}
	for k, new := range config.Request {
		if old, ok := c.configuration.GetRequest()[k]; ok {
			if !proto.Equal(old, new) {
				requestChanged[k] = true
			}
		}
	}

	// Make a copy of new targets so we can safely modify the map.
	newTargets := make(map[string]*pb.Target)
	for k, t := range config.GetTarget() {
		newTargets[k] = t
	}

	for k, t := range c.configuration.GetTarget() {
		nt := newTargets[k]
		switch {
		case nt == nil:
			if c.h.Delete != nil {
				c.h.Delete(k)
			}
		case !requestChanged[t.GetRequest()] && proto.Equal(t, nt):
			delete(newTargets, k)
		default:
			if c.h.Update != nil {
				r := config.GetRequest()[nt.GetRequest()]
				c.h.Update(Update{
					Name:    k,
					Request: r,
					Target:  nt,
				})
			}
			delete(newTargets, k)
		}
	}

	// Anything left in newTargets must be a new target.
	for k, t := range newTargets {
		r := config.GetRequest()[t.GetRequest()]
		if c.h.Add != nil {
			c.h.Add(Update{
				Name:    k,
				Request: r,
				Target:  t,
			})
		}
	}
}

// Validate confirms that the configuration is valid.
func Validate(config *pb.Configuration) error {
	for name, target := range config.Target {
		if name == "" {
			return errors.New("target with empty name")
		}
		if target == nil {
			return fmt.Errorf("missing target configuration for %q", name)
		}
		if len(target.Addresses) == 0 {
			return fmt.Errorf("target %q missing address", name)
		}
		if target.Request == "" {
			return fmt.Errorf("target %q missing request", name)
		}
		if _, ok := config.Request[target.Request]; !ok {
			return fmt.Errorf("missing request %q for target %q", target.Request, name)
		}
	}
	return nil
}
