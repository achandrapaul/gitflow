package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func newBranchApp(t *testing.T, setup func(fixture *gittest.Fixture)) (*App, *gittest.Fixture) {
	t.Helper()
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	fixture.Commit("base", map[string]string{"shared.txt": "base\n", "differs.txt": "base\n"})
	fixture.Git("push", "-q", "origin", "main")
	fixture.Git("fetch", "-q", "origin")
	fixture.Git("switch", "-q", "-c", "feature")
	fixture.Commit("feature", map[string]string{"differs.txt": "feature\n", "added.txt": "feature\n"})
	fixture.Git("switch", "-q", "main")
	if setup != nil {
		setup(fixture)
	}
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	return gitflowApp, fixture
}

func openBranches(t *testing.T, gitflowApp *App, keys ...string) *branchScreen {
	t.Helper()
	pressAndDrain(gitflowApp, keys...)
	branches, open := gitflowApp.top().(*branchScreen)
	if !open {
		t.Fatalf("%v opened %s, want S11", keys, gitflowApp.top().ID())
	}
	return branches
}

func focusBranch(t *testing.T, gitflowApp *App, branches *branchScreen, ref string) {
	t.Helper()
	for range len(branches.branches) {
		if focused, found := branches.focused(); found && focused.Ref == ref {
			return
		}
		press(gitflowApp, "j")
	}
	t.Fatalf("%s never focused in %+v", ref, branches.branches)
}

func typeText(gitflowApp *App, text string) {
	for _, letter := range text {
		pressAndDrain(gitflowApp, string(letter))
	}
}

func branchOID(fixture *gittest.Fixture, ref string) string {
	return strings.TrimSpace(fixture.Git("rev-parse", ref))
}

func checkedOut(fixture *gittest.Fixture) string {
	return strings.TrimSpace(fixture.Git("branch", "--show-current"))
}

func TestBranchesOpenSearchAndInspectWithoutCheckout(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, nil)
	before := fixture.Snapshot()
	featureOID := branchOID(fixture, "refs/heads/feature")

	globalBranches := openBranches(t, gitflowApp, "g", "b")
	globalView := flattenedView(gitflowApp)
	for _, want := range []string{branchesTitle, "* main", "feature " + featureOID[:shortOIDLength], "origin/main", "Local refs · last fetched"} {
		if !strings.Contains(globalView, want) {
			t.Errorf("T55 S11 missing %q:\n%s", want, globalView)
		}
	}
	pressAndDrain(gitflowApp, "b")
	if gitflowApp.top() != globalBranches || gitflowApp.notice != "" {
		t.Fatalf("T136 b gained a binding inside S11: top %s notice %q", gitflowApp.top().ID(), gitflowApp.notice)
	}

	focusBranch(t, gitflowApp, globalBranches, "refs/heads/feature")
	pressAndDrain(gitflowApp, "enter")
	inspection := flattenedView(gitflowApp)
	for _, want := range []string{"refs/heads/feature", featureOID, noUpstream, "Base origin/main ↑1 ↓0", "Local refs · last fetched"} {
		if !strings.Contains(inspection, want) {
			t.Errorf("T105 inspector missing %q:\n%s", want, inspection)
		}
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top() != globalBranches || globalBranches.inspecting {
		t.Fatalf("Esc did not close the inspector first")
	}

	pressAndDrain(gitflowApp, "/")
	typeText(gitflowApp, "feat")
	searchView := flattenedView(gitflowApp)
	if rows := globalBranches.visible(); !strings.Contains(searchView, branchSearchLabel) || len(rows) != 1 || rows[0].Ref != "refs/heads/feature" {
		t.Fatalf("T55 search must label its scope and filter rows to feature, rows %+v:\n%s", rows, searchView)
	}
	pressAndDrain(gitflowApp, "enter")
	if !globalBranches.inspecting || !strings.Contains(flattenedView(gitflowApp), featureOID) {
		t.Fatalf("Enter in search did not inspect feature:\n%s", flattenedView(gitflowApp))
	}
	pressAndDrain(gitflowApp, "esc", "esc", "esc")
	if gitflowApp.top().ID() != ScreenChanges {
		t.Fatalf("T55 Esc did not return to S01, top %s", gitflowApp.top().ID())
	}

	openBranches(t, gitflowApp, "b")
	if aliasView := flattenedView(gitflowApp); aliasView != globalView {
		t.Fatalf("T136 S01 b and g b differ:\n%s\nwant\n%s", aliasView, globalView)
	}
	pressAndDrain(gitflowApp, "esc")
	press(gitflowApp, "c", "b")
	if len(gitflowApp.stack) != 1 || gitflowApp.changes.draft.Value() != "b" {
		t.Fatalf("T55 b in the composer must be typed, top %s draft %q", gitflowApp.top().ID(), gitflowApp.changes.draft.Value())
	}
	if after := fixture.Snapshot(); after != before {
		t.Fatal("T55/T136 browsing, searching or inspecting changed Git state")
	}
}

