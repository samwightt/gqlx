/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
)

type referencesOptions struct {
	kind   string
	inType string
}

func formatReferenceText(ref ReferenceInfo) string {
	desc := ""
	if ref.Description != "" {
		desc = " # " + strings.ReplaceAll(ref.Description, "\n", " ")
	}
	return fmt.Sprintf("%s: %s%s", ref.Location, ref.Type, desc)
}

func formatReferencesPretty(refs []ReferenceInfo) string {
	t := makeTable()

	for _, ref := range refs {
		desc := strings.ReplaceAll(ref.Description, "\n", " ")
		t.Row(ref.Location, ref.Kind, ref.Type, desc)
	}
	t.Headers("location", "kind", "type", "description")

	return t.String()
}

func NewReferencesCmd() *cobra.Command {
	opts := &referencesOptions{}

	cmd := &cobra.Command{
		Use:   "references <type>",
		Short: "Shows where a type is used in the schema",
		Long: `Shows where a given type is used in the schema - specifically which fields
return it and which arguments use it.

This is useful for understanding the impact of changes to a type, finding
all entry points to a type, or exploring the schema structure.

Output formats:
  text    "Query.user: User", "Query.search.userId: ID!", etc. (default when piping)
  json    [{"location": "Query.user", "kind": "field", "type": "User"}, ...]
  pretty  Formatted table with columns (default in terminal)`,
		Example: `  # Find all references to the User type
  gqlx references User

  # Find only fields that return User
  gqlx references User --kind field

  # Find only arguments of type User
  gqlx references User --kind argument

  # Find references to User only within the Query type
  gqlx references User --in Query

  # JSON output for scripting
  gqlx references User -f json`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			schema, err := loadSchema()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			outputNames := []string{}
			for key := range schema.Types {
				if strings.Contains(strings.ToLower(key), strings.ToLower(toComplete)) {
					outputNames = append(outputNames, key)
				}
			}

			sort.Strings(outputNames)

			return outputNames, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReferences(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.kind, "kind", "", "Filter by reference kind: 'field' or 'argument'")
	cmd.Flags().StringVar(&opts.inType, "in", "", "Only show references from the specified type")

	return cmd
}

func runReferences(cmd *cobra.Command, args []string, opts *referencesOptions) error {
	targetType := args[0]

	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	// Validate target type exists
	if err := validateTypeExists(schema, targetType, "type"); err != nil {
		return err
	}

	// Validate --in filter type exists
	if opts.inType != "" {
		if err := validateTypeExists(schema, opts.inType, "type"); err != nil {
			return err
		}
	}

	// Validate --kind filter
	if opts.kind != "" && opts.kind != "field" && opts.kind != "argument" {
		return fmt.Errorf("--kind must be 'field' or 'argument', got '%s'", opts.kind)
	}

	var refs []ReferenceInfo

	for _, typeDef := range schema.Types {
		// Skip if --in filter is set and doesn't match
		if opts.inType != "" && typeDef.Name != opts.inType {
			continue
		}

		for _, field := range typeDef.Fields {
			// Check field return type
			if getBaseTypeName(field.Type) == targetType {
				if opts.kind == "" || opts.kind == "field" {
					refs = append(refs, ReferenceInfo{
						Location:    typeDef.Name + "." + field.Name,
						Kind:        "field",
						Type:        typeToString(field.Type),
						Description: field.Description,
					})
				}
			}

			// Check argument types
			for _, arg := range field.Arguments {
				if getBaseTypeName(arg.Type) == targetType {
					if opts.kind == "" || opts.kind == "argument" {
						refs = append(refs, ReferenceInfo{
							Location:    typeDef.Name + "." + field.Name + "." + arg.Name,
							Kind:        "argument",
							Type:        typeToString(arg.Type),
							Description: arg.Description,
						})
					}
				}
			}
		}
	}

	if len(refs) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No references found.")
	}

	renderer := render.Renderer[ReferenceInfo]{
		Data:         refs,
		TextFormat:   formatReferenceText,
		PrettyFormat: formatReferencesPretty,
	}

	output, err := renderer.Render(outputFormat)
	if err != nil {
		return fmt.Errorf("error rendering output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}
