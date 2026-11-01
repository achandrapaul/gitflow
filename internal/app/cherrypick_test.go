package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

const sideBranch = "side"

func sideBranchCommits(fixture *gittest.Fixture) {
	fixture.Commit("base", map[string]string{"tracked.txt": "tracked\n", "conflict.txt": "a\nb\nc\n"})
	fixture.Git("switch", "-q", "-c", sideBranch)
	fixture.Commit("first", map[string]string{"first.txt": "first\n"})
	fixture.Commit("second", map[string]string{"second.txt": "second\n"})
	fixture.Commit("conflicting", gittest.FeatureOne)
	fixture.Commit("third", map[string]string{"third.txt": "third\n", "extra.txt": "extra\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", gittest.ConflictMain)
}

func openCherryPick(t *testing.T, gitflowApp *App) *cherryPickScreen {
	t.Helper()
	pressAndDrain(gitflowApp, "g", "c")
	explorer, open := gitflowApp.top().(*cherryPickScreen)
	if !open {
		t.Fatalf("g c opened %s, want Cherry-pick Explorer (S06)", gitflowApp.top().ID())
	}
	return explorer
}

func browseSource(t *testing.T, gitflowApp *App, explorer *cherryPickScreen, ref string) {
	t.Helper()
	for !explorer.branchesFocused {
		pressAndDrain(gitflowApp, "tab")
	}
	for explorer.branchRow > 0 {
		pressAndDrain(gitflowApp, "k")
	}
	for explorer.branches[explorer.branchRow].Ref != ref {
		if explorer.branchRow == len(explorer.branches)-1 {
			t.Fatalf("source %s not listed in %v", ref, explorer.branches)
		}
		pressAndDrain(gitflowApp, "j")
	}
	pressAndDrain(gitflowApp, "enter")
	if explorer.commits.cursor.SourceRef != ref || explorer.branchesFocused {
		t.Fatalf("browsing %s left source %s, branches focused %v", ref, explorer.commits.cursor.SourceRef, explorer.branchesFocused)
	}
}

func focusSubject(t *testing.T, gitflowApp *App, commits *historyScreen, subject string) git.CommitSummary {
	t.Helper()
	rows := commits.visible()
	target := slices.IndexFunc(rows, func(commit git.CommitSummary) bool { return commit.Subject == subject })
	if target < 0 {
		t.Fatalf("commit %q not listed", subject)
	}
	for commits.focusedIndex(rows) > target {
		pressAndDrain(gitflowApp, "k")
	}
	for commits.focusedIndex(rows) < target {
		pressAndDrain(gitflowApp, "j")
	}
	return rows[target]
}

func queueSubjects(t *testing.T, gitflowApp *App, commits *historyScreen, subjects ...string) []string {
	t.Helper()
	var oids []string
	for _, subject := range subjects {
		oids = append(oids, focusSubject(t, gitflowApp, commits, subject).OID)
		pressAndDrain(gitflowApp, " ")
	}
	return oids
}

func availability(gitflowApp *App, id ActionID) (bool, string) {
	action, _ := gitflowApp.registry.ByID(id)
	return action.Availability(gitflowApp)
}

func overlayText(t *testing.T, gitflowApp *App) string {
	t.Helper()
	if gitflowApp.overlay == nil {
		t.Fatal("no overlay open")
	}
	if review, isReview := gitflowApp.overlay.(*confirmOverlay); isReview {
		return strings.Join(append([]string{review.title, strings.Join(review.command, " ")}, review.consequences...), "\n")
	}
	return gitflowApp.overlay.View(testWidth*2, testHeight*2)
}

func TestCherryPickExplorerBrowsesWithoutCheckout(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
	fixture.Write("notes.txt", "keep\n")
	drain(gitflowApp, gitflowApp.refresh())
	before := fixture.Snapshot()
	explorer := openCherryPick(t, gitflowApp)
	if explorer.commits.cursor.SourceRef != "refs/heads/main" || explorer.commits.label != cherryPickSourceLabel {
		t.Fatalf("default source = %s, want the base branch main", explorer.commits.cursor.SourceRef)
	}
	if explorer.branches[0].Remote || explorer.branches[0].Name != "main" {
		t.Fatalf("source selector starts with %+v, want local branches first", explorer.branches[0])
	}

	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	subjects := []string{}
	for _, commit := range explorer.commits.commits {
		subjects = append(subjects, commit.Subject)
	}
	if !slices.Equal(subjects, []string{"third", "conflicting", "second", "first", "base"}) {
		t.Fatalf("T94 side commits = %v", subjects)
	}
	pressAndDrain(gitflowApp, "tab")
	if !explorer.commits.detailFocused || !strings.Contains(strings.Join(explorer.commits.detailLines(), "\n"), "+third") {
		t.Fatalf("Tab did not focus the read-only diff of the focused commit")
	}
	pressAndDrain(gitflowApp, "tab", "tab")
	view := gitflowApp.View()
	for _, want := range []string{"Source side (browsing only) · Target main (checked out, fixed)", "Mode Whole commits (active) 0 queued · Files (inactive) 0 selected", "Source · side @"} {
		if !strings.Contains(view, want) {
			t.Errorf("S06 view missing %q:\n%s", want, view)
		}
	}
	if fixture.Snapshot() != before || checkedOut(fixture) != "main" {
		t.Fatal("T94 browsing a source branch changed HEAD, target or Git state")
	}
	if readWorktree(t, fixture, "notes.txt") != "keep\n" {
		t.Fatal("T91 non-colliding untracked file changed")
	}

	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("Esc returned to %s", gitflowApp.top().ID())
	}
	reopened := openCherryPick(t, gitflowApp)
	if reopened != explorer || reopened.commits.cursor.SourceRef != "refs/heads/"+sideBranch {
		t.Fatal("reopening S06 lost the browsed source")
	}
}

