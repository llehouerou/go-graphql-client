# Refactoring Plan

Generated: 2025-11-14

## Overview

This document outlines 10 focused refactoring actions to improve code organization, readability, and maintainability across the go-graphql-client codebase.

**Total Estimated Effort**: 35-50 hours
**Current Test Coverage**: 80-88%
**Target Coverage**: 90%+

---

## Top 10 Refactoring Actions

### 1. FIX: WithRequestModifier Clone Pattern Bug ✅
**Location**: `graphql.go:446-494`
**Effort**: 1 hour | **Risk**: Low | **Impact**: High
**Status**: COMPLETED

**Issue**: `WithRequestModifier()` doesn't copy the `debug` field when creating a new client, causing state loss.

**Action**: Extract a `clone()` helper method that copies all fields:
```go
func (c *Client) clone() *Client {
    return &Client{
        url:             c.url,
        httpClient:      c.httpClient,
        requestModifier: c.requestModifier,
        debug:           c.debug,
    }
}

func (c *Client) WithDebug(debug bool) *Client {
    clone := c.clone()
    clone.debug = debug
    return clone
}

func (c *Client) WithRequestModifier(f RequestModifier) *Client {
    clone := c.clone()
    clone.requestModifier = f
    return clone
}
```

**Value**: Prevents field-copying bugs when adding future fields. Improves maintainability.

**Tests Implemented**:
- ✅ Test that `WithRequestModifier()` preserves `debug` field
- ✅ Test that `WithDebug()` preserves `requestModifier` field
- ✅ Test chaining: both `WithDebug().WithRequestModifier()` and `WithRequestModifier().WithDebug()`

**Results**:
- All tests pass (0 failures)
- Linter: 0 issues
- Coverage improved: 80.1% → 81.3%
- Bug fixed: `WithRequestModifier()` now correctly copies all Client fields
- Future-proof: Adding new fields to Client won't cause bugs

---

### 2. READABILITY: Remove Magic Numbers and Strings ✅
**Location**: Multiple files
**Effort**: 1-2 hours | **Risk**: Very Low | **Impact**: High
**Status**: COMPLETED

**Locations**:
- `ident/ident.go:178` - `for i := minInitialismLength; i <= len(word)-minInitialismLength`
- `pkg/jsonutil/graphql.go:579-583` - Uses `maxTemplateSliceSize` constant

**Action Taken**: Extracted named constants:
```go
// In ident/ident.go
const (
    minInitialismLength = 2
)

// In pkg/jsonutil/graphql.go
const (
    maxTemplateSliceSize = 1
)
```

**Results**:
- All tests pass (0 failures)
- Linter: 0 issues
- Self-documenting code improves readability
- Easier to maintain and modify in the future
- Added test for template slice error case (pkg/jsonutil/graphql_test.go:1250-1272)
- Coverage improved: pkg/jsonutil 88.5% → 89.5%
- decodeArrayStart coverage: 85.7% → 100.0%

**Value**: Self-documenting code, easier to maintain and modify.

---

### 3. CONSISTENCY: Consolidate Error Creation Patterns ✅
**Location**: `graphql.go` (throughout)
**Effort**: 3-4 hours | **Risk**: Low | **Impact**: High
**Status**: COMPLETED

**Issue**: Inconsistent error handling patterns:
- Sometimes uses `newError()`
- Sometimes uses `c.NewRequestError()` with full context
- Sometimes creates `Errors{}` directly
- Inconsistent `%w` wrapping

**Action Taken**:
- Created `newSimpleErrors()` helper for simple error cases
- Standardized all simple error cases to use `newSimpleErrors()`
- Ensured all error wrapping uses `%w` where appropriate
- Maintained existing `NewRequestError()` and `DecorateError()` for context-rich errors
- Added documentation to `newError()` function

**Results**:
- All tests pass (0 failures)
- Linter: 0 issues
- Consistent error creation pattern throughout graphql.go
- Cleaner, more maintainable error handling code

**Value**: Easier debugging, consistent error messages, better maintainability.

---

### 4. ORGANIZATION: Split query.go into Focused Files
**Location**: `query.go` (586 lines)
**Effort**: 3-4 hours | **Risk**: Very Low | **Impact**: High
**Status**: PENDING

**Issue**: Single file mixes multiple responsibilities:
- High-level query construction
- Variable argument formatting
- Low-level query writing
- Type-specific handlers

**Action**: Split into:
- `query.go` - Public API (ConstructQuery, ConstructMutation, ConstructSubscription)
- `query_arguments.go` - Variable argument formatting logic
- `query_writer.go` - Low-level writing logic and type handlers

**Value**: Much easier to navigate, clear separation of concerns.

---

### 5. COMPLEXITY: Split request() Method
**Location**: `graphql.go:135-255` (120 lines)
**Effort**: 4-6 hours | **Risk**: Medium | **Impact**: Very High
**Status**: PENDING

**Issue**: `request()` does too much in one method:
- HTTP request execution
- Gzip decompression handling
- Debug mode response copying
- Status code checking
- Response decoding
- Error decoration

