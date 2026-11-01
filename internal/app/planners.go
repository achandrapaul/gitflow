package app

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

const (
	cherryPickSourceLabel  = "Source"
	cherryPickSourceLine   = "Source %s (browsing only%s) · Target %s (checked out, fixed)"
	cachedRemoteNote       = ", cached remote-tracking ref, not fetched"
	cherryPickModeLine     = "Mode Whole commits (active) %d queued · Files (inactive) %d selected · Tab panes · Space queue · J/K reorder · p Preview"
	cherryPickDirtyLine    = "Tracked changes on target: %s · Execute stays disabled until they are gone"
	sourceBranchesTitle    = "Source branches · Enter browses, never checks out"
	sourceBranchesHeight   = 7
	sourceMarker           = "> "
	queuePositionMark      = "[%d] "
	fileModeLine           = "Mode Files (active) %d selected · Whole commits (inactive) %d queued · Space selects files only"
	previewTitle           = "Cherry-pick preview · %d commits onto %s"
	previewTarget          = "Target %s (checked out) · creates %d new commits, in this order:"
	previewCommitLine      = "%d. %s %s · from %s"
	previewModes           = "Active: whole commits (%d) · Inactive: %d selected files are not applied"
	previewBlocked         = "Blocked: %s"
	previewDirty           = "Blocked: tracked changes: %s · Execute stays disabled until they are gone"
	previewReady           = "Ready: a Actions → " + cherryPickExecuteTitle
	previewHints           = "Enter/Esc closes · Execute from a Actions"
	cherryPickExecuteTitle = "Cherry-pick selected commits"
	cherryPickedNotice     = "Cherry-picked %d commits onto %s"
	cherryPickPausedNotice = "Cherry-pick paused after %d of %d commits · o opens the Paused Operation Dashboard"
	reasonExplorerClosed   = "Open the Cherry-pick Explorer (g c) first"
	reasonQueueEmpty       = "Queue at least one commit with Space"
	reasonPreviewFirst     = "Preview the current queue with p first"
	reasonPreviewBlocked   = "Preview found blockers: %s"
	reasonTrackedChanges   = "Tracked changes: %s"
	reasonFocusNotQueued   = "Focused commit is not queued"
	reasonListSeparator    = "; "
	pathListSeparator      = ", "
	applyExecuteTitle      = "Apply selected files (staged)"
	applyTitle             = "Apply Files Planner · %d files from %d commits onto %s"
	applyDestination       = "Destination: working tree and index (staged) · creates no commit · HEAD stays %s"
	applyGroupLine         = "Group %d · %s %s · from %s"
	applyStepLine          = "  %d. %c %s"
	applyOverlapLine       = "Overlap: %s is changed by steps %s · later steps apply on top of earlier ones"
	applyChecking          = "Checking the plan…"
	applyReady             = "Ready: a Actions → " + applyExecuteTitle
	applyHints             = "p Preview again · a Actions · Esc back to files"
	applyAppliedLabel      = "Applied (staged): "
	applyFailedLabel       = "Failed: "
	applyPendingLabel      = "Not attempted: "
	applyNoRollback        = "Nothing was rolled back: applied files stay staged and cherry-pick --abort does not apply · review them in Changes and unstage or restore by hand"
	applyTrackedNow        = "Tracked changes now (%d): %s"
	appliedNotice          = "Staged %d selected files onto %s · no commit created"
	messagePreparedNotice  = "Commit message prepared in the composer"
	applyMessageOneSource  = "Cherry-pick selected files from %s: %s"
	applyMessageSources    = "Cherry-pick selected files from %d commits"
	applyMessageSourceLine = "Source commit: "
	fileListBullet         = "- "
	replaceDraftTitle      = "Replace the commit draft?"
	replaceDraftKeeps      = "Your current draft stays unless you press c"
	replaceDraftWith       = "New message: %s"
	replaceDraftHints      = "c replaces the draft · Esc keeps it"
	reasonFileModeClosed   = "Open a commit's files from the Cherry-pick Explorer (g c, Enter) first"
	reasonNoFiles          = "Select at least one file with Space"
	reasonPreviewFiles     = "Preview the selected files with p first"
	overlapStepCount       = 2
)

var cherryPickCommand = []string{gitProgram, "cherry-pick"}

