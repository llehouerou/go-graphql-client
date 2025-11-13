// Package jsonutil provides a function for decoding JSON
// into a GraphQL query data structure.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/llehouerou/go-graphql-client/internal/reflectutil"
	"github.com/llehouerou/go-graphql-client/internal/tagparser"
	"github.com/llehouerou/go-graphql-client/types"
)


// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
//
// The implementation is created on top of the JSON tokenizer available
// in "encoding/json".Decoder.
//
// # Wrapper Types
//
// UnmarshalGraphQL supports transparent unwrapping of container types that
// implement the GetGraphQLWrapped() method. This allows GraphQL schemas with
// wrapper/container patterns to be cleanly represented in Go.
//
// Convention: Any type implementing GetGraphQLWrapped() MUST have an exported
// field named "Value" that holds the wrapped data. During unmarshaling, the
// library will detect the GetGraphQLWrapped() method and unmarshal JSON data
// directly into the Value field, bypassing the wrapper.
//
// Rationale: The GetGraphQLWrapped() method returns a value (used during query
// construction for reflection), but unmarshaling requires a writable field
// reference. The "Value" field provides this writable target.
//
// Example:
//
//	type Wrapper[T any] struct {
//	    Value T  // REQUIRED: Must be named reflectutil.WrapperFieldName ("Value")
//	}
//	func (w Wrapper[T]) GetGraphQLWrapped() T { return w.Value }
func UnmarshalGraphQL(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := (&decoder{tokenizer: dec}).Decode(v)
	if err != nil {
		return err
	}
	tok, err := dec.Token()
	switch err {
	case io.EOF:
		// Expect to get io.EOF. There shouldn't be any more
		// tokens left after we've decoded v successfully.
		return nil
	case nil:
		return fmt.Errorf("invalid token '%v' after top-level value", tok)
	default:
		return err
	}
}

// decoder is a JSON decoder that performs custom unmarshaling behavior
// for GraphQL query data structures. It's implemented on top of a JSON tokenizer.
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
		Decode(v any) error
	}

	// Stack of what part of input JSON we're in the middle of - objects, arrays.
	parseState []json.Delim

	// Stacks of values where to unmarshal.
	// The top of each stack is the reflect.Value where to unmarshal next JSON value.
	//
	// The reason there's more than one stack is because we might be unmarshaling
	// a single JSON value into multiple GraphQL fragments or embedded structs, so
	// we keep track of them all.
	vs []stack

	// fragmentTypes tracks the typename for each stack in vs. Empty string means not a fragment
	// or typename not applicable. This is used to filter inline fragments.
	fragmentTypes []string

	// currentTypename holds the __typename value for the current object being unmarshaled.
	// This is used to filter inline fragments so only the matching fragment is populated.
	currentTypename string

	// currentKey holds the current JSON key being processed, used to capture __typename.
	currentKey string
}

type stack []reflect.Value

func (s stack) Top() reflect.Value {
	return s[len(s)-1]
}

func (s stack) Pop() stack {
	return s[:len(s)-1]
}

// shouldIncludeFragment determines if a GraphQL fragment field should be included
// based on the current typename. Returns true if:
// - No typename is set (backward compatibility: include all fragments)
// - The fragment's typename matches the current typename
//
//nolint:unused // Reserved for future use
func (d *decoder) shouldIncludeFragment(
	field reflect.StructField,
) bool {
	tag, ok := field.Tag.Lookup(types.GraphQLTag)
	if !ok {
		return true
	}
	return d.shouldIncludeFragmentByTag(tag)
}

// shouldIncludeFragmentByTag determines if a fragment with the given tag should be included.
//
//nolint:unused // Reserved for future use
func (d *decoder) shouldIncludeFragmentByTag(
	tag string,
) bool {
	// If no typename is set, include all fragments (backward compatibility)
	if d.currentTypename == "" {
		return true
	}
	// Extract the typename from the fragment tag
	fragmentTypename := extractFragmentTypename(tag)
	if fragmentTypename == "" {
		return true // Not a fragment or malformed tag, include it
	}
	// Only include if the fragment typename matches the current typename
	return fragmentTypename == d.currentTypename
}

// Decode decodes a single JSON value from d.tokenizer into v.
func (d *decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot decode into non-pointer %T", v)
	}
	d.vs = []stack{{rv.Elem()}}
	d.fragmentTypes = []string{""} // Root is not a fragment
	return d.decode()
}

