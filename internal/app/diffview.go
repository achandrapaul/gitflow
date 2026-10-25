package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

type diffPane int

const (
	paneStaged diffPane = iota
	paneUnstaged
)

const hunkHeaderRow = -1

type diffRow struct {
	hunk int
	line int
}

type diffScreen struct {
	file       git.FileChange
	preview    filePreview
	loaded     bool
	pane       diffPane
	cursors    [2]int
	offsets    [2]int
	selections [2]git.LineSelection
}

func newDiffScreen(file git.FileChange) *diffScreen {
	pane := paneStaged
	if file.Unstaged() {
		pane = paneUnstaged
	}
	return &diffScreen{file: file, pane: pane, selections: [2]git.LineSelection{{}, {}}}
}

func (d *diffScreen) scope() Scope { return ScopeDiff }

func (d *diffScreen) opened(a *App) tea.Cmd {
	file, runner := d.file, a.runner
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		preview := loadPreview(ctx, runner, file)
		return func(a *App) tea.Cmd {
			if file.Signature() == d.file.Signature() {
				d.preview, d.loaded = preview, true
			}
			return nil
		}
	})
}

func (d *diffScreen) statusChanged(a *App) tea.Cmd {
	for _, file := range a.status.Files {
		if file.Path == d.file.Path {
			if file.Signature() != d.file.Signature() {
				d.selections = [2]git.LineSelection{{}, {}}
			}
			d.file = file
			return d.opened(a)
		}
	}
	d.file.Index, d.file.Worktree = '.', '.'
	d.preview, d.loaded = filePreview{}, true
	a.notice = displayPath(d.file.Path) + " no longer has changes"
	return nil
}

func (d *diffScreen) diff(pane diffPane) git.FileDiff {
	if pane == paneStaged {
		return d.preview.staged
	}
	return d.preview.unstaged
}

func (d *diffScreen) rows(pane diffPane) []diffRow {
	var rows []diffRow
	for hunkIndex, hunk := range d.diff(pane).Hunks {
		rows = append(rows, diffRow{hunk: hunkIndex, line: hunkHeaderRow})
		for lineIndex := range hunk.Lines {
			rows = append(rows, diffRow{hunk: hunkIndex, line: lineIndex})
		}
	}
	return rows
}

func (d *diffScreen) hunkSelection(pane diffPane, hunkIndex int) git.LineSelection {
	selection := git.LineSelection{}
	for lineIndex, line := range d.diff(pane).Hunks[hunkIndex].Lines {
		if line.Kind == '+' || line.Kind == '-' {
			selection[git.LineRef{Hunk: hunkIndex, Line: lineIndex}] = true
		}
	}
	return selection
}

func (d *diffScreen) patchSelection(pane diffPane) (git.LineSelection, string) {
	if len(d.selections[pane]) > 0 {
		return d.selections[pane], fmt.Sprintf("%d selected line(s)", len(d.selections[pane]))
	}
	rows := d.rows(pane)
	if d.cursors[pane] >= len(rows) {
		return nil, ""
	}
	return d.hunkSelection(pane, rows[d.cursors[pane]].hunk), "focused hunk"
}

func (d *diffScreen) targetScope(a *App) string {
	diff := d.diff(d.pane)
	if reason := diff.PartialUnsupportedReason(); reason != "" || d.file.Kind == git.KindUntracked || len(diff.Hunks) == 0 {
		return "Target: whole file " + displayPath(d.file.Path)
	}
	_, description := d.patchSelection(d.pane)
	return "Target: " + description + " in " + [...]string{"STAGED", "UNSTAGED"}[d.pane]
}

func (d *diffScreen) menuTitle(a *App) string { return "Actions for " + displayPath(d.file.Path) }

