package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

const plannerQueueShare = 55

func (a *App) headLabel() string {
	if a.status.Branch.Detached {
		return "DETACHED @ " + shortOID(a.status.Branch.OID)
	}
	return a.status.Branch.Head + " @ " + shortOID(a.status.Branch.OID)
}

func (a *App) discardPlanDialog() {
	a.overlay = &confirmOverlay{
		title:   "Discard this unexecuted plan?",
		lines:   []string{"Only the plan in GitFlow is discarded. Git is not changed."},
		buttons: []confirmButton{{label: "Cancel"}, {label: "Discard plan", run: func(a *App) tea.Cmd { a.pop(); return nil }}},
	}
}

func (a *App) routeOperation(planner screen, onFinished func(a *App) tea.Cmd) tea.Cmd {
	runner, gitDir := a.runner, a.repo.GitDir
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		operation, err := git.DetectOperation(ctx, runner, gitDir)
		return func(a *App) tea.Cmd {
			if err != nil {
				a.errorText = "Check operation state: " + err.Error()
				return nil
			}
			a.operation, a.operationLoaded = operation, true
			if !operation.Active() {
				if onFinished != nil {
					return onFinished(a)
				}
				a.popTo(1)
				return nil
			}
			for index, stacked := range a.stack {
				if stacked == planner {
					a.popTo(index)
				}
			}
			if _, already := a.top().(*operationScreen); !already {
				a.push(&operationScreen{})
			}
			a.notice = "Git paused the " + operation.Kind.Name() + ": see operation dashboard"
			return nil
		}
	})
}

type queueItem struct {
	commit   git.Commit
	included bool
}

type cherryPickScreen struct {
	oids        []string
	queue       []queueItem
	loading     bool
	err         error
	target      string
	headOID     string
	cursor      int
	offset      int
	dirty       bool
	invalidated string
	completed   []string
	details     map[string]string
}

func (c *cherryPickScreen) scope() Scope { return ScopeCherryPick }

func (c *cherryPickScreen) opened(a *App) tea.Cmd {
	runner, oids := a.runner, c.oids
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		commits, err := git.LoadCommits(ctx, runner, oids)
		return func(a *App) tea.Cmd {
			c.loading, c.err = false, err
			for _, commit := range commits {
				c.queue = append(c.queue, queueItem{commit: commit, included: true})
			}
			return c.requestDetail(a)
		}
	})
}

func (c *cherryPickScreen) requestDetail(a *App) tea.Cmd {
	if c.cursor >= len(c.queue) {
		return nil
	}
	oid := c.queue[c.cursor].commit.OID
	if _, cached := c.details[oid]; cached {
		return nil
	}
	c.details[oid] = "Loading…"
	runner := a.runner
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		text, err := git.ChangedFiles(ctx, runner, oid)
		return func(a *App) tea.Cmd {
			if err != nil {
				text = err.Error()
			}
			c.details[oid] = text
			return nil
		}
	})
}

func (c *cherryPickScreen) included() []git.Commit {
	var commits []git.Commit
	for _, item := range c.queue {
		if item.included {
			commits = append(commits, item.commit)
		}
	}
	return commits
}

func (c *cherryPickScreen) move(delta int) tea.Cmd {
	next := c.cursor + delta
	if next < 0 || next >= len(c.queue) || c.completed != nil {
		return nil
	}
	c.queue[c.cursor], c.queue[next] = c.queue[next], c.queue[c.cursor]
	c.cursor, c.dirty = next, true
	return nil
}

func (c *cherryPickScreen) executeUnavailable(a *App) string {
	included := c.included()
	switch {
	case c.loading:
		return "plan is loading"
	case c.err != nil:
		return c.err.Error()
	case c.completed != nil:
		return "plan already executed"
	case c.invalidated != "":
		return c.invalidated
	case len(included) == 0:
		return "no commits included"
	case a.operation.Active():
		return "a " + a.operation.Kind.Name() + " is already in progress"
	case a.status.HasConflicts():
		return "unmerged files present"
	case a.status.Branch.OID != c.headOID:
		return "HEAD moved since the plan was created: plan is stale"
	}
	for _, commit := range included {
		if commit.IsMerge() {
			return "merge commit " + commit.Short() + " needs a mainline parent (not supported, see OPEN_QUESTIONS Q10)"
		}
	}
	return ""
}

