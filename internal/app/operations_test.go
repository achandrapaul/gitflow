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

const conflictedPath = "conflict.txt"

func newPausedApp(t *testing.T, fixture *gittest.Fixture) *App {
	t.Helper()
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	return gitflowApp
}

func openResolver(t *testing.T, gitflowApp *App) *conflictScreen {
	t.Helper()
	pressAndDrain(gitflowApp, "o", "enter")
	conflict, open := gitflowApp.conflictOnTop()
	if !open || !conflict.loaded || conflict.err != nil {
		t.Fatalf("Enter on the Paused Operation Dashboard opened %s, err %v", gitflowApp.top().ID(), conflict)
	}
	return conflict
}

func runRegistered(gitflowApp *App, id ActionID) {
	action, _ := gitflowApp.registry.ByID(id)
	drain(gitflowApp, gitflowApp.runAction(action))
}

func readWorktree(t *testing.T, fixture *gittest.Fixture, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(fixture.Dir, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func TestPausedMergeDashboardAndResolver(t *testing.T) {
	fixture := gittest.NewPausedFixture(t, git.OperationMerge)
	gitflowApp := newPausedApp(t, fixture)
	before := fixture.Snapshot()

	if !strings.Contains(gitflowApp.View(), pausedPrefix+"merge") {
		t.Fatal("T17 header has no paused merge badge after start")
	}
	pressAndDrain(gitflowApp, "o")
	view := gitflowApp.View()
	for _, want := range []string{"Paused Operation Dashboard · merge", conflictedPath, "Reason: conflicts in 1 files", reasonMergeHasNoSkip, "Resolve 1 conflicted files first"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Paused Operation Dashboard lacks %q:\n%s", want, view)
		}
	}
	pressAndDrain(gitflowApp, "s")
	pressAndDrain(gitflowApp, "c")
	if gitflowApp.overlay != nil || fixture.Snapshot() != before {
		t.Fatalf("T22 merge Skip or Continue with conflicts opened %s or changed Git state", overlayID(gitflowApp))
	}

	conflict := openResolver(t, gitflowApp)
	view = gitflowApp.View()
	for _, want := range []string{"main (current)", gittest.FeatureBranch + " (incoming)", "common ancestor", "1 Choose current side", "2 Choose base", "3 Choose incoming side", "MAIN", "FEATURE"} {
		if !strings.Contains(view, want) {
			t.Fatalf("T118 Conflict Resolver lacks %q:\n%s", want, view)
		}
	}
	if fixture.Snapshot() != before {
		t.Fatal("opening the Conflict Resolver changed Git state")
	}

	runRegistered(gitflowApp, actionConflictResolved)
	if gitflowApp.overlay != nil {
		t.Fatal("Mark resolved offered while conflict markers remain")
	}
	pressAndDrain(gitflowApp, "3")
	if got := readWorktree(t, fixture, conflictedPath); got != gittest.FeatureOne[conflictedPath] {
		t.Fatalf("choose incoming wrote %q", got)
	}
	if !slices.Contains(gitflowApp.repository.status.Unmerged, conflictedPath) || len(conflict.file.Regions) != 0 {
		t.Fatalf("choosing a side staged the file or left regions: unmerged %v regions %d", gitflowApp.repository.status.Unmerged, len(conflict.file.Regions))
	}

	runRegistered(gitflowApp, actionConflictResolved)
	if overlayID(gitflowApp) != ScreenConfirm || !strings.Contains(gitflowApp.View(), "git add -- "+conflictedPath) {
		t.Fatalf("Mark resolved opened %s without the git add command", overlayID(gitflowApp))
	}
	pressAndDrain(gitflowApp, "enter")
	if !slices.Contains(gitflowApp.repository.status.Unmerged, conflictedPath) {
		t.Fatal("Enter on the Mark resolved Confirmation staged the file instead of cancelling")
	}
	runRegistered(gitflowApp, actionConflictResolved)
	pressAndDrain(gitflowApp, keyConfirm)
	if len(gitflowApp.repository.status.Unmerged) != 0 || gitflowApp.top().ID() != ScreenPaused || !gitflowApp.repository.operation.Active() {
		t.Fatalf("Mark resolved: unmerged %v, top %s, operation %+v", gitflowApp.repository.status.Unmerged, gitflowApp.top().ID(), gitflowApp.repository.operation)
	}

	pressAndDrain(gitflowApp, "c")
	if gitflowApp.repository.operation.Active() || strings.TrimSpace(fixture.Git("rev-parse", "HEAD^2")) != strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch)) {
		t.Fatalf("Continue did not complete the merge: %+v, error %q", gitflowApp.repository.operation, gitflowApp.errorText)
	}
}

func TestPausedNavigationLeavesGitUntouched(t *testing.T) {
	fixture := gittest.NewPausedFixture(t, git.OperationCherryPick)
	gitflowApp := newPausedApp(t, fixture)
	before := fixture.Snapshot()

	openResolver(t, gitflowApp)
	pressAndDrain(gitflowApp, "n", "N", "esc")
	if gitflowApp.top().ID() != ScreenPaused || fixture.Snapshot() != before {
		t.Fatalf("T16 Esc from the Conflict Resolver reached %s or changed Git state", gitflowApp.top().ID())
	}
	pressAndDrain(gitflowApp, "esc", "esc")
	press(gitflowApp, "q")
	if fixture.Snapshot() != before {
		t.Fatal("T28 root Esc and quit changed the paused cherry-pick")
	}

	restarted := newPausedApp(t, fixture)
	pressAndDrain(restarted, "o")
	view := restarted.View()
	for _, want := range []string{"Paused Operation Dashboard · cherry-pick", "s Skip", "A Abort", "more commits queued"} {
		if !strings.Contains(view, want) {
			t.Fatalf("T17 restarted dashboard lacks %q:\n%s", want, view)
		}
	}
	skip, _ := restarted.registry.ByID(actionOperationSkip)
	if available, reason := skip.Availability(restarted); !available {
		t.Fatalf("T17 Skip unavailable for a paused cherry-pick: %s", reason)
	}

	fixture.Write(conflictedPath, gittest.FeatureOne[conflictedPath])
	fixture.Git("add", "--", conflictedPath)
	pressAndDrain(restarted, "r")
	continueAction, _ := restarted.registry.ByID(actionOperationContinue)
	if available, reason := continueAction.Availability(restarted); !available || !restarted.repository.operation.Active() {
		t.Fatalf("r Refresh after an external resolve: Continue %v %q, operation %+v", available, reason, restarted.repository.operation)
	}
}

