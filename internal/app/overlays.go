package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
)

const (
	menuMaxWidth          = 64
	menuChromeHeight      = 3
	noCursor              = -1
	confirmHints          = "%s Confirm · Enter/Esc Cancel (default)"
	consequenceBullet     = "• "
	nameInputTitle        = "Create local branch from %s"
	nameInputPrompt       = "Name: "
	nameInputHints        = "Enter Create · Esc Cancel"
	invalidBranchName     = "Invalid branch name"
	branchExists          = "A local branch with this name already exists"
	validBranchName       = "Valid name"
	validatingName        = "Checking name…"
	createdNotice         = "Created %s · Still on %s"
	mainComparisonFormat  = "Compared with cached origin/main, not live: main ↑%d ↓%d"
	mainRemoteMissing     = "origin/main is not in cached refs; remote state unknown"
	mainChoiceTitle       = "Create %s from local main"
	mainBehindText        = "Local main is %d behind cached origin/main"
	mainDivergedText      = "Local main and cached origin/main have diverged; GitFlow never merges, rebases or resets main"
	mainUpdateFailed      = "Update main failed: "
	mainUpdateScope       = "p fast-forwards local main only; never merges, rebases or stashes"
	mainUpdatedNotCreated = "local main updated to %s; branch %s was not created: %w"
	updateAndCreateHints  = "p Update main & create · c Create directly from local main · Esc Cancel"
	createDirectlyHints   = "c Create directly from local main · Esc Cancel"
	keyUpdateMain         = "p"
	keyCreateDirectly     = "c"
	switchPromptTitle     = "Switch to %s?"
	switchPromptHints     = "s/Enter Switch · Esc Stay on the current branch"
	keySwitch             = "s"
)

var createConsequences = []string{"Creates the local branch only: no checkout, no push, no upstream"}

type branchNameValidatedMsg struct {
	name string
	err  error
}

func renderMenu(title string, entries []menuEntry, cursor, width, height int) string {
	visibleRows := max(1, height-menuChromeHeight)
	firstRow := listCursor{index: cursor}.windowStart(len(entries), visibleRows)
	lines := []string{title}
	for row := firstRow; row < min(len(entries), firstRow+visibleRows); row++ {
		entry := entries[row]
		line := fmt.Sprintf("%-6s %s", entry.keys, entry.action.Title)
		if !entry.available {
			line = appStyles.disabled.Render(line + " — " + entry.reason)
		}
		if row == cursor {
			line = appStyles.selected.Render(line)
		}
		lines = append(lines, line)
	}
	if len(entries) == 0 {
		lines = append(lines, appStyles.disabled.Render("No actions"))
	}
	return appStyles.overlay.Width(min(width-2, menuMaxWidth)).Render(strings.Join(lines, "\n"))
}

type actionsOverlay struct {
	commits []git.CommitSummary
	scope   Scope
	entries []menuEntry
	cursor  listCursor
}

func (overlay *actionsOverlay) ID() ScreenID { return ScreenActions }

func (overlay *actionsOverlay) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	keys := key.String()
	switch {
	case keys == keyEscape:
		app.overlay = nil
	case keys == keyEnter:
		app.overlay = nil
		if len(overlay.entries) > 0 {
			return app.runAction(overlay.entries[overlay.cursor.index].action), true
		}
	case overlay.cursor.navigate(keys, len(overlay.entries), app.height/2):
	default:
		return app.dispatchSecondKey(overlay.scope, contextualPrefix, keys), true
	}
	return nil, true
}

func (overlay *actionsOverlay) View(width, height int) string {
	return renderMenu(fmt.Sprintf("Actions · %s · Enter runs, Esc closes", overlay.scope), overlay.entries, overlay.cursor.index, width, height)
}

func (overlay *actionsOverlay) Scope() Scope { return ScopeOverlay }

type confirmOverlay struct {
	title        string
	command      []string
	consequences []string
	confirmKey   string
	hints        string
	onConfirm    func(app *App) tea.Cmd
}

func (overlay *confirmOverlay) ID() ScreenID { return ScreenConfirm }

func (overlay *confirmOverlay) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	switch key.String() {
	case overlay.confirmKey:
		app.overlay = nil
		return overlay.onConfirm(app), true
	case keyEscape, keyEnter:
		app.overlay = nil
	}
	return nil, true
}

