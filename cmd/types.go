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

// typesCmd represents the types command
var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "Lists all type names in the schema.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schema, err := loadCliForSchema()
		if err != nil {
			return err
		}

		if implementsFilter != "" {
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
		}

		var types []TypeInfo
		for _, graphqlType := range schema.Types {
			if implementsFilter != "" {
				if !slices.Contains(graphqlType.Interfaces, implementsFilter) {
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
}
