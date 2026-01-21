/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

type fieldsOptions struct {
	deprecated     bool
	hasArg         []string
	returns        string
	required       bool
	nullable       bool
	name           string
	nameRegex      string
	hasDescription bool
}

func fieldToInfo(fieldDef *ast.FieldDefinition) FieldInfo {
	var args []ArgumentInfo
	for _, arg := range fieldDef.Arguments {
		args = append(args, ArgumentInfo{
			Name: arg.Name,
			Type: typeToString(arg.Type),
		})
	}

	var defaultValue string
	if fieldDef.DefaultValue != nil {
		defaultValue = fieldDef.DefaultValue.String()
	}

	return FieldInfo{
		Name:         fieldDef.Name,
		Arguments:    args,
		Type:         typeToString(fieldDef.Type),
		DefaultValue: defaultValue,
		Description:  fieldDef.Description,
	}
}

func formatFieldName(field FieldInfo, format render.Format) string {
	name := field.Name
	if field.TypeName != "" {
		name = field.TypeName + "." + field.Name
	}

	if len(field.Arguments) == 0 {
		return name
	}

	var args []string
	for _, arg := range field.Arguments {
		args = append(args, fmt.Sprintf("%s: %s", arg.Name, arg.Type))
	}

	if format == render.FormatPretty {
		return fmt.Sprintf("%s(\n\t\t%s\n\t)", name, strings.Join(args, ",\n\t\t"))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}

func formatFieldText(field FieldInfo) string {
	name := formatFieldName(field, render.FormatText)

	typeStr := field.Type
	if field.DefaultValue != "" {
		typeStr += " = " + field.DefaultValue
	}

	desc := ""
	if field.Description != "" {
		desc = " # " + strings.ReplaceAll(field.Description, "\n", " ")
	}
	return fmt.Sprintf("%s: %s%s", name, typeStr, desc)
}

func formatFieldsPretty(fields []FieldInfo) string {
	t := makeTable()

	for _, field := range fields {
		name := formatFieldName(field, render.FormatPretty)
		typeStr := field.Type
		if field.DefaultValue != "" {
			typeStr += " = " + field.DefaultValue
		}
		desc := strings.ReplaceAll(field.Description, "\n", " ")
		t.Row(name, typeStr, desc)
	}
	t.Headers("field", "type", "description")

	return t.String()
}

func isFieldDeprecated(field *ast.FieldDefinition) bool {
	return field.Directives.ForName("deprecated") != nil
}

func matchesHasArgFilter(field *ast.FieldDefinition, hasArgFilter []string) bool {
	if len(hasArgFilter) == 0 {
		return true
	}
	for _, argName := range hasArgFilter {
		found := false
		for _, arg := range field.Arguments {
			if arg.Name == argName {
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

func NewFieldsCmd() *cobra.Command {
	opts := &fieldsOptions{}

	cmd := &cobra.Command{
		Use:   "fields [type]",
		Short: "Lists fields on a type or across all types",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		Args: cobra.MaximumNArgs(1),
		Long: `Lists fields on a type or across all types with optional filtering.

If a type is specified, shows fields for that type only.
If no type is specified, shows all fields prefixed with their type (User.id, Post.title, etc).

Output formats:
  text    "name: String! # Description", "id: ID!", etc. (default when piping)
  json    [{"name": "id", "type": "ID!", "description": "..."}, ...]
  pretty  Formatted table with columns (default in terminal)

Multiple filters can be combined and are applied with AND logic.`,
		Example: `  # See all fields on a type
  gqlx fields User

  # Find deprecated fields
  gqlx fields --deprecated

  # Find fields with pagination arguments that return a specific type
  gqlx fields --has-arg first --has-arg after --returns User

  # Find fields ending in "Id"
  gqlx fields --name "*Id"

  # Find fields starting with "get"
  gqlx fields --name "get*"

  # Find fields matching a regex pattern
  gqlx fields --name-regex "^(get|fetch)"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFields(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.deprecated, "deprecated", false, "Filter to only show deprecated fields")
	cmd.Flags().StringArrayVar(&opts.hasArg, "has-arg", nil, "Filter to fields that have the given argument (can be specified multiple times)")
	cmd.Flags().StringVar(&opts.returns, "returns", "", "Filter to fields that return the given type")
	cmd.Flags().BoolVar(&opts.required, "required", false, "Filter to only show required (non-null) fields")
	cmd.Flags().BoolVar(&opts.nullable, "nullable", false, "Filter to only show nullable fields")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter fields by name using a glob pattern (e.g., *Id, get*)")
	cmd.Flags().StringVar(&opts.nameRegex, "name-regex", "", "Filter fields by name using a regex pattern")
	cmd.Flags().BoolVar(&opts.hasDescription, "has-description", false, "Filter to only show fields that have a description")

	return cmd
}

func runFields(cmd *cobra.Command, args []string, opts *fieldsOptions) error {
	if opts.required && opts.nullable {
		return fmt.Errorf("--required and --nullable cannot be used together")
	}

	var nameRegex *regexp.Regexp
	if opts.nameRegex != "" {
		var err error
		nameRegex, err = regexp.Compile(opts.nameRegex)
		if err != nil {
			return fmt.Errorf("invalid regex pattern for --name-regex: %w", err)
		}
	}

	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	var fields []FieldInfo

	if len(args) == 0 {
		// List all fields from all types
		for _, graphqlType := range schema.Types {
			for _, field := range graphqlType.Fields {
				if opts.deprecated && !isFieldDeprecated(field) {
					continue
				}
				if !matchesHasArgFilter(field, opts.hasArg) {
					continue
				}
				if opts.returns != "" && getBaseTypeName(field.Type) != opts.returns {
					continue
				}
				if opts.required && !field.Type.NonNull {
					continue
				}
				if opts.nullable && field.Type.NonNull {
					continue
				}
				if opts.hasDescription && field.Description == "" {
					continue
				}
				if opts.name != "" {
					matched, _ := filepath.Match(opts.name, field.Name)
					if !matched {
						continue
					}
				}
				if nameRegex != nil && !nameRegex.MatchString(field.Name) {
					continue
				}
				info := fieldToInfo(field)
				info.TypeName = graphqlType.Name
				fields = append(fields, info)
			}
		}
	} else {
		// List fields from specific type
		searchString := args[0]
		if err := validateTypeExists(schema, searchString, "type"); err != nil {
			return err
		}
		graphqlType := schema.Types[searchString]

		for _, field := range graphqlType.Fields {
			if opts.deprecated && !isFieldDeprecated(field) {
				continue
			}
			if !matchesHasArgFilter(field, opts.hasArg) {
				continue
			}
			if opts.returns != "" && getBaseTypeName(field.Type) != opts.returns {
				continue
			}
			if opts.required && !field.Type.NonNull {
				continue
			}
			if opts.nullable && field.Type.NonNull {
				continue
			}
			if opts.hasDescription && field.Description == "" {
				continue
			}
			if opts.name != "" {
				matched, _ := filepath.Match(opts.name, field.Name)
				if !matched {
					continue
				}
			}
			if nameRegex != nil && !nameRegex.MatchString(field.Name) {
				continue
			}
			fields = append(fields, fieldToInfo(field))
		}
	}

	if len(fields) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No fields found that match the filters.")
	}

	renderer := render.Renderer[FieldInfo]{
		Data:         fields,
		TextFormat:   formatFieldText,
		PrettyFormat: formatFieldsPretty,
	}

	output, err := renderer.Render(outputFormat)
	if err != nil {
		return fmt.Errorf("error rendering output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}
