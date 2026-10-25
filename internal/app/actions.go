package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
)

const bulkConfirmThreshold = 20

func (a *App) screenOpened() tea.Cmd {
	if opener, ok := a.top().(interface{ opened(a *App) tea.Cmd }); ok {
		return opener.opened(a)
	}
	return nil
}

func (a *App) openScreen(next screen) tea.Cmd {
	a.push(next)
	return a.screenOpened()
}

func (a *App) indexMutation(label string, files []git.FileChange, hidden int, work func(ctx context.Context, files []git.FileChange) error) tea.Cmd {
	runner := a.runner
	run := func(a *App) tea.Cmd {
		return a.mutate(label, func(ctx context.Context) error {
			if _, err := git.Revalidate(ctx, runner, files); err != nil {
				return fmt.Errorf("%w: refreshed, reselect and retry", err)
			}
			return work(ctx, files)
		}, func(a *App, err error) tea.Cmd {
			if err == nil {
				for _, file := range files {
					delete(a.changes.selected, file.Path)
				}
				a.notice = fmt.Sprintf("%s: %d file(s); their UI selection was cleared", label, len(files))
			}
			return nil
		})
	}
	if hidden == 0 && len(files) <= bulkConfirmThreshold {
		return run(a)
	}
	var lines []string
	for _, file := range files {
		lines = append(lines, "  "+displayPath(file.Path))
	}
	a.overlay = &confirmOverlay{
		title:   fmt.Sprintf("%s %d file(s) (%d hidden by filter)?", label, len(files), hidden),
		lines:   lines,
		buttons: []confirmButton{{label: "Cancel"}, {label: label, run: run}},
	}
	return nil
}

func (a *App) changeTargets(predicate func(git.FileChange) bool) ([]git.FileChange, int, string) {
	targets, hidden := a.changes.targets(a)
	if len(targets) == 0 {
		return nil, 0, "no file focused or selected"
	}
	var matching []git.FileChange
	for _, file := range targets {
		if file.Conflicted() {
			return nil, 0, displayPath(file.Path) + " is unmerged: resolve it from the operation dashboard (o)"
		}
		if predicate(file) {
			matching = append(matching, file)
		}
	}
	if len(matching) == 0 {
		return nil, 0, "nothing in the target scope matches"
	}
	return matching, hidden, ""
}

func (a *App) diffPatchAction(staging bool) tea.Cmd {
	view := a.top().(*diffScreen)
	file := view.file
	pane, reverse := paneUnstaged, false
	if !staging {
		pane, reverse = paneStaged, true
	}
	diff := view.diff(pane)
	runner, unborn := a.runner, a.status.Branch.Unborn
	if diff.PartialUnsupportedReason() != "" || file.Kind == git.KindUntracked || len(diff.Hunks) == 0 {
		label := "Stage file"
		if !staging {
			label = "Unstage file"
		}
		return a.mutate(label, func(ctx context.Context) error {
			if _, err := git.Revalidate(ctx, runner, []git.FileChange{file}); err != nil {
				return err
			}
			if staging {
				return git.StageFiles(ctx, runner, []git.FileChange{file})
			}
			return git.UnstageFiles(ctx, runner, []git.FileChange{file}, unborn)
		}, nil)
	}
	selection, description := view.patchSelection(pane)
	patch, err := diff.BuildPatch(selection, reverse)
	if err != nil {
		a.errorText = err.Error()
		return nil
	}
	label := "Stage " + description
	if !staging {
		label = "Unstage " + description
	}
	displayed := diff.String()
	return a.mutate(label, func(ctx context.Context) error {
		current, err := git.LoadDiff(ctx, runner, file, !staging)
		if err != nil {
			return err
		}
		if current.String() != displayed {
			return fmt.Errorf("revalidate %s: %w: refreshed, reselect and retry", file.Path, git.ErrStale)
		}
		return git.ApplyToIndex(ctx, runner, patch, reverse)
	}, func(a *App, err error) tea.Cmd {
		view.selections[pane] = git.LineSelection{}
		return nil
	})
}

