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

package coalesce

import "testing"

func TestInsert(t *testing.T) {
	q := NewQueue()
	tests := []struct {
		s    string
		want bool
		size int
	}{
		{"hello", true, 1},
		{"hello", false, 1},
		{"world", true, 2},
		{"hello", false, 2},
		{"world", false, 2},
		{"world", false, 2},
		{"!", true, 3},
	}
	for x, tt := range tests {
		if got, _ := q.Insert(tt.s); got != tt.want {
			t.Errorf("#%d: q.Insert(%q): got %t, want %t", x, tt.s, got, tt.want)
		}
		if size := q.Len(); size != tt.size {
			t.Errorf("#%d: got queue length %d, want %d", x, size, tt.size)
		}
	}
}

func TestNext(t *testing.T) {
	q := NewQueue()
	type stringCount struct {
		s     string
		count int
	}
	tests := []struct {
		s    []string
		want []stringCount
	}{
		{[]string{"hello"}, []stringCount{{"hello", 0}}},
		{[]string{"hello", "hello", "hello"}, []stringCount{{"hello", 2}}},
		{[]string{"hello", "world", "hello"}, []stringCount{{"hello", 1}, {"world", 0}}},
		{[]string{"hello"}, []stringCount{{"hello", 0}}},
	}
	for x, tt := range tests {
		for _, s := range tt.s {
			q.Insert(s)
		}
		for _, want := range tt.want {
			i, count, err := q.Next()
			if err != nil {
				t.Errorf("#%d: q.Next() unexpected error %v", x, err)
				continue
			}
			s, ok := i.(string)
			if !ok {
				t.Errorf("#%d: unable to cast to string %v", x, i)
				continue
			}
			if s != want.s || count != want.count {
				t.Errorf("#%d: q.Next(): got %q (%d) want %q (%d)", x, s, count, want.s, want.count)
			}
		}
	}
}

func TestClose(t *testing.T) {
	for _, tt := range [][]string{
		{},
		{"a"},
		{"a", "b"},
		{"a", "b", "c"},
	} {
		q := NewQueue()
		if q.IsClosed() {
			t.Error("newly created queue IsClosed() got true, want false")
		}
		for _, s := range tt {
			_, err := q.Insert(s)
			if err != nil {
				t.Fatal(err)
			}
		}
		q.Close()
		if !q.IsClosed() {
			t.Error("closed queue IsClosed() got false, want true")
		}
		if _, err := q.Insert("foo"); !IsClosedQueue(err) {
			t.Errorf("q.Insert() got %v, want %v", err, errClosedQueue)
		}
		for x, want := range tt {
			i, _, err := q.Next()
			got, ok := i.(string)
			switch {
			case err != nil:
				t.Errorf("#%d: q.Next(): got error %v before result %q received.", x, err, want)
			case !ok || got != want:
				t.Errorf("#%d: q.Next(): got %q, want %q", x, got, want)
			}
		}
		if _, _, err := q.Next(); !IsClosedQueue(err) {
			t.Errorf("q.Next(): got %v, want %v", err, errClosedQueue)
		}
		if _, err := q.Insert("foo"); !IsClosedQueue(err) {
			t.Errorf("q.Insert() got %v, want %v", err, errClosedQueue)
		}
	}
}
