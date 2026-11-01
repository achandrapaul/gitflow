package app

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/prefs"
)

type changesFocus int

const (
	focusFiles changesFocus = iota
	focusDiff
	focusComposer
	changesPanelCount
)

const (
	panelBorderSize   = 2
	panelTitleHeight  = 1
	focusMarker       = "▶ "
	folderSeparator   = "/"
	indentUnit        = "  "
	checkedMark       = "[x] "
	uncheckedMark     = "[ ] "
	expandedMark      = "▾ "
	collapsedMark     = "▸ "
	renameArrow       = " → "
	statusLoading     = "Loading status…"
	noChangesState    = "No changes · working tree clean"
	noMatchesState    = "No files match this filter · press 1 for All"
	diffPlaceholder   = "Select a file to inspect its diff"
	diffLoading       = "Loading diff…"
	stagedHeading     = "Staged (HEAD → index)"
	unstagedHeading   = "Unstaged (index → worktree)"
	noStagedChanges   = "No staged changes"
	noUnstagedChanges = "No unstaged changes"
	keySpace          = " "
	keyCollapse       = "h"
	keyArrowLeft      = "left"
	keyExpand         = "l"
	keyArrowRight     = "right"
	keyRangeDown      = "J"
	keyShiftDown      = "shift+down"
	keyRangeUp        = "K"
	keyShiftUp        = "shift+up"
	noFileTarget      = "Focus a file or select files"
	nothingToStage    = "nothing to stage"
	nothingToUnstage  = "nothing to unstage"
	unmergedTarget    = "unmerged, resolve the conflict first"
	targetGone        = "no longer changed"
	targetChanged     = "%s blocked: %s changed since selection · review and retry"
	targetRecheck     = "%s: recheck targets: %w"
	skippedPrefix     = " · skipped "
	skippedFormat     = "%s: %s"
	singleFile        = "1 file"
	fileCountFormat   = "%d files"
	hiddenFormat      = " (%d hidden)"
	rowTagFormat      = " [%s]"
	reviewTitle       = "Review · the staged index below is what Commit records"
	reviewHints       = "a Actions (Commit · v Without verification hooks · Amend) · c Edit · Esc Back"
	reviewRefreshed   = "Review refreshed: staged index changed"
	reviewTarget      = "Tree %s · HEAD %s"
	unbornHead        = "none (first commit)"
	stagedCount       = "Staged files (%d)"
	emptyMessage      = "Commit message is empty"
	reviewFirst       = "Review the draft first · Esc from the editor"
	noCommitToAmend   = "No commit to amend"
	unmergedFiles     = "Unmerged files, resolve the conflicts first"
	pausedOperation   = "%s in progress, finish it first"
	publishedWarning  = "HEAD is published · amend rewrites published history"
	committedFormat   = "Committed %s · %s"
	matchesReview     = "matches review"
	differsFromReview = "differs from review"
	lastCommitFormat  = "Last commit %s · %s"
	actualMessage     = "Recorded message:"
	bypassHint        = "a v commits once without verification hooks"
	bypassFormat      = "%w · %s"
)

type commitReview struct {
	tree      string
	head      string
	message   string
	published bool
	loaded    bool
	err       error
}

type reviewCapturedMsg struct {
	review commitReview
}

func (changes *changesScreen) reviewing() bool {
	return changes.focus == focusComposer && !changes.draft.Focused()
}

func (app *App) commitAvailability(amend bool) (bool, string) {
	if !app.loaded {
		return false, reasonStatusLoading
	}
	status := app.repository.status
	if app.repository.operation.Active() {
		return false, fmt.Sprintf(pausedOperation, operationNames[app.repository.operation.Kind])
	}
	if slices.ContainsFunc(status.Changes, func(change git.FileChange) bool { return change.Kind == git.KindUnmerged }) {
		return false, unmergedFiles
	}
	if !amend && !slices.ContainsFunc(status.Changes, git.FileChange.HasStaged) {
		return false, noStagedChanges
	}
	if amend && status.Unborn {
		return false, noCommitToAmend
	}
	review := app.changes.review
	if !app.changes.reviewing() || !review.loaded {
		return false, reviewFirst
	}
	if review.err != nil {
		return false, review.err.Error()
	}
	if strings.TrimSpace(review.message) == "" {
		return false, emptyMessage
	}
	return true, ""
}

