package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/prefs"
)

const (
	paletteQueryPrompt  = ": "
	reasonStatusLoading = "Status loading"
	keyConfirm          = "y"
	keyConfirmGuarded   = "c"
	noVerifyTitle       = "Commit once without verification hooks"
	amendTitle          = "Amend HEAD with the reviewed index and message"
)

var (
	noVerifyCommand      = []string{"git", "commit", "-F", "-", "--no-verify"}
	amendCommand         = []string{"git", "commit", "-F", "-", "--amend"}
	switchCommand        = []string{"git", "switch"}
	deleteBranchCommand  = []string{"git", "branch", "-d"}
	noVerifyConsequences = []string{"Skips the pre-commit and commit-msg hooks for this one commit only", "prepare-commit-msg, post-commit, signing and GitFlow checks still run", "The next commit runs every hook again"}
	amendConsequences    = []string{"Replaces HEAD with the reviewed staged index and draft message", "GitFlow never pushes"}
)

const (
	actionOpenHistory        ActionID = "history.open"
	actionOpenBranches       ActionID = "branches.open"
	actionOpenStashes        ActionID = "stashes.open"
	actionOpenCherryPick     ActionID = "cherry-pick.open"
	actionOpenPaused         ActionID = "operation.open"
	actionOpenActions        ActionID = "actions.open"
	actionOpenGitMenu        ActionID = "git-menu.open"
	actionOpenPalette        ActionID = "palette.open"
	actionOpenHelp           ActionID = "help.open"
	actionSearch             ActionID = "search"
	actionBack               ActionID = "back"
	actionPullConfigured     ActionID = "sync.pull"
	actionPullRebase         ActionID = "sync.pull-rebase"
	actionPullFastForward    ActionID = "sync.pull-ff-only"
	actionPush               ActionID = "sync.push"
	actionForcePush          ActionID = "sync.force-with-lease"
	actionFetch              ActionID = "sync.fetch"
	actionCreateBranch       ActionID = "branch.create"
	actionCreateFromRef      ActionID = "branch.create-from-ref"
	actionCreateFromMain     ActionID = "branch.create-from-main"
	actionSetUpstream        ActionID = "branch.set-upstream"
	actionSwitchBranch       ActionID = "branch.switch"
	actionStashAndSwitch     ActionID = "branch.stash-and-switch"
	actionDeleteBranch       ActionID = "branch.delete"
	actionFilterAll          ActionID = "changes.filter-all"
	actionFilterStaged       ActionID = "changes.filter-staged"
	actionFilterNotStaged    ActionID = "changes.filter-not-staged"
	actionFilterMixed        ActionID = "changes.filter-mixed"
	actionStage              ActionID = "changes.stage"
	actionUnstage            ActionID = "changes.unstage"
	actionDiscard            ActionID = "changes.discard"
	actionDeleteFile         ActionID = "changes.delete-file"
	actionToggleLayout       ActionID = "changes.toggle-layout"
	actionSelectAll          ActionID = "changes.select-all"
	actionFocusComposer      ActionID = "changes.focus-composer"
	actionCommit             ActionID = "commit.commit"
	actionCommitNoVerify     ActionID = "commit.no-verify"
	actionAmend              ActionID = "commit.amend"
	actionCherryPickPreview  ActionID = "cherry-pick.preview"
	actionCherryPickExecute  ActionID = "cherry-pick.execute"
	actionStashAndProceed    ActionID = "cherry-pick.stash-and-proceed"
	actionContinueUnstash    ActionID = "cherry-pick.continue-unstash"
	actionAbortUnstash       ActionID = "cherry-pick.abort-unstash"
	actionRebaseStart        ActionID = "rebase.start"
	actionRebasePreview      ActionID = "rebase.preview"
	actionRebaseExecute      ActionID = "rebase.execute"
	actionRebasePick         ActionID = "rebase.pick"
	actionRebaseReword       ActionID = "rebase.reword"
	actionRebaseEdit         ActionID = "rebase.edit"
	actionRebaseSquash       ActionID = "rebase.squash"
	actionRebaseFixup        ActionID = "rebase.fixup"
	actionRebaseDrop         ActionID = "rebase.drop"
	actionCherryPickMoveDown ActionID = "cherry-pick.move-down"
	actionCherryPickMoveUp   ActionID = "cherry-pick.move-up"
	actionRebaseMoveDown     ActionID = "rebase.move-down"
	actionRebaseMoveUp       ActionID = "rebase.move-up"
	actionApplyPreview       ActionID = "selective-apply.preview"
	actionApplyExecute       ActionID = "selective-apply.execute"
	actionCompareCommits     ActionID = "history.compare"
	actionOperationContinue  ActionID = "operation.continue"
	actionOperationSkip      ActionID = "operation.skip"
	actionOperationAbort     ActionID = "operation.abort"
	actionOperationRefresh   ActionID = "operation.refresh"
	actionConflictEditor     ActionID = "conflict.editor"
	actionConflictResolved   ActionID = "conflict.mark-resolved"
	actionConflictBase       ActionID = "conflict.choose-base"
	actionConflictOurs       ActionID = "conflict.choose-ours"
	actionConflictTheirs     ActionID = "conflict.choose-theirs"
	actionConflictNext       ActionID = "conflict.next-region"
	actionConflictPrevious   ActionID = "conflict.previous-region"
)

