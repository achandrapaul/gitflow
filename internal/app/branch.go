package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

const (
	branchesTitle             = "Branches · local and remote-tracking refs"
	branchSearchLabel         = "Search all local and remote-tracking refs"
	branchSearchPrompt        = branchSearchLabel + ": "
	branchesLoading           = "Loading branches…"
	noBranchesMatch           = "No refs match"
	currentBranchMarker       = "* "
	otherBranchMarker         = "  "
	upstreamArrow             = " → "
	inspectingTitle           = "Inspecting %s · %s"
	localBranchKind           = "Local branch"
	remoteTrackingKind        = "Remote-tracking ref, locally cached"
	inspectionLoading         = "Inspecting…"
	switchBlockedTitle        = "Switch to %s blocked"
	switchedNotice            = "Switched to %s"
	stashAndSwitchHints       = "s Stash and Switch · Esc Stay and fix manually"
	blockedHints              = "Esc Close"
	keyStashAndSwitch         = "s"
	deleteBranchTitle         = "Delete local branch %s"
	deleteLocalOnly           = "Deletes only this local ref; remote refs untouched"
	deleteTipPrefix           = "Tip "
	deleteUnmergedWarning     = "Git refuses unmerged branches; GitFlow never forces"
	deletedNotice             = "Deleted %s"
	reasonNoCommits           = "No commits yet"
	reasonNoBranchFocused     = "No branch focused"
	reasonBranchCheckedOut    = "Branch is checked out"
	reasonRemoteTrackingRef   = "Remote-tracking ref; only local branches qualify"
	reasonDetachedHead        = "Detached HEAD; switch to a local branch"
	reasonNoUpstream          = "No upstream configured"
	reasonUpstreamContractGap = "Choosing an upstream target awaits a contract decision (OPEN_QUESTIONS Q28)"
	gitProgram                = "git"
	fetchTitle                = "Fetch"
	pullTitle                 = "Pull"
	pullFastForwardTitle      = "Pull (fast-forward only)"
	pullRebaseTitle           = "Pull (rebase)"
	pushTitle                 = "Push"
	forcePushTitle            = "Force push with lease"
	syncBlockedTitle          = "%s blocked"
	syncedNotice              = "%s: %s"
	syncFinished              = "done"
	firstPushTitle            = "First push of %s · no upstream yet"
	firstPushExists           = "%s exists in cached refs at %s · local ↑%d ↓%d"
	firstPushAbsent           = "%s is not in cached refs (%s); the push creates it"
	firstPushUnknown          = "%s is not in cached refs and was never fetched; remote state unknown"
	firstPushLocal            = "Local %s at %s"
	firstPushTracking         = "Tracking is set only after the push succeeds"
	forceReviewTitle          = "Force push %s with lease"
	forceDestination          = "Rewrites %s %s · expected remote %s, local %s"
	forceRewrite              = "Remote commits missing from the local branch are dropped from that ref"
	forceStale                = "Fails if the remote moved since the last fetch; never falls back to --force"
	stashesTitle              = "Stash list · read-only · %d entries"
	stashFilterLabel          = "Filter loaded stash entries"
	stashFilterPrompt         = stashFilterLabel + ": "
	stashesLoading            = "Reading git stash list…"
	noStashes                 = "No stash entries"
	noStashesMatch            = "No stash entries match"
	stashRowFormat            = "%s %s %s %s"
)

var (
	fetchCommand           = []string{gitProgram, "fetch"}
	pullCommand            = []string{gitProgram, "pull"}
	pullFastForwardCommand = []string{gitProgram, "pull", "--ff-only"}
	pullRebaseCommand      = []string{gitProgram, "pull", "--rebase"}
	pushCommand            = []string{gitProgram, "push"}
)

var syncBlockerText = map[git.SyncBlocker]string{
	git.SyncNoRemote:           "No remote configured · add one with git remote add, then fetch",
	git.SyncNoOrigin:           "No origin remote · the first push targets origin/<branch>; add origin, then push again",
	git.SyncUnborn:             "No commits yet · commit first",
	git.SyncDetached:           "Detached HEAD · switch to a local branch with g b",
	git.SyncNoUpstream:         "No upstream configured · g P reviews a first push that sets it",
	git.SyncOperation:          "A paused Git operation is in progress · finish or abort it from o",
	git.SyncUnmerged:           "Unmerged paths must be resolved first:",
	git.SyncDirty:              "Tracked changes would be at risk · commit or discard them, then retry:",
	git.SyncUnresolvedStrategy: "Branch has diverged and no pull.rebase, pull.ff or branch.<name>.rebase is set · set one with git config; GitFlow never guesses",
	git.SyncLeaseUnknown:       "Expected remote OID unknown: upstream is not in cached refs · fetch with g e, then review again",
}

