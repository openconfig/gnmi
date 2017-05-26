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

package errlist_test

// Use package errlist_test rather than errlist to prevent the tests from having
// any internal access to the actual List or Error types.

import (
	"errors"
	"strings"
	"testing"

	"github.com/openconfig/gnmi/errlist"
)

func TestNil(t *testing.T) {
	var err errlist.List

	func(e error) {
		if e != nil {
			t.Error("did not get expected nil")
		}
	}(err.Err())
}

func TestInteface(t *testing.T) {
	var err errlist.List
	var i interface{}
	i = err

	if _, ok := i.(error); ok {
		t.Error("List implements error, it should not")
	}
	err.Add(errors.New("some error"))
	i = err.Err()
	if _, ok := i.(error); !ok {
		t.Error("List.Err does not implement error, it should")
	}
}

func errsEqual(a, b []error) bool {
	if len(a) != len(b) {
		return false
	}
	for x, s := range a {
		if b[x] != s {
			return false
		}
	}
	return true
}

func TestAdd(t *testing.T) {
	var err errlist.List
	errs := []error{
		errors.New("error 1"),
		errors.New("error 2"),
		errors.New("error 3"),
	}

	check := func(i int, err error) {
		switch {
		case err == nil && i == 0:
		case err == nil:
			t.Errorf("#%d: got nil, expected errors", i)
		case i == 0:
			t.Errorf("#%d: got unexpected errors: %v", i, err)
		default:
			e := err.(errlist.Error).Errors()
			if !errsEqual(errs[:i], e) {
				t.Errorf("#%d: got %v, want %v", i, e, errs[:i])
			}
		}
	}
	err.Add(nil) // should be a no-op
	check(0, err.Err())
	for i, e := range errs {
		err.Add(e)
		err.Add(nil) // should be a no-op
		check(i+1, err.Err())
	}
}

// elist implements the error interface.
type elist []error

func (e elist) Err() error { return e }
func (e elist) Error() string {
	var m []string
	for _, err := range e {
		m = append(m, err.Error())
	}
	// We use :: to join to be different from what errlist.Error will
	// use to join.
	return strings.Join(m, "::")
}

func TestAddList(t *testing.T) {
	var err, err1 errlist.List
	err.Add(err1.Err())
	if e := err.Err(); e != nil {
		t.Fatalf("got error %v, want nil", e)
	}
	err1.Add(errors.New("error1"))
	err1.Add(errors.New("error2"))
	err.Add(err1.Err())

	er := err.Err()

	switch e := er.(type) {
	case errlist.Error:
		if n := len(e.Errors()); n != 2 {
			t.Fatalf("got %d errors, want 2", n)
		}
	default:
		t.Fatalf("got error type %T, want errlist.Error", er)
	}

	var errNil error
	el := elist{nil, errors.New("error3"), errors.New("error4"), errNil}
	err.Add(el.Err())

	if got, want := err.Err().Error(), "error1, error2, error3, error4"; got != want {
		t.Fatalf("got error %q, want %q", got, want)
	}
}

func TestSep(t *testing.T) {
	var list errlist.List
	defer func(s string) { errlist.Separator = s }(errlist.Separator)
	errlist.Separator = ":"

	list.Add(errors.New("one"))
	list.Add(errors.New("two"))
	err := list.Err()
	if got, want := err.Error(), "one:two"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	list.Separator = "-"
	err = list.Err()
	if got, want := err.Error(), "one-two"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
