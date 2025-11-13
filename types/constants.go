package types

// GraphQL-related constants used throughout the codebase.
// Centralizing these prevents typos and makes refactoring safer.
const (
	// GraphQLTag is the struct tag name used to specify GraphQL field
	// mappings, arguments, and directives.
	GraphQLTag = "graphql"

	// ScalarTag is the struct tag name used to mark a field as a scalar
	// type that should not be recursively expanded during query
	// construction.
	ScalarTag = "scalar"

	// TypenameField is the GraphQL introspection field used for type
	// discrimination in unions and interfaces.
	TypenameField = "__typename"

	// FragmentPrefix is the prefix used in struct tags to indicate an
	// inline fragment.
	FragmentPrefix = "..."

	// FragmentOnPrefix is the full prefix for typed inline fragments
	// (e.g., "... on Droid").
	FragmentOnPrefix = "... on "
)
