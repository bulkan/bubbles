// Package progress provides a simple progress bar for Bubble Tea applications.
package progress

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/x/ansi"
)

// ColorFunc is a function that can be used to dynamically fill the progress
// bar based on the current percentage. total is the total filled percentage,
// and current is the current percentage that is actively being filled with a
// color.
type ColorFunc func(total, current float64) color.Color

// Internal ID management. Used during animating to assure that frame messages
// can only be received by progress components that sent them.
var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

// LabelAlignment specifies how the label is aligned within the progress bar.
type LabelAlignment int

// Label alignment options.
const (
	// LabelAlignCenter centers the label within the progress bar.
	LabelAlignCenter LabelAlignment = iota
	// LabelAlignLeft left-aligns the label within the progress bar.
	LabelAlignLeft
	// LabelAlignRight right-aligns the label within the progress bar.
	LabelAlignRight
)

const (
	// DefaultFullCharHalfBlock is the default character used to fill the progress
	// bar. It is a half block, which allows more granular color blending control,
	// by having a different foreground and background color, doubling blending
	// resolution.
	DefaultFullCharHalfBlock = '▌'

	// DefaultFullCharFullBlock can also be used as a fill character for the
	// progress bar. Use this to disable the higher resolution blending which is
	// enabled when using [DefaultFullCharHalfBlock].
	DefaultFullCharFullBlock = '█'

	// DefaultEmptyCharBlock is the default character used to fill the empty
	// portion of the progress bar.
	DefaultEmptyCharBlock = '░'

	fps              = 60
	defaultWidth     = 40
	defaultFrequency = 18.0
	defaultDamping   = 1.0
)

var (
	defaultBlendStart       = lipgloss.Color("#5A56E0") // Purple haze.
	defaultBlendEnd         = lipgloss.Color("#EE6FF8") // Neon pink.
	defaultFullColor        = lipgloss.Color("#7571F9") // Blueberry.
	defaultEmptyColor       = lipgloss.Color("#606060") // Slate gray.
	defaultLabelActiveColor = lipgloss.Color("#1A1A1A")
)

// Option is used to set options in [New]. For example:
//
//	progress := New(
//		WithColors(
//			lipgloss.Color("#5A56E0"),
//			lipgloss.Color("#EE6FF8"),
//		),
//		WithoutPercentage(),
//	)
type Option func(*Model)

// WithDefaultBlend sets a default blend of colors, which is a blend of purple
// haze to neon pink.
func WithDefaultBlend() Option {
	return WithColors(
		defaultBlendStart,
		defaultBlendEnd,
	)
}

// WithColors sets the colors to use to fill the progress bar. Depending on the
// number of colors passed in, will determine whether to use a solid fill or a
// blend of colors.
//
//   - 0 colors: clears all previously set colors, setting them back to defaults.
//   - 1 color: uses a solid fill with the given color.
//   - 2+ colors: uses a blend of the provided colors.
func WithColors(colors ...color.Color) Option {
	if len(colors) == 0 {
		return func(m *Model) {
			m.FullColor = defaultFullColor
			m.blend = nil
			m.colorFunc = nil
		}
	}
	if len(colors) == 1 {
		return func(m *Model) {
			m.FullColor = colors[0]
			m.colorFunc = nil
			m.blend = nil
		}
	}
	return func(m *Model) {
		m.blend = colors
	}
}

// WithColorFunc sets a function that can be used to dynamically fill the progress
// bar based on the current percentage. total is the total filled percentage, and
// current is the current percentage that is actively being filled with a color.
// When specified, this overrides any other defined colors and scaling.
//
// Example: A progress bar that changes color based on the total completed
// percentage:
//
//	WithColorFunc(func(total, current float64) color.Color {
//		if total <= 0.3 {
//			return lipgloss.Color("#FF0000")
//		}
//		if total <= 0.7 {
//			return lipgloss.Color("#00FF00")
//		}
//		return lipgloss.Color("#0000FF")
//	}),
func WithColorFunc(fn ColorFunc) Option {
	return func(m *Model) {
		m.colorFunc = fn
		m.blend = nil
	}
}

// WithFillCharacters sets the characters used to construct the full and empty
// components of the progress bar.
func WithFillCharacters(full rune, empty rune) Option {
	return func(m *Model) {
		m.Full = full
		m.Empty = empty
	}
}

// WithLabel sets a label that will be rendered within the progress bar, on
// top of the fill. By default it's centered; use [WithLabelAlignment] to
// change the alignment. If the label is wider than the bar it won't be
// rendered.
func WithLabel(label string) Option {
	return func(m *Model) {
		m.Label = label
	}
}