var switchBlockerText = map[git.SwitchBlocker]string{
	git.BlockedCurrent:   "Already checked out",
	git.BlockedNotLocal:  "Remote-tracking ref; never checked out as a branch",
	git.BlockedOperation: "A paused Git operation is in progress",
	git.BlockedUnmerged:  "Unmerged paths must be resolved before switching",
	git.BlockedUntracked: "Untracked files would be overwritten by the switch",
	git.BlockedDirty:     "Tracked changes differ from the target branch",
}

type branchesLoadedMsg struct {
	branches []git.BranchInfo
	err      error
}

type branchInspectedMsg struct {
	inspection branchInspection
}

type branchInspection struct {
	branch    git.BranchInfo
	upstream  git.AheadBehind
	base      git.AheadBehind
	baseRef   string
	freshness git.FetchFreshness
	loaded    bool
	err       error
}

type branchScreen struct {
	branches    []git.BranchInfo
	focusRef    string
	loaded      bool
	err         error
	search      textinput.Model
	searching   bool
	inspecting  bool
	inspection  branchInspection
	switchStash *recovery.SwitchReview
}

func (app *App) branchesOnTop() (*branchScreen, bool) {
	if len(app.stack) == 0 {
		return nil, false
	}
	branches, open := app.top().(*branchScreen)
	return branches, open
}

func (app *App) focusedBranch() (git.BranchInfo, bool) {
	branches, open := app.branchesOnTop()
	if !open {
		return git.BranchInfo{}, false
	}
	return branches.focused()
}

func (branches *branchScreen) load(app *App) tea.Cmd {
	return func() tea.Msg {
		list, err := app.deps.Branches.List(app.ctx, true)
		return branchesLoadedMsg{branches: list, err: err}
	}
}

func (branches *branchScreen) visible() []git.BranchInfo {
	query := strings.ToLower(branches.search.Value())
	if query == "" {
		return branches.branches
	}
	return slices.DeleteFunc(slices.Clone(branches.branches), func(branch git.BranchInfo) bool {
		return !strings.Contains(strings.ToLower(branch.Name), query)
	})
}

func (branches *branchScreen) focusedIndex(rows []git.BranchInfo) int {
	return slices.IndexFunc(rows, func(branch git.BranchInfo) bool { return branch.Ref == branches.focusRef })
}

func (branches *branchScreen) focused() (git.BranchInfo, bool) {
	rows := branches.visible()
	index := branches.focusedIndex(rows)
	if index < 0 {
		return git.BranchInfo{}, false
	}
	return rows[index], true
}

func (branches *branchScreen) refocus(index int) {
	rows := branches.visible()
	branches.focusRef = ""
	if len(rows) > 0 {
		branches.focusRef = rows[min(index, len(rows)-1)].Ref
	}
}

func (branches *branchScreen) ID() ScreenID { return ScreenBranches }

