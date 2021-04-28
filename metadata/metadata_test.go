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

package metadata

import "testing"

func TestPath(t *testing.T) {
	if path := Path("invalid"); path != nil {
		t.Errorf("Path(%q) returned %v for invalid value.", "invalid", path)
	}
	for value := range TargetBoolValues {
		if path := Path(value); path == nil {
			t.Errorf("Path(%q) returned nil for valid value.", value)
		}
	}
	for value := range TargetIntValues {
		if path := Path(value); path == nil {
			t.Errorf("Path(%q) returned nil for valid value.", value)
		}
	}
	for value := range TargetStrValues {
		if path := Path(value); path == nil {
			t.Errorf("Path(%q) returned nil for valid value.", value)
		}
	}
}

func TestGetInt(t *testing.T) {
	m := New()
	for value := range TargetIntValues {
		v, err := m.GetInt(value)
		switch {
		case err != nil:
			t.Errorf("GetInt(%q) got error: %q, want nil", value, err)
		case v != 0:
			t.Errorf("GetInt(%q) got %d for uninitialized value, want 0", value, v)
		}
	}
}

func TestGetBool(t *testing.T) {
	m := New()
	for value := range TargetBoolValues {
		v, err := m.GetBool(value)
		switch {
		case err != nil:
			t.Errorf("GetBool(%q) got error: %q, want nil", value, err)
		case v != false:
			t.Errorf("GetBool(%q) got %t for uninitialized value, want false", value, v)
		}
	}
}

func TestGetStr(t *testing.T) {
	m := New()
	for k, val := range TargetStrValues {
		if !val.InitEmptyStr {
			m.SetStr(k, "")
		}
		v, err := m.GetStr(k)
		switch {
		case err != nil:
			t.Errorf("GetStr(%q) got error: %q, want nil", k, err)
		case v != "":
			t.Errorf("GetStr(%q) got %q for uninitialized value, want empty string", k, v)
		}
	}
}

func TestAddGetInt(t *testing.T) {
	m := New()
	if err := m.AddInt("invalid", 1); err != ErrInvalidValue {
		t.Error("AddInt accepted invalid metadata value.")
	}
	for i := 1; i < 6; i++ {
		for value := range TargetIntValues {
			if err := m.AddInt(value, 1); err != nil {
				t.Errorf("AddInt(%q, 1) returned error: %v", value, err)
			}
			got, err := m.GetInt(value)
			if err != nil {
				t.Errorf("GetInt(%q) returned error: %v", value, err)
				continue
			}
			want := int64(i)
			if got != want {
				t.Errorf("%d: GetInt(%q): got %d, want %d", i, value, got, want)
			}
		}
	}
	if _, err := m.GetInt("invalid"); err != ErrInvalidValue {
		t.Error("GetInt accepted invalid metadata value.")
	}
}

func TestSetGetInt(t *testing.T) {
	m := New()
	if err := m.SetInt("invalid", 1); err != ErrInvalidValue {
		t.Error("SetInt accepted invalid metadata value.")
	}
	for i := 0; i < 10; i++ {
		var x int
		for value := range TargetIntValues {
			x++
			want := int64(x + i)
			if err := m.SetInt(value, want); err != nil {
				t.Errorf("SetInt(%q, %d) returned error: %v", value, want, err)
			}
			got, err := m.GetInt(value)
			if err != nil {
				t.Errorf("GetInt(%q) returned error: %v", value, err)
			}
			if got != want {
				t.Errorf("GetInt(%q): got %d, want %d", value, got, want)
			}
		}
	}
}

func TestSetGetBool(t *testing.T) {
	m := New()
	if err := m.SetBool("invalid", true); err != ErrInvalidValue {
		t.Error("SetBool accepted invalid metadata value.")
	}
	for _, want := range []bool{true, false, true, false} {
		for value := range TargetBoolValues {
			if err := m.SetBool(value, want); err != nil {
				t.Errorf("SetBool(%q, %t) returned error: %v", value, want, err)
			}
			got, err := m.GetBool(value)
			if err != nil {
				t.Errorf("GetBool(%q) returned error: %v", value, err)
			}
			if got != want {
				t.Errorf("GetBool(%q): got %t, want %t", value, got, want)
			}
		}
	}
	if _, err := m.GetBool("invalid"); err != ErrInvalidValue {
		t.Error("GetBool accepted invalid metadata value.")
	}
}