func TestBranchSwitch(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(fixture *gittest.Fixture)
		focusRef   string
		wantBranch string
		wantNotice string
		wantText   []string
		stashHint  bool
	}{
		{name: "T105 eligible local branch", focusRef: "refs/heads/feature", wantBranch: "feature", wantNotice: "Switched to feature"},
		{name: "T105 remote-tracking ref", focusRef: "refs/remotes/origin/main", wantBranch: "main", wantText: []string{"origin/main", switchBlockerText[git.BlockedNotLocal]}},
		{name: "current branch", focusRef: "refs/heads/main", wantBranch: "main", wantText: []string{switchBlockerText[git.BlockedCurrent]}},
		{name: "dirty tracked path", setup: func(fixture *gittest.Fixture) { fixture.Dirty("differs.txt") }, focusRef: "refs/heads/feature", wantBranch: "main",
			wantText: []string{switchBlockerText[git.BlockedDirty], "differs.txt", stashAndSwitchHints}, stashHint: true},
		{name: "untracked collision", setup: func(fixture *gittest.Fixture) { fixture.Write("added.txt", "local\n") }, focusRef: "refs/heads/feature", wantBranch: "main",
			wantText: []string{switchBlockerText[git.BlockedUntracked], "added.txt"}},
		{name: "unmerged paths", setup: func(fixture *gittest.Fixture) {
			fixture.Write("shared.txt", "stashed\n")
			fixture.Git("stash", "-q")
			fixture.Commit("conflicting", map[string]string{"shared.txt": "committed\n"})
			fixture.GitFails("stash", "pop", "-q")
		}, focusRef: "refs/heads/feature", wantBranch: "main", wantText: []string{switchBlockerText[git.BlockedUnmerged], "shared.txt"}},
		{name: "paused operation", setup: func(fixture *gittest.Fixture) {
			fixture.Git("switch", "-q", "-c", "side")
			pick := fixture.Commit("side", map[string]string{"shared.txt": "side\n"})
			fixture.Git("switch", "-q", "main")
			fixture.Commit("main", map[string]string{"shared.txt": "main\n"})
			fixture.GitFails("cherry-pick", pick)
		}, focusRef: "refs/heads/feature", wantBranch: "main", wantText: []string{switchBlockerText[git.BlockedOperation]}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newBranchApp(t, testCase.setup)
			branches := openBranches(t, gitflowApp, "g", "b")
			focusBranch(t, gitflowApp, branches, testCase.focusRef)
			before := fixture.Snapshot()

			pressAndDrain(gitflowApp, "s")
			if got := checkedOut(fixture); got != testCase.wantBranch {
				t.Fatalf("checked out %q, want %q", got, testCase.wantBranch)
			}
			if gitflowApp.notice != testCase.wantNotice || gitflowApp.errorText != "" {
				t.Fatalf("notice %q error %q, want notice %q", gitflowApp.notice, gitflowApp.errorText, testCase.wantNotice)
			}
			if focused, _ := branches.focused(); gitflowApp.top() != branches || focused.Ref != testCase.focusRef {
				t.Fatalf("focus moved to %s on %s", focused.Ref, gitflowApp.top().ID())
			}
			if len(testCase.wantText) == 0 {
				if overlayID(gitflowApp) != "" || !strings.Contains(flattenedView(gitflowApp), "* feature") {
					t.Fatalf("switch did not refresh S11:\n%s", flattenedView(gitflowApp))
				}
				return
			}
			view := flattenedView(gitflowApp)
			for _, want := range testCase.wantText {
				if !strings.Contains(view, want) {
					t.Errorf("blocker view missing %q:\n%s", want, view)
				}
			}
			if strings.Contains(view, stashAndSwitchHints) != testCase.stashHint {
				t.Errorf("Stash and Switch offered = %v, want %v:\n%s", !testCase.stashHint, testCase.stashHint, view)
			}
			if after := fixture.Snapshot(); after != before {
				t.Fatal("blocked switch changed Git state")
			}
			if testCase.stashHint {
				pressAndDrain(gitflowApp, "s")
				if gitflowApp.notice != "Stash and switch: "+reasonNotAvailableYet {
					t.Fatalf("s in the blocker = notice %q", gitflowApp.notice)
				}
				return
			}
			pressAndDrain(gitflowApp, "esc")
			if overlayID(gitflowApp) != "" || gitflowApp.top() != branches || checkedOut(fixture) != "main" {
				t.Fatalf("Esc did not stay on main in S11")
			}
		})
	}
}

func TestCreateBranchFromHeadAndFocusedRef(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, nil)
	headOID := branchOID(fixture, "HEAD")
	featureOID := branchOID(fixture, "refs/heads/feature")
	branches := openBranches(t, gitflowApp, "g", "b")
	focusBranch(t, gitflowApp, branches, "refs/heads/feature")
	testCases := []struct {
		name     string
		keys     []string
		branch   string
		startOID string
	}{
		{name: "T137 g n starts at HEAD even with feature focused", keys: []string{"g", "n"}, branch: "from-head", startOID: headOID},
		{name: "T138 a n starts at the focused ref", keys: []string{"a", "n"}, branch: "from-feature", startOID: featureOID},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pressAndDrain(gitflowApp, testCase.keys...)
			if !strings.Contains(flattenedView(gitflowApp), testCase.startOID) {
				t.Fatalf("review does not show start OID %s:\n%s", testCase.startOID, flattenedView(gitflowApp))
			}
			typeText(gitflowApp, testCase.branch)
			pressAndDrain(gitflowApp, "enter")
			if got := branchOID(fixture, "refs/heads/"+testCase.branch); got != testCase.startOID {
				t.Fatalf("%s at %s, want %s", testCase.branch, got, testCase.startOID)
			}
			if checkedOut(fixture) != "main" || gitflowApp.notice != "Created "+testCase.branch+" · Still on main" {
				t.Fatalf("checked out %q notice %q", checkedOut(fixture), gitflowApp.notice)
			}
			if _, prompted := gitflowApp.overlay.(*switchPrompt); !prompted {
				t.Fatalf("overlay after create = %T, want the Switch? prompt", gitflowApp.overlay)
			}
			pressAndDrain(gitflowApp, "esc")
			if upstream := fixture.Git("for-each-ref", "--format=%(upstream)", "refs/heads/"+testCase.branch); strings.TrimSpace(upstream) != "" {
				t.Fatalf("created branch has upstream %q", upstream)
			}
			if focused, _ := branches.focused(); focused.Ref != "refs/heads/feature" || !slices.ContainsFunc(branches.branches, func(branch git.BranchInfo) bool { return branch.Name == testCase.branch }) {
				t.Fatalf("S11 lost focus %s or did not list the new branch", focused.Ref)
			}
		})
	}
}