func (d *diffScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "tab", "shift+tab":
		d.pane = 1 - d.pane
		return nil, true
	case "enter":
		a.notice = "Enter never stages: use s / u"
		return nil, true
	}
	rows := d.rows(d.pane)
	if next, moved := moveCursor(key, d.cursors[d.pane], len(rows), a.height/2); moved {
		d.cursors[d.pane] = next
		return nil, true
	}
	if key != " " {
		return nil, false
	}
	if len(rows) == 0 {
		a.notice = "Nothing selectable here"
		return nil, true
	}
	if reason := d.diff(d.pane).PartialUnsupportedReason(); reason != "" {
		a.notice = "Line selection unavailable: " + reason
		return nil, true
	}
	row := rows[d.cursors[d.pane]]
	selection := d.selections[d.pane]
	if row.line == hunkHeaderRow {
		hunk := d.hunkSelection(d.pane, row.hunk)
		allSelected := true
		for ref := range hunk {
			allSelected = allSelected && selection[ref]
		}
		for ref := range hunk {
			if allSelected {
				delete(selection, ref)
			} else {
				selection[ref] = true
			}
		}
		return nil, true
	}
	line := d.diff(d.pane).Hunks[row.hunk].Lines[row.line]
	if line.Kind != '+' && line.Kind != '-' {
		a.notice = "Context lines are not selectable"
		return nil, true
	}
	ref := git.LineRef{Hunk: row.hunk, Line: row.line}
	if selection[ref] {
		delete(selection, ref)
	} else {
		selection[ref] = true
	}
	return nil, true
}

func (d *diffScreen) paneView(pane diffPane, width, height int) string {
	title := [...]string{"STAGED · HEAD → INDEX", "UNSTAGED · INDEX → WORKTREE"}[pane]
	diff := d.diff(pane)
	var body []string
	switch {
	case !d.loaded:
		body = append(body, styleMuted.Render("Loading…"))
	case d.preview.err != nil:
		body = append(body, styleError.Render(d.preview.err.Error()))
	case pane == paneUnstaged && d.file.Kind == git.KindUntracked:
		body = append(body, styleDirty.Render("Untracked file: s stages the whole file"))
		for _, line := range strings.Split(d.preview.untracked, "\n") {
			body = append(body, styleDirty.Render("+"+line))
		}
	case diff.Empty():
		body = append(body, styleMuted.Render("No changes in this section"))
	}
	if reason := diff.PartialUnsupportedReason(); reason != "" {
		body = append(body, styleMuted.Render(reason))
	}
	var rows []string
	for _, row := range d.rows(pane) {
		hunk := diff.Hunks[row.hunk]
		if row.line == hunkHeaderRow {
			mark := "   "
			if len(d.selections[pane]) > 0 {
				allSelected := true
				for ref := range d.hunkSelection(pane, row.hunk) {
					allSelected = allSelected && d.selections[pane][ref]
				}
				if allSelected {
					mark = styleAccent.Render("[x]")
				}
			}
			rows = append(rows, mark+" "+styleAccent.Render(fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount, hunk.Section)))
			continue
		}
		line := hunk.Lines[row.line]
		mark := "   "
		if d.selections[pane][git.LineRef{Hunk: row.hunk, Line: row.line}] {
			mark = styleAccent.Render("[x]")
		}
		text := string(line.Kind) + line.Text
		switch line.Kind {
		case '+':
			text = styleStaged.Render(text)
		case '-':
			text = styleError.Render(text)
		}
		rows = append(rows, mark+" "+text)
	}
	rendered, offset := renderRows(rows, d.cursors[pane], d.offsets[pane], height-panelBorderSize-len(body)-1, d.pane == pane)
	d.offsets[pane] = offset
	body = append(body, rendered...)
	if count := len(d.selections[pane]); count > 0 {
		body = append(body, styleAccent.Render(fmt.Sprintf("%d line(s) selected", count)))
	}
	return panel(title, body, width, height, d.pane == pane)
}

func (d *diffScreen) view(a *App, width, height int) string {
	state := fmt.Sprintf("%s · INDEX %s · WORKTREE %s", displayPath(d.file.Path), statusCode(d.file.Index, false), statusCode(d.file.Worktree, d.file.Kind == git.KindUntracked))
	if d.file.Conflicted() {
		state += " · " + styleConflict.Render("UNMERGED: resolve from operation dashboard (o)")
	}
	paneHeight := (height - 1) / 2
	return lipgloss.JoinVertical(lipgloss.Left, styleBold.Render(clip(state, width)),
		d.paneView(paneStaged, width, paneHeight), d.paneView(paneUnstaged, width, height-1-paneHeight))
}

func (d *diffScreen) footer() string {
	return "j/k Move  Tab Section  Space Select hunk/line  s Stage  u Unstage  a Actions  Esc Back"
}