type cherryPickPreviewMsg struct {
	reviewedHead string
	result       recovery.PreflightResult
	err          error
}

type cherryPickPreview struct {
	reviewedHead string
	result       recovery.PreflightResult
}

type cherryPickScreen struct {
	app             *App
	commits         *historyScreen
	branches        []git.BranchInfo
	branchesErr     error
	branchRow       int
	branchesFocused bool
	preview         *cherryPickPreview
	pending         *recovery.PendingRestore
	restore         *recovery.RestoreResult
	rollback        recovery.RollbackVerdict
	stashPrompt     bool
}

func (cherryPick *cherryPickScreen) queued() []git.CommitSummary {
	var commits []git.CommitSummary
	for _, oid := range cherryPick.commits.selection.Ordered() {
		commits = append(commits, cherryPick.commits.selected[oid].summary)
	}
	return commits
}

func (app *App) cherryPickQueueAvailability() (bool, string) {
	switch {
	case app.cherryPick == nil || app.top() != app.cherryPick:
		return false, reasonExplorerClosed
	case app.cherryPick.commits.selection.Len() == 0:
		return false, reasonQueueEmpty
	}
	return true, ""
}

func (app *App) cherryPickFocusQueued() (bool, string) {
	if available, reason := app.cherryPickQueueAvailability(); !available {
		return false, reason
	}
	if !app.cherryPick.commits.selection.Contains(app.cherryPick.commits.focusOID) {
		return false, reasonFocusNotQueued
	}
	return true, ""
}

func (cherryPick *cherryPickScreen) ID() ScreenID { return ScreenCherryPick }

