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

// Package flags defines extra flag types for use in command line flag parsing.
package flags

import (
	"strings"
)

// StringList is a []string container for command line flags. The default
// initialization will create an empty slice. If you need default backing slice
// map use NewStringList.
type StringList []string

func (ss *StringList) String() string {
	return strings.Join(*ss, ",")
}

// Get returns the values of ss. The interface will need to be type asserted to
// StringList for use.
func (ss *StringList) Get() interface{} {
	return *ss
}

// Set sets the value of ss to the comma separated values in s.
func (ss *StringList) Set(s string) error {
	*ss = StringList(strings.Split(s, ","))
	return nil
}

// NewStringList will wrap the pointer to the slice in a StringList and set the
// underlying slice to val.
func NewStringList(p *[]string, val []string) *StringList {
	*p = val
	return (*StringList)(p)
}