// decode decodes a single JSON value from d.tokenizer into d.vs.
func (d *decoder) decode() error {
	rawMessageValue := reflect.ValueOf(json.RawMessage{})

	// The loop invariant is that the top of each d.vs stack
	// is where we try to unmarshal the next JSON value we see.
	for len(d.vs) > 0 {
		var tok any
		tok, err := d.tokenizer.Token()

		if err == io.EOF {
			return errors.New("unexpected end of JSON input")
		} else if err != nil {
			return err
		}

		switch {

		// Are we inside an object and seeing next key (rather than end of object)?
		case d.state() == '{' && tok != json.Delim('}'):
			key, ok := tok.(string)
			if !ok {
				return errors.New("unexpected non-key in JSON input")
			}

			tok, err = d.decodeObjectKey(key, rawMessageValue)
			if err != nil {
				return err
			}

		// Are we inside an array and seeing next value (rather than end of array)?
		case d.state() == '[' && tok != json.Delim(']'):
			err = d.decodeArrayValue()
			if err != nil {
				return err
			}
		}

		switch tok := tok.(type) {
		case string, json.Number, bool, nil, json.RawMessage:
			// Value.
			err := d.decodeScalarValue(tok)
			if err != nil {
				return err
			}

		case json.Delim:
			switch tok {
			case '{':
				// Start of object.
				d.decodeObjectStart()
			case '[':
				// Start of array.
				err := d.decodeArrayStart()
				if err != nil {
					return err
				}
			case '}':
				// End of object.
				d.popAllVs()
				d.popState()
			case ']':
				// End of array.
				d.popLeftArrayTemplates()
				d.popAllVs()
				d.popState()
			default:
				return errors.New("unexpected delimiter in JSON input")
			}
		default:
			return errors.New("unexpected token in JSON input")
		}
	}
	return nil
}

// fieldInfo holds information about a field discovered during JSON object unmarshaling.
type fieldInfo struct {
	field         reflect.Value
	isScalar      bool
	fragmentMatch bool
}

// decodeObjectKey handles the processing of an object key and its value.
// This is called when we're inside an object and see the next key.
func (d *decoder) decodeObjectKey(
	key string,
	rawMessageValue reflect.Value,
) (any, error) {
	// Track current key for typename capture
	d.currentKey = key

	// First pass: find all fields and check if any matching fragment has it
	fields, hasMatchingFragmentWithField, rawMessage := d.findFieldsForKey(
		key,
		rawMessageValue,
	)

	// Second pass: decide which fields to use and push to value stacks
	someFieldExist, isScalar := d.selectAndPushFields(
		fields,
		hasMatchingFragmentWithField,
	)

	if !someFieldExist {
		return nil, fmt.Errorf(
			"struct field for %q doesn't exist in any of %v places to unmarshal",
			key,
			len(d.vs),
		)
	}

	// Read the next token based on field type
	return d.readNextToken(rawMessage, isScalar)
}

// findFieldsForKey discovers fields matching the given key across all value stacks.
// It returns:
// - fields: slice of fieldInfo (one per stack)
// - hasMatchingFragmentWithField: whether any matching fragment has the field
// - rawMessage: whether any field is of json.RawMessage type
func (d *decoder) findFieldsForKey(
	key string,
	rawMessageValue reflect.Value,
) ([]fieldInfo, bool, bool) {
	fields := make([]fieldInfo, len(d.vs))
	hasMatchingFragmentWithField := false
	rawMessage := false

	for i := range d.vs {
		v := d.vs[i].Top()
		v = reflectutil.UnwrapToConcreteValue(v)

		var f reflect.Value
		var scalar bool

		switch v.Kind() {
		case reflect.Struct:
			f, scalar = fieldByGraphQLName(v, key)
			if f.IsValid() {
				// Check if this is a wrapper type and unwrap if needed
				unwrapped := reflectutil.UnwrapValueField(f)
				if unwrapped.IsValid() {
					// Wrapper type detected. Unmarshal directly into
					// the unwrapped Value field, bypassing the wrapper.
					f = unwrapped
				}
				// Check for special embedded json
				if f.Type() == rawMessageValue.Type() {
					rawMessage = true
				}
			}
		case reflect.Slice:
			f = orderedMapValueByGraphQLName(v, key)
			// For ordered maps, we need to be careful about unwrapping
			// Unwrap pointers, but keep interfaces as they are
			// (unmarshalValue can handle interface types)
			for f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
		}

		fragmentMatch := true
		if i < len(d.fragmentTypes) && d.fragmentTypes[i] != "" &&
			d.currentTypename != "" {
			fragmentMatch = d.fragmentTypes[i] == d.currentTypename
		}

		fields[i] = fieldInfo{
			field:         f,
			isScalar:      scalar,
			fragmentMatch: fragmentMatch,
		}

		if f.IsValid() && fragmentMatch {
			hasMatchingFragmentWithField = true
		}
	}

	return fields, hasMatchingFragmentWithField, rawMessage
}