func TestCreateBranchBlocksInvalidNames(t *testing.T) {
	testCases := []struct {
		name     string
		branch   string
		wantText string
	}{
		{name: "invalid name", branch: "bad..name", wantText: invalidBranchName},
		{name: "existing branch", branch: "feature", wantText: branchExists},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newBranchApp(t, nil)
			before := fixture.Snapshot()
			pressAndDrain(gitflowApp, "g", "n")
			typeText(gitflowApp, testCase.branch)
			if view := flattenedView(gitflowApp); !strings.Contains(view, testCase.wantText) {
				t.Fatalf("live validation missing %q:\n%s", testCase.wantText, view)
			}
			pressAndDrain(gitflowApp, "enter")
			if overlayID(gitflowApp) != ScreenConfirm || fixture.Snapshot() != before {
				t.Fatal("Enter with an invalid name closed the review or changed refs")
			}
		})
	}
}

func TestCreateBranchBlocksStaleAndMissingStart(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, nil)
	branches := openBranches(t, gitflowApp, "g", "b")
	focusBranch(t, gitflowApp, branches, "refs/heads/feature")
	pressAndDrain(gitflowApp, "a", "n")
	fixture.Git("update-ref", "refs/heads/feature", "refs/heads/main")
	typeText(gitflowApp, "late")
	pressAndDrain(gitflowApp, "enter")
	if !strings.Contains(gitflowApp.errorText, git.ErrStaleRef.Error()) {
		t.Fatalf("T138 stale start error = %q", gitflowApp.errorText)
	}
	if refs := fixture.Git("for-each-ref", "--format=%(refname)", "refs/heads/late"); refs != "" {
		t.Fatalf("stale start created %s", refs)
	}

	unborn := gittest.NewFixture(t)
	unbornApp := newTestAppFor(t, unborn, filepath.Join(t.TempDir(), "prefs.json"))
	drain(unbornApp, unbornApp.Init())
	pressAndDrain(unbornApp, "g", "n")
	if overlayID(unbornApp) != "" || unbornApp.notice != "Create local branch: "+reasonNoCommits {
		t.Fatalf("unborn HEAD g n = overlay %s notice %q", overlayID(unbornApp), unbornApp.notice)
	}
}

const overlayBorder = "│"

func unwrappedOverlayText(t *testing.T, gitflowApp *App) string {
	t.Helper()
	return strings.Join(strings.Fields(strings.ReplaceAll(overlayText(t, gitflowApp), overlayBorder, " ")), " ")
}

func checkoutState(fixture *gittest.Fixture) string {
	return fixture.Git("status", "--porcelain=v2", "-z", "--untracked-files=all") + fixture.Git("ls-files", "--stage") + checkedOut(fixture) + branchOID(fixture, "HEAD")
}

func behindOrigin(fixture *gittest.Fixture) string {
	remoteOID := fixture.PushFromOtherClone("main", "remote", map[string]string{"remote.txt": "remote\n"})
	fixture.Git("fetch", "-q", "origin")
	return remoteOID
}

func createFromMain(t *testing.T, gitflowApp *App, name string) {
	t.Helper()
	branches := openBranches(t, gitflowApp, "g", "b")
	focusBranch(t, gitflowApp, branches, "refs/heads/feature")
	pressAndDrain(gitflowApp, "a", "m")
	if _, named := gitflowApp.overlay.(*nameInput); !named {
		t.Fatalf("a m opened %T (error %q), want the name input", gitflowApp.overlay, gitflowApp.errorText)
	}
	typeText(gitflowApp, name)
	pressAndDrain(gitflowApp, "enter")
}

