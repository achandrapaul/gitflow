package app

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
)

const defaultRemoteName = "origin"

type branchScreen struct {
	branches []git.Branch
	remotes  []string
	loaded   bool
	err      error
	cursor   int
	offset   int
}

func newBranchScreen() *branchScreen { return &branchScreen{} }

func (b *branchScreen) scope() Scope { return ScopeBranch }

func (b *branchScreen) opened(a *App) tea.Cmd {
	runner := a.runner
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		branches, err := git.ListBranches(ctx, runner)
		remotes, remotesErr := git.ListRemotes(ctx, runner)
		if err == nil {
			err = remotesErr
		}
		return func(a *App) tea.Cmd {
			b.branches, b.remotes, b.err, b.loaded = branches, remotes, err, true
			b.cursor = min(b.cursor, max(len(branches)-1, 0))
			return nil
		}
	})
}

func (b *branchScreen) statusChanged(a *App) tea.Cmd { return b.opened(a) }

func (b *branchScreen) focused() (git.Branch, bool) {
	if b.cursor < 0 || b.cursor >= len(b.branches) {
		return git.Branch{}, false
	}
	return b.branches[b.cursor], true
}

func (b *branchScreen) defaultRemote() string {
	for _, remote := range b.remotes {
		if remote == defaultRemoteName {
			return remote
		}
	}
	if len(b.remotes) > 0 {
		return b.remotes[0]
	}
	return ""
}

func (b *branchScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	if next, moved := moveCursor(key, b.cursor, len(b.branches), a.height/2); moved {
		b.cursor = next
		return nil, true
	}
	if key == "enter" {
		branch, ok := b.focused()
		if !ok {
			return nil, true
		}
		history := newHistoryScreen(branch.Name)
		a.push(history)
		return history.opened(a), true
	}
	return nil, false
}

func (b *branchScreen) view(a *App, width, height int) string {
	branch := a.status.Branch
	name := branch.Head
	if branch.Detached {
		name = styleWarning.Render("DETACHED HEAD (no branch)")
	}
	upstream := styleMuted.Render("No upstream (ahead/behind unknown)")
	switch {
	case branch.Upstream != "" && branch.HasAheadBehind:
		upstream = fmt.Sprintf("%s · %d ahead, %d behind", branch.Upstream, branch.Ahead, branch.Behind)
	case branch.Upstream != "":
		upstream = branch.Upstream + styleWarning.Render(" (configured, but the remote-tracking ref is missing locally)")
	}
	base := styleMuted.Render("Base unavailable (no origin/main or main)")
	switch {
	case a.base.Available && a.base.CountsKnown:
		base = fmt.Sprintf("%s · %d ahead, %d behind", a.base.ShortRef(), a.base.Ahead, a.base.Behind)
	case a.base.Available:
		base = a.base.ShortRef() + styleMuted.Render(" (counts unknown)")
	}
	operation := "none"
	if a.operation.Active() {
		operation = styleConflict.Render(a.operation.Kind.Name() + " paused (o opens dashboard)")
	}
	body := []string{
		"Branch: " + name,
		"HEAD: " + branch.OID + " (" + shortOID(branch.OID) + ")",
		"Upstream: " + upstream,
		"Comparison base (independent of upstream): " + base,
		styleMuted.Render("Remote-tracking refs are local: fetch time unknown. Nothing is fetched automatically."),
		fmt.Sprintf("Working tree: %d changed path(s)", len(a.status.Files)),
		"Operation: " + operation,
		"",
		styleBold.Render("LOCAL BRANCHES") + styleMuted.Render("  (Enter opens history · a Actions)"),
	}
	if b.err != nil {
		body = append(body, styleError.Render(b.err.Error()))
	}
	var rows []string
	for _, local := range b.branches {
		marker := "  "
		if local.Current {
			marker = styleStaged.Render("* ")
		}
		upstreamLabel := styleMuted.Render("no upstream")
		if local.Upstream != "" {
			upstreamLabel = "→ " + local.Upstream
		}
		rows = append(rows, fmt.Sprintf("%s%-40s %s %s", marker, local.Name, shortOID(local.OID), upstreamLabel))
	}
	rendered, offset := renderRows(rows, b.cursor, b.offset, height-panelBorderSize-len(body)-1, true)
	b.offset = offset
	body = append(body, rendered...)
	return panel("BRANCH INSPECTOR", body, width, height, true)
}

func (b *branchScreen) footer() string {
	return "j/k Move  Enter History of branch  a Actions (fetch/pull/push/upstream/create/switch/delete/base)  r i Rebase onto base  Esc Back"
}

func (b *branchScreen) menuTitle(a *App) string {
	if branch, ok := b.focused(); ok {
		return "Actions for branch " + branch.Name
	}
	return "Branch actions"
}
