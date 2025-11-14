package graphql

import (
	"bytes"
	"reflect"
	"testing"
)

// TestWriteArgumentsFromMap tests the writeArgumentsFromMap helper function
func TestWriteArgumentsFromMap(t *testing.T) {
	t.Run("empty map produces empty string", func(t *testing.T) {
		var buf bytes.Buffer
		writeArgumentsFromMap(&buf, map[string]any{})

		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})

	t.Run("single entry map", func(t *testing.T) {
		var buf bytes.Buffer
		writeArgumentsFromMap(&buf, map[string]any{
			"id": 123,
		})

		expected := "$id:Int!"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("multiple entries are sorted alphabetically", func(t *testing.T) {
		var buf bytes.Buffer
		writeArgumentsFromMap(&buf, map[string]any{
			"zebra":  "test",
			"apple":  42,
			"banana": true,
		})

		expected := "$apple:Int!$banana:Boolean!$zebra:String!"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("handles various Go types", func(t *testing.T) {
		var buf bytes.Buffer
		writeArgumentsFromMap(&buf, map[string]any{
			"str":   "hello",
			"num":   42,
			"float": 3.14,
			"bool":  true,
		})

		result := buf.String()
		// Check that all types are present (order is alphabetical)
		if !contains(result, "$bool:Boolean!") {
			t.Error("expected boolean type")
		}
		if !contains(result, "$float:Float!") {
			t.Error("expected float type")
		}
		if !contains(result, "$num:Int!") {
			t.Error("expected int type")
		}
		if !contains(result, "$str:String!") {
			t.Error("expected string type")
		}
	})
}

// TestCollectStructFieldsForArguments tests the collectStructFieldsForArguments helper function
func TestCollectStructFieldsForArguments(t *testing.T) {
	t.Run("collects fields with json tags", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		fields := collectStructFieldsForArguments(TestStruct{
			Name: "test",
			Age:  42,
		})

		if len(fields) != 2 {
			t.Fatalf("expected 2 fields, got %d", len(fields))
		}

		// Check sorting (age < name alphabetically)
		if fields[0].jsonName != "age" {
			t.Errorf("expected first field to be 'age', got %q", fields[0].jsonName)
		}
		if fields[1].jsonName != "name" {
			t.Errorf("expected second field to be 'name', got %q", fields[1].jsonName)
		}
	})

	t.Run("skips unexported fields", func(t *testing.T) {
		type TestStruct struct {
			Public  string `json:"public"`
			private string `json:"private"` //nolint:govet,unused // Intentionally unexported for testing
		}

		fields := collectStructFieldsForArguments(TestStruct{
			Public: "visible",
		})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "public" {
			t.Errorf("expected field 'public', got %q", fields[0].jsonName)
		}
	})

	t.Run("skips fields without json tags", func(t *testing.T) {
		type TestStruct struct {
			WithTag    string `json:"withTag"`
			WithoutTag string
		}

		fields := collectStructFieldsForArguments(TestStruct{})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "withTag" {
			t.Errorf("expected field 'withTag', got %q", fields[0].jsonName)
		}
	})

	t.Run("skips fields with json:- tag", func(t *testing.T) {
		type TestStruct struct {
			Include string `json:"include"`
			Exclude string `json:"-"`
		}

		fields := collectStructFieldsForArguments(TestStruct{})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "include" {
			t.Errorf("expected field 'include', got %q", fields[0].jsonName)
		}
	})

	t.Run("extracts field name from json tag with options", func(t *testing.T) {
		type TestStruct struct {
			Field string `json:"customName,omitempty"`
		}

		fields := collectStructFieldsForArguments(TestStruct{})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "customName" {
			t.Errorf("expected field name 'customName', got %q", fields[0].jsonName)
		}
	})

	t.Run("skips fields with empty json name", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:",omitempty"`
			Field2 string `json:"valid"`
		}

		fields := collectStructFieldsForArguments(TestStruct{})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "valid" {
			t.Errorf("expected field 'valid', got %q", fields[0].jsonName)
		}
	})

	t.Run("handles pointer to struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		fields := collectStructFieldsForArguments(&TestStruct{Name: "test"})

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].jsonName != "name" {
			t.Errorf("expected field 'name', got %q", fields[0].jsonName)
		}
	})

	t.Run("returns empty slice for struct with no valid fields", func(t *testing.T) {
		type TestStruct struct {
			unexported string `json:"unexported"` //nolint:govet,unused // Intentionally unexported for testing
			NoTag      string
		}

		fields := collectStructFieldsForArguments(TestStruct{})

		if len(fields) != 0 {
			t.Errorf("expected 0 fields, got %d", len(fields))
		}
	})

	t.Run("preserves field type information", func(t *testing.T) {
		type TestStruct struct {
			Str   string  `json:"str"`
			Num   int     `json:"num"`
			Float float64 `json:"float"`
			Bool  bool    `json:"bool"`
		}

		fields := collectStructFieldsForArguments(TestStruct{
			Str:   "test",
			Num:   42,
			Float: 3.14,
			Bool:  true,
		})

		if len(fields) != 4 {
			t.Fatalf("expected 4 fields, got %d", len(fields))
		}

		// Check types are preserved
		typeMap := make(map[string]reflect.Kind)
		for _, f := range fields {
			typeMap[f.jsonName] = f.fieldType.Kind()
		}

		if typeMap["bool"] != reflect.Bool {
			t.Error("expected bool field type")
		}
		if typeMap["float"] != reflect.Float64 {
			t.Error("expected float64 field type")
		}
		if typeMap["num"] != reflect.Int {
			t.Error("expected int field type")
		}
		if typeMap["str"] != reflect.String {
			t.Error("expected string field type")
		}
	})

	t.Run("preserves field values", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		fields := collectStructFieldsForArguments(TestStruct{
			Name: "Alice",
			Age:  30,
		})

		valueMap := make(map[string]any)
		for _, f := range fields {
			valueMap[f.jsonName] = f.value.Interface()
		}

		if valueMap["age"] != 30 {
			t.Errorf("expected age value 30, got %v", valueMap["age"])
		}
		if valueMap["name"] != "Alice" {
			t.Errorf("expected name value 'Alice', got %v", valueMap["name"])
		}
	})

	t.Run("panics on invalid type - string", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on string input")
			}
		}()

		collectStructFieldsForArguments("not a struct")
	})

	t.Run("panics on invalid type - int", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on int input")
			}
		}()

		collectStructFieldsForArguments(42)
	})

	t.Run("panics on invalid type - slice", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on slice input")
			}
		}()

		collectStructFieldsForArguments([]string{"not", "a", "struct"})
	})
}

// TestWriteArgumentsFromFields tests the writeArgumentsFromFields helper function
func TestWriteArgumentsFromFields(t *testing.T) {
	t.Run("empty fields produces empty string", func(t *testing.T) {
		var buf bytes.Buffer
		writeArgumentsFromFields(&buf, []argumentFieldInfo{})

		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})

	t.Run("single field", func(t *testing.T) {
		var buf bytes.Buffer
		fields := []argumentFieldInfo{
			{
				jsonName:  "id",
				fieldType: reflect.TypeOf(123),
				value:     reflect.ValueOf(123),
			},
		}

		writeArgumentsFromFields(&buf, fields)

		expected := "$id:Int!"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		var buf bytes.Buffer
		fields := []argumentFieldInfo{
			{
				jsonName:  "name",
				fieldType: reflect.TypeOf(""),
				value:     reflect.ValueOf("Alice"),
			},
			{
				jsonName:  "age",
				fieldType: reflect.TypeOf(0),
				value:     reflect.ValueOf(30),
			},
			{
				jsonName:  "active",
				fieldType: reflect.TypeOf(true),
				value:     reflect.ValueOf(true),
			},
		}

		writeArgumentsFromFields(&buf, fields)

		expected := "$name:String!$age:Int!$active:Boolean!"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("handles pointer types (optional)", func(t *testing.T) {
		var buf bytes.Buffer
		var optionalInt *int
		fields := []argumentFieldInfo{
			{
				jsonName:  "optional",
				fieldType: reflect.TypeOf(optionalInt),
				value:     reflect.ValueOf(optionalInt),
			},
		}

		writeArgumentsFromFields(&buf, fields)

		// Pointer types should not have ! at the end
		expected := "$optional:Int"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestProcessStructField tests the processStructField helper function
func TestProcessStructField(t *testing.T) {
	tests := []struct {
		name       string
		field      reflect.StructField
		value      reflect.Value
		wantSkip   bool
		wantName   string
		wantInline bool
	}{
		{
			name: "simple field with no tag",
			field: reflect.StructField{
				Name: "UserName",
				Type: reflect.TypeOf(""),
			},
			value:      reflect.ValueOf("test"),
			wantSkip:   false,
			wantName:   "userName",
			wantInline: false,
		},
		{
			name: "field with graphql tag",
			field: reflect.StructField{
				Name: "User",
				Type: reflect.TypeOf(""),
				Tag:  `graphql:"user(id: $userId)"`,
			},
			value:      reflect.ValueOf("test"),
			wantSkip:   false,
			wantName:   "user(id: $userId)",
			wantInline: false,
		},
		{
			name: "field with hyphen tag (should skip)",
			field: reflect.StructField{
				Name: "Internal",
				Type: reflect.TypeOf(""),
				Tag:  `graphql:"-"`,
			},
			value:      reflect.ValueOf("test"),
			wantSkip:   true,
			wantName:   "",
			wantInline: false,
		},
		{
			name: "anonymous field without tag (should inline)",
			field: reflect.StructField{
				Name:      "EmbeddedStruct",
				Type:      reflect.TypeOf(struct{}{}),
				Anonymous: true,
			},
			value:      reflect.ValueOf(struct{}{}),
			wantSkip:   false,
			wantName:   "",
			wantInline: true,
		},
		{
			name: "anonymous field with tag (should not inline)",
			field: reflect.StructField{
				Name:      "EmbeddedStruct",
				Type:      reflect.TypeOf(struct{}{}),
				Anonymous: true,
				Tag:       `graphql:"... on IssueComment"`,
			},
			value:      reflect.ValueOf(struct{}{}),
			wantSkip:   false,
			wantName:   "... on IssueComment",
			wantInline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := processStructField(tt.field, tt.value)

			if output.shouldSkip != tt.wantSkip {
				t.Errorf("shouldSkip = %v, want %v", output.shouldSkip, tt.wantSkip)
			}
			if output.name != tt.wantName {
				t.Errorf("name = %q, want %q", output.name, tt.wantName)
			}
			if output.isInline != tt.wantInline {
				t.Errorf("isInline = %v, want %v", output.isInline, tt.wantInline)
			}
		})
	}
}

// TestProcessStructField_scalar tests the scalar field handling specifically
func TestProcessStructField_scalar(t *testing.T) {
	t.Run("field with scalar tag", func(t *testing.T) {
		field := reflect.StructField{
			Name: "CustomDate",
			Type: reflect.TypeOf(""),
			Tag:  `graphql:"customDate" scalar:"true"`,
		}
		value := reflect.ValueOf("2024-01-01")

		output := processStructField(field, value)

		if !output.isScalar {
			t.Errorf("expected isScalar to be true for field with scalar tag")
		}
		if output.shouldSkip {
			t.Errorf("expected shouldSkip to be false")
		}
		if output.name != "customDate" {
			t.Errorf("expected name to be 'customDate', got %q", output.name)
		}
	})

	t.Run("field without scalar tag", func(t *testing.T) {
		field := reflect.StructField{
			Name: "NormalField",
			Type: reflect.TypeOf(""),
		}
		value := reflect.ValueOf("test")

		output := processStructField(field, value)

		if output.isScalar {
			t.Errorf("expected isScalar to be false for field without scalar tag")
		}
	})
}

// TestQuery_ErrorPath tests the error path in the query() function
func TestQuery_ErrorPath(t *testing.T) {
	t.Run("error from writeQuery is wrapped", func(t *testing.T) {
		// Use a map type which is not supported and should cause an error
		invalidQuery := map[string]string{"key": "value"}

		_, err := query(invalidQuery)
		if err == nil {
			t.Fatal("expected error from query with map type, got nil")
		}

		// Check that error is wrapped with "failed to write query"
		if !contains(err.Error(), "failed to write query") {
			t.Errorf("expected error to contain 'failed to write query', got: %v", err)
		}

		// Check that original error message is preserved
		if !contains(err.Error(), "not supported") {
			t.Errorf("expected error to contain 'not supported', got: %v", err)
		}
	})
}

// TestWriteQuery_MapError tests that maps produce an error
func TestWriteQuery_MapError(t *testing.T) {
	t.Run("map type produces error", func(t *testing.T) {
		var buf bytes.Buffer
		mapType := reflect.TypeOf(map[string]string{})
		mapValue := reflect.ValueOf(map[string]string{"key": "value"})

		err := writeQuery(&buf, mapType, mapValue, false)
		if err == nil {
			t.Fatal("expected error for map type, got nil")
		}

		if !contains(err.Error(), "not supported") {
			t.Errorf("expected error to contain 'not supported', got: %v", err)
		}
		if !contains(err.Error(), "[][2]any") {
			t.Errorf("expected error to suggest '[][2]any', got: %v", err)
		}
	})
}

// TestWriteSliceQuery_ErrorPath tests error handling in writeSliceQuery
func TestWriteSliceQuery_ErrorPath(t *testing.T) {
	t.Run("error from nested writeQuery is wrapped", func(t *testing.T) {
		var buf bytes.Buffer
		// Slice of maps - should error because maps are not supported
		sliceType := reflect.TypeOf([]map[string]string{})
		sliceValue := reflect.ValueOf([]map[string]string{{"key": "value"}})

		err := writeSliceQuery(&buf, sliceType, sliceValue)
		if err == nil {
			t.Fatal("expected error from slice of maps, got nil")
		}

		if !contains(err.Error(), "failed to write query for slice item") {
			t.Errorf("expected error to contain 'failed to write query for slice item', got: %v", err)
		}
	})
}

// TestWriteOrderedMapQuery_ErrorPath tests error handling in writeOrderedMapQuery
func TestWriteOrderedMapQuery_ErrorPath(t *testing.T) {
	t.Run("invalid array length produces error", func(t *testing.T) {
		var buf bytes.Buffer
		// [][3]any - should fail because only [][2]any is supported
		sliceType := reflect.TypeOf([][3]any{})
		sliceValue := reflect.ValueOf([][3]any{{"key", "value", "extra"}})

		err := writeOrderedMapQuery(&buf, sliceType, sliceValue)
		if err == nil {
			t.Fatal("expected error for array length != 2, got nil")
		}

		if !contains(err.Error(), "only arrays of len 2 are supported") {
			t.Errorf("expected error about array length, got: %v", err)
		}
	})

	t.Run("non-string key produces error", func(t *testing.T) {
		var buf bytes.Buffer
		// [][2]any with non-string key
		sliceType := reflect.TypeOf([][2]any{})
		sliceValue := reflect.ValueOf([][2]any{{123, "value"}})

		err := writeOrderedMapQuery(&buf, sliceType, sliceValue)
		if err == nil {
			t.Fatal("expected error for non-string key, got nil")
		}

		if !contains(err.Error(), "expected pair (string") {
			t.Errorf("expected error about string key, got: %v", err)
		}
	})

	t.Run("error from nested writeQuery is wrapped", func(t *testing.T) {
		var buf bytes.Buffer
		// [][2]any with map value - should error
		sliceType := reflect.TypeOf([][2]any{})
		sliceValue := reflect.ValueOf([][2]any{{"key", map[string]string{"nested": "map"}}})

		err := writeOrderedMapQuery(&buf, sliceType, sliceValue)
		if err == nil {
			t.Fatal("expected error from nested map value, got nil")
		}

		if !contains(err.Error(), "failed to write query for pair[1]") {
			t.Errorf("expected error to contain 'failed to write query for pair[1]', got: %v", err)
		}
	})
}

// TestWriteInterfaceQuery_ErrorPath tests error handling in writeInterfaceQuery
func TestWriteInterfaceQuery_ErrorPath(t *testing.T) {
	t.Run("error from nested writeQuery is wrapped", func(t *testing.T) {
		var buf bytes.Buffer
		// Interface containing a map - should error
		var iface any = map[string]string{"key": "value"}
		ifaceType := reflect.TypeOf((*any)(nil)).Elem()
		ifaceValue := reflect.ValueOf(iface)

		err := writeInterfaceQuery(&buf, ifaceType, ifaceValue, false)
		if err == nil {
			t.Fatal("expected error from interface containing map, got nil")
		}

		if !contains(err.Error(), "failed to write query for interface") {
			t.Errorf("expected error to contain 'failed to write query for interface', got: %v", err)
		}
	})

	t.Run("nil interface is handled", func(t *testing.T) {
		var buf bytes.Buffer
		var iface any
		ifaceType := reflect.TypeOf((*any)(nil)).Elem()
		ifaceValue := reflect.ValueOf(&iface).Elem()

		err := writeInterfaceQuery(&buf, ifaceType, ifaceValue, false)
		if err != nil {
			t.Fatalf("expected no error for nil interface, got: %v", err)
		}

		if buf.String() != "" {
			t.Errorf("expected empty output for nil interface, got: %q", buf.String())
		}
	})

	t.Run("interface containing nil pointer is handled", func(t *testing.T) {
		var buf bytes.Buffer
		var ptr *string
		iface := any(ptr)
		ifaceType := reflect.TypeOf((*any)(nil)).Elem()
		ifaceValue := reflect.ValueOf(iface)

		err := writeInterfaceQuery(&buf, ifaceType, ifaceValue, false)
		if err != nil {
			t.Fatalf("expected no error for interface with nil pointer, got: %v", err)
		}

		if buf.String() != "" {
			t.Errorf("expected empty output for interface with nil pointer, got: %q", buf.String())
		}
	})
}

// TestWriteStructQuery_ErrorPath tests error handling in writeStructQuery
func TestWriteStructQuery_ErrorPath(t *testing.T) {
	t.Run("error from nested field writeQuery is wrapped", func(t *testing.T) {
		var buf bytes.Buffer
		type TestStruct struct {
			InvalidField map[string]string
		}
		structType := reflect.TypeOf(TestStruct{})
		structValue := reflect.ValueOf(TestStruct{
			InvalidField: map[string]string{"key": "value"},
		})

		err := writeStructQuery(&buf, structType, structValue, false)
		if err == nil {
			t.Fatal("expected error from struct with map field, got nil")
		}

		if !contains(err.Error(), "failed to write query for struct field") {
			t.Errorf("expected error to contain 'failed to write query for struct field', got: %v", err)
		}
	})
}

// TestWriteQuery_PtrErrorPath tests error handling for pointer types in writeQuery
func TestWriteQuery_PtrErrorPath(t *testing.T) {
	t.Run("error from pointer element is wrapped", func(t *testing.T) {
		var buf bytes.Buffer
		// Pointer to map - should error
		mapVal := map[string]string{"key": "value"}
		ptrType := reflect.TypeOf(&mapVal)
		ptrValue := reflect.ValueOf(&mapVal)

		err := writeQuery(&buf, ptrType, ptrValue, false)
		if err == nil {
			t.Fatal("expected error from pointer to map, got nil")
		}

		if !contains(err.Error(), "failed to write query for ptr") {
			t.Errorf("expected error to contain 'failed to write query for ptr', got: %v", err)
		}
	})
}

// TestConstructOptions_InvalidType tests the error path for invalid option types
func TestConstructOptions_InvalidType(t *testing.T) {
	// Create a mock option with an invalid type
	invalidOption := &mockInvalidOption{}

	_, err := constructOptions([]Option{invalidOption})
	if err == nil {
		t.Fatal("expected error for invalid option type, got nil")
	}

	if !contains(err.Error(), "invalid query option type") {
		t.Errorf("expected error to contain 'invalid query option type', got: %v", err)
	}
}

// mockInvalidOption implements Option with an invalid type for testing
type mockInvalidOption struct{}

func (m *mockInvalidOption) Type() OptionType {
	return OptionType("invalid-option-type")
}

func (m *mockInvalidOption) String() string {
	return "invalid"
}

// TestConstructOperation_OptionError tests error propagation from constructOptions
func TestConstructOperation_OptionError(t *testing.T) {
	type TestQuery struct {
		Field string
	}

	_, err := ConstructQuery(TestQuery{}, nil, &mockInvalidOption{})
	if err == nil {
		t.Fatal("expected error from invalid option, got nil")
	}

	if !contains(err.Error(), "invalid query option type") {
		t.Errorf("expected error about invalid option type, got: %v", err)
	}
}
