# Go GraphQL Client Refactoring Plan

## High Impact Changes

### 1. Centralize wrapper type and GraphQLType detection logic ✅
**Location**: `query.go:487`, `pkg/jsonutil/graphql.go:19-23`
- `wrapperMethodName` constant is defined in **two places**
- Wrapper detection logic (`MethodByName(wrapperMethodName)`) duplicated in both files
- `GraphqlTypeInterface` checks appear in 4+ locations

**Action**: Create `internal/reflectutil/graphql_types.go` with:
- Constants: `WrapperMethodName`, `WrapperFieldName`
- Functions: `IsWrapperType()`, `UnwrapValue()`, `ImplementsGraphQLType()`

**Impact**: Eliminates duplication, single source of truth, easier to maintain wrapper convention.

---

### 2. Decompose `writeQuery()` function (188 lines)
**Location**: `query.go:295-483`
- Single function handles: structs, slices, arrays, interfaces, pointers, ordered maps
- Deeply nested switch/case with complex logic
- Hard to test individual behaviors

**Action**: Extract methods:
- `writeStructQuery(w, t, v, inline)` - handles reflect.Struct case
- `writeSliceQuery(w, t, v)` - handles reflect.Slice case
- `writeOrderedMapQuery(w, v)` - handles [][2]any pattern
- `writeInterfaceQuery(w, t, v, inline)` - handles reflect.Interface case

**Impact**: Improves testability, readability, and reduces cognitive complexity.

---

### 3. Decompose `decode()` method (318 lines)
**Location**: `pkg/jsonutil/graphql.go:160-479`
- Main decoding loop handles objects, arrays, fragments, wrappers all in one method
- Multiple levels of nesting (for loops within switch within for)
- Most complex function in the codebase

**Action**: Extract methods:
- `decodeObjectStart(d *decoder)` - handles '{' token
- `decodeArrayStart(d *decoder)` - handles '[' token
- `decodeObjectKey(d *decoder, key string)` - handles object key processing
- `decodeValue(d *decoder, tok any)` - handles scalar values

**Impact**: Dramatically improves readability and makes unit testing individual decode phases possible.

---

### 4. Centralize magic string constants
**Location**: Throughout codebase
- `"scalar"` tag - `query.go:424`, `pkg/jsonutil/graphql.go:600`
- `"graphql"` tag name - used in 10+ places
- `"__typename"` - `pkg/jsonutil/graphql.go:357`
- `"Value"` field name - hardcoded as string

**Action**: Create `types/constants.go`:
```go
const (
    GraphQLTag     = "graphql"
    ScalarTag      = "scalar"
    TypenameField  = "__typename"
    FragmentPrefix = "..."
    FragmentOnPrefix = "... on "
)
```

**Impact**: Prevents typos, makes refactoring safer, improves code searchability.

---

## Medium Impact Changes

### 5. Abstract error decoration pattern in Client
**Location**: `graphql.go` - 9 `if c.debug` checks, 8 `withRequest/withResponse` calls
- Repetitive pattern: create error → check debug → add context
- Lines 151-154, 173-176, 203-206, 236-240, etc.

**Action**: Create helper methods on Client:
```go
func (c *Client) decorateError(err Error, req *http.Request, resp *http.Response, ...) Error
func (c *Client) newRequestError(code string, err error, req, resp, ...) Error
```

**Impact**: Reduces boilerplate, ensures consistent error decoration, easier to modify debug behavior.

---

### 6. Improve GraphQL tag parsing
**Location**: `pkg/jsonutil/graphql.go:632,657` - TODOs indicate current parsing is inadequate
- Current: `strings.TrimSpace(value)` + basic `HasPrefix`/`Index` checks
- Doesn't handle complex tags with multiple components properly

**Action**: Create `internal/tagparser` package with proper parser:
```go
type ParsedTag struct {
    FieldName  string
    Arguments  string
    Alias      string
    IsFragment bool
    TypeName   string // for fragments
}

func ParseGraphQLTag(tag string) (ParsedTag, error)
```

**Impact**: More robust tag handling, fixes edge cases, removes TODOs.

---

### 7. Consolidate reflection utilities
**Location**: `internal/reflectutil/safe.go` exists but underutilized
- Add common patterns currently duplicated:
  - Unwrapping pointers/interfaces (appears 20+ times)
  - Nil checking for different kinds (appears 15+ times)
  - Type interface checks

**Action**: Extend `internal/reflectutil` with:
```go
func UnwrapToConcreteValue(v reflect.Value) reflect.Value
func IsNilValue(v reflect.Value) bool
func NewZeroOrPointerValue(t reflect.Type) reflect.Value
```

**Impact**: Reduces reflection complexity in main logic, centralizes tricky unsafe operations.

---

### 8. Extract HTTP request building logic
**Location**: `graphql.go:116-260` - `request()` method does too much
- Builds JSON request body
- Creates HTTP request
- Executes request
- Handles gzip
- Decodes response
- All error decoration

**Action**: Split into:
```go
func (c *Client) buildRequest(ctx, query, variables) (*http.Request, error)
func (c *Client) executeRequest(req) (*http.Response, io.Reader, error)
func (c *Client) decodeResponse(resp, reader) ([]byte, Errors)
```

**Impact**: Better separation of concerns, easier to test HTTP logic independently.

---

## Lower Impact (Nice to Have)

### 9. Add structured types for error extensions
**Location**: `graphql.go:376-382` - `Error.Extensions` is `map[string]any`
- Internal extensions are also untyped maps
- No type safety for common fields like "code", "request", "response"

**Action**: Create typed extension structs:
```go
type ErrorExtensions struct {
    Code     string
    Internal *InternalExtensions
    Custom   map[string]any
}

type InternalExtensions struct {
    Request  *RequestInfo
    Response *ResponseInfo
    Error    error
}
```

**Impact**: Improves type safety, better IDE autocomplete, self-documenting.

---

### 10. Document immutable Client pattern
**Location**: `graphql.go:351-367` - `WithRequestModifier` and `WithDebug`
- These methods return NEW Client instances (immutable/functional pattern)
- Not obvious from API, could surprise users expecting mutation
- No clear "builder" pattern established

**Action**: Add documentation and consider consistent API:
```go
// WithDebug returns a new Client with debug mode enabled.
// The original Client is not modified (immutable pattern).
func (c *Client) WithDebug(debug bool) *Client

// Alternative: Add explicit NewClientBuilder() if builder pattern preferred
```

**Impact**: Clearer API expectations, better developer experience, prevents confusion.

---

## Summary Statistics
- **2 large functions** need decomposition (188 and 318 lines)
- **2 constants** duplicated across packages
- **9 debug checks** could be abstracted
- **6 TODOs** could be addressed through these refactorings
- **Magic strings** used in 30+ locations

These refactorings focus on **code organization** (extracting large functions), **eliminating duplication** (constants, reflection logic), and **improving maintainability** (centralized utilities, typed structures).
