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
var usedByFilter string

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

  # Find input types used directly in the args of 'User'
  gqlx types --kind input --used-by User

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

		// Validate and get types used by the specified type
		var usedByTypes map[string]bool
		if usedByFilter != "" {
			if schema.Types[usedByFilter] == nil {
				var typeNames []string
				for name := range schema.Types {
					typeNames = append(typeNames, name)
				}
				if suggestion := findClosest(usedByFilter, typeNames); suggestion != "" {
					return fmt.Errorf("type '%s' does not exist in schema, did you mean '%s'?", usedByFilter, suggestion)
				}
				return fmt.Errorf("type '%s' does not exist in schema", usedByFilter)
			}
			usedByTypes = getTypesUsedBy(schema, usedByFilter)
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
			if usedByFilter != "" && !usedByTypes[graphqlType.Name] {
				continue
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
	typesCmd.Flags().StringVar(&usedByFilter, "used-by", "", "Filter to types that are used by the given type (in fields or arguments)")
}
