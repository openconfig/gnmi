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
	"flag"
	"reflect"
	"strings"
	"testing"
)

var tests = []struct {
	desc           string
	in             []string
	wantStringList StringList
	wantStringMap  StringMap
	wantIntMap     IntMap
	wantErr        string
}{{
	desc:           "empty args",
	in:             []string{},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
}, {
	desc:           "Stringlist args",
	in:             []string{"-test_stringList=asdf,foo"},
	wantStringList: []string{"asdf", "foo"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
}, {
	desc:           "StringMap args",
	in:             []string{"-test_stringMap=a=10,b=20"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "10", "b": "20"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
}, {
	desc:           "StringMap args (missing default key)",
	in:             []string{"-test_stringMap=a=10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "10", "c": "30"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
}, {
	desc:           "StringMap args (invalid key/value)",
	in:             []string{"-test_stringMap=a10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "10", "c": "30"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
	wantErr:        "invalid value",
}, {
	desc:           "StringMap args (nil key)",
	in:             []string{"-test_stringMap==10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "10", "c": "30"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
	wantErr:        "invalid value",
}, {
	desc:           "StringMap args (empty)",
	in:             []string{"-test_stringMap="},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
	wantErr:        "invalid value",
}, {
	desc:           "IntMap args",
	in:             []string{"-test_intMap=a=10,b=20"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "b": 20},
}, {
	desc:           "IntMap args (additional)",
	in:             []string{"-test_intMap=a=10,b=20,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "b": 20, "c": 30},
}, {
	desc:           "IntMap args (additional spaces)",
	in:             []string{"-test_intMap=a= 10, b=20, c= 30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "b": 20, "c": 30},
}, {
	desc:           "IntMap args (parse error)",
	in:             []string{"-test_intMap=a= 10, b=20, c=asdf"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "b": 20, "c": 30},
	wantErr:        "invalid value",
}, {
	desc:           "IntMap args (missing default key)",
	in:             []string{"-test_intMap=a=10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "c": 30},
	wantErr:        "invalid value",
}, {
	desc:           "IntMap args (invalid key/value)",
	in:             []string{"-test_intMap=a10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "c": 30},
	wantErr:        "invalid value",
}, {
	desc:           "IntMap args (nil key)",
	in:             []string{"-test_intMap==10,c=30"},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{"a": 10, "c": 30},
	wantErr:        "invalid value",
}, {
	desc:           "IntMap args (empty)",
	in:             []string{"-test_intMap="},
	wantStringList: []string{"0"},
	wantStringMap:  map[string]string{"a": "1", "b": "2"},
	wantIntMap:     map[string]int64{},
	wantErr:        "invalid value",
}, {
	desc:           "All Set",
	in:             []string{"-test_intMap=a=1,b=2", "-test_stringMap=foo=bar,baz=faz", "-test_stringList=asdf,foo,bar"},
	wantStringList: []string{"asdf", "foo", "bar"},
	wantStringMap:  map[string]string{"foo": "bar", "baz": "faz"},
	wantIntMap:     map[string]int64{"a": 1, "b": 2},
	wantErr:        "invalid value",
}}

func TestFlags(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			stringList := StringList{"0"}
			stringMap := StringMap{"a": "1", "b": "2"}
			intMap := IntMap{"a": 1, "b": 2}
			f := &flag.FlagSet{}
			f.Var(&stringList, "test_stringList", "[]string value")
			f.Var(&stringMap, "test_stringMap", "map[string]string")
			f.Var(&intMap, "test_intMap", "map[string]int64")
			err := f.Parse(tt.in)
			if err != nil {
				if strings.Contains(err.Error(), tt.wantErr) {
					return
				}
				t.Errorf("flag.CommandLine.Parse(%v) err: %v", tt.in, err)
			}
			if !reflect.DeepEqual(stringList, tt.wantStringList) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if !reflect.DeepEqual(stringList.Get().(StringList), tt.wantStringList) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList.Get() got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if stringList.String() != tt.wantStringList.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, stringList.String(), tt.wantStringList.String())
			}
			if !reflect.DeepEqual(stringMap, tt.wantStringMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringMap got %v, want %v", tt.in, stringMap, tt.wantStringMap)
			}
			if !reflect.DeepEqual(stringMap.Get().(StringMap), tt.wantStringMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringMap.Get() got %v, want %v", tt.in, stringMap, tt.wantStringMap)
			}
			if stringMap.String() != tt.wantStringMap.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, stringMap.String(), tt.wantStringMap.String())
			}
			if !reflect.DeepEqual(intMap, tt.wantIntMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: intMap got %v, want %v", tt.in, intMap, tt.wantIntMap)
			}
			if !reflect.DeepEqual(intMap.Get().(IntMap), tt.wantIntMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: intMap.Get() got %v, want %v", tt.in, intMap, tt.wantIntMap)
			}
			if intMap.String() != tt.wantIntMap.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, intMap.String(), tt.wantIntMap.String())
			}
		})
	}
}

