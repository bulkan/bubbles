package progress

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/golden"
)

func TestBlend(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		width   int
		percent float64
	}{
		{
			name: "10w-red-to-green-50perc",
			options: []Option{
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithScaled(false),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "10w-red-to-green-50perc-full-block",
			options: []Option{
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithFillCharacters('█', DefaultEmptyCharBlock),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "30w-red-to-green-100perc",
			options: []Option{
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithScaled(false),
				WithoutPercentage(),
			},
			width:   30,
			percent: 1.0,
		},
		{
			name: "10w-red-to-green-scaled-50perc",
			options: []Option{
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithScaled(true),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "30w-red-to-green-scaled-100perc",
			options: []Option{
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithScaled(true),
				WithoutPercentage(),
			},
			width:   30,
			percent: 1.0,
		},
		{
			name: "30w-colorfunc-rgb-100perc",
			options: []Option{
				WithColorFunc(func(_, current float64) color.Color {
					if current <= 0.3 {
						return lipgloss.Color("#FF0000")
					}
					if current <= 0.7 {
						return lipgloss.Color("#00FF00")
					}
					return lipgloss.Color("#0000FF")
				}),
				WithoutPercentage(),
			},
			width:   30,
			percent: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.options...)
			p.SetWidth(tt.width)
			golden.RequireEqual(t, []byte(p.ViewAs(tt.percent)))
		})
	}
}

func TestLabel(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		width   int
		percent float64
	}{
		{
			name: "plain-0perc",
			options: []Option{
				WithLabel("hi"),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.0,
		},
		{
			name: "plain-50perc",
			options: []Option{
				WithLabel("hi"),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "plain-100perc",
			options: []Option{
				WithLabel("hi"),
				WithoutPercentage(),
			},
			width:   10,
			percent: 1.0,
		},
		{
			name: "styled-50perc",
			options: []Option{
				WithLabel("hi"),
				WithLabelStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "active-50perc",
			options: []Option{
				WithLabel("hi"),
				WithLabelActiveStyle(lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000000")).
					Background(lipgloss.Color("#FFFFFF"))),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "left-aligned-50perc",
			options: []Option{
				WithLabel("hi"),
				WithLabelAlignment(LabelAlignLeft),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "right-aligned-50perc",
			options: []Option{
				WithLabel("hi"),
				WithLabelAlignment(LabelAlignRight),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "left-aligned-0perc",
			options: []Option{
				WithLabel("hi"),
				WithLabelAlignment(LabelAlignLeft),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.0,
		},
		{
			name: "too-wide-50perc",
			options: []Option{
				WithLabel("this label is way too long"),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "blend-50perc",
			options: []Option{
				WithLabel("hi"),
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "scaled-blend-50perc",
			options: []Option{
				WithLabel("hi"),
				WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#00FF00")),
				WithScaled(true),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "colorfunc-50perc",
			options: []Option{
				WithLabel("hi"),
				WithColorFunc(func(_, current float64) color.Color {
					if current <= 0.5 {
						return lipgloss.Color("#FF0000")
					}
					return lipgloss.Color("#00FF00")
				}),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
		{
			name: "full-block-50perc",
			options: []Option{
				WithLabel("hi"),
				WithFillCharacters('█', DefaultEmptyCharBlock),
				WithoutPercentage(),
			},
			width:   10,
			percent: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.options...)
			p.SetWidth(tt.width)
			golden.RequireEqual(t, []byte(p.ViewAs(tt.percent)))
		})
	}
}
