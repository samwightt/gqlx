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
	"github.com/vektah/gqlparser/v2/ast"
)

type valuesOptions struct {
	deprecated     bool
	hasDescription bool
}

func isValueDeprecated(value *ast.EnumValueDefinition) bool {
	return value.Directives.ForName("deprecated") != nil
}

func formatValueName(v ValueInfo) string {
	if v.EnumName != "" {
		return v.EnumName + "." + v.Name
	}
	return v.Name
}

func formatValueText(v ValueInfo) string {
	name := formatValueName(v)
	if v.Description != "" {
		desc := strings.ReplaceAll(v.Description, "\n", " ")
		return fmt.Sprintf("%s # %s", name, desc)
	}
	return name
}

func formatValuesPretty(values []ValueInfo) string {
	t := makeTable()

	for _, v := range values {
		name := formatValueName(v)
		desc := strings.ReplaceAll(v.Description, "\n", " ")
		t.Row(name, desc)
	}
	t.Headers("value", "description")

	return t.String()
}

func NewValuesCmd() *cobra.Command {
	opts := &valuesOptions{}

	cmd := &cobra.Command{
		Use:   "values [enum]",
		Short: "Lists values of an enum type.",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			schema, err := loadSchema()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			outputNames := []string{}
			for name, def := range schema.Types {
				if def.Kind == ast.Enum {
					if strings.Contains(strings.ToLower(name), strings.ToLower(toComplete)) {
						outputNames = append(outputNames, name)
					}
				}
			}

			sort.Strings(outputNames)

			return outputNames, cobra.ShellCompDirectiveNoFileComp
		},
		Args: cobra.MaximumNArgs(1),
		Long: `Lists values of an enum type in the schema.

If an enum is specified, only values for that enum are shown.
If no enum is specified, all enum values for all enums are shown.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValues(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.deprecated, "deprecated", false, "Filter to only show deprecated values")
	cmd.Flags().BoolVar(&opts.hasDescription, "has-description", false, "Filter to only show values that have a description")

	return cmd
}

func runValues(cmd *cobra.Command, args []string, opts *valuesOptions) error {
	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	var values []ValueInfo

	if len(args) == 0 {
		// List all values from all enums
		for _, graphqlType := range schema.Types {
			if graphqlType.Kind != ast.Enum {
				continue
			}
			for _, value := range graphqlType.EnumValues {
				if opts.deprecated && !isValueDeprecated(value) {
					continue
				}
				if opts.hasDescription && value.Description == "" {
					continue
				}
				values = append(values, ValueInfo{
					EnumName:    graphqlType.Name,
					Name:        value.Name,
					Description: value.Description,
				})
			}
		}
	} else {
		// List values from specific enum
		enumName := args[0]
		graphqlType := schema.Types[enumName]
		if graphqlType == nil {
			var enumNames []string
			for name, def := range schema.Types {
				if def.Kind == ast.Enum {
					enumNames = append(enumNames, name)
				}
			}
			if suggestion := findClosest(enumName, enumNames); suggestion != "" {
				return fmt.Errorf("enum '%s' does not exist in schema, did you mean '%s'?", enumName, suggestion)
			}
			return fmt.Errorf("enum '%s' does not exist in schema", enumName)
		}

		if graphqlType.Kind != ast.Enum {
			return fmt.Errorf("'%s' is not an enum (it's a %s)", enumName, kindToString(string(graphqlType.Kind)))
		}

		for _, value := range graphqlType.EnumValues {
			if opts.deprecated && !isValueDeprecated(value) {
				continue
			}
			if opts.hasDescription && value.Description == "" {
				continue
			}
			values = append(values, ValueInfo{
				Name:        value.Name,
				Description: value.Description,
			})
		}
	}

	if len(values) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No values found that match the filters.")
	}

	renderer := render.Renderer[ValueInfo]{
		Data:         values,
		TextFormat:   formatValueText,
		PrettyFormat: formatValuesPretty,
	}

	output, err := renderer.Render(outputFormat)
	if err != nil {
		return fmt.Errorf("error rendering output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}
