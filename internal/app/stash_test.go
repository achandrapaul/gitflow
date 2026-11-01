package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/gittest"
)

const (
	missingStashObject = "1111111111111111111111111111111111111111\n"
	stashRefFileMode   = 0o644
	stashTitleWord     = "stash"
)

func openStashes(t *testing.T, gitflowApp *App) *stashScreen {
	t.Helper()
	pressAndDrain(gitflowApp, "g", "s")
	stashes, open := gitflowApp.top().(*stashScreen)
	if !open {
		t.Fatalf("g s opened %s, want S16", gitflowApp.top().ID())
	}
	return stashes
}

func threeStashes(fixture *gittest.Fixture) {
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	for _, message := range []string{"oldest work", "middle work", "newest work"} {
		fixture.Write("a.txt", message+"\n")
		fixture.Git("stash", "push", "-q", "-m", message)
	}
}

func TestStashListReadOnly(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeStashes)
	before, beforeList := fixture.Snapshot(), fixture.Git("stash", "list")
	stashes := openStashes(t, gitflowApp)
	if len(stashes.stashes) != 3 || stashes.focusOID != stashes.stashes[0].OID {
		t.Fatalf("T145 S16 loaded %+v, focus %s", stashes.stashes, stashes.focusOID)
	}
	view := gitflowApp.View()
	for _, stash := range stashes.stashes {
		for _, want := range []string{stash.Ref(), shortOID(stash.OID), stash.Message, stash.Date.Format(commitDateLayout)} {
			if !strings.Contains(view, want) {
				t.Errorf("T145 S16 view missing %q:\n%s", want, view)
			}
		}
	}

	pressAndDrain(gitflowApp, "j", "enter", "G", "k")
	middle := stashes.stashes[1].OID
	if gitflowApp.top() != stashes || stashes.focusOID != middle {
		t.Fatalf("T145 scroll: top %s focus %s, want %s", gitflowApp.top().ID(), stashes.focusOID, middle)
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("Esc returned to %s, want S01", gitflowApp.top().ID())
	}
	reopened := openStashes(t, gitflowApp)
	if reopened != stashes || reopened.focusOID != middle || len(gitflowApp.stack) != 2 {
		t.Fatalf("T145 reopen lost focus: %s, stack %d", reopened.focusOID, len(gitflowApp.stack))
	}

	pressAndDrain(gitflowApp, "/", "o", "l", "d")
	rows := stashes.visible()
	if len(rows) != 1 || rows[0].Message != stashes.stashes[2].Message || stashes.focusOID != rows[0].OID {
		t.Fatalf("filter rows = %+v, focus %s", rows, stashes.focusOID)
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top() != stashes || stashes.search.Value() != "" || len(stashes.visible()) != 3 {
		t.Fatalf("Esc did not clear the filter only: top %s, query %q", gitflowApp.top().ID(), stashes.search.Value())
	}
	pressAndDrain(gitflowApp, "q")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("q returned to %s, want S01", gitflowApp.top().ID())
	}
	if fixture.Snapshot() != before || fixture.Git("stash", "list") != beforeList {
		t.Fatal("T145 S16 navigation changed Git state")
	}
}

func TestStashListEmptyAndError(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeCommits)
	stashes := openStashes(t, gitflowApp)
	if len(stashes.stashes) != 0 || stashes.err != nil || !strings.Contains(gitflowApp.View(), noStashes) {
		t.Fatalf("T146 empty S16: %+v err %v\n%s", stashes.stashes, stashes.err, gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("T146 Esc from empty S16 returned to %s", gitflowApp.top().ID())
	}

	fixture.Write("a.txt", "dirty\n")
	fixture.Git("stash", "push", "-q")
	if err := os.WriteFile(filepath.Join(fixture.Dir, ".git", "refs", "stash"), []byte(missingStashObject), stashRefFileMode); err != nil {
		t.Fatalf("corrupt refs/stash: %v", err)
	}
	before := fixture.Snapshot()
	openStashes(t, gitflowApp)
	if len(stashes.stashes) != 0 || stashes.err == nil || !strings.Contains(strings.Join(strings.Fields(gitflowApp.View()), " "), stashes.err.Error()) {
		t.Fatalf("T146 broken S16: %+v err %v\n%s", stashes.stashes, stashes.err, gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges || fixture.Snapshot() != before {
		t.Fatalf("T146 Esc from broken S16 returned to %s or changed Git state", gitflowApp.top().ID())
	}
}

func TestStashKeyNeverStagesOrSkips(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeStashes)
	fixture.Write("a.txt", "unstaged\n")
	drain(gitflowApp, gitflowApp.refresh())
	before := fixture.Snapshot()

	pressAndDrain(gitflowApp, "g", "S")
	if gitflowApp.top().ID() != ScreenChanges || fixture.Snapshot() != before {
		t.Fatalf("K46 g S opened %s or changed Git state", gitflowApp.top().ID())
	}
	openStashes(t, gitflowApp)
	pressAndDrain(gitflowApp, "esc")

	gitflowApp.push(&pausedScreen{app: gitflowApp})
	stashes := openStashes(t, gitflowApp)
	if gitflowApp.overlay != nil || len(gitflowApp.stack) != 3 || gitflowApp.stack[1].ID() != ScreenPaused {
		t.Fatalf("T144 g s from S08: overlay %T, stack %d", gitflowApp.overlay, len(gitflowApp.stack))
	}

	for _, menuKey := range []string{"a", ":"} {
		pressAndDrain(gitflowApp, menuKey)
		var entries []menuEntry
		switch overlay := gitflowApp.overlay.(type) {
		case *actionsOverlay:
			entries = overlay.entries
		case *paletteOverlay:
			entries = overlay.entries
		default:
			t.Fatalf("%s in S16 opened %T", menuKey, gitflowApp.overlay)
		}
		for _, entry := range entries {
			if !strings.Contains(strings.ToLower(entry.action.Title), stashTitleWord) || entry.action.ID == actionOpenStashes {
				continue
			}
			if !slices.Contains(approvedStashWorkflows, entry.action.ID) || entry.available {
				t.Errorf("T145 %s menu offers stash action %s (available %v)", menuKey, entry.action.ID, entry.available)
			}
		}
		pressAndDrain(gitflowApp, "esc")
	}
	if gitflowApp.top() != stashes || fixture.Snapshot() != before {
		t.Fatalf("menus in S16 left %s on top or changed Git state", gitflowApp.top().ID())
	}
}

var approvedStashWorkflows = []ActionID{actionStashAndSwitch, actionStashAndProceed, actionContinueUnstash, actionAbortUnstash}

func TestNoGenericStashMutation(t *testing.T) {
	registry, err := NewRegistry(defaultActions())
	if err != nil {
		t.Fatalf("default actions: %v", err)
	}
	for _, action := range registry.Search("") {
		if !strings.Contains(strings.ToLower(action.Title), stashTitleWord) || action.ID == actionOpenStashes {
			continue
		}
		if !slices.Contains(approvedStashWorkflows, action.ID) {
			t.Errorf("ST01 registry has unapproved stash action %s %q", action.ID, action.Title)
		}
		if available, reason := action.Availability(&App{}); available || reason != reasonNotAvailableYet {
			t.Errorf("%s: available=%v reason=%q, want %q until its slice", action.ID, available, reason, reasonNotAvailableYet)
		}
	}
	if scoped := registry.InScope(ScopeStash); len(scoped) != 0 {
		t.Errorf("ST01 S16 scope has actions %v", scoped)
	}
}
