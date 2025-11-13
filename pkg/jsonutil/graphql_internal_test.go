package jsonutil

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// TestFindFieldsForKey tests the findFieldsForKey helper method
func TestFindFieldsForKey(t *testing.T) {
	rawMessageValue := reflect.ValueOf(json.RawMessage{})

	t.Run("finds struct field by name", func(t *testing.T) {
		type testStruct struct {
			Name string
			Age  int
		}
		target := testStruct{}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, hasMatching, rawMsg := d.findFieldsForKey("name", rawMessageValue)

		if len(fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(fields))
		}
		if !fields[0].field.IsValid() {
			t.Error("expected valid field")
		}
		if fields[0].isScalar {
			t.Error("expected non-scalar field")
		}
		if !fields[0].fragmentMatch {
			t.Error("expected fragment match")
		}
		// When field is valid and fragmentMatch is true, hasMatching should be true
		if !hasMatching {
			t.Error("expected hasMatchingFragmentWithField to be true")
		}
		if rawMsg {
			t.Error("expected rawMessage to be false")
		}
	})

	t.Run("finds field with graphql tag", func(t *testing.T) {
		type testStruct struct {
			UserName string `graphql:"name"`
		}
		target := testStruct{}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, _, _ := d.findFieldsForKey("name", rawMessageValue)

		if !fields[0].field.IsValid() {
			t.Error("expected valid field with graphql tag")
		}
	})

	t.Run("detects scalar field", func(t *testing.T) {
		type testStruct struct {
			Data string `scalar:"true"`
		}
		target := testStruct{}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, _, _ := d.findFieldsForKey("data", rawMessageValue)

		if !fields[0].isScalar {
			t.Error("expected scalar field to be detected")
		}
	})

	t.Run("detects json.RawMessage field", func(t *testing.T) {
		type testStruct struct {
			Data json.RawMessage
		}
		target := testStruct{}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		_, _, rawMsg := d.findFieldsForKey("data", rawMessageValue)

		if !rawMsg {
			t.Error("expected rawMessage to be true for json.RawMessage field")
		}
	})

	t.Run("handles wrapper types", func(t *testing.T) {
		type StringWrapper struct {
			Value string
		}
		// Method must be defined outside the test function
		type testStruct struct {
			Wrapped StringWrapper
		}
		target := testStruct{}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, _, _ := d.findFieldsForKey("wrapped", rawMessageValue)

		// Even without GetGraphQLWrapped, the field should be found
		if !fields[0].field.IsValid() {
			t.Error("expected valid field")
		}
	})

	t.Run("handles ordered map (slice of pairs)", func(t *testing.T) {
		target := [][2]interface{}{
			{"name", new(string)},
			{"age", new(int)},
		}
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, _, _ := d.findFieldsForKey("name", rawMessageValue)

		if !fields[0].field.IsValid() {
			t.Error("expected valid field from ordered map")
		}
	})

	t.Run("fragment matching when typename matches", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		d := &decoder{
			vs:              []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes:   []string{"User"},
			currentTypename: "User",
		}

		fields, hasMatching, _ := d.findFieldsForKey("name", rawMessageValue)

		if !fields[0].fragmentMatch {
			t.Error("expected fragment match when typenames match")
		}
		if !hasMatching {
			t.Error("expected hasMatchingFragmentWithField to be true")
		}
	})

	t.Run("fragment no match when typename differs", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		d := &decoder{
			vs:              []stack{{reflect.ValueOf(&target).Elem()}},
			fragmentTypes:   []string{"User"},
			currentTypename: "Admin",
		}

		fields, hasMatching, _ := d.findFieldsForKey("name", rawMessageValue)

		if fields[0].fragmentMatch {
			t.Error("expected no fragment match when typenames differ")
		}
		if hasMatching {
			t.Error("expected hasMatchingFragmentWithField to be false")
		}
	})

	t.Run("handles multiple stacks with mixed validity", func(t *testing.T) {
		type struct1 struct {
			Name string
		}
		type struct2 struct {
			Age int
		}
		target1 := struct1{}
		target2 := struct2{}
		d := &decoder{
			vs: []stack{
				{reflect.ValueOf(&target1).Elem()},
				{reflect.ValueOf(&target2).Elem()},
			},
			fragmentTypes: []string{"", ""},
		}

		fields, _, _ := d.findFieldsForKey("name", rawMessageValue)

		if len(fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(fields))
		}
		// First stack should have valid field
		if !fields[0].field.IsValid() {
			t.Error("expected valid field in first stack")
		}
		// Second stack should have invalid field (no "name" in struct2)
		if fields[1].field.IsValid() {
			t.Error("expected invalid field in second stack")
		}
	})

	t.Run("unwraps pointers and interfaces", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		targetPtr := &target
		targetInterface := interface{}(targetPtr)
		d := &decoder{
			vs:            []stack{{reflect.ValueOf(&targetInterface).Elem()}},
			fragmentTypes: []string{""},
		}

		fields, _, _ := d.findFieldsForKey("name", rawMessageValue)

		if !fields[0].field.IsValid() {
			t.Error("expected valid field after unwrapping pointer and interface")
		}
	})
}

