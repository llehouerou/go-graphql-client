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

func TestUnwrapToConcreteValue(t *testing.T) {
	intValue := 42
	strValue := "hello"
	intPtr := &intValue
	intPtrPtr := &intPtr

	tests := []struct {
		name      string
		value     reflect.Value
		wantValid bool
		wantKind  reflect.Kind
		wantValue any
	}{
		{
			name:      "concrete int value",
			value:     reflect.ValueOf(intValue),
			wantValid: true,
			wantKind:  reflect.Int,
			wantValue: 42,
		},
		{
			name:      "pointer to int",
			value:     reflect.ValueOf(&intValue),
			wantValid: true,
			wantKind:  reflect.Int,
			wantValue: 42,
		},
		{
			name:      "pointer to pointer to int",
			value:     reflect.ValueOf(&intPtrPtr),
			wantValid: true,
			wantKind:  reflect.Int,
			wantValue: 42,
		},
		{
			name:      "interface containing string",
			value:     reflect.ValueOf(any(strValue)),
			wantValid: true,
			wantKind:  reflect.String,
			wantValue: "hello",
		},
		{
			name:      "nil pointer",
			value:     reflect.ValueOf((*int)(nil)),
			wantValid: false,
		},
		{
			name:      "invalid value",
			value:     reflect.ValueOf(nil),
			wantValid: false,
		},
		{
			name:      "nil interface",
			value:     reflect.ValueOf((any)(nil)),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnwrapToConcreteValue(tt.value)
			if got.IsValid() != tt.wantValid {
				t.Errorf(
					"UnwrapToConcreteValue() validity = %v, want %v",
					got.IsValid(),
					tt.wantValid,
				)
				return
			}
			if tt.wantValid {
				if got.Kind() != tt.wantKind {
					t.Errorf(
						"UnwrapToConcreteValue() kind = %v, want %v",
						got.Kind(),
						tt.wantKind,
					)
				}
				if got.Interface() != tt.wantValue {
					t.Errorf(
						"UnwrapToConcreteValue() = %v, want %v",
						got.Interface(),
						tt.wantValue,
					)
				}
			}
		})
	}
}

