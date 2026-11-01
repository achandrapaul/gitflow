package app

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/prefs"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

type ScreenID string

const (
	defaultWidth     = 80
	defaultHeight    = 24
	minimumWidth     = 40
	minimumHeight    = 12
	globalPrefix     = "g "
	contextualPrefix = "a "
	keyEscape        = "esc"
	keyEnter         = "enter"
	keyTab           = "tab"
	keyShiftTab      = "shift+tab"
	keyDown          = "j"
	keyArrowDown     = "down"
	keyUp            = "k"
	keyArrowUp       = "up"
	keyLastRow       = "G"
	keyHalfPageDown  = "ctrl+d"
	keyHalfPageUp    = "ctrl+u"
	draftHeight      = 3
	draftPlaceholder = "Commit subject"
	indexSummary     = "%d staged · %d mixed · %d unstaged"
	branchLoading    = "Branch: loading"
	shortOIDLength   = 7
	headRef          = "HEAD"
	noUpstream       = "No upstream"
	baseUnavailable  = "Base unavailable"
	freshnessUnknown = "Local refs · last fetched unknown"
	freshnessPrefix  = "Local refs · last fetched "
	fetchTimeLayout  = "2006-01-02 15:04"
	unbornLabel      = " · no commits yet"
	detachedLabel    = "DETACHED @ "
	localRefPrefix   = "refs/heads/"
	remoteRefPrefix  = "refs/remotes/"
	headerSeparator  = " · "
	pausedPrefix     = "Paused: "
	layoutNotSaved   = "Layout changed but not saved"
	layoutNotLoaded  = "Layout preference not loaded, using Flat"
	refreshFailed    = "refresh repository state: %w"
	actionBusy       = "Busy: another Git action is running"
)

var operationNames = map[git.OperationKind]string{
	git.OperationCherryPick: "cherry-pick",
	git.OperationRebase:     "rebase",
	git.OperationMerge:      "merge",
	git.OperationRevert:     "revert",
}

var footerActions = []ActionID{actionOpenActions, actionOpenGitMenu, actionOpenPalette, actionOpenHelp, actionBack}

const (
	ScreenChanges        ScreenID = "S01"
	ScreenDiff           ScreenID = "S02"
	ScreenComposer       ScreenID = "S03"
	ScreenHistory        ScreenID = "S04"
	ScreenActions        ScreenID = "S05"
	ScreenCherryPick     ScreenID = "S06"
	ScreenRebase         ScreenID = "S07"
	ScreenPaused         ScreenID = "S08"
	ScreenConflict       ScreenID = "S09"
	ScreenConfirm        ScreenID = "S10"
	ScreenBranches       ScreenID = "S11"
	ScreenPalette        ScreenID = "S12"
	ScreenCommitFiles    ScreenID = "S13"
	ScreenSelectiveApply ScreenID = "S14"
	ScreenSync           ScreenID = "S15"
	ScreenStashes        ScreenID = "S16"
)

type Dependencies struct {
	Runner         *git.Runner
	Status         *git.StatusReader
	Indexer        *git.Indexer
	Differ         *git.Differ
	History        *git.HistoryPager
	Branches       *git.Brancher
	Remote         *git.RemoteSync
	CherryPicker   *git.CherryPicker
	Rebaser        *git.Rebaser
	Conflicts      *git.ConflictReader
	Stasher        *git.Stasher
	Operations     *git.OperationDetector
	Controls       *git.OperationController
	Restore        *recovery.CherryPickRestore
	SelectiveApply *recovery.SelectiveApply
	StashSwitch    *recovery.StashSwitch
	Prefs          *prefs.Store
	Deleter        *git.FileDeleter
}

type screen interface {
	ID() ScreenID
	Update(app *App, msg tea.Msg) (tea.Cmd, bool)
	View(width, height int) string
	Scope() Scope
}

func shortOID(oid string) string {
	return oid[:min(shortOIDLength, len(oid))]
}

func shortRefName(ref string) string {
	return strings.TrimPrefix(strings.TrimPrefix(ref, remoteRefPrefix), localRefPrefix)
}

func upstreamLabel(upstream string, tracking git.AheadBehind) string {
	if upstream == "" {
		return noUpstream
	}
	return fmt.Sprintf("%s ↑%d ↓%d", upstream, tracking.Ahead, tracking.Behind)
}

func baseLabel(baseRef string, counts git.AheadBehind) string {
	if baseRef == "" {
		return baseUnavailable
	}
	return fmt.Sprintf("Base %s ↑%d ↓%d", shortRefName(baseRef), counts.Ahead, counts.Behind)
}

func freshnessLabel(freshness git.FetchFreshness) string {
	if !freshness.Known {
		return freshnessUnknown
	}
	return freshnessPrefix + freshness.LastFetch.Format(fetchTimeLayout)
}