// selectAndPushFields processes discovered fields, filtering by fragment matching,
// and pushes selected fields to the value stacks.
// Returns (someFieldExist, isScalar) flags.
func (d *decoder) selectAndPushFields(
	fields []fieldInfo,
	hasMatchingFragmentWithField bool,
) (someFieldExist, isScalar bool) {
	for i := range d.vs {
		f := fields[i].field

		if f.IsValid() {
			someFieldExist = true
			if fields[i].isScalar {
				isScalar = true
			}
		}

		// Skip this field if:
		// 1. It's from a non-matching fragment AND
		// 2. A matching fragment also has this field
		if f.IsValid() && !fields[i].fragmentMatch &&
			hasMatchingFragmentWithField {
			f = reflect.Value{}
		}

		d.vs[i] = append(d.vs[i], f)
	}

	return someFieldExist, isScalar
}

// readNextToken reads the next JSON token based on whether the field is raw or scalar.
// For raw/scalar fields, it decodes the entire value as json.RawMessage.
// For regular fields, it returns the next token for further processing.
func (d *decoder) readNextToken(rawMessage, isScalar bool) (any, error) {
	if rawMessage || isScalar {
		// Read the next complete object from the json stream
		var data json.RawMessage
		err := d.tokenizer.Decode(&data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	// We've just consumed the current token, which was the key.
	// Read the next token, which should be the value,
	// and let the rest of code process it.
	tok, err := d.tokenizer.Token()
	if err == io.EOF {
		return nil, errors.New("unexpected end of JSON input")
	} else if err != nil {
		return nil, err
	}

	return tok, nil
}

// decodeArrayValue handles processing an array value by appending a new element
// to slices in the decoder's value stack.
func (d *decoder) decodeArrayValue() error {
	someSliceExist := false
	for i := range d.vs {
		v := d.vs[i].Top()
		v = reflectutil.UnwrapToConcreteValue(v)

		// Check if this is a wrapper type (has GetGraphQLWrapped method).
		// If so, unwrap to get the actual slice field per "Value" convention.
		if v.IsValid() {
			unwrapped := reflectutil.UnwrapValueField(v)
			if unwrapped.IsValid() {
				v = unwrapped
			}
		}

		var f reflect.Value
		if v.Kind() == reflect.Slice {
			// we want to append the template item copy
			// so that all the inner structure gets preserved
			copied, err := copyTemplate(v.Index(0))
			if err != nil {
				return fmt.Errorf("failed to copy template: %w", err)
			}
			v.Set(reflect.Append(v, copied)) // v = append(v, T).
			f = v.Index(v.Len() - 1)
			someSliceExist = true
		}
		d.vs[i] = append(d.vs[i], f)
	}
	if !someSliceExist {
		return fmt.Errorf(
			"slice doesn't exist in any of %v places to unmarshal",
			len(d.vs),
		)
	}
	return nil
}

// decodeScalarValue handles decoding of scalar values
// (string, number, bool, nil, json.RawMessage).
func (d *decoder) decodeScalarValue(tok any) error {
	// Capture __typename value to filter inline fragments
	if d.currentKey == types.TypenameField {
		if typename, ok := tok.(string); ok {
			d.currentTypename = typename
		}
	}

	for i := range d.vs {
		v := d.vs[i].Top()
		if !v.IsValid() {
			continue
		}
		err := unmarshalValue(tok, v)
		if err != nil {
			return err
		}
	}
	d.popAllVs()
	return nil
}

// decodeObjectStart handles the start of a JSON object ('{' token).
// It initializes values and discovers GraphQL fragments and embedded structs.
func (d *decoder) decodeObjectStart() {
	d.pushState('{')

	frontier := make([]reflect.Value, len(d.vs))
	for i := range d.vs {
		v := d.vs[i].Top()
		frontier[i] = v
		// TODO: Do this recursively or not? Add a test case if needed.
		if v.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(v.Type().Elem())) // v = new(T).
		}
	}
	// Find GraphQL fragments/embedded structs recursively, adding to frontier
	// as new ones are discovered and exploring them further.
	for len(frontier) > 0 {
		v := frontier[0]
		frontier = frontier[1:]
		v = reflectutil.UnwrapToConcreteValue(v)

		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				field := v.Type().Field(i)
				if isGraphQLFragment(field) {
					// Add GraphQL fragment and track its typename
					d.vs = append(d.vs, []reflect.Value{v.Field(i)})
					tag, _ := field.Tag.Lookup(types.GraphQLTag)
					d.fragmentTypes = append(
						d.fragmentTypes,
						extractFragmentTypename(tag),
					)
					frontier = append(frontier, v.Field(i))
				} else if field.Anonymous {
					// Add embedded struct (not a fragment)
					d.vs = append(d.vs, []reflect.Value{v.Field(i)})
					d.fragmentTypes = append(d.fragmentTypes, "")
					frontier = append(frontier, v.Field(i))
				}
			}
		} else if isOrderedMap(v) {
			for i := 0; i < v.Len(); i++ {
				pair := v.Index(i)
				key, val := pair.Index(0), pair.Index(1)
				keyStr := key.Interface().(string)
				if keyForGraphQLFragment(keyStr) {
					// Add GraphQL fragment and track its typename
					d.vs = append(d.vs, []reflect.Value{val})
					d.fragmentTypes = append(
						d.fragmentTypes,
						extractFragmentTypename(keyStr),
					)
					frontier = append(frontier, val)
				}
			}
		}
	}
}

