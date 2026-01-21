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

func runValidate(querySource string, queryContent string, schema *ast.Schema) *ValidationResult {
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

func formatValidationResultText(result *ValidationResult, sourceName string) string {
	if result.Valid {
		return "✓ Query is valid"
	}

	var output string
	if len(result.Errors) == 1 {
		output = "✗ Query has 1 error:\n"
	} else {
		output = fmt.Sprintf("✗ Query has %d errors:\n", len(result.Errors))
	}

	for _, err := range result.Errors {
		if len(err.Locations) > 0 {
			loc := err.Locations[0]
			output += fmt.Sprintf("  %s:%d:%d - %s\n", sourceName, loc.Line, loc.Column, err.Message)
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

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
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
	RunE: func(cmd *cobra.Command, args []string) error {
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

		result := runValidate(querySource, queryContent, schema)

		// Output the result
		switch outputFormat {
		case "json":
			output, err := formatValidationResultJSON(result)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), output)
		default:
			fmt.Fprint(cmd.OutOrStdout(), formatValidationResultText(result, querySource))
		}

		// Return error if validation failed (causes exit code 1)
		if !result.Valid {
			return ErrValidationFailed
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
