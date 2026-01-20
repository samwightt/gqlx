package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Format string

const (
	FormatJSON   Format = "json"
	FormatText   Format = "text"
	FormatPretty Format = "pretty"
)

var ValidFormats = []Format{FormatJSON, FormatText, FormatPretty}

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON, nil
	case "text":
		return FormatText, nil
	case "pretty":
		return FormatPretty, nil
	default:
		return "", fmt.Errorf("invalid format: %s (valid: json, text, pretty)", s)
	}
}

type Renderer[T any] struct {
	Data         []T
	TextFormat   func(T) string
	PrettyFormat func([]T) string
}

func (r Renderer[T]) Render(format Format) (string, error) {
	switch format {
	case FormatJSON:
		return r.renderJSON()
	case FormatPretty:
		return r.renderPretty()
	case FormatText:
		return r.renderText()
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (r Renderer[T]) renderPretty() (string, error) {
	if r.PrettyFormat == nil {
		return "", fmt.Errorf("pretty format not defined for this type")
	}
	return r.PrettyFormat(r.Data), nil
}

func (r Renderer[T]) renderJSON() (string, error) {
	bytes, err := json.MarshalIndent(r.Data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r Renderer[T]) renderText() (string, error) {
	if r.TextFormat == nil {
		return "", fmt.Errorf("text format not defined for this type")
	}

	var lines []string
	for _, item := range r.Data {
		lines = append(lines, r.TextFormat(item))
	}
	return strings.Join(lines, "\n"), nil
}
