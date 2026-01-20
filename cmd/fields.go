/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if len(field.Arguments) == 0 {
		return field.Name
	}

	var args []string
	for _, arg := range field.Arguments {
		args = append(args, fmt.Sprintf("%s: %s", arg.Name, arg.Type))
	}

	if format == render.FormatPretty {
		return fmt.Sprintf("%s(\n\t\t%s\n\t)", field.Name, strings.Join(args, ",\n\t\t"))
	}
	return fmt.Sprintf("%s(%s)", field.Name, strings.Join(args, ", "))
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

// fieldsCmd represents the fields command
var fieldsCmd = &cobra.Command{
	Use:   "fields [flags] ",
	Short: "Lists fields on a type or input type.",
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
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		searchString := args[0]

		schema, err := loadCliForSchema()
		if err != nil {
			return err
		}

		graphqlType := schema.Types[searchString]
		if graphqlType == nil {
			var typeNames []string
			for name := range schema.Types {
				typeNames = append(typeNames, name)
			}
			if suggestion := findClosest(searchString, typeNames); suggestion != "" {
				return fmt.Errorf("type '%s' does not exist in schema, did you mean '%s'?", searchString, suggestion)
			}
			return fmt.Errorf("type '%s' does not exist in schema", searchString)
		}

		var fields []FieldInfo
		for _, field := range graphqlType.Fields {
			fields = append(fields, fieldToInfo(field))
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fieldsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fieldsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
