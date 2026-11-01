package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/gittest"
)

func newHistoryApp(t *testing.T, setup func(fixture *gittest.Fixture)) (*App, *gittest.Fixture) {
	t.Helper()
	fixture := gittest.NewFixture(t)
	setup(fixture)
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	return gitflowApp, fixture
}

func openHistory(t *testing.T, gitflowApp *App) *historyScreen {
	t.Helper()
	pressAndDrain(gitflowApp, "g", "h")
	history, open := gitflowApp.historyOnTop()
	if !open {
		t.Fatalf("g h opened %s, want S04", gitflowApp.top().ID())
	}
	return history
}

func threeCommits(fixture *gittest.Fixture) {
	fixture.Commit("first", map[string]string{"a.txt": "1\n"})
	fixture.Commit("second", map[string]string{"a.txt": "2\n"})
	fixture.Commit("third", map[string]string{"a.txt": "3\n", "b.txt": "b\n"})
}

func TestHistoryOpenAndReturn(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeCommits)
	before := fixture.Snapshot()
	history := openHistory(t, gitflowApp)
	tip := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	if len(history.commits) != 3 || history.focusOID != tip || !history.done {
		t.Fatalf("S04 loaded %d commits, focus %s, done %v", len(history.commits), history.focusOID, history.done)
	}
	if history.detail.Summary.OID != tip || !strings.Contains(history.detail.Patch, "+3") {
		t.Fatalf("detail of focused commit = %+v", history.detail)
	}

	pressAndDrain(gitflowApp, "j", " ", "H")
	second := history.commits[1].OID
	if gitflowApp.top() != history || history.focusOID != second || !history.selection.Contains(second) {
		t.Fatalf("T135 after j Space H: top %s, focus %s, selected %v", gitflowApp.top().ID(), history.focusOID, history.selection.Ordered())
	}
	view := gitflowApp.View()
	for _, want := range []string{"History · main @ " + shortOID(tip), "3 loaded · 1 selected · 0 hidden · all loaded", "[x] ", historyFilterLabel} {
		if !strings.Contains(view, want) {
			t.Errorf("S04 view missing %q:\n%s", want, view)
		}
	}

	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("Esc returned to %s, want S01", gitflowApp.top().ID())
	}
	reopened := openHistory(t, gitflowApp)
	if reopened != history || reopened.focusOID != second || !reopened.selection.Contains(second) || len(gitflowApp.stack) != 2 {
		t.Fatalf("T135 reopen lost context: focus %s, selected %v, stack %d", reopened.focusOID, reopened.selection.Ordered(), len(gitflowApp.stack))
	}
	pressAndDrain(gitflowApp, "g", "h")
	if history.focusOID != second || !history.selection.Contains(second) || len(history.commits) != 3 || len(gitflowApp.stack) != 2 {
		t.Fatalf("g h on S04 reload lost focus or selection: focus %s, selected %v", history.focusOID, history.selection.Ordered())
	}
	if fixture.Snapshot() != before {
		t.Fatal("history navigation changed Git state")
	}
}

func TestHistoryLargePagination(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, func(fixture *gittest.Fixture) { fixture.ImportHistory(1100, 10) })
	before := fixture.Snapshot()
	history := openHistory(t, gitflowApp)
	if len(history.commits) != historyPageSize || history.done {
		t.Fatalf("first page loaded %d commits, done %v", len(history.commits), history.done)
	}
	first := history.focusOID
	pressAndDrain(gitflowApp, " ", "G")
	if len(history.commits) != 2*historyPageSize || history.focusOID != history.commits[historyPageSize-1].OID {
		t.Fatalf("T14 G loaded %d commits, focus %s", len(history.commits), history.focusOID)
	}
	for !history.done {
		pressAndDrain(gitflowApp, "G")
	}
	want := strings.Fields(fixture.Git("rev-list", "--topo-order", "main"))
	if len(history.commits) != len(want) || !history.done {
		t.Fatalf("T14 loaded %d of %d commits, done %v", len(history.commits), len(want), history.done)
	}
	for position, commit := range history.commits {
		if commit.OID != want[position] {
			t.Fatalf("T14 row %d = %s, want %s", position, commit.OID, want[position])
		}
	}
	if history.selection.Len() != 1 || !history.selection.Contains(first) {
		t.Fatalf("T14 selection after paging = %v", history.selection.Ordered())
	}
	if fixture.Snapshot() != before {
		t.Fatal("paging history changed Git state")
	}
}