func (app *App) runCommit(noVerify, amend bool) tea.Cmd {
	review := app.changes.review
	request := git.CommitRequest{Message: review.message, ReviewedTree: review.tree, ReviewedHead: review.head, NoVerify: noVerify, Amend: amend}
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		result, err := app.deps.Indexer.Commit(ctx, request)
		var commandFailure *git.CommandError
		if err != nil && !noVerify && errors.As(err, &commandFailure) {
			return nil, fmt.Errorf(bypassFormat, err, bypassHint)
		}
		if err != nil {
			return nil, err
		}
		return func(app *App) {
			verdict := matchesReview
			if !result.MatchesReview() {
				verdict = differsFromReview
			}
			app.changes.lastResult = result
			app.changes.draft.Reset()
			app.notice = fmt.Sprintf(committedFormat, result.OID[:shortOIDLength], verdict)
		}, nil
	}, RefreshAll, "")
}

type indexAction struct {
	done         string
	withOriginal bool
	blocker      func(git.FileChange) string
}

var indexActions = map[ActionID]indexAction{
	actionStage: {done: "Staged", blocker: func(change git.FileChange) string {
		switch {
		case change.Kind == git.KindUnmerged:
			return unmergedTarget
		case !change.HasUnstaged():
			return nothingToStage
		}
		return ""
	}},
	actionUnstage: {done: "Unstaged", withOriginal: true, blocker: func(change git.FileChange) string {
		if !change.HasStaged() {
			return nothingToUnstage
		}
		return ""
	}},
}

type fileTargets struct {
	eligible []git.FileChange
	hidden   int
	skipped  []string
}

func (targets fileTargets) describe() string {
	text := fmt.Sprintf(fileCountFormat, len(targets.eligible))
	if len(targets.eligible) == 1 {
		text = singleFile
	}
	if targets.hidden > 0 {
		text += fmt.Sprintf(hiddenFormat, targets.hidden)
	}
	return text
}

func (app *App) indexTargets(id ActionID) fileTargets {
	changes := app.changes
	rows := changes.rows()
	visible := map[string]bool{}
	for _, row := range rows {
		visible[row.path] = !row.folder
	}
	members := changes.checked.With(changes.ranged.Ordered()...).Ordered()
	if inspected, inspecting := app.top().(*diffScreen); inspecting {
		members, visible = []string{inspected.path}, map[string]bool{inspected.path: true}
	}
	if len(members) == 0 && len(rows) > 0 && !rows[changes.focusIndex(rows)].folder {
		members = []string{rows[changes.focusIndex(rows)].path}
	}
	var targets fileTargets
	for _, member := range members {
		index := slices.IndexFunc(app.repository.status.Changes, func(change git.FileChange) bool { return change.Path == member })
		reason := targetGone
		if index >= 0 {
			reason = indexActions[id].blocker(app.repository.status.Changes[index])
		}
		if reason != "" {
			targets.skipped = append(targets.skipped, fmt.Sprintf(skippedFormat, member, reason))
			continue
		}
		targets.eligible = append(targets.eligible, app.repository.status.Changes[index])
		if !visible[member] {
			targets.hidden++
		}
	}
	return targets
}

func (app *App) indexAvailability(id ActionID) (bool, string) {
	if !app.loaded {
		return false, reasonStatusLoading
	}
	targets := app.indexTargets(id)
	switch {
	case len(targets.eligible) > 0:
		return true, ""
	case len(targets.skipped) > 0:
		return false, strings.Join(targets.skipped, ", ")
	}
	return false, noFileTarget
}

