package tagparser

import (
	"testing"
)

func TestParseGraphQLTag_SimpleFieldName(t *testing.T) {
	tag := "name"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "name" {
		t.Errorf("expected FieldName 'name', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
	if parsed.Alias != "" {
		t.Errorf("expected empty Alias, got '%s'", parsed.Alias)
	}
	if parsed.IsFragment {
		t.Error("expected IsFragment to be false")
	}
}

func TestParseGraphQLTag_FieldWithArguments(t *testing.T) {
	tag := "height(unit: METER)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "height" {
		t.Errorf("expected FieldName 'height', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "unit: METER" {
		t.Errorf("expected Arguments 'unit: METER', got '%s'", parsed.Arguments)
	}
	if parsed.Alias != "" {
		t.Errorf("expected empty Alias, got '%s'", parsed.Alias)
	}
}

func TestParseGraphQLTag_Alias(t *testing.T) {
	tag := "node1: node"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_AliasWithArguments(t *testing.T) {
	tag := `node1: node(id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	expectedArgs := `id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng=="`
	if parsed.Arguments != expectedArgs {
		t.Errorf("expected Arguments '%s', got '%s'", expectedArgs, parsed.Arguments)
	}
}

func TestParseGraphQLTag_Fragment(t *testing.T) {
	tag := "... on Droid"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "Droid" {
		t.Errorf("expected TypeName 'Droid', got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_FragmentNoTypename(t *testing.T) {
	tag := "..."

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "" {
		t.Errorf("expected empty TypeName, got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_SkipField(t *testing.T) {
	tag := "-"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "-" {
		t.Errorf("expected FieldName '-', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_WithWhitespace(t *testing.T) {
	tag := "  height(unit: METER)  "

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "height" {
		t.Errorf("expected FieldName 'height', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "unit: METER" {
		t.Errorf("expected Arguments 'unit: METER', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_EmptyString(t *testing.T) {
	tag := ""

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "" {
		t.Errorf("expected empty FieldName, got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_VariableInArguments(t *testing.T) {
	tag := "human(id: $id)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "human" {
		t.Errorf("expected FieldName 'human', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "id: $id" {
		t.Errorf("expected Arguments 'id: $id', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_ComplexRealWorldExample(t *testing.T) {
	tag := `node1: node(id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_MultipleColonsInArguments(t *testing.T) {
	tag := "field(a: 1, b: 2, c: 3)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "a: 1, b: 2, c: 3" {
		t.Errorf("expected Arguments 'a: 1, b: 2, c: 3', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_FragmentWithExtraWhitespace(t *testing.T) {
	tag := "  ...   on   Droid  "

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "Droid" {
		t.Errorf("expected TypeName 'Droid', got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_NoArguments(t *testing.T) {
	tag := "field()"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_UnbalancedParentheses(t *testing.T) {
	// Missing closing paren - should handle gracefully
	tag := "field(arg: value"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still extract field name
	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_NestedParentheses(t *testing.T) {
	tag := "field(arg: func(nested))"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	// Should capture everything inside outer parentheses
	if parsed.Arguments != "arg: func(nested)" {
		t.Errorf("expected Arguments 'arg: func(nested)', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_AliasWithColonInFieldName(t *testing.T) {
	// Field name can contain colons (only first colon is alias separator)
	tag := "alias: http://example.com"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "alias" {
		t.Errorf("expected Alias 'alias', got '%s'", parsed.Alias)
	}
	// The rest after the FIRST colon is the field name
	// (including subsequent colons which are part of the field name)
	if parsed.FieldName != "http://example.com" {
		t.Errorf("expected FieldName 'http://example.com', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_AliasWithArgumentsRealWorld(t *testing.T) {
	// Real-world pattern from sorare: alias with arguments
	tag := "shortDisplayName:displayName(short:true)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "shortDisplayName" {
		t.Errorf("expected Alias 'shortDisplayName', got '%s'", parsed.Alias)
	}
	if parsed.FieldName != "displayName" {
		t.Errorf("expected FieldName 'displayName', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "short:true" {
		t.Errorf("expected Arguments 'short:true', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_EscapedQuotesInArguments(t *testing.T) {
	// Real-world pattern from sorare: escaped quotes inside arguments
	tag := `videoUrl(derivative:\"low_res\")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "videoUrl" {
		t.Errorf("expected FieldName 'videoUrl', got '%s'", parsed.FieldName)
	}
	expectedArgs := `derivative:\"low_res\"`
	if parsed.Arguments != expectedArgs {
		t.Errorf("expected Arguments '%s', got '%s'", expectedArgs, parsed.Arguments)
	}
}

func TestParseGraphQLTag_LongFragmentTypename(t *testing.T) {
	// Real-world pattern from sorare: long typename in fragment
	tag := "... on SolanaTokenTransferAuthorizationRequest"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "SolanaTokenTransferAuthorizationRequest" {
		t.Errorf("expected TypeName 'SolanaTokenTransferAuthorizationRequest', got '%s'", parsed.TypeName)
	}
}
