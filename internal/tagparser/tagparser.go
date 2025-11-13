package tagparser

import "strings"

// ParsedTag represents a parsed GraphQL struct tag.
type ParsedTag struct {
	// FieldName is the GraphQL field name (after alias if present).
	FieldName string
	// Arguments contains the content inside parentheses, if any.
	Arguments string
	// Alias is the field alias (before the colon), if any.
	Alias string
	// IsFragment indicates whether this is a GraphQL fragment ("...").
	IsFragment bool
	// TypeName is the typename for fragments ("... on TypeName").
	TypeName string
}

// ParseGraphQLTag parses a GraphQL struct tag value and returns structured information.
// Examples:
//   - "name" -> {FieldName: "name"}
//   - "height(unit: METER)" -> {FieldName: "height", Arguments: "unit: METER"}
//   - "node1: node(id: $id)" -> {FieldName: "node", Alias: "node1", Arguments: "id: $id"}
//   - "... on Droid" -> {IsFragment: true, TypeName: "Droid"}
func ParseGraphQLTag(tag string) (ParsedTag, error) {
	tag = strings.TrimSpace(tag)

	var parsed ParsedTag

	// Handle empty string
	if tag == "" {
		return parsed, nil
	}

	// Handle skip field
	if tag == "-" {
		parsed.FieldName = "-"
		return parsed, nil
	}

	// Handle fragments
	if strings.HasPrefix(tag, "...") {
		parsed.IsFragment = true
		// Remove "..." prefix
		remaining := strings.TrimSpace(tag[3:])
		// Check for "on TypeName"
		if strings.HasPrefix(remaining, "on ") {
			parsed.TypeName = strings.TrimSpace(remaining[3:])
		}
		return parsed, nil
	}

	// Find arguments first (content in parentheses)
	var fieldPart string
	parenIdx := strings.Index(tag, "(")
	if parenIdx != -1 {
		// Extract arguments
		closeIdx := strings.LastIndex(tag, ")")
		if closeIdx > parenIdx {
			parsed.Arguments = tag[parenIdx+1 : closeIdx]
		}
		fieldPart = strings.TrimSpace(tag[:parenIdx])
	} else {
		fieldPart = tag
	}

	// Handle alias in the field part (before arguments)
	if colonIdx := strings.Index(fieldPart, ":"); colonIdx != -1 {
		parsed.Alias = strings.TrimSpace(fieldPart[:colonIdx])
		parsed.FieldName = strings.TrimSpace(fieldPart[colonIdx+1:])
	} else {
		parsed.FieldName = strings.TrimSpace(fieldPart)
	}

	return parsed, nil
}
