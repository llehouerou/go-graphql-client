package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/llehouerou/go-graphql-client/ident"
	"github.com/llehouerou/go-graphql-client/internal/reflectutil"
	"github.com/llehouerou/go-graphql-client/types"
)

// query uses writeQuery to recursively construct
// a minified query string from the provided struct v.
//
// E.g., struct{Foo Int, BarBaz *bool} -> "{foo,barBaz}".
func query(v any) (string, error) {
	var buf bytes.Buffer
	err := writeQuery(&buf, reflect.TypeOf(v), reflect.ValueOf(v), false)
	if err != nil {
		return "", fmt.Errorf("failed to write query: %w", err)
	}
	return buf.String(), nil
}

// fieldOutput contains the processed information for a struct field
// used during GraphQL query construction
type fieldOutput struct {
	shouldSkip bool
	name       string
	isInline   bool
	value      reflect.Value
	isScalar   bool
}

// processStructField processes a single struct field and returns
// information needed for query construction
func processStructField(
	f reflect.StructField,
	fieldValue reflect.Value,
) fieldOutput {
	value := ""
	ok := false

	// Check if the field type implements GraphQLType
	if reflectutil.ImplementsGraphQLType(f.Type) {
		// Only skip nil pointers and nil interfaces (not nil slices/maps)
		kind := fieldValue.Kind()
		if !fieldValue.IsValid() ||
			((kind == reflect.Ptr || kind == reflect.Interface) &&
				fieldValue.IsNil()) {
			// Skip this field if it's a nil pointer or nil interface
			return fieldOutput{shouldSkip: true}
		}
		typeName, typeok := reflectutil.GetGraphQLType(fieldValue, f.Type)
		if typeok {
			value = typeName
			ok = true
		} else {
			// Skip this field if the concrete value is a nil pointer
			return fieldOutput{shouldSkip: true}
		}
	} else if f.Type.Kind() == reflect.Slice &&
		reflectutil.ImplementsGraphQLType(f.Type.Elem()) {
		// For slices, check if the element type implements GraphQLType
		typeName, typeok := reflectutil.GetGraphQLTypeFromType(f.Type.Elem())
		if typeok {
			value = typeName
			ok = true
		}
	}

	if !ok {
		value, ok = f.Tag.Lookup(types.GraphQLTag)
	}
	// Skip this field if the tag value is hyphen
	if value == "-" {
		return fieldOutput{shouldSkip: true}
	}

	inlineField := f.Anonymous && !ok
	var fieldName string
	if !inlineField {
		if ok {
			fieldName = value
		} else {
			fieldName = ident.ParseMixedCaps(f.Name).ToLowerCamelCase()
		}
	}

	isScalar := reflectutil.IsTrue(f.Tag.Get(types.ScalarTag))

	return fieldOutput{
		shouldSkip: false,
		name:       fieldName,
		isInline:   inlineField,
		value:      fieldValue,
		isScalar:   isScalar,
	}
}

// writeStructQuery writes a minified query for a struct type to w.
// If inline is true, the struct fields are inlined into parent struct.
func writeStructQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {
	if v.IsValid() && reflectutil.IsWrapperType(v) {
		wrapped := reflectutil.UnwrapValue(v)
		if wrapped.IsValid() {
			err := writeQuery(
				w,
				wrapped.Type(),
				wrapped,
				inline,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to write query for wrapped struct `%v`: %w",
					t,
					err,
				)
			}
			return nil
		}
	}

	// If the type implements json.Unmarshaler, it's a scalar. Don't expand it.
	if reflect.PointerTo(t).Implements(jsonUnmarshaler) {
		return nil
	}
	if t.AssignableTo(idType) {
		return nil
	}
	if !inline {
		_, _ = io.WriteString(w, "{")
	}
	iter := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldVal := reflectutil.FieldSafe(v, i)
		output := processStructField(f, fieldVal)

		// Skip this field if indicated by processStructField
		if output.shouldSkip {
			continue
		}

		if iter != 0 {
			_, _ = io.WriteString(w, ",")
		}
		iter++

		if !output.isInline {
			_, _ = io.WriteString(w, output.name)
		}
		// Skip writeQuery if the GraphQL type associated with the field is scalar
		if output.isScalar {
			continue
		}

		err := writeQuery(w, f.Type, fieldVal, output.isInline)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for struct field `%v`: %w",
				f.Name,
				err,
			)
		}
	}
	if !inline {
		_, _ = io.WriteString(w, "}")
	}
	return nil
}

// writeSliceQuery writes a minified query for a slice type to w.
func writeSliceQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Kind() != reflect.Array {
		err := writeQuery(w, t.Elem(), reflectutil.IndexSafe(v, 0), false)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for slice item `%v`: %w",
				t,
				err,
			)
		}
		return nil
	}
	// handle [][2]any like an ordered map
	return writeOrderedMapQuery(w, t, v)
}

// writeOrderedMapQuery writes a minified query for [][2]any pattern to w.
func writeOrderedMapQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Len() != 2 {
		return fmt.Errorf("only arrays of len 2 are supported, got %v", t.Elem())
	}
	sliceOfPairs := v
	_, _ = io.WriteString(w, "{")
	for i := 0; i < sliceOfPairs.Len(); i++ {
		pair := sliceOfPairs.Index(i)
		// it.Value() returns any, so we need to use reflect.ValueOf
		// to cast it away
		key, val := pair.Index(0), reflect.ValueOf(pair.Index(1).Interface())
		keyString, ok := key.Interface().(string)
		if !ok {
			return fmt.Errorf("expected pair (string, %v), got (%v, %v)",
				val.Type(), key.Type(), val.Type())
		}
		_, _ = io.WriteString(w, keyString)
		err := writeQuery(w, val.Type(), val, false)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for pair[1] `%v`: %w",
				val.Type(),
				err,
			)
		}
	}
	_, _ = io.WriteString(w, "}")
	return nil
}

// writeInterfaceQuery writes a minified query for an interface type to w.
func writeInterfaceQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {
	val := reflect.ValueOf(v.Interface())
	if !val.IsValid() {
		return nil
	}
	// Check if the interface contains a nil pointer
	kind := val.Kind()
	if (kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Slice ||
		kind == reflect.Map || kind == reflect.Chan || kind == reflect.Func) &&
		val.IsNil() {
		return nil
	}
	err := writeQuery(w, val.Type(), val, inline)
	if err != nil {
		return fmt.Errorf("failed to write query for interface `%v`: %w", t, err)
	}
	return nil
}

// writeQuery writes a minified query for t to w.
// If inline is true, the struct fields of t are inlined into parent struct.
func writeQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {

	switch t.Kind() {
	case reflect.Interface:
		return writeInterfaceQuery(w, t, v, inline)
	case reflect.Ptr:
		err := writeQuery(w, t.Elem(), reflectutil.ElemSafe(v), false)
		if err != nil {
			return fmt.Errorf("failed to write query for ptr `%v`: %w", t, err)
		}
	case reflect.Struct:
		return writeStructQuery(w, t, v, inline)
	case reflect.Slice:
		return writeSliceQuery(w, t, v)
	case reflect.Map:
		return fmt.Errorf("type %v is not supported, use [][2]any instead", t)
	}
	return nil
}

var jsonUnmarshaler = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var idType = reflect.TypeOf(ID(""))