func TestPausedRebaseSkipAndAbortNeedConfirmation(t *testing.T) {
	fixture := gittest.NewPausedFixture(t, git.OperationRebase)
	featureTip := strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch))
	gitflowApp := newPausedApp(t, fixture)
	before := fixture.Snapshot()

	openResolver(t, gitflowApp)
	view := gitflowApp.View()
	for _, want := range []string{"main (rebase destination)", "(replayed commit)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("T119 rebase resolver lacks %q:\n%s", want, view)
		}
	}
	pressAndDrain(gitflowApp, "esc")

	for _, cancel := range []string{"enter", "esc"} {
		pressAndDrain(gitflowApp, "s")
		if overlayID(gitflowApp) != ScreenConfirm || !strings.Contains(gitflowApp.View(), "git rebase --skip") {
			t.Fatalf("s opened %s without git rebase --skip", overlayID(gitflowApp))
		}
		pressAndDrain(gitflowApp, cancel)
		if gitflowApp.overlay != nil || fixture.Snapshot() != before {
			t.Fatalf("%s on the Skip Confirmation did not cancel cleanly", cancel)
		}
	}
	pressAndDrain(gitflowApp, "s", keyConfirmGuarded)
	if _, err := os.Stat(filepath.Join(fixture.Dir, "second.txt")); err != nil || gitflowApp.errorText != "" || gitflowApp.repository.operation.Active() {
		t.Fatalf("Skip did not replay the next commit: %v, error %q", err, gitflowApp.errorText)
	}

	fixture = gittest.NewPausedFixture(t, git.OperationRebase)
	featureTip = strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch))
	gitflowApp = newPausedApp(t, fixture)
	pressAndDrain(gitflowApp, "o", "A")
	if overlayID(gitflowApp) != ScreenConfirm || !strings.Contains(gitflowApp.View(), "git rebase --abort") || !strings.Contains(gitflowApp.View(), "not a universal undo") {
		t.Fatalf("A opened %s without the abort review", overlayID(gitflowApp))
	}
	pressAndDrain(gitflowApp, keyConfirmGuarded)
	head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	if gitflowApp.repository.operation.Active() || head != featureTip || gitflowApp.errorText != "" {
		t.Fatalf("Abort left %+v at %s, error %q", gitflowApp.repository.operation, head, gitflowApp.errorText)
	}
}

func TestPausedRevertShownWithoutControls(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", gittest.ConflictBase)
	fixture.Commit("change", gittest.FeatureOne)
	fixture.Commit("other", gittest.ConflictMain)
	fixture.GitFails("revert", "HEAD~1")
	gitflowApp := newPausedApp(t, fixture)
	before := fixture.Snapshot()

	pressAndDrain(gitflowApp, "o")
	view := gitflowApp.View()
	if !strings.Contains(view, "Paused Operation Dashboard · revert") || strings.Count(view, reasonRevertUnsupported) != 3 {
		t.Fatalf("revert dashboard does not mark Continue, Skip and Abort unavailable:\n%s", view)
	}
	pressAndDrain(gitflowApp, "enter", "c", "s", "A")
	if gitflowApp.top().ID() != ScreenPaused || gitflowApp.overlay != nil || fixture.Snapshot() != before {
		t.Fatalf("revert controls reached %s / %s or changed Git state", gitflowApp.top().ID(), overlayID(gitflowApp))
	}
}

func TestConflictResolverFallbacksAndEditor(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.ConflictingBranches(map[string]string{"image.bin": "base\x00"}, map[string]string{"image.bin": "main\x00"}, map[string]string{"image.bin": "feature\x00"})
	fixture.GitFails("merge", gittest.FeatureBranch)
	gitflowApp := newPausedApp(t, fixture)
	before := fixture.Snapshot()

	openResolver(t, gitflowApp)
	if !strings.Contains(gitflowApp.View(), "Binary conflict") {
		t.Fatalf("T24 resolver hides the binary fallback:\n%s", gitflowApp.View())
	}
	for _, id := range []ActionID{actionConflictOurs, actionConflictBase, actionConflictTheirs} {
		action, _ := gitflowApp.registry.ByID(id)
		if available, reason := action.Availability(gitflowApp); available || reason != reasonEditorOnly {
			t.Fatalf("T24 %s on a binary conflict: %v %q", action.Title, available, reason)
		}
	}
	pressAndDrain(gitflowApp, "1", "2", "3")
	if fixture.Snapshot() != before {
		t.Fatal("T24 side keys changed a binary conflict")
	}

	ready, isReady := press(gitflowApp, "e")().(handoffReadyMsg)
	want := []string{"sh", "-c", `true "$@"`, "true", filepath.Join(fixture.Dir, "image.bin")}
	if !isReady || ready.err != nil || !slices.Equal(ready.command.Args, want) || ready.command.Dir != fixture.Dir {
		t.Fatalf("e prepared %+v, want %v in %s", ready, want, fixture.Dir)
	}
}
