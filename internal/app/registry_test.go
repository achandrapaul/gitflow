package app

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func runNothing(*App) tea.Cmd { return nil }

func TestNewRegistryRejectsInvalidActions(t *testing.T) {
	testCases := []struct {
		name    string
		actions []Action
		wantErr string
	}{
		{
			name: "duplicate id",
			actions: []Action{
				{ID: "one", Scope: ScopeGlobal, Keys: []string{"x"}},
				{ID: "one", Scope: ScopeChanges, Keys: []string{"y"}},
			},
			wantErr: "duplicate id",
		},
		{
			name: "duplicate key in one scope",
			actions: []Action{
				{ID: "one", Scope: ScopeChanges, Keys: []string{"x"}},
				{ID: "two", Scope: ScopeChanges, Keys: []string{"x"}},
			},
			wantErr: "already bound",
		},
		{
			name: "alias collides with key",
			actions: []Action{
				{ID: "one", Scope: ScopeChanges, Keys: []string{"x"}},
				{ID: "two", Scope: ScopeGlobal, Keys: []string{"g x"}, Aliases: []Binding{{Scope: ScopeChanges, Keys: []string{"x"}}}},
			},
			wantErr: "already bound",
		},
		{name: "reserved a e", actions: []Action{{ID: "one", Scope: ScopeChanges, Keys: []string{"a e"}}}, wantErr: "reserved"},
		{name: "reserved gg", actions: []Action{{ID: "one", Scope: ScopeGlobal, Keys: []string{"gg"}}}, wantErr: "reserved"},
		{name: "reserved g g", actions: []Action{{ID: "one", Scope: ScopeGlobal, Keys: []string{"g g"}}}, wantErr: "reserved"},
		{name: "retired H", actions: []Action{{ID: "one", Scope: ScopeChanges, Keys: []string{"H"}}}, wantErr: "reserved"},
		{name: "deferred r i", actions: []Action{{ID: "one", Scope: ScopeRebase, Keys: []string{"r i"}}}, wantErr: "reserved"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewRegistry(testCase.actions)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func TestRegistryQueries(t *testing.T) {
	registry, err := NewRegistry([]Action{
		{ID: "branches.open", Title: "Branches", Scope: ScopeGlobal, Keys: []string{"g b"}, Aliases: []Binding{{Scope: ScopeChanges, Keys: []string{"b"}}}},
		{ID: "sync.pull", Title: "Pull", Scope: ScopeGlobal, Keys: []string{"g p"}},
		{ID: "sync.push", Title: "Push", Scope: ScopeGlobal, Keys: []string{"g P"}},
		{ID: "changes.stage", Title: "Stage", Scope: ScopeChanges, Keys: []string{"s"}},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	t.Run("alias resolves to the same action", func(t *testing.T) {
		canonical, canonicalFound := registry.Lookup(ScopeGlobal, "g b")
		alias, aliasFound := registry.Lookup(ScopeChanges, "b")
		if !canonicalFound || !aliasFound || canonical != alias {
			t.Fatalf("canonical %v alias %v", canonical, alias)
		}
	})

	t.Run("second key is case-sensitive", func(t *testing.T) {
		pull, _ := registry.Lookup(ScopeGlobal, "g p")
		push, _ := registry.Lookup(ScopeGlobal, "g P")
		if pull.ID != "sync.pull" || push.ID != "sync.push" {
			t.Fatalf("g p = %s, g P = %s", pull.ID, push.ID)
		}
	})

	t.Run("lookup is scoped", func(t *testing.T) {
		if _, found := registry.Lookup(ScopeGlobal, "s"); found {
			t.Fatal("changes key leaked into global scope")
		}
	})

	t.Run("in scope includes aliases in registration order", func(t *testing.T) {
		var ids []ActionID
		for _, action := range registry.InScope(ScopeChanges) {
			ids = append(ids, action.ID)
		}
		if !slices.Equal(ids, []ActionID{"branches.open", "changes.stage"}) {
			t.Fatalf("ids = %v", ids)
		}
	})

	t.Run("by id", func(t *testing.T) {
		action, found := registry.ByID("changes.stage")
		if !found || action.Title != "Stage" {
			t.Fatalf("action = %v", action)
		}
	})

	t.Run("search matches title case-insensitively", func(t *testing.T) {
		var ids []ActionID
		for _, action := range registry.Search("pu") {
			ids = append(ids, action.ID)
		}
		if !slices.Equal(ids, []ActionID{"sync.pull", "sync.push"}) {
			t.Fatalf("ids = %v", ids)
		}
		if len(registry.Search("")) != 4 {
			t.Fatal("empty query must list every action")
		}
	})
}

func TestActionAvailability(t *testing.T) {
	testCases := []struct {
		name       string
		action     Action
		wantOK     bool
		wantReason string
	}{
		{name: "no handler", action: Action{ID: "x"}, wantReason: reasonNotAvailableYet},
		{name: "handler", action: Action{ID: "x", Run: runNothing}, wantOK: true},
		{name: "handler with blocker", action: Action{ID: "x", Run: runNothing, Available: func(*App) (bool, string) { return false, "blocked" }}, wantReason: "blocked"},
		{name: "blocker without handler", action: Action{ID: "x", Available: func(*App) (bool, string) { return false, "blocked" }}, wantReason: "blocked"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ok, reason := testCase.action.Availability(&App{})
			if ok != testCase.wantOK || reason != testCase.wantReason {
				t.Fatalf("availability = %v %q", ok, reason)
			}
		})
	}
}

func TestDefaultActions(t *testing.T) {
	registry, err := NewRegistry(defaultActions())
	if err != nil {
		t.Fatalf("default actions: %v", err)
	}
	sliceActions := map[ActionID]bool{
		actionOpenActions:      true,
		actionOpenGitMenu:      true,
		actionOpenPalette:      true,
		actionOpenHelp:         true,
		actionBack:             true,
		actionOpenStashes:      true,
		actionOpenCherryPick:   true,
		actionFocusComposer:    true,
		actionFilterAll:        true,
		actionFilterStaged:     true,
		actionFilterNotStaged:  true,
		actionFilterMixed:      true,
		actionToggleLayout:     true,
		actionSelectAll:        true,
		actionOpenBranches:     true,
		actionOpenHistory:      true,
		actionFetch:            true,
		actionOperationRefresh: true,
	}
	for _, action := range registry.Search("") {
		available, reason := action.Availability(&App{})
		if slices.Contains([]ActionID{actionCreateFromRef, actionSwitchBranch, actionDeleteBranch}, action.ID) {
			if available || reason != reasonNoBranchFocused {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonNoBranchFocused)
			}
			continue
		}
		if slices.Contains([]ActionID{actionCompareCommits, actionRebaseStart}, action.ID) {
			if available || reason != reasonHistoryClosed {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonHistoryClosed)
			}
			continue
		}
		if action.ID == actionSetUpstream {
			if available || reason != reasonUpstreamContractGap {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonUpstreamContractGap)
			}
			continue
		}
		if slices.Contains([]ActionID{actionCommit, actionCommitNoVerify, actionAmend, actionStage, actionUnstage, actionCreateBranch,
			actionPullConfigured, actionPullRebase, actionPullFastForward, actionPush, actionForcePush}, action.ID) {
			if available || reason != reasonStatusLoading {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonStatusLoading)
			}
			continue
		}
		if slices.Contains([]ActionID{actionOpenPaused, actionOperationContinue, actionOperationSkip, actionOperationAbort}, action.ID) {
			if available || reason != reasonNoPausedOperation {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonNoPausedOperation)
			}
			continue
		}
		if slices.Contains([]ActionID{actionConflictEditor, actionConflictResolved, actionConflictOurs, actionConflictBase, actionConflictTheirs, actionConflictNext, actionConflictPrevious}, action.ID) {
			if available || reason != reasonResolverNotLoaded {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonResolverNotLoaded)
			}
			continue
		}
		if slices.Contains([]ActionID{actionCherryPickPreview, actionCherryPickExecute, actionCherryPickMoveDown, actionCherryPickMoveUp}, action.ID) {
			if available || reason != reasonExplorerClosed {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonExplorerClosed)
			}
			continue
		}
		if slices.Contains([]ActionID{actionRebasePreview, actionRebasePick, actionRebaseReword, actionRebaseEdit, actionRebaseSquash, actionRebaseFixup,
			actionRebaseDrop, actionRebaseMoveDown, actionRebaseMoveUp, actionRebaseExecute}, action.ID) {
			if available || reason != reasonPlannerClosed {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonPlannerClosed)
			}
			continue
		}
		if slices.Contains([]ActionID{actionApplyPreview, actionApplyExecute}, action.ID) {
			if available || reason != reasonFileModeClosed {
				t.Errorf("%s: available=%v reason=%q, want %q", action.ID, available, reason, reasonFileModeClosed)
			}
			continue
		}
		if sliceActions[action.ID] {
			if !available {
				t.Errorf("%s should be available, got %q", action.ID, reason)
			}
			continue
		}
		if available || reason != reasonNotAvailableYet {
			t.Errorf("%s: available=%v reason=%q, want disabled %q", action.ID, available, reason, reasonNotAvailableYet)
		}
	}

	specKeys := []struct {
		scope Scope
		keys  string
		id    ActionID
	}{
		{ScopeGlobal, "g h", actionOpenHistory},
		{ScopeGlobal, "g b", actionOpenBranches},
		{ScopeChanges, "b", actionOpenBranches},
		{ScopeGlobal, "g s", actionOpenStashes},
		{ScopeGlobal, "g c", actionOpenCherryPick},
		{ScopeCommitFiles, "p", actionApplyPreview},
		{ScopeSelectiveApply, "p", actionApplyPreview},
		{ScopeGlobal, "g n", actionCreateBranch},
		{ScopeBranch, "a n", actionCreateFromRef},
		{ScopeGlobal, "/", actionSearch},
		{ScopeBranch, "a m", actionCreateFromMain},
		{ScopeGlobal, "g u", actionSetUpstream},
		{ScopeBranch, "a u", actionSetUpstream},
		{ScopeGlobal, "g p", actionPullConfigured},
		{ScopeGlobal, "g r", actionPullRebase},
		{ScopeGlobal, "g f", actionPullFastForward},
		{ScopeGlobal, "g P", actionPush},
		{ScopeGlobal, "g F", actionForcePush},
		{ScopeGlobal, "g e", actionFetch},
		{ScopeGlobal, "o", actionOpenPaused},
		{ScopeComposer, "a v", actionCommitNoVerify},
		{ScopeChanges, "c", actionFocusComposer},
		{ScopeChanges, "s", actionStage},
		{ScopeDiff, "s", actionStage},
		{ScopeChanges, "u", actionUnstage},
		{ScopeDiff, "u", actionUnstage},
		{ScopeChanges, "S", actionSelectAll},
		{ScopeChanges, "v", actionToggleLayout},
		{ScopeCherryPick, "J", actionCherryPickMoveDown},
		{ScopeRebase, "J", actionRebaseMoveDown},
		{ScopeBranch, "s", actionSwitchBranch},
	}
	for _, spec := range specKeys {
		action, found := registry.Lookup(spec.scope, spec.keys)
		if !found || action.ID != spec.id {
			t.Errorf("%s %q = %v, want %s", spec.scope, spec.keys, action, spec.id)
		}
	}
	for _, unbound := range []string{"a e", "g g", "gg", "H", "r i"} {
		for _, scope := range []Scope{ScopeGlobal, ScopeChanges, ScopeComposer, ScopeBranch, ScopeRebase} {
			if _, found := registry.Lookup(scope, unbound); found {
				t.Errorf("%q must stay unbound in %s", unbound, scope)
			}
		}
	}
}