func TestCreateFromMainChoices(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(fixture *gittest.Fixture) string
		choice     string
		wantText   []string
		wantUpdate bool
		wantBranch string
	}{
		{name: "T155 c creates from unchanged local main without network", setup: func(fixture *gittest.Fixture) string {
			behindOrigin(fixture)
			fixture.Git("switch", "-q", "feature")
			fixture.Dirty("differs.txt")
			fixture.Git("remote", "set-url", "origin", filepath.Join(fixture.Dir, "unreachable.git"))
			return ""
		}, choice: "c", wantText: []string{fmt.Sprintf(mainBehindText, 1), updateAndCreateHints}, wantBranch: "feature"},
		{name: "T156 p from dirty feature updates local main only", setup: func(fixture *gittest.Fixture) string {
			remoteOID := behindOrigin(fixture)
			fixture.Git("switch", "-q", "feature")
			fixture.Write("shared.txt", "staged\n")
			fixture.Git("add", "shared.txt")
			fixture.Dirty("differs.txt")
			return remoteOID
		}, choice: "p", wantText: []string{updateAndCreateHints}, wantUpdate: true, wantBranch: "feature"},
		{name: "T157 p on clean main pulls fast-forward only", setup: behindOrigin, choice: "p", wantUpdate: true, wantBranch: "main"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var remoteOID string
			gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) { remoteOID = testCase.setup(fixture) })
			oldMain := branchOID(fixture, "refs/heads/main")
			before := checkoutState(fixture)

			createFromMain(t, gitflowApp, "topic")
			view := unwrappedOverlayText(t, gitflowApp)
			for _, want := range testCase.wantText {
				if !strings.Contains(view, want) {
					t.Errorf("choice prompt missing %q:\n%s", want, view)
				}
			}
			if branchOID(fixture, "refs/heads/main") != oldMain || strings.Contains(fixture.Git("branch", "--list", "topic"), "topic") {
				t.Fatal("the choice prompt changed Git before a choice")
			}
			pressAndDrain(gitflowApp, testCase.choice)

			wantStart := oldMain
			if testCase.wantUpdate {
				wantStart = remoteOID
			}
			if got := branchOID(fixture, "refs/heads/topic"); got != wantStart || branchOID(fixture, "refs/heads/main") != wantStart {
				t.Fatalf("topic %s main %s, want both %s (error %q)", got, branchOID(fixture, "refs/heads/main"), wantStart, gitflowApp.errorText)
			}
			if checkedOut(fixture) != testCase.wantBranch || gitflowApp.notice != "Created topic · Still on "+testCase.wantBranch {
				t.Fatalf("checked out %q notice %q", checkedOut(fixture), gitflowApp.notice)
			}
			if testCase.wantBranch == "feature" && checkoutState(fixture) != before {
				t.Fatal("feature checkout, index or worktree changed")
			}
			if stashes := fixture.Git("stash", "list"); stashes != "" || upstreamOf(fixture, "topic") != "" {
				t.Fatalf("stash %q upstream %q, want neither", stashes, upstreamOf(fixture, "topic"))
			}
			if _, prompted := gitflowApp.overlay.(*switchPrompt); !prompted {
				t.Fatalf("overlay after create = %T, want the Switch? prompt", gitflowApp.overlay)
			}
		})
	}
}

func TestCreateFromMainWithoutPrompt(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(fixture *gittest.Fixture)
	}{
		{name: "synchronized with cached origin/main"},
		{name: "ahead of cached origin/main", setup: func(fixture *gittest.Fixture) { fixture.Commit("local", map[string]string{"local.txt": "local\n"}) }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newBranchApp(t, testCase.setup)
			createFromMain(t, gitflowApp, "topic")
			if got := branchOID(fixture, "refs/heads/topic"); got != branchOID(fixture, "refs/heads/main") {
				t.Fatalf("topic at %s, want local main (error %q)", got, gitflowApp.errorText)
			}
			if _, prompted := gitflowApp.overlay.(*switchPrompt); !prompted {
				t.Fatalf("overlay = %T, want the Switch? prompt with no behind warning", gitflowApp.overlay)
			}
		})
	}
}

func TestCreateFromMainShowsComparison(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) { behindOrigin(fixture) })
	pressAndDrain(gitflowApp, "a", "m")
	view := flattenedView(gitflowApp)
	for _, want := range []string{"refs/heads/main", branchOID(fixture, "refs/heads/main"), fmt.Sprintf(mainComparisonFormat, 0, 1)} {
		if !strings.Contains(view, want) {
			t.Errorf("S01 a m name input missing %q:\n%s", want, view)
		}
	}
	pressAndDrain(gitflowApp, "esc")
	if overlayID(gitflowApp) != "" {
		t.Fatalf("Esc left overlay %s", overlayID(gitflowApp))
	}
}

func TestCreateFromMainDirectOnly(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(fixture *gittest.Fixture)
		pressP   bool
		wantText []string
	}{
		{name: "T158 diverged main", setup: func(fixture *gittest.Fixture) {
			behindOrigin(fixture)
			fixture.Commit("local", map[string]string{"local.txt": "local\n"})
		}, wantText: []string{mainDivergedText}},
		{name: "T158 no cached origin/main", setup: func(fixture *gittest.Fixture) {
			fixture.Git("remote", "remove", "origin")
		}, wantText: []string{mainRemoteMissing}},
		{name: "T157 dirty main blocks the update", setup: func(fixture *gittest.Fixture) {
			behindOrigin(fixture)
			fixture.Dirty("shared.txt")
		}, pressP: true, wantText: []string{syncBlockerText[git.SyncDirty], "shared.txt"}},
		{name: "T158 network failure", setup: func(fixture *gittest.Fixture) {
			behindOrigin(fixture)
			fixture.Git("remote", "set-url", "origin", filepath.Join(fixture.Dir, "unreachable.git"))
		}, pressP: true, wantText: []string{mainUpdateFailed}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newBranchApp(t, testCase.setup)
			before := fixture.Snapshot()
			createFromMain(t, gitflowApp, "topic")
			if testCase.pressP {
				pressAndDrain(gitflowApp, "p")
			}
			view := unwrappedOverlayText(t, gitflowApp)
			for _, want := range append(testCase.wantText, createDirectlyHints) {
				if !strings.Contains(view, want) {
					t.Errorf("prompt missing %q:\n%s", want, view)
				}
			}
			if strings.Contains(view, updateAndCreateHints) {
				t.Errorf("update offered:\n%s", view)
			}
			pressAndDrain(gitflowApp, "p")
			if fixture.Snapshot() != before {
				t.Fatal("refused update changed the repository")
			}
			pressAndDrain(gitflowApp, "c")
			if got := branchOID(fixture, "refs/heads/topic"); got != branchOID(fixture, "refs/heads/main") {
				t.Fatalf("topic at %s, want unchanged local main (error %q)", got, gitflowApp.errorText)
			}
		})
	}
}

