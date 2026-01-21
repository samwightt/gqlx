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

type typesOptions struct {
	implements     string
	hasField       []string
	kind           []string
	usedBy         []string
	usedByAny      []string
	notUsedBy      []string
	notUsedByAll   []string
	name           string
	nameRegex      string
	hasDescription bool
	scalar         bool
	typeFilter     bool
	interfaceFlag  bool
	union          bool
	enum           bool
	input          bool
}

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

var validKinds = map[string]ast.DefinitionKind{
	"scalar":    ast.Scalar,
	"type":      ast.Object,
	"object":    ast.Object,
	"interface": ast.Interface,
	"union":     ast.Union,
	"enum":      ast.Enum,
	"input":     ast.InputObject,
}

func matchesKindFilter(t *ast.Definition, opts *typesOptions) bool {
	// Check individual kind flags first (OR logic between them)
	hasIndividualFilter := opts.scalar || opts.typeFilter || opts.interfaceFlag || opts.union || opts.enum || opts.input
	if hasIndividualFilter {
		switch t.Kind {
		case ast.Scalar:
			if opts.scalar {
				return true
			}
		case ast.Object:
			if opts.typeFilter {
				return true
			}
		case ast.Interface:
			if opts.interfaceFlag {
				return true
			}
		case ast.Union:
			if opts.union {
				return true
			}
		case ast.Enum:
			if opts.enum {
				return true
			}
		case ast.InputObject:
			if opts.input {
				return true
			}
		}
	}

	// Then check --kind flag
	if len(opts.kind) > 0 {
		for _, k := range opts.kind {
			if expectedKind, ok := validKinds[strings.ToLower(k)]; ok {
				if t.Kind == expectedKind {
					return true
				}
			}
		}
	}

	// If no filters specified, match everything
	if !hasIndividualFilter && len(opts.kind) == 0 {
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

func validateImplementsFilter(schema *ast.Schema, implementsFilter string) error {
	if implementsFilter == "" {
		return nil
	}

	iface := schema.Types[implementsFilter]
	if iface == nil {
		if suggestion := findClosest(implementsFilter, filterKeys(schema.Types, func(_ string, def *ast.Definition) bool {
			return def.Kind == ast.Interface
		})); suggestion != "" {
			return fmt.Errorf("interface '%s' does not exist in schema, did you mean '%s'?", implementsFilter, suggestion)
		}
		return fmt.Errorf("interface '%s' does not exist in schema", implementsFilter)
	}
	if iface.Kind != ast.Interface {
		return fmt.Errorf("'%s' is not an interface (it's a %s)", implementsFilter, kindToString(string(iface.Kind)))
	}
	return nil
}

func matchesImplementsFilter(t *ast.Definition, implementsFilter string) bool {
	if implementsFilter == "" {
		return true
	}
	return slices.Contains(t.Interfaces, implementsFilter)
}

func matchesHasFieldFilter(t *ast.Definition, hasFieldFilter []string) bool {
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

func NewTypesCmd() *cobra.Command {
	opts := &typesOptions{}

	cmd := &cobra.Command{
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
			return runTypes(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.implements, "implements", "", "Filter to types that implement the given interface")
	cmd.Flags().StringArrayVar(&opts.hasField, "has-field", nil, "Filter to types that have the given field (can be specified multiple times)")
	cmd.Flags().StringArrayVar(&opts.kind, "kind", nil, "Filter to types of the given kind: scalar, type, interface, union, enum, input (if specified multiple times, applied using OR logic)")
	cmd.Flags().StringArrayVar(&opts.usedBy, "used-by", nil, "Filter to types used by the given type (AND logic when specified multiple times)")
	cmd.Flags().StringArrayVar(&opts.usedByAny, "used-by-any", nil, "Filter to types used by any of the given types (OR logic)")
	cmd.Flags().StringArrayVar(&opts.notUsedBy, "not-used-by", nil, "Exclude types used by any of the given types")
	cmd.Flags().StringArrayVar(&opts.notUsedByAll, "not-used-by-all", nil, "Exclude types only if used by all of the given types")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter types by name using a glob pattern (e.g., *Connection, User*)")
	cmd.Flags().StringVar(&opts.nameRegex, "name-regex", "", "Filter types by name using a regex pattern")
	cmd.Flags().BoolVar(&opts.hasDescription, "has-description", false, "Filter to only show types that have a description")
	cmd.Flags().BoolVar(&opts.scalar, "scalar", false, "Filter to scalar types")
	cmd.Flags().BoolVar(&opts.typeFilter, "type", false, "Filter to object types")
	cmd.Flags().BoolVar(&opts.interfaceFlag, "interface", false, "Filter to interface types")
	cmd.Flags().BoolVar(&opts.union, "union", false, "Filter to union types")
	cmd.Flags().BoolVar(&opts.enum, "enum", false, "Filter to enum types")
	cmd.Flags().BoolVar(&opts.input, "input", false, "Filter to input types")

	return cmd
}

func runTypes(cmd *cobra.Command, args []string, opts *typesOptions) error {
	var typesNameRegex *regexp.Regexp
	if opts.nameRegex != "" {
		var err error
		typesNameRegex, err = regexp.Compile(opts.nameRegex)
		if err != nil {
			return fmt.Errorf("invalid regex pattern for --name-regex: %w", err)
		}
	}

	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	if err := validateImplementsFilter(schema, opts.implements); err != nil {
		return err
	}

	// Collect type sets for all used-by filters
	usedBySets, err := collectUsedBySets(schema, opts.usedBy)
	if err != nil {
		return err
	}
	usedByAnySets, err := collectUsedBySets(schema, opts.usedByAny)
	if err != nil {
		return err
	}
	notUsedBySets, err := collectUsedBySets(schema, opts.notUsedBy)
	if err != nil {
		return err
	}
	notUsedByAllSets, err := collectUsedBySets(schema, opts.notUsedByAll)
	if err != nil {
		return err
	}

	var types []TypeInfo
	for _, graphqlType := range schema.Types {
		if !matchesImplementsFilter(graphqlType, opts.implements) {
			continue
		}
		if !matchesHasFieldFilter(graphqlType, opts.hasField) {
			continue
		}
		if !matchesKindFilter(graphqlType, opts) {
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

		if opts.hasDescription && graphqlType.Description == "" {
			continue
		}
		if opts.name != "" {
			matched, _ := filepath.Match(opts.name, graphqlType.Name)
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
}
