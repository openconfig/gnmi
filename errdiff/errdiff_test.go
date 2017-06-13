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

package errdiff

import (
	"errors"
	"io"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
)

func TestText(t *testing.T) {
	tests := []struct {
		desc   string
		got    error
		want   string
		result string
	}{
		{desc: "empty error"},
		{desc: "match", got: errors.New("abc"), want: "abc", result: ""},
		{desc: "message no match", got: errors.New("ab"), want: "abc", result: `got err=ab, want err with exact text abc`},
		{desc: "want nil", got: errors.New("ab"), result: `got err=ab, want err=nil`},
		{desc: "want nil got message", got: errors.New(""), result: `got err=, want err=nil`},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := Text(tt.got, tt.want); got != tt.result {
				t.Errorf("Text(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
		})
	}
}

func TestSubstring(t *testing.T) {
	tests := []struct {
		desc   string
		got    error
		want   string
		result string
	}{
		{desc: "empty"},
		{desc: "subsring match", got: errors.New("abc"), want: "bc", result: ""},
		{desc: "exact match", got: errors.New("abc"), want: "abc", result: ""},
		{desc: "message no match", got: errors.New("ab"), want: "abc", result: `got err=ab, want err containing abc`},
		{desc: "want nil", got: errors.New("ab"), result: `got err=ab, want err=nil`},
		{desc: "want nil got message", got: errors.New(""), result: `got err=, want err=nil`},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := Substring(tt.got, tt.want); got != tt.result {
				t.Errorf("Substring(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
			if got := Check(tt.got, tt.want); got != tt.result {
				t.Errorf("Check(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
		})
	}
}

func TestCode(t *testing.T) {
	tests := []struct {
		desc   string
		got    error
		want   codes.Code
		result string
	}{
		{desc: "empty message"},
		{
			desc:   "Unimplemented match",
			got:    grpc.Errorf(codes.Unimplemented, ""),
			want:   codes.Unimplemented,
			result: "",
		},
		{
			desc:   "code no match",
			got:    grpc.Errorf(codes.Unimplemented, ""),
			want:   codes.InvalidArgument,
			result: `got err=rpc error: code = Unimplemented desc = , want code InvalidArgument`,
		},
		{
			desc:   "nil match",
			got:    grpc.Errorf(codes.Unimplemented, ""),
			result: `got err=rpc error: code = Unimplemented desc = , want OK`,
		},
		{
			desc:   "no code",
			got:    errors.New("other"),
			want:   codes.InvalidArgument,
			result: `got err=other, want code InvalidArgument`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := Code(tt.got, tt.want); got != tt.result {
				t.Errorf("CodeCompare(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
			if got := Check(tt.got, tt.want); got != tt.result {
				t.Errorf("Check(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
		})
	}
}

type elist []string

func (e elist) Error() string { return strings.Join(e, ", ") }

func TestCheck(t *testing.T) {
	elist1 := elist{"error1a", "error1b"}
	elist2 := elist{"error2a", "error2b"}

	tests := []struct {
		desc   string
		got    error
		want   interface{}
		result string
	}{
		{desc: "empty"},
		{
			desc: "exact same error",
			got:  io.EOF,
			want: io.EOF,
		},
		{
			desc: "different errors with same string",
			want: errors.New("an error"),
			got:  errors.New("an error"),
		},
		{
			desc:   "nil want",
			got:    io.EOF,
			result: `got err=EOF, want err=nil`,
		},
		{
			desc:   "nil got",
			want:   io.EOF,
			result: `got err=nil, want err=EOF`,
		},
		{
			desc:   "different error",
			want:   errors.New("this error"),
			got:    errors.New("that error"),
			result: `got err=that error, want err=this error`,
		},
		{
			desc: "do not want error",
			want: false,
		},
		{
			desc:   "do want error",
			want:   true,
			result: `did not get expected error`,
		},
		{
			desc:   "got err=error, want err=none",
			got:    io.EOF,
			want:   false,
			result: `got err=EOF, want err=nil`,
		},
		{
			desc: "got error, want error",
			got:  io.EOF,
			want: true,
		},
		{
			desc:   "unexpected errlist",
			got:    elist1,
			result: `got err=error1a, error1b, want err=nil`,
		},
		{
			desc:   "missing errlist",
			want:   elist1,
			result: `got err=nil, want err=error1a, error1b`,
		},
		{
			desc: "correct errlist",
			got:  elist1,
			want: elist1,
		},
		{
			desc:   "wrong errlist",
			got:    elist1,
			want:   elist2,
			result: `got err=error1a, error1b, want err=error2a, error2b`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := Check(tt.got, tt.want); got != tt.result {
				t.Errorf("Check(%v, %v): got=%q want=%q", tt.got, tt.want, got, tt.result)
			}
		})
	}
}
