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
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
	usedByFilter = ""
	// fields command flags
	deprecatedFilter = false
	hasArgFilter = nil
	returnsFilter = ""
	requiredFilter = false
	nullableFilter = false
	// args command flags
	argsDeprecatedFilter = false
	argsTypeFilter = ""
	argsRequiredFilter = false
	argsNullableFilter = false
	// paths command flags
	pathsMaxDepth = 5
	pathsFromType = ""
	pathsShortestOnly = false
	pathsThroughType = ""
	// values command flags
	valuesDeprecatedFilter = false
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
