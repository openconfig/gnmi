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

	pb "github.com/openconfig/gnmi/proto/target"
)

// Validate confirms that the configuration is valid.
func Validate(config *pb.Configuration) error {
	for name, target := range config.Target {
		if name == "" {
			return errors.New("target with empty name")
		}
		if target == nil {
			return fmt.Errorf("missing target configuration for %q", name)
		}
		if target.Address == "" {
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