func (c *cherryPickScreen) execute(a *App) tea.Cmd {
	included := c.included()
	invocation, err := git.CherryPickInvocation(included)
	if err != nil {
		a.errorText = err.Error()
		return nil
	}
	lines := []string{"Target: " + c.target + " (checked out)", "Order (applied top to bottom):"}
	for index, commit := range included {
		lines = append(lines, fmt.Sprintf("  %d. %s %s", index+1, commit.Short(), commit.Subject))
	}
	lines = append(lines, "", "Git applies these one at a time; this is not an atomic transaction.", "On a conflict or empty commit Git pauses and the operation dashboard opens.")
	a.overlay = &confirmOverlay{
		title:   fmt.Sprintf("Execute cherry-pick of %d commit(s)?", len(included)),
		lines:   lines,
		command: "git cherry-pick " + strings.Join(invocation.Args[1:], " "),
		buttons: []confirmButton{{label: "Cancel"}, {label: "Execute plan", run: func(a *App) tea.Cmd {
			status, err := git.LoadStatus(a.ctx, a.runner)
			if err != nil {
				a.errorText = "Revalidate before cherry-pick: " + err.Error()
				return nil
			}
			a.status = status
			if status.Branch.OID != c.headOID {
				c.invalidated = "HEAD moved since the plan was created: plan invalidated, reopen the planner"
				a.errorText = c.invalidated
				return a.refresh()
			}
			previousHead := c.headOID
			return a.gitHandoff("Cherry-pick", invocation, func(a *App, err error) tea.Cmd {
				return a.routeOperation(c, func(a *App) tea.Cmd {
					if err != nil {
						return nil
					}
					runner := a.runner
					return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
						applied, rangeErr := git.ListRange(ctx, runner, previousHead, "HEAD")
						return func(a *App) tea.Cmd {
							c.completed = []string{"Cherry-pick finished. Git reports no active operation."}
							switch {
							case rangeErr != nil:
								c.completed = append(c.completed, "Result mapping unknown: "+rangeErr.Error())
							case len(applied) == len(included):
								for index, commit := range applied {
									c.completed = append(c.completed, fmt.Sprintf("%s → %s %s", included[index].Short(), commit.Short(), commit.Subject))
								}
							default:
								c.completed = append(c.completed, fmt.Sprintf("%d new commit(s) on HEAD; mapping to the %d source commit(s) is unknown", len(applied), len(included)))
							}
							return nil
						}
					})
				})
			})
		}}},
	}
	return nil
}

func (c *cherryPickScreen) confirmLeave(a *App) bool {
	if c.dirty && c.completed == nil {
		a.discardPlanDialog()
		return false
	}
	return true
}

func (c *cherryPickScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	if next, moved := moveCursor(key, c.cursor, len(c.queue), a.height/2); moved {
		c.cursor = next
		return c.requestDetail(a), true
	}
	if c.cursor >= len(c.queue) {
		return nil, false
	}
	switch key {
	case " ":
		if c.completed == nil {
			c.queue[c.cursor].included = !c.queue[c.cursor].included
			c.dirty = true
		}
		return nil, true
	case "enter":
		commit, runner := c.queue[c.cursor].commit, a.runner
		return a.openViewer("commit "+commit.Short(), func(ctx context.Context) (string, error) {
			return git.ShowCommit(ctx, runner, commit.OID)
		}), true
	}
	return nil, false
}