func (cherryPick *cherryPickScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		cherryPick.branches, cherryPick.branchesErr = msg.branches, msg.err
		cherryPick.branchRow = max(0, min(cherryPick.branchRow, len(msg.branches)-1))
		return nil, true
	case cherryPickPreviewMsg:
		if msg.err != nil {
			app.errorText = msg.err.Error()
			return nil, true
		}
		cherryPick.preview = &cherryPickPreview{reviewedHead: msg.reviewedHead, result: msg.result}
		if app.top() != cherryPick {
			return nil, true
		}
		result := msg.result
		consequences := []string{fmt.Sprintf(previewTarget, result.Target, len(result.Commits))}
		for position, commit := range cherryPick.queued() {
			consequences = append(consequences, fmt.Sprintf(previewCommitLine, position+1, shortOID(commit.OID), commit.Subject, cherryPick.commits.selected[commit.OID].source))
		}
		consequences = append(consequences, fmt.Sprintf(previewModes, len(result.Commits), cherryPick.commits.files.Len()))
		for _, blocker := range result.Blockers {
			consequences = append(consequences, appStyles.conflict.Render(fmt.Sprintf(previewBlocked, blocker.Reason())))
		}
		if result.NeedsStash {
			consequences = append(consequences, appStyles.conflict.Render(fmt.Sprintf(previewDirty, strings.Join(result.Status.TrackedDirtyPaths, pathListSeparator))))
		}
		if len(result.Blockers) == 0 && !result.NeedsStash {
			consequences = append(consequences, previewReady)
		}
		app.overlay = &confirmOverlay{title: fmt.Sprintf(previewTitle, len(result.Commits), result.Target), command: append(slices.Clone(cherryPickCommand), result.Commits...), consequences: consequences, hints: previewHints}
		return nil, true
	case tea.KeyMsg:
		commits := cherryPick.commits
		keys := msg.String()
		branchCursor := listCursor{index: cherryPick.branchRow}
		switch {
		case commits.searching:
			return commits.Update(app, msg)
		case keys == keyTab && cherryPick.branchesFocused:
			cherryPick.branchesFocused, commits.detailFocused = false, false
		case keys == keyTab && !commits.detailFocused:
			commits.detailFocused = true
		case keys == keyTab:
			cherryPick.branchesFocused, commits.detailFocused = true, false
		case !cherryPick.branchesFocused:
			return commits.Update(app, msg)
		case keys == keyEnter && cherryPick.branchRow < len(cherryPick.branches):
			commits.sourceRef, commits.focusOID = cherryPick.branches[cherryPick.branchRow].Ref, ""
			cherryPick.branchesFocused = false
			return commits.reload(app), true
		case len(cherryPick.branches) > 0 && branchCursor.navigate(keys, len(cherryPick.branches), sourceBranchesHeight):
			cherryPick.branchRow = branchCursor.index
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

func (cherryPick *cherryPickScreen) View(width, height int) string {
	commits := cherryPick.commits
	remoteNote := ""
	if strings.HasPrefix(commits.cursor.SourceRef, remoteRefPrefix) {
		remoteNote = cachedRemoteNote
	}
	status := cherryPick.app.repository.status
	lines := []string{
		appStyles.header.Render(fmt.Sprintf(cherryPickSourceLine, commits.cursor.Label, remoteNote, status.Branch)),
		fmt.Sprintf(cherryPickModeLine, commits.selection.Len(), commits.files.Len()),
	}
	if len(status.TrackedDirtyPaths) > 0 {
		lines = append(lines, appStyles.warning.Render(fmt.Sprintf(cherryPickDirtyLine, strings.Join(status.TrackedDirtyPaths, pathListSeparator))))
	}
	var branchLines []string
	switch {
	case cherryPick.branchesErr != nil:
		branchLines = append(branchLines, appStyles.conflict.Render(cherryPick.branchesErr.Error()))
	case cherryPick.branches == nil:
		branchLines = append(branchLines, branchesLoading)
	}
	visibleBranches := max(1, sourceBranchesHeight-panelBorderSize-panelTitleHeight)
	firstBranch := listCursor{index: cherryPick.branchRow}.windowStart(len(cherryPick.branches), visibleBranches)
	for row := firstBranch; row < min(len(cherryPick.branches), firstBranch+visibleBranches); row++ {
		branch := cherryPick.branches[row]
		marker := otherBranchMarker
		if branch.Ref == commits.cursor.SourceRef {
			marker = sourceMarker
		}
		line := marker + branch.Name
		if branch.Remote {
			line += " · " + remoteTrackingKind
		}
		if row == cherryPick.branchRow && cherryPick.branchesFocused {
			line = appStyles.selected.Render(line)
		}
		branchLines = append(branchLines, line)
	}
	lines = append(lines, renderPanel(sourceBranchesTitle, cherryPick.branchesFocused, strings.Join(branchLines, "\n"), width, sourceBranchesHeight))
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(append(lines, commits.View(width, max(1, height-lipgloss.Height(strings.Join(lines, "\n"))))), "\n"))
}

func (cherryPick *cherryPickScreen) Scope() Scope {
	if cherryPick.restore != nil && len(cherryPick.restore.Outcome.Conflicted) > 0 {
		return ScopeStashRestore
	}
	return ScopeCherryPick
}

type rebaseScreen struct {
	base       string
	steps      []git.TodoStep
	row        int
	preview    bool
	published  bool
	validation string
}

func (rebase *rebaseScreen) ID() ScreenID { return ScreenRebase }

func (rebase *rebaseScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) { return nil, false }

func (rebase *rebaseScreen) View(width, height int) string { return "" }

func (rebase *rebaseScreen) Scope() Scope { return ScopeRebase }

type selectiveApplyPreviewMsg struct {
	screen       *selectiveApplyScreen
	reviewedHead string
	steps        []git.ApplyStep
	result       recovery.PreflightResult
	err          error
}

type selectiveApplyScreen struct {
	app      *App
	history  *historyScreen
	steps    []git.ApplyStep
	scroll   int
	overlaps map[string][]int
	preview  *cherryPickPreview
	report   *git.ApplyReport
	applyErr error
}

func (app *App) selectiveApplyAvailability() (*historyScreen, bool, string) {
	if len(app.stack) == 0 {
		return nil, false, reasonFileModeClosed
	}
	var history *historyScreen
	switch top := app.top().(type) {
	case *commitFilesScreen:
		history = top.history
	case *selectiveApplyScreen:
		history = top.history
	}
	switch {
	case history == nil || history.explorer == nil:
		return nil, false, reasonFileModeClosed
	case history.files.Len() == 0:
		return nil, false, reasonNoFiles
	}
	return history, true, ""
}

func (selectiveApply *selectiveApplyScreen) lines() []string {
	status := selectiveApply.app.repository.status
	var groups int
	var stepLines []string
	for position, step := range selectiveApply.steps {
		selection := step.Selection
		if position == 0 || selectiveApply.steps[position-1].Selection.OID != selection.OID {
			groups++
			source := selectiveApply.history.fileSources[selection.OID]
			stepLines = append(stepLines, appStyles.header.Render(fmt.Sprintf(applyGroupLine, groups, selection.OID, source.summary.Subject, source.source)))
		}
		stepLines = append(stepLines, fmt.Sprintf(applyStepLine, position+1, selection.Status, selection.Path))
	}
	lines := []string{
		appStyles.header.Render(fmt.Sprintf(applyTitle, len(selectiveApply.steps), groups, status.Branch)),
		appStyles.warning.Render(fmt.Sprintf(fileModeLine, selectiveApply.history.files.Len(), selectiveApply.history.selection.Len())),
		fmt.Sprintf(applyDestination, shortOID(status.Head)),
	}
	lines = append(lines, stepLines...)
	overlapping := slices.Sorted(maps.Keys(selectiveApply.overlaps))
	for _, path := range overlapping {
		var positions []string
		for _, position := range selectiveApply.overlaps[path] {
			positions = append(positions, strconv.Itoa(position+1))
		}
		lines = append(lines, appStyles.warning.Render(fmt.Sprintf(applyOverlapLine, path, strings.Join(positions, pathListSeparator))))
	}
	switch {
	case selectiveApply.report != nil:
		report := *selectiveApply.report
		for _, section := range []struct {
			label      string
			selections []git.FileSelection
		}{{applyAppliedLabel, report.Applied}, {applyFailedLabel, report.Failed}, {applyPendingLabel, report.Skipped}} {
			if len(section.selections) == 0 {
				continue
			}
			var paths []string
			for _, selection := range section.selections {
				paths = append(paths, selection.Path)
			}
			lines = append(lines, section.label+strings.Join(paths, pathListSeparator))
		}
		if selectiveApply.applyErr != nil {
			lines = append(lines, appStyles.conflict.Render(selectiveApply.applyErr.Error()))
		}
		if len(report.Failed) > 0 {
			lines = append(lines, appStyles.warning.Render(applyNoRollback))
		}
		lines = append(lines, fmt.Sprintf(applyTrackedNow, len(status.TrackedDirtyPaths), strings.Join(status.TrackedDirtyPaths, pathListSeparator)))
	case selectiveApply.preview == nil:
		lines = append(lines, applyChecking)
	default:
		for _, blocker := range selectiveApply.preview.result.Blockers {
			lines = append(lines, appStyles.conflict.Render(fmt.Sprintf(previewBlocked, blocker.Reason())))
		}
		if len(selectiveApply.preview.result.Blockers) == 0 {
			lines = append(lines, applyReady)
		}
	}
	return append(lines, "", applyHints)
}

func (selectiveApply *selectiveApplyScreen) ID() ScreenID { return ScreenSelectiveApply }

func (selectiveApply *selectiveApplyScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case selectiveApplyPreviewMsg:
		if msg.err != nil {
			app.errorText = msg.err.Error()
			return nil, true
		}
		selectiveApply.steps, selectiveApply.report, selectiveApply.applyErr = msg.steps, nil, nil
		selectiveApply.preview = &cherryPickPreview{reviewedHead: msg.reviewedHead, result: msg.result}
		touched := map[string][]int{}
		for position, step := range msg.steps {
			touched[step.Selection.Path] = append(touched[step.Selection.Path], position)
		}
		maps.DeleteFunc(touched, func(path string, positions []int) bool { return len(positions) < overlapStepCount })
		selectiveApply.overlaps = touched
		return nil, true
	case tea.KeyMsg:
		cursor := listCursor{index: selectiveApply.scroll}
		if !cursor.navigate(msg.String(), len(selectiveApply.lines()), app.height/2) {
			return nil, false
		}
		selectiveApply.scroll = cursor.index
		return nil, true
	}
	return nil, false
}

func (selectiveApply *selectiveApplyScreen) View(width, height int) string {
	lines := selectiveApply.lines()
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(lines[min(selectiveApply.scroll, len(lines)-1):], "\n"))
}

func (selectiveApply *selectiveApplyScreen) Scope() Scope { return ScopeSelectiveApply }
