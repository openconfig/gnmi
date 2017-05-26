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

package flags

import (
	"fmt"
	"sort"
	"strings"
)

// StringMap is a map[string]string container for command line flags. The
// default initialization will create an empty map. If you need default backing
// map use NewStringMap.
type StringMap map[string]string

func (m *StringMap) String() string {
	s := make([]string, len(*m))
	i := 0
	for k, v := range *m {
		s[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

// Get returns the values of m. The interface will need to be type asserted to
// StringMap for use.
func (m *StringMap) Get() interface{} {
	return *m
}

// Set will take a string in the format <key1>=<value1>,<key2>=<value2> and
// parse the resulting value into a map[string]string.
func (m *StringMap) Set(v string) error {
	*m = StringMap{}
	for _, entry := range strings.Split(v, ",") {
		data := strings.SplitN(entry, "=", 2)
		if len(data) != 2 {
			return fmt.Errorf("invalid key=value pair: %s", entry)
		}
		k := strings.TrimSpace(data[0])
		v := strings.TrimSpace(data[1])
		if len(k) == 0 {
			return fmt.Errorf("invalid key=value pair: %s", entry)
		}
		(*m)[k] = v
	}
	return nil
}

// NewStringMap will wrap the pointer to the map in a StringMap and set the
// underlying map to val.
func NewStringMap(p *map[string]string, val map[string]string) *StringMap {
	*p = val
	return (*StringMap)(p)
}