func TestHistoryFilterLoadedOnly(t *testing.T) {
	gitflowApp, _ := newHistoryApp(t, threeCommits)
	history := openHistory(t, gitflowApp)
	tip := history.focusOID
	pressAndDrain(gitflowApp, " ", "/", "s", "e", "c")
	rows := history.visible()
	if len(rows) != 1 || rows[0].Subject != "second" || history.focusOID != rows[0].OID {
		t.Fatalf("filter rows = %+v, focus %s", rows, history.focusOID)
	}
	if view := gitflowApp.View(); !strings.Contains(view, "3 loaded · 1 selected · 1 hidden") || !strings.Contains(view, historyFilterPrompt+"sec") {
		t.Fatalf("filter view lacks hidden count or scope label:\n%s", view)
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top() != history || history.search.Value() != "" || !history.selection.Contains(tip) || len(history.visible()) != 3 {
		t.Fatalf("Esc did not clear the filter only: top %s, query %q", gitflowApp.top().ID(), history.search.Value())
	}
}

func TestHistoryCommitFiles(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeCommits)
	before := fixture.Snapshot()
	history := openHistory(t, gitflowApp)
	pressAndDrain(gitflowApp, " ", "enter")
	commitFiles, open := gitflowApp.top().(*commitFilesScreen)
	if !open {
		t.Fatalf("Enter opened %s, want S13", gitflowApp.top().ID())
	}
	parent := history.commits[1].OID
	if commitFiles.parent != parent || len(commitFiles.files) != 2 || commitFiles.files[0].Path != "a.txt" || !strings.Contains(commitFiles.patch, "+3") {
		t.Fatalf("S13 = parent %s files %+v patch %q", commitFiles.parent, commitFiles.files, commitFiles.patch)
	}
	view := gitflowApp.View()
	for _, want := range []string{history.focusOID, fmt.Sprintf(commitFilesParent, parent), fmt.Sprintf(commitFilesSummary, 2)} {
		if !strings.Contains(view, want) {
			t.Errorf("S13 view missing %q:\n%s", want, view)
		}
	}
	pressAndDrain(gitflowApp, "j")
	if commitFiles.patchPath != "b.txt" || !strings.Contains(commitFiles.patch, "+b") {
		t.Fatalf("focused file patch = %s %q", commitFiles.patchPath, commitFiles.patch)
	}
	pressAndDrain(gitflowApp, "enter")
	if !commitFiles.fullWidth {
		t.Fatal("Enter in S13 did not open the full-width diff")
	}
	pressAndDrain(gitflowApp, "esc")
	if commitFiles.fullWidth || gitflowApp.top() != commitFiles {
		t.Fatal("Esc did not leave the full-width diff first")
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top() != history || history.focusOID != commitFiles.commit.OID || !history.selection.Contains(commitFiles.commit.OID) {
		t.Fatalf("H03 Esc from S13 returned to %s, focus %s, selected %v", gitflowApp.top().ID(), history.focusOID, history.selection.Ordered())
	}
	pressAndDrain(gitflowApp, "G", "enter")
	root, open := gitflowApp.top().(*commitFilesScreen)
	if !open || root.parent != "" || !strings.Contains(gitflowApp.View(), commitFilesRootParent) {
		t.Fatalf("root commit S13 open %v parent %q", open, root.parent)
	}
	if fixture.Snapshot() != before {
		t.Fatal("inspecting commit files changed Git state")
	}
}

func TestHistoryMergeAndCompareBlocked(t *testing.T) {
	gitflowApp, _ := newHistoryApp(t, func(fixture *gittest.Fixture) {
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("switch", "-q", "-c", "side")
		fixture.Commit("side", map[string]string{"b.txt": "b\n"})
		fixture.Git("switch", "-q", "main")
		fixture.Commit("main", map[string]string{"c.txt": "c\n"})
		fixture.Git("merge", "-q", "--no-ff", "-m", "merge side", "side")
	})
	history := openHistory(t, gitflowApp)
	pressAndDrain(gitflowApp, "enter")
	if gitflowApp.top() != history || gitflowApp.notice != reasonMergeContractGap {
		t.Fatalf("Enter on merge: top %s, notice %q", gitflowApp.top().ID(), gitflowApp.notice)
	}
	compare, _ := gitflowApp.registry.ByID(actionCompareCommits)
	pressAndDrain(gitflowApp, " ")
	if _, reason := compare.Availability(gitflowApp); reason != reasonSelectTwoCommits {
		t.Fatalf("compare with one selected: %q", reason)
	}
	pressAndDrain(gitflowApp, "j", " ")
	if available, reason := compare.Availability(gitflowApp); available || reason != reasonCompareContractGap {
		t.Fatalf("compare with two selected: %v %q", available, reason)
	}
}

func TestHistoryStaleAndDetached(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, threeCommits)
	history := openHistory(t, gitflowApp)
	pressAndDrain(gitflowApp, "esc")
	fixture.Commit("moved", map[string]string{"moved.txt": "x\n"})
	openHistory(t, gitflowApp)
	if !history.stale || gitflowApp.notice != historyStaleNotice || len(history.commits) != 3 {
		t.Fatalf("stale %v notice %q commits %d", history.stale, gitflowApp.notice, len(history.commits))
	}
	pressAndDrain(gitflowApp, "g", "h")
	if history.stale || len(history.commits) != 4 {
		t.Fatalf("g h reload: stale %v commits %d", history.stale, len(history.commits))
	}

	fixture.Git("switch", "-q", "--detach", "HEAD~1")
	pressAndDrain(gitflowApp, "g", "h")
	if !history.cursor.Detached || !strings.Contains(gitflowApp.View(), "History · "+detachedLabel+shortOID(history.cursor.TipOID)) {
		t.Fatalf("detached S04 header missing:\n%s", gitflowApp.View())
	}
}