func TestSetGetStr(t *testing.T) {
	m := New()
	if err := m.SetStr("invalid", "invalidValue"); err != ErrInvalidValue {
		t.Error("SetStr accepted invalid metadata value.")
	}
	for _, want := range []string{"value1", "value2", "value3", "value4"} {
		for value := range TargetStrValues {
			if err := m.SetStr(value, want); err != nil {
				t.Errorf("SetStr(%q, %q) returned error: %v", value, want, err)
			}
			got, err := m.GetStr(value)
			if err != nil {
				t.Errorf("GetStr(%q) returned error: %v", value, err)
			}
			if got != want {
				t.Errorf("GetStr(%q): got %q, want %q", value, got, want)
			}
		}
	}
	if _, err := m.GetStr("invalid"); err != ErrInvalidValue {
		t.Error("GetStr accepted invalid metadata value.")
	}
}

func TestResetEntry(t *testing.T) {
	m := New()

	for k := range TargetBoolValues {
		m.SetBool(k, true)
		m.ResetEntry(k)
		v, err := m.GetBool(k)
		if err != nil {
			t.Errorf("ResetEntry for %q failed with error: %v", k, err)
		}
		if v != false {
			t.Errorf("ResetEntry for %q failed. got %t, want %t", k, false, v)
		}
	}

	for k, val := range TargetIntValues {
		m.SetInt(k, 1)
		m.ResetEntry(k)
		v, err := m.GetInt(k)

		if !val.InitZero {
			if err == nil {
				t.Errorf("ResetEntry for %q failed. Expect not exist, but found.", k)
			}
		} else {
			if err != nil {
				t.Errorf("ResetEntry for %q failed with error: %v", k, err)
			}
			if v != 0 {
				t.Errorf("ResetEntry for %q failed. got %q, want %q", k, 0, v)
			}
		}

	}

	for k, val := range TargetStrValues {
		m.SetStr(k, "xx")
		m.ResetEntry(k)
		v, err := m.GetStr(k)

		if !val.InitEmptyStr {
			if err == nil {
				t.Errorf("ResetEntry for %q failed. Expect not exist, but found.", k)
			}
		} else {
			if err != nil {
				t.Errorf("ResetEntry for %q failed with error: %v", k, err)
			}
			if v != "" {
				t.Errorf("ResetEntry for %q failed. got %q, want %q", k, "", v)
			}
		}

	}

	if err := m.ResetEntry("unsupported"); err == nil {
		t.Errorf("ResetEntry expected error, but got nil")
	}
}

func TestClear(t *testing.T) {
	m := New()
	m.Clear()

	for k := range TargetBoolValues {
		v, err := m.GetBool(k)
		if err != nil {
			t.Errorf("ResetEntry for %q failed with error %t", k, err)
		}
		if v {
			t.Errorf("ResetEntry for %q failed. got %t, want %t", k, false, v)
		}
	}

	for k, val := range TargetIntValues {
		v, err := m.GetInt(k)
		if !val.InitZero {
			if err == nil {
				t.Errorf("ResetEntry for %q failed. Expect not exist, but found.", k)
			}
		} else {
			if err != nil {
				t.Errorf("ResetEntry for %q failed with error: %v", k, err)
			}
			if v != 0 {
				t.Errorf("ResetEntry for %q failed. got %q, want %q", k, 0, v)
			}
		}

	}

	for k, val := range TargetStrValues {
		v, err := m.GetStr(k)
		if !val.InitEmptyStr {
			if err == nil {
				t.Errorf("ResetEntry for %q failed. Expect not exist, but found.", k)
			}
		} else {
			if err != nil {
				t.Errorf("ResetEntry for %q failed with error: %v", k, err)
			}
			if v != "" {
				t.Errorf("ResetEntry for %q failed. got %q, want %q", k, "", v)
			}
		}

	}
}
