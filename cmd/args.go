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

type argsOptions struct {
	deprecated     bool
	typeFilter     string
	required       bool
	nullable       bool
	name           string
	nameRegex      string
	hasDescription bool
}

func isArgDeprecated(arg *ast.ArgumentDefinition) bool {
	return arg.Directives.ForName("deprecated") != nil
}

func matchesArgFilters(arg *ast.ArgumentDefinition, opts *argsOptions) bool {
	if opts.typeFilter != "" && getBaseTypeName(arg.Type) != opts.typeFilter {
		return false
	}
	if opts.required && !arg.Type.NonNull {
		return false
	}
	if opts.nullable && arg.Type.NonNull {
		return false
	}
	if opts.hasDescription && arg.Description == "" {
		return false
	}
	return true
}

func formatArgName(arg ArgInfo) string {
	if arg.TypeName != "" && arg.FieldName != "" {
		return fmt.Sprintf("%s.%s.%s", arg.TypeName, arg.FieldName, arg.Name)
	}
	return arg.Name
}

func formatArgText(arg ArgInfo) string {
	name := formatArgName(arg)

	typeStr := arg.Type
	if arg.DefaultValue != "" {
		typeStr += " = " + arg.DefaultValue
	}

	desc := ""
	if arg.Description != "" {
		desc = " # " + strings.ReplaceAll(arg.Description, "\n", " ")
	}
	return fmt.Sprintf("%s: %s%s", name, typeStr, desc)
}

func formatArgsPretty(args []ArgInfo) string {
	t := makeTable()

	for _, arg := range args {
		name := formatArgName(arg)
		typeStr := arg.Type
		if arg.DefaultValue != "" {
			typeStr += " = " + arg.DefaultValue
		}
		desc := strings.ReplaceAll(arg.Description, "\n", " ")
		t.Row(name, typeStr, desc)
	}
	t.Headers("argument", "type", "description")

	return t.String()
}

func argToInfo(arg *ast.ArgumentDefinition) ArgInfo {
	var defaultValue string
	if arg.DefaultValue != nil {
		defaultValue = arg.DefaultValue.String()
	}

	return ArgInfo{
		Name:         arg.Name,
		Type:         typeToString(arg.Type),
		DefaultValue: defaultValue,
		Description:  arg.Description,
	}
}

func NewArgsCmd() *cobra.Command {
	opts := &argsOptions{}

	cmd := &cobra.Command{
		Use:   "args [field]",
		Short: "Lists arguments on fields.",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			schema, err := loadSchema()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			outputNames := []string{}
			for _, typeDef := range schema.Types {
				for _, field := range typeDef.Fields {
					if len(field.Arguments) == 0 { continue }
					fieldName := fmt.Sprintf("%s.%s", typeDef.Name, field.Name)
					if strings.Contains(strings.ToLower(fieldName), strings.ToLower(toComplete)) {
						outputNames = append(outputNames, fieldName)
					}
				}
			}

			sort.Strings(outputNames)

			return outputNames, cobra.ShellCompDirectiveNoFileComp
		},
		Args: cobra.MaximumNArgs(1),
		Long: `Lists arguments on fields in the schema.

If a field is specified (as Type.field), only arguments for that field are shown.
If no field is specified, all arguments for all fields are shown.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArgs(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.deprecated, "deprecated", false, "Filter to only show deprecated arguments")
	cmd.Flags().StringVar(&opts.typeFilter, "type", "", "Filter to arguments of the given type")
	cmd.Flags().BoolVar(&opts.required, "required", false, "Filter to only show required (non-null) arguments")
	cmd.Flags().BoolVar(&opts.nullable, "nullable", false, "Filter to only show nullable arguments")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter arguments by name using a glob pattern (e.g., *Id, first*)")
	cmd.Flags().StringVar(&opts.nameRegex, "name-regex", "", "Filter arguments by name using a regex pattern")
	cmd.Flags().BoolVar(&opts.hasDescription, "has-description", false, "Filter to only show arguments that have a description")

	return cmd
}

func runArgs(cmd *cobra.Command, args []string, opts *argsOptions) error {
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

	var argInfos []ArgInfo

	if len(args) == 0 {
		// List all arguments from all fields
		for _, graphqlType := range schema.Types {
			for _, field := range graphqlType.Fields {
				for _, arg := range field.Arguments {
					if opts.deprecated && !isArgDeprecated(arg) {
						continue
					}
					if !matchesArgFilters(arg, opts) {
						continue
					}
					if opts.name != "" {
						matched, _ := filepath.Match(opts.name, arg.Name)
						if !matched {
							continue
						}
					}
					if nameRegex != nil && !nameRegex.MatchString(arg.Name) {
						continue
					}
					info := argToInfo(arg)
					info.TypeName = graphqlType.Name
					info.FieldName = field.Name
					argInfos = append(argInfos, info)
				}
			}
		}
	} else {
		// Parse Type.field format
		fieldPath := args[0]
		parts := strings.Split(fieldPath, ".")
		if len(parts) != 2 {
			return fmt.Errorf("field must be specified as Type.field (e.g., Query.user)")
		}
		typeName, fieldName := parts[0], parts[1]

		if err := validateTypeExists(schema, typeName, "type"); err != nil {
			return err
		}
		graphqlType := schema.Types[typeName]

		var field *ast.FieldDefinition
		for _, f := range graphqlType.Fields {
			if f.Name == fieldName {
				field = f
				break
			}
		}

		if field == nil {
			if suggestion := findClosest(fieldName, pluck(graphqlType.Fields, func(f *ast.FieldDefinition) string { return f.Name })); suggestion != "" {
				return fmt.Errorf("field '%s' does not exist on type '%s', did you mean '%s'?", fieldName, typeName, suggestion)
			}
			return fmt.Errorf("field '%s' does not exist on type '%s'", fieldName, typeName)
		}

		for _, arg := range field.Arguments {
			if opts.deprecated && !isArgDeprecated(arg) {
				continue
			}
			if !matchesArgFilters(arg, opts) {
				continue
			}
			if opts.name != "" {
				matched, _ := filepath.Match(opts.name, arg.Name)
				if !matched {
					continue
				}
			}
			if nameRegex != nil && !nameRegex.MatchString(arg.Name) {
				continue
			}
			argInfos = append(argInfos, argToInfo(arg))
		}
	}

	if len(argInfos) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No arguments found that match the filters.")
	}

	renderer := render.Renderer[ArgInfo]{
		Data:         argInfos,
		TextFormat:   formatArgText,
		PrettyFormat: formatArgsPretty,
	}

	output, err := renderer.Render(outputFormat)
	if err != nil {
		return fmt.Errorf("error rendering output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}
