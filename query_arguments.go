package graphql

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"

	"github.com/llehouerou/go-graphql-client/internal/reflectutil"
)

// argumentFieldInfo holds information about a struct field used for GraphQL query arguments.
type argumentFieldInfo struct {
	jsonName  string
	fieldType reflect.Type
	value     reflect.Value
}

// queryArguments constructs a minified arguments string for variables.
//
// E.g., map[string]any{"a": int(123), "b": true} -> "$a:Int!$b:Boolean!".
func queryArguments(variables any) string {
	var buf bytes.Buffer

	switch v := variables.(type) {
	case map[string]any:
		writeArgumentsFromMap(&buf, v)
	default:
		fields := collectStructFieldsForArguments(variables)
		writeArgumentsFromFields(&buf, fields)
	}

	return buf.String()
}

// writeArgumentsFromMap writes GraphQL query arguments from a map of variables.
// Keys are sorted alphabetically for deterministic output.
func writeArgumentsFromMap(buf *bytes.Buffer, variables map[string]any) {
	var keys []string
	for k := range variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		_, _ = io.WriteString(buf, "$")
		_, _ = io.WriteString(buf, k)
		_, _ = io.WriteString(buf, ":")
		writeArgumentType(buf, reflect.TypeOf(variables[k]), variables[k], true)
	}
}

// collectStructFieldsForArguments extracts field information from a struct for use in GraphQL arguments.
// It validates the struct, collects exported fields with json tags, and returns them sorted by json name.
func collectStructFieldsForArguments(variables any) []argumentFieldInfo {
	val := reflect.ValueOf(variables)
	typ := reflect.TypeOf(variables)

	// Unwrap pointer if necessary
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	// Validate it's a struct
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("variables must be a struct or a map; got %T", variables))
	}

	// Collect field information
	var fields []argumentFieldInfo

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")

		// Skip fields without json tag or with json:"-"
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name from json tag (before comma if present)
		jsonName := jsonTag
		if commaIdx := bytes.IndexByte([]byte(jsonTag), ','); commaIdx > -1 {
			jsonName = jsonTag[:commaIdx]
		}

		// Skip if field name is empty after extraction
		if jsonName == "" || jsonName == "-" {
			continue
		}

		fields = append(fields, argumentFieldInfo{
			jsonName:  jsonName,
			fieldType: field.Type,
			value:     val.Field(i),
		})
	}

	// Sort fields by json name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].jsonName < fields[j].jsonName
	})

	return fields
}

// writeArgumentsFromFields writes GraphQL query arguments from collected field information.
func writeArgumentsFromFields(buf *bytes.Buffer, fields []argumentFieldInfo) {
	for _, f := range fields {
		_, _ = io.WriteString(buf, "$")
		_, _ = io.WriteString(buf, f.jsonName)
		_, _ = io.WriteString(buf, ":")
		writeArgumentType(buf, f.fieldType, f.value.Interface(), true)
	}
}

// writeArgumentType writes a minified GraphQL type for t to w.
// value indicates whether t is a value (required) type or pointer (optional) type.
// If value is true, then "!" is written at the end of t.
func writeArgumentType(w io.Writer, t reflect.Type, v any, value bool) {

	if reflectutil.ImplementsGraphQLType(t) {
		value = t.Kind() != reflect.Ptr
		var typeName string
		var ok bool

		// Try to use the actual value first if provided
		if v != nil {
			typeName, ok = reflectutil.GetGraphQLType(reflect.ValueOf(v), t)
		}

		// Fall back to type-based extraction if no value or extraction failed
		if !ok {
			typeName, ok = reflectutil.GetGraphQLTypeFromType(t)
		}

		if ok {
			_, _ = io.WriteString(w, typeName)
			if value {
				// Value is a required type, so add "!" to the end.
				_, _ = io.WriteString(w, "!")
			}
			return
		}
	}

	if t.Kind() == reflect.Ptr {
		// Pointer is an optional type, so no "!" at the end of the pointer's underlying type.
		writeArgumentType(w, t.Elem(), v, false)
		return
	}

	if reflectutil.IsIntegerKind(t.Kind()) {
		_, _ = io.WriteString(w, "Int")
	} else {
		switch t.Kind() {
		case reflect.Slice, reflect.Array:
			// List. E.g., "[Int]".
			_, _ = io.WriteString(w, "[")
			writeArgumentType(w, t.Elem(), nil, true)
			_, _ = io.WriteString(w, "]")
		case reflect.Float32, reflect.Float64:
			_, _ = io.WriteString(w, "Float")
		case reflect.Bool:
			_, _ = io.WriteString(w, "Boolean")
		default:
			n := t.Name()
			if n == "string" {
				n = "String"
			}
			_, _ = io.WriteString(w, n)
		}
	}

	if value {
		// Value is a required type, so add "!" to the end.
		_, _ = io.WriteString(w, "!")
	}
}