type repositoryState struct {
	status     git.Status
	operation  git.Operation
	baseRef    string
	baseCounts git.AheadBehind
	freshness  git.FetchFreshness
}

type statusLoadedMsg struct {
	repository repositoryState
	err        error
}

type diffLoadedMsg struct {
	path string
	diff git.StagedUnstaged
	err  error
}

type mutationDoneMsg struct {
	refresh RefreshPolicy
	outcome string
	settle  func(app *App)
	err     error
}

type menuEntry struct {
	action    *Action
	keys      string
	available bool
	reason    string
}

type listCursor struct {
	index int
}

func (cursor *listCursor) navigate(keys string, count, halfPage int) bool {
	switch keys {
	case keyDown, keyArrowDown:
		cursor.index++
	case keyUp, keyArrowUp:
		cursor.index--
	case keyLastRow:
		cursor.index = count - 1
	case keyHalfPageDown:
		cursor.index += halfPage
	case keyHalfPageUp:
		cursor.index -= halfPage
	default:
		return false
	}
	cursor.index = max(0, min(cursor.index, count-1))
	return true
}

func (cursor listCursor) windowStart(count, visible int) int {
	return max(0, min(cursor.index-visible/2, count-visible))
}

type App struct {
	ctx        context.Context
	deps       Dependencies
	registry   *Registry
	width      int
	height     int
	repository repositoryState
	loaded     bool
	changes    *changesScreen
	history    *historyScreen
	cherryPick *cherryPickScreen
	stashes    *stashScreen
	stack      []screen
	overlay    screen
	notice     string
	errorText  string
	busy       bool
}

func NewApp(ctx context.Context, deps Dependencies) (*App, error) {
	registry, err := NewRegistry(defaultActions())
	if err != nil {
		return nil, fmt.Errorf("build action registry: %w", err)
	}
	draft := textarea.New()
	draft.Placeholder = draftPlaceholder
	draft.SetHeight(draftHeight)
	gitflowApp := &App{ctx: ctx, deps: deps, registry: registry, width: defaultWidth, height: defaultHeight}
	preferences, err := deps.Prefs.Load()
	if err != nil {
		gitflowApp.errorText = layoutNotLoaded + ": " + err.Error()
	}
	gitflowApp.changes = &changesScreen{app: gitflowApp, draft: draft, layout: preferences.Layout, collapsedDirs: map[string]bool{}}
	gitflowApp.stack = []screen{gitflowApp.changes}
	return gitflowApp, nil
}

func (app *App) Init() tea.Cmd {
	return app.refresh()
}

func (app *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.width, app.height = msg.Width, msg.Height
		return app, nil
	case tea.KeyMsg:
		app.notice, app.errorText = "", ""
		return app, app.dispatchKey(msg)
	case statusLoadedMsg:
		if msg.err != nil {
			app.errorText = msg.err.Error()
			return app, nil
		}
		changes := app.changes
		previous := changes.rows()
		app.repository, app.loaded = msg.repository, true
		present := map[string]bool{}
		for _, change := range msg.repository.status.Changes {
			present[change.Path] = true
		}
		for _, member := range append(changes.checked.Ordered(), changes.ranged.Ordered()...) {
			if !present[member] {
				changes.checked, changes.ranged = changes.checked.Without(member), changes.ranged.Without(member)
			}
		}
		if !present[changes.rangeAnchor] {
			changes.rangeAnchor = ""
		}
		rows := changes.rows()
		survives := func(candidate changeRow) bool {
			return slices.ContainsFunc(rows, func(row changeRow) bool { return row.path == candidate.path })
		}
		focused := slices.IndexFunc(previous, func(row changeRow) bool { return row.path == changes.focusPath })
		if focused >= 0 && !survives(previous[focused]) {
			for distance := 1; distance < len(previous); distance++ {
				if after := focused + distance; after < len(previous) && survives(previous[after]) {
					changes.focusPath = previous[after].path
					break
				}
				if before := focused - distance; before >= 0 && survives(previous[before]) {
					changes.focusPath = previous[before].path
					break
				}
			}
		}
		changes.previewPath = ""
		followUps := []tea.Cmd{changes.previewFocused(app)}
		if changes.reviewing() {
			message := changes.draft.Value()
			followUps = append(followUps, func() tea.Msg {
				review := commitReview{message: message, loaded: true}
				if review.tree, review.err = app.deps.Indexer.IndexTree(app.ctx); review.err != nil {
					return reviewCapturedMsg{review: review}
				}
				if review.head, review.err = app.deps.Indexer.Head(app.ctx); review.err != nil || review.head == "" {
					return reviewCapturedMsg{review: review}
				}
				review.published, review.err = app.deps.Branches.HeadPublished(app.ctx)
				return reviewCapturedMsg{review: review}
			})
		}
		if branches, open := app.branchesOnTop(); open {
			followUps = append(followUps, branches.load(app))
		}
		if conflict, open := app.conflictOnTop(); open {
			followUps = append(followUps, conflict.load(app))
		}
		inspected, inspecting := app.top().(*diffScreen)
		if !inspecting {
			return app, tea.Batch(followUps...)
		}
		index := slices.IndexFunc(msg.repository.status.Changes, func(change git.FileChange) bool { return change.Path == inspected.path })
		if index < 0 {
			inspected.diff, inspected.err, inspected.loaded = git.StagedUnstaged{}, nil, true
			return app, tea.Batch(followUps...)
		}
		return app, tea.Batch(append(followUps, app.loadDiff(msg.repository.status.Changes[index]))...)
	case branchNameValidatedMsg:
		if input, open := app.overlay.(*nameInput); open && input.input.Value() == msg.name {
			input.validated, input.validation = msg.name, msg.err
		}
		return app, nil
	case mutationDoneMsg:
		app.busy = false
		if msg.err != nil {
			app.errorText = msg.err.Error()
		} else {
			app.notice = msg.outcome
		}
		if msg.settle != nil {
			msg.settle(app)
		}
		if msg.refresh == RefreshNone {
			return app, nil
		}
		return app, app.refresh()
	case stashesLoadedMsg:
		if app.stashes == nil {
			return app, nil
		}
		cmd, _ := app.stashes.Update(app, msg)
		return app, cmd
	case historyLoadedMsg:
		cmd, _ := msg.history.Update(app, msg)
		return app, cmd
	case commitDetailMsg:
		cmd, _ := msg.history.Update(app, msg)
		return app, cmd
	case selectiveApplyPreviewMsg:
		cmd, _ := msg.screen.Update(app, msg)
		return app, cmd
	case cherryPickPreviewMsg:
		cmd, _ := app.cherryPick.Update(app, msg)
		return app, cmd
	case diffLoadedMsg:
		for _, stacked := range app.stack {
			stacked.Update(app, msg)
		}
		return app, nil
	}
	cmd, _ := app.top().Update(app, msg)
	return app, cmd
}

