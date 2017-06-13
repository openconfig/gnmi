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

// Package errdiff makes it easy to compare Error by code, substring or exact
// match in tests.
//
// Similar in intended usage to messagediff.Diff and pretty.Compare,
// particularly in table-driven tests.
//
// Example usage:
//
//	testCases := []struct {
//		...
//		wantSubstring string
//	}{
//		// Success
//		{...},
//		// Failures
//		{..., wantSubstring: "failed"},
//		{..., wantSubstring: "too many users"},
//	}
//	for _, c := range testCases {
//		got, err := fn(...)
//		if diff := errdiff.Substring(err, c.wantSubstring); diff != "" {
//			t.Errorf("fn() %v", diff)
//			continue
//		}
//		...
//	}
//
// The generic function Check may be used in place of Code or
// Substring or when comparing against another error or for simple
// existance of an error:
//
//	testCases := []struct {
//		...
//		err interface{}
//	}{
//		// Success
//		{...},
//		// Failures
//		{..., err: io.EOF},  // An explicit error
//		{..., err: "my expected error string"},  // contains text
//		{..., err: true}, // expect an error, don't care what
//	}
//	for _, c := range testCases {
//		got, err := fn(...)
//		if diff := errdiff.Check(err, c.err); diff != "" {
//			t.Errorf("fn() %v", diff)
//			continue
//		}
//		...
package errdiff

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
)

// Text returns a message describing the difference between the
// error text and the desired input. want="" indicates that no error is
// expected.
func Text(got error, want string) string {
	if want == "" {
		if got == nil {
			return ""
		}
		return fmt.Sprintf("got err=%v, want err=nil", got)
	}
	if got == nil {
		return fmt.Sprintf("got err=nil, want err with exact text %v", want)
	}
	if got.Error() != want {
		return fmt.Sprintf("got err=%v, want err with exact text %v", got, want)
	}
	return ""
}

// Substring returns a message describing the difference between the
// error text and the desired input. want="" indicates that no error is
// expected.
func Substring(got error, want string) string {
	if want == "" {
		if got == nil {
			return ""
		}
		return fmt.Sprintf("got err=%v, want err=nil", got)
	}
	if got == nil {
		return fmt.Sprintf("got err=nil, want err containing %v", want)
	}
	if !strings.Contains(got.Error(), want) {
		return fmt.Sprintf("got err=%v, want err containing %v", got, want)
	}
	return ""
}

// Code returns a message describing the difference between the error's
// code and the desired codes. want=codes.OK indicates that no error is
// expected.
func Code(got error, want codes.Code) string {
	if want == codes.OK {
		if got == nil {
			return ""
		}
		return fmt.Sprintf("got err=%v, want OK", got)
	}
	if got == nil {
		return fmt.Sprintf("got err=nil, want code %v", want)
	}
	if grpc.Code(got) != want {
		return fmt.Sprintf("got err=%v, want code %v", got, want)
	}
	return ""
}

// Check returns a message describing the difference between the error err and
// want.  If want is a codes.Code, this function is the same as Code.
// If want is a string, this function is the same as Substring.  If
// want is an error, this is essentially the same as ExactTextCompare(got,
// w.Error()).  If want is a bool, err is simply tested for existance (want of
// true means an error is wanted).
func Check(got error, want interface{}) string {
	switch w := want.(type) {
	case nil:
		if got == nil {
			return ""
		}
		return fmt.Sprintf("got err=%v, want err=nil", got)
	case bool:
		switch {
		case w && got == nil:
			return "did not get expected error"
		case !w && got != nil:
			return fmt.Sprintf("got err=%v, want err=nil", got)
		}
		return ""
	case codes.Code:
		return Code(got, w)
	case string:
		return Substring(got, w)
	case error:
		switch {
		case got == nil && w == nil:
			return ""
		case got == nil && w != nil:
			return fmt.Sprintf("got err=nil, want err=%v", w)
		case w == nil:
			return fmt.Sprintf("got err=%v, want err=nil", got)
		case got.Error() == w.Error():
			return ""
		default:
			return fmt.Sprintf("got err=%v, want err=%v", got, w)
		}
	default:
		return fmt.Sprintf("unsupported type in Check: %T", want)
	}
}
