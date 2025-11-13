package reflectutil

import "reflect"

// IndexSafe safely indexes into a reflect.Value.
// Returns the value at index i if v is valid and i is within bounds,
// otherwise returns an invalid reflect.Value.
func IndexSafe(v reflect.Value, i int) reflect.Value {
	if v.IsValid() && i >= 0 && i < v.Len() {
		return v.Index(i)
	}
	return reflect.ValueOf(nil)
}

// ElemSafe safely gets the element of a pointer or interface reflect.Value.
// Returns the element if v is valid and is a pointer or interface,
// otherwise returns an invalid reflect.Value.
func ElemSafe(v reflect.Value) reflect.Value {
	if v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		return v.Elem()
	}
	return reflect.ValueOf(nil)
}

// FieldSafe safely gets a struct field by index.
// Returns the field at index i if valStruct is valid,
// otherwise returns an invalid reflect.Value.
func FieldSafe(valStruct reflect.Value, i int) reflect.Value {
	if valStruct.IsValid() {
		return valStruct.Field(i)
	}
	return reflect.ValueOf(nil)
}

// IsNillable returns true if the given kind can hold a nil value.
func IsNillable(kind reflect.Kind) bool {
	switch kind {
	case reflect.Ptr,
		reflect.Interface,
		reflect.Slice,
		reflect.Map,
		reflect.Chan,
		reflect.Func:
		return true
	default:
		return false
	}
}

// UnwrapToConcreteValue unwraps pointers and interfaces to get to the concrete value.
// This is a common pattern used throughout the codebase to get past indirection layers.
// Returns the concrete value, or an invalid reflect.Value if unwrapping fails.
//
// Example:
//   var x **int
//   v := reflect.ValueOf(x)
//   concrete := UnwrapToConcreteValue(v) // returns the int value (if not nil)
func UnwrapToConcreteValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// IsNilValue safely checks if a reflect.Value is nil.
// Returns true if:
// - The value is invalid
// - The value's kind can hold nil (pointer, interface, slice, map, chan, func) AND it is nil
// Returns false for non-nillable kinds (int, string, struct, etc.)
//
// This consolidates the common pattern of checking both the kind and IsNil().
func IsNilValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	return IsNillable(v.Kind()) && v.IsNil()
}

// NewZeroOrPointerValue creates a new reflect.Value based on the type.
// If t is a pointer type, it creates a pointer to a new zero value: reflect.New(t.Elem())
// If t is a non-pointer type, it creates a zero value: reflect.Zero(t)
//
// This is useful when you need to instantiate a value of a type but don't know
// if it's a pointer or not.
//
// Example:
//   t := reflect.TypeOf((*int)(nil))  // *int
//   v := NewZeroOrPointerValue(t)      // returns new(int)
//
//   t := reflect.TypeOf(0)             // int
//   v := NewZeroOrPointerValue(t)      // returns zero int
func NewZeroOrPointerValue(t reflect.Type) reflect.Value {
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem())
	}
	return reflect.Zero(t)
}