func (app *App) runIndexAction(id ActionID, apply func(context.Context, []string) error) tea.Cmd {
	action, _ := app.registry.ByID(id)
	targets := app.indexTargets(id)
	var paths []string
	for _, change := range targets.eligible {
		paths = append(paths, change.Path)
		if indexActions[id].withOriginal && change.OriginalPath != "" {
			paths = append(paths, change.OriginalPath)
		}
	}
	outcome := indexActions[id].done + " " + targets.describe()
	if len(targets.skipped) > 0 {
		outcome += skippedPrefix + strings.Join(targets.skipped, ", ")
	}
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		status, err := app.deps.Status.Snapshot(ctx)
		if err != nil {
			return nil, fmt.Errorf(targetRecheck, action.Title, err)
		}
		for _, target := range targets.eligible {
			if !slices.Contains(status.Changes, target) {
				return nil, fmt.Errorf(targetChanged, action.Title, target.Path)
			}
		}
		return nil, apply(ctx, paths)
	}, action.Refresh, outcome)
}

var (
	panelTitles = map[changesFocus]string{focusFiles: "Files", focusDiff: "Diff", focusComposer: "Commit composer"}
	filterNames = map[git.StatusFilter]string{git.FilterAll: "All", git.FilterStaged: "Staged", git.FilterNotStaged: "Not staged", git.FilterMixed: "Mixed"}
	layoutNames = map[prefs.Layout]string{prefs.LayoutFlat: "Flat", prefs.LayoutFolder: "By folder"}
)

func renderPanel(title string, focused bool, content string, width, height int) string {
	style := appStyles.blurred
	if focused {
		style = appStyles.focused
		title = focusMarker + title
	}
	return style.Width(max(1, width-panelBorderSize)).Height(max(1, height-panelBorderSize)).MaxHeight(max(1, height)).Render(title + "\n" + content)
}

func diffLines(diff git.StagedUnstaged, err error) []string {
	if err != nil {
		return []string{appStyles.conflict.Render(err.Error())}
	}
	var lines []string
	sections := []struct{ heading, body, empty string }{
		{heading: stagedHeading, body: diff.Staged, empty: noStagedChanges},
		{heading: unstagedHeading, body: diff.Unstaged, empty: noUnstagedChanges},
	}
	for _, section := range sections {
		lines = append(lines, appStyles.header.Render(section.heading))
		if section.body == "" {
			lines = append(lines, section.empty)
			continue
		}
		lines = append(lines, strings.Split(strings.TrimSuffix(section.body, "\n"), "\n")...)
	}
	return lines
}

type changeRow struct {
	path      string
	folder    bool
	depth     int
	fileCount int
	change    git.FileChange
}

type changesScreen struct {
	app           *App
	filter        git.StatusFilter
	layout        prefs.Layout
	focusPath     string
	rangeAnchor   string
	checked       orderedSelection[string]
	ranged        orderedSelection[string]
	collapsedDirs map[string]bool
	focus         changesFocus
	draft         textarea.Model
	previewPath   string
	preview       git.StagedUnstaged
	previewErr    error
	previewLoaded bool
	review        commitReview
	lastResult    git.CommitResult
}

func (changes *changesScreen) ID() ScreenID {
	if changes.focus == focusComposer {
		return ScreenComposer
	}
	return ScreenChanges
}

func (changes *changesScreen) rows() []changeRow {
	var visible []git.FileChange
	for _, change := range changes.app.repository.status.Changes {
		if change.Matches(changes.filter) {
			visible = append(visible, change)
		}
	}
	slices.SortFunc(visible, func(left, right git.FileChange) int { return strings.Compare(left.Path, right.Path) })
	rows := make([]changeRow, 0, len(visible))
	if changes.layout == prefs.LayoutFlat {
		for _, change := range visible {
			rows = append(rows, changeRow{path: change.Path, change: change})
		}
		return rows
	}
	fileCounts := map[string]int{}
	for _, change := range visible {
		for directory := path.Dir(change.Path); directory != "."; directory = path.Dir(directory) {
			fileCounts[directory]++
		}
	}
	emitted := map[string]bool{}
	for _, change := range visible {
		parts := strings.Split(change.Path, folderSeparator)
		hidden := false
		for depth := 1; depth < len(parts) && !hidden; depth++ {
			directory := strings.Join(parts[:depth], folderSeparator)
			if !emitted[directory] {
				emitted[directory] = true
				rows = append(rows, changeRow{path: directory, folder: true, depth: depth - 1, fileCount: fileCounts[directory]})
			}
			hidden = changes.collapsedDirs[directory]
		}
		if !hidden {
			rows = append(rows, changeRow{path: change.Path, depth: len(parts) - 1, change: change})
		}
	}
	return rows
}

