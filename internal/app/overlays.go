package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	overlayMaxWidth   = 90
	overlayMinWidth   = 40
	overlayPadding    = 4
	listOverlayRows   = 16
	textPromptWidth   = 60
	dialogStyleColor  = "5"
	dangerDialogColor = "1"
)

func overlayBox(title string, lines []string, width int, danger bool) string {
	boxWidth := max(min(width-overlayPadding, overlayMaxWidth), overlayMinWidth)
	color := dialogStyleColor
	if danger {
		color = dangerDialogColor
	}
	wrap := lipgloss.NewStyle().Width(boxWidth - overlayPadding)
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, wrap.Render(line))
	}
	content := styleBold.Render(clip(title, boxWidth-overlayPadding)) + "\n\n" + strings.Join(wrapped, "\n")
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(color)).Padding(0, 1).Width(boxWidth).Render(content)
}

type confirmButton struct {
	label string
	run   func(a *App) tea.Cmd
}

type confirmOverlay struct {
	title   string
	lines   []string
	command string
	buttons []confirmButton
	focus   int
	danger  bool
}

func (c *confirmOverlay) handleKey(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		a.overlay = nil
	case "tab", "right", "l":
		c.focus = (c.focus + 1) % len(c.buttons)
	case "shift+tab", "left", "h":
		c.focus = (c.focus - 1 + len(c.buttons)) % len(c.buttons)
	case "enter":
		a.overlay = nil
		if run := c.buttons[c.focus].run; run != nil {
			return run(a)
		}
	}
	return nil
}

func (c *confirmOverlay) view(a *App, width, height int) string {
	lines := append([]string{}, c.lines...)
	if c.command != "" {
		lines = append(lines, "", styleMuted.Render("Git command: ")+c.command)
	}
	var buttons []string
	for index, button := range c.buttons {
		label := "[ " + button.label + " ]"
		if index == c.focus {
			label = styleFocus.Render(label)
		}
		buttons = append(buttons, label)
	}
	lines = append(lines, "", strings.Join(buttons, "  "), styleMuted.Render("Tab switch · Enter choose · Esc cancel"))
	return overlayBox(c.title, lines, width, c.danger)
}

type menuOverlay struct {
	title   string
	target  string
	actions []*Action
	cursor  int
}

func (m *menuOverlay) handleKey(a *App, msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	if key == "esc" {
		a.overlay = nil
		return nil
	}
	if key == "enter" && len(m.actions) > 0 {
		a.overlay = nil
		return a.invoke(m.actions[m.cursor])
	}
	m.cursor, _ = moveCursor(key, m.cursor, len(m.actions), listOverlayRows)
	return nil
}

