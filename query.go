package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/llehouerou/go-graphql-client/ident"
	"github.com/llehouerou/go-graphql-client/internal/reflectutil"
	"github.com/llehouerou/go-graphql-client/types"
)

type constructOptionsOutput struct {
	operationName       string
	operationDirectives []string
}

func (coo constructOptionsOutput) OperationDirectivesString() string {
	operationDirectivesStr := strings.Join(coo.operationDirectives, " ")
	if operationDirectivesStr != "" {
		return fmt.Sprintf(" %s ", operationDirectivesStr)
	}
	return ""
}

func constructOptions(options []Option) (*constructOptionsOutput, error) {
	output := &constructOptionsOutput{}

	for _, option := range options {
		switch option.Type() {
		case optionTypeOperationName:
			output.operationName = option.String()
		case OptionTypeOperationDirective:
			output.operationDirectives = append(
				output.operationDirectives,
				option.String(),
			)
		default:
			return nil, fmt.Errorf("invalid query option type: %s", option.Type())
		}
	}

	return output, nil
}

// hasVariables checks if variables exist and should be used.
// Returns false for nil or empty maps, true otherwise.
func hasVariables(variables any) bool {
	if variables == nil {
		return false
	}
	reflectVal := reflect.ValueOf(variables)
	// If it's not a map, we have variables
	// If it's a map, only return true if it has entries
	return reflectVal.Kind() != reflect.Map || reflectVal.Len() > 0
}

// constructOperation builds a GraphQL operation string from struct and variables.
// operationType should be "query", "mutation", or "subscription".
// includeOperationTypeInDefault determines whether to prepend the operation type
// when no operation name or directives are specified (true for mutation/subscription, false for query).
func constructOperation(
	operationType string,
	v any,
	variables any,
	includeOperationTypeInDefault bool,
	options ...Option,
) (string, error) {
	query, err := query(v)
	if err != nil {
		return "", err
	}

	optionsOutput, err := constructOptions(options)
	if err != nil {
		return "", err
	}

	if hasVariables(variables) {
		return fmt.Sprintf(
			"%s %s(%s)%s%s",
			operationType,
			optionsOutput.operationName,
			queryArguments(variables),
			optionsOutput.OperationDirectivesString(),
			query,
		), nil
	}

	if optionsOutput.operationName == "" &&
		len(optionsOutput.operationDirectives) == 0 {
		if includeOperationTypeInDefault {
			return operationType + query, nil
		}
		return query, nil
	}

	return fmt.Sprintf(
		"%s %s%s%s",
		operationType,
		optionsOutput.operationName,
		optionsOutput.OperationDirectivesString(),
		query,
	), nil
}

// ConstructQuery builds GraphQL query string from struct and variables
func ConstructQuery(v any, variables any, options ...Option) (string, error) {
	return constructOperation("query", v, variables, false, options...)
}

// ConstructMutation builds GraphQL mutation string from struct and variables
func ConstructMutation(
	v any,
	variables any,
	options ...Option,
) (string, error) {
	return constructOperation("mutation", v, variables, true, options...)
}

// ConstructSubscription builds GraphQL subscription string from struct and variables
func ConstructSubscription(
	v any,
	variables any,
	options ...Option,
) (string, error) {
	return constructOperation("subscription", v, variables, true, options...)
}

// queryArguments constructs a minified arguments string for variables.
//
// E.g., map[string]any{"a": int(123), "b": true} -> "$a:Int!$b:Boolean!".
func queryArguments(variables any) string {
	var keys []string
	var buf bytes.Buffer

	switch v := variables.(type) {
	case map[string]any:
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			_, _ = io.WriteString(&buf, "$")
			_, _ = io.WriteString(&buf, k)
			_, _ = io.WriteString(&buf, ":")
			writeArgumentType(&buf, reflect.TypeOf(v[k]), v[k], true)
		}
	default:
		val := reflect.ValueOf(variables)
		typ := reflect.TypeOf(variables)
		if typ.Kind() == reflect.Ptr {
			val = val.Elem()
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct {
			panic(fmt.Sprintf("variables must be a struct or a map; got %T", variables))
		}

		// Collect field information in a single pass
		type fieldInfo struct {
			jsonName  string
			fieldType reflect.Type
			value     reflect.Value
		}
		var fields []fieldInfo

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

			fields = append(fields, fieldInfo{
				jsonName:  jsonName,
				fieldType: field.Type,
				value:     val.Field(i),
			})
		}

		// Sort fields by json name for deterministic output
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].jsonName < fields[j].jsonName
		})

		// Build the arguments string
		for _, f := range fields {
			_, _ = io.WriteString(&buf, "$")
			_, _ = io.WriteString(&buf, f.jsonName)
			_, _ = io.WriteString(&buf, ":")
			writeArgumentType(&buf, f.fieldType, f.value.Interface(), true)
		}
	}

	return buf.String()
}