func (branches *branchScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		previous := branches.focusedIndex(branches.visible())
		branches.branches, branches.err, branches.loaded = msg.branches, msg.err, true
		if _, found := branches.focused(); !found {
			branches.refocus(max(0, previous))
		}
		return nil, true
	case branchInspectedMsg:
		if msg.inspection.branch.Ref == branches.inspection.branch.Ref {
			branches.inspection = msg.inspection
		}
		return nil, true
	case tea.KeyMsg:
		keys := msg.String()
		rows := branches.visible()
		cursor := listCursor{index: max(0, branches.focusedIndex(rows))}
		switch {
		case keys == keyEnter:
			branches.searching = false
			branches.search.Blur()
			focused, found := branches.focused()
			if !found {
				return nil, true
			}
			branches.inspecting, branches.inspection = true, branchInspection{branch: focused}
			baseRef, freshness := app.repository.baseRef, app.repository.freshness
			return func() tea.Msg {
				inspection := branchInspection{branch: focused, baseRef: baseRef, freshness: freshness, loaded: true}
				if focused.Upstream != "" {
					if inspection.upstream, inspection.err = app.deps.Branches.AheadBehind(app.ctx, focused.Ref, focused.Upstream); inspection.err != nil {
						return branchInspectedMsg{inspection: inspection}
					}
				}
				if baseRef != "" {
					inspection.base, inspection.err = app.deps.Branches.AheadBehind(app.ctx, focused.Ref, baseRef)
				}
				return branchInspectedMsg{inspection: inspection}
			}, true
		case keys == keyEscape && branches.inspecting:
			branches.inspecting = false
		case keys == keyEscape && (branches.searching || branches.search.Value() != ""):
			branches.searching = false
			branches.search.Blur()
			branches.search.SetValue("")
		case keys == keyEscape:
			return nil, false
		case len(rows) > 0 && (!branches.searching || msg.Type == tea.KeyUp || msg.Type == tea.KeyDown) && cursor.navigate(keys, len(rows), app.height/2):
			branches.focusRef, branches.inspecting = rows[cursor.index].Ref, false
		case branches.searching:
			var cmd tea.Cmd
			branches.search, cmd = branches.search.Update(msg)
			if _, found := branches.focused(); !found {
				branches.refocus(0)
			}
			branches.inspecting = false
			return cmd, true
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

func (branches *branchScreen) View(width, height int) string {
	lines := []string{appStyles.header.Render(branchesTitle), appStyles.disabled.Render("/ " + branchSearchLabel)}
	if branches.searching || branches.search.Value() != "" {
		lines[1] = branches.search.View()
	}
	rows := branches.visible()
	switch {
	case branches.err != nil:
		lines = append(lines, appStyles.conflict.Render(branches.err.Error()))
	case !branches.loaded:
		lines = append(lines, branchesLoading)
	case len(rows) == 0:
		lines = append(lines, appStyles.disabled.Render(noBranchesMatch))
	}
	var inspector []string
	if branches.inspecting {
		inspected := branches.inspection
		kind := localBranchKind
		if inspected.branch.Remote {
			kind = remoteTrackingKind
		}
		inspector = append(inspector, "", appStyles.header.Render(fmt.Sprintf(inspectingTitle, inspected.branch.Ref, kind)), inspected.branch.OID)
		switch {
		case inspected.err != nil:
			inspector = append(inspector, appStyles.conflict.Render(inspected.err.Error()))
		case !inspected.loaded:
			inspector = append(inspector, inspectionLoading)
		default:
			inspector = append(inspector,
				upstreamLabel(shortRefName(inspected.branch.Upstream), inspected.upstream),
				baseLabel(inspected.baseRef, inspected.base),
				freshnessLabel(inspected.freshness))
		}
	}
	listHeight := max(1, height-len(lines)-len(inspector))
	focusedIndex := branches.focusedIndex(rows)
	firstRow := listCursor{index: focusedIndex}.windowStart(len(rows), listHeight)
	for row := firstRow; row < min(len(rows), firstRow+listHeight); row++ {
		branch := rows[row]
		marker := otherBranchMarker
		if branch.Current {
			marker = currentBranchMarker
		}
		line := fmt.Sprintf("%s%s %s", marker, branch.Name, shortOID(branch.OID))
		if branch.Upstream != "" {
			line += upstreamArrow + shortRefName(branch.Upstream)
		}
		if row == focusedIndex {
			line = appStyles.selected.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, inspector...)
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(lines, "\n"))
}

func (branches *branchScreen) Scope() Scope { return ScopeBranch }

func (app *App) syncAvailability(needsUpstream bool) (bool, string) {
	status := app.repository.status
	switch {
	case !app.loaded:
		return false, reasonStatusLoading
	case status.Unborn:
		return false, reasonNoCommits
	case status.Detached:
		return false, reasonDetachedHead
	case needsUpstream && status.Upstream == "":
		return false, reasonNoUpstream
	}
	return true, ""
}

func (app *App) switchBranch(name, ref, oid string) tea.Cmd {
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		var blocked *git.SwitchBlocked
		if err := app.deps.Branches.Switch(ctx, name, oid); !errors.As(err, &blocked) {
			return nil, err
		}
		return func(app *App) {
			app.notice = ""
			review := &confirmOverlay{title: fmt.Sprintf(switchBlockedTitle, ref), command: slices.Concat(switchCommand, []string{name}),
				consequences: append([]string{switchBlockerText[blocked.Blocker]}, blocked.Paths...), hints: blockedHints}
			if blocked.Blocker == git.BlockedDirty {
				review.confirmKey, review.hints = keyStashAndSwitch, stashAndSwitchHints
				review.onConfirm = func(app *App) tea.Cmd {
					stashAndSwitch, _ := app.registry.ByID(actionStashAndSwitch)
					return app.runAction(stashAndSwitch)
				}
			}
			app.overlay = review
		}, nil
	}, RefreshAll, fmt.Sprintf(switchedNotice, name))
}

