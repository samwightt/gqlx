/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/samwightt/gqlx/pkg/diagnostic"
	"github.com/spf13/cobra"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/validator"
)

// ErrValidationFailed is returned when a query fails validation.
// This is a sentinel error that indicates the query is invalid,
// not that the command itself failed.
var ErrValidationFailed = errors.New("validation failed")

func convertGQLErrors(errs gqlerror.List) []ValidationError {
	var result []ValidationError
	for _, err := range errs {
		valErr := ValidationError{
			Message: err.Message,
			Rule:    err.Rule,
		}
		for _, loc := range err.Locations {
			valErr.Locations = append(valErr.Locations, Location{
				Line:   loc.Line,
				Column: loc.Column,
			})
		}
		result = append(result, valErr)
	}
	return result
}

func validateQuery(querySource string, queryContent string, schema *ast.Schema) *ValidationResult {
	// Parse query document
	source := &ast.Source{Input: queryContent, Name: querySource}
	doc, parseErr := gqlparser.LoadQuery(schema, source.Input)
	if parseErr != nil {
		// Parse errors are also validation failures
		return &ValidationResult{Valid: false, Errors: convertGQLErrors(parseErr)}
	}

	// Validate against schema
	errs := validator.Validate(schema, doc)
	if len(errs) > 0 {
		return &ValidationResult{Valid: false, Errors: convertGQLErrors(errs)}
	}

	return &ValidationResult{Valid: true}
}

// Validation Error Display
//
// gqlparser returns errors with a Rule name (e.g., "FieldsOnCorrectType") and
// Location (line, column). However, the Location only has start position - no
// end position or span length.
//
// To show nice underlines like Rust/Elm, we handle specific rules specially:
// - For known rules, we parse the error message to extract relevant info
//   (field name, type name) and use that to calculate span length and suggestions.
// - For unknown rules, we fall back to a single caret (^).
//
// This approach lets us progressively add nicer error display for specific
// validation rules while still handling everything else gracefully.

// Regex to parse FieldsOnCorrectType error messages
// Example: Cannot query field "badField" on type "Query".
var fieldsOnCorrectTypeRegex = regexp.MustCompile(`Cannot query field "([^"]+)" on type "([^"]+)"`)

// parseFieldsOnCorrectTypeError extracts field name and type name from the error message.
// Returns empty strings if the message doesn't match.
func parseFieldsOnCorrectTypeError(message string) (fieldName, typeName string) {
	matches := fieldsOnCorrectTypeRegex.FindStringSubmatch(message)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return "", ""
}

// errorSpanLength returns the length to underline for a given error.
// For known rules, it calculates the actual span. Otherwise returns 1.
func errorSpanLength(err ValidationError) int {
	switch err.Rule {
	case "FieldsOnCorrectType":
		fieldName, _ := parseFieldsOnCorrectTypeError(err.Message)
		if fieldName != "" {
			return len(fieldName)
		}
	}
	return 1
}

// detectZshEscapeIssue checks if a parse error might be caused by zsh's history
// expansion escaping `!` as `\!`. Returns a help message if detected.
func detectZshEscapeIssue(err ValidationError, sourceContent string, sourceName string) string {
	if sourceName != "stdin" {
		return ""
	}
	// Check if content contains \! which is likely zsh escape
	if !strings.Contains(sourceContent, `\!`) {
		return ""
	}
	// Check if error is near a \! sequence
	if len(err.Locations) == 0 {
		return ""
	}
	loc := err.Locations[0]
	lines := strings.Split(sourceContent, "\n")
	if loc.Line < 1 || loc.Line > len(lines) {
		return ""
	}
	line := lines[loc.Line-1]
	// Check if there's a \! at or near the error column
	col := loc.Column - 1
	if col >= 0 && col < len(line)-1 && line[col] == '\\' && line[col+1] == '!' {
		return "it looks like zsh escaped `!` as `\\!`. Try using a heredoc instead:\n" +
			"       cat <<'EOF' | gqlx validate\n" +
			"       query { ... }\n" +
			"       EOF"
	}
	return ""
}