func TestCherryPickPreviewAndExecute(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
	fixture.Write("notes.txt", "keep\n")
	drain(gitflowApp, gitflowApp.refresh())
	reviewedHead := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	explorer := openCherryPick(t, gitflowApp)
	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	commits := explorer.commits
	before := fixture.Snapshot()

	oids := queueSubjects(t, gitflowApp, commits, "first", "third", "second")
	pressAndDrain(gitflowApp, " ")
	if !slices.Equal(commits.selection.Ordered(), oids[:2]) {
		t.Fatalf("Space toggled queue to %v, want first, third", commits.selection.Ordered())
	}
	focusSubject(t, gitflowApp, commits, "third")
	pressAndDrain(gitflowApp, "K")
	if !slices.Equal(commits.selection.Ordered(), []string{oids[1], oids[0]}) {
		t.Fatalf("K reordered to %v, want third, first", commits.selection.Ordered())
	}
	if !strings.Contains(gitflowApp.View(), "[1] ") || !strings.Contains(gitflowApp.View(), "[2] ") {
		t.Fatal("queue positions not shown")
	}

	pressAndDrain(gitflowApp, "enter")
	files, inspecting := gitflowApp.top().(*commitFilesScreen)
	if !inspecting {
		t.Fatalf("T114 Enter opened %s, want Commit File Inspector (S13)", gitflowApp.top().ID())
	}
	pressAndDrain(gitflowApp, " ")
	if commits.files.Len() != 1 || !strings.Contains(gitflowApp.View(), "Mode Files (active) 1 selected · Whole commits (inactive) 2 queued") {
		t.Fatalf("T96 S13 file mode: files %d, view:\n%s", commits.files.Len(), gitflowApp.View())
	}
	selectedFile := files.files[files.row].Path
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top() != explorer || commits.selection.Len() != 2 || commits.files.Len() != 1 {
		t.Fatal("T97 Esc from S13 lost the whole-commit queue or the inactive file selection")
	}
	if fixture.Snapshot() != before {
		t.Fatal("T114 Space and Enter changed Git state")
	}

	if available, reason := availability(gitflowApp, actionCherryPickExecute); available || reason != reasonPreviewFirst {
		t.Fatalf("Execute before preview = %v %q", available, reason)
	}
	pressAndDrain(gitflowApp, "p")
	preview := overlayText(t, gitflowApp)
	for _, want := range []string{"Cherry-pick preview · 2 commits onto main", "git cherry-pick " + oids[1] + " " + oids[0], "1. " + shortOID(oids[1]) + " third · from side", "2. " + shortOID(oids[0]) + " first · from side", "Active: whole commits (2) · Inactive: 1 selected files are not applied", previewReady} {
		if !strings.Contains(preview, want) {
			t.Errorf("T99 preview missing %q:\n%s", want, preview)
		}
	}
	pressAndDrain(gitflowApp, "esc")
	if fixture.Snapshot() != before {
		t.Fatal("preview changed Git state")
	}

	pressAndDrain(gitflowApp, "a")
	if !strings.Contains(overlayText(t, gitflowApp), cherryPickExecuteTitle) {
		t.Fatal("a menu does not list the explicit whole-commit Execute")
	}
	pressAndDrain(gitflowApp, "esc")
	runRegistered(gitflowApp, actionCherryPickExecute)
	if subjects := fixture.Git("log", "--format=%s", "--reverse", reviewedHead+"..HEAD"); subjects != "third\nfirst\n" {
		t.Fatalf("T15/T99 picked %q, want third then first", subjects)
	}
	if gitflowApp.notice != "Cherry-picked 2 commits onto main" || commits.selection.Len() != 0 {
		t.Fatalf("outcome %q, queue %v", gitflowApp.notice, commits.selection.Ordered())
	}
	if checkedOut(fixture) != "main" || readWorktree(t, fixture, "notes.txt") != "keep\n" {
		t.Fatal("Execute switched branch or touched the untracked file")
	}
	if selectedFile != "extra.txt" && selectedFile != "third.txt" {
		t.Fatalf("selected file %s not from the inspected commit", selectedFile)
	}
	if strings.TrimSpace(fixture.Git("status", "--porcelain", "--untracked-files=no")) != "" {
		t.Fatal("T99 individual file selection was applied separately")
	}
}

