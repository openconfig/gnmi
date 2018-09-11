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

package main

import (
	"path/filepath"
	"testing"
	"time"

	"context"
)

func TestErrorsRunCollector(t *testing.T) {
	tests := []struct {
		desc, config, cert, key string
		valid                   bool
	}{{
		desc:   "valid config, cert and key",
		config: "good.cfg",
		cert:   "good.crt",
		key:    "good.key",
		valid:  true,
	}, {
		desc: "empty config name",
		cert: "good.crt",
		key:  "good.key",
	}, {
		desc:   "unparseable config",
		config: "unparseable.cfg",
		cert:   "good.crt",
		key:    "good.key",
	}, {
		desc:   "invalid config",
		config: "bad.cfg",
		cert:   "good.crt",
		key:    "good.key",
	}, {
		desc:   "missing config",
		config: "missing.cfg",
		cert:   "good.crt",
		key:    "good.key",
	}, {
		desc:   "empty cert name",
		config: "good.cfg",
		key:    "good.key",
	}, {
		desc:   "invalid cert",
		config: "good.cfg",
		cert:   "bad.crt",
		key:    "good.key",
	}, {
		desc:   "missing cert",
		config: "good.cfg",
		cert:   "missing.crt",
		key:    "good.key",
	}, {
		desc:   "empty key name",
		config: "good.cfg",
		cert:   "good.crt",
	}, {
		desc:   "invalid key",
		config: "good.cfg",
		cert:   "good.crt",
		key:    "bad.key",
	}, {
		key:    "missing.key",
		config: "good.cfg",
		cert:   "good.crt",
		desc:   "missing key",
	}}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.config != "" {
				*configFile = filepath.Join("testdata", test.config)
				defer func() { *configFile = "" }()
			}
			if test.cert != "" {
				*certFile = filepath.Join("testdata", test.cert)
				defer func() { *certFile = "" }()
			}
			if test.key != "" {
				*keyFile = filepath.Join("testdata", test.key)
				defer func() { *keyFile = "" }()
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := runCollector(ctx)
			t.Logf("runCollector(ctx) : %v", err)
			if test.valid {
				if err != context.DeadlineExceeded {
					t.Errorf("got %v, want %v", err, context.DeadlineExceeded)
				}
			} else {
				if err == nil {
					t.Error("got nil error, wanted error")
				}
			}
		})
	}
}
