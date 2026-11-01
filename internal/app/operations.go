package app

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

const (
	pausedTitle              = "Paused Operation Dashboard · %s"
	noPausedOperationNext    = "Git has no paused operation · Esc returns"
	pausedBranchFormat       = "Branch: %s"
	pausedCommitFormat       = "Stopped at commit %s"
	pausedStepFormat         = " · step %d of %d"
	pausedRemainingFormat    = " · %d more commits queued"
	pausedCompletedFormat    = " · %d commits already applied"
	pausedConflictReason     = "Reason: conflicts in %d files"
	pausedCleanReason        = "Reason: Git stopped without conflicts"
	pausedStopReason         = "Reason: Git stopped at the %s step without conflicts"
	pausedConflictsHeader    = "Conflicted files · Enter opens the Conflict Resolver"
	pausedControlsHeader     = "Controls"
	pausedNextConflicts      = "Next: resolve each file, mark it resolved, then c Continue"
	pausedNextContinue       = "Next: c Continue runs git %s --continue"
	pausedNextRevert         = "Next: finish or abort the revert with Git outside GitFlow"
	reasonNoPausedOperation  = "No paused operation"
	reasonRevertUnsupported  = "revert is not in the v2.3 contract"
	reasonMergeHasNoSkip     = "merge has no native skip"
	reasonUnresolvedFiles    = "Resolve %d conflicted files first"
	resolverTitle            = "Conflict Resolver · %s · paused %s"
	resolverLoading          = "Loading conflict…"
	resolverRegionFormat     = "Region %d of %d · n next · N previous"
	resolverNoRegions        = "No conflict markers remain · Mark resolved from a Actions or : Command palette"
	resolverStageFormat      = "Stage %d · %s: %s"
	resolverChoiceFormat     = "%s %s · %s"
	resolverEditorHint       = "e opens the configured editor · Mark resolved from a Actions or : Command palette"
	stagePresent             = "present"
	stageAbsent              = "absent"
	contentIndent            = "  "
	reasonResolverNotLoaded  = "Conflict not loaded"
	reasonEditorOnly         = "This conflict can only be resolved in the editor"
	reasonNoBase             = "Git has no common-ancestor version for this region"
	reasonNoRegion           = "No conflict region"
	chosenNotice             = "Chose %s for region %d · not staged"
	handoffReturned          = "Returned from %s"
	handoffFailed            = "run %s: %w"
	continuedNotice          = "Continued %s"
	skippedNotice            = "Skipped the current %s commit"
	abortedNotice            = "Aborted %s"
	skipTitle                = "Skip the current %s commit %s"
	abortTitle               = "Abort the paused %s"
	skipDropsCommit          = "Git drops the current commit from this %s; its changes are not applied"
	skipMayStopAgain         = "Git then applies the next queued commit and may stop again"
	abortRestoresStart       = "Git returns HEAD, index and tracked files to where the %s started"
	abortDiscardsResolutions = "Conflict resolutions made during the %s are discarded"
	abortNotUndo             = "Abort is not a universal undo: untracked files and other branches are not restored"
	operationCancelDefault   = "Enter or Esc cancels without touching Git"
	markResolvedTitle        = "Mark %s resolved"
	markResolvedNotice       = "Marked %s resolved · not continued"
	markResolvedAsIs         = "Stages the file exactly as it is now in the working tree"
	markResolvedRemoval      = "Stages the removal of the file, because it is absent from the working tree"
	markResolvedNoContinue   = "Does not continue the %s: use c Continue on the Paused Operation Dashboard"
	reasonMarkersRemain      = "Conflict markers remain in %d regions"
)

var (
	fallbackNotices = map[git.ConflictFallback]string{
		git.FallbackBinary:         "Binary conflict: GitFlow cannot choose a side",
		git.FallbackDeleted:        "Deleted or renamed on one side: GitFlow cannot choose a side",
		git.FallbackSpecialFile:    "Symlink or submodule conflict: GitFlow cannot choose a side",
		git.FallbackMissingFile:    "The file is absent from the working tree",
		git.FallbackMarkersChanged: "Conflict markers were edited and no longer match Git's stages",
	}
	conflictChoices = []struct {
		action ActionID
		side   git.ConflictSide
	}{{actionConflictOurs, git.SideOurs}, {actionConflictBase, git.SideBase}, {actionConflictTheirs, git.SideTheirs}}
)