func TestCherryPickHandoffFromHistory(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
	before := fixture.Snapshot()
	fixture.Git("switch", "-q", sideBranch)
	drain(gitflowApp, gitflowApp.refresh())
	history := openHistory(t, gitflowApp)
	oids := queueSubjects(t, gitflowApp, history, "second", "first")
	fixture.Git("switch", "-q", "main")
	drain(gitflowApp, gitflowApp.refresh())
	explorer := openCherryPick(t, gitflowApp)
	if !slices.Equal(explorer.commits.selection.Ordered(), oids) || explorer.commits.cursor.SourceRef != "refs/heads/"+sideBranch {
		t.Fatalf("T95 S06 queue %v source %s, want %v from side", explorer.commits.selection.Ordered(), explorer.commits.cursor.SourceRef, oids)
	}
	if len(oids[0]) != 40 || !strings.Contains(gitflowApp.View(), "Whole commits (active) 2 queued") {
		t.Fatal("T95 full OIDs or selection count not preserved")
	}

	browseSource(t, gitflowApp, explorer, "refs/heads/main")
	queueSubjects(t, gitflowApp, explorer.commits, "main")
	if explorer.commits.selection.Len() != 3 || explorer.commits.selected[oids[0]].source != sideBranch || !strings.Contains(gitflowApp.View(), "2 hidden") {
		t.Fatalf("T98 cross-source queue lost context: %v %+v", explorer.commits.selection.Ordered(), explorer.commits.selected)
	}
	if fixture.Snapshot() != before {
		t.Fatal("T98 browsing two sources changed Git state")
	}
}