func (c *cherryPickScreen) preview(a *App) tea.Cmd {
	included, runner, target := c.included(), a.runner, c.target
	return a.openViewer("Cherry-pick preview", func(ctx context.Context) (string, error) {
		var text strings.Builder
		fmt.Fprintf(&text, "Target: %s\nQueue (%d commit(s), applied in this order):\n", target, len(included))
		for index, commit := range included {
			fmt.Fprintf(&text, "\n%d. %s %s\n", index+1, commit.Short(), commit.Subject)
			if commit.IsMerge() {
				text.WriteString("   ! merge commit: needs an explicit mainline parent; execution is blocked\n")
				continue
			}
			duplicate, err := git.PatchAlreadyInHead(ctx, runner, commit)
			if err != nil {
				fmt.Fprintf(&text, "   ! duplicate check failed: %v\n", err)
			} else if duplicate {
				text.WriteString("   ! an equivalent patch already exists on the target; Git may stop with an empty commit\n")
			}
			files, err := git.ChangedFiles(ctx, runner, commit.OID)
			if err != nil {
				return "", err
			}
			text.WriteString(files)
		}
		text.WriteString("\nConflicts cannot be predicted reliably: Git decides when it applies each commit.\n")
		return text.String(), nil
	})
}

func (c *cherryPickScreen) view(a *App, width, height int) string {
	header := []string{
		fmt.Sprintf("Source: %d explicitly selected commit(s) (full OIDs; may span branches)", len(c.oids)),
		"Target: " + c.target + " (currently checked out)",
	}
	if reason := c.executeUnavailable(a); reason != "" && c.completed == nil {
		header = append(header, styleMuted.Render("Execute unavailable: "+reason))
	}
	if c.err != nil {
		header = append(header, styleError.Render(c.err.Error()))
	}
	var rows []string
	for index, item := range c.queue {
		mark := styleStaged.Render("[✓]")
		if !item.included {
			mark = styleMuted.Render("[ ]")
		}
		row := fmt.Sprintf("%d. %s %s %s", index+1, mark, styleDirty.Render(item.commit.Short()), item.commit.Subject)
		if item.commit.IsMerge() {
			row += " " + styleWarning.Render("(merge: needs mainline)")
		}
		rows = append(rows, row)
	}
	queueWidth := width * plannerQueueShare / percent
	rendered, offset := renderRows(rows, c.cursor, c.offset, height-panelBorderSize-len(header)-2, true)
	c.offset = offset
	body := append(header, "")
	body = append(body, rendered...)
	if c.completed != nil {
		body = append(body, "")
		body = append(body, c.completed...)
	}
	queuePanel := panel("CHERRY-PICK PLAN", body, queueWidth, height, true)
	var detail []string
	if c.cursor < len(c.queue) {
		detail = strings.Split(c.details[c.queue[c.cursor].commit.OID], "\n")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, queuePanel, panel("CHANGES IN FOCUSED COMMIT", detail, width-queueWidth, height, false))
}

func (c *cherryPickScreen) footer() string {
	return "j/k Move  Shift+J/K Reorder  Space Include  Enter Inspect  p Preview  a Actions (Execute plan)  Esc Back"
}

func (c *cherryPickScreen) menuTitle(a *App) string {
	return fmt.Sprintf("Actions for cherry-pick plan (%d commit(s))", len(c.included()))
}

type rebaseScreen struct {
	onto          string
	ontoLabel     string
	rows          []git.TodoRow
	loading       bool
	err           error
	branch        string
	headOID       string
	cursor        int
	offset        int
	dirty         bool
	publishedRefs []string
	invalidated   string
	completed     []string
}

func newRebaseScreen(a *App, onto, ontoLabel string) *rebaseScreen {
	return &rebaseScreen{onto: onto, ontoLabel: ontoLabel, loading: true, branch: a.headLabel(), headOID: a.status.Branch.OID}
}

func (r *rebaseScreen) scope() Scope { return ScopeRebase }