func defaultActions() []Action {
	return []Action{
		{ID: actionOpenHistory, Title: "History", Scope: ScopeGlobal, Keys: []string{"g h"}, Run: func(app *App) tea.Cmd {
			if history, open := app.historyOnTop(); open {
				return history.reload(app)
			}
			if app.history != nil {
				app.push(app.history)
				if app.history.loading {
					return nil
				}
				return app.history.loadNext(app)
			}
			search := textinput.New()
			search.Prompt = historyFilterPrompt
			app.history = &historyScreen{label: historyLabel, sourceRef: headRef, search: search}
			app.push(app.history)
			return app.history.reload(app)
		}},
		{ID: actionOpenBranches, Title: "Branches", Scope: ScopeGlobal, Keys: []string{"g b"}, Aliases: []Binding{{Scope: ScopeChanges, Keys: []string{"b"}}}, Run: func(app *App) tea.Cmd {
			if branches, open := app.branchesOnTop(); open {
				return branches.load(app)
			}
			search := textinput.New()
			search.Prompt = branchSearchPrompt
			branches := &branchScreen{search: search}
			app.push(branches)
			return branches.load(app)
		}},
		{ID: actionOpenStashes, Title: "Stash list", Scope: ScopeGlobal, Keys: []string{"g s"}, Run: func(app *App) tea.Cmd {
			if app.stashes == nil {
				search := textinput.New()
				search.Prompt = stashFilterPrompt
				app.stashes = &stashScreen{search: search}
			}
			if _, open := app.stashesOnTop(); !open {
				app.push(app.stashes)
			}
			return func() tea.Msg {
				stashes, err := app.deps.Stasher.List(app.ctx)
				return stashesLoadedMsg{stashes: stashes, err: err}
			}
		}},
		{ID: actionOpenCherryPick, Title: "Cherry-pick explorer", Scope: ScopeGlobal, Keys: []string{"g c"}, Aliases: []Binding{{Scope: ScopeHistory}}, Run: func(app *App) tea.Cmd {
			if app.cherryPick == nil {
				search := textinput.New()
				search.Prompt = historyFilterPrompt
				sourceRef := app.repository.baseRef
				if sourceRef == "" {
					sourceRef = headRef
				}
				app.cherryPick = &cherryPickScreen{app: app}
				app.cherryPick.commits = &historyScreen{label: cherryPickSourceLabel, sourceRef: sourceRef, explorer: app.cherryPick, search: search}
			}
			explorer, commits := app.cherryPick, app.cherryPick.commits
			if history, fromHistory := app.historyOnTop(); fromHistory && history.cursor.SourceRef != "" {
				commits.sourceRef, commits.focusOID = history.cursor.SourceRef, ""
				commits.selection = commits.selection.With(history.selection.Ordered()...)
				if commits.selected == nil {
					commits.selected = map[string]selectedCommit{}
				}
				for _, oid := range history.selection.Ordered() {
					commits.selected[oid] = history.selected[oid]
				}
				commits.files = commits.files.With(history.files.Ordered()...)
				if commits.fileSources == nil {
					commits.fileSources = map[string]selectedCommit{}
				}
				for _, selection := range history.files.Ordered() {
					commits.fileSources[selection.OID] = history.fileSources[selection.OID]
				}
			}
			if app.top() != explorer {
				app.push(explorer)
			}
			return tea.Batch(commits.reload(app), func() tea.Msg {
				list, err := app.deps.Branches.List(app.ctx, true)
				return branchesLoadedMsg{branches: list, err: err}
			})
		}},
		{ID: actionOpenPaused, Title: "Paused Operation Dashboard", Scope: ScopeGlobal, Keys: []string{"o"},
			Available: func(app *App) (bool, string) {
				if !app.repository.operation.Active() {
					return false, reasonNoPausedOperation
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				if _, open := app.top().(*pausedScreen); !open {
					app.push(&pausedScreen{app: app})
				}
				return nil
			}},
		{ID: actionOpenActions, Title: "Actions", Scope: ScopeGlobal, Keys: []string{"a"}, Run: func(app *App) tea.Cmd {
			scope := app.top().Scope()
			app.overlay = &actionsOverlay{scope: scope, entries: app.menuEntries(scope, app.registry.InScope(scope))}
			return nil
		}},
		{ID: actionOpenGitMenu, Title: "Git menu", Scope: ScopeGlobal, Keys: []string{"g"}, Run: func(app *App) tea.Cmd {
			var prefixed []*Action
			for _, action := range app.registry.InScope(ScopeGlobal) {
				for _, keys := range action.Keys {
					if strings.HasPrefix(keys, globalPrefix) {
						prefixed = append(prefixed, action)
						break
					}
				}
			}
			app.overlay = &gitMenuOverlay{entries: app.menuEntries(ScopeGlobal, prefixed)}
			return nil
		}},
		{ID: actionOpenPalette, Title: "Command palette", Scope: ScopeGlobal, Keys: []string{":"}, Run: func(app *App) tea.Cmd {
			query := textinput.New()
			query.Prompt = paletteQueryPrompt
			palette := &paletteOverlay{query: query, entries: app.menuEntries(ScopeGlobal, app.registry.Search(""))}
			app.overlay = palette
			return palette.query.Focus()
		}},
		{ID: actionOpenHelp, Title: "Help", Scope: ScopeGlobal, Keys: []string{"?"}, Run: func(app *App) tea.Cmd {
			scope := app.top().Scope()
			entries := append(app.menuEntries(scope, app.registry.InScope(scope)), app.menuEntries(ScopeGlobal, app.registry.InScope(ScopeGlobal))...)
			app.overlay = &paletteOverlay{helpScope: scope, entries: entries}
			return nil
		}},
		{ID: actionSearch, Title: "Search", Scope: ScopeGlobal, Keys: []string{"/"},
			Available: func(app *App) (bool, string) {
				_, branchesOpen := app.branchesOnTop()
				_, historyOpen := app.historyOnTop()
				_, stashesOpen := app.stashesOnTop()
				explorerOpen := app.cherryPick != nil && app.top() == app.cherryPick
				if !branchesOpen && !historyOpen && !stashesOpen && !explorerOpen {
					return false, reasonNotAvailableYet
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				if history, open := app.historyOnTop(); open {
					history.searching, history.detailFocused = true, false
					return history.search.Focus()
				}
				if explorer := app.cherryPick; explorer != nil && app.top() == explorer {
					explorer.branchesFocused, explorer.commits.searching, explorer.commits.detailFocused = false, true, false
					return explorer.commits.search.Focus()
				}
				if stashes, open := app.stashesOnTop(); open {
					stashes.searching = true
					return stashes.search.Focus()
				}
				branches, _ := app.branchesOnTop()
				branches.searching, branches.inspecting = true, false
				return branches.search.Focus()
			}},
		{ID: actionBack, Title: "Back / Quit", Scope: ScopeGlobal, Keys: []string{"q"}, Run: func(app *App) tea.Cmd {
			if len(app.stack) == 1 {
				return tea.Quit
			}
			app.pop()
			return nil
		}},
		{ID: actionPullConfigured, Title: pullTitle, Scope: ScopeGlobal, Keys: []string{"g p"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.syncAvailability(true) },
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(pullTitle, pullCommand, func(ctx context.Context) (string, func(app *App), error) {
					summary, err := app.deps.Remote.Pull(ctx, git.PullConfigured)
					return summary, nil, err
				})
			}},
		{ID: actionPullRebase, Title: pullRebaseTitle, Scope: ScopeGlobal, Keys: []string{"g r"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.syncAvailability(true) },
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(pullRebaseTitle, pullRebaseCommand, func(ctx context.Context) (string, func(app *App), error) {
					summary, err := app.deps.Remote.Pull(ctx, git.PullRebase)
					if err == nil {
						return summary, nil, nil
					}
					operation, activeErr := app.deps.Operations.Active(ctx)
					if activeErr != nil {
						return "", nil, errors.Join(err, activeErr)
					}
					if operation.Kind != git.OperationRebase {
						return "", nil, err
					}
					return "", func(app *App) { app.push(&pausedScreen{app: app}) }, err
				})
			}},
		{ID: actionPullFastForward, Title: pullFastForwardTitle, Scope: ScopeGlobal, Keys: []string{"g f"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.syncAvailability(true) },
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(pullFastForwardTitle, pullFastForwardCommand, func(ctx context.Context) (string, func(app *App), error) {
					summary, err := app.deps.Remote.Pull(ctx, git.PullFastForwardOnly)
					return summary, nil, err
				})
			}},
		{ID: actionPush, Title: pushTitle, Scope: ScopeGlobal, Keys: []string{"g P"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.syncAvailability(false) },
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(pushTitle, pushCommand, func(ctx context.Context) (string, func(app *App), error) {
					target, err := app.deps.Remote.PushTarget(ctx)
					if err != nil {
						return "", nil, err
					}
					if !target.FirstPush {
						summary, err := app.deps.Remote.Push(ctx, target)
						return summary, nil, err
					}
					command := slices.Concat([]string{gitProgram}, target.PushArgs())
					return "", func(app *App) {
						remoteBranch := target.Remote + "/" + target.Branch
						destination := fmt.Sprintf(firstPushUnknown, remoteBranch)
						switch target.State {
						case git.DestinationExists:
							destination = fmt.Sprintf(firstPushExists, remoteBranch, target.CachedOID, target.Tracking.Ahead, target.Tracking.Behind)
						case git.DestinationAbsent:
							destination = fmt.Sprintf(firstPushAbsent, remoteBranch, freshnessLabel(app.repository.freshness))
						}
						app.overlay = &confirmOverlay{title: fmt.Sprintf(firstPushTitle, target.Branch), command: command,
							consequences: []string{destination, fmt.Sprintf(firstPushLocal, target.Branch, target.LocalOID), firstPushTracking}, confirmKey: keyConfirmGuarded,
							onConfirm: func(app *App) tea.Cmd {
								return app.syncRemote(pushTitle, command, func(ctx context.Context) (string, func(app *App), error) {
									summary, err := app.deps.Remote.Push(ctx, target)
									return summary, nil, err
								})
							}}
					}, nil
				})
			}},
		{ID: actionForcePush, Title: forcePushTitle, Scope: ScopeGlobal, Keys: []string{"g F"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.syncAvailability(true) },
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(forcePushTitle, pushCommand, func(ctx context.Context) (string, func(app *App), error) {
					target, err := app.deps.Remote.PushTarget(ctx)
					if err != nil {
						return "", nil, err
					}
					if err := target.LeaseBlocked(); err != nil {
						return "", nil, err
					}
					command := slices.Concat([]string{gitProgram}, target.ForceArgs())
					return "", func(app *App) {
						app.overlay = &forceOverlay{&confirmOverlay{title: fmt.Sprintf(forceReviewTitle, target.Branch), command: command,
							consequences: []string{fmt.Sprintf(forceDestination, target.Remote, target.RemoteRef, target.CachedOID, target.LocalOID), forceRewrite, forceStale},
							confirmKey:   keyConfirmGuarded,
							onConfirm: func(app *App) tea.Cmd {
								return app.syncRemote(forcePushTitle, command, func(ctx context.Context) (string, func(app *App), error) {
									summary, err := app.deps.Remote.ForceWithLease(ctx, target)
									return summary, nil, err
								})
							}}}
					}, nil
				})
			}},
		{ID: actionFetch, Title: fetchTitle, Scope: ScopeGlobal, Keys: []string{"g e"}, Refresh: RefreshAll,
			Run: func(app *App) tea.Cmd {
				return app.syncRemote(fetchTitle, fetchCommand, func(ctx context.Context) (string, func(app *App), error) {
					summary, err := app.deps.Remote.Fetch(ctx)
					return summary, nil, err
				})
			}},
		{ID: actionCreateBranch, Title: "Create local branch", Scope: ScopeGlobal, Keys: []string{"g n"}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				switch {
				case !app.loaded:
					return false, reasonStatusLoading
				case app.repository.status.Unborn:
					return false, reasonNoCommits
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				var cmd tea.Cmd
				app.overlay, cmd = newNameInput(headRef, app.repository.status.Head)
				return cmd
			}},
		{ID: actionCreateFromRef, Title: "Create local branch from focused ref", Scope: ScopeBranch, Keys: []string{"a n"}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				if _, found := app.focusedBranch(); !found {
					return false, reasonNoBranchFocused
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				focused, _ := app.focusedBranch()
				var cmd tea.Cmd
				app.overlay, cmd = newNameInput(focused.Ref, focused.OID)
				return cmd
			}},
		{ID: actionSetUpstream, Title: "Set upstream", Scope: ScopeGlobal, Keys: []string{"g u"}, Aliases: []Binding{{Scope: ScopeBranch, Keys: []string{"a u"}}}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return false, reasonUpstreamContractGap }},
		{ID: actionCreateFromMain, Title: "Create branch from main", Scope: ScopeBranch, Keys: []string{"a m"}, Review: ReviewConfirm, Refresh: RefreshAll},
		{ID: actionSwitchBranch, Title: "Switch branch", Scope: ScopeBranch, Keys: []string{"s"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				if _, found := app.focusedBranch(); !found {
					return false, reasonNoBranchFocused
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				focused, _ := app.focusedBranch()
				return app.mutate(func(ctx context.Context) (func(app *App), error) {
					var blocked *git.SwitchBlocked
					if err := app.deps.Branches.Switch(ctx, focused.Name, focused.OID); !errors.As(err, &blocked) {
						return nil, err
					}
					return func(app *App) {
						app.notice = ""
						review := &confirmOverlay{title: fmt.Sprintf(switchBlockedTitle, focused.Ref), command: slices.Concat(switchCommand, []string{focused.Name}),
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
				}, RefreshAll, fmt.Sprintf(switchedNotice, focused.Name))
			}},
		{ID: actionStashAndSwitch, Title: "Stash and switch", Scope: ScopeBranch, Review: ReviewPlan, Refresh: RefreshAll},
		{ID: actionDeleteBranch, Title: "Delete branch", Scope: ScopeBranch, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				focused, found := app.focusedBranch()
				switch {
				case !found:
					return false, reasonNoBranchFocused
				case focused.Remote:
					return false, reasonRemoteTrackingRef
				case focused.Current:
					return false, reasonBranchCheckedOut
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				focused, _ := app.focusedBranch()
				app.overlay = &confirmOverlay{title: fmt.Sprintf(deleteBranchTitle, focused.Name), command: slices.Concat(deleteBranchCommand, []string{focused.Name}),
					consequences: []string{deleteLocalOnly, deleteTipPrefix + focused.OID, deleteUnmergedWarning}, confirmKey: keyConfirm,
					onConfirm: func(app *App) tea.Cmd {
						return app.mutate(func(ctx context.Context) (func(app *App), error) {
							return nil, app.deps.Branches.Delete(ctx, focused.Name, focused.OID)
						}, RefreshAll, fmt.Sprintf(deletedNotice, focused.Name))
					}}
				return nil
			}},
		{ID: actionFilterAll, Title: "Filter: All", Scope: ScopeChanges, Keys: []string{"1"}, Run: func(app *App) tea.Cmd {
			app.changes.filter = git.FilterAll
			return app.changes.previewFocused(app)
		}},
		{ID: actionFilterStaged, Title: "Filter: Staged", Scope: ScopeChanges, Keys: []string{"2"}, Run: func(app *App) tea.Cmd {
			app.changes.filter = git.FilterStaged
			return app.changes.previewFocused(app)
		}},
		{ID: actionFilterNotStaged, Title: "Filter: Not staged", Scope: ScopeChanges, Keys: []string{"3"}, Run: func(app *App) tea.Cmd {
			app.changes.filter = git.FilterNotStaged
			return app.changes.previewFocused(app)
		}},
		{ID: actionFilterMixed, Title: "Filter: Mixed", Scope: ScopeChanges, Keys: []string{"4"}, Run: func(app *App) tea.Cmd {
			app.changes.filter = git.FilterMixed
			return app.changes.previewFocused(app)
		}},
		{ID: actionStage, Title: "Stage", Scope: ScopeChanges, Keys: []string{"s"}, Aliases: []Binding{{Scope: ScopeDiff, Keys: []string{"s"}}}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) { return app.indexAvailability(actionStage) },
			Run:       func(app *App) tea.Cmd { return app.runIndexAction(actionStage, app.deps.Indexer.Stage) }},
		{ID: actionUnstage, Title: "Unstage", Scope: ScopeChanges, Keys: []string{"u"}, Aliases: []Binding{{Scope: ScopeDiff, Keys: []string{"u"}}}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) { return app.indexAvailability(actionUnstage) },
			Run:       func(app *App) tea.Cmd { return app.runIndexAction(actionUnstage, app.deps.Indexer.Unstage) }},
		{ID: actionDiscard, Title: "Discard changes", Scope: ScopeChanges, Keys: []string{"d"}, Review: ReviewConfirm, Refresh: RefreshStatus},
		{ID: actionDeleteFile, Title: "Delete file from disk", Scope: ScopeChanges, Review: ReviewConfirm, Refresh: RefreshStatus},
		{ID: actionToggleLayout, Title: "Toggle flat/folder", Scope: ScopeChanges, Keys: []string{"v"}, Run: func(app *App) tea.Cmd {
			nextLayout := prefs.LayoutFolder
			if app.changes.layout == prefs.LayoutFolder {
				nextLayout = prefs.LayoutFlat
			}
			app.changes.layout = nextLayout
			if err := app.deps.Prefs.Save(prefs.Preferences{Layout: app.changes.layout}); err != nil {
				app.errorText = layoutNotSaved + ": " + err.Error()
			}
			return app.changes.previewFocused(app)
		}},
		{ID: actionSelectAll, Title: "Select all files", Scope: ScopeChanges, Keys: []string{"S"}, Run: func(app *App) tea.Cmd {
			for _, change := range app.repository.status.Changes {
				app.changes.checked = app.changes.checked.With(change.Path)
			}
			return nil
		}},
		{ID: actionFocusComposer, Title: "Commit composer", Scope: ScopeChanges, Keys: []string{"c"}, Aliases: []Binding{{Scope: ScopeComposer, Keys: []string{"c"}}}, Run: func(app *App) tea.Cmd {
			app.changes.focus, app.changes.review = focusComposer, commitReview{}
			return app.changes.draft.Focus()
		}},
		{ID: actionCommit, Title: "Commit", Scope: ScopeComposer, Review: ReviewPlan, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.commitAvailability(false) },
			Run:       func(app *App) tea.Cmd { return app.runCommit(false, false) }},
		{ID: actionCommitNoVerify, Title: "Commit without verification hooks", Scope: ScopeComposer, Keys: []string{"a v"}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.commitAvailability(false) },
			Run: func(app *App) tea.Cmd {
				app.overlay = &confirmOverlay{title: noVerifyTitle, command: noVerifyCommand, consequences: noVerifyConsequences, confirmKey: keyConfirm,
					onConfirm: func(app *App) tea.Cmd { return app.runCommit(true, false) }}
				return nil
			}},
		{ID: actionAmend, Title: "Amend last commit", Scope: ScopeComposer, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.commitAvailability(true) },
			Run: func(app *App) tea.Cmd {
				consequences := amendConsequences
				if app.changes.review.published {
					consequences = append([]string{publishedWarning}, amendConsequences...)
				}
				app.overlay = &confirmOverlay{title: amendTitle, command: amendCommand, consequences: consequences, confirmKey: keyConfirm,
					onConfirm: func(app *App) tea.Cmd { return app.runCommit(false, true) }}
				return nil
			}},
		{ID: actionCompareCommits, Title: "Compare two commits", Scope: ScopeHistory,
			Available: func(app *App) (bool, string) {
				history, open := app.historyOnTop()
				switch {
				case !open:
					return false, reasonHistoryClosed
				case history.selection.Len() != compareSelectionSize:
					return false, reasonSelectTwoCommits
				}
				return false, reasonCompareContractGap
			}},
		{ID: actionRebaseStart, Title: "Rebase onto base", Scope: ScopeHistory,
			Available: func(app *App) (bool, string) {
				history, open := app.historyOnTop()
				status := app.repository.status
				switch {
				case !open:
					return false, reasonHistoryClosed
				case history.focusOID == "":
					return false, reasonNoFocusedCommit
				case app.repository.operation.Active():
					return false, syncBlockerText[git.SyncOperation]
				case status.Unborn:
					return false, reasonNoCommits
				case status.Detached:
					return false, reasonDetachedHead
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				history, _ := app.historyOnTop()
				base := history.focusOID
				app.push(&rebaseScreen{app: app, history: history, plan: git.RebaseRange{Base: base}})
				return func() tea.Msg {
					plan, err := app.deps.Rebaser.Range(app.ctx, base)
					return rebaseRangeMsg{plan: plan, err: err}
				}
			}},
		{ID: actionCherryPickPreview, Title: "Preview", Scope: ScopeCherryPick, Keys: []string{"p"},
			Available: func(app *App) (bool, string) { return app.cherryPickQueueAvailability() },
			Run: func(app *App) tea.Cmd {
				commits, reviewedHead := app.cherryPick.queued(), app.repository.status.Head
				return func() tea.Msg {
					result, err := app.deps.Restore.Preflight(app.ctx, commits, reviewedHead)
					return cherryPickPreviewMsg{reviewedHead: reviewedHead, result: result, err: err}
				}
			}},
		{ID: actionCherryPickExecute, Title: cherryPickExecuteTitle, Scope: ScopeCherryPick, Review: ReviewPlan, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				if available, reason := app.cherryPickQueueAvailability(); !available {
					return false, reason
				}
				preview, status := app.cherryPick.preview, app.repository.status
				if preview == nil || preview.reviewedHead != status.Head || !slices.Equal(preview.result.Commits, app.cherryPick.commits.selection.Ordered()) {
					return false, reasonPreviewFirst
				}
				var blockers []string
				for _, blocker := range preview.result.Blockers {
					blockers = append(blockers, blocker.Reason())
				}
				switch {
				case len(blockers) > 0:
					return false, fmt.Sprintf(reasonPreviewBlocked, strings.Join(blockers, reasonListSeparator))
				case len(status.TrackedDirtyPaths) > 0:
					return false, fmt.Sprintf(reasonTrackedChanges, strings.Join(status.TrackedDirtyPaths, pathListSeparator))
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				explorer := app.cherryPick
				commits, reviewedHead, target := explorer.queued(), explorer.preview.reviewedHead, explorer.preview.result.Target
				return app.mutate(func(ctx context.Context) (func(app *App), error) {
					outcome, err := app.deps.Restore.Execute(ctx, commits, reviewedHead)
					if err != nil {
						return nil, err
					}
					return func(app *App) {
						explorer.commits.selection, explorer.commits.selected, explorer.preview = orderedSelection[string]{}, nil, nil
						if outcome.Paused {
							app.notice = fmt.Sprintf(cherryPickPausedNotice, len(outcome.Completed), len(commits))
							app.push(&pausedScreen{app: app})
						}
					}, nil
				}, RefreshAll, fmt.Sprintf(cherryPickedNotice, len(commits), target))
			}},
		{ID: actionStashAndProceed, Title: "Stash and proceed", Scope: ScopeCherryPick, Keys: []string{"s"}, Review: ReviewPlan, Refresh: RefreshAll},
		{ID: actionCherryPickMoveDown, Title: "Move commit down", Scope: ScopeCherryPick, Keys: []string{"J"},
			Available: func(app *App) (bool, string) { return app.cherryPickFocusQueued() },
			Run: func(app *App) tea.Cmd {
				commits := app.cherryPick.commits
				commits.selection = commits.selection.Move(commits.focusOID, 1)
				return nil
			}},
		{ID: actionCherryPickMoveUp, Title: "Move commit up", Scope: ScopeCherryPick, Keys: []string{"K"},
			Available: func(app *App) (bool, string) { return app.cherryPickFocusQueued() },
			Run: func(app *App) tea.Cmd {
				commits := app.cherryPick.commits
				commits.selection = commits.selection.Move(commits.focusOID, -1)
				return nil
			}},
		{ID: actionContinueUnstash, Title: "Continue unstash", Scope: ScopeStashRestore, Keys: []string{"c"}, Refresh: RefreshAll},
		{ID: actionAbortUnstash, Title: "Abort unstash", Scope: ScopeStashRestore, Keys: []string{"a"}, Review: ReviewConfirm, Refresh: RefreshAll},
		{ID: actionRebasePreview, Title: "Preview", Scope: ScopeRebase, Keys: []string{"v"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run: func(app *App) tea.Cmd {
				rebase, _ := app.rebaseOnTop()
				reviewed := rebase.plan
				reviewed.Steps = slices.Clone(reviewed.Steps)
				return func() tea.Msg {
					published, err := app.deps.Rebaser.Published(app.ctx, reviewed)
					return rebasePreviewMsg{preview: rebasePreview{reviewed: reviewed, published: published, blocker: app.deps.Rebaser.Validate(reviewed.Steps)}, err: err}
				}
			}},
		{ID: actionRebasePick, Title: "Pick", Scope: ScopeRebase, Keys: []string{"p"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoPick) }},
		{ID: actionRebaseReword, Title: "Reword", Scope: ScopeRebase, Keys: []string{"r"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoReword) }},
		{ID: actionRebaseEdit, Title: "Edit", Scope: ScopeRebase, Keys: []string{"e"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoEdit) }},
		{ID: actionRebaseSquash, Title: "Squash", Scope: ScopeRebase, Keys: []string{"s"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoSquash) }},
		{ID: actionRebaseFixup, Title: "Fixup", Scope: ScopeRebase, Keys: []string{"f"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoFixup) }},
		{ID: actionRebaseDrop, Title: "Drop", Scope: ScopeRebase, Keys: []string{"d"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.setTodoAction(git.TodoDrop) }},
		{ID: actionRebaseMoveDown, Title: "Move todo down", Scope: ScopeRebase, Keys: []string{"J"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.moveTodo(1) }},
		{ID: actionRebaseMoveUp, Title: "Move todo up", Scope: ScopeRebase, Keys: []string{"K"},
			Available: func(app *App) (bool, string) { return app.rebasePlanAvailability() },
			Run:       func(app *App) tea.Cmd { return app.moveTodo(-1) }},
		{ID: actionRebaseExecute, Title: rebaseExecuteTitle, Scope: ScopeRebase, Review: ReviewPlan, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) {
				if available, reason := app.rebasePlanAvailability(); !available {
					return false, reason
				}
				rebase, _ := app.rebaseOnTop()
				preview, status := rebase.preview, app.repository.status
				switch {
				case rebase.plan.Head != status.Head:
					return false, reasonPlanStale
				case preview == nil || !slices.Equal(preview.reviewed.Steps, rebase.plan.Steps):
					return false, reasonPreviewPlan
				case preview.blocker != nil:
					return false, fmt.Sprintf(reasonPreviewBlocked, preview.blocker.Error())
				case app.repository.operation.Active():
					return false, syncBlockerText[git.SyncOperation]
				case !status.TrackedClean():
					return false, fmt.Sprintf(reasonTrackedChanges, strings.Join(append(slices.Clone(status.TrackedDirtyPaths), status.Unmerged...), pathListSeparator))
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				rebase, _ := app.rebaseOnTop()
				reviewed, published := rebase.preview.reviewed, rebase.preview.published
				start := func(app *App) tea.Cmd {
					return app.runBusy(func() tea.Msg {
						run, err := app.deps.Rebaser.Prepare(app.ctx, reviewed)
						if err != nil {
							return mutationDoneMsg{refresh: RefreshAll, err: err}
						}
						return handoffReadyMsg{command: run.Command, finish: func(exitErr error) tea.Msg {
							outcome, err := app.deps.Rebaser.Finish(app.ctx, run)
							if err == nil && exitErr != nil && !outcome.Paused {
								err = fmt.Errorf(handoffFailed, gitProgram, exitErr)
							}
							return rebaseFinishedMsg{outcome: outcome, err: err}
						}}
					})
				}
				if len(published) == 0 {
					return start(app)
				}
				app.overlay = &confirmOverlay{title: rebasePublishedTitle, command: append(slices.Clone(rebaseCommand), reviewed.Base),
					consequences: []string{fmt.Sprintf(rebasePublishedRefs, strings.Join(published, pathListSeparator)), rebaseNoPush}, confirmKey: keyConfirmGuarded, hints: rebasePublishedHints, onConfirm: start}
				return nil
			}},
		{ID: actionApplyPreview, Title: "Preview", Scope: ScopeSelectiveApply, Keys: []string{"p"}, Aliases: []Binding{{Scope: ScopeCommitFiles, Keys: []string{"p"}}},
			Available: func(app *App) (bool, string) {
				_, available, reason := app.selectiveApplyAvailability()
				return available, reason
			},
			Run: func(app *App) tea.Cmd {
				history, _, _ := app.selectiveApplyAvailability()
				planner, open := app.top().(*selectiveApplyScreen)
				if !open {
					planner = &selectiveApplyScreen{app: app, history: history}
					app.push(planner)
				}
				selections, reviewedHead := history.orderedFiles(), app.repository.status.Head
				return func() tea.Msg {
					var steps []git.ApplyStep
					for _, selection := range selections {
						patch, err := app.deps.Differ.FilePatch(app.ctx, selection)
						if err != nil {
							return selectiveApplyPreviewMsg{screen: planner, err: fmt.Errorf("build patch for %s from %s: %w", selection.Path, shortOID(selection.OID), err)}
						}
						steps = append(steps, git.ApplyStep{Selection: selection, Patch: patch})
					}
					result, err := app.deps.SelectiveApply.Preflight(app.ctx, steps, reviewedHead)
					if err != nil {
						return selectiveApplyPreviewMsg{screen: planner, err: fmt.Errorf("check selected files: %w", err)}
					}
					return selectiveApplyPreviewMsg{screen: planner, reviewedHead: reviewedHead, steps: steps, result: result}
				}
			}},
		{ID: actionApplyExecute, Title: applyExecuteTitle, Scope: ScopeSelectiveApply, Review: ReviewPlan, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) {
				history, available, reason := app.selectiveApplyAvailability()
				if !available {
					return false, reason
				}
				planner, open := app.top().(*selectiveApplyScreen)
				status := app.repository.status
				if !open || planner.preview == nil || planner.report != nil || planner.preview.reviewedHead != status.Head {
					return false, reasonPreviewFiles
				}
				var planned []git.FileSelection
				for _, step := range planner.steps {
					planned = append(planned, step.Selection)
				}
				if !slices.Equal(planned, history.orderedFiles()) {
					return false, reasonPreviewFiles
				}
				var blockers []string
				for _, blocker := range planner.preview.result.Blockers {
					blockers = append(blockers, blocker.Reason())
				}
				switch {
				case len(blockers) > 0:
					return false, fmt.Sprintf(reasonPreviewBlocked, strings.Join(blockers, reasonListSeparator))
				case len(status.TrackedDirtyPaths) > 0:
					return false, fmt.Sprintf(reasonTrackedChanges, strings.Join(status.TrackedDirtyPaths, pathListSeparator))
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				planner, open := app.top().(*selectiveApplyScreen)
				if !open || planner.preview == nil {
					return nil
				}
				steps, reviewedHead, target := planner.steps, planner.preview.reviewedHead, planner.preview.result.Target
				return app.mutate(func(ctx context.Context) (func(app *App), error) {
					report, err := app.deps.SelectiveApply.Execute(ctx, steps, reviewedHead)
					return func(app *App) {
						planner.report, planner.applyErr = &report, err
						if err != nil || len(report.Applied) == 0 {
							return
						}
						history := planner.history
						var groups []string
						paths := map[string][]string{}
						for _, selection := range report.Applied {
							history.files = history.files.Without(selection)
							if _, seen := paths[selection.OID]; !seen {
								groups = append(groups, selection.OID)
							}
							paths[selection.OID] = append(paths[selection.OID], fileListBullet+selection.Path)
						}
						subject := fmt.Sprintf(applyMessageSources, len(groups))
						lines := []string{subject, ""}
						if len(groups) == 1 {
							subject = fmt.Sprintf(applyMessageOneSource, shortOID(groups[0]), history.fileSources[groups[0]].summary.Subject)
							lines = []string{subject, "", applyMessageSourceLine + groups[0]}
						}
						for _, oid := range groups {
							if len(groups) > 1 {
								lines = append(lines, oid+" "+history.fileSources[oid].summary.Subject)
							}
							lines = append(lines, paths[oid]...)
						}
						message := strings.Join(lines, "\n")
						if strings.TrimSpace(app.changes.draft.Value()) == "" {
							app.changes.draft.SetValue(message)
							app.notice += headerSeparator + messagePreparedNotice
							return
						}
						app.overlay = &confirmOverlay{title: replaceDraftTitle, consequences: []string{replaceDraftKeeps, fmt.Sprintf(replaceDraftWith, subject)}, confirmKey: keyConfirmGuarded, hints: replaceDraftHints,
							onConfirm: func(app *App) tea.Cmd {
								app.changes.draft.SetValue(message)
								app.notice = messagePreparedNotice
								return nil
							}}
					}, err
				}, RefreshStatus, fmt.Sprintf(appliedNotice, len(steps), target))
			}},
		{ID: actionOperationContinue, Title: "Continue", Scope: ScopeOperation, Keys: []string{"c"}, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.operationControlAvailability(actionOperationContinue) },
			Run: func(app *App) tea.Cmd {
				kind := app.repository.operation.Kind
				return app.mutate(func(ctx context.Context) (func(app *App), error) {
					return nil, app.deps.Controls.Continue(ctx, kind)
				}, RefreshAll, fmt.Sprintf(continuedNotice, operationNames[kind]))
			}},
		{ID: actionOperationSkip, Title: "Skip", Scope: ScopeOperation, Keys: []string{"s"}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.operationControlAvailability(actionOperationSkip) },
			Run: func(app *App) tea.Cmd {
				operation := app.repository.operation
				name := operationNames[operation.Kind]
				app.overlay = &confirmOverlay{title: fmt.Sprintf(skipTitle, name, shortOID(operation.Commit)), command: []string{"git", name, "--skip"},
					consequences: []string{fmt.Sprintf(skipDropsCommit, name), skipMayStopAgain, operationCancelDefault}, confirmKey: keyConfirmGuarded,
					onConfirm: func(app *App) tea.Cmd {
						return app.mutate(func(ctx context.Context) (func(app *App), error) {
							return nil, app.deps.Controls.Skip(ctx, operation.Kind)
						}, RefreshAll, fmt.Sprintf(skippedNotice, name))
					}}
				return nil
			}},
		{ID: actionOperationAbort, Title: "Abort", Scope: ScopeOperation, Keys: []string{"A"}, Review: ReviewConfirm, Refresh: RefreshAll,
			Available: func(app *App) (bool, string) { return app.operationControlAvailability(actionOperationAbort) },
			Run: func(app *App) tea.Cmd {
				operation := app.repository.operation
				name := operationNames[operation.Kind]
				app.overlay = &confirmOverlay{title: fmt.Sprintf(abortTitle, name), command: []string{"git", name, "--abort"},
					consequences: []string{fmt.Sprintf(abortRestoresStart, name), fmt.Sprintf(abortDiscardsResolutions, name), abortNotUndo, operationCancelDefault}, confirmKey: keyConfirmGuarded,
					onConfirm: func(app *App) tea.Cmd {
						return app.mutate(func(ctx context.Context) (func(app *App), error) {
							return nil, app.deps.Controls.Abort(ctx, operation.Kind)
						}, RefreshAll, fmt.Sprintf(abortedNotice, name))
					}}
				return nil
			}},
		{ID: actionOperationRefresh, Title: "Refresh", Scope: ScopeOperation, Keys: []string{"r"}, Refresh: RefreshAll,
			Run: func(app *App) tea.Cmd { return app.refresh() }},
		{ID: actionConflictEditor, Title: "Open in editor", Scope: ScopeConflict, Keys: []string{"e"}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) {
				if _, open := app.conflictOnTop(); !open {
					return false, reasonResolverNotLoaded
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				conflict, _ := app.conflictOnTop()
				path := conflict.path
				return func() tea.Msg {
					command, err := app.deps.Conflicts.EditorCommand(app.ctx, path)
					return handoffReadyMsg{command: command, err: err}
				}
			}},
		{ID: actionConflictResolved, Title: "Mark resolved", Scope: ScopeConflict, Review: ReviewConfirm, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) {
				conflict, open := app.conflictOnTop()
				switch {
				case !open || !conflict.loaded || conflict.err != nil:
					return false, reasonResolverNotLoaded
				case len(conflict.file.Regions) > 0:
					return false, fmt.Sprintf(reasonMarkersRemain, len(conflict.file.Regions))
				}
				return true, ""
			},
			Run: func(app *App) tea.Cmd {
				conflict, _ := app.conflictOnTop()
				path, name := conflict.path, operationNames[conflict.operation.Kind]
				staged := markResolvedAsIs
				if conflict.file.Fallback == git.FallbackMissingFile {
					staged = markResolvedRemoval
				}
				app.overlay = &confirmOverlay{title: fmt.Sprintf(markResolvedTitle, path), command: []string{"git", "add", "--", path},
					consequences: []string{staged, fmt.Sprintf(markResolvedNoContinue, name)}, confirmKey: keyConfirm,
					onConfirm: func(app *App) tea.Cmd {
						return app.mutate(func(ctx context.Context) (func(app *App), error) {
							if err := app.deps.Conflicts.MarkResolved(ctx, path); err != nil {
								return nil, err
							}
							return func(app *App) {
								if resolved, open := app.conflictOnTop(); open && resolved.path == path {
									app.pop()
								}
							}, nil
						}, RefreshStatus, fmt.Sprintf(markResolvedNotice, path))
					}}
				return nil
			}},
		{ID: actionConflictOurs, Title: "Choose current side", Scope: ScopeConflict, Keys: []string{"1"}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) { return app.conflictChoiceAvailability(git.SideOurs) },
			Run:       func(app *App) tea.Cmd { return app.chooseConflictSide(git.SideOurs) }},
		{ID: actionConflictBase, Title: "Choose base", Scope: ScopeConflict, Keys: []string{"2"}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) { return app.conflictChoiceAvailability(git.SideBase) },
			Run:       func(app *App) tea.Cmd { return app.chooseConflictSide(git.SideBase) }},
		{ID: actionConflictTheirs, Title: "Choose incoming side", Scope: ScopeConflict, Keys: []string{"3"}, Refresh: RefreshStatus,
			Available: func(app *App) (bool, string) { return app.conflictChoiceAvailability(git.SideTheirs) },
			Run:       func(app *App) tea.Cmd { return app.chooseConflictSide(git.SideTheirs) }},
		{ID: actionConflictNext, Title: "Next region", Scope: ScopeConflict, Keys: []string{"n"},
			Available: func(app *App) (bool, string) { return app.conflictChoiceAvailability(git.SideOurs) },
			Run:       func(app *App) tea.Cmd { return app.moveConflictRegion(1) }},
		{ID: actionConflictPrevious, Title: "Previous region", Scope: ScopeConflict, Keys: []string{"N"},
			Available: func(app *App) (bool, string) { return app.conflictChoiceAvailability(git.SideOurs) },
			Run:       func(app *App) tea.Cmd { return app.moveConflictRegion(-1) }},
	}
}