func TestCherryPickModeSeparation(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
	before := fixture.Snapshot()
	explorer := openCherryPick(t, gitflowApp)
	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	queueSubjects(t, gitflowApp, explorer.commits, "first", "second")
	focusSubject(t, gitflowApp, explorer.commits, "third")
	pressAndDrain(gitflowApp, "enter", " ", "j", " ", "esc")
	focusSubject(t, gitflowApp, explorer.commits, "first")
	pressAndDrain(gitflowApp, "enter", " ", "esc")
	browseSource(t, gitflowApp, explorer, "refs/heads/main")
	view := gitflowApp.View()
	for _, want := range []string{"Whole commits (active) 2 queued · Files (inactive) 3 selected", "2 selected · 2 hidden"} {
		if !strings.Contains(view, want) {
			t.Errorf("T133 view missing %q:\n%s", want, view)
		}
	}
	focusSubject(t, gitflowApp, explorer.commits, "main")
	pressAndDrain(gitflowApp, "enter")
	if !strings.Contains(gitflowApp.View(), "Mode Files (active) 3 selected · Whole commits (inactive) 2 queued") {
		t.Fatalf("T133 S13 mode line wrong:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc", "p")
	if preview := overlayText(t, gitflowApp); !strings.Contains(preview, "Active: whole commits (2) · Inactive: 3 selected files are not applied") || !strings.Contains(preview, "creates 2 new commits") {
		t.Fatalf("T133 preview consequence does not match the active mode only:\n%s", preview)
	}
	if fixture.Snapshot() != before {
		t.Fatal("T133 switching modes and sources changed Git state")
	}
}

func TestCherryPickGuards(t *testing.T) {
	testCases := []struct {
		name       string
		prepare    func(fixture *gittest.Fixture)
		wantReason string
		wantBlock  string
	}{
		{name: "T89 staged tracked file", prepare: func(fixture *gittest.Fixture) {
			fixture.Write("tracked.txt", "staged\n")
			fixture.Git("add", "tracked.txt")
		}, wantReason: "Tracked changes: tracked.txt", wantBlock: "Blocked: tracked changes: tracked.txt"},
		{name: "unstaged tracked edit blocks until Slice 15 ST03 offer", prepare: func(fixture *gittest.Fixture) { fixture.Dirty("tracked.txt") }, wantReason: "Tracked changes: tracked.txt", wantBlock: "Blocked: tracked changes: tracked.txt"},
		{name: "T92 untracked collision", prepare: func(fixture *gittest.Fixture) { fixture.Write("first.txt", "mine\n") },
			wantReason: "Preview found blockers: untracked files would be overwritten: first.txt", wantBlock: "Blocked: untracked files would be overwritten: first.txt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
			testCase.prepare(fixture)
			drain(gitflowApp, gitflowApp.refresh())
			before := fixture.Snapshot()
			explorer := openCherryPick(t, gitflowApp)
			browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
			queueSubjects(t, gitflowApp, explorer.commits, "first")
			pressAndDrain(gitflowApp, "p")
			if preview := overlayText(t, gitflowApp); !strings.Contains(preview, testCase.wantBlock) || strings.Contains(preview, previewReady) {
				t.Fatalf("preview missing %q:\n%s", testCase.wantBlock, preview)
			}
			pressAndDrain(gitflowApp, "esc")
			if available, reason := availability(gitflowApp, actionCherryPickExecute); available || reason != testCase.wantReason {
				t.Fatalf("Execute = %v %q, want disabled with %q", available, reason, testCase.wantReason)
			}
			runRegistered(gitflowApp, actionCherryPickExecute)
			if fixture.Snapshot() != before || fixture.Git("stash", "list") != "" {
				t.Fatal("blocked Execute changed Git state or created a stash")
			}
		})
	}
	t.Run("T93 paused operation blocks the new plan", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
		drain(gitflowApp, gitflowApp.Init())
		before := fixture.Snapshot()
		explorer := openCherryPick(t, gitflowApp)
		browseSource(t, gitflowApp, explorer, "refs/heads/"+gittest.FeatureBranch)
		queueSubjects(t, gitflowApp, explorer.commits, "feature 2")
		pressAndDrain(gitflowApp, "p")
		if preview := overlayText(t, gitflowApp); !strings.Contains(preview, "Blocked: a paused operation is in progress") || !strings.Contains(preview, "Blocked: unmerged files: conflict.txt") {
			t.Fatalf("T93 preview:\n%s", preview)
		}
		pressAndDrain(gitflowApp, "esc")
		if available, _ := availability(gitflowApp, actionCherryPickExecute); available {
			t.Fatal("T93 Execute available during a paused merge")
		}
		pressAndDrain(gitflowApp, "o")
		if _, open := gitflowApp.top().(*pausedScreen); !open || !strings.Contains(gitflowApp.View(), "Paused Operation Dashboard · merge") {
			t.Fatalf("T93 o opened %s", gitflowApp.top().ID())
		}
		if fixture.Snapshot() != before {
			t.Fatal("T93 paused state changed")
		}
	})
	t.Run("HEAD moved after preview", func(t *testing.T) {
		gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
		explorer := openCherryPick(t, gitflowApp)
		browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
		queueSubjects(t, gitflowApp, explorer.commits, "first")
		pressAndDrain(gitflowApp, "p", "esc")
		fixture.Commit("moved", map[string]string{"moved.txt": "moved\n"})
		drain(gitflowApp, gitflowApp.refresh())
		if available, reason := availability(gitflowApp, actionCherryPickExecute); available || reason != reasonPreviewFirst {
			t.Fatalf("Execute after HEAD moved = %v %q", available, reason)
		}
	})
}

