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

// NewRootCmd creates and returns the root command with all subcommands attached.
// This function creates a fresh command tree, ensuring no state leaks between invocations.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
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

	// Persistent flags
	cmd.PersistentFlags().StringVarP(&schemaFilePath, "schema", "s", "schema.graphql", "File path of GraphQL schema")

	var formatStr string
	cmd.PersistentFlags().StringVarP(&formatStr, "format", "f", formatFlag(), "Output format: json, text, pretty (default: pretty if interactive, text otherwise)")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		outputFormat, err = render.ParseFormat(formatStr)
		return err
	}

	// Add all subcommands
	cmd.AddCommand(NewTypesCmd())
	cmd.AddCommand(NewFieldsCmd())
	cmd.AddCommand(NewArgsCmd())
	cmd.AddCommand(NewPathsCmd())
	cmd.AddCommand(NewValuesCmd())
	cmd.AddCommand(NewReferencesCmd())
	cmd.AddCommand(NewValidateCmd())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// ExecuteWithArgs runs the CLI with the given arguments and returns stdout, stderr, and any error.
// This is useful for testing.
func ExecuteWithArgs(args []string) (stdout string, stderr string, err error) {
	return ExecuteWithArgsAndStdin(args, nil)
}

// ExecuteWithArgsAndStdin runs the CLI with the given arguments and stdin, returns stdout, stderr, and any error.
// This is useful for testing commands that read from stdin.
func ExecuteWithArgsAndStdin(args []string, stdin *bytes.Buffer) (stdout string, stderr string, err error) {
	// Create a fresh command tree - no need for ResetFlags() anymore!
	cmd := NewRootCmd()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	cmd.SetOut(stdoutBuf)
	cmd.SetErr(stderrBuf)
	cmd.SetArgs(args)
	if stdin != nil {
		cmd.SetIn(stdin)
	}

	err = cmd.Execute()

	return stdoutBuf.String(), stderrBuf.String(), err
}