func TestCreateFromMainFailures(t *testing.T) {
	t.Run("T158 missing local main", func(t *testing.T) {
		gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) {
			fixture.Git("switch", "-q", "feature")
			fixture.Git("branch", "-q", "-D", "main")
		})
		before := fixture.Snapshot()
		pressAndDrain(gitflowApp, "a", "m")
		if overlayID(gitflowApp) != "" || !strings.Contains(gitflowApp.errorText, git.ErrNoLocalMain.Error()) {
			t.Fatalf("overlay %s error %q, want %q", overlayID(gitflowApp), gitflowApp.errorText, git.ErrNoLocalMain)
		}
		if fixture.Snapshot() != before {
			t.Fatal("missing main changed the repository")
		}
	})
	t.Run("T158 update succeeds then create fails", func(t *testing.T) {
		var remoteOID string
		gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) {
			remoteOID = behindOrigin(fixture)
			fixture.Git("switch", "-q", "feature")
		})
		createFromMain(t, gitflowApp, "topic")
		featureOID := branchOID(fixture, "refs/heads/feature")
		fixture.Git("branch", "topic", "feature")
		pressAndDrain(gitflowApp, "p")
		if branchOID(fixture, "refs/heads/main") != remoteOID || branchOID(fixture, "refs/heads/topic") != featureOID {
			t.Fatalf("main %s topic %s, want main updated to %s and topic untouched", branchOID(fixture, "refs/heads/main"), branchOID(fixture, "refs/heads/topic"), remoteOID)
		}
		if want := fmt.Sprintf("local main updated to %s; branch topic was not created", shortOID(remoteOID)); !strings.Contains(gitflowApp.errorText, want) {
			t.Fatalf("error %q, want %q", gitflowApp.errorText, want)
		}
		if overlayID(gitflowApp) != "" || checkedOut(fixture) != "feature" {
			t.Fatalf("overlay %s checked out %q after failed create", overlayID(gitflowApp), checkedOut(fixture))
		}
	})
	t.Run("Esc cancels the choice", func(t *testing.T) {
		gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) { behindOrigin(fixture) })
		before := fixture.Snapshot()
		createFromMain(t, gitflowApp, "topic")
		pressAndDrain(gitflowApp, "esc")
		if overlayID(gitflowApp) != "" || fixture.Snapshot() != before {
			t.Fatal("Esc did not cancel without Git changes")
		}
	})
}

func TestPostCreateSwitchPrompt(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, nil)
	featureOID := branchOID(fixture, "refs/heads/feature")

	pressAndDrain(gitflowApp, "g", "n")
	typeText(gitflowApp, "stay")
	pressAndDrain(gitflowApp, "enter")
	if view := flattenedView(gitflowApp); !strings.Contains(view, fmt.Sprintf(switchPromptTitle, "stay")) || !strings.Contains(view, switchPromptHints) {
		t.Fatalf("Switch? prompt missing:\n%s", view)
	}
	pressAndDrain(gitflowApp, "x", "esc")
	if overlayID(gitflowApp) != "" || checkedOut(fixture) != "main" || branchOID(fixture, "refs/heads/stay") != branchOID(fixture, "HEAD") {
		t.Fatalf("T151 g n Esc: overlay %s checked out %q", overlayID(gitflowApp), checkedOut(fixture))
	}

	branches := openBranches(t, gitflowApp, "g", "b")
	focusBranch(t, gitflowApp, branches, "refs/heads/feature")
	pressAndDrain(gitflowApp, "a", "n")
	typeText(gitflowApp, "from-feature")
	pressAndDrain(gitflowApp, "enter", "s")
	if checkedOut(fixture) != "from-feature" || branchOID(fixture, "HEAD") != featureOID || gitflowApp.notice != "Switched to from-feature" {
		t.Fatalf("T151 a n s: checked out %q at %s notice %q", checkedOut(fixture), branchOID(fixture, "HEAD"), gitflowApp.notice)
	}

	pressAndDrain(gitflowApp, "a", "m")
	typeText(gitflowApp, "from-main")
	pressAndDrain(gitflowApp, "enter", "enter")
	if checkedOut(fixture) != "from-main" || branchOID(fixture, "HEAD") != branchOID(fixture, "refs/heads/main") {
		t.Fatalf("T151 a m Enter: checked out %q", checkedOut(fixture))
	}
	for _, branch := range []string{"stay", "from-feature", "from-main"} {
		if upstream := upstreamOf(fixture, branch); upstream != "" {
			t.Errorf("%s has upstream %q", branch, upstream)
		}
	}
	if remoteRefs := fixture.Git("--git-dir", fixture.Remote, "for-each-ref", "--format=%(refname)"); remoteRefs != "refs/heads/main\n" {
		t.Fatalf("remote refs changed:\n%s", remoteRefs)
	}
}

func TestPostCreateSwitchBlocked(t *testing.T) {
	gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) { fixture.Dirty("differs.txt") })
	branches := openBranches(t, gitflowApp, "g", "b")
	focusBranch(t, gitflowApp, branches, "refs/heads/feature")
	pressAndDrain(gitflowApp, "a", "n")
	typeText(gitflowApp, "from-feature")
	pressAndDrain(gitflowApp, "enter")
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "s")
	view := flattenedView(gitflowApp)
	for _, want := range []string{switchBlockerText[git.BlockedDirty], "differs.txt", stashAndSwitchHints} {
		if !strings.Contains(view, want) {
			t.Errorf("blocked switch missing %q:\n%s", want, view)
		}
	}
	if fixture.Snapshot() != before || checkedOut(fixture) != "main" {
		t.Fatal("blocked switch changed Git state")
	}
}

