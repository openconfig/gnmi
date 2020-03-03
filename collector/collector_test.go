/*
Copyright 2020 Google Inc.

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

package collector

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/openconfig/gnmi/proto/collector"
)

func TestReconnect(t *testing.T) {
	server := New(func(target string) error {
		targets := map[string]bool{
			"target1": true,
			"target2": true,
		}
		if _, ok := targets[target]; !ok {
			return fmt.Errorf("invalid target %q", target)
		}
		return nil
	})
	for _, tt := range []struct {
		desc    string
		targets []string
		err     bool
	}{
		{desc: "empty request"},
		{desc: "one target", targets: []string{"target1"}},
		{desc: "multiple targets", targets: []string{"target1", "target2"}},
		{desc: "error target", targets: []string{"unknown"}, err: true},
		{desc: "multiple targets, one error", targets: []string{"target1", "unknown"}, err: true},
		{desc: "multiple errors", targets: []string{"unknown1", "unknown2"}, err: true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := server.Reconnect(context.Background(), &pb.ReconnectRequest{Target: tt.targets})
			switch {
			case err != nil && !tt.err:
				t.Errorf("got error %v, want nil", err)
			case err == nil && tt.err:
				t.Error("got nil, want error")
			}
		})
	}
}