// decodeArrayStart handles the start of a JSON array ('[' token).
// It initializes slices and ensures they have a template element.
func (d *decoder) decodeArrayStart() error {
	d.pushState('[')

	for i := range d.vs {
		v := d.vs[i].Top()
		// TODO: Confirm this is needed, write a test case.
		//if v.Kind() == reflect.Ptr && v.IsNil() {
		//	v.Set(reflect.New(v.Type().Elem())) // v = new(T).
		//}

		// Reset slice to empty (in case it had non-zero initial value).
		v = reflectutil.UnwrapToConcreteValue(v)

		if v.Kind() != reflect.Slice {
			continue
		}
		newSlice := reflect.MakeSlice(v.Type(), 0, 0) // v = make(T, 0, 0).
		switch v.Len() {
		case 0:
			// if there is no template we need to create one so that we can
			// handle both cases (with or without a template) in the same way
			newSlice = reflect.Append(newSlice, reflect.Zero(v.Type().Elem()))
		case 1:
			// if there is a template, we need to keep it at index 0
			newSlice = reflect.Append(newSlice, v.Index(0))
		case 2:
			return fmt.Errorf("template slice can only have 1 item, got %d", v.Len())
		}
		v.Set(newSlice)
	}
	return nil
}

func copyTemplate(template reflect.Value) (reflect.Value, error) {
	if isOrderedMap(template) {
		// copy slice if it's actually an ordered map
		return copyOrderedMap(template), nil
	}
	if template.Kind() == reflect.Map {
		return reflect.Value{}, fmt.Errorf(
			"unsupported template type `%v`, use [][2]any for ordered map instead",
			template.Type(),
		)
	}
	// don't need to copy regular slice
	return template, nil
}

func isOrderedMap(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	t := v.Type()
	return t.Kind() == reflect.Slice &&
		t.Elem().Kind() == reflect.Array &&
		t.Elem().Len() == 2
}

func copyOrderedMap(m reflect.Value) reflect.Value {
	newMap := reflect.MakeSlice(m.Type(), 0, m.Len())
	for i := 0; i < m.Len(); i++ {
		pair := m.Index(i)
		newMap = reflect.Append(newMap, pair)
	}
	return newMap
}

// pushState pushes a new parse state s onto the stack.
func (d *decoder) pushState(s json.Delim) {
	d.parseState = append(d.parseState, s)
}

// popState pops a parse state (already obtained) off the stack.
// The stack must be non-empty.
func (d *decoder) popState() {
	d.parseState = d.parseState[:len(d.parseState)-1]
}

// state reports the parse state on top of stack, or 0 if empty.
func (d *decoder) state() json.Delim {
	if len(d.parseState) == 0 {
		return 0
	}
	return d.parseState[len(d.parseState)-1]
}

// popAllVs pops from all d.vs stacks, keeping only non-empty ones.
func (d *decoder) popAllVs() {
	var nonEmpty []stack
	var nonEmptyTypes []string
	for i := range d.vs {
		d.vs[i] = d.vs[i].Pop()
		if len(d.vs[i]) > 0 {
			nonEmpty = append(nonEmpty, d.vs[i])
			// Keep fragment type in sync, using empty string if index out of bounds
			if i < len(d.fragmentTypes) {
				nonEmptyTypes = append(nonEmptyTypes, d.fragmentTypes[i])
			} else {
				nonEmptyTypes = append(nonEmptyTypes, "")
			}
		}
	}
	d.vs = nonEmpty
	d.fragmentTypes = nonEmptyTypes
}