func TestDeleteBranch(t *testing.T) {
	testCases := []struct {
		name       string
		focusRef   string
		wantReason string
		wantGone   bool
		wantError  bool
	}{
		{name: "T128 merged local branch", focusRef: "refs/heads/done", wantGone: true},
		{name: "T128 unmerged branch is protected by native -d", focusRef: "refs/heads/feature", wantError: true},
		{name: "T128 current branch", focusRef: "refs/heads/main", wantReason: reasonBranchCheckedOut},
		{name: "remote-tracking ref", focusRef: "refs/remotes/origin/main", wantReason: reasonRemoteTrackingRef},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) { fixture.Git("branch", "done") })
			branches := openBranches(t, gitflowApp, "g", "b")
			focusBranch(t, gitflowApp, branches, testCase.focusRef)
			before := fixture.Snapshot()
			name := strings.TrimPrefix(testCase.focusRef, localRefPrefix)

			runComposerAction(t, gitflowApp, actionDeleteBranch)
			if testCase.wantReason != "" {
				if overlayID(gitflowApp) != "" || gitflowApp.notice != "Delete branch: "+testCase.wantReason || fixture.Snapshot() != before {
					t.Fatalf("delete = overlay %s notice %q", overlayID(gitflowApp), gitflowApp.notice)
				}
				return
			}
			review := flattenedView(gitflowApp)
			for _, want := range []string{"git branch -d " + name, deleteUnmergedWarning} {
				if !strings.Contains(review, want) {
					t.Fatalf("S10 delete review missing %q:\n%s", want, review)
				}
			}
			pressAndDrain(gitflowApp, keyConfirm)
			exists := fixture.Git("for-each-ref", "--format=%(refname)", testCase.focusRef) != ""
			if exists == testCase.wantGone || (gitflowApp.errorText != "") != testCase.wantError || checkedOut(fixture) != "main" {
				t.Fatalf("ref exists %v error %q notice %q", exists, gitflowApp.errorText, gitflowApp.notice)
			}
			if testCase.wantGone && gitflowApp.notice != "Deleted "+name {
				t.Fatalf("notice %q", gitflowApp.notice)
			}
		})
	}
}

func newSyncApp(t *testing.T, setup func(fixture *gittest.Fixture) string) (*App, *gittest.Fixture, string) {
	t.Helper()
	var wantOID string
	gitflowApp, fixture := newBranchApp(t, func(fixture *gittest.Fixture) {
		fixture.Git("branch", "-q", "--set-upstream-to=origin/main", "main")
		if setup != nil {
			wantOID = setup(fixture)
		}
	})
	return gitflowApp, fixture, wantOID
}

func reviewText(t *testing.T, gitflowApp *App) string {
	t.Helper()
	review, isConfirm := gitflowApp.overlay.(*confirmOverlay)
	if force, isForce := gitflowApp.overlay.(*forceOverlay); isForce {
		review, isConfirm = force.confirmOverlay, true
	}
	if !isConfirm {
		t.Fatalf("overlay = %T, want a review", gitflowApp.overlay)
	}
	return strings.Join(slices.Concat([]string{review.title, strings.Join(review.command, " ")}, review.consequences), "\n")
}

func upstreamOf(fixture *gittest.Fixture, branch string) string {
	return strings.TrimSpace(fixture.Git("for-each-ref", "--format=%(upstream:short)", "refs/heads/"+branch))
}

func TestSyncRunsWithoutReview(t *testing.T) {
	pushedRemotely := func(fixture *gittest.Fixture) string {
		return fixture.PushFromOtherClone("main", "remote", map[string]string{"remote.txt": "remote\n"})
	}
	testCases := []struct {
		name      string
		setup     func(fixture *gittest.Fixture) string
		keys      []string
		title     string
		observed  func(fixture *gittest.Fixture) string
		wantError bool
	}{
		{name: "T86 g e fetches remote-tracking refs only", setup: pushedRemotely, keys: []string{"g", "e"}, title: "Fetch",
			observed: func(fixture *gittest.Fixture) string { return branchOID(fixture, "refs/remotes/origin/main") }},
		{name: "T85 g f fast-forwards the checked-out branch", setup: pushedRemotely, keys: []string{"g", "f"}, title: "Pull (fast-forward only)",
			observed: func(fixture *gittest.Fixture) string { return branchOID(fixture, "HEAD") }},
		{name: "T85 g r replays local commits on the upstream", keys: []string{"g", "r"}, title: "Pull (rebase)",
			setup: func(fixture *gittest.Fixture) string {
				remoteOID := pushedRemotely(fixture)
				fixture.Commit("local", map[string]string{"local.txt": "local\n"})
				return remoteOID
			},
			observed: func(fixture *gittest.Fixture) string { return branchOID(fixture, "HEAD~1") }},
		{name: "T84 g p runs the configured pull", setup: pushedRemotely, keys: []string{"g", "p"}, title: "Pull",
			observed: func(fixture *gittest.Fixture) string { return branchOID(fixture, "HEAD") }},
		{name: "T86 g P pushes to the existing upstream", keys: []string{"g", "P"}, title: "Push",
			setup: func(fixture *gittest.Fixture) string {
				return fixture.Commit("local", map[string]string{"local.txt": "local\n"})
			},
			observed: func(fixture *gittest.Fixture) string { return fixture.RemoteRefOID(fixture.Remote, "refs/heads/main") }},
		{name: "T142 g P keeps the native non-fast-forward rejection", keys: []string{"g", "P"}, title: "Push", wantError: true,
			setup: func(fixture *gittest.Fixture) string {
				remoteOID := pushedRemotely(fixture)
				fixture.Commit("local", map[string]string{"local.txt": "local\n"})
				return remoteOID
			},
			observed: func(fixture *gittest.Fixture) string { return fixture.RemoteRefOID(fixture.Remote, "refs/heads/main") }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, wantOID := newSyncApp(t, testCase.setup)
			headBefore := branchOID(fixture, "HEAD")

			pressAndDrain(gitflowApp, testCase.keys...)
			if gitflowApp.overlay != nil {
				t.Fatalf("%v opened %T; routine sync never asks for review", testCase.keys, gitflowApp.overlay)
			}
			if got := testCase.observed(fixture); got != wantOID {
				t.Fatalf("observed %s, want %s", got, wantOID)
			}
			if testCase.wantError {
				if !strings.Contains(gitflowApp.errorText, "rejected") || branchOID(fixture, "HEAD") != headBefore {
					t.Fatalf("error %q notice %q", gitflowApp.errorText, gitflowApp.notice)
				}
				return
			}
			if gitflowApp.errorText != "" || !strings.HasPrefix(gitflowApp.notice, testCase.title+": ") {
				t.Fatalf("error %q notice %q, want an inline %s summary", gitflowApp.errorText, gitflowApp.notice, testCase.title)
			}
		})
	}
}