// WithLabelAlignment sets the alignment of the label within the progress bar.
// The default is to center the label.
func WithLabelAlignment(a LabelAlignment) Option {
	return func(m *Model) {
		m.LabelAlignment = a
	}
}

// WithLabelStyle sets the style of label characters that the fill hasn't
// reached yet.
func WithLabelStyle(style lipgloss.Style) Option {
	return func(m *Model) {
		m.LabelStyle = style
	}
}

// WithLabelActiveStyle sets the style of label characters that the fill has
// crossed. Its foreground is used as the text color; its background defaults
// to the bar's fill color at that position, so the label appears painted on
// the bar. Only the foreground and background colors of the style are
// applied; other style properties are ignored for these characters.
func WithLabelActiveStyle(style lipgloss.Style) Option {
	return func(m *Model) {
		m.LabelActiveStyle = style
	}
}

// WithoutPercentage hides the numeric percentage.
func WithoutPercentage() Option {
	return func(m *Model) {
		m.ShowPercentage = false
	}
}

// WithWidth sets the initial width of the progress bar. Note that you can also
// set the width via the Width property, which can come in handy if you're
// waiting for a tea.WindowSizeMsg.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.SetWidth(w)
	}
}

// WithSpringOptions sets the initial frequency and damping options for the
// progress bar's built-in spring-based animation. Frequency corresponds to
// speed, and damping to bounciness. For details see:
//
// https://github.com/charmbracelet/harmonica
func WithSpringOptions(frequency, damping float64) Option {
	return func(m *Model) {
		m.SetSpringOptions(frequency, damping)
		m.springCustomized = true
	}
}

// WithScaled sets whether to scale the blend/gradient to fit the width of only
// the filled portion of the progress bar. The default is false, which means the
// percentage must be 100% to see the full color blend/gradient.
//
// This is ignored when not using blending/multiple colors.
func WithScaled(enabled bool) Option {
	return func(m *Model) {
		m.scaleBlend = enabled
	}
}

// FrameMsg indicates that an animation step should occur.
type FrameMsg struct {
	id  int
	tag int
}

// Model stores values we'll use when rendering the progress bar.
type Model struct {
	// An identifier to keep us from receiving messages intended for other
	// progress bars.
	id int

	// An identifier to keep us from receiving frame messages too quickly.
	tag int

	// Total width of the progress bar, including percentage, if set.
	width int

	// "Filled" sections of the progress bar.
	Full      rune
	FullColor color.Color

	// "Empty" sections of the progress bar.
	Empty      rune
	EmptyColor color.Color

	// Settings for rendering the numeric percentage.
	ShowPercentage  bool
	PercentFormat   string // a fmt string for a float
	PercentageStyle lipgloss.Style

	// LabelStyle is applied to label characters that the fill hasn't reached
	// yet. LabelActiveStyle is applied to characters the fill has crossed: its
	// foreground is the text color and its background defaults to the bar's
	// fill color, so the label appears painted on the bar.
	Label            string
	LabelStyle       lipgloss.Style
	LabelActiveStyle lipgloss.Style
	// LabelAlignment determines how the label is aligned within the progress
	// bar. The default is LabelAlignCenter.
	LabelAlignment LabelAlignment

	// Members for animated transitions.
	spring           harmonica.Spring
	springCustomized bool
	percentShown     float64 // percent currently displaying
	targetPercent    float64 // percent to which we're animating
	velocity         float64

	// Blend of colors to use. When len < 1, we use FullColor.
	blend []color.Color

	// When true, we scale the blended colors to fit the width of the filled
	// section of the progress bar. When false, the width of the blend will be
	// set to the full width of the progress bar.
	scaleBlend bool

	// colorFunc is used to dynamically fill the progress bar based on the
	// current percentage.
	colorFunc ColorFunc
}

// New returns a model with default values.
func New(opts ...Option) Model {
	m := Model{
		id:             nextID(),
		width:          defaultWidth,
		Full:           DefaultFullCharHalfBlock,
		FullColor:      defaultFullColor,
		Empty:          DefaultEmptyCharBlock,
		EmptyColor:     defaultEmptyColor,
		ShowPercentage: true,
		PercentFormat:  " %3.0f%%",
		LabelAlignment: LabelAlignCenter,
	}

	for _, opt := range opts {
		opt(&m)
	}

	if !m.springCustomized {
		m.SetSpringOptions(defaultFrequency, defaultDamping)
	}

	return m
}