func TestCherryPickPausesIntoDashboard(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, sideBranchCommits)
	explorer := openCherryPick(t, gitflowApp)
	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	queueSubjects(t, gitflowApp, explorer.commits, "first", "conflicting", "third")
	pressAndDrain(gitflowApp, "p", "esc")
	runRegistered(gitflowApp, actionCherryPickExecute)
	if _, open := gitflowApp.top().(*pausedScreen); !open {
		t.Fatalf("T101 pause opened %s, want Paused Operation Dashboard (S08)", gitflowApp.top().ID())
	}
	view := gitflowApp.View()
	for _, want := range []string{"Cherry-pick paused after 1 of 3 commits", "Paused: cherry-pick", "1 commits already applied", "1 more commits queued", "conflict.txt"} {
		if !strings.Contains(view, want) {
			t.Errorf("T101 view missing %q:\n%s", want, view)
		}
	}
	pausedState := fixture.Snapshot()
	pressAndDrain(gitflowApp, "esc", "esc")
	if fixture.Snapshot() != pausedState {
		t.Fatal("T101 Esc changed the paused cherry-pick")
	}

	restarted := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(restarted, restarted.Init())
	pressAndDrain(restarted, "o")
	if view := restarted.View(); !strings.Contains(view, "1 commits already applied") || !strings.Contains(view, "1 more commits queued") {
		t.Fatalf("T127 restart lost sequencer progress:\n%s", view)
	}
	if subjects := fixture.Git("log", "--format=%s", "-2"); subjects != "first\nmain\n" {
		t.Fatalf("T127 history after restart = %q, want first applied once", subjects)
	}
}

func TestCherryPickEmptyPickPauses(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, func(fixture *gittest.Fixture) {
		fixture.Commit("base", gittest.ConflictBase)
		fixture.Git("switch", "-q", "-c", sideBranch)
		fixture.Commit("duplicate", gittest.ConflictMain)
		fixture.Git("switch", "-q", "main")
		fixture.Commit("main", gittest.ConflictMain)
	})
	head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	explorer := openCherryPick(t, gitflowApp)
	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	queueSubjects(t, gitflowApp, explorer.commits, "duplicate")
	pressAndDrain(gitflowApp, "p", "esc")
	runRegistered(gitflowApp, actionCherryPickExecute)
	if strings.TrimSpace(fixture.Git("rev-parse", "HEAD")) != head || !strings.Contains(gitflowApp.notice, "paused after 0 of 1") {
		t.Fatalf("T27 empty pick reported %q", gitflowApp.notice)
	}
	if available, reason := availability(gitflowApp, actionOperationSkip); !available {
		t.Fatalf("T27 Skip unavailable: %s", reason)
	}
	if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", "CHERRY_PICK_HEAD")); err != nil {
		t.Fatalf("T27 no native paused cherry-pick: %v", err)
	}
}

const (
	applyMiddle           = "2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n"
	readOnlyDirectoryMode = 0o555
	writableDirectoryMode = 0o755
	stashOfferMarker      = "Stash and"
)

