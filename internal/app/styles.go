package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	styleStaged   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleDirty    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleConflict = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleAccent   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleBold     = lipgloss.NewStyle().Bold(true)
	styleFocus    = lipgloss.NewStyle().Reverse(true)
	styleWarning  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
)

const panelBorderSize = 2

func clip(line string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(line, width, "…")
}

func panel(title string, body []string, width, height int, focused bool) string {
	innerWidth, innerHeight := max(width-panelBorderSize, 1), max(height-panelBorderSize, 1)
	borderColor := lipgloss.Color("8")
	if focused {
		borderColor = lipgloss.Color("5")
		title = styleAccent.Render(title)
	}
	lines := append([]string{styleBold.Render(title)}, body...)
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	for index, line := range lines {
		lines[index] = clip(strings.ReplaceAll(line, "\t", "    "), innerWidth)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Width(innerWidth).
		Height(innerHeight).
		MaxHeight(height).
		Render(strings.Join(lines, "\n"))
}

func moveCursor(key string, cursor, total, pageRows int) (int, bool) {
	if total == 0 {
		return 0, key == "j" || key == "k" || key == "up" || key == "down" || key == "G" || key == "g g" || key == "ctrl+d" || key == "ctrl+u"
	}
	switch key {
	case "j", "down":
		return min(cursor+1, total-1), true
	case "k", "up":
		return max(cursor-1, 0), true
	case "g g":
		return 0, true
	case "G":
		return total - 1, true
	case "ctrl+d":
		return min(cursor+max(pageRows/2, 1), total-1), true
	case "ctrl+u":
		return max(cursor-max(pageRows/2, 1), 0), true
	}
	return cursor, false
}

func renderRows(rows []string, cursor, offset, visibleRows int, focused bool) ([]string, int) {
	switch {
	case visibleRows <= 0:
		offset = 0
	case cursor < offset:
		offset = cursor
	case cursor >= offset+visibleRows:
		offset = cursor - visibleRows + 1
	}
	var rendered []string
	for index := offset; index < len(rows) && index < offset+visibleRows; index++ {
		row := rows[index]
		if index == cursor && focused {
			row = styleFocus.Render("> " + ansi.Strip(row))
		} else if index == cursor {
			row = "> " + row
		} else {
			row = "  " + row
		}
		rendered = append(rendered, row)
	}
	return rendered, offset
}

func newTextInput() textinput.Model {
	input := textinput.New()
	input.Cursor.SetMode(cursor.CursorStatic)
	return input
}