// Init exists to satisfy the tea.Model interface.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update is used to animate the progress bar during transitions. Use
// SetPercent to create the command you'll need to trigger the animation.
//
// If you're rendering with ViewAs you won't need this.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FrameMsg:
		if msg.id != m.id || msg.tag != m.tag {
			return m, nil
		}

		// If we've more or less reached equilibrium, stop updating.
		if !m.IsAnimating() {
			return m, nil
		}

		m.percentShown, m.velocity = m.spring.Update(m.percentShown, m.velocity, m.targetPercent)
		return m, m.nextFrame()

	default:
		return m, nil
	}
}

// SetSpringOptions sets the frequency and damping for the current spring.
// Frequency corresponds to speed, and damping to bounciness. For details see:
//
// https://github.com/charmbracelet/harmonica
func (m *Model) SetSpringOptions(frequency, damping float64) {
	m.spring = harmonica.NewSpring(harmonica.FPS(fps), frequency, damping)
}

// Percent returns the current visible percentage on the model. This is only
// relevant when you're animating the progress bar.
//
// If you're rendering with ViewAs you won't need this.
func (m Model) Percent() float64 {
	return m.targetPercent
}

// SetPercent sets the percentage state of the model as well as a command
// necessary for animating the progress bar to this new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) SetPercent(p float64) tea.Cmd {
	m.targetPercent = math.Max(0, math.Min(1, p))
	m.tag++
	return m.nextFrame()
}

// IncrPercent increments the percentage by a given amount, returning a command
// necessary to animate the progress bar to the new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) IncrPercent(v float64) tea.Cmd {
	return m.SetPercent(m.Percent() + v)
}

// DecrPercent decrements the percentage by a given amount, returning a command
// necessary to animate the progress bar to the new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) DecrPercent(v float64) tea.Cmd {
	return m.SetPercent(m.Percent() - v)
}

// View renders an animated progress bar in its current state. To render
// a static progress bar based on your own calculations use ViewAs instead.
func (m Model) View() string {
	return m.ViewAs(m.percentShown)
}

// ViewAs renders the progress bar with a given percentage.
func (m Model) ViewAs(percent float64) string {
	b := strings.Builder{}
	percentView := m.percentageView(percent)
	m.barView(&b, percent, ansi.StringWidth(percentView))
	b.WriteString(percentView)
	return b.String()
}

// SetWidth sets the width of the progress bar.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// Width returns the width of the progress bar.
func (m Model) Width() int {
	return m.width
}

func (m *Model) nextFrame() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(time.Time) tea.Msg {
		return FrameMsg{id: m.id, tag: m.tag}
	})
}

func (m Model) barView(b *strings.Builder, percent float64, textWidth int) {
	var (
		tw = max(0, m.width-textWidth)                // total width
		fw = int(math.Round((float64(tw) * percent))) // filled width
	)

	fw = max(0, min(tw, fw))

	if lw := ansi.StringWidth(m.Label); lw > 0 && lw <= tw {
		m.labelView(b, percent, tw, fw, lw)
		return
	}

	isHalfBlock := m.Full == DefaultFullCharHalfBlock

	if m.colorFunc != nil { //nolint:nestif
		var style lipgloss.Style
		var current float64
		halfBlockPerc := 0.5 / float64(tw)
		for i := range fw {
			current = float64(i) / float64(tw)
			style = style.Foreground(m.colorFunc(percent, current))
			if isHalfBlock {
				style = style.Background(m.colorFunc(percent, min(current+halfBlockPerc, 1)))
			}
			b.WriteString(style.Render(string(m.Full)))
		}
	} else if len(m.blend) > 0 {
		var blend []color.Color

		multiplier := 1
		if isHalfBlock {
			multiplier = 2
		}

		if m.scaleBlend {
			blend = lipgloss.Blend1D(fw*multiplier, m.blend...)
		} else {
			blend = lipgloss.Blend1D(tw*multiplier, m.blend...)
		}

		// Blend fill.
		var blendIndex int
		for i := range fw {
			if !isHalfBlock {
				b.WriteString(lipgloss.NewStyle().
					Foreground(blend[i]).
					Render(string(m.Full)))
				continue
			}

			b.WriteString(lipgloss.NewStyle().
				Foreground(blend[blendIndex]).
				Background(blend[blendIndex+1]).
				Render(string(m.Full)))
			blendIndex += 2
		}
	} else {
		// Solid fill.
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.FullColor).
			Render(strings.Repeat(string(m.Full), fw)))
	}

	// Empty fill.
	n := max(0, tw-fw)
	b.WriteString(lipgloss.NewStyle().
		Foreground(m.EmptyColor).
		Render(strings.Repeat(string(m.Empty), n)))
}

