# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a GraphQL client library for Go, forked from `shurcooL/graphql` with extended features including subscription support via WebSocket. The library provides a reflection-based approach to construct GraphQL queries from Go structs and unmarshal responses back into those structs.

## Key Commands

### Using Makefile (Recommended)
```bash
# Build the project
make build

# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run linters (golangci-lint with custom config)
make lint

# Format code with gofmt
make fmt

# Format code with golines (80 char lines, uses goimports-reviser)
make format

# Run go vet
make vet

# Build all examples
make examples

# Clean build artifacts
make clean
```

### Direct Go Commands
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test .
go test ./pkg/jsonutil
go test ./ident

# Run a single test
go test -run TestName

# Run tests with verbose output
go test -v ./...

# Build (library, no main package)
go build ./...

# Install dependencies
go mod download
go mod tidy
```

### Examples
```bash
# Run subscription example
go run ./example/subscription/main.go

# Run graphqldev example
go run ./example/graphqldev/main.go

# Run realworld example
go run ./example/realworld/main.go
```

## Code Quality & Modernization

This project uses modern Go practices and tooling:

### Go Version
- **Minimum**: Go 1.25
- Uses modern Go features: `any` type alias, `io.ReadAll`, `reflect.PointerTo`

### Code Formatting
- **golines**: Enforces 80-character line limits with automatic wrapping
  - Command: `make format`
  - Config: Uses `goimports-reviser` as base formatter, preserves struct tags
  - Applied across entire codebase for consistency

### Linting
- **golangci-lint v2.6**: Comprehensive static analysis
  - Config: `.golangci.yml` (version 2 format)
  - Enabled linters: errcheck, govet, ineffassign, staticcheck, unused, misspell, unconvert
  - Custom exclusions for test files and examples
  - Disabled overly strict checks (shadow, naming conventions that would break API)

### Dependencies
All dependencies updated to latest stable versions:
- `google/uuid`: v1.6.0
- `nhooyr.io/websocket`: v1.8.17 (marked for migration to coder/websocket)
- `graph-gophers/graphql-go`: v1.8.0
- `gorilla/websocket`: v1.5.3

### Deprecated Patterns Removed
- ✅ Replaced `io/ioutil` with `io` package (deprecated since Go 1.16)
- ✅ Replaced `interface{}` with `any` (162 occurrences across 10 files)
- ✅ Replaced `reflect.PtrTo()` with `reflect.PointerTo()` (deprecated since Go 1.22)
- ✅ Fixed all linter issues (36 errcheck, 11 staticcheck, 8 unused)
- ✅ Fixed comment typos (identifier, compatibility)

### Error Handling
- Explicit error handling using blank identifier (`_ =`) where appropriate
- Proper error bubbling in cleanup paths (e.g., `subscription.Close()`)
- Uses `//nolint` directives with explanations for intentional suppressions

## Architecture

### Core Components

**1. Client (`graphql.go`)**: The main GraphQL HTTP client
- Handles query and mutation operations via POST requests
- Supports debug mode with detailed request/response logging
- Uses reflection to construct queries from Go structs via `query.go`
- Returns structured `Errors` type for GraphQL errors

**2. Subscription Client (`subscription.go`)**: WebSocket-based subscription client
- Implements Apollo's `subscriptions-transport-ws` protocol
- Manages WebSocket lifecycle: connection, reconnection, keep-alive
- Supports multiple concurrent subscriptions with unique IDs
- Event-driven with callbacks for `OnConnected`, `OnDisconnected`, `OnError`
- Uses `nhooyr.io/websocket` by default but supports custom WebSocket implementations

**3. Query Construction (`query.go`)**: Reflection-based query builder
- Three main functions: `ConstructQuery`, `ConstructMutation`, `ConstructSubscription`
- Uses Go struct tags (`graphql:"..."`) to specify GraphQL field names and arguments
- Supports variables with type inference from Go types
- Handles inline fragments with `... on TypeName` syntax
- Supports operation names and directives via `Option` interface

**4. JSON Unmarshaling (`pkg/jsonutil/graphql.go`)**: Custom JSON decoder
- `UnmarshalGraphQL()` decodes GraphQL responses into Go structs
- Handles GraphQL-specific patterns: fragments, embedded structs, ordered maps
- Uses `__typename` discrimination for union/interface type resolution
- Supports both struct fields and ordered maps (`[][2]interface{}`)
- Includes fragment matching logic to populate only the correct union/interface variant

### Key Patterns

**Struct Tags**: The `graphql` struct tag drives both query construction and unmarshaling:
- Field arguments: `graphql:"height(unit: METER)"`
- Variables: `graphql:"human(id: $id)"`
- Inline fragments: `graphql:"... on Droid"`
- Skip field: `graphql:"-"`
- Custom scalars: `scalar:"true"` (prevents expansion during query generation)

**Type System**:
- `ID` type for GraphQL IDs
- `GraphQLType` interface (`types/types.go`) for custom type name specification
- Reflection-based type mapping: Go types → GraphQL types
- Special handling for slices (lists), pointers (nullable), interfaces

**Subscription Protocol**: Follows Apollo's message types:
- `GQL_CONNECTION_INIT` → `GQL_CONNECTION_ACK`
- `GQL_START` → `GQL_DATA` (multiple) → `GQL_COMPLETE`
- `GQL_CONNECTION_KEEP_ALIVE` for connection maintenance
- `GQL_STOP` for unsubscribing

**Error Handling**:
- GraphQL errors are returned as `Errors` (slice of `Error`)
- Partial data can exist alongside errors
- Debug mode adds request/response details to error extensions

### Fragment Matching

The library handles GraphQL unions and interfaces using inline fragments with `__typename` discrimination:
- During unmarshaling, captures `__typename` from response
- Filters inline fragments to populate only the matching type
- Supports both struct fields and ordered map keys as fragments
- See `pkg/jsonutil/graphql.go:84-109` for fragment filtering logic

### Ordered Maps

GraphQL requires fields in specific order for mutations. Use `[][2]interface{}` instead of regular maps:
```go
// [][2]interface{} is treated as an ordered map
m := [][2]interface{}{
    {"createUser(login: $login1)", &CreateUser{}},
    {"createUser(login: $login2)", &CreateUser{}},
}
```

## Important Implementation Details

1. **Variable Requirements**: When constructing queries with variables, both the struct tag must reference the variable (e.g., `$id`) AND the variable must be passed in the variables map.

2. **Scalar Types**: Types implementing `json.Unmarshaler` or `ID` are treated as scalars and not recursively expanded during query construction.

3. **WebSocket Reconnection**: The subscription client automatically retries connection with exponential backoff until `retryTimeout` is reached.

4. **Template Slices**: When unmarshaling arrays, the first element in the target slice acts as a template that gets copied for each array item.

5. **Pointer vs Value**: Pointer types in structs indicate optional/nullable GraphQL fields. Value types are required (append `!` in GraphQL schema).

6. **Pre-built Queries**: `Exec()` and `ExecRaw()` methods allow executing dynamically-constructed query strings (useful for CLI tools or dynamic filtering).
