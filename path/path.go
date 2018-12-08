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

// Package path provides utility functions to convert a given gnmi.Path
// into index strings.
package path

import (
	"errors"
	"sort"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// ToStrings converts gnmi.Path to index strings. When index strings are generated,
// gnmi.Path will be irreversibly lost. Index strings will be built by using name field
// in gnmi.PathElem. If gnmi.PathElem has key field, values will be included in
// alphabetical order of the keys.
// E.g. <target>/<origin>/a/b[b:d, a:c]/e will be returned as <target>/<origin>/a/b/c/d/e
// If prefix parameter is set to true, <target> and <origin> fields of
// the gnmi.Path will be prepended in the index strings unless they are empty string.
// gnmi.Path.Element field is deprecated, but being gracefully handled by this function
// in the absence of gnmi.Path.Elem.
func ToStrings(p *gpb.Path, prefix bool) []string {
	is := []string{}
	if p == nil {
		return is
	}
	if prefix {
		// add target to the list of index strings
		if t := p.GetTarget(); t != "" {
			is = append(is, t)
		}
		// add origin to the list of index strings
		if o := p.GetOrigin(); o != "" {
			is = append(is, o)
		}
	}
	pe := []string{}
	for _, e := range p.GetElem() {
		pe = append(pe, e.GetName())
		pe = append(pe, sortedVals(e.GetKey())...)
	}
	// gnmi.Path.Element is deprecated, but being gracefully handled
	// when gnmi.PathElem doesn't exist
	if len(pe) > 0 {
		is = append(is, pe...)
	} else {
		is = append(is, p.GetElement()...)
	}
	return is
}

func sortedVals(m map[string]string) []string {
	// Special case single key lists.
	if len(m) == 1 {
		for _, v := range m {
			return []string{v}
		}
	}
	// Return deterministic ordering of multi-key lists.
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	vs := make([]string, 0, len(m))
	for _, k := range ks {
		vs = append(vs, m[k])
	}
	return vs
}

// CompletePath joins provided prefix and subscription paths. Also, verifies
// whether origin is set properly according to:
// https://github.com/openconfig/reference/blob/master/rpc/gnmi/mixed-schema.md
// Note that this function doesn't add "openconfig" default origin if neither
// prefix nor path specifies the origin. Also, target field isn't prepended in
// the returned path.
func CompletePath(prefix, path *gpb.Path) ([]string, error) {
	oPre, oPath := prefix.GetOrigin(), path.GetOrigin()

	var fullPrefix []string
	indexedPrefix := ToStrings(prefix, false)
	switch {
	case oPre != "" && oPath != "":
		return nil, errors.New("origin is set both in prefix and path")
	case oPre != "":
		fullPrefix = append(fullPrefix, oPre)
		fullPrefix = append(fullPrefix, indexedPrefix...)
	case oPath != "":
		if len(indexedPrefix) > 0 {
			return nil, errors.New("path elements in prefix are set even though origin is set in path")
		}
		fullPrefix = append(fullPrefix, oPath)
	default:
		// Neither prefix nor path specified an origin. Include the path elements in prefix.
		fullPrefix = append(fullPrefix, indexedPrefix...)
	}

	return append(fullPrefix, ToStrings(path, false)...), nil
}