**Action**: Extract focused helper methods:
- `handleGzipResponse(resp) (io.Reader, error)`
- `copyResponseForDebug(resp) ([]byte, error)`
- Use existing `BuildRequest()` and `DecodeResponse()` more directly

**Value**: Each step becomes testable in isolation. Much easier to understand control flow.

---

### 6. COMPLEXITY: Extract Query Handler Methods
**Location**: `query.go:399-582`
**Effort**: 6-8 hours | **Risk**: Medium | **Impact**: Medium
**Status**: PENDING

**Issue**: While `writeQuery()` delegates to type-specific handlers, `writeStructQuery()` is still 75 lines and handles multiple concerns.

**Action**:
- Break down `writeStructQuery()` into smaller pieces
- Extract field processing logic
- Clarify template pattern handling

**Value**: Core query construction becomes easier to understand and extend.

---

### 7. TECH DEBT: Address TODOs with Tests
**Location**: `pkg/jsonutil/graphql.go` (4 TODOs)
**Effort**: 4-5 hours | **Risk**: Low | **Impact**: Medium
**Status**: PENDING

**TODOs**:
- Line 503: Recursive handling uncertainty
- Line 551: Nested wrapper type handling
- Line 682: Performance optimization opportunity
- Line 744: Short-circuit optimization

**Action**:
- Write test cases for recursive and edge cases (lines 503, 551)
- Profile to determine if optimizations are needed (lines 682, 744)
- Document decisions or implement fixes
- Remove TODOs once resolved

**Value**: Removes uncertainty, improves test coverage, may uncover bugs.

---

### 8. QUALITY: Add Edge Case Test Coverage
**Location**: Various test files
**Effort**: 6-8 hours | **Risk**: Low | **Impact**: High
**Status**: PENDING

**Current Coverage**: 80-88%
**Target**: 90%+

**Gaps**:
- Recursive struct handling in unmarshaling
- Nested wrapper types
- Error paths in gzip handling
- Fragment matching edge cases
- Ordered map template copying

**Action**:
- Add tests for each TODO scenario
- Add fuzz testing for unmarshaling
- Add property-based tests for query construction

**Value**: Safety net for all refactorings. Catches regressions early.

---

### 9. DOCUMENTATION: Clarify Panic vs Error Return
**Location**: `query.go:189`
**Effort**: 2-3 hours | **Risk**: Medium | **Impact**: Medium
**Status**: PENDING

**Issue**: Code panics when variables aren't a struct/map:
```go
if typ.Kind() != reflect.Struct {
    panic(fmt.Sprintf("variables must be a struct or a map; got %T", variables))
}
```

**Decision Needed**:
1. Document that this is intentional for programming errors (add godoc)
2. Return error instead (breaking change, requires major version bump)

**Value**: Clearer API contract, better error handling guidance.

---

### 10. ADVANCED: Refactor decode() Loop ⚠️ HIGH RISK
**Location**: `pkg/jsonutil/graphql.go:191-264` (73 lines)
**Effort**: 8-10 hours | **Risk**: HIGH | **Impact**: High
**Status**: PENDING

**Issue**: Main unmarshaling loop handles all JSON token types in a complex nested switch with state management mixed into processing logic.

**Action**:
- Extract token handlers: `handleObjectToken()`, `handleArrayToken()`, `handleScalarToken()`
- Document state machine transitions
- Consider state pattern or clearer state machine design

**Value**: Core unmarshaling becomes more understandable.

**Warning**: Should only be attempted AFTER:
- Other refactorings are complete
- Test coverage is 90%+
- High confidence in the test suite

---

## Execution Phases

### Phase 1: Quick Wins (5-7 hours) ✅ COMPLETED
- [x] #1: Fix WithRequestModifier bug
- [x] #2: Remove magic numbers
- [x] #3: Consolidate error patterns

**Additional Work Completed**: Added comprehensive error path tests:
- Test for invalid gzip data handling
- Test for debug mode body read error
- Test for BuildRequest error paths (unmarshalable variables)
- Test for HTTP request execution error (network failures)
- **Coverage improved**: 80.9% → 82.4% (+1.5%)
- **request() function coverage**: 69.0% → 83.3% (+14.3%)

### Phase 2: Organization (7-10 hours)
- [ ] #4: Split query.go file
- [ ] #8: Add edge case tests (safety net - partially complete)

### Phase 3: Complexity Reduction (10-14 hours)
- [ ] #5: Split request() method
- [ ] #6: Extract query handlers

### Phase 4: Clean Up (6-8 hours)
- [ ] #7: Address TODOs
- [ ] #9: Document panic usage

### Phase 5: Advanced (8-10 hours, optional)
- [ ] #10: Refactor decode() loop (only after phases 1-4)

---

## Impact Summary

**Organization**: #3, #4 (split files, consolidate patterns)
**Readability**: #2, #5, #6 (magic numbers, split methods)
**Maintainability**: #1, #7, #8 (bug fixes, tests, tech debt)

**Immediate Priority**: Phase 1 (Quick Wins) to build momentum before larger refactorings.
