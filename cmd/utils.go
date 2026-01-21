package cmd

import (
	"fmt"

	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/vektah/gqlparser/v2/ast"
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