func (app *App) syncRemote(title string, command []string, work func(ctx context.Context) (string, func(app *App), error)) tea.Cmd {
	return app.mutate(func(ctx context.Context) (func(app *App), error) {
		summary, review, err := work(ctx)
		var blocked *git.SyncBlocked
		switch {
		case errors.As(err, &blocked):
			return func(app *App) {
				app.overlay = &confirmOverlay{title: fmt.Sprintf(syncBlockedTitle, title), command: command,
					consequences: append([]string{syncBlockerText[blocked.Blocker]}, blocked.Paths...), hints: blockedHints}
			}, nil
		case err != nil:
			return review, err
		case review != nil:
			return review, nil
		}
		if summary == "" {
			summary = syncFinished
		}
		return func(app *App) { app.notice = fmt.Sprintf(syncedNotice, title, summary) }, nil
	}, RefreshAll, "")
}

type forceOverlay struct {
	*confirmOverlay
}

func (force *forceOverlay) ID() ScreenID { return ScreenSync }

type stashesLoadedMsg struct {
	stashes []git.StashRef
	err     error
}

type stashScreen struct {
	stashes   []git.StashRef
	focusOID  string
	loaded    bool
	err       error
	search    textinput.Model
	searching bool
}

func (app *App) stashesOnTop() (*stashScreen, bool) {
	if len(app.stack) == 0 {
		return nil, false
	}
	stashes, open := app.top().(*stashScreen)
	return stashes, open
}

func (stashes *stashScreen) rowText(stash git.StashRef) string {
	return fmt.Sprintf(stashRowFormat, stash.Ref(), shortOID(stash.OID), stash.Date.Format(commitDateLayout), stash.Message)
}

func (stashes *stashScreen) visible() []git.StashRef {
	query := strings.ToLower(stashes.search.Value())
	if query == "" {
		return stashes.stashes
	}
	return slices.DeleteFunc(slices.Clone(stashes.stashes), func(stash git.StashRef) bool {
		return !strings.Contains(strings.ToLower(stashes.rowText(stash)), query)
	})
}

func (stashes *stashScreen) focusedIndex(rows []git.StashRef) int {
	return slices.IndexFunc(rows, func(stash git.StashRef) bool { return stash.OID == stashes.focusOID })
}

func (stashes *stashScreen) ID() ScreenID { return ScreenStashes }

func (stashes *stashScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case stashesLoadedMsg:
		stashes.stashes, stashes.err, stashes.loaded = msg.stashes, msg.err, true
		if rows := stashes.visible(); stashes.focusedIndex(rows) < 0 && len(rows) > 0 {
			stashes.focusOID = rows[0].OID
		}
		return nil, true
	case tea.KeyMsg:
		keys := msg.String()
		rows := stashes.visible()
		cursor := listCursor{index: max(0, stashes.focusedIndex(rows))}
		switch {
		case keys == keyEscape && (stashes.searching || stashes.search.Value() != ""):
			stashes.searching = false
			stashes.search.Blur()
			stashes.search.SetValue("")
		case keys == keyEscape:
			return nil, false
		case keys == keyEnter && stashes.searching:
			stashes.searching = false
			stashes.search.Blur()
		case len(rows) > 0 && (!stashes.searching || msg.Type == tea.KeyUp || msg.Type == tea.KeyDown) && cursor.navigate(keys, len(rows), app.height/2):
			stashes.focusOID = rows[cursor.index].OID
		case stashes.searching:
			var cmd tea.Cmd
			stashes.search, cmd = stashes.search.Update(msg)
			if filtered := stashes.visible(); stashes.focusedIndex(filtered) < 0 && len(filtered) > 0 {
				stashes.focusOID = filtered[0].OID
			}
			return cmd, true
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

func (stashes *stashScreen) View(width, height int) string {
	lines := []string{appStyles.header.Render(fmt.Sprintf(stashesTitle, len(stashes.stashes))), appStyles.disabled.Render("/ " + stashFilterLabel)}
	if stashes.searching || stashes.search.Value() != "" {
		lines[1] = stashes.search.View()
	}
	rows := stashes.visible()
	switch {
	case stashes.err != nil:
		lines = append(lines, appStyles.conflict.Render(stashes.err.Error()))
	case !stashes.loaded:
		lines = append(lines, stashesLoading)
	case len(stashes.stashes) == 0:
		lines = append(lines, appStyles.disabled.Render(noStashes))
	case len(rows) == 0:
		lines = append(lines, appStyles.disabled.Render(noStashesMatch))
	}
	listHeight := max(1, height-len(lines))
	focusedIndex := stashes.focusedIndex(rows)
	firstRow := listCursor{index: focusedIndex}.windowStart(len(rows), listHeight)
	for row := firstRow; row < min(len(rows), firstRow+listHeight); row++ {
		line := stashes.rowText(rows[row])
		if row == focusedIndex {
			line = appStyles.selected.Render(line)
		}
		lines = append(lines, line)
	}
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(lines, "\n"))
}

func (stashes *stashScreen) Scope() Scope { return ScopeStash }
