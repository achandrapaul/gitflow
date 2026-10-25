package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	return "j/k Move  Enter History of branch  a Actions (fetch/pull/push/upstream/create/switch/delete/base)  g f/g l/g p Fetch/Pull/Push review  r i Rebase onto base  Esc Back"
}

func (b *branchScreen) menuActions(a *App) []*Action {
	actions := []*Action{a.registry.ByID("sync.fetch"), a.registry.ByID("sync.pull"), a.registry.ByID("sync.push")}
	return append(actions, a.registry.InScope(ScopeBranch)...)
}

func (b *branchScreen) menuTitle(a *App) string {
	if branch, ok := b.focused(); ok {
		return "Actions for branch " + branch.Name
	}
	return "Branch actions"
}

type syncControl int

const (
	syncReview syncControl = iota
	syncExecute
	syncCancel
	syncControlCount
)

type syncScreen struct {
	kind     git.SyncKind
	upstream git.Upstream
	loaded   bool
	err      error
	focus    syncControl
	reviewed bool
	result   []string
}

func (s *syncScreen) scope() Scope { return ScopeSync }

func (s *syncScreen) opened(a *App) tea.Cmd {
	runner, branch := a.runner, a.status.Branch
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		var upstream git.Upstream
		var err error
		if !branch.Detached && branch.Head != "" {
			upstream, err = git.LoadUpstream(ctx, runner, branch.Head)
		}
		return func(a *App) tea.Cmd {
			s.upstream, s.err, s.loaded = upstream, err, true
			return nil
		}
	})
}

func (s *syncScreen) statusChanged(a *App) tea.Cmd { return s.opened(a) }

func (s *syncScreen) executeUnavailable(a *App) string {
	switch {
	case !s.loaded:
		return "loading upstream"
	case s.err != nil:
		return s.err.Error()
	case !s.reviewed:
		return "choose Review first"
	case s.kind != git.SyncFetch && a.status.Branch.Detached:
		return "detached HEAD: no current branch"
	}
	if _, err := git.SyncInvocation(s.kind, s.upstream); err != nil {
		if errors.Is(err, git.ErrNoUpstream) {
			return "No upstream: use Set upstream in the branch inspector first"
		}
		return err.Error()
	}
	if s.kind == git.SyncPull && a.operation.Active() {
		return "a " + a.operation.Kind.Name() + " is in progress"
	}
	return ""
}

func (s *syncScreen) execute(a *App) tea.Cmd {
	invocation, err := git.SyncInvocation(s.kind, s.upstream)
	if err != nil {
		a.errorText = err.Error()
		return nil
	}
	s.result = []string{"Running git " + strings.Join(invocation.Args, " ") + "…"}
	return a.gitHandoff(s.kind.Name(), invocation, func(a *App, err error) tea.Cmd {
		if err == nil {
			s.result = []string{"Done: Git reported success. Refs and status were re-read."}
			return s.opened(a)
		}
		hint := map[git.SyncKind]string{
			git.SyncFetch: "Check the remote name, network and credentials; nothing local changed.",
			git.SyncPull:  "Pull is fast-forward-only: if histories diverged, integrate them yourself (merge or rebase). Nothing was merged or rebased.",
			git.SyncPush:  "If the remote has commits you do not have, fetch and integrate them first. GitFlow never force-pushes.",
		}[s.kind]
		s.result = []string{"Failed: Git's error is shown above.", hint}
		return s.opened(a)
	})
}

func (s *syncScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "tab":
		s.focus = (s.focus + 1) % syncControlCount
		return nil, true
	case "shift+tab":
		s.focus = (s.focus + syncControlCount - 1) % syncControlCount
		return nil, true
	case "enter":
		switch s.focus {
		case syncReview:
			s.reviewed = true
			s.result = []string{"Reviewed: local refs re-read; no command executed."}
			return tea.Batch(a.refresh(), s.opened(a)), true
		case syncExecute:
			return a.invoke(a.registry.ByID("sync.execute")), true
		}
		a.back()
		return nil, true
	}
	return nil, false
}

func (s *syncScreen) view(a *App, width, height int) string {
	branch := a.status.Branch
	action := map[git.SyncKind]string{git.SyncFetch: "Fetch", git.SyncPull: "Pull (fast-forward-only)", git.SyncPush: "Push (ordinary, never forced)"}[s.kind]
	localBranch := branch.Head
	if branch.Detached {
		localBranch = styleWarning.Render("DETACHED @ " + shortOID(branch.OID))
	}
	remote := s.upstream.Remote
	switch {
	case !s.loaded:
		remote = styleMuted.Render("loading…")
	case remote == "" && s.kind == git.SyncFetch:
		remote = "Git's default remote (no upstream configured)"
	case remote == "":
		remote = styleMuted.Render("none")
	}
	upstream := branch.Upstream
	if upstream == "" {
		upstream = styleWarning.Render("No upstream: Pull and Push stay disabled until you use Set upstream in the branch inspector")
	}
	tracking := styleMuted.Render("unknown (no upstream)")
	if branch.HasAheadBehind {
		tracking = fmt.Sprintf("↑%d ↓%d (from local refs; fetch freshness unknown)", branch.Ahead, branch.Behind)
	}
	base := "Base unavailable (independent metric, not the pull/push target)"
	if a.base.Available && a.base.CountsKnown {
		base = fmt.Sprintf("%s ↑%d ↓%d (independent metric, not the pull/push target)", a.base.ShortRef(), a.base.Ahead, a.base.Behind)
	}
	command := ""
	if invocation, err := git.SyncInvocation(s.kind, s.upstream); err != nil {
		command = styleMuted.Render("unavailable: " + err.Error())
	} else {
		command = "git " + strings.Join(invocation.Args, " ")
	}
	effect := map[git.SyncKind]string{
		git.SyncFetch: "Updates remote-tracking refs only; branches, index and working tree are unchanged.",
		git.SyncPull:  "Fast-forwards " + branch.Head + " and the working tree to the upstream; stops if histories diverged (no merge, no rebase).",
		git.SyncPush:  "Updates the upstream branch only if it fast-forwards; otherwise Git rejects it and nothing changes.",
	}[s.kind]
	result := s.result
	if result == nil {
		result = []string{"Pending; no command executed"}
	}
	body := []string{
		"Action: " + action,
		"Local branch: " + localBranch,
		"Remote: " + remote,
		"Upstream: " + upstream,
		"Tracking: " + tracking,
		"Compared base: " + base,
		"Command: " + command,
		"What might change: " + effect,
		"Result: " + result[0],
	}
	body = append(body, result[1:]...)
	if reason := s.executeUnavailable(a); reason != "" {
		body = append(body, styleMuted.Render("Execute unavailable: "+reason))
	}
	var buttons []string
	for control, label := range []string{"[ Review ]", "[ Execute ]", "[ Cancel ]"} {
		if syncControl(control) == s.focus {
			label = styleFocus.Render(label)
		}
		buttons = append(buttons, label)
	}
	body = append(body, "", strings.Join(buttons, "  "))
	return panel("REMOTE SYNC REVIEW · "+strings.ToUpper(s.kind.Name()), body, width, height, true)
}

func (s *syncScreen) footer() string {
	return "Tab Move  Enter Activate (Review never runs Git)  Esc Back · g l Pull · g p Push · g f Fetch"
}

func (s *syncScreen) menuTitle(a *App) string { return "Actions for " + s.kind.Name() + " review" }