func selectiveApplyCommits(fixture *gittest.Fixture) {
	fixture.Commit("base", map[string]string{"tracked.txt": "tracked\n", "shared.txt": "top\n" + applyMiddle + "bottom\n", "keep.txt": "keep\n", "docs/readme.txt": "readme\n"})
	fixture.Git("switch", "-q", "-c", sideBranch)
	fixture.Commit("edit shared", map[string]string{"shared.txt": "SIDE\n" + applyMiddle + "bottom\n"})
	fixture.Git("rm", "-q", "keep.txt")
	fixture.Commit("add files", map[string]string{"added.txt": "added\n", "docs/guide.txt": "guide\n", "image.bin": "\x00\x01"})
	fixture.Commit("edit bottom", map[string]string{"shared.txt": "SIDE\n" + applyMiddle + "BOTTOM\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", map[string]string{"shared.txt": "top\n" + strings.Replace(applyMiddle, "7\n", "SEVEN\n", 1) + "bottom\n"})
}

func newSelectiveApplyApp(t *testing.T) (*App, *gittest.Fixture, *cherryPickScreen) {
	t.Helper()
	gitflowApp, fixture := newHistoryApp(t, selectiveApplyCommits)
	explorer := openCherryPick(t, gitflowApp)
	browseSource(t, gitflowApp, explorer, "refs/heads/"+sideBranch)
	return gitflowApp, fixture, explorer
}

func selectCommitFiles(t *testing.T, gitflowApp *App, commits *historyScreen, subject string, paths ...string) {
	t.Helper()
	focusSubject(t, gitflowApp, commits, subject)
	pressAndDrain(gitflowApp, "enter")
	inspector, open := gitflowApp.top().(*commitFilesScreen)
	if !open {
		t.Fatalf("Enter opened %s, want Commit File Inspector (S13)", gitflowApp.top().ID())
	}
	for _, path := range paths {
		target := slices.IndexFunc(inspector.files, func(file git.CommitFile) bool { return file.Path == path })
		if target < 0 {
			t.Fatalf("%s not listed in %q", path, subject)
		}
		for inspector.row > target {
			pressAndDrain(gitflowApp, "k")
		}
		for inspector.row < target {
			pressAndDrain(gitflowApp, "j")
		}
		pressAndDrain(gitflowApp, " ")
	}
}

func openApplyPlanner(t *testing.T, gitflowApp *App) *selectiveApplyScreen {
	t.Helper()
	pressAndDrain(gitflowApp, "p")
	planner, open := gitflowApp.top().(*selectiveApplyScreen)
	if !open || planner.preview == nil {
		t.Fatalf("p opened %s, want a previewed Apply Files Planner (S14)", gitflowApp.top().ID())
	}
	return planner
}

func gitLines(fixture *gittest.Fixture, args ...string) string {
	return strings.TrimSpace(fixture.Git(args...))
}

func TestSelectiveApplyStagesFiles(t *testing.T) {
	gitflowApp, fixture, explorer := newSelectiveApplyApp(t)
	fixture.Write("notes.txt", "keep\n")
	drain(gitflowApp, gitflowApp.refresh())
	reviewedHead := gitLines(fixture, "rev-parse", "HEAD")
	selectCommitFiles(t, gitflowApp, explorer.commits, "edit shared", "shared.txt")
	if available, reason := availability(gitflowApp, actionApplyExecute); available || reason != reasonPreviewFiles {
		t.Fatalf("T100 Execute before preview = %v %q", available, reason)
	}
	before := fixture.Snapshot()

	planner := openApplyPlanner(t, gitflowApp)
	view := gitflowApp.View()
	for _, want := range []string{"Apply Files Planner · 1 files from 1 commits onto main", "creates no commit", "Group 1 · ", "1. M shared.txt", applyReady} {
		if !strings.Contains(view, want) {
			t.Errorf("T100 planner missing %q:\n%s", want, view)
		}
	}
	if fixture.Snapshot() != before {
		t.Fatal("T100 preview changed Git state")
	}
	pressAndDrain(gitflowApp, "a")
	if menu := overlayText(t, gitflowApp); !strings.Contains(menu, applyExecuteTitle) || strings.Contains(menu, stashOfferMarker) {
		t.Fatalf("T100/T170 a menu:\n%s", menu)
	}
	pressAndDrain(gitflowApp, "esc")
	runRegistered(gitflowApp, actionApplyExecute)

	if staged := gitLines(fixture, "diff", "--cached", "--name-status"); staged != "M\tshared.txt" {
		t.Fatalf("T115 staged %q, want only shared.txt", staged)
	}
	if unstaged := gitLines(fixture, "diff", "--name-only"); unstaged != "" {
		t.Fatalf("T115 left unstaged changes %q", unstaged)
	}
	if want := "SIDE\n" + strings.Replace(applyMiddle, "7\n", "SEVEN\n", 1) + "bottom\n"; readWorktree(t, fixture, "shared.txt") != want {
		t.Fatalf("T115 applied a snapshot instead of the patch:\n%s", readWorktree(t, fixture, "shared.txt"))
	}
	if gitLines(fixture, "rev-parse", "HEAD") != reviewedHead || readWorktree(t, fixture, "notes.txt") != "keep\n" {
		t.Fatal("T115 created a commit or touched the untracked file")
	}
	if planner.history.files.Len() != 0 || gitflowApp.notice != "Staged 1 selected files onto main · no commit created · "+messagePreparedNotice {
		t.Fatalf("outcome %q, %d files still selected", gitflowApp.notice, planner.history.files.Len())
	}
	source := planner.steps[0].Selection.OID
	if want := "Cherry-pick selected files from " + shortOID(source) + ": edit shared\n\nSource commit: " + source + "\n- shared.txt"; gitflowApp.changes.draft.Value() != want {
		t.Fatalf("message %q, want %q", gitflowApp.changes.draft.Value(), want)
	}
	if !strings.Contains(gitflowApp.View(), applyAppliedLabel+"shared.txt") {
		t.Fatal("result does not list the applied file")
	}
	pressAndDrain(gitflowApp, "esc")
	if _, inspecting := gitflowApp.top().(*commitFilesScreen); !inspecting {
		t.Fatalf("Esc from S14 opened %s, want S13", gitflowApp.top().ID())
	}
}

func TestSelectiveApplyKeepsDraft(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		key       string
		wantDraft func(prepared string) string
	}{
		{"Esc keeps the draft", "esc", func(string) string { return "my draft" }},
		{"c replaces the draft", "c", func(prepared string) string { return prepared }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, explorer := newSelectiveApplyApp(t)
			gitflowApp.changes.draft.SetValue("my draft")
			selectCommitFiles(t, gitflowApp, explorer.commits, "add files", "keep.txt", "added.txt")
			pressAndDrain(gitflowApp, "esc")
			selectCommitFiles(t, gitflowApp, explorer.commits, "edit shared", "shared.txt")
			planner := openApplyPlanner(t, gitflowApp)
			var order []string
			for _, step := range planner.steps {
				order = append(order, step.Selection.Path)
			}
			if !slices.Equal(order, []string{"keep.txt", "added.txt", "shared.txt"}) {
				t.Fatalf("T116 step order %v, want first-selected commit then file order", order)
			}
			runRegistered(gitflowApp, actionApplyExecute)
			if staged := gitLines(fixture, "diff", "--cached", "--name-status"); staged != "A\tadded.txt\nD\tkeep.txt\nM\tshared.txt" {
				t.Fatalf("T116 staged %q", staged)
			}
			if gitflowApp.changes.draft.Value() != "my draft" || !strings.Contains(overlayText(t, gitflowApp), "Cherry-pick selected files from 2 commits") {
				t.Fatal("T116 draft replaced before a deliberate choice")
			}
			prepared := strings.Join([]string{"Cherry-pick selected files from 2 commits", "",
				planner.steps[0].Selection.OID + " add files", "- keep.txt", "- added.txt",
				planner.steps[2].Selection.OID + " edit shared", "- shared.txt"}, "\n")
			pressAndDrain(gitflowApp, testCase.key)
			if gitflowApp.overlay != nil || gitflowApp.changes.draft.Value() != testCase.wantDraft(prepared) {
				t.Fatalf("T116 after %s draft %q", testCase.key, gitflowApp.changes.draft.Value())
			}
		})
	}
}

