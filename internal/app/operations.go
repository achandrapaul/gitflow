package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

const conflictPaneRows = 8

func (a *App) conflictedFiles() []git.FileChange {
	var conflicted []git.FileChange
	for _, file := range a.status.Files {
		if file.Conflicted() {
			conflicted = append(conflicted, file)
		}
	}
	return conflicted
}

func (a *App) operationSummary() []string {
	operation := a.operation
	branch := operation.HeadName
	if branch == "" {
		branch = a.headLabel()
	}
	target := "Unknown"
	if operation.Onto != "" {
		target = shortOID(operation.Onto)
	}
	progress := "progress Unknown"
	switch {
	case operation.HasProgress:
		progress = fmt.Sprintf("step %d/%d", operation.Step, operation.Total)
	case operation.HasRemaining:
		progress = fmt.Sprintf("%d commit(s) remaining incl. current", operation.Remaining)
	}
	current := "Unknown"
	if operation.CurrentCommit != "" {
		current = shortOID(operation.CurrentCommit)
	}
	return []string{fmt.Sprintf("Operation: %s · branch: %s · onto/base: %s · %s · current commit: %s", operation.Kind.Name(), branch, target, progress, current)}
}

type operationScreen struct {
	cursor int
	offset int
}

func (o *operationScreen) scope() Scope { return ScopeOperation }

func (o *operationScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	conflicted := a.conflictedFiles()
	if next, moved := moveCursor(key, o.cursor, len(conflicted), a.height/2); moved {
		o.cursor = next
		return nil, true
	}
	if key == "enter" {
		if o.cursor >= len(conflicted) {
			a.notice = "No conflicted file to open"
			return nil, true
		}
		resolver := &conflictScreen{path: conflicted[o.cursor].Path, choices: map[int]git.Choice{}}
		a.push(resolver)
		return resolver.opened(a), true
	}
	return nil, false
}

func (o *operationScreen) view(a *App, width, height int) string {
	if !a.operation.Active() {
		return panel("OPERATIONS", []string{"No active operation.", "", "Esc Back"}, width, height, true)
	}
	reason := "No unresolved files. Continue when ready."
	conflicts := len(a.conflictedFiles())
	switch {
	case conflicts > 0:
		reason = fmt.Sprintf("%d conflict(s) need resolution. Git operation still active.", conflicts)
	case a.operation.EditStop:
		reason = "Stopped for edit at " + shortOID(a.operation.CurrentCommit) + ": amend then Continue."
	case a.operation.EmptyPick:
		reason = "The current cherry-pick is empty (its changes are already present). Skip it, or Continue if Git allows."
	}
	body := append(a.operationSummary(), "Reason: "+reason, "")
	conflicted := a.conflictedFiles()
	rows := make([]string, len(conflicted))
	for index, file := range conflicted {
		rows[index] = fmt.Sprintf("%-50s %s  [Enter resolve]", displayPath(file.Path), styleConflict.Render("UNRESOLVED "+string(file.Index)+string(file.Worktree)))
	}
	o.cursor = min(o.cursor, max(len(rows)-1, 0))
	rendered, offset := renderRows(rows, o.cursor, o.offset, height-panelBorderSize-len(body)-5, true)
	o.offset = offset
	body = append(body, rendered...)
	body = append(body, "")
	for _, id := range []string{"op.continue", "op.skip", "op.abort"} {
		action := a.registry.ByID(id)
		line := keyLabel(action) + " " + action.Title
		if reason := a.unavailableReason(action); reason != "" {
			line = styleMuted.Render(line + " [disabled: " + reason + "]")
		}
		body = append(body, line)
	}
	body = append(body, styleMuted.Render("r Refresh · a Actions · Esc Back (operation remains paused) · q Quit leaves Git paused"))
	return panel("OPERATIONS · "+strings.ToUpper(a.operation.Kind.Name())+" PAUSED", body, width, height, true)
}

func (o *operationScreen) footer() string {
	return "j/k Move  Enter Resolve file  c Continue  s Skip  A Abort  r Refresh  a Actions  Esc Back"
}

func (o *operationScreen) menuTitle(a *App) string {
	return "Actions for paused " + a.operation.Kind.Name()
}

func (a *App) operationStepDone(label string) func(a *App, err error) tea.Cmd {
	return func(a *App, err error) tea.Cmd {
		return a.routeOperation(a.top(), func(a *App) tea.Cmd {
			if err == nil {
				a.notice = label + " finished: Git reports no active operation"
			}
			a.popTo(1)
			return nil
		})
	}
}

