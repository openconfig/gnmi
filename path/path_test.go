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

package path

import (
	"reflect"
	"testing"

	"github.com/openconfig/gnmi/errdiff"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestSortedValues(t *testing.T) {
	tests := []struct {
		inp map[string]string
		exp []string
	}{
		{
			inp: map[string]string{},
			exp: []string{},
		},
		{
			inp: map[string]string{"a": "1", "b": "2"},
			exp: []string{"1", "2"},
		},
		{
			inp: map[string]string{"c": "1", "a": "2", "d": "3"},
			exp: []string{"2", "1", "3"},
		},
	}

	for _, tt := range tests {
		ret := sortedVals(tt.inp)
		if !reflect.DeepEqual(ret, tt.exp) {
			t.Errorf("sortedVals(%v) = got %v, want %v", tt.inp, ret, tt.exp)
		}
	}
}

func TestToStrings(t *testing.T) {
	tests := []struct {
		p   *gpb.Path
		wOd bool
		e   []string
	}{
		// test nil gnmi.Path
		{
			e: []string{},
		},
		// test non-nil gnmi.Path
		{
			p: &gpb.Path{},
			e: []string{},
		},
		// test deprecated gnmi.Path.Element
		{
			p: &gpb.Path{
				Element: []string{"x", "y"},
			},
			e: []string{"x", "y"},
		},
		// test <target> is prepended into index string list
		{
			p: &gpb.Path{
				Elem: []*gpb.PathElem{
					&gpb.PathElem{Name: "a"},
					&gpb.PathElem{Name: "b", Key: map[string]string{"n": "c"}},
					&gpb.PathElem{Name: "d"},
				},
				Target: "t",
			},
			e:   []string{"t", "a", "b", "c", "d"},
			wOd: true,
		},
		// test missing <target> isn't added to the index string list
		{
			p: &gpb.Path{
				Elem: []*gpb.PathElem{
					&gpb.PathElem{Name: "a"},
					&gpb.PathElem{Name: "b", Key: map[string]string{"n": "c"}},
					&gpb.PathElem{Name: "d"},
				},
			},
			e:   []string{"a", "b", "c", "d"},
			wOd: true,
		},
		// test both <target> and <origin> are prepended into index string list
		{
			p: &gpb.Path{
				Elem: []*gpb.PathElem{
					&gpb.PathElem{Name: "a"},
					&gpb.PathElem{Name: "b", Key: map[string]string{"n": "c"}},
					&gpb.PathElem{Name: "d"},
				},
				Target: "t",
				Origin: "o",
			},
			e:   []string{"t", "o", "a", "b", "c", "d"},
			wOd: true,
		},
		// test having multiple keys in one gnmi.PathElem
		{
			p: &gpb.Path{
				Elem: []*gpb.PathElem{
					&gpb.PathElem{Name: "a"},
					&gpb.PathElem{Name: "b", Key: map[string]string{"b": "2", "a": "1", "c": "3"}},
					&gpb.PathElem{Name: "c"},
				},
			},
			e: []string{"a", "b", "1", "2", "3", "c"},
		},
		// multiple gnmi.PathElems have multiple keys in them
		{
			p: &gpb.Path{
				Elem: []*gpb.PathElem{
					&gpb.PathElem{Name: "a", Key: map[string]string{"l": "2", "k": "1"}},
					&gpb.PathElem{Name: "b", Key: map[string]string{"b": "2", "a": "1", "c": "3"}},
					&gpb.PathElem{Name: "c"},
				},
			},
			e: []string{"a", "1", "2", "b", "1", "2", "3", "c"},
		},
		// missign gnmi.PathElems, but have gnmi.Path.Element
		{
			p: &gpb.Path{
				Element: []string{"a", "b", "c", "d"},
			},
			e: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		r := ToStrings(tt.p, tt.wOd)
		if !reflect.DeepEqual(r, tt.e) {
			t.Errorf("ToStrings(%v) = got %v, want %v", tt.p, r, tt.e)
		}
	}
}

func TestCompletePath(t *testing.T) {
	tests := []struct {
		desc          string
		inPrefix      *gpb.Path
		inPath        *gpb.Path
		wantSlice     []string
		wantErrSubstr string
	}{
		{
			desc:      "origin is just set in prefix",
			inPrefix:  &gpb.Path{Target: "t", Origin: "o"},
			wantSlice: []string{"o"},
		},
		{
			desc:          "origin is set both in prefix and path",
			inPrefix:      &gpb.Path{Target: "t", Origin: "o"},
			inPath:        &gpb.Path{Origin: "o"},
			wantErrSubstr: "origin is set both in prefix and path",
		},
		{
			desc:          "origin is set in path, but prefix has path elements",
			inPrefix:      &gpb.Path{Target: "t", Elem: []*gpb.PathElem{{Name: "e"}}},
			inPath:        &gpb.Path{Origin: "o"},
			wantErrSubstr: "path elements in prefix are set even though origin is set in path",
		},
		{
			desc:      "origin is set in path properly",
			inPrefix:  &gpb.Path{Target: "t"},
			inPath:    &gpb.Path{Origin: "o", Elem: []*gpb.PathElem{{Name: "e"}}},
			wantSlice: []string{"o", "e"},
		},
	}

	for _, tt := range tests {
		gotSlice, err := CompletePath(tt.inPrefix, tt.inPath)
		if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
			t.Errorf("%s: %v", tt.desc, diff)
		}
		if err != nil {
			continue
		}
		if !reflect.DeepEqual(tt.wantSlice, gotSlice) {
			t.Errorf("%s: got %v, want %v", tt.desc, gotSlice, tt.wantSlice)
		}
	}
}