func (r *rebaseScreen) opened(a *App) tea.Cmd {
	runner, onto := a.runner, r.onto
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		commits, err := git.RebaseRange(ctx, runner, onto)
		var published []string
		if err == nil && len(commits) > 0 {
			published, err = git.RemoteRefsContaining(ctx, runner, commits[0].OID)
		}
		return func(a *App) tea.Cmd {
			r.loading, r.err, r.publishedRefs = false, err, published
			for _, commit := range commits {
				r.rows = append(r.rows, git.TodoRow{Action: git.TodoPick, Commit: commit})
			}
			return nil
		}
	})
}

func (r *rebaseScreen) setAction(action git.TodoAction) tea.Cmd {
	if r.cursor < len(r.rows) && r.completed == nil {
		r.rows[r.cursor].Action = action
		r.dirty = true
	}
	return nil
}

func (r *rebaseScreen) move(delta int) tea.Cmd {
	next := r.cursor + delta
	if next < 0 || next >= len(r.rows) || r.completed != nil {
		return nil
	}
	r.rows[r.cursor], r.rows[next] = r.rows[next], r.rows[r.cursor]
	r.cursor, r.dirty = next, true
	return nil
}

func (r *rebaseScreen) executeUnavailable(a *App) string {
	switch {
	case r.loading:
		return "range is loading"
	case r.err != nil:
		return r.err.Error()
	case r.completed != nil:
		return "plan already executed"
	case r.invalidated != "":
		return r.invalidated
	case a.operation.Active():
		return "a " + a.operation.Kind.Name() + " is already in progress"
	case a.status.HasConflicts():
		return "unmerged files present"
	case a.status.HasTrackedChanges():
		return "tracked changes present: commit or stash them first"
	case a.status.Branch.OID != r.headOID:
		return "HEAD moved since the plan was created: plan is stale"
	}
	if err := git.ValidateTodo(r.rows); err != nil {
		if errors.Is(err, git.ErrSquashWithoutPick) {
			return "first commit cannot be squash/fixup: it needs a preceding picked commit"
		}
		return err.Error()
	}
	return ""
}

func (r *rebaseScreen) execute(a *App) tea.Cmd {
	lines := []string{"Branch: " + r.branch, "Onto: " + r.ontoLabel + " (" + shortOID(r.onto) + ")", "Todo (top runs first):"}
	for _, row := range r.rows {
		lines = append(lines, fmt.Sprintf("  %-6s %s %s", row.Action, row.Commit.Short(), row.Commit.Subject))
	}
	lines = append(lines, "", "Commit IDs in the range will change. Git may stop for edit, reword messages or conflicts.")
	confirmLabel := "Execute plan"
	if len(r.publishedRefs) > 0 {
		lines = append(lines, styleWarning.Render("Published: commits in this range exist on "+strings.Join(r.publishedRefs, ", ")+"."), styleWarning.Render("Rebasing rewrites published history. GitFlow never force-pushes."))
		confirmLabel = "Acknowledge risk and execute"
	}
	a.overlay = &confirmOverlay{
		title:   "Execute interactive rebase?",
		lines:   lines,
		command: "git rebase --interactive --no-update-refs " + shortOID(r.onto) + " (todo supplied via GIT_SEQUENCE_EDITOR for this command only)",
		buttons: []confirmButton{{label: "Cancel"}, {label: confirmLabel, run: func(a *App) tea.Cmd {
			status, err := git.LoadStatus(a.ctx, a.runner)
			if err != nil {
				a.errorText = "Revalidate before rebase: " + err.Error()
				return nil
			}
			a.status = status
			if status.Branch.OID != r.headOID {
				r.invalidated = "HEAD moved since the plan was created: plan invalidated, reopen the planner"
				a.errorText = r.invalidated
				return a.refresh()
			}
			invocation, cleanup, err := git.StartRebaseInvocation(r.onto, r.rows)
			if err != nil {
				a.errorText = err.Error()
				return nil
			}
			previousHead := r.headOID
			return a.gitHandoff("Interactive rebase", invocation, func(a *App, err error) tea.Cmd {
				cleanup()
				return a.routeOperation(r, func(a *App) tea.Cmd {
					if err != nil {
						return nil
					}
					r.completed = []string{"Rebase finished: HEAD " + shortOID(previousHead) + " → " + shortOID(a.status.Branch.OID) + " (refreshing)", "Rewritten commits have new IDs; see history."}
					return nil
				})
			})
		}}},
	}
	return nil
}

