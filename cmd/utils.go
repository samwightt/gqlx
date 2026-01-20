package cmd

import (
	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var tableStyle = lipgloss.NewStyle().PaddingRight(1)

func makeTable() *table.Table {
	return table.New().
		Width(120).
		Wrap(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			return tableStyle
		})
}

const maxSuggestionDistance = 5

func findClosest(input string, candidates []string) string {
	minDist := -1
	closest := ""
	for _, c := range candidates {
		dist := levenshtein.ComputeDistance(input, c)
		if minDist == -1 || dist < minDist {
			minDist = dist
			closest = c
		}
	}
	if minDist > maxSuggestionDistance {
		return ""
	}
	return closest
}
