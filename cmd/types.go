/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
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
	if len(kindFilter) == 0 {
		return true
	}
	for _, k := range kindFilter {
		if expectedKind, ok := validKinds[strings.ToLower(k)]; ok {
			if t.Kind == expectedKind {
				return true
			}
		}
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
  gqlx types --kind type --kind interface

  # Find input types used by Query
  gqlx types --kind input --used-by Query

  # Find types used by both Query AND Mutation
  gqlx types --used-by Query --used-by Mutation

  # Find types used by Query OR Mutation
  gqlx types --used-by-any Query --used-by-any Mutation

  # Find types not used by Query (potentially orphaned)
  gqlx types --not-used-by Query

  # Find all node types for Relay-style pagination
  gqlx types --implements Node

  # Pipe to other tools
  gqlx types --kind type -f json | jq '.[].name'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schema, err := loadCliForSchema()
		if err != nil {
			return err
		}

		if err := validateImplementsFilter(schema); err != nil {
			return err
		}

		// Helper to validate type exists and get types used by it
		validateAndGetUsedBy := func(typeName string) (map[string]bool, error) {
			if schema.Types[typeName] == nil {
				var typeNames []string
				for name := range schema.Types {
					typeNames = append(typeNames, name)
				}
				if suggestion := findClosest(typeName, typeNames); suggestion != "" {
					return nil, fmt.Errorf("type '%s' does not exist in schema, did you mean '%s'?", typeName, suggestion)
				}
				return nil, fmt.Errorf("type '%s' does not exist in schema", typeName)
			}
			return getTypesUsedBy(schema, typeName), nil
		}

		// Collect type sets for all filters
		var usedBySets []map[string]bool
		for _, typeName := range usedByFilter {
			usedBy, err := validateAndGetUsedBy(typeName)
			if err != nil {
				return err
			}
			usedBySets = append(usedBySets, usedBy)
		}

		var usedByAnySets []map[string]bool
		for _, typeName := range usedByAnyFilter {
			usedBy, err := validateAndGetUsedBy(typeName)
			if err != nil {
				return err
			}
			usedByAnySets = append(usedByAnySets, usedBy)
		}

		var notUsedBySets []map[string]bool
		for _, typeName := range notUsedByFilter {
			usedBy, err := validateAndGetUsedBy(typeName)
			if err != nil {
				return err
			}
			notUsedBySets = append(notUsedBySets, usedBy)
		}

		var notUsedByAllSets []map[string]bool
		for _, typeName := range notUsedByAllFilter {
			usedBy, err := validateAndGetUsedBy(typeName)
			if err != nil {
				return err
			}
			notUsedByAllSets = append(notUsedByAllSets, usedBy)
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
			if len(usedBySets) > 0 {
				usedByAll := true
				for _, usedBySet := range usedBySets {
					if !usedBySet[graphqlType.Name] {
						usedByAll = false
						break
					}
				}
				if !usedByAll {
					continue
				}
			}

			// --used-by-any (OR): must be used by ANY of the specified types
			if len(usedByAnySets) > 0 {
				usedByAny := false
				for _, usedBySet := range usedByAnySets {
					if usedBySet[graphqlType.Name] {
						usedByAny = true
						break
					}
				}
				if !usedByAny {
					continue
				}
			}

			// --not-used-by (AND): must NOT be used by ANY of the specified types
			if len(notUsedBySets) > 0 {
				usedByAny := false
				for _, usedBySet := range notUsedBySets {
					if usedBySet[graphqlType.Name] {
						usedByAny = true
						break
					}
				}
				if usedByAny {
					continue
				}
			}

			// --not-used-by-all (OR): exclude only if used by ALL specified types
			if len(notUsedByAllSets) > 0 {
				usedByAll := true
				for _, usedBySet := range notUsedByAllSets {
					if !usedBySet[graphqlType.Name] {
						usedByAll = false
						break
					}
				}
				if usedByAll {
					continue
				}
			}

			types = append(types, TypeInfo{
				Name:        graphqlType.Name,
				Kind:        string(graphqlType.Kind),
				Description: graphqlType.Description,
			})
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
}