func (app *App) operationControlAvailability(control ActionID) (bool, string) {
	operation := app.repository.operation
	switch {
	case !operation.Active():
		return false, reasonNoPausedOperation
	case operation.Kind == git.OperationRevert:
		return false, reasonRevertUnsupported
	case control == actionOperationSkip && !operation.SkipAllowed:
		return false, reasonMergeHasNoSkip
	case control == actionOperationContinue && len(app.repository.status.Unmerged) > 0:
		return false, fmt.Sprintf(reasonUnresolvedFiles, len(app.repository.status.Unmerged))
	}
	return true, ""
}

type pausedScreen struct {
	app     *App
	cursor  listCursor
	pending *recovery.PendingRestore
}

func (paused *pausedScreen) ID() ScreenID { return ScreenPaused }

func (paused *pausedScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	conflicted := app.repository.status.Unmerged
	keys := key.String()
	switch {
	case paused.cursor.navigate(keys, len(conflicted), app.height/2):
		return nil, true
	case keys != keyEnter || len(conflicted) == 0:
		return nil, false
	case app.repository.operation.Kind == git.OperationRevert:
		app.notice = reasonRevertUnsupported
		return nil, true
	}
	conflict := &conflictScreen{app: app, path: conflicted[min(paused.cursor.index, len(conflicted)-1)]}
	app.push(conflict)
	return conflict.load(app), true
}

func (paused *pausedScreen) View(width, height int) string {
	app := paused.app
	operation := app.repository.operation
	lines := []string{appStyles.header.Render(fmt.Sprintf(pausedTitle, reasonNoPausedOperation)), noPausedOperationNext}
	if operation.Active() {
		name := operationNames[operation.Kind]
		branch := operation.Branch
		switch {
		case branch != "":
		case app.repository.status.Detached:
			branch = detachedLabel + shortOID(app.repository.status.Head)
		default:
			branch = app.repository.status.Branch
		}
		commit := fmt.Sprintf(pausedCommitFormat, shortOID(operation.Commit))
		if operation.Total > 0 {
			commit += fmt.Sprintf(pausedStepFormat, operation.Step, operation.Total)
		}
		if operation.Completed > 0 {
			commit += fmt.Sprintf(pausedCompletedFormat, operation.Completed)
		}
		if len(operation.Remaining) > 0 {
			commit += fmt.Sprintf(pausedRemainingFormat, len(operation.Remaining))
		}
		conflicted := app.repository.status.Unmerged
		reason, next := pausedCleanReason, fmt.Sprintf(pausedNextContinue, name)
		switch {
		case len(conflicted) > 0:
			reason, next = fmt.Sprintf(pausedConflictReason, len(conflicted)), pausedNextConflicts
		case operation.StopAction != "":
			reason = fmt.Sprintf(pausedStopReason, operation.StopAction)
		}
		if operation.Kind == git.OperationRevert {
			next = pausedNextRevert
		}
		lines = []string{appStyles.header.Render(fmt.Sprintf(pausedTitle, name)), fmt.Sprintf(pausedBranchFormat, branch), commit, reason, ""}
		if len(conflicted) > 0 {
			lines = append(lines, appStyles.header.Render(pausedConflictsHeader))
			for row, path := range conflicted {
				line := appStyles.conflict.Render(path)
				if row == min(paused.cursor.index, len(conflicted)-1) {
					line = appStyles.selected.Render(path)
				}
				lines = append(lines, line)
			}
			lines = append(lines, "")
		}
		lines = append(lines, appStyles.header.Render(pausedControlsHeader))
		for _, entry := range app.menuEntries(ScopeOperation, app.registry.InScope(ScopeOperation)) {
			line := entry.keys + " " + entry.action.Title
			if !entry.available {
				line = appStyles.disabled.Render(line + " — " + entry.reason)
			}
			lines = append(lines, line)
		}
		lines = append(lines, "", next)
	}
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(lines, "\n"))
}

func (paused *pausedScreen) Scope() Scope { return ScopeOperation }

type conflictLoadedMsg struct {
	path      string
	operation git.Operation
	labels    git.SideLabels
	file      git.ConflictFile
	err       error
}

type handoffReadyMsg struct {
	command *exec.Cmd
	finish  func(exitErr error) tea.Msg
	err     error
}

type conflictScreen struct {
	app       *App
	path      string
	loaded    bool
	err       error
	operation git.Operation
	labels    git.SideLabels
	file      git.ConflictFile
	region    int
}

func (app *App) conflictOnTop() (*conflictScreen, bool) {
	if len(app.stack) == 0 {
		return nil, false
	}
	conflict, open := app.top().(*conflictScreen)
	return conflict, open
}

