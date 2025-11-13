package reflectutil

import (
	"reflect"

	"github.com/llehouerou/go-graphql-client/types"
)

const (
	// WrapperMethodName is the name of the method that unwraps container types.
	// Types implementing this method should follow the wrapper convention:
	// they must have an exported field named "Value" that holds the wrapped data.
	WrapperMethodName = "GetGraphQLWrapped"

	// WrapperFieldName is the required name of the field holding wrapped data
	// in types that implement the wrapper pattern (GetGraphQLWrapped method).
	WrapperFieldName = "Value"
)

// ImplementsGraphQLType reports whether the given type implements the GraphQLType interface.
// This checks if the type provides a custom GraphQL type name via GetGraphQLType().
func ImplementsGraphQLType(t reflect.Type) bool {
	return t.Implements(types.GraphqlTypeInterface)
}

// IsWrapperType reports whether the given reflect.Value is a wrapper type.
// A wrapper type is one that implements the GetGraphQLWrapped() method.
// Returns false if the value is invalid or nil.
func IsWrapperType(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	// Unwrap pointers and interfaces to check the concrete type
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return false
	}

	method := v.MethodByName(WrapperMethodName)
	return method.IsValid()
}

// UnwrapValue unwraps a wrapper type by calling its GetGraphQLWrapped() method.
// If the value is not a wrapper type, returns an invalid reflect.Value.
// If the value is a wrapper type, calls GetGraphQLWrapped() and returns the result.
//
// Note: This is used for query construction. For unmarshaling, the wrapped data
// must be stored in a field named "Value" per the wrapper convention.
func UnwrapValue(v reflect.Value) reflect.Value {
	if !IsWrapperType(v) {
		return reflect.Value{}
	}

	// Unwrap pointers and interfaces to get to the struct
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if !v.IsValid() {
		return reflect.Value{}
	}

	// Call GetGraphQLWrapped() method
	method := v.MethodByName(WrapperMethodName)
	if !method.IsValid() {
		return reflect.Value{}
	}

	// Call the method with no arguments and get the first return value
	results := method.Call(nil)
	if len(results) == 0 {
		return reflect.Value{}
	}

	return results[0]
}

// UnwrapValueField unwraps a wrapper type by accessing its Value field.
// This is specifically for unmarshaling where we need a writable field reference.
// If the value is not a wrapper type or doesn't have a Value field, returns an invalid reflect.Value.
//
// Convention: Wrapper types MUST have an exported field named "Value" for unmarshaling.
func UnwrapValueField(v reflect.Value) reflect.Value {
	if !IsWrapperType(v) {
		return reflect.Value{}
	}

	// Unwrap pointers and interfaces to get to the struct
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if !v.IsValid() {
		return reflect.Value{}
	}

	// Access the Value field per wrapper convention
	valueField := v.FieldByName(WrapperFieldName)
	if !valueField.IsValid() {
		return reflect.Value{}
	}

	return valueField
}

// UnwrapValueOrOriginal unwraps a wrapper type if possible, otherwise returns the original value.
// This is a convenience function that always returns a valid value for further processing.
func UnwrapValueOrOriginal(v reflect.Value) reflect.Value {
	unwrapped := UnwrapValue(v)
	if unwrapped.IsValid() {
		return unwrapped
	}
	return v
}

// GetGraphQLType extracts the GraphQL type name from a value that implements GraphQLType interface.
// Returns empty string if the value doesn't implement GraphQLType or if extraction fails.
func GetGraphQLType(v reflect.Value, t reflect.Type) (string, bool) {
	if !ImplementsGraphQLType(t) {
		return "", false
	}

	// Handle nil pointers and interfaces
	if !v.IsValid() {
		return "", false
	}

	kind := v.Kind()
	if (kind == reflect.Ptr || kind == reflect.Interface) && v.IsNil() {
		return "", false
	}

	// Try to get the GraphQLType from the value
	graphqlType, ok := v.Interface().(types.GraphQLType)
	if !ok {
		return "", false
	}

	// Additional check: if the interface contains a nil pointer, reject it
	graphqlTypeVal := reflect.ValueOf(graphqlType)
	if graphqlTypeVal.IsValid() &&
		graphqlTypeVal.Kind() == reflect.Ptr &&
		graphqlTypeVal.IsNil() {
		return "", false
	}

	return graphqlType.GetGraphQLType(), true
}

// GetGraphQLTypeFromType extracts the GraphQL type name from a type (not value).
// This creates a zero value or pointer to call GetGraphQLType().
// Useful when you don't have an instance but need the type name.
func GetGraphQLTypeFromType(t reflect.Type) (string, bool) {
	if !ImplementsGraphQLType(t) {
		return "", false
	}

	var graphqlType types.GraphQLType
	var ok bool

	if t.Kind() == reflect.Ptr {
		graphqlType, ok = reflect.New(t.Elem()).Interface().(types.GraphQLType)
	} else {
		graphqlType, ok = reflect.Zero(t).Interface().(types.GraphQLType)
	}

	if !ok {
		return "", false
	}

	return graphqlType.GetGraphQLType(), true
}