// TestSelectAndPushFields tests the selectAndPushFields helper method
func TestSelectAndPushFields(t *testing.T) {
	t.Run("pushes valid fields to stacks", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		field := reflect.ValueOf(&target).Elem().Field(0)

		d := &decoder{
			vs: []stack{{reflect.ValueOf(&target).Elem()}},
		}

		fields := []fieldInfo{
			{field: field, isScalar: false, fragmentMatch: true},
		}

		someExist, isScalar := d.selectAndPushFields(fields, false)

		if !someExist {
			t.Error("expected someFieldExist to be true")
		}
		if isScalar {
			t.Error("expected isScalar to be false")
		}
		if len(d.vs[0]) != 2 {
			t.Errorf("expected stack length 2, got %d", len(d.vs[0]))
		}
	})

	t.Run("detects scalar field", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		field := reflect.ValueOf(&target).Elem().Field(0)

		d := &decoder{
			vs: []stack{{reflect.ValueOf(&target).Elem()}},
		}

		fields := []fieldInfo{
			{field: field, isScalar: true, fragmentMatch: true},
		}

		_, isScalar := d.selectAndPushFields(fields, false)

		if !isScalar {
			t.Error("expected isScalar to be true")
		}
	})

	t.Run("filters non-matching fragments when matching fragment exists", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		field := reflect.ValueOf(&target).Elem().Field(0)

		d := &decoder{
			vs: []stack{{reflect.ValueOf(&target).Elem()}},
		}

		// Field is from non-matching fragment, and a matching fragment also has this field
		fields := []fieldInfo{
			{field: field, isScalar: false, fragmentMatch: false},
		}

		d.selectAndPushFields(fields, true)

		// The field should be replaced with invalid value
		pushedField := d.vs[0][1]
		if pushedField.IsValid() {
			t.Error("expected invalid field when filtering non-matching fragment")
		}
	})

	t.Run("keeps non-matching fragment field when no matching fragment has it", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		target := testStruct{}
		field := reflect.ValueOf(&target).Elem().Field(0)

		d := &decoder{
			vs: []stack{{reflect.ValueOf(&target).Elem()}},
		}

		// Field is from non-matching fragment, but no matching fragment has this field
		fields := []fieldInfo{
			{field: field, isScalar: false, fragmentMatch: false},
		}

		d.selectAndPushFields(fields, false)

		// The field should be kept
		pushedField := d.vs[0][1]
		if !pushedField.IsValid() {
			t.Error("expected valid field when no matching fragment exists")
		}
	})

	t.Run("handles multiple stacks", func(t *testing.T) {
		type struct1 struct {
			Name string
		}
		type struct2 struct {
			Age int
		}
		target1 := struct1{}
		target2 := struct2{}
		field1 := reflect.ValueOf(&target1).Elem().Field(0)
		field2 := reflect.ValueOf(&target2).Elem().Field(0)

		d := &decoder{
			vs: []stack{
				{reflect.ValueOf(&target1).Elem()},
				{reflect.ValueOf(&target2).Elem()},
			},
		}

		fields := []fieldInfo{
			{field: field1, isScalar: false, fragmentMatch: true},
			{field: field2, isScalar: false, fragmentMatch: true},
		}

		d.selectAndPushFields(fields, false)

		if len(d.vs[0]) != 2 {
			t.Errorf("expected first stack length 2, got %d", len(d.vs[0]))
		}
		if len(d.vs[1]) != 2 {
			t.Errorf("expected second stack length 2, got %d", len(d.vs[1]))
		}
	})

	t.Run("returns false when no valid fields exist", func(t *testing.T) {
		d := &decoder{
			vs: []stack{
				{reflect.Value{}},
			},
		}

		fields := []fieldInfo{
			{field: reflect.Value{}, isScalar: false, fragmentMatch: true},
		}

		someExist, _ := d.selectAndPushFields(fields, false)

		if someExist {
			t.Error("expected someFieldExist to be false when no valid fields")
		}
	})
}

