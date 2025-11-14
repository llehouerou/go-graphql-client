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
