package reflectutil

import (
	"reflect"
	"testing"
)

// Test types for wrapper pattern
type TestWrapper[T any] struct {
	Value T
}

func (w TestWrapper[T]) GetGraphQLWrapped() T {
	return w.Value
}

type TestWrapperNoValueField struct {
	Data string
}

func (w TestWrapperNoValueField) GetGraphQLWrapped() string {
	return w.Data
}

// Test types for GraphQLType interface
type CustomType struct {
	Data string
}

func (c CustomType) GetGraphQLType() string {
	return "CustomScalar"
}

type CustomPointerType struct {
	Data string
}

func (c *CustomPointerType) GetGraphQLType() string {
	return "CustomPointerScalar"
}

// Regular struct without special interfaces
type RegularStruct struct {
	Field string
}

func TestImplementsGraphQLType(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{
			name:     "CustomType implements GraphQLType",
			typ:      reflect.TypeOf(CustomType{}),
			expected: true,
		},
		{
			name:     "Pointer to CustomPointerType implements GraphQLType",
			typ:      reflect.TypeOf(&CustomPointerType{}),
			expected: true,
		},
		{
			name:     "RegularStruct does not implement GraphQLType",
			typ:      reflect.TypeOf(RegularStruct{}),
			expected: false,
		},
		{
			name:     "int does not implement GraphQLType",
			typ:      reflect.TypeOf(0),
			expected: false,
		},
		{
			name:     "string does not implement GraphQLType",
			typ:      reflect.TypeOf(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ImplementsGraphQLType(tt.typ)
			if result != tt.expected {
				t.Errorf(
					"ImplementsGraphQLType() = %v, want %v",
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestIsWrapperType(t *testing.T) {
	wrapper := TestWrapper[string]{Value: "test"}
	pointerWrapper := &TestWrapper[int]{Value: 42}
	noValueField := TestWrapperNoValueField{Data: "test"}
	regular := RegularStruct{Field: "test"}

	tests := []struct {
		name     string
		value    reflect.Value
		expected bool
	}{
		{
			name:     "TestWrapper is a wrapper type",
			value:    reflect.ValueOf(wrapper),
			expected: true,
		},
		{
			name:     "Pointer to TestWrapper is a wrapper type",
			value:    reflect.ValueOf(pointerWrapper),
			expected: true,
		},
		{
			name:     "TestWrapperNoValueField is a wrapper type (has method)",
			value:    reflect.ValueOf(noValueField),
			expected: true,
		},
		{
			name:     "RegularStruct is not a wrapper type",
			value:    reflect.ValueOf(regular),
			expected: false,
		},
		{
			name:     "Invalid value is not a wrapper type",
			value:    reflect.Value{},
			expected: false,
		},
		{
			name:     "Nil pointer is not a wrapper type",
			value:    reflect.ValueOf((*TestWrapper[string])(nil)),
			expected: false,
		},
		{
			name:     "int is not a wrapper type",
			value:    reflect.ValueOf(42),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWrapperType(tt.value)
			if result != tt.expected {
				t.Errorf("IsWrapperType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUnwrapValue(t *testing.T) {
	wrapper := TestWrapper[string]{Value: "test"}
	pointerWrapper := &TestWrapper[int]{Value: 42}
	regular := RegularStruct{Field: "test"}

	tests := []struct {
		name      string
		value     reflect.Value
		wantValid bool
		wantValue any
	}{
		{
			name:      "Unwrap TestWrapper calls GetGraphQLWrapped",
			value:     reflect.ValueOf(wrapper),
			wantValid: true,
			wantValue: "test",
		},
		{
			name:      "Unwrap pointer to TestWrapper calls GetGraphQLWrapped",
			value:     reflect.ValueOf(pointerWrapper),
			wantValid: true,
			wantValue: 42,
		},
		{
			name:      "Unwrap regular struct returns invalid value",
			value:     reflect.ValueOf(regular),
			wantValid: false,
			wantValue: nil,
		},
		{
			name:      "Unwrap invalid value returns invalid value",
			value:     reflect.Value{},
			wantValid: false,
			wantValue: nil,
		},
		{
			name:      "Unwrap nil pointer returns invalid value",
			value:     reflect.ValueOf((*TestWrapper[string])(nil)),
			wantValid: false,
			wantValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnwrapValue(tt.value)
			if result.IsValid() != tt.wantValid {
				t.Errorf(
					"UnwrapValue().IsValid() = %v, want %v",
					result.IsValid(),
					tt.wantValid,
				)
			}
			if tt.wantValid && result.Interface() != tt.wantValue {
				t.Errorf(
					"UnwrapValue().Interface() = %v, want %v",
					result.Interface(),
					tt.wantValue,
				)
			}
		})
	}
}

func TestUnwrapValueField(t *testing.T) {
	wrapper := TestWrapper[string]{Value: "test"}
	pointerWrapper := &TestWrapper[int]{Value: 42}
	regular := RegularStruct{Field: "test"}

	tests := []struct {
		name      string
		value     reflect.Value
		wantValid bool
		wantValue any
	}{
		{
			name:      "Unwrap TestWrapper returns Value field",
			value:     reflect.ValueOf(wrapper),
			wantValid: true,
			wantValue: "test",
		},
		{
			name:      "Unwrap pointer to TestWrapper returns Value field",
			value:     reflect.ValueOf(pointerWrapper),
			wantValid: true,
			wantValue: 42,
		},
		{
			name:      "Unwrap regular struct returns invalid value",
			value:     reflect.ValueOf(regular),
			wantValid: false,
			wantValue: nil,
		},
		{
			name:      "Unwrap invalid value returns invalid value",
			value:     reflect.Value{},
			wantValid: false,
			wantValue: nil,
		},
		{
			name:      "Unwrap nil pointer returns invalid value",
			value:     reflect.ValueOf((*TestWrapper[string])(nil)),
			wantValid: false,
			wantValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnwrapValueField(tt.value)
			if result.IsValid() != tt.wantValid {
				t.Errorf(
					"UnwrapValueField().IsValid() = %v, want %v",
					result.IsValid(),
					tt.wantValid,
				)
			}
			if tt.wantValid && result.Interface() != tt.wantValue {
				t.Errorf(
					"UnwrapValueField().Interface() = %v, want %v",
					result.Interface(),
					tt.wantValue,
				)
			}
		})
	}
}

func TestUnwrapValueOrOriginal(t *testing.T) {
	wrapper := TestWrapper[string]{Value: "test"}
	regular := RegularStruct{Field: "test"}

	tests := []struct {
		name      string
		value     reflect.Value
		wantValue any
	}{
		{
			name:      "Wrapper returns unwrapped value",
			value:     reflect.ValueOf(wrapper),
			wantValue: "test",
		},
		{
			name:      "Regular struct returns original value",
			value:     reflect.ValueOf(regular),
			wantValue: regular,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnwrapValueOrOriginal(tt.value)
			if !result.IsValid() {
				t.Error("UnwrapValueOrOriginal() returned invalid value")
			}
			if result.Interface() != tt.wantValue {
				t.Errorf(
					"UnwrapValueOrOriginal() = %v, want %v",
					result.Interface(),
					tt.wantValue,
				)
			}
		})
	}
}

func TestGetGraphQLType(t *testing.T) {
	customType := CustomType{Data: "test"}
	customPointerType := &CustomPointerType{Data: "test"}
	regular := RegularStruct{Field: "test"}

	tests := []struct {
		name         string
		value        reflect.Value
		typ          reflect.Type
		wantTypeName string
		wantOk       bool
	}{
		{
			name:         "CustomType returns type name",
			value:        reflect.ValueOf(customType),
			typ:          reflect.TypeOf(customType),
			wantTypeName: "CustomScalar",
			wantOk:       true,
		},
		{
			name:         "Pointer to CustomPointerType returns type name",
			value:        reflect.ValueOf(customPointerType),
			typ:          reflect.TypeOf(customPointerType),
			wantTypeName: "CustomPointerScalar",
			wantOk:       true,
		},
		{
			name:         "RegularStruct returns empty and false",
			value:        reflect.ValueOf(regular),
			typ:          reflect.TypeOf(regular),
			wantTypeName: "",
			wantOk:       false,
		},
		{
			name:         "Nil pointer returns empty and false",
			value:        reflect.ValueOf((*CustomPointerType)(nil)),
			typ:          reflect.TypeOf((*CustomPointerType)(nil)),
			wantTypeName: "",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeName, ok := GetGraphQLType(tt.value, tt.typ)
			if ok != tt.wantOk {
				t.Errorf("GetGraphQLType() ok = %v, want %v", ok, tt.wantOk)
			}
			if typeName != tt.wantTypeName {
				t.Errorf(
					"GetGraphQLType() typeName = %q, want %q",
					typeName,
					tt.wantTypeName,
				)
			}
		})
	}
}

func TestGetGraphQLTypeFromType(t *testing.T) {
	tests := []struct {
		name         string
		typ          reflect.Type
		wantTypeName string
		wantOk       bool
	}{
		{
			name:         "CustomType returns type name",
			typ:          reflect.TypeOf(CustomType{}),
			wantTypeName: "CustomScalar",
			wantOk:       true,
		},
		{
			name:         "Pointer to CustomPointerType returns type name",
			typ:          reflect.TypeOf(&CustomPointerType{}),
			wantTypeName: "CustomPointerScalar",
			wantOk:       true,
		},
		{
			name:         "RegularStruct returns empty and false",
			typ:          reflect.TypeOf(RegularStruct{}),
			wantTypeName: "",
			wantOk:       false,
		},
		{
			name:         "int returns empty and false",
			typ:          reflect.TypeOf(0),
			wantTypeName: "",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeName, ok := GetGraphQLTypeFromType(tt.typ)
			if ok != tt.wantOk {
				t.Errorf("GetGraphQLTypeFromType() ok = %v, want %v", ok, tt.wantOk)
			}
			if typeName != tt.wantTypeName {
				t.Errorf(
					"GetGraphQLTypeFromType() typeName = %q, want %q",
					typeName,
					tt.wantTypeName,
				)
			}
		})
	}
}

// Nested wrapper for testing deep unwrapping
type NestedWrapper struct {
	Value TestWrapper[string]
}

func (w NestedWrapper) GetGraphQLWrapped() TestWrapper[string] {
	return w.Value
}

func TestUnwrapValue_deeplyNested(t *testing.T) {
	// Test deeply nested wrappers
	innerWrapper := TestWrapper[string]{Value: "deep"}
	outerWrapper := NestedWrapper{Value: innerWrapper}

	result := UnwrapValue(reflect.ValueOf(outerWrapper))
	if !result.IsValid() {
		t.Fatal("UnwrapValue returned invalid value for nested wrapper")
	}

	// Result should be the inner TestWrapper[string]
	innerResult, ok := result.Interface().(TestWrapper[string])
	if !ok {
		t.Fatalf("got type: %T, want: TestWrapper[string]", result.Interface())
	}

	if innerResult.Value != "deep" {
		t.Errorf("got: %q, want: %q", innerResult.Value, "deep")
	}
}

func TestUnwrapValue_interfaceWrapper(t *testing.T) {
	// Test unwrapping through interface type
	wrapper := TestWrapper[string]{Value: "test"}
	var iface any = wrapper

	result := UnwrapValue(reflect.ValueOf(iface))
	if !result.IsValid() {
		t.Fatal("UnwrapValue returned invalid value for interface wrapper")
	}

	if result.Interface() != "test" {
		t.Errorf("got: %v, want: %q", result.Interface(), "test")
	}
}

func TestUnwrapValueField_noValueField(t *testing.T) {
	// Test wrapper without a Value field
	wrapper := TestWrapperNoValueField{Data: "test"}

	result := UnwrapValueField(reflect.ValueOf(wrapper))
	if result.IsValid() {
		t.Errorf(
			"UnwrapValueField should return invalid for wrapper without Value field, got: %v",
			result,
		)
	}
}

func TestUnwrapValue_multiLevelPointer(t *testing.T) {
	// Test multi-level pointer unwrapping
	wrapper := TestWrapper[int]{Value: 99}
	ptr1 := &wrapper
	ptr2 := &ptr1

	result := UnwrapValue(reflect.ValueOf(ptr2))
	if !result.IsValid() {
		t.Fatal("UnwrapValue returned invalid value for double pointer")
	}

	if result.Interface() != 99 {
		t.Errorf("got: %v, want: 99", result.Interface())
	}
}

func TestGetGraphQLType_nilValue(t *testing.T) {
	// Test GetGraphQLType with nil value
	var nilPtr *CustomPointerType
	v := reflect.ValueOf(nilPtr)
	typeName, ok := GetGraphQLType(v, v.Type())

	if ok {
		t.Errorf(
			"GetGraphQLType on nil pointer should return false, got: true with type %q",
			typeName,
		)
	}
}

func TestGetGraphQLType_interfaceValue(t *testing.T) {
	// Test GetGraphQLType with value wrapped in interface
	custom := CustomType{Data: "test"}
	var iface any = custom

	v := reflect.ValueOf(iface)
	typeName, ok := GetGraphQLType(v, v.Type())
	if !ok {
		t.Fatal("GetGraphQLType should return true for interface containing CustomType")
	}

	if typeName != "CustomScalar" {
		t.Errorf("got: %q, want: %q", typeName, "CustomScalar")
	}
}