func (m Model) labelView(b *strings.Builder, percent float64, tw, fw, lw int) {
	var ls int // label start column
	switch m.LabelAlignment {
	case LabelAlignLeft:
		ls = 0
	case LabelAlignRight:
		ls = max(0, tw-lw)
	default:
		ls = max(0, (tw-lw)/2)
	}
	le := min(tw, ls+lw) // label end column

	blend := m.makeBlend(tw, fw)

	label := []rune(m.Label)
	for i := 0; i < tw; i++ {
		switch {
		case i >= ls && i < le:
			j := i - ls
			if j >= len(label) {
				continue
			}
			if i < fw {
				b.WriteString(m.paintedLabelChar(label[j], percent, i, tw, blend))
			} else {
				b.WriteString(m.LabelStyle.Inline(true).Render(string(label[j])))
			}
		case i < fw:
			b.WriteString(m.fillChar(percent, i, tw, blend))
		default:
			b.WriteString(lipgloss.NewStyle().
				Foreground(m.EmptyColor).
				Render(string(m.Empty)))
		}
	}
}

// makeBlend precomputes the blend colors for the fill, or returns nil when no
// blend is configured.
func (m Model) makeBlend(tw, fw int) []color.Color {
	if len(m.blend) == 0 {
		return nil
	}

	multiplier := 1
	if m.Full == DefaultFullCharHalfBlock {
		multiplier = 2
	}

	if m.scaleBlend {
		return lipgloss.Blend1D(fw*multiplier, m.blend...)
	}
	return lipgloss.Blend1D(tw*multiplier, m.blend...)
}

// fillChar returns the rendered filled character for bar column i, applying
// the colorFunc, blend, or solid fill mode.
func (m Model) fillChar(percent float64, i, tw int, blend []color.Color) string {
	if m.colorFunc != nil {
		var style lipgloss.Style
		current := float64(i) / float64(tw)
		style = style.Foreground(m.colorFunc(percent, current))
		if m.Full == DefaultFullCharHalfBlock {
			style = style.Background(m.colorFunc(percent, min(current+0.5/float64(tw), 1)))
		}
		return style.Render(string(m.Full))
	}

	if blend != nil {
		multiplier := 1
		if m.Full == DefaultFullCharHalfBlock {
			multiplier = 2
		}
		if m.Full == DefaultFullCharHalfBlock {
			return lipgloss.NewStyle().
				Foreground(blend[i*multiplier]).
				Background(blend[i*multiplier+1]).
				Render(string(m.Full))
		}
		return lipgloss.NewStyle().
			Foreground(blend[i*multiplier]).
			Render(string(m.Full))
	}

	// Solid fill.
	return lipgloss.NewStyle().
		Foreground(m.FullColor).
		Render(string(m.Full))
}

// fillColor returns the color of the fill at bar column i, used as the
// background of label characters painted on the bar. In blend mode this is
// the cell's primary color.
func (m Model) fillColor(percent float64, i, tw int, blend []color.Color) color.Color {
	if m.colorFunc != nil {
		return m.colorFunc(percent, float64(i)/float64(tw))
	}
	if blend != nil {
		multiplier := 1
		if m.Full == DefaultFullCharHalfBlock {
			multiplier = 2
		}
		return blend[i*multiplier]
	}
	return m.FullColor
}

// paintedLabelChar renders a single label character that the fill has
// crossed. The character is drawn in the active label style's foreground
// color on the bar's fill color, so it appears painted on the bar.
func (m Model) paintedLabelChar(ch rune, percent float64, i, tw int, blend []color.Color) string {
	var (
		fg = m.LabelActiveStyle.GetForeground()
		bg = m.LabelActiveStyle.GetBackground()
	)

	// Foreground from the active style, defaulting to a dark color.
	if _, ok := fg.(lipgloss.NoColor); ok {
		fg = defaultLabelActiveColor
	}

	// Background defaults to the bar's fill color at this column; an explicit
	// background in the active style overrides it.
	if _, ok := bg.(lipgloss.NoColor); ok {
		bg = m.fillColor(percent, i, tw, blend)
	}

	return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(string(ch))
}

func (m Model) percentageView(percent float64) string {
	if !m.ShowPercentage {
		return ""
	}
	percent = math.Max(0, math.Min(1, percent))
	percentage := fmt.Sprintf(m.PercentFormat, percent*100) //nolint:mnd
	percentage = m.PercentageStyle.Inline(true).Render(percentage)
	return percentage
}

// IsAnimating returns false if the progress bar reached equilibrium and is no
// longer animating.
func (m *Model) IsAnimating() bool {
	dist := math.Abs(m.percentShown - m.targetPercent)
	return !(dist < 0.001 && m.velocity < 0.01)
}
