/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func loadSchema() (*ast.Schema, error) {
	path, err := filepath.Abs(schemaFilePath)
	if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	strVal := string(bytes)

	fileName := filepath.Base(path)
	source := ast.Source{
		Input: strVal,
		Name:  fileName,
	}
	schema, err := gqlparser.LoadSchema(&source)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func loadCliForSchema() (*ast.Schema, error) {
	schema, err := loadSchema()

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("schema file does not exist: %s", schemaFilePath)
		}
		var parsingError *gqlerror.Error

		if errors.As(err, &parsingError) {
			return nil, fmt.Errorf("GraphQL schema parsing error: %v", parsingError)
		}

		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	return schema, nil
}

func typeToString(typeDef *ast.Type) string {
	requiredStr := ""
	if typeDef.NonNull {
		requiredStr = "!"
	}
	if typeDef.Elem != nil {
		return fmt.Sprintf("[%s]%s", typeToString(typeDef.Elem), requiredStr)
	} else {
		return typeDef.NamedType + requiredStr
	}
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

var deprecatedFilter bool
var hasArgFilter []string
var returnsFilter string
var requiredFilter bool
var nullableFilter bool
var nameFilter string
var nameRegexFilter string
var hasDescriptionFilter bool

func isFieldDeprecated(field *ast.FieldDefinition) bool {
	return field.Directives.ForName("deprecated") != nil
}

func getBaseTypeName(t *ast.Type) string {
	if t.Elem != nil {
		return getBaseTypeName(t.Elem)
	}
	return t.NamedType
}

func matchesHasArgFilter(field *ast.FieldDefinition) bool {
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

// fieldsCmd represents the fields command
var fieldsCmd = &cobra.Command{
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
		if requiredFilter && nullableFilter {
			return fmt.Errorf("--required and --nullable cannot be used together")
		}

		var nameRegex *regexp.Regexp
		if nameRegexFilter != "" {
			var err error
			nameRegex, err = regexp.Compile(nameRegexFilter)
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
					if deprecatedFilter && !isFieldDeprecated(field) {
						continue
					}
					if !matchesHasArgFilter(field) {
						continue
					}
					if returnsFilter != "" && getBaseTypeName(field.Type) != returnsFilter {
						continue
					}
					if requiredFilter && !field.Type.NonNull {
						continue
					}
					if nullableFilter && field.Type.NonNull {
						continue
					}
					if hasDescriptionFilter && field.Description == "" {
						continue
					}
					if nameFilter != "" {
						matched, _ := filepath.Match(nameFilter, field.Name)
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
				if deprecatedFilter && !isFieldDeprecated(field) {
					continue
				}
				if !matchesHasArgFilter(field) {
					continue
				}
				if returnsFilter != "" && getBaseTypeName(field.Type) != returnsFilter {
					continue
				}
				if requiredFilter && !field.Type.NonNull {
					continue
				}
				if nullableFilter && field.Type.NonNull {
					continue
				}
				if hasDescriptionFilter && field.Description == "" {
					continue
				}
				if nameFilter != "" {
					matched, _ := filepath.Match(nameFilter, field.Name)
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
	},
}

func init() {
	rootCmd.AddCommand(fieldsCmd)

	fieldsCmd.Flags().BoolVar(&deprecatedFilter, "deprecated", false, "Filter to only show deprecated fields")
	fieldsCmd.Flags().StringArrayVar(&hasArgFilter, "has-arg", nil, "Filter to fields that have the given argument (can be specified multiple times)")
	fieldsCmd.Flags().StringVar(&returnsFilter, "returns", "", "Filter to fields that return the given type")
	fieldsCmd.Flags().BoolVar(&requiredFilter, "required", false, "Filter to only show required (non-null) fields")
	fieldsCmd.Flags().BoolVar(&nullableFilter, "nullable", false, "Filter to only show nullable fields")
	fieldsCmd.Flags().StringVar(&nameFilter, "name", "", "Filter fields by name using a glob pattern (e.g., *Id, get*)")
	fieldsCmd.Flags().StringVar(&nameRegexFilter, "name-regex", "", "Filter fields by name using a regex pattern")
	fieldsCmd.Flags().BoolVar(&hasDescriptionFilter, "has-description", false, "Filter to only show fields that have a description")
}