// errorSuggestion returns a "did you mean" suggestion for the error, if applicable.
func errorSuggestion(err ValidationError, schema *ast.Schema) string {
	switch err.Rule {
	case "FieldsOnCorrectType":
		fieldName, typeName := parseFieldsOnCorrectTypeError(err.Message)
		if fieldName == "" || typeName == "" {
			return ""
		}

		// Look up the type in the schema
		typeDef := schema.Types[typeName]
		if typeDef == nil {
			return ""
		}

		// Find closest match
		closest := findClosest(fieldName, pluck(typeDef.Fields, func(f *ast.FieldDefinition) string { return f.Name }))
		if closest != "" {
			return fmt.Sprintf("did you mean `%s`?", closest)
		}
	}
	return ""
}

func formatValidationResultText(result *ValidationResult, sourceName string, sourceContent string, schema *ast.Schema) string {
	if result.Valid {
		return "✓ Query is valid"
	}

	lines := strings.Split(sourceContent, "\n")

	var output string
	if len(result.Errors) == 1 {
		output = "✗ Query has 1 error:\n"
	} else {
		output = fmt.Sprintf("✗ Query has %d errors:\n", len(result.Errors))
	}

	for _, err := range result.Errors {
		if len(err.Locations) > 0 {
			loc := err.Locations[0]
			output += diagnostic.RenderLocation(sourceName, loc.Line, loc.Column) + "\n"

			// Get the source line if available
			if loc.Line > 0 && loc.Line <= len(lines) {
				sourceLine := lines[loc.Line-1]
				length := errorSpanLength(err)
				output += diagnostic.RenderSnippet(sourceLine, loc.Line, loc.Column, length, err.Message) + "\n"
			}

			// Check for zsh escape issue first
			if zshHelp := detectZshEscapeIssue(err, sourceContent, sourceName); zshHelp != "" {
				output += "  = help: " + zshHelp + "\n"
			} else if suggestion := errorSuggestion(err, schema); suggestion != "" {
				// Add suggestion if available
				output += "  = help: " + suggestion + "\n"
			}
		} else {
			output += fmt.Sprintf("  %s\n", err.Message)
		}
	}

	return output
}

func formatValidationResultJSON(result *ValidationResult) (string, error) {
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [file]",
		Short: "Type-check a GraphQL query against the schema",
		Long: `Validates a GraphQL query, mutation, or subscription against the schema.

The query can be provided as a file path argument or piped via stdin.

Exit codes:
  0 - Query is valid
  1 - Query has validation or parse errors

Output formats:
  text    Human-readable error messages with locations
  json    {"valid": bool, "errors": [...]}`,
		Example: `  # Validate from a file
  gqlx validate query.graphql

  # Validate from stdin
  echo "query { user { id } }" | gqlx validate

  # JSON output for CI integration
  gqlx validate query.graphql -f json`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runValidateCmd,
	}

	return cmd
}

func runValidateCmd(cmd *cobra.Command, args []string) error {
	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	var queryContent string
	var querySource string

	if len(args) == 1 {
		// Read from file
		querySource = args[0]
		bytes, err := os.ReadFile(querySource)
		if err != nil {
			return fmt.Errorf("failed to read query file: %w", err)
		}
		queryContent = string(bytes)
	} else {
		// Read from stdin
		querySource = "stdin"
		bytes, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		queryContent = string(bytes)
	}

	result := validateQuery(querySource, queryContent, schema)

	// Output the result
	switch outputFormat {
	case "json":
		output, err := formatValidationResultJSON(result)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), output)
	default:
		fmt.Fprint(cmd.OutOrStdout(), formatValidationResultText(result, querySource, queryContent, schema))
	}

	// Return error if validation failed (causes exit code 1)
	if !result.Valid {
		return ErrValidationFailed
	}

	return nil
}