// writeArgumentType writes a minified GraphQL type for t to w.
// value indicates whether t is a value (required) type or pointer (optional) type.
// If value is true, then "!" is written at the end of t.
func writeArgumentType(w io.Writer, t reflect.Type, v any, value bool) {

	if t.Implements(types.GraphqlTypeInterface) {
		var graphqlType types.GraphQLType
		var ok bool
		value = t.Kind() != reflect.Ptr
		if v != nil {
			graphqlType, ok = v.(types.GraphQLType)
		} else if t.Kind() == reflect.Ptr {
			graphqlType, ok = reflect.New(t.Elem()).Interface().(types.GraphQLType)
		} else {
			graphqlType, ok = reflect.Zero(t).Interface().(types.GraphQLType)
		}
		if ok {
			_, _ = io.WriteString(w, graphqlType.GetGraphQLType())
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

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		// List. E.g., "[Int]".
		_, _ = io.WriteString(w, "[")
		writeArgumentType(w, t.Elem(), nil, true)
		_, _ = io.WriteString(w, "]")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_, _ = io.WriteString(w, "Int")
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

	if value {
		// Value is a required type, so add "!" to the end.
		_, _ = io.WriteString(w, "!")
	}
}

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
	case reflect.Ptr:
		err := writeQuery(w, t.Elem(), reflectutil.ElemSafe(v), false)
		if err != nil {
			return fmt.Errorf("failed to write query for ptr `%v`: %w", t, err)
		}
	case reflect.Struct:

		if v.IsValid() {
			method := v.MethodByName(wrapperMethodName)
			if method.IsValid() {
				wrapped := method.Call(nil)[0]
				err := writeQuery(
					w,
					reflect.TypeOf(wrapped.Interface()),
					reflect.ValueOf(wrapped.Interface()),
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
			value := ""
			ok := false

			// Check if the field type implements GraphQLType
			if f.Type.Implements(types.GraphqlTypeInterface) {
				fieldVal := v.Field(i)
				// Only skip nil pointers and nil interfaces (not nil slices/maps)
				kind := fieldVal.Kind()
				if !fieldVal.IsValid() || ((kind == reflect.Ptr || kind == reflect.Interface) && fieldVal.IsNil()) {
					// Skip this field if it's a nil pointer or nil interface
					continue
				}
				graphqlType, typeok := fieldVal.Interface().(types.GraphQLType)
				if typeok {
					// Use reflection to check if the value inside the interface is nil pointer
					graphqlTypeVal := reflect.ValueOf(graphqlType)
					if graphqlTypeVal.IsValid() && (graphqlTypeVal.Kind() != reflect.Ptr || !graphqlTypeVal.IsNil()) {
						value = graphqlType.GetGraphQLType()
						ok = true
					} else {
						// Skip this field if the concrete value is a nil pointer
						continue
					}
				}
			} else if f.Type.Kind() == reflect.Slice && f.Type.Elem().Implements(types.GraphqlTypeInterface) {
				// For slices, check if the element type implements GraphQLType
				elemType := f.Type.Elem()
				// Create a zero value of the element type to call GetGraphQLType()
				var graphqlType types.GraphQLType
				var typeok bool
				if elemType.Kind() == reflect.Ptr {
					graphqlType, typeok = reflect.New(elemType.Elem()).Interface().(types.GraphQLType)
				} else {
					graphqlType, typeok = reflect.Zero(elemType).Interface().(types.GraphQLType)
				}
				if typeok {
					value = graphqlType.GetGraphQLType()
					ok = true
				}
			}

			if !ok {
				value, ok = f.Tag.Lookup("graphql")
			}
			// Skip this field if the tag value is hyphen
			if value == "-" {
				continue
			}
			if iter != 0 {
				_, _ = io.WriteString(w, ",")
			}
			iter++

			inlineField := f.Anonymous && !ok
			if !inlineField {
				if ok {
					_, _ = io.WriteString(w, value)
				} else {
					_, _ = io.WriteString(w, ident.ParseMixedCaps(f.Name).ToLowerCamelCase())
				}
			}
			// Skip writeQuery if the GraphQL type associated with the filed is scalar
			if isTrue(f.Tag.Get("scalar")) {
				continue
			}

			err := writeQuery(w, f.Type, reflectutil.FieldSafe(v, i), inlineField)
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
	case reflect.Slice:
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
	case reflect.Map:
		return fmt.Errorf("type %v is not supported, use [][2]any instead", t)
	}
	return nil
}

const (
	// wrapperMethodName is the method name for wrapper type unwrapping
	wrapperMethodName = "GetGraphQLWrapped"
)

var jsonUnmarshaler = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var idType = reflect.TypeOf(ID(""))

func isTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