func (conflict *conflictScreen) load(app *App) tea.Cmd {
	path := conflict.path
	return func() tea.Msg {
		operation, err := app.deps.Operations.Active(app.ctx)
		if err != nil {
			return conflictLoadedMsg{path: path, err: fmt.Errorf("load conflict %s: %w", path, err)}
		}
		labels, err := app.deps.Conflicts.Sides(app.ctx, operation)
		if err != nil {
			return conflictLoadedMsg{path: path, err: fmt.Errorf("label conflict %s: %w", path, err)}
		}
		file, err := app.deps.Conflicts.Load(app.ctx, path)
		if err != nil {
			return conflictLoadedMsg{path: path, err: fmt.Errorf("load conflict %s: %w", path, err)}
		}
		return conflictLoadedMsg{path: path, operation: operation, labels: labels, file: file}
	}
}

func (conflict *conflictScreen) ID() ScreenID { return ScreenConflict }

func (conflict *conflictScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case conflictLoadedMsg:
		if msg.path != conflict.path {
			return nil, false
		}
		conflict.loaded, conflict.err = true, msg.err
		conflict.operation, conflict.labels, conflict.file = msg.operation, msg.labels, msg.file
		conflict.region = max(0, min(conflict.region, len(conflict.file.Regions)-1))
		return nil, true
	}
	return nil, false
}

func (conflict *conflictScreen) View(width, height int) string {
	lines := []string{appStyles.header.Render(fmt.Sprintf(resolverTitle, conflict.path, operationNames[conflict.operation.Kind]))}
	file := conflict.file
	switch {
	case conflict.err != nil:
		lines = append(lines, appStyles.conflict.Render(conflict.err.Error()))
	case !conflict.loaded:
		lines = append(lines, resolverLoading)
	default:
		for stage, entry := range file.Stages {
			presence := stagePresent
			if entry.OID == "" {
				presence = stageAbsent
			}
			lines = append(lines, fmt.Sprintf(resolverStageFormat, stage+1, conflict.labels.Of(git.ConflictSide(stage)), presence))
		}
		lines = append(lines, "")
		notice, fallback := fallbackNotices[file.Fallback]
		switch {
		case fallback:
			lines = append(lines, appStyles.warning.Render(notice))
		case len(file.Regions) == 0:
			lines = append(lines, resolverNoRegions)
		default:
			region := file.Regions[conflict.region]
			lines = append(lines, fmt.Sprintf(resolverRegionFormat, conflict.region+1, len(file.Regions)))
			for _, choice := range conflictChoices {
				action, found := conflict.app.registry.ByID(choice.action)
				if !found || (choice.side == git.SideBase && !region.HasBase) {
					continue
				}
				lines = append(lines, appStyles.header.Render(fmt.Sprintf(resolverChoiceFormat, action.Keys[0], action.Title, conflict.labels.Of(choice.side))))
				for _, line := range region.Side(choice.side) {
					lines = append(lines, contentIndent+strings.TrimRight(line, "\r\n"))
				}
			}
		}
		lines = append(lines, "", resolverEditorHint)
	}
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(lines, "\n"))
}

func (conflict *conflictScreen) Scope() Scope { return ScopeConflict }

func (app *App) conflictChoiceAvailability(side git.ConflictSide) (bool, string) {
	conflict, open := app.conflictOnTop()
	switch {
	case !open || !conflict.loaded || conflict.err != nil:
		return false, reasonResolverNotLoaded
	case conflict.file.Fallback != git.FallbackNone:
		return false, reasonEditorOnly
	case len(conflict.file.Regions) == 0:
		return false, reasonNoRegion
	case side == git.SideBase && !conflict.file.Regions[conflict.region].HasBase:
		return false, reasonNoBase
	}
	return true, ""
}

func (app *App) chooseConflictSide(side git.ConflictSide) tea.Cmd {
	conflict, _ := app.conflictOnTop()
	path, region := conflict.path, conflict.region
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		return nil, app.deps.Conflicts.Resolve(ctx, path, region, side)
	}, RefreshStatus, fmt.Sprintf(chosenNotice, conflict.labels.Of(side), region+1))
}

func (app *App) moveConflictRegion(step int) tea.Cmd {
	conflict, _ := app.conflictOnTop()
	conflict.region = max(0, min(conflict.region+step, len(conflict.file.Regions)-1))
	return nil
}

func (app *App) handoff(command *exec.Cmd, finish func(exitErr error) tea.Msg) tea.Cmd {
	program := filepath.Base(command.Path)
	return tea.ExecProcess(command, func(err error) tea.Msg {
		if finish != nil {
			return finish(err)
		}
		if err != nil {
			err = fmt.Errorf(handoffFailed, program, err)
		}
		return mutationDoneMsg{refresh: RefreshStatus, outcome: fmt.Sprintf(handoffReturned, program), err: err}
	})
}
