package reflectutil

import (
	"reflect"
	"testing"
)

func TestIndexSafe(t *testing.T) {
	tests := []struct {
		name      string
		value     reflect.Value
		index     int
		wantValid bool
		wantValue any
	}{
		{
			name:      "valid slice with valid index",
			value:     reflect.ValueOf([]int{1, 2, 3}),
			index:     1,
			wantValid: true,
			wantValue: 2,
		},
		{
			name:      "valid slice with index 0",
			value:     reflect.ValueOf([]string{"a", "b", "c"}),
			index:     0,
			wantValid: true,
			wantValue: "a",
		},
		{
			name:      "valid slice with last index",
			value:     reflect.ValueOf([]int{10, 20, 30}),
			index:     2,
			wantValid: true,
			wantValue: 30,
		},
		{
			name:      "index out of bounds",
			value:     reflect.ValueOf([]int{1, 2}),
			index:     5,
			wantValid: false,
		},
		{
			name:      "negative index",
			value:     reflect.ValueOf([]int{1, 2, 3}),
			index:     -1,
			wantValid: false,
		},
		{
			name:      "invalid value",
			value:     reflect.ValueOf(nil),
			index:     0,
			wantValid: false,
		},
		{
			name:      "empty slice valid index 0",
			value:     reflect.ValueOf([]int{}),
			index:     0,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IndexSafe(tt.value, tt.index)
			if got.IsValid() != tt.wantValid {
				t.Errorf(
					"IndexSafe() validity = %v, want %v",
					got.IsValid(),
					tt.wantValid,
				)
				return
			}
			if tt.wantValid && got.Interface() != tt.wantValue {
				t.Errorf(
					"IndexSafe() = %v, want %v",
					got.Interface(),
					tt.wantValue,
				)
			}
		})
	}
}

func TestElemSafe(t *testing.T) {
	intValue := 42
	strValue := "hello"

	tests := []struct {
		name      string
		value     reflect.Value
		wantValid bool
		wantValue any
	}{
		{
			name:      "valid pointer to int",
			value:     reflect.ValueOf(&intValue),
			wantValid: true,
			wantValue: 42,
		},
		{
			name:      "valid pointer to string",
			value:     reflect.ValueOf(&strValue),
			wantValid: true,
			wantValue: "hello",
		},
		{
			name:      "non-pointer non-interface value",
			value:     reflect.ValueOf(100),
			wantValid: false,
		},
		{
			name:      "invalid value",
			value:     reflect.ValueOf(nil),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ElemSafe(tt.value)
			if got.IsValid() != tt.wantValid {
				t.Errorf(
					"ElemSafe() validity = %v, want %v",
					got.IsValid(),
					tt.wantValid,
				)
				return
			}
			if tt.wantValid && got.Interface() != tt.wantValue {
				t.Errorf("ElemSafe() = %v, want %v", got.Interface(), tt.wantValue)
			}
		})
	}
}

func TestFieldSafe(t *testing.T) {
	type testStruct struct {
		Field1 string
		Field2 int
		Field3 bool
	}

	s := testStruct{
		Field1: "test",
		Field2: 123,
		Field3: true,
	}

	tests := []struct {
		name      string
		value     reflect.Value
		index     int
		wantValid bool
		wantValue any
	}{
		{
			name:      "valid struct field 0",
			value:     reflect.ValueOf(s),
			index:     0,
			wantValid: true,
			wantValue: "test",
		},
		{
			name:      "valid struct field 1",
			value:     reflect.ValueOf(s),
			index:     1,
			wantValid: true,
			wantValue: 123,
		},
		{
			name:      "valid struct field 2",
			value:     reflect.ValueOf(s),
			index:     2,
			wantValid: true,
			wantValue: true,
		},
		{
			name:      "invalid value",
			value:     reflect.ValueOf(nil),
			index:     0,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FieldSafe(tt.value, tt.index)
			if got.IsValid() != tt.wantValid {
				t.Errorf(
					"FieldSafe() validity = %v, want %v",
					got.IsValid(),
					tt.wantValid,
				)
				return
			}
			if tt.wantValid && got.Interface() != tt.wantValue {
				t.Errorf(
					"FieldSafe() = %v, want %v",
					got.Interface(),
					tt.wantValue,
				)
			}
		})
	}
}

func TestIsNillable(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Kind
		want bool
	}{
		// Nillable types
		{name: "ptr", kind: reflect.Ptr, want: true},
		{name: "interface", kind: reflect.Interface, want: true},
		{name: "slice", kind: reflect.Slice, want: true},
		{name: "map", kind: reflect.Map, want: true},
		{name: "chan", kind: reflect.Chan, want: true},
		{name: "func", kind: reflect.Func, want: true},

		// Non-nillable types
		{name: "bool", kind: reflect.Bool, want: false},
		{name: "int", kind: reflect.Int, want: false},
		{name: "int8", kind: reflect.Int8, want: false},
		{name: "int16", kind: reflect.Int16, want: false},
		{name: "int32", kind: reflect.Int32, want: false},
		{name: "int64", kind: reflect.Int64, want: false},
		{name: "uint", kind: reflect.Uint, want: false},
		{name: "uint8", kind: reflect.Uint8, want: false},
		{name: "uint16", kind: reflect.Uint16, want: false},
		{name: "uint32", kind: reflect.Uint32, want: false},
		{name: "uint64", kind: reflect.Uint64, want: false},
		{name: "uintptr", kind: reflect.Uintptr, want: false},
		{name: "float32", kind: reflect.Float32, want: false},
		{name: "float64", kind: reflect.Float64, want: false},
		{name: "complex64", kind: reflect.Complex64, want: false},
		{name: "complex128", kind: reflect.Complex128, want: false},
		{name: "array", kind: reflect.Array, want: false},
		{name: "string", kind: reflect.String, want: false},
		{name: "struct", kind: reflect.Struct, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNillable(tt.kind); got != tt.want {
				t.Errorf("IsNillable(%v) = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}