func (changes *changesScreen) focusIndex(rows []changeRow) int {
	return max(0, slices.IndexFunc(rows, func(row changeRow) bool { return row.path == changes.focusPath }))
}

func (changes *changesScreen) previewFocused(app *App) tea.Cmd {
	rows := changes.rows()
	if len(rows) == 0 {
		return nil
	}
	row := rows[changes.focusIndex(rows)]
	if row.folder || row.path == changes.previewPath {
		return nil
	}
	changes.previewPath, changes.previewLoaded = row.path, false
	return app.loadDiff(row.change)
}

func (changes *changesScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	if loaded, isDiff := msg.(diffLoadedMsg); isDiff {
		if loaded.path == changes.previewPath {
			changes.preview, changes.previewErr, changes.previewLoaded = loaded.diff, loaded.err, true
		}
		return nil, true
	}
	if captured, isReview := msg.(reviewCapturedMsg); isReview {
		if !changes.reviewing() {
			return nil, true
		}
		if changes.review.loaded && captured.review.head == changes.review.head && captured.review.tree != changes.review.tree && app.errorText == "" {
			app.notice = reviewRefreshed
		}
		changes.review = captured.review
		return nil, true
	}
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		if changes.focus != focusComposer {
			return nil, false
		}
		var cmd tea.Cmd
		changes.draft, cmd = changes.draft.Update(msg)
		return cmd, false
	}
	keys := key.String()
	if keys == keyTab || keys == keyShiftTab {
		step := changesFocus(1)
		if keys == keyShiftTab {
			step = changesPanelCount - 1
		}
		changes.focus = (changes.focus + step) % changesPanelCount
		if changes.focus == focusComposer {
			return changes.draft.Focus(), true
		}
		changes.draft.Blur()
		return nil, true
	}
	if changes.reviewing() {
		if keys == keyEscape {
			changes.focus = focusFiles
			return nil, true
		}
		return nil, false
	}
	if changes.focus == focusComposer {
		if keys == keyEscape {
			changes.draft.Blur()
			changes.review = commitReview{}
			return app.refresh(), true
		}
		var cmd tea.Cmd
		changes.draft, cmd = changes.draft.Update(key)
		return cmd, true
	}
	if keys == keyEscape {
		if changes.checked.Len() == 0 && changes.ranged.Len() == 0 && changes.rangeAnchor == "" {
			return nil, false
		}
		changes.checked, changes.ranged, changes.rangeAnchor = orderedSelection[string]{}, orderedSelection[string]{}, ""
		return nil, true
	}
	rows := changes.rows()
	if changes.focus != focusFiles || len(rows) == 0 {
		return nil, false
	}
	focused := changes.focusIndex(rows)
	cursor := listCursor{index: focused}
	if cursor.navigate(keys, len(rows), app.height/2) {
		changes.focusPath, changes.rangeAnchor = rows[cursor.index].path, ""
		return changes.previewFocused(app), true
	}
	row := rows[focused]
	switch keys {
	case keySpace:
		switch {
		case row.folder:
		case changes.checked.Contains(row.path) || changes.ranged.Contains(row.path):
			changes.checked, changes.ranged = changes.checked.Without(row.path), changes.ranged.Without(row.path)
		default:
			changes.checked = changes.checked.With(row.path)
		}
		return nil, true
	case keyCollapse, keyArrowLeft, keyExpand, keyArrowRight:
		if row.folder {
			changes.collapsedDirs[row.path] = keys == keyCollapse || keys == keyArrowLeft
		}
		return nil, true
	case keyRangeDown, keyShiftDown, keyRangeUp, keyShiftUp:
		step := 1
		if keys == keyRangeUp || keys == keyShiftUp {
			step = -1
		}
		next := -1
		for index := focused + step; index >= 0 && index < len(rows) && next < 0; index += step {
			if !rows[index].folder {
				next = index
			}
		}
		if next < 0 && row.folder {
			return nil, true
		}
		if next < 0 {
			next = focused
		}
		anchor := slices.IndexFunc(rows, func(candidate changeRow) bool { return !candidate.folder && candidate.path == changes.rangeAnchor })
		if anchor < 0 {
			anchor = focused
			if row.folder {
				anchor = next
			}
			changes.checked = changes.checked.With(changes.ranged.Ordered()...)
			changes.rangeAnchor = rows[anchor].path
		}
		var spanned []string
		for index := min(anchor, next); index <= max(anchor, next); index++ {
			if !rows[index].folder {
				spanned = append(spanned, rows[index].path)
			}
		}
		changes.ranged = orderedSelection[string]{}.With(spanned...)
		changes.focusPath = rows[next].path
		return changes.previewFocused(app), true
	case keyEnter:
		if row.folder {
			return nil, true
		}
		app.push(&diffScreen{path: row.path})
		return app.loadDiff(row.change), true
	}
	return nil, false
}