func (app *App) View() string {
	if app.width < minimumWidth || app.height < minimumHeight {
		message := fmt.Sprintf("GitFlow needs at least %d×%d (now %d×%d). Keys stay active: ? help · q quit.", minimumWidth, minimumHeight, app.width, app.height)
		return lipgloss.NewStyle().Width(app.width).MaxHeight(app.height).Render(message)
	}
	top := app.top()
	branchLine := branchLoading
	if app.loaded {
		status := app.repository.status
		shortHead := shortOID(status.Head)
		branch := fmt.Sprintf("Branch %s @ %s", status.Branch, shortHead)
		switch {
		case status.Detached:
			branch = appStyles.warning.Render(detachedLabel + shortHead)
		case status.Unborn:
			branch = "Branch " + status.Branch + unbornLabel
		}
		base := baseUnavailable
		if !status.Unborn {
			base = baseLabel(app.repository.baseRef, app.repository.baseCounts)
		}
		parts := []string{branch, upstreamLabel(status.Upstream, status.Tracking), base, freshnessLabel(app.repository.freshness)}
		if app.repository.operation.Active() {
			parts = append(parts, appStyles.conflict.Render(pausedPrefix+operationNames[app.repository.operation.Kind]))
		}
		branchLine = strings.Join(parts, headerSeparator)
	}
	summary := statusLoading
	if app.loaded {
		var staged, mixed, unstaged int
		for _, change := range app.repository.status.Changes {
			switch {
			case change.Mixed():
				mixed++
			case change.HasStaged():
				staged++
			case change.HasUnstaged():
				unstaged++
			}
		}
		summary = fmt.Sprintf(indexSummary, staged, mixed, unstaged)
		if len(app.repository.status.Changes) == 0 {
			summary = noChangesState
		}
	}
	header := lipgloss.NewStyle().Width(app.width).Render(lipgloss.JoinVertical(lipgloss.Left,
		appStyles.header.Render(fmt.Sprintf("GitFlow · %s · %s", top.ID(), filepath.Base(app.deps.Runner.WorkTree()))),
		branchLine,
		summary,
	))
	var hints []string
	if scope := top.Scope(); app.loaded && (scope == ScopeChanges || scope == ScopeDiff) {
		for _, id := range []ActionID{actionStage, actionUnstage} {
			action, found := app.registry.ByID(id)
			if targets := app.indexTargets(id); found && len(targets.eligible) > 0 {
				hints = append(hints, action.Keys[0]+" "+action.Title+" "+targets.describe())
			}
		}
	}
	for _, id := range footerActions {
		if action, found := app.registry.ByID(id); found {
			hints = append(hints, action.Keys[0]+" "+action.Title)
		}
	}
	result := app.notice
	if app.errorText != "" {
		result = appStyles.conflict.Render(app.errorText)
	}
	footer := lipgloss.NewStyle().Width(app.width).Render(lipgloss.JoinVertical(lipgloss.Left, result, appStyles.statusBar.Render(strings.Join(hints, " · "))))
	bodyHeight := app.height - lipgloss.Height(header) - lipgloss.Height(footer)
	body := top.View(app.width, bodyHeight)
	if app.overlay != nil {
		body = lipgloss.Place(app.width, bodyHeight, lipgloss.Center, lipgloss.Center, app.overlay.View(app.width, bodyHeight))
	}
	body = lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight).Render(body)
	return lipgloss.NewStyle().MaxWidth(app.width).MaxHeight(app.height).Render(lipgloss.JoinVertical(lipgloss.Left, header, body, footer))
}

