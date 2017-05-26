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
	"strconv"
	"strings"
)

// IntMap is a map[string]int64 container for command line flags.  The default
// initialization will create an empty map. If you need default backing map use
// NewIntMap.
type IntMap map[string]int64

func (m *IntMap) String() string {
	s := make([]string, 0, len(*m))
	for k, v := range *m {
		s = append(s, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

// Get returns the values of m. The interface will need to be type asserted to
// IntMap for use.
func (m *IntMap) Get() interface{} {
	return *m
}

// Set will take a string in the format <key1>=<value1>,<key2>=<value2> and
// parse the resulting value into a map[string]int64. Values may contain "=",
// keys may not.
func (m *IntMap) Set(v string) error {
	*m = IntMap{}
	for _, entry := range strings.Split(v, ",") {
		data := strings.SplitN(entry, "=", 2)
		if len(data) != 2 {
			return fmt.Errorf("invalid key=value pair: %s", entry)
		}
		k := strings.TrimSpace(data[0])
		vString := strings.TrimSpace(data[1])
		if len(k) == 0 {
			return fmt.Errorf("invalid key=value pair: %s", entry)
		}
		var err error
		v := 0
		if len(vString) != 0 {
			v, err = strconv.Atoi(vString)
			if err != nil {
				return err
			}
		}
		(*m)[k] = int64(v)
	}
	return nil
}

// NewIntMap will wrap the pointer to the map in a IntMap and set the
// underlying map to val.
func NewIntMap(p *map[string]int64, val map[string]int64) *IntMap {
	*p = val
	return (*IntMap)(p)
}