func actionDefinitions() []Action {
	operationActive := func(a *App) string {
		if !a.operation.Active() {
			return "no active operation"
		}
		return ""
	}
	cherryPick := func(a *App) *cherryPickScreen { return a.top().(*cherryPickScreen) }
	rebase := func(a *App) *rebaseScreen { return a.top().(*rebaseScreen) }
	conflict := func(a *App) *conflictScreen { return a.top().(*conflictScreen) }
	branches := func(a *App) *branchScreen { return a.top().(*branchScreen) }
	history := func(a *App) *historyScreen { return a.top().(*historyScreen) }
	rebaseAction := func(id, title, key string, action git.TodoAction) Action {
		return Action{ID: id, Title: title, Scope: ScopeRebase, Keys: []string{key}, Run: func(a *App) tea.Cmd { return rebase(a).setAction(action) }}
	}
	conflictChoice := func(id, title, key string, choice git.Choice) Action {
		return Action{ID: id, Title: title, Scope: ScopeConflict, Keys: []string{key},
			Available: func(a *App) string { return conflict(a).sideUnavailable() },
			Run:       func(a *App) tea.Cmd { return conflict(a).choose(a, choice) }}
	}
	notDetached := func(a *App) string {
		if a.status.Branch.Detached {
			return "detached HEAD: no current branch"
		}
		return ""
	}

	return []Action{
		{ID: "global.operation", Title: "Open operation dashboard", Scope: ScopeGlobal, Keys: []string{"o"},
			Run: func(a *App) tea.Cmd {
				if !a.operation.Active() {
					a.notice = "No active operation"
					return nil
				}
				if _, already := a.top().(*operationScreen); !already {
					a.push(&operationScreen{})
				}
				return nil
			}},
		{ID: "global.history", Title: "Open commit history", Scope: ScopeGlobal,
			Run: func(a *App) tea.Cmd {
				if a.history == nil {
					a.history = newHistoryScreen("HEAD")
				}
				return a.openScreen(a.history)
			}},
		{ID: "global.branch", Title: "Open branch inspector", Scope: ScopeGlobal,
			Run: func(a *App) tea.Cmd { return a.openScreen(newBranchScreen()) }},

		{ID: "changes.stage", Title: "Stage", Scope: ScopeChanges, Keys: []string{"s"}, Mutates: true,
			Available: func(a *App) string { _, _, reason := a.changeTargets(git.FileChange.Unstaged); return reason },
			Run: func(a *App) tea.Cmd {
				files, hidden, _ := a.changeTargets(git.FileChange.Unstaged)
				runner := a.runner
				return a.indexMutation("Stage", files, hidden, func(ctx context.Context, files []git.FileChange) error {
					return git.StageFiles(ctx, runner, files)
				})
			}},
		{ID: "changes.unstage", Title: "Unstage", Scope: ScopeChanges, Keys: []string{"u"}, Mutates: true,
			Available: func(a *App) string { _, _, reason := a.changeTargets(git.FileChange.Staged); return reason },
			Run: func(a *App) tea.Cmd {
				files, hidden, _ := a.changeTargets(git.FileChange.Staged)
				runner, unborn := a.runner, a.status.Branch.Unborn
				return a.indexMutation("Unstage", files, hidden, func(ctx context.Context, files []git.FileChange) error {
					return git.UnstageFiles(ctx, runner, files, unborn)
				})
			}},
		{ID: "changes.discard", Title: "Discard worktree changes", Scope: ScopeChanges, Keys: []string{"d"}, Mutates: true,
			Available: func(a *App) string {
				_, _, reason := a.changeTargets(func(file git.FileChange) bool { return file.Unstaged() && file.Kind != git.KindUntracked })
				return reason
			},
			Run: func(a *App) tea.Cmd {
				files, hidden, _ := a.changeTargets(func(file git.FileChange) bool { return file.Unstaged() && file.Kind != git.KindUntracked })
				lines := []string{"Working tree changes in these tracked files are replaced with the index version:"}
				var paths []string
				for _, file := range files {
					lines = append(lines, "  "+displayPath(file.Path))
					paths = append(paths, displayPath(file.Path))
				}
				lines = append(lines, "", fmt.Sprintf("%d hidden by filter are included.", hidden), "Staged content is kept. Untracked files are never discarded.", "This cannot be undone by GitFlow.")
				runner := a.runner
				a.overlay = &confirmOverlay{
					title:   fmt.Sprintf("Discard worktree changes in %d file(s)?", len(files)),
					lines:   lines,
					command: "git restore --worktree -- " + strings.Join(paths, " "),
					danger:  true,
					buttons: []confirmButton{{label: "Cancel"}, {label: "Discard", run: func(a *App) tea.Cmd {
						return a.indexMutation("Discard", files, 0, func(ctx context.Context, files []git.FileChange) error {
							return git.DiscardWorktreeChanges(ctx, runner, files)
						})
					}}},
				}
				return nil
			}},
		{ID: "changes.compose", Title: "Open commit composer", Scope: ScopeChanges, Keys: []string{"c c"},
			Run: func(a *App) tea.Cmd { return a.openScreen(newComposerScreen(a, composeCommit)) }},
		{ID: "changes.amend", Title: "Open amend composer", Scope: ScopeChanges, Keys: []string{"c a"},
			Available: func(a *App) string {
				if a.status.Branch.Unborn {
					return "no commit to amend"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return a.openScreen(newComposerScreen(a, composeAmend)) }},

		{ID: "diff.stage", Title: "Stage", Scope: ScopeDiff, Keys: []string{"s"}, Mutates: true,
			Available: func(a *App) string {
				view := a.top().(*diffScreen)
				switch {
				case view.file.Conflicted():
					return "unmerged: resolve from the operation dashboard (o)"
				case view.pane != paneUnstaged:
					return "focus the UNSTAGED section (Tab) to stage"
				case !view.file.Unstaged():
					return "no unstaged changes"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return a.diffPatchAction(true) }},
		{ID: "diff.unstage", Title: "Unstage", Scope: ScopeDiff, Keys: []string{"u"}, Mutates: true,
			Available: func(a *App) string {
				view := a.top().(*diffScreen)
				switch {
				case view.file.Conflicted():
					return "unmerged: resolve from the operation dashboard (o)"
				case view.pane != paneStaged:
					return "focus the STAGED section (Tab) to unstage"
				case !view.file.Staged():
					return "no staged changes"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return a.diffPatchAction(false) }},

		{ID: "composer.review", Title: "Review commit", Scope: ScopeComposer,
			Available: func(a *App) string { return a.top().(*composerScreen).reviewUnavailable(a) },
			Run:       func(a *App) tea.Cmd { return a.top().(*composerScreen).review(a) }},

		{ID: "history.cherrypick", Title: "Cherry-pick selected", Scope: ScopeHistory, Keys: []string{"c p"},
			Available: func(a *App) string {
				if len(history(a).targets(a)) == 0 {
					return "no commit focused or selected"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				return a.openScreen(&cherryPickScreen{oids: history(a).targets(a), loading: true, target: a.headLabel(), headOID: a.status.Branch.OID, details: map[string]string{}})
			}},
		{ID: "history.rebase", Title: "Interactive rebase onto focused commit", Scope: ScopeHistory, Keys: []string{"r i"},
			Available: func(a *App) string {
				if _, ok := history(a).focused(); !ok {
					return "focus the commit to use as base"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				commit, _ := history(a).focused()
				return a.openScreen(newRebaseScreen(a, commit.OID, commit.Short()+" "+commit.Subject))
			}},
		{ID: "history.compare", Title: "Compare selected pair", Scope: ScopeHistory, Keys: []string{"d c"},
			Available: func(a *App) string {
				if count := len(a.commitSelection.order); count != 2 {
					return fmt.Sprintf("select exactly 2 commits (%d selected)", count)
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				older, newer, runner := a.commitSelection.order[0], a.commitSelection.order[1], a.runner
				return a.openViewer("diff "+shortOID(older)+" "+shortOID(newer), func(ctx context.Context) (string, error) {
					return git.CompareCommits(ctx, runner, older, newer)
				})
			}},

		{ID: "cp.down", Title: "Move queue item down", Scope: ScopeCherryPick, Keys: []string{"J"},
			Run: func(a *App) tea.Cmd { return cherryPick(a).move(1) }},
		{ID: "cp.up", Title: "Move queue item up", Scope: ScopeCherryPick, Keys: []string{"K"},
			Run: func(a *App) tea.Cmd { return cherryPick(a).move(-1) }},
		{ID: "cp.preview", Title: "Preview plan", Scope: ScopeCherryPick, Keys: []string{"p"},
			Run: func(a *App) tea.Cmd { return cherryPick(a).preview(a) }},
		{ID: "cp.execute", Title: "Execute plan", Scope: ScopeCherryPick, Mutates: true,
			Available: func(a *App) string { return cherryPick(a).executeUnavailable(a) },
			Run:       func(a *App) tea.Cmd { return cherryPick(a).execute(a) }},

		{ID: "rb.down", Title: "Move todo row down", Scope: ScopeRebase, Keys: []string{"J"},
			Run: func(a *App) tea.Cmd { return rebase(a).move(1) }},
		{ID: "rb.up", Title: "Move todo row up", Scope: ScopeRebase, Keys: []string{"K"},
			Run: func(a *App) tea.Cmd { return rebase(a).move(-1) }},
		rebaseAction("rb.pick", "Pick", "p", git.TodoPick),
		rebaseAction("rb.reword", "Reword", "r", git.TodoReword),
		rebaseAction("rb.edit", "Edit", "e", git.TodoEdit),
		rebaseAction("rb.squash", "Squash", "s", git.TodoSquash),
		rebaseAction("rb.fixup", "Fixup", "f", git.TodoFixup),
		rebaseAction("rb.drop", "Drop", "d", git.TodoDrop),
		{ID: "rb.preview", Title: "Preview plan", Scope: ScopeRebase, Keys: []string{"v"},
			Run: func(a *App) tea.Cmd { return rebase(a).preview(a) }},
		{ID: "rb.execute", Title: "Execute plan", Scope: ScopeRebase, Mutates: true,
			Available: func(a *App) string { return rebase(a).executeUnavailable(a) },
			Run:       func(a *App) tea.Cmd { return rebase(a).execute(a) }},

		{ID: "op.continue", Title: "Continue", Scope: ScopeOperation, Keys: []string{"c"}, Mutates: true,
			Available: func(a *App) string {
				if reason := operationActive(a); reason != "" {
					return reason
				}
				if conflicts := len(a.conflictedFiles()); conflicts > 0 {
					return fmt.Sprintf("%d conflict(s) need resolution", conflicts)
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				operation, err := a.revalidateOperation()
				if err != nil {
					a.errorText = err.Error()
					return a.refresh()
				}
				if len(a.conflictedFiles()) > 0 {
					a.errorText = "Continue blocked after revalidation: unresolved files remain"
					return nil
				}
				label := "Continue " + operation.Kind.Name()
				return a.gitHandoff(label, operation.ContinueInvocation(), a.operationStepDone(label))
			}},
		{ID: "op.skip", Title: "Skip current commit", Scope: ScopeOperation, Keys: []string{"s"}, Mutates: true,
			Available: func(a *App) string {
				if reason := operationActive(a); reason != "" {
					return reason
				}
				if !a.operation.SupportsSkip() {
					return "Merge does not support skip"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				invocation, err := a.operation.SkipInvocation()
				if err != nil {
					a.errorText = err.Error()
					return nil
				}
				a.confirmOperationStep("Skip current commit?", "Git drops the current commit from this "+a.operation.Kind.Name()+" and resets its changes in the index and working tree. Remaining commits continue.", invocation, false)
				return nil
			}},
		{ID: "op.abort", Title: "Abort operation", Scope: ScopeOperation, Keys: []string{"A"}, Mutates: true,
			Available: operationActive,
			Run: func(a *App) tea.Cmd {
				title := map[git.OperationKind]string{git.OperationRebase: "Abort interactive rebase?", git.OperationCherryPick: "Abort cherry-pick?", git.OperationMerge: "Abort merge?"}[a.operation.Kind]
				a.confirmOperationStep(title, "Git tries to restore the state before the operation started. Abort is not a universal undo: edits made during the operation may be lost or may block the abort.", a.operation.AbortInvocation(), true)
				return nil
			}},
		{ID: "op.refresh", Title: "Refresh operation state", Scope: ScopeOperation, Keys: []string{"r"},
			Run: func(a *App) tea.Cmd { return a.refresh() }},

		conflictChoice("cf.ours", "Accept ours for region", "1", git.ChoiceOurs),
		conflictChoice("cf.theirs", "Accept theirs for region", "2", git.ChoiceTheirs),
		conflictChoice("cf.both", "Accept both for region", "3", git.ChoiceBoth),
		{ID: "cf.editor", Title: "Open in configured Git editor", Scope: ScopeConflict, Keys: []string{"e"}, Mutates: true,
			Available: func(a *App) string {
				if conflict(a).conflict.Missing {
					return "file does not exist in the working tree"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return conflict(a).openEditor(a) }},
		{ID: "cf.resolve", Title: "Mark resolved (stage result)", Scope: ScopeConflict, Mutates: true,
			Available: func(a *App) string { return conflict(a).resolveUnavailable() },
			Run:       func(a *App) tea.Cmd { return conflict(a).markResolved(a) }},

		{ID: "br.fetch", Title: "Fetch", Scope: ScopeBranch, Mutates: true,
			Run: func(a *App) tea.Cmd { return a.gitHandoff("Fetch", git.Invocation{Args: []string{"fetch"}}, nil) }},
		{ID: "br.pull", Title: "Pull", Scope: ScopeBranch, Mutates: true,
			Available: func(a *App) string {
				if reason := notDetached(a); reason != "" {
					return reason
				}
				if a.status.Branch.Upstream == "" {
					return "No upstream: set one first"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return a.gitHandoff("Pull", git.Invocation{Args: []string{"pull"}}, nil) }},
		{ID: "br.push", Title: "Push", Scope: ScopeBranch, Mutates: true,
			Available: func(a *App) string {
				if reason := notDetached(a); reason != "" {
					return reason
				}
				if a.status.Branch.Upstream == "" {
					return "No upstream: use Push and set upstream"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd { return a.gitHandoff("Push", git.Invocation{Args: []string{"push"}}, nil) }},
		{ID: "br.pushUpstream", Title: "Push and set upstream", Scope: ScopeBranch, Mutates: true,
			Available: func(a *App) string {
				if reason := notDetached(a); reason != "" {
					return reason
				}
				switch {
				case a.status.Branch.Upstream != "":
					return "upstream already set: use Push"
				case len(branches(a).remotes) == 0:
					return "no remotes configured"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				branch := a.status.Branch.Head
				a.overlay = newPrompt("Push "+branch+" and set upstream", "Remote name (git push --set-upstream <remote> "+branch+")", branches(a).defaultRemote(), func(a *App, remote string) tea.Cmd {
					invocation, err := git.PushSetUpstreamInvocation(remote, branch)
					if err != nil {
						a.errorText = err.Error()
						return nil
					}
					return a.gitHandoff("Push and set upstream", invocation, nil)
				})
				return nil
			}},
		{ID: "br.setUpstream", Title: "Set upstream", Scope: ScopeBranch, Mutates: true,
			Available: notDetached,
			Run: func(a *App) tea.Cmd {
				initial := branches(a).defaultRemote() + "/" + a.status.Branch.Head
				runner := a.runner
				a.overlay = newPrompt("Set upstream of "+a.status.Branch.Head, "Existing remote-tracking ref (git branch --set-upstream-to=<ref>)", initial, func(a *App, ref string) tea.Cmd {
					return a.mutate("Set upstream", func(ctx context.Context) error { return git.SetUpstream(ctx, runner, ref) }, nil)
				})
				return nil
			}},
		{ID: "br.create", Title: "Create branch at HEAD", Scope: ScopeBranch, Mutates: true,
			Run: func(a *App) tea.Cmd {
				runner := a.runner
				a.overlay = newPrompt("Create local branch", "New branch name (created at HEAD, not switched)", "", func(a *App, name string) tea.Cmd {
					return a.mutate("Create branch "+name, func(ctx context.Context) error { return git.CreateBranch(ctx, runner, name) }, nil)
				})
				return nil
			}},
		{ID: "br.switch", Title: "Switch to focused branch", Scope: ScopeBranch, Mutates: true,
			Available: func(a *App) string {
				branch, ok := branches(a).focused()
				switch {
				case !ok:
					return "no branch focused"
				case branch.Current:
					return "already on this branch"
				case a.operation.Active():
					return "a " + a.operation.Kind.Name() + " is in progress"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				branch, _ := branches(a).focused()
				runner := a.runner
				return a.mutate("Switch to "+branch.Name, func(ctx context.Context) error { return git.SwitchBranch(ctx, runner, branch.Name) }, nil)
			}},
		{ID: "br.delete", Title: "Delete focused local branch", Scope: ScopeBranch, Mutates: true,
			Available: func(a *App) string {
				branch, ok := branches(a).focused()
				switch {
				case !ok:
					return "no branch focused"
				case branch.Current:
					return "cannot delete the checked-out branch"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				branch, _ := branches(a).focused()
				runner := a.runner
				a.overlay = &confirmOverlay{
					title:   "Delete local branch " + branch.Name + "?",
					lines:   []string{"Git refuses if the branch is not fully merged. GitFlow never force-deletes.", "Remote branches are not touched."},
					command: "git branch -d " + branch.Name,
					danger:  true,
					buttons: []confirmButton{{label: "Cancel"}, {label: "Delete", run: func(a *App) tea.Cmd {
						return a.mutate("Delete branch "+branch.Name, func(ctx context.Context) error { return git.DeleteBranch(ctx, runner, branch.Name) }, nil)
					}}},
				}
				return nil
			}},
		{ID: "br.base", Title: "Set comparison base", Scope: ScopeBranch,
			Run: func(a *App) tea.Cmd {
				initial := a.base.ShortRef()
				runner := a.runner
				a.overlay = newPrompt("Set comparison base (this session only)", "Any local ref, e.g. origin/main, main, origin/release", initial, func(a *App, ref string) tea.Cmd {
					return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
						base, err := git.ResolveBase(ctx, runner, ref)
						return func(a *App) tea.Cmd {
							if err != nil {
								a.errorText = "Comparison base unchanged: " + err.Error()
								return nil
							}
							a.baseOverride, a.base, a.baseLoaded, a.baseErr = ref, base, true, nil
							a.notice = "Comparison base set to " + ref + " for this session"
							return nil
						}
					})
				})
				return nil
			}},
		{ID: "br.rebase", Title: "Interactive rebase onto comparison base", Scope: ScopeBranch, Keys: []string{"r i"},
			Available: func(a *App) string {
				if !a.base.Available {
					return "Base unavailable"
				}
				return ""
			},
			Run: func(a *App) tea.Cmd {
				oid, err := git.ResolveCommit(a.ctx, a.runner, a.base.Ref)
				if err != nil {
					a.errorText = err.Error()
					return nil
				}
				return a.openScreen(newRebaseScreen(a, oid, a.base.ShortRef()))
			}},
	}
}