func (app *App) top() screen {
	return app.stack[len(app.stack)-1]
}

func (app *App) push(next screen) {
	app.stack = append(app.stack, next)
}

func (app *App) loadDiff(file git.FileChange) tea.Cmd {
	return func() tea.Msg {
		diff, err := app.deps.Differ.WorktreeDiff(app.ctx, file)
		return diffLoadedMsg{path: file.Path, diff: diff, err: err}
	}
}

func (app *App) pop() {
	if len(app.stack) > 1 {
		app.stack = app.stack[:len(app.stack)-1]
	}
}

func (app *App) runAction(action *Action) tea.Cmd {
	if available, reason := action.Availability(app); !available {
		app.notice = fmt.Sprintf("%s: %s", action.Title, reason)
		return nil
	}
	return action.Run(app)
}

func (app *App) dispatchSecondKey(scope Scope, prefix, keys string) tea.Cmd {
	app.overlay = nil
	action, found := app.registry.Lookup(scope, prefix+keys)
	if !found {
		app.notice = fmt.Sprintf("%s%s is not assigned", prefix, keys)
		return nil
	}
	return app.runAction(action)
}

func (app *App) menuEntries(scope Scope, actions []*Action) []menuEntry {
	entries := make([]menuEntry, 0, len(actions))
	for _, action := range actions {
		keys := action.Keys
		for _, binding := range action.bindings() {
			if binding.Scope == scope {
				keys = binding.Keys
			}
		}
		available, reason := action.Availability(app)
		entries = append(entries, menuEntry{action: action, keys: strings.Join(keys, ", "), available: available, reason: reason})
	}
	return entries
}

func (app *App) refresh() tea.Cmd {
	return func() tea.Msg {
		var repository repositoryState
		var err error
		if repository.status, err = app.deps.Status.Snapshot(app.ctx); err != nil {
			return statusLoadedMsg{err: fmt.Errorf(refreshFailed, err)}
		}
		if repository.operation, err = app.deps.Operations.Active(app.ctx); err != nil {
			return statusLoadedMsg{err: fmt.Errorf(refreshFailed, err)}
		}
		if repository.baseRef, err = app.deps.Branches.ResolveBase(app.ctx); err != nil {
			return statusLoadedMsg{err: fmt.Errorf(refreshFailed, err)}
		}
		if repository.baseRef != "" && !repository.status.Unborn {
			if repository.baseCounts, err = app.deps.Branches.AheadBehind(app.ctx, headRef, repository.baseRef); err != nil {
				return statusLoadedMsg{err: fmt.Errorf(refreshFailed, err)}
			}
		}
		if repository.freshness, err = app.deps.Branches.Freshness(app.ctx); err != nil {
			return statusLoadedMsg{err: fmt.Errorf(refreshFailed, err)}
		}
		return statusLoadedMsg{repository: repository}
	}
}

func (app *App) mutate(work func(ctx context.Context) (func(app *App), error), refresh RefreshPolicy, outcome string) tea.Cmd {
	if app.busy {
		app.notice = actionBusy
		return nil
	}
	app.busy = true
	return func() tea.Msg {
		settle, err := work(app.ctx)
		return mutationDoneMsg{refresh: refresh, outcome: outcome, settle: settle, err: err}
	}
}

func (app *App) dispatchKey(key tea.KeyMsg) tea.Cmd {
	if app.overlay != nil {
		cmd, _ := app.overlay.Update(app, key)
		return cmd
	}
	top := app.top()
	if cmd, consumed := top.Update(app, key); consumed {
		return cmd
	}
	keys := key.String()
	if action, found := app.registry.Lookup(top.Scope(), keys); found {
		return app.runAction(action)
	}
	if action, found := app.registry.Lookup(ScopeGlobal, keys); found {
		return app.runAction(action)
	}
	if keys == keyEscape {
		app.pop()
	}
	return nil
}
