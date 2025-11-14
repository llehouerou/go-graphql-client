package graphql

import (
	"fmt"
	"reflect"
	"strings"
)

type constructOptionsOutput struct {
	operationName       string
	operationDirectives []string
}

func (coo constructOptionsOutput) OperationDirectivesString() string {
	operationDirectivesStr := strings.Join(coo.operationDirectives, " ")
	if operationDirectivesStr != "" {
		return fmt.Sprintf(" %s ", operationDirectivesStr)
	}
	return ""
}

func constructOptions(options []Option) (*constructOptionsOutput, error) {
	output := &constructOptionsOutput{}

	for _, option := range options {
		switch option.Type() {
		case optionTypeOperationName:
			output.operationName = option.String()
		case OptionTypeOperationDirective:
			output.operationDirectives = append(
				output.operationDirectives,
				option.String(),
			)
		default:
			return nil, fmt.Errorf("invalid query option type: %s", option.Type())
		}
	}

	return output, nil
}

// hasVariables checks if variables exist and should be used.
// Returns false for nil or empty maps, true otherwise.
func hasVariables(variables any) bool {
	if variables == nil {
		return false
	}
	reflectVal := reflect.ValueOf(variables)
	// If it's not a map, we have variables
	// If it's a map, only return true if it has entries
	return reflectVal.Kind() != reflect.Map || reflectVal.Len() > 0
}

// constructOperation builds a GraphQL operation string from struct and variables.
// operationType should be "query", "mutation", or "subscription".
// includeOperationTypeInDefault determines whether to prepend the operation type
// when no operation name or directives are specified (true for mutation/subscription, false for query).
func constructOperation(
	operationType string,
	v any,
	variables any,
	includeOperationTypeInDefault bool,
	options ...Option,
) (string, error) {
	query, err := query(v)
	if err != nil {
		return "", err
	}

	optionsOutput, err := constructOptions(options)
	if err != nil {
		return "", err
	}

	if hasVariables(variables) {
		return fmt.Sprintf(
			"%s %s(%s)%s%s",
			operationType,
			optionsOutput.operationName,
			queryArguments(variables),
			optionsOutput.OperationDirectivesString(),
			query,
		), nil
	}

	if optionsOutput.operationName == "" &&
		len(optionsOutput.operationDirectives) == 0 {
		if includeOperationTypeInDefault {
			return operationType + query, nil
		}
		return query, nil
	}

	return fmt.Sprintf(
		"%s %s%s%s",
		operationType,
		optionsOutput.operationName,
		optionsOutput.OperationDirectivesString(),
		query,
	), nil
}

// ConstructQuery builds GraphQL query string from struct and variables.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func ConstructQuery(v any, variables any, options ...Option) (string, error) {
	return constructOperation("query", v, variables, false, options...)
}

// ConstructMutation builds GraphQL mutation string from struct and variables.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func ConstructMutation(
	v any,
	variables any,
	options ...Option,
) (string, error) {
	return constructOperation("mutation", v, variables, true, options...)
}

// ConstructSubscription builds GraphQL subscription string from struct and variables.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func ConstructSubscription(
	v any,
	variables any,
	options ...Option,
) (string, error) {
	return constructOperation("subscription", v, variables, true, options...)
}