func (changes *changesScreen) View(width, height int) string {
	app := changes.app
	rows := changes.rows()
	focused := changes.focusIndex(rows)
	selected := changes.checked.With(changes.ranged.Ordered()...)
	hidden := selected.Len()
	for _, row := range rows {
		if !row.folder && selected.Contains(row.path) {
			hidden--
		}
	}
	commitState := ""
	if commit, found := app.registry.ByID(actionCommit); found {
		_, commitState = commit.Availability(app)
	}
	orientation := fmt.Sprintf("Focus: %s · %s · %s · %d selected · %d hidden · Commit: %s", panelTitles[changes.focus], filterNames[changes.filter], layoutNames[changes.layout], selected.Len(), hidden, commitState)
	panelsHeight := height - lipgloss.Height(orientation)
	composerHeight := draftHeight + panelBorderSize + 1
	listHeight := panelsHeight - composerHeight
	stacked := layoutFor(width, height).stacked
	filesWidth, filesHeight, diffWidth, diffHeight := width/2, listHeight, width-width/2, listHeight
	if stacked {
		filesWidth, filesHeight, diffWidth, diffHeight = width, listHeight/2, width, listHeight-listHeight/2
	}

	visibleCount := max(1, filesHeight-panelBorderSize-panelTitleHeight)
	start := listCursor{index: focused}.windowStart(len(rows), visibleCount)
	var fileLines []string
	for index, row := range rows[start:min(len(rows), start+visibleCount)] {
		line := strings.Repeat(indentUnit, row.depth)
		if row.folder {
			mark := expandedMark
			if changes.collapsedDirs[row.path] {
				mark = collapsedMark
			}
			line += fmt.Sprintf("%s%s/ (%d)", mark, path.Base(row.path), row.fileCount)
		} else {
			mark := uncheckedMark
			if selected.Contains(row.path) {
				mark = checkedMark
			}
			name := row.path
			if changes.layout == prefs.LayoutFolder {
				name = path.Base(row.path)
			}
			if row.change.Kind == git.KindRenamed {
				name = row.change.OriginalPath + renameArrow + name
			}
			line += fmt.Sprintf("%s%c%c %s", mark, row.change.Staged, row.change.Unstaged, name)
			switch {
			case row.change.Mixed():
				line += appStyles.warning.Render(fmt.Sprintf(rowTagFormat, filterNames[git.FilterMixed]))
			case row.change.HasStaged():
				line += appStyles.staged.Render(fmt.Sprintf(rowTagFormat, filterNames[git.FilterStaged]))
			}
		}
		if start+index == focused && changes.focus == focusFiles {
			line = appStyles.selected.Render(line)
		}
		fileLines = append(fileLines, line)
	}
	filesContent := strings.Join(fileLines, "\n")
	switch {
	case !app.loaded:
		filesContent = statusLoading
	case len(app.repository.status.Changes) == 0:
		filesContent = noChangesState
	case len(rows) == 0:
		filesContent = noMatchesState
	}
	diffContent := diffPlaceholder
	if len(rows) > 0 && rows[focused].path == changes.previewPath {
		diffContent = diffLoading
		if changes.previewLoaded {
			diffContent = strings.Join(diffLines(changes.preview, changes.previewErr), "\n")
		}
	}

	draft := changes.draft
	draft.SetWidth(max(1, width-panelBorderSize))
	composer := renderPanel(panelTitles[focusComposer], changes.focus == focusComposer, draft.View(), width, composerHeight)
	if changes.reviewing() {
		review := changes.review
		lines := []string{appStyles.header.Render(reviewTitle), appStyles.statusBar.Render(reviewHints)}
		if !review.loaded {
			lines = append(lines, statusLoading)
		}
		if review.err != nil {
			lines = append(lines, appStyles.conflict.Render(review.err.Error()))
		}
		if review.loaded && review.err == nil {
			head := unbornHead
			if review.head != "" {
				head = review.head[:shortOIDLength]
			}
			lines = append(lines, fmt.Sprintf(reviewTarget, review.tree[:shortOIDLength], head))
			if review.published {
				lines = append(lines, appStyles.warning.Render(publishedWarning))
			}
			var stagedLines []string
			for _, change := range app.repository.status.Changes {
				if change.HasStaged() {
					stagedLines = append(stagedLines, fmt.Sprintf("%s%c %s", indentUnit, change.Staged, change.Path))
				}
			}
			lines = append(lines, fmt.Sprintf(stagedCount, len(stagedLines)))
			lines = append(lines, stagedLines...)
		}
		if result := changes.lastResult; result.OID != "" {
			verdict := matchesReview
			if !result.MatchesReview() {
				verdict = appStyles.warning.Render(differsFromReview)
			}
			lines = append(lines, "", fmt.Sprintf(lastCommitFormat, result.OID[:shortOIDLength], verdict))
			for _, treeChange := range result.TreeChanges {
				lines = append(lines, indentUnit+strings.ReplaceAll(treeChange, "\t", " "))
			}
			if result.MessageChanged {
				lines = append(lines, actualMessage, result.Message)
			}
		}
		reviewPanel := renderPanel(panelTitles[focusComposer], changes.focus == focusComposer, strings.Join(lines, "\n"), width, listHeight)
		return lipgloss.JoinVertical(lipgloss.Left, orientation, reviewPanel, composer)
	}
	files := renderPanel(panelTitles[focusFiles], changes.focus == focusFiles, filesContent, filesWidth, filesHeight)
	diff := renderPanel(panelTitles[focusDiff], changes.focus == focusDiff, diffContent, diffWidth, diffHeight)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, files, diff)
	if stacked {
		panes = lipgloss.JoinVertical(lipgloss.Left, files, diff)
	}
	return lipgloss.JoinVertical(lipgloss.Left, orientation, panes, composer)
}

