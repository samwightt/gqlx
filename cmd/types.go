/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

func kindToString(kind string) string {
	switch kind {
	case "SCALAR":
		return "scalar"
	case "OBJECT":
		return "type"
	case "INTERFACE":
		return "interface"
	case "UNION":
		return "union"
	case "ENUM":
		return "enum"
	case "INPUT_OBJECT":
		return "input"
	default:
		return strings.ToLower(kind)
	}
}

func formatTypeText(t TypeInfo) string {
	kind := kindToString(t.Kind)
	if t.Description != "" {
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		return fmt.Sprintf("%s %s # %s", kind, t.Name, desc)
	}
	return fmt.Sprintf("%s %s", kind, t.Name)
}

func formatTypesPretty(types []TypeInfo) string {
	tbl := makeTable()

	for _, t := range types {
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		tbl.Row(kindToString(t.Kind), t.Name, desc)
	}
	tbl.Headers("kind", "name", "description")

	return tbl.String()
}

var implementsFilter string
var hasFieldFilter []string
var kindFilter []string
var usedByFilter []string
var usedByAnyFilter []string
var notUsedByFilter []string
var notUsedByAllFilter []string
var typesNameFilter string
var typesNameRegexFilter string
var typesHasDescriptionFilter bool
var scalarFilter bool
var typeFilter bool
var interfaceFilter bool
var unionFilter bool
var enumFilter bool
var inputFilter bool

var validKinds = map[string]ast.DefinitionKind{
	"scalar":    ast.Scalar,
	"type":      ast.Object,
	"object":    ast.Object,
	"interface": ast.Interface,
	"union":     ast.Union,
	"enum":      ast.Enum,
	"input":     ast.InputObject,
}

func matchesKindFilter(t *ast.Definition) bool {
	// Check individual kind flags first (OR logic between them)
	hasIndividualFilter := scalarFilter || typeFilter || interfaceFilter || unionFilter || enumFilter || inputFilter
	if hasIndividualFilter {
		switch t.Kind {
		case ast.Scalar:
			if scalarFilter {
				return true
			}
		case ast.Object:
			if typeFilter {
				return true
			}
		case ast.Interface:
			if interfaceFilter {
				return true
			}
		case ast.Union:
			if unionFilter {
				return true
			}
		case ast.Enum:
			if enumFilter {
				return true
			}
		case ast.InputObject:
			if inputFilter {
				return true
			}
		}
	}

	// Then check --kind flag
	if len(kindFilter) > 0 {
		for _, k := range kindFilter {
			if expectedKind, ok := validKinds[strings.ToLower(k)]; ok {
				if t.Kind == expectedKind {
					return true
				}
			}
		}
	}

	// If no filters specified, match everything
	if !hasIndividualFilter && len(kindFilter) == 0 {
		return true
	}

	return false
}

func getTypesUsedBy(schema *ast.Schema, typeName string) map[string]bool {
	usedTypes := make(map[string]bool)

	typeDef := schema.Types[typeName]
	if typeDef == nil {
		return usedTypes
	}

	// Collect types from fields
	for _, field := range typeDef.Fields {
		usedTypes[getBaseTypeName(field.Type)] = true

		// Collect types from field arguments
		for _, arg := range field.Arguments {
			usedTypes[getBaseTypeName(arg.Type)] = true
		}
	}

	// Collect types from input fields (for input types)
	for _, field := range typeDef.Fields {
		usedTypes[getBaseTypeName(field.Type)] = true
	}

	return usedTypes
}

func validateImplementsFilter(schema *ast.Schema) error {
	if implementsFilter == "" {
		return nil
	}

	iface := schema.Types[implementsFilter]
	if iface == nil {
		var interfaces []string
		for name, def := range schema.Types {
			if def.Kind == ast.Interface {
				interfaces = append(interfaces, name)
			}
		}
		if suggestion := findClosest(implementsFilter, interfaces); suggestion != "" {
			return fmt.Errorf("interface '%s' does not exist in schema, did you mean '%s'?", implementsFilter, suggestion)
		}
		return fmt.Errorf("interface '%s' does not exist in schema", implementsFilter)
	}
	if iface.Kind != ast.Interface {
		return fmt.Errorf("'%s' is not an interface (it's a %s)", implementsFilter, kindToString(string(iface.Kind)))
	}
	return nil
}