func TestSyncBlockers(t *testing.T) {
	diverged := func(fixture *gittest.Fixture) string {
		fixture.PushFromOtherClone("main", "remote", map[string]string{"remote.txt": "remote\n"})
		fixture.Git("fetch", "-q", "origin")
		fixture.Commit("local", map[string]string{"local.txt": "local\n"})
		return ""
	}
	testCases := []struct {
		name     string
		setup    func(fixture *gittest.Fixture) string
		keys     []string
		blocker  git.SyncBlocker
		wantText []string
	}{
		{name: "g p lists tracked dirty paths", keys: []string{"g", "p"}, blocker: git.SyncDirty, wantText: []string{"git pull", "shared.txt"},
			setup: func(fixture *gittest.Fixture) string { fixture.Write("shared.txt", "dirty\n"); return "" }},
		{name: "g f lists tracked dirty paths", keys: []string{"g", "f"}, blocker: git.SyncDirty, wantText: []string{"git pull --ff-only", "shared.txt"},
			setup: func(fixture *gittest.Fixture) string { fixture.Write("shared.txt", "dirty\n"); return "" }},
		{name: "T84 g p on a diverged branch without a configured strategy", keys: []string{"g", "p"}, blocker: git.SyncUnresolvedStrategy, setup: diverged},
		{name: "g e without any remote", keys: []string{"g", "e"}, blocker: git.SyncNoRemote, wantText: []string{"git fetch"},
			setup: func(fixture *gittest.Fixture) string { fixture.Git("remote", "remove", "origin"); return "" }},
		{name: "T141 g P first push without origin", keys: []string{"g", "P"}, blocker: git.SyncNoOrigin,
			setup: func(fixture *gittest.Fixture) string {
				fixture.Git("switch", "-q", "feature")
				fixture.Git("remote", "rename", "origin", "mirror")
				return ""
			}},
		{name: "g F with no cached upstream ref to lease against", keys: []string{"g", "F"}, blocker: git.SyncLeaseUnknown,
			setup: func(fixture *gittest.Fixture) string {
				fixture.Git("config", "branch.main.merge", "refs/heads/gone")
				return ""
			}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, _ := newSyncApp(t, testCase.setup)
			before := fixture.Snapshot()

			pressAndDrain(gitflowApp, testCase.keys...)
			if overlayID(gitflowApp) != ScreenConfirm {
				t.Fatalf("overlay %s error %q notice %q, want the blocker", overlayID(gitflowApp), gitflowApp.errorText, gitflowApp.notice)
			}
			review := reviewText(t, gitflowApp)
			for _, want := range append([]string{syncBlockerText[testCase.blocker]}, testCase.wantText...) {
				if !strings.Contains(review, want) {
					t.Fatalf("blocker missing %q:\n%s", want, review)
				}
			}
			pressAndDrain(gitflowApp, "esc")
			if gitflowApp.overlay != nil || fixture.Snapshot() != before || gitflowApp.errorText != "" {
				t.Fatalf("blocker changed state: overlay %T error %q", gitflowApp.overlay, gitflowApp.errorText)
			}
		})
	}
}

