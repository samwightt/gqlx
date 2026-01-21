package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectZshEscapeIssue_NonStdin(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 1},
		},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "query.graphql")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_NoBackslashBang(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 1},
		},
	}
	content := `query { bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_NoLocations(t *testing.T) {
	err := ValidationError{
		Message:   "some error",
		Locations: []Location{},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_LineOutOfBounds_TooLow(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 0, Column: 1},
		},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_LineOutOfBounds_TooHigh(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 10, Column: 1},
		},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_BackslashBangAtErrorLocation(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 9}, // Position of \!
		},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "zsh escaped")
	assert.Contains(t, result, "heredoc")
}

func TestDetectZshEscapeIssue_BackslashBangNotAtErrorLocation(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 1}, // Not at \! position
		},
	}
	content := `query { \!bad }`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_MultilineContent(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 2, Column: 3}, // Position of \! on line 2
		},
	}
	content := "query {\n  \x5c!bad\n}"

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "heredoc")
}

func TestDetectZshEscapeIssue_ColumnNegative(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 0}, // Column 0 becomes col=-1
		},
	}
	content := `\!query`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestDetectZshEscapeIssue_ColumnAtEndOfLine(t *testing.T) {
	err := ValidationError{
		Message: "some error",
		Locations: []Location{
			{Line: 1, Column: 6}, // At the last character
		},
	}
	content := `hello\`

	result := detectZshEscapeIssue(err, content, "stdin")
	assert.Empty(t, result)
}

func TestErrorSpanLength_FieldsOnCorrectType(t *testing.T) {
	err := ValidationError{
		Message: `Cannot query field "badField" on type "Query".`,
		Rule:    "FieldsOnCorrectType",
	}

	length := errorSpanLength(err)
	assert.Equal(t, len("badField"), length)
}

func TestErrorSpanLength_FieldsOnCorrectType_NoMatch(t *testing.T) {
	err := ValidationError{
		Message: "Some other message format",
		Rule:    "FieldsOnCorrectType",
	}

	length := errorSpanLength(err)
	assert.Equal(t, 1, length)
}

func TestErrorSpanLength_UnknownRule(t *testing.T) {
	err := ValidationError{
		Message: "Some error message",
		Rule:    "SomeOtherRule",
	}

	length := errorSpanLength(err)
	assert.Equal(t, 1, length)
}

func TestParseFieldsOnCorrectTypeError_Valid(t *testing.T) {
	fieldName, typeName := parseFieldsOnCorrectTypeError(`Cannot query field "badField" on type "Query".`)
	assert.Equal(t, "badField", fieldName)
	assert.Equal(t, "Query", typeName)
}

func TestParseFieldsOnCorrectTypeError_Invalid(t *testing.T) {
	fieldName, typeName := parseFieldsOnCorrectTypeError("Some other message")
	assert.Empty(t, fieldName)
	assert.Empty(t, typeName)
}