// TestReadNextToken tests the readNextToken helper method
func TestReadNextToken(t *testing.T) {
	t.Run("reads raw message for scalar field", func(t *testing.T) {
		jsonData := `{"nested": "value"}`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		rawMsg, ok := tok.(json.RawMessage)
		if !ok {
			t.Errorf("expected json.RawMessage, got %T", tok)
		}
		expected := `{"nested": "value"}`
		if string(rawMsg) != expected {
			t.Errorf("expected %q, got %q", expected, string(rawMsg))
		}
	})

	t.Run("reads raw message for rawMessage field", func(t *testing.T) {
		jsonData := `{"nested": "value"}`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		rawMsg, ok := tok.(json.RawMessage)
		if !ok {
			t.Errorf("expected json.RawMessage, got %T", tok)
		}
		expected := `{"nested": "value"}`
		if string(rawMsg) != expected {
			t.Errorf("expected %q, got %q", expected, string(rawMsg))
		}
	})

	t.Run("reads next token for regular field - string", func(t *testing.T) {
		jsonData := `"hello"`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		str, ok := tok.(string)
		if !ok {
			t.Errorf("expected string, got %T", tok)
		}
		if str != "hello" {
			t.Errorf("expected 'hello', got %q", str)
		}
	})

	t.Run("reads next token for regular field - number", func(t *testing.T) {
		jsonData := `42`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		num, ok := tok.(json.Number)
		if !ok {
			t.Errorf("expected json.Number, got %T", tok)
		}
		if num.String() != "42" {
			t.Errorf("expected '42', got %q", num.String())
		}
	})

	t.Run("reads next token for regular field - boolean", func(t *testing.T) {
		jsonData := `true`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		b, ok := tok.(bool)
		if !ok {
			t.Errorf("expected bool, got %T", tok)
		}
		if !b {
			t.Error("expected true, got false")
		}
	})

	t.Run("reads next token for regular field - null", func(t *testing.T) {
		jsonData := `null`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tok != nil {
			t.Errorf("expected nil, got %v", tok)
		}
	})

	t.Run("reads next token for regular field - object delimiter", func(t *testing.T) {
		jsonData := `{`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		delim, ok := tok.(json.Delim)
		if !ok {
			t.Errorf("expected json.Delim, got %T", tok)
		}
		if delim != '{' {
			t.Errorf("expected '{', got %v", delim)
		}
	})

	t.Run("reads next token for regular field - array delimiter", func(t *testing.T) {
		jsonData := `[`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		tok, err := d.readNextToken(false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		delim, ok := tok.(json.Delim)
		if !ok {
			t.Errorf("expected json.Delim, got %T", tok)
		}
		if delim != '[' {
			t.Errorf("expected '[', got %v", delim)
		}
	})

	t.Run("returns error on EOF for regular field", func(t *testing.T) {
		jsonData := ``
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		_, err := d.readNextToken(false, false)
		if err == nil {
			t.Error("expected error on EOF")
		}
		if err.Error() != "unexpected end of JSON input" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("returns error on decode failure for raw/scalar field", func(t *testing.T) {
		jsonData := `{invalid`
		dec := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
		dec.UseNumber()

		d := &decoder{
			tokenizer: dec,
		}

		_, err := d.readNextToken(true, false)
		if err == nil {
			t.Error("expected error on invalid JSON")
		}
	})
}