func TestSyncUnavailable(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(fixture *gittest.Fixture) string
		keys       []string
		wantNotice string
	}{
		{name: "g p without upstream", keys: []string{"g", "p"}, wantNotice: "Pull: " + reasonNoUpstream,
			setup: func(fixture *gittest.Fixture) string { fixture.Git("switch", "-q", "feature"); return "" }},
		{name: "g F without upstream", keys: []string{"g", "F"}, wantNotice: "Force push with lease: " + reasonNoUpstream,
			setup: func(fixture *gittest.Fixture) string { fixture.Git("switch", "-q", "feature"); return "" }},
		{name: "g P on detached HEAD", keys: []string{"g", "P"}, wantNotice: "Push: " + reasonDetachedHead,
			setup: func(fixture *gittest.Fixture) string { fixture.Git("switch", "-q", "--detach"); return "" }},
		{name: "T139 g u awaits a contract decision", keys: []string{"g", "u"}, wantNotice: "Set upstream: " + reasonUpstreamContractGap},
		{name: "T139 a u awaits a contract decision", keys: []string{"g", "b", "a", "u"}, wantNotice: "Set upstream: " + reasonUpstreamContractGap},
		{name: "g r on detached HEAD", keys: []string{"g", "r"}, wantNotice: "Pull (rebase): " + reasonDetachedHead,
			setup: func(fixture *gittest.Fixture) string { fixture.Git("switch", "-q", "--detach"); return "" }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, _ := newSyncApp(t, testCase.setup)
			before := fixture.Snapshot()

			pressAndDrain(gitflowApp, testCase.keys...)
			if gitflowApp.overlay != nil || gitflowApp.notice != testCase.wantNotice || fixture.Snapshot() != before {
				t.Fatalf("overlay %T notice %q, want %q", gitflowApp.overlay, gitflowApp.notice, testCase.wantNotice)
			}
		})
	}
}

func TestFirstPush(t *testing.T) {
	testCases := []struct {
		name            string
		setup           func(fixture *gittest.Fixture) string
		wantDestination string
	}{
		{name: "T140 destination absent from cached refs", wantDestination: "creates it",
			setup: func(fixture *gittest.Fixture) string { fixture.Git("switch", "-q", "feature"); return "" }},
		{name: "T140 destination already exists", wantDestination: "exists in cached refs at",
			setup: func(fixture *gittest.Fixture) string {
				fixture.Git("push", "-q", "origin", "main:refs/heads/feature")
				fixture.Git("fetch", "-q", "origin")
				fixture.Git("switch", "-q", "feature")
				return ""
			}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture, _ := newSyncApp(t, testCase.setup)
			remoteBefore := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature")
			before := fixture.Snapshot()

			pressAndDrain(gitflowApp, "g", "P")
			review := reviewText(t, gitflowApp)
			for _, want := range []string{"git push -u origin feature", "origin/feature", testCase.wantDestination, firstPushTracking} {
				if !strings.Contains(review, want) {
					t.Fatalf("first-push review missing %q:\n%s", want, review)
				}
			}
			pressAndDrain(gitflowApp, "esc")
			if gitflowApp.overlay != nil || fixture.Snapshot() != before || upstreamOf(fixture, "feature") != "" ||
				fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature") != remoteBefore {
				t.Fatalf("Esc changed state")
			}

			pressAndDrain(gitflowApp, "g", "P", keyConfirmGuarded)
			if gitflowApp.errorText != "" || !strings.HasPrefix(gitflowApp.notice, "Push: ") {
				t.Fatalf("error %q notice %q", gitflowApp.errorText, gitflowApp.notice)
			}
			if got := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature"); got != branchOID(fixture, "feature") || upstreamOf(fixture, "feature") != "origin/feature" {
				t.Fatalf("remote feature %s upstream %q", got, upstreamOf(fixture, "feature"))
			}
		})
	}
}

func TestForcePushWithLease(t *testing.T) {
	var remoteOID string
	gitflowApp, fixture, _ := newSyncApp(t, func(fixture *gittest.Fixture) string {
		remoteOID = fixture.PushFromOtherClone("main", "remote", map[string]string{"remote.txt": "remote\n"})
		fixture.Git("fetch", "-q", "origin")
		fixture.Commit("local", map[string]string{"local.txt": "local\n"})
		return ""
	})
	before := fixture.Snapshot()

	pressAndDrain(gitflowApp, "g", "F")
	if _, isForce := gitflowApp.overlay.(*forceOverlay); !isForce || overlayID(gitflowApp) != ScreenSync {
		t.Fatalf("T87 g F opened %T (%s), want the S15 lease review", gitflowApp.overlay, overlayID(gitflowApp))
	}
	review := reviewText(t, gitflowApp)
	for _, want := range []string{"--force-with-lease=refs/heads/main:" + remoteOID, "origin", remoteOID, forceRewrite} {
		if !strings.Contains(review, want) {
			t.Fatalf("lease review missing %q:\n%s", want, review)
		}
	}
	pressAndDrain(gitflowApp, "enter")
	if gitflowApp.overlay != nil || fixture.Snapshot() != before || fixture.RemoteRefOID(fixture.Remote, "refs/heads/main") != remoteOID {
		t.Fatalf("Enter changed state")
	}

	pressAndDrain(gitflowApp, "g", "F")
	othersOID := fixture.PushFromOtherClone("main", "someone else", nil)
	pressAndDrain(gitflowApp, keyConfirmGuarded)
	if gitflowApp.errorText == "" || fixture.RemoteRefOID(fixture.Remote, "refs/heads/main") != othersOID {
		t.Fatalf("stale lease: error %q remote %s, want rejection keeping %s", gitflowApp.errorText, fixture.RemoteRefOID(fixture.Remote, "refs/heads/main"), othersOID)
	}

	pressAndDrain(gitflowApp, "g", "e")
	pressAndDrain(gitflowApp, "g", "F", keyConfirmGuarded)
	if gitflowApp.errorText != "" || fixture.RemoteRefOID(fixture.Remote, "refs/heads/main") != branchOID(fixture, "main") {
		t.Fatalf("T87 lease push: error %q notice %q", gitflowApp.errorText, gitflowApp.notice)
	}
}