func (r *rebaseScreen) statusChanged(a *App) tea.Cmd {
	if r.completed != nil && len(r.completed) > 0 {
		r.completed[0] = "Rebase finished: HEAD is now " + shortOID(a.status.Branch.OID)
	}
	return nil
}

func (r *rebaseScreen) confirmLeave(a *App) bool {
	if r.dirty && r.completed == nil {
		a.discardPlanDialog()
		return false
	}
	return true
}

func (r *rebaseScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	if next, moved := moveCursor(key, r.cursor, len(r.rows), a.height/2); moved {
		r.cursor = next
		return nil, true
	}
	if key == "enter" && r.cursor < len(r.rows) {
		commit, runner := r.rows[r.cursor].Commit, a.runner
		return a.openViewer("commit "+commit.Short(), func(ctx context.Context) (string, error) {
			return git.ShowCommit(ctx, runner, commit.OID)
		}), true
	}
	return nil, false
}

func (r *rebaseScreen) preview(a *App) tea.Cmd {
	rows, onto, label, reason := append([]git.TodoRow{}, r.rows...), r.onto, r.ontoLabel, r.executeUnavailable(a)
	return a.openViewer("Rebase plan preview", func(ctx context.Context) (string, error) {
		var text strings.Builder
		fmt.Fprintf(&text, "git rebase --interactive %s (%s)\n\nTodo as Git will receive it:\n", shortOID(onto), label)
		for _, row := range rows {
			fmt.Fprintf(&text, "%s %s %s\n", row.Action, row.Commit.Short(), row.Commit.Subject)
		}
		if reason != "" {
			fmt.Fprintf(&text, "\nExecution unavailable: %s\n", reason)
		}
		return text.String(), nil
	})
}

func (r *rebaseScreen) view(a *App, width, height int) string {
	risk := styleStaged.Render("none detected in local remote-tracking refs")
	if len(r.publishedRefs) > 0 {
		risk = styleWarning.Render("PUBLISHED on " + strings.Join(r.publishedRefs, ", "))
	}
	body := []string{
		"Branch: " + r.branch,
		"Onto (explicit base): " + r.ontoLabel + " @ " + shortOID(r.onto),
		fmt.Sprintf("Affected range: %s..HEAD · %d commit(s)", shortOID(r.onto), len(r.rows)),
		"Published-history risk: " + risk,
	}
	if r.err != nil {
		body = append(body, styleError.Render(r.err.Error()))
	}
	if reason := r.executeUnavailable(a); reason != "" && r.completed == nil {
		style := styleMuted
		if strings.HasPrefix(reason, "first commit") {
			style = styleError
		}
		body = append(body, style.Render("Execute unavailable: "+reason))
	}
	body = append(body, "")
	var rows []string
	for index, row := range r.rows {
		rows = append(rows, fmt.Sprintf("%2d. %-6s %s %s", index+1, row.Action, styleDirty.Render(row.Commit.Short()), row.Commit.Subject))
	}
	rendered, offset := renderRows(rows, r.cursor, r.offset, height-panelBorderSize-len(body)-2, true)
	r.offset = offset
	body = append(body, rendered...)
	if r.completed != nil {
		body = append(append(body, ""), r.completed...)
	}
	return panel("INTERACTIVE REBASE PLAN", body, width, height, true)
}

func (r *rebaseScreen) footer() string {
	return "j/k Move  Shift+J/K Reorder  p/r/e/s/f/d Pick/Reword/Edit/Squash/Fixup/Drop  v Preview  a Actions (Execute)  Esc Back"
}

func (r *rebaseScreen) menuTitle(a *App) string { return "Actions for rebase plan" }