func (overlay *confirmOverlay) View(width, height int) string {
	lines := []string{appStyles.header.Render(overlay.title), strings.Join(overlay.command, " "), ""}
	for _, consequence := range overlay.consequences {
		lines = append(lines, consequenceBullet+consequence)
	}
	hints := fmt.Sprintf(confirmHints, overlay.confirmKey)
	if overlay.hints != "" {
		hints = overlay.hints
	}
	lines = append(lines, "", hints)
	return appStyles.overlay.Width(min(width-2, menuMaxWidth)).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (overlay *confirmOverlay) Scope() Scope { return ScopeOverlay }

type gitMenuOverlay struct {
	entries []menuEntry
}

func (overlay *gitMenuOverlay) ID() ScreenID { return ScreenSync }

func (overlay *gitMenuOverlay) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	if key.String() == keyEscape {
		app.overlay = nil
		return nil, true
	}
	return app.dispatchSecondKey(ScopeGlobal, globalPrefix, key.String()), true
}

func (overlay *gitMenuOverlay) View(width, height int) string {
	return renderMenu("Git menu · press a key, Esc closes", overlay.entries, noCursor, width, height)
}

func (overlay *gitMenuOverlay) Scope() Scope { return ScopeOverlay }

type paletteOverlay struct {
	query     textinput.Model
	helpScope Scope
	entries   []menuEntry
	cursor    listCursor
}

func (overlay *paletteOverlay) ID() ScreenID { return ScreenPalette }

func (overlay *paletteOverlay) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	keys := key.String()
	switch {
	case keys == keyEscape:
		app.overlay = nil
		return nil, true
	case overlay.helpScope != "":
		overlay.cursor.navigate(keys, len(overlay.entries), app.height/2)
		return nil, true
	case keys == keyEnter:
		app.overlay = nil
		if len(overlay.entries) > 0 {
			return app.runAction(overlay.entries[overlay.cursor.index].action), true
		}
		return nil, true
	case key.Type != tea.KeyRunes && key.Type != tea.KeySpace && overlay.cursor.navigate(keys, len(overlay.entries), app.height/2):
		return nil, true
	}
	var cmd tea.Cmd
	overlay.query, cmd = overlay.query.Update(key)
	overlay.entries = app.menuEntries(ScopeGlobal, app.registry.Search(overlay.query.Value()))
	overlay.cursor = listCursor{}
	return cmd, true
}

func (overlay *paletteOverlay) View(width, height int) string {
	if overlay.helpScope != "" {
		return renderMenu(fmt.Sprintf("Help · %s and global keys · Esc closes", overlay.helpScope), overlay.entries, overlay.cursor.index, width, height)
	}
	return renderMenu(overlay.query.View(), overlay.entries, overlay.cursor.index, width, height)
}

func (overlay *paletteOverlay) Scope() Scope { return ScopeOverlay }

type mainComparedMsg struct {
	comparison git.MainComparison
	err        error
}

func mainComparisonText(comparison git.MainComparison) string {
	if !comparison.RemoteFound {
		return mainRemoteMissing
	}
	return fmt.Sprintf(mainComparisonFormat, comparison.Counts.Ahead, comparison.Counts.Behind)
}

func (app *App) checkoutLabel() string {
	if app.repository.status.Detached {
		return detachedLabel + shortOID(app.repository.status.Head)
	}
	return app.repository.status.Branch
}

func (app *App) createBranch(name, startRef, startOID string) tea.Cmd {
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		if err := app.deps.Branches.Create(ctx, name, startRef, startOID); err != nil {
			return nil, err
		}
		return func(app *App) { app.overlay = &switchPrompt{branch: name, oid: startOID} }, nil
	}, RefreshAll, fmt.Sprintf(createdNotice, name, app.checkoutLabel()))
}

type switchPrompt struct {
	branch string
	oid    string
}

func (prompt *switchPrompt) ID() ScreenID { return ScreenConfirm }

func (prompt *switchPrompt) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	switch key.String() {
	case keySwitch, keyEnter:
		app.overlay = nil
		return app.switchBranch(prompt.branch, localRefPrefix+prompt.branch, prompt.oid), true
	case keyEscape:
		app.overlay = nil
	}
	return nil, true
}