func (a *App) revalidateOperation() (git.Operation, error) {
	status, err := git.LoadStatus(a.ctx, a.runner)
	if err != nil {
		return git.Operation{}, fmt.Errorf("revalidate status: %w", err)
	}
	operation, err := git.DetectOperation(a.ctx, a.runner, a.repo.GitDir)
	if err != nil {
		return git.Operation{}, fmt.Errorf("revalidate operation: %w", err)
	}
	a.status, a.operation = status, operation
	if !operation.Active() {
		return operation, fmt.Errorf("revalidate operation: Git reports no active operation")
	}
	return operation, nil
}

func (a *App) confirmOperationStep(title, consequence string, invocation git.Invocation, captured bool) {
	expectedKind := a.operation.Kind
	lines := append(a.operationSummary(), "", consequence)
	a.overlay = &confirmOverlay{
		title:   title,
		lines:   lines,
		command: "git " + strings.Join(invocation.Args, " "),
		danger:  true,
		buttons: []confirmButton{{label: "Cancel"}, {label: "Confirm", run: func(a *App) tea.Cmd {
			operation, err := a.revalidateOperation()
			if err != nil || operation.Kind != expectedKind {
				a.errorText = "State changed before confirmation; nothing was run. " + fmt.Sprint(err)
				return a.refresh()
			}
			label := strings.TrimSuffix(title, "?")
			if !captured {
				return a.gitHandoff(label, invocation, a.operationStepDone(label))
			}
			runner, gitDir := a.runner, a.repo.GitDir
			return a.mutate(label, func(ctx context.Context) error {
				if _, err := runner.Execute(ctx, invocation); err != nil {
					return err
				}
				after, err := git.DetectOperation(ctx, runner, gitDir)
				if err != nil {
					return fmt.Errorf("verify %s: %w", label, err)
				}
				if after.Active() {
					return fmt.Errorf("verify %s: Git still reports an active %s", label, after.Kind.Name())
				}
				return nil
			}, a.operationStepDone(label))
		}}},
	}
}

type conflictScreen struct {
	path     string
	conflict git.ConflictFile
	loaded   bool
	err      error
	region   int
	choices  map[int]git.Choice
}

func (c *conflictScreen) scope() Scope { return ScopeConflict }

func (c *conflictScreen) opened(a *App) tea.Cmd {
	runner, path := a.runner, c.path
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		conflict, err := git.LoadConflict(ctx, runner, path)
		return func(a *App) tea.Cmd {
			c.conflict, c.err, c.loaded = conflict, err, true
			c.choices = map[int]git.Choice{}
			c.region = min(c.region, max(len(conflict.Regions)-1, 0))
			return nil
		}
	})
}

func (c *conflictScreen) choose(a *App, choice git.Choice) tea.Cmd {
	c.choices[c.region] = choice
	a.notice = fmt.Sprintf("Region %d: %s chosen in GitFlow only. Nothing is written until Mark resolved.", c.region+1, [...]string{"", "ours", "theirs", "both"}[choice])
	return nil
}

func (c *conflictScreen) sideUnavailable() string {
	if !c.loaded {
		return "loading"
	}
	if c.conflict.TextSidesSupported() != nil {
		return "no text sides for this conflict: use e (external editor) or resolve manually"
	}
	return ""
}

func (c *conflictScreen) resolveUnavailable() string {
	switch {
	case !c.loaded:
		return "loading"
	case c.err != nil:
		return c.err.Error()
	case c.conflict.Missing:
		return "file deleted on one side: resolve with git add or git rm outside GitFlow, then Refresh"
	case c.conflict.Binary:
		return ""
	}
	unresolved := 0
	for index := range c.conflict.Regions {
		if c.choices[index] == git.ChoiceNone {
			unresolved++
		}
	}
	if unresolved > 0 {
		return fmt.Sprintf("%d region(s) unresolved: choose 1/2/3 or edit with e", unresolved)
	}
	return ""
}

func (c *conflictScreen) markResolved(a *App) tea.Cmd {
	var resolved *string
	if len(c.conflict.Regions) > 0 {
		content, err := c.conflict.Resolve(c.choices)
		if err != nil {
			a.errorText = err.Error()
			return nil
		}
		resolved = &content
	}
	runner, path := a.runner, c.path
	run := func(a *App) tea.Cmd {
		return a.mutate("Mark "+displayPath(path)+" resolved", func(ctx context.Context) error {
			return git.MarkResolved(ctx, runner, path, resolved)
		}, func(a *App, err error) tea.Cmd {
			if err == nil {
				a.pop()
			}
			return nil
		})
	}
	if c.conflict.Binary {
		a.overlay = &confirmOverlay{
			title:   "Stage the current binary file as resolved?",
			lines:   []string{displayPath(path), "The working tree file is staged as-is. No side selection is applied.", "Continue still requires a separate explicit step."},
			command: "git add -- " + displayPath(path),
			buttons: []confirmButton{{label: "Cancel"}, {label: "Mark resolved", run: run}},
		}
		return nil
	}
	return run(a)
}