func (m *menuOverlay) view(a *App, width, height int) string {
	var lines []string
	if m.target != "" {
		lines = append(lines, styleDirty.Render(m.target), "")
	}
	if len(m.actions) == 0 {
		lines = append(lines, styleMuted.Render("No actions for this item"))
	}
	for index, action := range m.actions {
		row := fmt.Sprintf("%-34s %s", action.Title, keyLabel(action))
		reason := a.unavailableReason(action)
		if reason != "" {
			row = styleMuted.Render(row)
		}
		if index == m.cursor {
			row = styleFocus.Render("> " + ansi.Strip(row))
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
		if reason != "" {
			lines = append(lines, styleMuted.Render("    unavailable: "+reason))
		}
	}
	lines = append(lines, "", styleMuted.Render("j/k Move · Enter Run · Esc Close"))
	return overlayBox(m.title, lines, width, false)
}

type paletteOverlay struct {
	input   textinput.Model
	actions []*Action
	cursor  int
}

func (p *paletteOverlay) matching() []*Action {
	query := strings.ToLower(p.input.Value())
	var matches []*Action
	for _, action := range p.actions {
		title := strings.ToLower(action.Title)
		position := 0
		for _, character := range query {
			found := strings.IndexRune(title[position:], character)
			if found < 0 {
				position = -1
				break
			}
			position += found + 1
		}
		if position >= 0 {
			matches = append(matches, action)
		}
	}
	return matches
}

func (p *paletteOverlay) handleKey(a *App, msg tea.KeyMsg) tea.Cmd {
	matches := p.matching()
	switch msg.String() {
	case "esc":
		a.overlay = nil
		return nil
	case "enter":
		if len(matches) == 0 {
			return nil
		}
		a.overlay = nil
		return a.invoke(matches[min(p.cursor, len(matches)-1)])
	case "up", "ctrl+p":
		p.cursor = max(p.cursor-1, 0)
		return nil
	case "down", "ctrl+n":
		p.cursor = min(p.cursor+1, max(len(matches)-1, 0))
		return nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.cursor = 0
	return cmd
}

func (p *paletteOverlay) view(a *App, width, height int) string {
	lines := []string{p.input.View(), ""}
	matches := p.matching()
	for index, action := range matches {
		if index >= listOverlayRows {
			lines = append(lines, styleMuted.Render(fmt.Sprintf("… %d more", len(matches)-listOverlayRows)))
			break
		}
		row := fmt.Sprintf("%-34s %s", action.Title, keyLabel(action))
		if reason := a.unavailableReason(action); reason != "" {
			row = styleMuted.Render(row + "  (unavailable: " + reason + ")")
		}
		if index == p.cursor {
			row = styleFocus.Render(row)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", styleMuted.Render("↑/↓ Move · Enter Run · Esc Close · typed keys never run commands"))
	return overlayBox("Command palette", lines, width, false)
}

var navigationHelp = map[Scope][]string{
	ScopeGlobal:     {"Enter  Open focused item", "Space  Toggle selection (UI only, never stages)", "Esc  Close overlay / back (never changes Git)", "j/k ↑/↓  Move", "Tab / Shift+Tab  Next / previous panel", "gg / G  First / last loaded row", "Ctrl+D / Ctrl+U  Half page", "a  Actions for selection or focused item", ":  Command palette", "?  This help", "q  Back, or Quit control at the root"},
	ScopeChanges:    {"1 2 3 4  All / Staged / Not staged / Mixed", "S  Select all visible files, or clear them if all selected (UI only)", "/  Filter paths (Enter keep, Esc clear)"},
	ScopeHistory:    {"/  Filter loaded commits (Enter keep, Esc clear)"},
	ScopeConflict:   {"n / N  Next / previous conflict region"},
	ScopeComposer:   {"Tab  Next field or button", "Esc  Leave text field, then leave composer"},
	ScopeDiff:       {"Tab  Switch staged / unstaged section"},
	ScopeCherryPick: {"Space  Include / exclude queue row", "Enter  Inspect commit"},
}

type helpOverlay struct {
	scope Scope
}

func (h *helpOverlay) handleKey(a *App, msg tea.KeyMsg) tea.Cmd {
	if key := msg.String(); key == "esc" || key == "?" || key == "q" {
		a.overlay = nil
	}
	return nil
}

func (h *helpOverlay) view(a *App, width, height int) string {
	lines := append([]string{}, navigationHelp[ScopeGlobal]...)
	lines = append(lines, navigationHelp[h.scope]...)
	lines = append(lines, "", styleBold.Render("Commands in "+string(h.scope)))
	for _, action := range a.registry.InScope(h.scope, ScopeGlobal) {
		keys := keyLabel(action)
		if keys == "" {
			keys = "a / :"
		}
		line := fmt.Sprintf("%-10s %s", keys, action.Title)
		if reason := a.unavailableReason(action); reason != "" {
			line = styleMuted.Render(line + "  (unavailable: " + reason + ")")
		}
		lines = append(lines, line)
	}
	if len(lines) > height-overlayPadding {
		lines = lines[:max(height-overlayPadding, 1)]
	}
	return overlayBox("Shortcuts", lines, width, false)
}

type promptOverlay struct {
	title    string
	hint     string
	input    textinput.Model
	onSubmit func(a *App, value string) tea.Cmd
}

func newPrompt(title, hint, initial string, onSubmit func(a *App, value string) tea.Cmd) *promptOverlay {
	input := newTextInput()
	input.SetValue(initial)
	input.Prompt = "> "
	input.Width = textPromptWidth
	input.Focus()
	return &promptOverlay{title: title, hint: hint, input: input, onSubmit: onSubmit}
}

func (p *promptOverlay) handleKey(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		a.overlay = nil
		return nil
	case "enter":
		a.overlay = nil
		return p.onSubmit(a, strings.TrimSpace(p.input.Value()))
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd
}

func (p *promptOverlay) view(a *App, width, height int) string {
	return overlayBox(p.title, []string{p.input.View(), "", styleMuted.Render(p.hint), styleMuted.Render("Enter confirm · Esc cancel")}, width, false)
}