func (changes *changesScreen) Scope() Scope {
	if changes.focus == focusComposer {
		return ScopeComposer
	}
	return ScopeChanges
}

type diffScreen struct {
	path       string
	staged     bool
	diff       git.StagedUnstaged
	hunk       int
	lineAnchor int
	lineCursor int
	lineMode   bool
	scrollTop  int
	loaded     bool
	err        error
}

func (diff *diffScreen) ID() ScreenID { return ScreenDiff }

func (diff *diffScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case diffLoadedMsg:
		if msg.path == diff.path {
			diff.diff, diff.err, diff.loaded = msg.diff, msg.err, true
		}
		return nil, true
	case tea.KeyMsg:
		cursor := listCursor{index: diff.scrollTop}
		if cursor.navigate(msg.String(), len(diffLines(diff.diff, diff.err)), app.height/2) {
			diff.scrollTop = cursor.index
			return nil, true
		}
	}
	return nil, false
}

func (diff *diffScreen) View(width, height int) string {
	body := diffLoading
	if diff.loaded {
		lines := diffLines(diff.diff, diff.err)
		body = strings.Join(lines[min(diff.scrollTop, len(lines)):], "\n")
	}
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(appStyles.header.Render(diff.path) + "\n" + body)
}

func (diff *diffScreen) Scope() Scope { return ScopeDiff }
