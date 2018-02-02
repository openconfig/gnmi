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

package main

import (
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		delim  string
		output []string
		err    bool
	}{
		{name: "Invalid delimiter", delim: "ab", err: true},
		{name: "Dot delimiter", input: "a.b", delim: ".", output: []string{"a", "b"}},
		{name: "Leading delimiter", input: "/foo", delim: "/", output: []string{"foo"}},
		{name: "Trailing delimiter", input: "foo/", delim: "/", output: []string{"foo"}},
		{name: "Leading and trailing delimiter", input: "/foo/", delim: "/", output: []string{"foo"}},
		{name: "No leading and trailing delimiter", input: "foo", delim: "/", output: []string{"foo"}},
		{name: "Leading delimiter multi", input: "/foo/bar", delim: "/", output: []string{"foo", "bar"}},
		{name: "Trailing delimiter multi", input: "foo/bar/", delim: "/", output: []string{"foo", "bar"}},
		{name: "Leading and trailing delimiter multi", input: "/foo/bar/", delim: "/", output: []string{"foo", "bar"}},
		{name: "No leading and trailing delimiter multi", input: "foo/bar", delim: "/", output: []string{"foo", "bar"}},
		{name: "Key value", input: "foo[key=value]/bar", delim: "/", output: []string{"foo[key=value]", "bar"}},
		{name: "Key value contains delimiter", input: "foo[key=a/b]/bar", delim: "/", output: []string{"foo[key=a/b]", "bar"}},
		{name: "Multiple key value contains delimiter", input: "foo[key=a/b]/bar[key=c/d]/blah", delim: "/", output: []string{"foo[key=a/b]", "bar[key=c/d]", "blah"}},
		{name: "Missing [ for key value", input: "fookey=a/b]/bar", err: true},
		{name: "Missing ] for key value", input: "foo[key=a/b/bar", err: true},
		{name: "Nested [] for key value", input: "foo[key=[nest]]/bar", err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseQuery(test.input, test.delim)
			switch {
			case test.err && err != nil:
				return
			case test.err && err == nil:
				t.Errorf("parseQuery(%q): want error, got nil", test.input)
			case err != nil:
				t.Errorf("parseQuery(%q): got error %v, want nil", test.input, err)
			default:
				if !reflect.DeepEqual(got, test.output) {
					t.Errorf("parseQuery(%q): got %q, want %q", test.input, got, test.output)
				}
			}
		})
	}
}
