package app

import "github.com/charmbracelet/lipgloss"

const (
	colorAccent            = lipgloss.Color("12")
	colorMuted             = lipgloss.Color("8")
	colorError             = lipgloss.Color("9")
	colorWarning           = lipgloss.Color("11")
	colorStaged            = lipgloss.Color("10")
	stackedWidthThreshold  = 100
	stackedHeightThreshold = 24
)

type styles struct {
	header    lipgloss.Style
	focused   lipgloss.Style
	blurred   lipgloss.Style
	selected  lipgloss.Style
	staged    lipgloss.Style
	unstaged  lipgloss.Style
	conflict  lipgloss.Style
	warning   lipgloss.Style
	disabled  lipgloss.Style
	statusBar lipgloss.Style
	overlay   lipgloss.Style
}

var appStyles = styles{
	header:    lipgloss.NewStyle().Bold(true),
	focused:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent),
	blurred:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted),
	selected:  lipgloss.NewStyle().Reverse(true),
	staged:    lipgloss.NewStyle().Foreground(colorStaged),
	conflict:  lipgloss.NewStyle().Foreground(colorError),
	warning:   lipgloss.NewStyle().Foreground(colorWarning),
	disabled:  lipgloss.NewStyle().Faint(true),
	statusBar: lipgloss.NewStyle().Faint(true),
	overlay:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
}

type layout struct {
	stacked      bool
	listWidth    int
	detailWidth  int
	listHeight   int
	detailHeight int
}

func layoutFor(width, height int) layout {
	return layout{stacked: width < stackedWidthThreshold || height < stackedHeightThreshold}
}
