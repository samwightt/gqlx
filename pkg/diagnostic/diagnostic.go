// Package diagnostic provides utilities for rendering diagnostic messages
// with source code snippets and underlines.
package diagnostic

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	gutterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	caretStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// RenderSnippet renders a source line with line number, gutter, and underline caret.
// Returns something like:
//
//	3 | query { user }
//	  |         ^^^^ error message here
func RenderSnippet(source string, lineNum int, column int, length int, message string) string {
	if length < 1 {
		length = 1
	}
	if column < 1 {
		column = 1
	}

	numStr := strconv.Itoa(lineNum)
	gutterWidth := len(numStr)

	lineNumStyled := gutterStyle.Render(numStr)
	pipe := gutterStyle.Render("|")
	emptyGutter := strings.Repeat(" ", gutterWidth)

	// Line with number: "3 | query { user }"
	codeLine := lineNumStyled + " " + pipe + " " + source

	// Underline line: "  |         ^^^^"
	padding := strings.Repeat(" ", column-1)
	carets := caretStyle.Render(strings.Repeat("^", length))
	msgRendered := ""
	if message != "" {
		msgRendered = " " + messageStyle.Render(message)
	}
	underLine := emptyGutter + " " + pipe + " " + padding + carets + msgRendered

	return codeLine + "\n" + underLine
}

// RenderLocation renders a location header like "--> file.graphql:3:9"
func RenderLocation(filename string, line int, column int) string {
	loc := filename + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column)
	arrow := gutterStyle.Render("-->")
	return arrow + " " + loc
}
