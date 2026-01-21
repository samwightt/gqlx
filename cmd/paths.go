/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/samwightt/gqlx/pkg/render"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

type pathsOptions struct {
	maxDepth     int
	fromType     string
	shortestOnly bool
	throughType  string
}

func formatPathStep(step pathStep) string {
	if step.hasArgs {
		return fmt.Sprintf("%s.%s(...)", step.typeName, step.fieldName)
	}
	return fmt.Sprintf("%s.%s", step.typeName, step.fieldName)
}

func formatPath(steps []pathStep, targetType string) string {
	if len(steps) == 0 {
		return targetType
	}

	parts := make([]string, len(steps))
	for i, step := range steps {
		parts[i] = formatPathStep(step)
	}

	return strings.Join(parts, " -> ") + " -> " + targetType
}

func formatPathText(p PathInfo) string {
	return p.Path
}

func formatPathsPretty(paths []PathInfo) string {
	t := makeTable()

	for _, p := range paths {
		t.Row(p.Path)
	}
	t.Headers("path")

	return t.String()
}

func findPaths(schema *ast.Schema, fromType string, targetType string, maxDepth int) []PathInfo {
	var results []PathInfo

	startType := schema.Types[fromType]
	if startType == nil {
		return results
	}

	type searchState struct {
		steps   []pathStep
		visited map[string]bool
	}

	queue := []searchState{{
		steps:   []pathStep{},
		visited: map[string]bool{fromType: true},
	}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if len(current.steps) > maxDepth {
			continue
		}

		// Determine current type we're exploring
		currentTypeName := fromType
		if len(current.steps) > 0 {
			lastStep := current.steps[len(current.steps)-1]
			// Get the return type of the last field
			parentType := schema.Types[lastStep.typeName]
			if parentType != nil {
				for _, f := range parentType.Fields {
					if f.Name == lastStep.fieldName {
						currentTypeName = getBaseTypeName(f.Type)
						break
					}
				}
			}
		}

		currentType := schema.Types[currentTypeName]
		if currentType == nil {
			continue
		}

		for _, field := range currentType.Fields {
			fieldReturnType := getBaseTypeName(field.Type)

			newStep := pathStep{
				typeName:  currentTypeName,
				fieldName: field.Name,
				hasArgs:   len(field.Arguments) > 0,
			}

			newSteps := make([]pathStep, len(current.steps)+1)
			copy(newSteps, current.steps)
			newSteps[len(current.steps)] = newStep

			// Check if this field returns our target type
			if fieldReturnType == targetType {
				results = append(results, PathInfo{
					Path: formatPath(newSteps, targetType),
				})
			}

			// Continue searching if we haven't visited this type and haven't exceeded depth
			if !current.visited[fieldReturnType] && len(newSteps) < maxDepth {
				returnTypeDef := schema.Types[fieldReturnType]
				// Only continue if it's an object type with fields
				if returnTypeDef != nil && (returnTypeDef.Kind == ast.Object || returnTypeDef.Kind == ast.Interface) && len(returnTypeDef.Fields) > 0 {
					newVisited := make(map[string]bool)
					maps.Copy(newVisited, current.visited)
					newVisited[fieldReturnType] = true

					queue = append(queue, searchState{
						steps:   newSteps,
						visited: newVisited,
					})
				}
			}
		}
	}

	// Sort results for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results
}

func NewPathsCmd() *cobra.Command {
	opts := &pathsOptions{
		maxDepth: 5,
	}

	cmd := &cobra.Command{
		Use:   "paths <type>",
		Short: "Lists all paths from Query to a given type.",
		Args:  cobra.ExactArgs(1),
		Long: `Lists all possible paths from a root type to reach a given type.

By default, searches from Query. Use --from to start from a different type.
Use --shortest to only show the shortest path(s).

For example, if User can be reached via Query.user(id: ID!) or via
Query.viewer -> Viewer.friends, both paths will be shown.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaths(cmd, args, opts)
		},
	}

	cmd.Flags().IntVar(&opts.maxDepth, "max-depth", 5, "Maximum depth to search for paths")
	cmd.Flags().StringVar(&opts.fromType, "from", "", "Type to start searching from (default: Query)")
	cmd.Flags().BoolVar(&opts.shortestOnly, "shortest", false, "Only show the shortest path(s)")
	cmd.Flags().StringVar(&opts.throughType, "through", "", "Only show paths that pass through the given type")

	return cmd
}

func runPaths(cmd *cobra.Command, args []string, opts *pathsOptions) error {
	targetType := args[0]

	schema, err := loadCliForSchema()
	if err != nil {
		return err
	}

	// Validate target type exists
	if err := validateTypeExists(schema, targetType, "type"); err != nil {
		return err
	}

	// Validate from type exists
	fromType := opts.fromType
	if fromType == "" {
		fromType = "Query"
	}
	if err := validateTypeExists(schema, fromType, "type"); err != nil {
		return err
	}

	// Validate through type exists if specified
	if opts.throughType != "" {
		if err := validateTypeExists(schema, opts.throughType, "type"); err != nil {
			return err
		}
	}

	paths := findPaths(schema, fromType, targetType, opts.maxDepth)

	// Filter to paths through specific type if requested
	if opts.throughType != "" {
		paths = filterSlice(paths, func(p PathInfo) bool {
			return strings.Contains(p.Path, opts.throughType+".")
		})
	}

	// Filter to shortest paths if requested
	if opts.shortestOnly && len(paths) > 0 {
		minDepth := len(strings.Split(paths[0].Path, " -> "))
		for _, p := range paths {
			depth := len(strings.Split(p.Path, " -> "))
			if depth < minDepth {
				minDepth = depth
			}
		}
		paths = filterSlice(paths, func(p PathInfo) bool {
			return len(strings.Split(p.Path, " -> ")) == minDepth
		})
	}

	if len(paths) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No paths found that match the filters.")
	}

	renderer := render.Renderer[PathInfo]{
		Data:         paths,
		TextFormat:   formatPathText,
		PrettyFormat: formatPathsPretty,
	}

	output, err := renderer.Render(outputFormat)
	if err != nil {
		return fmt.Errorf("error rendering output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), output)
	return nil
}