func (c *conflictScreen) openEditor(a *App) tea.Cmd {
	editor, err := git.ConfiguredEditor(a.ctx, a.runner)
	if err != nil {
		a.errorText = err.Error()
		return nil
	}
	arguments := git.EditorInvocation(editor, c.path)
	command := exec.CommandContext(a.ctx, arguments[0], arguments[1:]...)
	command.Dir = a.repo.WorkTree
	return a.handoff("Edit "+displayPath(c.path), command, func(a *App, err error) tea.Cmd { return c.opened(a) })
}

func (c *conflictScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "n":
		c.region = min(c.region+1, max(len(c.conflict.Regions)-1, 0))
		return nil, true
	case "N":
		c.region = max(c.region-1, 0)
		return nil, true
	case "enter":
		a.notice = "Choose 1/2/3, then a > Mark resolved"
		return nil, true
	}
	return nil, false
}

func (c *conflictScreen) view(a *App, width, height int) string {
	ours, theirs := "ours = stage 2 (Unknown operation)", "theirs = stage 3"
	switch a.operation.Kind {
	case git.OperationRebase:
		ours, theirs = "ours = the base being rebased onto (already rewritten commits)", "theirs = your commit being replayed ("+shortOID(a.operation.CurrentCommit)+")"
	case git.OperationCherryPick:
		ours, theirs = "ours = current HEAD ("+a.headLabel()+")", "theirs = the commit being cherry-picked ("+shortOID(a.operation.CurrentCommit)+")"
	case git.OperationMerge:
		ours, theirs = "ours = current branch ("+a.headLabel()+")", "theirs = the branch being merged ("+shortOID(a.operation.CurrentCommit)+")"
	}
	body := []string{displayPath(c.path), styleMuted.Render(ours), styleMuted.Render(theirs)}
	switch {
	case !c.loaded:
		body = append(body, styleMuted.Render("Loading…"))
		return panel("CONFLICT RESOLVER", body, width, height, true)
	case c.err != nil:
		body = append(body, styleError.Render(c.err.Error()))
		return panel("CONFLICT RESOLVER", body, width, height, true)
	case c.conflict.Missing:
		body = append(body, styleWarning.Render("Deleted on one side (modify/delete or rename conflict)."), "Keep the file: git add -- <path>. Delete it: git rm -- <path>. Then press Esc and r to refresh.")
		return panel("CONFLICT RESOLVER", body, width, height, true)
	case c.conflict.Binary:
		body = append(body, styleWarning.Render("Binary conflict: no three-way text view or side selection."), "Replace the file with the content you want (e opens your Git editor), then a > Mark resolved.")
		return panel("CONFLICT RESOLVER", body, width, height, true)
	case len(c.conflict.Regions) == 0:
		body = append(body, styleStaged.Render("No conflict markers remain in the working tree file."), "a > Mark resolved stages it after checking the index.")
		return panel("CONFLICT RESOLVER", body, width, height, true)
	}
	region := c.conflict.Regions[c.region]
	body = append(body, fmt.Sprintf("Conflict region %d of %d · choice: %s", c.region+1, len(c.conflict.Regions), [...]string{"unresolved", "ours", "theirs", "both"}[c.choices[c.region]]))
	paneWidth := width / 3
	baseLines := region.Base
	if !region.HasBase {
		baseLines = []string{styleMuted.Render("(base not in markers: set merge.conflictStyle=diff3 to see it)")}
	}
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		panel("BASE", baseLines, paneWidth, conflictPaneRows, false),
		panel("OURS · 1 "+region.OursLabel, region.Ours, paneWidth, conflictPaneRows, c.choices[c.region] == git.ChoiceOurs),
		panel("THEIRS · 2 "+region.TheirsLabel, region.Theirs, width-2*paneWidth, conflictPaneRows, c.choices[c.region] == git.ChoiceTheirs))
	var result []string
	switch c.choices[c.region] {
	case git.ChoiceOurs:
		result = region.Ours
	case git.ChoiceTheirs:
		result = region.Theirs
	case git.ChoiceBoth:
		result = append(append([]string{}, region.Ours...), region.Theirs...)
	default:
		result = []string{styleMuted.Render("(unresolved: choose 1 ours, 2 theirs, 3 both, or e to edit)")}
	}
	header := panel("CONFLICT RESOLVER", body, width, len(body)+panelBorderSize+1, true)
	resultPanel := panel("RESULT PREVIEW (written only on Mark resolved)", result, width, max(height-lipgloss.Height(header)-conflictPaneRows, 3), false)
	return lipgloss.JoinVertical(lipgloss.Left, header, panes, resultPanel)
}

func (c *conflictScreen) footer() string {
	return "n/N Region  1 Ours  2 Theirs  3 Both  e Editor  a Actions (Mark resolved)  Esc Back to dashboard"
}

func (c *conflictScreen) menuTitle(a *App) string { return "Actions for " + displayPath(c.path) }
