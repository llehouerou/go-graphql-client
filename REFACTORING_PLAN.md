# Refactoring Plan - Code Organization & Maintainability

This document outlines focused refactoring opportunities to improve code organization, readability, and maintainability across the go-graphql-client module.

## 1. Extract long method: `decodeObjectKey` in pkg/jsonutil/graphql.go
**Impact: High** | **Complexity: Medium** | **Location:** pkg/jsonutil/graphql.go:231-356

This 125-line method does too much. Extract into helper methods:
- `findFieldsForKey(key string) []fieldInfo` - First pass: field discovery
- `selectFieldsByFragment(fields []fieldInfo) []reflect.Value` - Second pass: fragment filtering
- Core method reduces to ~40 lines, much more readable

---

## 2. Extract struct variable handling from `queryArguments`
**Impact: High** | **Complexity: Medium** | **Location:** query.go:138-221

The struct case is 64 lines within an 83-line function. Extract to:
- `collectStructFieldsForArguments(typ, val) []fieldInfo` - Lines 166-204
- `writeArgumentsFromFields(buf, fields)` - Lines 211-217

Makes the main function a simple dispatcher between map vs struct cases.

---

## 3. Consolidate pointer/interface unwrapping patterns
**Impact: Medium** | **Complexity: Low** | **Files:** query.go, pkg/jsonutil/graphql.go

Replace manual unwrapping loops throughout the codebase with the existing `reflectutil.UnwrapToConcreteValue()`. Found at:
- query.go:256-258 (in writeStructQuery)
- query.go:442-444 (in writeInterfaceQuery)
- pkg/jsonutil/graphql.go:256-258, 364-366, 496-498, 596-599

---

## 4. Remove dead code: unused fragment filtering methods
**Impact: Low** | **Complexity: Low** | **Location:** pkg/jsonutil/graphql.go:109-142

Delete `shouldIncludeFragment` and `shouldIncludeFragmentByTag` (lines 109-142) marked with `//nolint:unused`. The filtering logic was refactored inline into `decodeObjectKey` and these are no longer called.

---

## 5. Extract field processing logic from `writeStructQuery`
**Impact: High** | **Complexity: Medium** | **Location:** query.go:299-410

The struct field loop (lines 337-405, 68 lines) is complex. Extract to:
```go
type fieldOutput struct {
    shouldSkip bool
    name string
    isInline bool
    value reflect.Value
}

func processStructField(f reflect.StructField, v reflect.Value) fieldOutput
```

---

## 6. Deduplicate `isTrue()` helper function
**Impact: Low** | **Complexity: Low** | **Files:** query.go:524, pkg/jsonutil/graphql.go:642

Same function exists in both locations. Extract to `internal/reflectutil/helpers.go` or similar and import from both locations.

---

## 7. Split mixed const block separating operation types and error codes
**Impact: Low** | **Complexity: Low** | **Location:** graphql.go:700-712

Lines 700-712 mix operationType enum with error code strings. Split into two const blocks for clarity:
```go
const (
    queryOperation operationType = iota
    mutationOperation
)

const (
    ErrRequestError  = "request_error"
    ErrJsonEncode    = "json_encode_error"
    // ...
)
```

---

## 8. Consolidate error decoration logic in `withRequest`/`withResponse`
**Impact: Medium** | **Complexity: Low** | **Location:** graphql.go:655-688

These methods (lines 655-688) are nearly identical. Extract common pattern:
```go
func (e Error) withDebugInfo(infoType string, headers http.Header, bodyReader io.Reader) Error
```

Then implement both methods as thin wrappers.

---

## 9. Add helper for numeric kind checking in `writeArgumentType`
**Impact: Low** | **Complexity: Low** | **Location:** query.go:265-266

Lines 265-266 list 10 numeric kinds that all map to "Int". Extract:
```go
func isIntegerKind(k reflect.Kind) bool {
    return k >= reflect.Int && k <= reflect.Uint64 && k != reflect.Uintptr
}
```

---

## 10. Group related decoder stack operations into a helper struct
**Impact: Medium** | **Complexity: High** | **Location:** pkg/jsonutil/graphql.go:70-97 and related methods

The decoder manages 3 parallel slices (`vs`, `fragmentTypes`, and state). Encapsulate as:
```go
type valueStack struct {
    values []stack
    fragmentTypes []string
}
// Methods: push, pop, filter, etc.
```

Reduces cognitive load and prevents sync bugs between parallel slices.

---

## Priority Recommendations

### High Priority (biggest impact on maintainability)
- #1: Extract decodeObjectKey
- #2: Extract queryArguments struct handling
- #5: Extract writeStructQuery field processing

### Medium Priority (reduce duplication)
- #3: Consolidate unwrapping patterns
- #8: Consolidate error decoration

### Low Priority (quick wins)
- #4: Remove dead code
- #6: Deduplicate isTrue
- #7: Split const block

---

## Notes
- All changes maintain backward compatibility
- Focus on internal refactoring without API changes
- Subscription code excluded as per requirements
