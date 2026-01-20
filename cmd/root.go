/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"os"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gqlx",
	Short: "Search and explore GraphQL schema files by structure, not just text",
	Long: `gqlx is a tool for searching GraphQL schema files by their structure.
It understands the syntax in .graphql files and lets you explore using GraphQL concepts:
what interfaces a type implements, what a field returns, whether arguments are required, etc.
It's grep for GraphQL schema files.

Commands usually output lists that are filtered by various flags. You should check the help of
the command before using it, as there are lots of useful tools.

By default, gqlx tries to read ./schema.graphql in the current directory.
A different schema file can be specified using -s.

Output can be formatted as pretty tables (default in terminals), plain text
(default when piping), or JSON for integration with other tools.`,
	Example: `  # List all types in the schema
  gqlx types

  # Find all types used in Query fields that implement Node interface
  gqlx types --used-by Query --implements Node

  # Find deprecated pagination fields that need cleanup
  gqlx fields --deprecated --has-arg first --has-arg after

  # See the required ID arguments for a resolver.
  gqlx args Query.users --required --type ID

  # Find shortest way to query comments that go through Post
  gqlx paths Comment --through Post --shortest

  # Pipe JSON output to other tools
  gqlx types -f json | jq '.[].name'`,
}

var (
	schemaFilePath string
	outputFormat   render.Format
)

func formatFlag() string {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return string(render.FormatPretty)
	}
	return string(render.FormatText)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// ResetFlags resets command-specific flags to their default values.
// This is useful for testing to avoid state leaking between tests.
func ResetFlags() {
	// types command flags
	implementsFilter = ""
	hasFieldFilter = nil
	kindFilter = nil
	usedByFilter = nil
	usedByAnyFilter = nil
	notUsedByFilter = nil
	notUsedByAllFilter = nil
	typesNameFilter = ""
	typesNameRegexFilter = ""
	typesHasDescriptionFilter = false
	scalarFilter = false
	typeFilter = false
	interfaceFilter = false
	unionFilter = false
	enumFilter = false
	inputFilter = false
	// fields command flags
	deprecatedFilter = false
	hasArgFilter = nil
	returnsFilter = ""
	requiredFilter = false
	nullableFilter = false
	nameFilter = ""
	nameRegexFilter = ""
	hasDescriptionFilter = false
	// args command flags
	argsDeprecatedFilter = false
	argsTypeFilter = ""
	argsRequiredFilter = false
	argsNullableFilter = false
	argsNameFilter = ""
	argsNameRegexFilter = ""
	argsHasDescriptionFilter = false
	// paths command flags
	pathsMaxDepth = 5
	pathsFromType = ""
	pathsShortestOnly = false
	pathsThroughType = ""
	// values command flags
	valuesDeprecatedFilter = false
	valuesHasDescriptionFilter = false
	// references command flags
	refsKindFilter = ""
	refsInTypeFilter = ""
}

// ExecuteWithArgs runs the CLI with the given arguments and returns stdout, stderr, and any error.
// This is useful for testing.
func ExecuteWithArgs(args []string) (stdout string, stderr string, err error) {
	// Reset command-specific flags to avoid state leaking between tests
	ResetFlags()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	rootCmd.SetOut(stdoutBuf)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()

	return stdoutBuf.String(), stderrBuf.String(), err
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gqlx.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVarP(&schemaFilePath, "schema", "s", "schema.graphql", "File path of GraphQL schema")

	var formatStr string
	rootCmd.PersistentFlags().StringVarP(&formatStr, "format", "f", formatFlag(), "Output format: json, text, pretty (default: pretty if interactive, text otherwise)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		outputFormat, err = render.ParseFormat(formatStr)
		return err
	}
}
