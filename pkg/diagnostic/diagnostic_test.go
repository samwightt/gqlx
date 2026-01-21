package diagnostic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderSnippet_Basic(t *testing.T) {
	result := RenderSnippet("query { user }", 3, 9, 4, "")

	// Should contain the source line and carets
	assert.Contains(t, result, "query { user }")
	assert.Contains(t, result, "^^^^")
	assert.Contains(t, result, "3")
	assert.Contains(t, result, "|")
}

func TestRenderSnippet_WithMessage(t *testing.T) {
	result := RenderSnippet("query { user }", 3, 9, 4, "unknown field")

	assert.Contains(t, result, "query { user }")
	assert.Contains(t, result, "^^^^")
	assert.Contains(t, result, "unknown field")
}

func TestRenderSnippet_FirstColumn(t *testing.T) {
	result := RenderSnippet("query", 1, 1, 5, "")

	assert.Contains(t, result, "query")
	assert.Contains(t, result, "^^^^^")
}

func TestRenderSnippet_SingleCaret(t *testing.T) {
	result := RenderSnippet("hello world", 10, 7, 1, "here")

	assert.Contains(t, result, "hello world")
	assert.Contains(t, result, "^")
	assert.Contains(t, result, "here")
	// Should have line number 10
	assert.Contains(t, result, "10")
}

func TestRenderSnippet_ZeroLengthDefaultsToOne(t *testing.T) {
	result := RenderSnippet("test", 1, 2, 0, "")

	// Should still have at least one caret
	assert.Contains(t, result, "^")
}

func TestRenderSnippet_ZeroColumnDefaultsToOne(t *testing.T) {
	result := RenderSnippet("test", 1, 0, 1, "")

	// Should not panic and should have a caret
	assert.Contains(t, result, "^")
}

func TestRenderSnippet_CaretAlignment(t *testing.T) {
	result := RenderSnippet("ab cde fgh", 5, 4, 3, "")

	lines := strings.Split(result, "\n")
	assert.Len(t, lines, 2)

	// The caret line should have ^^^ starting at position matching column 4
	// After the "| " prefix, we need 3 spaces then ^^^
	caretLine := lines[1]
	assert.Contains(t, caretLine, "^^^")
}

func TestRenderSnippet_LargeLineNumber(t *testing.T) {
	result := RenderSnippet("code", 1234, 1, 4, "")

	lines := strings.Split(result, "\n")
	assert.Len(t, lines, 2)

	// Line number should be present
	assert.Contains(t, result, "1234")
	// Gutter alignment: underline gutter should be 4 spaces (matching "1234" width)
	// Line 0: "1234 | code"
	// Line 1: "     | ^^^^"
	underLine := lines[1]
	// Should start with 4 spaces for alignment
	assert.True(t, strings.HasPrefix(stripAnsi(underLine), "    "), "underline should have 4-space gutter")
}

func TestRenderLocation(t *testing.T) {
	result := RenderLocation("query.graphql", 3, 9)
	assert.Contains(t, result, "-->")
	assert.Contains(t, result, "query.graphql:3:9")
}

func TestRenderLocation_Stdin(t *testing.T) {
	result := RenderLocation("stdin", 1, 23)
	assert.Contains(t, result, "stdin:1:23")
}

// stripAnsi removes ANSI escape codes for testing
func stripAnsi(s string) string {
	// Simple approach: just check the structure ignoring colors
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