func (prompt *switchPrompt) View(width, height int) string {
	lines := []string{appStyles.header.Render(fmt.Sprintf(switchPromptTitle, prompt.branch)), prompt.oid, "", switchPromptHints}
	return appStyles.overlay.Width(min(width-2, menuMaxWidth)).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (prompt *switchPrompt) Scope() Scope { return ScopeOverlay }

type mainBehindPrompt struct {
	name       string
	comparison git.MainComparison
	failure    error
}

func (prompt *mainBehindPrompt) updatable() bool {
	counts := prompt.comparison.Counts
	return prompt.failure == nil && prompt.comparison.RemoteFound && counts.Ahead == 0 && counts.Behind > 0
}

func (prompt *mainBehindPrompt) ID() ScreenID { return ScreenConfirm }

func (prompt *mainBehindPrompt) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	switch key.String() {
	case keyEscape:
		app.overlay = nil
	case keyCreateDirectly:
		app.overlay = nil
		return app.createBranch(prompt.name, git.LocalMainRef, prompt.comparison.LocalOID), true
	case keyUpdateMain:
		if !prompt.updatable() {
			return nil, true
		}
		app.overlay = nil
		name, comparison := prompt.name, prompt.comparison
		return app.mutate(func(ctx context.Context) (func(app *App), error) {
			updatedOID, err := app.deps.Remote.UpdateMain(ctx, comparison.LocalOID)
			if err != nil {
				return func(app *App) {
					app.notice = ""
					app.overlay = &mainBehindPrompt{name: name, comparison: comparison, failure: err}
				}, nil
			}
			if err := app.deps.Branches.Create(ctx, name, git.LocalMainRef, updatedOID); err != nil {
				return nil, fmt.Errorf(mainUpdatedNotCreated, shortOID(updatedOID), name, err)
			}
			return func(app *App) { app.overlay = &switchPrompt{branch: name, oid: updatedOID} }, nil
		}, RefreshAll, fmt.Sprintf(createdNotice, name, app.checkoutLabel())), true
	}
	return nil, true
}

func (prompt *mainBehindPrompt) View(width, height int) string {
	counts := prompt.comparison.Counts
	lines := []string{appStyles.header.Render(fmt.Sprintf(mainChoiceTitle, prompt.name)), git.LocalMainRef + " " + prompt.comparison.LocalOID, mainComparisonText(prompt.comparison), ""}
	var blocked *git.SyncBlocked
	switch {
	case errors.As(prompt.failure, &blocked):
		lines = append(append(lines, appStyles.conflict.Render(syncBlockerText[blocked.Blocker])), blocked.Paths...)
	case prompt.failure != nil:
		lines = append(lines, appStyles.conflict.Render(mainUpdateFailed+prompt.failure.Error()))
	case counts.Ahead > 0 && counts.Behind > 0:
		lines = append(lines, appStyles.warning.Render(mainDivergedText))
	case counts.Behind > 0:
		lines = append(lines, appStyles.warning.Render(fmt.Sprintf(mainBehindText, counts.Behind)))
	}
	hints := createDirectlyHints
	if prompt.updatable() {
		lines, hints = append(lines, mainUpdateScope), updateAndCreateHints
	}
	lines = append(lines, "", hints)
	return appStyles.overlay.Width(min(width-2, menuMaxWidth)).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (prompt *mainBehindPrompt) Scope() Scope { return ScopeOverlay }

type nameInput struct {
	input      textinput.Model
	startRef   string
	startOID   string
	validated  string
	validation error
	main       *git.MainComparison
}

func newNameInput(startRef, startOID string) (*nameInput, tea.Cmd) {
	input := textinput.New()
	input.Prompt = nameInputPrompt
	overlay := &nameInput{input: input, startRef: startRef, startOID: startOID}
	return overlay, overlay.input.Focus()
}

func (input *nameInput) ID() ScreenID { return ScreenConfirm }

func (input *nameInput) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return nil, false
	}
	name := input.input.Value()
	switch key.String() {
	case keyEscape:
		app.overlay = nil
		return nil, true
	case keyEnter:
		if name == "" || input.validated != name || input.validation != nil {
			return nil, true
		}
		app.overlay = nil
		if input.main != nil && (!input.main.RemoteFound || input.main.Counts.Behind > 0) {
			app.overlay = &mainBehindPrompt{name: name, comparison: *input.main}
			return nil, true
		}
		return app.createBranch(name, input.startRef, input.startOID), true
	}
	var cmd tea.Cmd
	input.input, cmd = input.input.Update(key)
	typed := input.input.Value()
	if typed == "" {
		return cmd, true
	}
	return tea.Batch(cmd, func() tea.Msg {
		return branchNameValidatedMsg{name: typed, err: app.deps.Branches.ValidateName(app.ctx, typed)}
	}), true
}

func (input *nameInput) View(width, height int) string {
	status := ""
	switch name := input.input.Value(); {
	case name == "":
	case input.validated != name:
		status = validatingName
	case errors.Is(input.validation, git.ErrBranchExists):
		status = appStyles.conflict.Render(branchExists)
	case input.validation != nil:
		status = appStyles.conflict.Render(invalidBranchName)
	default:
		status = validBranchName
	}
	lines := []string{appStyles.header.Render(fmt.Sprintf(nameInputTitle, input.startRef)), input.startOID}
	if input.main != nil {
		lines = append(lines, mainComparisonText(*input.main))
	}
	lines = append(lines, "", input.input.View(), status, "")
	for _, consequence := range createConsequences {
		lines = append(lines, consequenceBullet+consequence)
	}
	lines = append(lines, "", nameInputHints)
	return appStyles.overlay.Width(min(width-2, menuMaxWidth)).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (input *nameInput) Scope() Scope { return ScopeOverlay }