func TestNewFlags(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			listVar := []string{}
			stringMapVar := map[string]string{}
			intMapVar := map[string]int64{}
			stringList := NewStringList(&listVar, []string{"0"})
			stringMap := NewStringMap(&stringMapVar, map[string]string{"a": "1", "b": "2"})
			intMap := NewIntMap(&intMapVar, map[string]int64{"a": 1, "b": 2})
			f := &flag.FlagSet{}
			f.Var(stringList, "test_stringList", "[]string value")
			f.Var(stringMap, "test_stringMap", "map[string]string")
			f.Var(intMap, "test_intMap", "map[string]int64")
			err := f.Parse(tt.in)
			if err != nil {
				if strings.Contains(err.Error(), tt.wantErr) {
					return
				}
				t.Errorf("flag.CommandLine.Parse(%v) err: %v", tt.in, err)
			}
			if reflect.ValueOf(listVar).Pointer() != reflect.ValueOf(*stringList).Pointer() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList Pointer() got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if !reflect.DeepEqual(*stringList, tt.wantStringList) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if !reflect.DeepEqual(stringList.Get().(StringList), tt.wantStringList) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList.Get() got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if stringList.String() != tt.wantStringList.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, stringList.String(), tt.wantStringList.String())
			}
			if reflect.ValueOf(stringMapVar).Pointer() != reflect.ValueOf(*stringMap).Pointer() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringMap Pointer() got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if !reflect.DeepEqual(*stringMap, tt.wantStringMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringMap got %v, want %v", tt.in, stringMap, tt.wantStringMap)
			}
			if !reflect.DeepEqual(stringMap.Get().(StringMap), tt.wantStringMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringMap.Get() got %v, want %v", tt.in, stringMap, tt.wantStringMap)
			}
			if stringMap.String() != tt.wantStringMap.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, stringMap.String(), tt.wantStringMap.String())
			}
			if reflect.ValueOf(intMapVar).Pointer() != reflect.ValueOf(*intMap).Pointer() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: intMap Pointer() got %v, want %v", tt.in, stringList, tt.wantStringList)
			}
			if !reflect.DeepEqual(*intMap, tt.wantIntMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: intMap got %v, want %v", tt.in, intMap, tt.wantIntMap)
			}
			if !reflect.DeepEqual(intMap.Get().(IntMap), tt.wantIntMap) {
				t.Errorf("flag.CommandLine.Parse(%v) failed: intMap.Get() got %v, want %v", tt.in, intMap, tt.wantIntMap)
			}
			if intMap.String() != tt.wantIntMap.String() {
				t.Errorf("flag.CommandLine.Parse(%v) failed: stringList got %q, want %q", tt.in, intMap.String(), tt.wantIntMap.String())
			}
		})
	}
}