func TestSelectiveApplyBlockers(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		subject     string
		path        string
		setup       func(fixture *gittest.Fixture)
		wantBlocker string
	}{
		{"T117 binary file", "add files", "image.bin", func(*gittest.Fixture) {}, "image.bin (binary file)"},
		{"T117 untracked collision", "add files", "added.txt", func(fixture *gittest.Fixture) { fixture.Write("added.txt", "mine\n") }, "added.txt"},
		{"T170/T148 dirty tracked tree", "edit shared", "shared.txt", func(fixture *gittest.Fixture) { fixture.Write("tracked.txt", "dirty\n") }, "tracked.txt"},
		{"T148 staged tracked change", "edit shared", "shared.txt", func(fixture *gittest.Fixture) {
			fixture.Write("tracked.txt", "staged\n")
			fixture.Git("add", "tracked.txt")
		}, "tracked.txt"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, explorer := newSelectiveApplyApp(t)
			testCase.setup(fixture)
			drain(gitflowApp, gitflowApp.refresh())
			before := fixture.Snapshot()
			selectCommitFiles(t, gitflowApp, explorer.commits, testCase.subject, testCase.path)
			openApplyPlanner(t, gitflowApp)
			if view := flattenedView(gitflowApp); !strings.Contains(view, "Blocked: ") || !strings.Contains(view, testCase.wantBlocker) {
				t.Fatalf("preview does not block %q:\n%s", testCase.wantBlocker, view)
			}
			if available, reason := availability(gitflowApp, actionApplyExecute); available || !strings.Contains(reason, testCase.wantBlocker) {
				t.Fatalf("Execute = %v %q, want blocked by %q", available, reason, testCase.wantBlocker)
			}
			pressAndDrain(gitflowApp, "a")
			if menu := overlayText(t, gitflowApp); strings.Contains(menu, stashOfferMarker) {
				t.Fatalf("T170 S14 offers a stash:\n%s", menu)
			}
			pressAndDrain(gitflowApp, "esc")
			runRegistered(gitflowApp, actionApplyExecute)
			if fixture.Snapshot() != before || gitLines(fixture, "stash", "list") != "" {
				t.Fatal("blocked Execute changed Git state or created a stash")
			}
		})
	}
}