func TestIsNilValue(t *testing.T) {
	intValue := 42
	var nilPtr *int
	var nilSlice []int
	var nilMap map[string]int
	var nilInterface any
	var nilChan chan int
	var nilFunc func()

	tests := []struct {
		name string
		value reflect.Value
		want bool
	}{
		// Nil values
		{
			name:  "nil pointer",
			value: reflect.ValueOf(nilPtr),
			want:  true,
		},
		{
			name:  "nil slice",
			value: reflect.ValueOf(nilSlice),
			want:  true,
		},
		{
			name:  "nil map",
			value: reflect.ValueOf(nilMap),
			want:  true,
		},
		{
			name:  "nil interface",
			value: reflect.ValueOf(nilInterface),
			want:  true,
		},
		{
			name:  "nil channel",
			value: reflect.ValueOf(nilChan),
			want:  true,
		},
		{
			name:  "nil func",
			value: reflect.ValueOf(nilFunc),
			want:  true,
		},
		{
			name:  "invalid value",
			value: reflect.Value{},
			want:  true,
		},

		// Non-nil values
		{
			name:  "valid pointer",
			value: reflect.ValueOf(&intValue),
			want:  false,
		},
		{
			name:  "valid slice",
			value: reflect.ValueOf([]int{1, 2, 3}),
			want:  false,
		},
		{
			name:  "valid map",
			value: reflect.ValueOf(map[string]int{"a": 1}),
			want:  false,
		},
		{
			name:  "valid interface",
			value: reflect.ValueOf(any(42)),
			want:  false,
		},
		{
			name:  "valid channel",
			value: reflect.ValueOf(make(chan int)),
			want:  false,
		},
		{
			name:  "valid func",
			value: reflect.ValueOf(func() {}),
			want:  false,
		},

		// Non-nillable types (should always return false)
		{
			name:  "int value",
			value: reflect.ValueOf(42),
			want:  false,
		},
		{
			name:  "string value",
			value: reflect.ValueOf("hello"),
			want:  false,
		},
		{
			name:  "bool value",
			value: reflect.ValueOf(true),
			want:  false,
		},
		{
			name:  "struct value",
			value: reflect.ValueOf(struct{ X int }{X: 10}),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNilValue(tt.value); got != tt.want {
				t.Errorf("IsNilValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnwrapToConcreteValue_Integration(t *testing.T) {
	// This test shows how UnwrapToConcreteValue and IsNilValue work together,
	// which is a common pattern in the codebase
	t.Run("unwrap and check nil on valid pointer", func(t *testing.T) {
		val := 42
		ptr := &val
		v := reflect.ValueOf(ptr)

		// First unwrap, then check if nil
		unwrapped := UnwrapToConcreteValue(v)
		if IsNilValue(unwrapped) {
			t.Error("expected non-nil value after unwrapping")
		}
		if unwrapped.Kind() != reflect.Int {
			t.Errorf("expected int kind, got %v", unwrapped.Kind())
		}
		if unwrapped.Interface().(int) != 42 {
			t.Errorf("expected 42, got %v", unwrapped.Interface())
		}
	})

	t.Run("unwrap and check nil on nil pointer", func(t *testing.T) {
		var ptr *int
		v := reflect.ValueOf(ptr)

		// Unwrap returns invalid value for nil pointer
		unwrapped := UnwrapToConcreteValue(v)
		if unwrapped.IsValid() {
			t.Error("expected invalid value when unwrapping nil pointer")
		}
		// IsNilValue should return true for invalid values
		if !IsNilValue(unwrapped) {
			t.Error("expected IsNilValue to return true for invalid value")
		}
	})

	t.Run("create pointer and unwrap", func(t *testing.T) {
		// Create a new pointer using NewZeroOrPointerValue
		typ := reflect.TypeOf((*string)(nil))
		v := NewZeroOrPointerValue(typ)

		// Set a value
		v.Elem().SetString("test")

		// Unwrap to get the concrete value
		unwrapped := UnwrapToConcreteValue(v)
		if unwrapped.Kind() != reflect.String {
			t.Errorf("expected string kind, got %v", unwrapped.Kind())
		}
		if unwrapped.Interface().(string) != "test" {
			t.Errorf("expected 'test', got %v", unwrapped.Interface())
		}
	})
}

func TestNewZeroOrPointerValue(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		wantKind reflect.Kind
		check    func(t *testing.T, v reflect.Value)
	}{
		{
			name:     "pointer type creates new pointer",
			typ:      reflect.TypeOf((*int)(nil)),
			wantKind: reflect.Ptr,
			check: func(t *testing.T, v reflect.Value) {
				if v.IsNil() {
					t.Error("expected non-nil pointer")
				}
				if v.Elem().Kind() != reflect.Int {
					t.Errorf("pointer elem kind = %v, want %v", v.Elem().Kind(), reflect.Int)
				}
			},
		},
		{
			name:     "pointer to string type",
			typ:      reflect.TypeOf((*string)(nil)),
			wantKind: reflect.Ptr,
			check: func(t *testing.T, v reflect.Value) {
				if v.IsNil() {
					t.Error("expected non-nil pointer")
				}
				if v.Elem().Kind() != reflect.String {
					t.Errorf(
						"pointer elem kind = %v, want %v",
						v.Elem().Kind(),
						reflect.String,
					)
				}
			},
		},
		{
			name:     "pointer to struct type",
			typ:      reflect.TypeOf((*struct{ X int })(nil)),
			wantKind: reflect.Ptr,
			check: func(t *testing.T, v reflect.Value) {
				if v.IsNil() {
					t.Error("expected non-nil pointer")
				}
				if v.Elem().Kind() != reflect.Struct {
					t.Errorf(
						"pointer elem kind = %v, want %v",
						v.Elem().Kind(),
						reflect.Struct,
					)
				}
			},
		},
		{
			name:     "int type creates zero value",
			typ:      reflect.TypeOf(0),
			wantKind: reflect.Int,
			check: func(t *testing.T, v reflect.Value) {
				if v.Interface().(int) != 0 {
					t.Errorf("int value = %v, want 0", v.Interface())
				}
			},
		},
		{
			name:     "string type creates zero value",
			typ:      reflect.TypeOf(""),
			wantKind: reflect.String,
			check: func(t *testing.T, v reflect.Value) {
				if v.Interface().(string) != "" {
					t.Errorf("string value = %v, want empty string", v.Interface())
				}
			},
		},
		{
			name:     "bool type creates zero value",
			typ:      reflect.TypeOf(false),
			wantKind: reflect.Bool,
			check: func(t *testing.T, v reflect.Value) {
				if v.Interface().(bool) != false {
					t.Errorf("bool value = %v, want false", v.Interface())
				}
			},
		},
		{
			name:     "struct type creates zero value",
			typ:      reflect.TypeOf(struct{ X int }{}),
			wantKind: reflect.Struct,
			check: func(t *testing.T, v reflect.Value) {
				s := v.Interface().(struct{ X int })
				if s.X != 0 {
					t.Errorf("struct field X = %v, want 0", s.X)
				}
			},
		},
		{
			name:     "slice type creates zero value",
			typ:      reflect.TypeOf([]int{}),
			wantKind: reflect.Slice,
			check: func(t *testing.T, v reflect.Value) {
				if !v.IsNil() {
					t.Error("expected nil slice")
				}
			},
		},
		{
			name:     "map type creates zero value",
			typ:      reflect.TypeOf(map[string]int{}),
			wantKind: reflect.Map,
			check: func(t *testing.T, v reflect.Value) {
				if !v.IsNil() {
					t.Error("expected nil map")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewZeroOrPointerValue(tt.typ)
			if !got.IsValid() {
				t.Fatal("NewZeroOrPointerValue() returned invalid value")
			}
			if got.Kind() != tt.wantKind {
				t.Errorf(
					"NewZeroOrPointerValue() kind = %v, want %v",
					got.Kind(),
					tt.wantKind,
				)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