func matchesImplementsFilter(t *ast.Definition) bool {
	if implementsFilter == "" {
		return true
	}
	return slices.Contains(t.Interfaces, implementsFilter)
}

func matchesHasFieldFilter(t *ast.Definition) bool {
	if len(hasFieldFilter) == 0 {
		return true
	}
	for _, fieldName := range hasFieldFilter {
		found := false
		for _, field := range t.Fields {
			if field.Name == fieldName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// collectUsedBySets validates each type name in the filter list and returns
// a slice of sets where each set contains the types used by the corresponding filter type.
func collectUsedBySets(schema *ast.Schema, filterTypes []string) ([]map[string]bool, error) {
	var sets []map[string]bool
	for _, typeName := range filterTypes {
		if err := validateTypeExists(schema, typeName, "type"); err != nil {
			return nil, err
		}
		sets = append(sets, getTypesUsedBy(schema, typeName))
	}
	return sets, nil
}

// isInAllSets returns true if the name is present in ALL of the given sets.
// Returns true if sets is empty.
func isInAllSets(name string, sets []map[string]bool) bool {
	for _, set := range sets {
		if !set[name] {
			return false
		}
	}
	return true
}

// isInAnySets returns true if the name is present in ANY of the given sets.
// Returns false if sets is empty.
func isInAnySets(name string, sets []map[string]bool) bool {
	for _, set := range sets {
		if set[name] {
			return true
		}
	}
	return false
}

// typesCmd represents the types command
var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "Lists all types in the schema",
	Long: `Lists all types in the schema with optional filtering.

Shows the type's kind (enum, type, input, etc.) and the type name.

Output formats:
  text    "type User", "enum Status", etc. (default when piping)
  json    [{"name": "User", "kind": "OBJECT", "description": "..."}, ...]
  pretty  Formatted table with columns (default in terminal)

Multiple filters can be combined and are applied with AND logic.`,
	Example: `  # Find all types that could be returned by the API
  gqlx types --type --interface

  # Find input types used by Query
  gqlx types --input --used-by Query

  # Find all enums
  gqlx types --enum

  # Find types used by both Query AND Mutation
  gqlx types --used-by Query --used-by Mutation

  # Find types used by Query OR Mutation
  gqlx types --used-by-any Query --used-by-any Mutation

  # Find types not used by Query (potentially orphaned)
  gqlx types --not-used-by Query

  # Find all node types for Relay-style pagination
  gqlx types --implements Node

  # Find types ending in "Connection" (Relay pagination)
  gqlx types --name "*Connection"

  # Find types matching a regex pattern
  gqlx types --name-regex "^(User|Post)"

  # Pipe to other tools
  gqlx types --kind type -f json | jq '.[].name'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var typesNameRegex *regexp.Regexp
		if typesNameRegexFilter != "" {
			var err error
			typesNameRegex, err = regexp.Compile(typesNameRegexFilter)
			if err != nil {
				return fmt.Errorf("invalid regex pattern for --name-regex: %w", err)
			}
		}

		schema, err := loadCliForSchema()
		if err != nil {
			return err
		}

		if err := validateImplementsFilter(schema); err != nil {
			return err
		}

		// Collect type sets for all used-by filters
		usedBySets, err := collectUsedBySets(schema, usedByFilter)
		if err != nil {
			return err
		}
		usedByAnySets, err := collectUsedBySets(schema, usedByAnyFilter)
		if err != nil {
			return err
		}
		notUsedBySets, err := collectUsedBySets(schema, notUsedByFilter)
		if err != nil {
			return err
		}
		notUsedByAllSets, err := collectUsedBySets(schema, notUsedByAllFilter)
		if err != nil {
			return err
		}

		var types []TypeInfo
		for _, graphqlType := range schema.Types {
			if !matchesImplementsFilter(graphqlType) {
				continue
			}
			if !matchesHasFieldFilter(graphqlType) {
				continue
			}
			if !matchesKindFilter(graphqlType) {
				continue
			}

			// --used-by (AND): must be used by ALL specified types
			if len(usedBySets) > 0 && !isInAllSets(graphqlType.Name, usedBySets) {
				continue
			}

			// --used-by-any (OR): must be used by ANY of the specified types
			if len(usedByAnySets) > 0 && !isInAnySets(graphqlType.Name, usedByAnySets) {
				continue
			}

			// --not-used-by (AND): must NOT be used by ANY of the specified types
			if len(notUsedBySets) > 0 && isInAnySets(graphqlType.Name, notUsedBySets) {
				continue
			}

			// --not-used-by-all (OR): exclude only if used by ALL specified types
			if len(notUsedByAllSets) > 0 && isInAllSets(graphqlType.Name, notUsedByAllSets) {
				continue
			}

			if typesHasDescriptionFilter && graphqlType.Description == "" {
				continue
			}
			if typesNameFilter != "" {
				matched, _ := filepath.Match(typesNameFilter, graphqlType.Name)
				if !matched {
					continue
				}
			}
			if typesNameRegex != nil && !typesNameRegex.MatchString(graphqlType.Name) {
				continue
			}

			types = append(types, TypeInfo{
				Name:        graphqlType.Name,
				Kind:        string(graphqlType.Kind),
				Description: graphqlType.Description,
			})
		}

		if len(types) == 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "No types found that match the filters.")
		}

		renderer := render.Renderer[TypeInfo]{
			Data:         types,
			TextFormat:   formatTypeText,
			PrettyFormat: formatTypesPretty,
		}

		output, err := renderer.Render(outputFormat)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(typesCmd)

	typesCmd.Flags().StringVar(&implementsFilter, "implements", "", "Filter to types that implement the given interface")
	typesCmd.Flags().StringArrayVar(&hasFieldFilter, "has-field", nil, "Filter to types that have the given field (can be specified multiple times)")
	typesCmd.Flags().StringArrayVar(&kindFilter, "kind", nil, "Filter to types of the given kind: scalar, type, interface, union, enum, input (if specified multiple times, applied using OR logic)")
	typesCmd.Flags().StringArrayVar(&usedByFilter, "used-by", nil, "Filter to types used by the given type (AND logic when specified multiple times)")
	typesCmd.Flags().StringArrayVar(&usedByAnyFilter, "used-by-any", nil, "Filter to types used by any of the given types (OR logic)")
	typesCmd.Flags().StringArrayVar(&notUsedByFilter, "not-used-by", nil, "Exclude types used by any of the given types")
	typesCmd.Flags().StringArrayVar(&notUsedByAllFilter, "not-used-by-all", nil, "Exclude types only if used by all of the given types")
	typesCmd.Flags().StringVar(&typesNameFilter, "name", "", "Filter types by name using a glob pattern (e.g., *Connection, User*)")
	typesCmd.Flags().StringVar(&typesNameRegexFilter, "name-regex", "", "Filter types by name using a regex pattern")
	typesCmd.Flags().BoolVar(&typesHasDescriptionFilter, "has-description", false, "Filter to only show types that have a description")
	typesCmd.Flags().BoolVar(&scalarFilter, "scalar", false, "Filter to scalar types")
	typesCmd.Flags().BoolVar(&typeFilter, "type", false, "Filter to object types")
	typesCmd.Flags().BoolVar(&interfaceFilter, "interface", false, "Filter to interface types")
	typesCmd.Flags().BoolVar(&unionFilter, "union", false, "Filter to union types")
	typesCmd.Flags().BoolVar(&enumFilter, "enum", false, "Filter to enum types")
	typesCmd.Flags().BoolVar(&inputFilter, "input", false, "Filter to input types")
}