// popLeftArrayTemplates pops left from last array items of all d.vs stacks.
func (d *decoder) popLeftArrayTemplates() {
	for i := range d.vs {
		v := d.vs[i].Top()
		// Unwrap pointers and interfaces to get to the actual slice
		v = reflectutil.UnwrapToConcreteValue(v)

		// Only call Slice if it's actually a slice type
		if v.IsValid() && v.Kind() == reflect.Slice {
			v.Set(v.Slice(1, v.Len()))
		}
	}
}

// fieldByGraphQLName returns an exported struct field of struct v
// that matches GraphQL name, or invalid reflect.Value if none found.
func fieldByGraphQLName(
	v reflect.Value,
	name string,
) (val reflect.Value, taggedAsScalar bool) {
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).PkgPath != "" {
			// Skip unexported field.
			continue
		}
		if hasGraphQLName(v.Type().Field(i), v.Field(i), name) {
			return v.Field(i), hasScalarTag(v.Type().Field(i))
		}
	}
	return reflect.Value{}, false
}

// orderedMapValueByGraphQLName takes [][2]string, interprets it as an ordered map
// and returns value for corresponding key, or invalid reflect.Value if none found.
func orderedMapValueByGraphQLName(v reflect.Value, name string) reflect.Value {
	for i := 0; i < v.Len(); i++ {
		pair := v.Index(i)
		key := pair.Index(0).Interface().(string)
		if keyHasGraphQLName(key, name) {
			return pair.Index(1)
		}
	}
	return reflect.Value{}
}

func hasScalarTag(f reflect.StructField) bool {
	return isTrue(f.Tag.Get(types.ScalarTag))
}

func isTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

// hasGraphQLName reports whether struct field f has GraphQL name.
func hasGraphQLName(f reflect.StructField, v reflect.Value, name string) bool {
	value := ""
	ok := false
	if reflectutil.ImplementsGraphQLType(f.Type) {
		typeName, typeok := reflectutil.GetGraphQLType(v, f.Type)
		if typeok {
			value = typeName
			ok = true
		}
	}
	if !ok {
		value, ok = f.Tag.Lookup(types.GraphQLTag)
	}
	if !ok {
		// TODO: caseconv package is relatively slow. Optimize it, then consider using it here.
		//return caseconv.MixedCapsToLowerCamelCase(f.Name) == name
		return strings.EqualFold(f.Name, name)
	}
	return keyHasGraphQLName(value, name)
}

func keyHasGraphQLName(value, name string) bool {
	parsed, err := tagparser.ParseGraphQLTag(value)
	if err != nil {
		return false
	}
	if parsed.IsFragment {
		// GraphQL fragment. It doesn't have a name.
		return false
	}
	// When there's an alias, the response uses the alias name.
	// Otherwise, it uses the field name.
	if parsed.Alias != "" {
		return parsed.Alias == name
	}
	return parsed.FieldName == name
}

// isGraphQLFragment reports whether struct field f is a GraphQL fragment.
func isGraphQLFragment(f reflect.StructField) bool {
	value, ok := f.Tag.Lookup(types.GraphQLTag)
	if !ok {
		return false
	}
	return keyForGraphQLFragment(value)
}

// isGraphQLFragment reports whether ordered map kv pair f is a GraphQL fragment.
func keyForGraphQLFragment(value string) bool {
	parsed, err := tagparser.ParseGraphQLTag(value)
	if err != nil {
		return false
	}
	return parsed.IsFragment
}

// extractFragmentTypename extracts the typename from a GraphQL fragment tag.
// For example, "... on SolanaTokenTransferAuthorizationRequest" returns "SolanaTokenTransferAuthorizationRequest".
// Returns empty string if not a valid fragment tag.
func extractFragmentTypename(tag string) string {
	parsed, err := tagparser.ParseGraphQLTag(tag)
	if err != nil {
		return ""
	}
	if !parsed.IsFragment {
		return ""
	}
	return parsed.TypeName
}

// unmarshalValue unmarshals JSON value into v.
// v must be addressable and not obtained by the use of unexported
// struct fields, otherwise unmarshalValue will panic.
func unmarshalValue(value any, v reflect.Value) error {
	b, err := json.Marshal(
		value,
	) // TODO: Short-circuit (if profiling says it's worth it).
	if err != nil {
		return err
	}
	ty := v.Type()
	if ty.Kind() == reflect.Interface {
		if !v.Elem().IsValid() {
			return json.Unmarshal(b, v.Addr().Interface())
		}
		ty = v.Elem().Type()
	}
	newVal := reflect.New(ty)
	err = json.Unmarshal(b, newVal.Interface())
	if err != nil {
		return err
	}
	v.Set(newVal.Elem())
	return nil
}
