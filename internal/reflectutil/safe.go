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