func TestSelectiveApplyOverlapWarning(t *testing.T) {
	gitflowApp, _, explorer := newSelectiveApplyApp(t)
	selectCommitFiles(t, gitflowApp, explorer.commits, "edit shared", "shared.txt")
	pressAndDrain(gitflowApp, "esc")
	selectCommitFiles(t, gitflowApp, explorer.commits, "edit bottom", "shared.txt")
	openApplyPlanner(t, gitflowApp)
	view := flattenedView(gitflowApp)
	if !strings.Contains(view, "Overlap: shared.txt is changed by steps 1, 2") || !strings.Contains(view, applyReady) {
		t.Fatalf("T116/T117 overlap applying in order:\n%s", view)
	}
}

func TestSelectiveApplyPartialFailure(t *testing.T) {
	gitflowApp, fixture, explorer := newSelectiveApplyApp(t)
	selectCommitFiles(t, gitflowApp, explorer.commits, "edit shared", "shared.txt")
	pressAndDrain(gitflowApp, "esc")
	selectCommitFiles(t, gitflowApp, explorer.commits, "add files", "docs/guide.txt", "added.txt")
	planner := openApplyPlanner(t, gitflowApp)
	docs := filepath.Join(fixture.Dir, "docs")
	if err := os.Chmod(docs, readOnlyDirectoryMode); err != nil {
		t.Fatalf("make docs read-only: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(docs, writableDirectoryMode); err != nil {
			t.Errorf("restore docs permissions: %v", err)
		}
	})

	runRegistered(gitflowApp, actionApplyExecute)
	view := flattenedView(gitflowApp)
	for _, want := range []string{applyAppliedLabel + "shared.txt", applyFailedLabel + "docs/guide.txt", applyPendingLabel + "added.txt", applyNoRollback, "Tracked changes now (1): shared.txt"} {
		if !strings.Contains(view, want) {
			t.Errorf("T117 result missing %q:\n%s", want, view)
		}
	}
	if gitflowApp.errorText == "" || gitflowApp.changes.draft.Value() != "" || gitflowApp.overlay != nil {
		t.Fatal("T117 partial apply hid the error or prepared a message")
	}
	if staged := gitLines(fixture, "diff", "--cached", "--name-status"); staged != "M\tshared.txt" {
		t.Fatalf("T117 staged %q, want only the applied step kept", staged)
	}
	if planner.history.files.Len() != 3 {
		t.Fatalf("partial apply changed the file selections: %d left, want all 3 kept", planner.history.files.Len())
	}
}

func TestSelectiveApplyCarriesHistoryFiles(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, selectiveApplyCommits)
	fixture.Git("switch", "-q", sideBranch)
	drain(gitflowApp, gitflowApp.refresh())
	history := openHistory(t, gitflowApp)
	selectCommitFiles(t, gitflowApp, history, "edit shared", "shared.txt")
	if available, reason := availability(gitflowApp, actionApplyPreview); available || reason != reasonFileModeClosed {
		t.Fatalf("p from History S13 = %v %q, want %q", available, reason, reasonFileModeClosed)
	}
	pressAndDrain(gitflowApp, "esc")
	fixture.Git("switch", "-q", "main")
	drain(gitflowApp, gitflowApp.refresh())
	explorer := openCherryPick(t, gitflowApp)
	carried := explorer.commits.orderedFiles()
	if len(carried) != 1 || carried[0].Path != "shared.txt" || explorer.commits.fileSources[carried[0].OID].summary.Subject != "edit shared" {
		t.Fatalf("g c carried %+v", carried)
	}
}
