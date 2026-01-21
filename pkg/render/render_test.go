package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFormat_JSON(t *testing.T) {
	format, err := ParseFormat("json")
	require.NoError(t, err)
	assert.Equal(t, FormatJSON, format)
}

func TestParseFormat_JSON_CaseInsensitive(t *testing.T) {
	format, err := ParseFormat("JSON")
	require.NoError(t, err)
	assert.Equal(t, FormatJSON, format)

	format, err = ParseFormat("Json")
	require.NoError(t, err)
	assert.Equal(t, FormatJSON, format)
}

func TestParseFormat_Text(t *testing.T) {
	format, err := ParseFormat("text")
	require.NoError(t, err)
	assert.Equal(t, FormatText, format)
}

func TestParseFormat_Text_CaseInsensitive(t *testing.T) {
	format, err := ParseFormat("TEXT")
	require.NoError(t, err)
	assert.Equal(t, FormatText, format)
}

func TestParseFormat_Pretty(t *testing.T) {
	format, err := ParseFormat("pretty")
	require.NoError(t, err)
	assert.Equal(t, FormatPretty, format)
}

func TestParseFormat_Pretty_CaseInsensitive(t *testing.T) {
	format, err := ParseFormat("PRETTY")
	require.NoError(t, err)
	assert.Equal(t, FormatPretty, format)
}

func TestParseFormat_Invalid(t *testing.T) {
	_, err := ParseFormat("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
	assert.Contains(t, err.Error(), "json, text, pretty")
}

func TestParseFormat_Empty(t *testing.T) {
	_, err := ParseFormat("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

// Test data type for Renderer tests
type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestRenderer_RenderJSON(t *testing.T) {
	data := []testItem{
		{Name: "first", Value: 1},
		{Name: "second", Value: 2},
	}

	renderer := Renderer[testItem]{
		Data: data,
	}

	output, err := renderer.Render(FormatJSON)
	require.NoError(t, err)

	assert.Contains(t, output, `"name": "first"`)
	assert.Contains(t, output, `"value": 1`)
	assert.Contains(t, output, `"name": "second"`)
	assert.Contains(t, output, `"value": 2`)
}

func TestRenderer_RenderJSON_EmptyData(t *testing.T) {
	renderer := Renderer[testItem]{
		Data: []testItem{},
	}

	output, err := renderer.Render(FormatJSON)
	require.NoError(t, err)
	assert.Equal(t, "[]", output)
}

func TestRenderer_RenderJSON_NilData(t *testing.T) {
	renderer := Renderer[testItem]{
		Data: nil,
	}

	output, err := renderer.Render(FormatJSON)
	require.NoError(t, err)
	assert.Equal(t, "null", output)
}

func TestRenderer_RenderText(t *testing.T) {
	data := []testItem{
		{Name: "first", Value: 1},
		{Name: "second", Value: 2},
	}

	renderer := Renderer[testItem]{
		Data: data,
		TextFormat: func(item testItem) string {
			return item.Name
		},
	}

	output, err := renderer.Render(FormatText)
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond", output)
}

func TestRenderer_RenderText_SingleItem(t *testing.T) {
	data := []testItem{
		{Name: "only", Value: 42},
	}

	renderer := Renderer[testItem]{
		Data: data,
		TextFormat: func(item testItem) string {
			return item.Name
		},
	}

	output, err := renderer.Render(FormatText)
	require.NoError(t, err)
	assert.Equal(t, "only", output)
}

func TestRenderer_RenderText_EmptyData(t *testing.T) {
	renderer := Renderer[testItem]{
		Data: []testItem{},
		TextFormat: func(item testItem) string {
			return item.Name
		},
	}

	output, err := renderer.Render(FormatText)
	require.NoError(t, err)
	assert.Equal(t, "", output)
}

func TestRenderer_RenderText_NilTextFormat(t *testing.T) {
	data := []testItem{
		{Name: "test", Value: 1},
	}

	renderer := Renderer[testItem]{
		Data:       data,
		TextFormat: nil,
	}

	_, err := renderer.Render(FormatText)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text format not defined")
}

func TestRenderer_RenderPretty(t *testing.T) {
	data := []testItem{
		{Name: "first", Value: 1},
		{Name: "second", Value: 2},
	}

	renderer := Renderer[testItem]{
		Data: data,
		PrettyFormat: func(items []testItem) string {
			return "pretty table output"
		},
	}

	output, err := renderer.Render(FormatPretty)
	require.NoError(t, err)
	assert.Equal(t, "pretty table output", output)
}

func TestRenderer_RenderPretty_NilPrettyFormat(t *testing.T) {
	data := []testItem{
		{Name: "test", Value: 1},
	}

	renderer := Renderer[testItem]{
		Data:         data,
		PrettyFormat: nil,
	}

	_, err := renderer.Render(FormatPretty)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pretty format not defined")
}

func TestRenderer_RenderUnknownFormat(t *testing.T) {
	data := []testItem{
		{Name: "test", Value: 1},
	}

	renderer := Renderer[testItem]{
		Data: data,
	}

	_, err := renderer.Render(Format("unknown"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestValidFormats(t *testing.T) {
	assert.Len(t, ValidFormats, 3)
	assert.Contains(t, ValidFormats, FormatJSON)
	assert.Contains(t, ValidFormats, FormatText)
	assert.Contains(t, ValidFormats, FormatPretty)
}
