package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var tableStyle = lipgloss.NewStyle().PaddingRight(1)

func makeTable() *table.Table {
	return table.New().
		Width(120).
		Wrap(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			return tableStyle
		})
}

const maxSuggestionDistance = 5

func findClosest(input string, candidates []string) string {
	minDist := -1
	closest := ""
	for _, c := range candidates {
		dist := levenshtein.ComputeDistance(input, c)
		if minDist == -1 || dist < minDist {
			minDist = dist
			closest = c
		}
	}
	if minDist > maxSuggestionDistance {
		return ""
	}
	return closest
}

// validateTypeExists checks if a type exists in the schema and returns a helpful
// error with a "did you mean" suggestion if it doesn't.
// The context parameter is used to customize the error message (e.g., "type", "interface").
func validateTypeExists(schema *ast.Schema, typeName, context string) error {
	if schema.Types[typeName] == nil {
		var typeNames []string
		for name := range schema.Types {
			typeNames = append(typeNames, name)
		}
		if suggestion := findClosest(typeName, typeNames); suggestion != "" {
			return fmt.Errorf("%s '%s' does not exist in schema, did you mean '%s'?", context, typeName, suggestion)
		}
		return fmt.Errorf("%s '%s' does not exist in schema", context, typeName)
	}
	return nil
}

// typeToString converts an ast.Type to a human-readable string (e.g., "String!", "[User!]!").
func typeToString(typeDef *ast.Type) string {
	requiredStr := ""
	if typeDef.NonNull {
		requiredStr = "!"
	}
	if typeDef.Elem != nil {
		return fmt.Sprintf("[%s]%s", typeToString(typeDef.Elem), requiredStr)
	}
	return typeDef.NamedType + requiredStr
}

// getBaseTypeName returns the underlying named type from a (potentially wrapped) type.
// For example, [User!]! returns "User".
func getBaseTypeName(t *ast.Type) string {
	if t.Elem != nil {
		return getBaseTypeName(t.Elem)
	}
	return t.NamedType
}

// filterSlice returns a new slice containing only the elements that satisfy the predicate.
func filterSlice[T any](items []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func loadSchema() (*ast.Schema, error) {
	path, err := filepath.Abs(schemaFilePath)
	if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	strVal := string(bytes)

	fileName := filepath.Base(path)
	source := ast.Source{
		Input: strVal,
		Name:  fileName,
	}
	schema, err := gqlparser.LoadSchema(&source)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func loadCliForSchema() (*ast.Schema, error) {
	schema, err := loadSchema()

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("schema file does not exist: %s", schemaFilePath)
		}
		var parsingError *gqlerror.Error

		if errors.As(err, &parsingError) {
			return nil, fmt.Errorf("GraphQL schema parsing error: %v", parsingError)
		}

		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	return schema, nil
}
