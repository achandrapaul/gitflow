package app

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func rebaseHistory(fixture *gittest.Fixture) {
	fixture.Commit("base", map[string]string{"base.txt": "base\n"})
	fixture.Commit("first", map[string]string{"first.txt": "first\n"})
	fixture.Commit("second", map[string]string{"second.txt": "second\n"})
	fixture.Commit("third", map[string]string{"third.txt": "third\n"})
}

func openRebasePlanner(t *testing.T, gitflowApp *App, baseSubject string) *rebaseScreen {
	t.Helper()
	history := openHistory(t, gitflowApp)
	focusSubject(t, gitflowApp, history, baseSubject)
	runRegistered(gitflowApp, actionRebaseStart)
	planner, open := gitflowApp.top().(*rebaseScreen)
	if !open {
		t.Fatalf("Rebase onto base opened %s, want the Interactive Rebase Planner", gitflowApp.top().ID())
	}
	return planner
}

func todoSubjects(planner *rebaseScreen) []string {
	var todo []string
	for _, step := range planner.plan.Steps {
		todo = append(todo, step.Action.String()+" "+step.Subject)
	}
	return todo
}

func runRebaseHandoff(t *testing.T, gitflowApp *App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("Execute dispatched nothing: notice %q error %q", gitflowApp.notice, gitflowApp.errorText)
	}
	ready, isReady := cmd().(handoffReadyMsg)
	if !isReady || ready.err != nil {
		t.Fatalf("Execute produced %+v, want a terminal handoff to git rebase -i (error %q)", ready, gitflowApp.errorText)
	}
	exitErr := ready.command.Run()
	drain(gitflowApp, func() tea.Msg { return ready.finish(exitErr) })
}

func executeRebase(t *testing.T, gitflowApp *App) {
	t.Helper()
	action, _ := gitflowApp.registry.ByID(actionRebaseExecute)
	runRebaseHandoff(t, gitflowApp, gitflowApp.runAction(action))
}

func TestRebasePlannerT18(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, rebaseHistory)
	base := strings.TrimSpace(fixture.Git("rev-parse", "HEAD~3"))
	before := fixture.Snapshot()
	planner := openRebasePlanner(t, gitflowApp, "base")
	if got := todoSubjects(planner); !slices.Equal(got, []string{"pick first", "pick second", "pick third"}) {
		t.Fatalf("todo = %v, want base..HEAD oldest first", got)
	}
	view := flattenedView(gitflowApp)
	for _, want := range []string{"Interactive Rebase Planner", "main", "onto " + shortOID(base), "3 commits"} {
		if !strings.Contains(view, want) {
			t.Errorf("planner view missing %q:\n%s", want, view)
		}
	}

	pressAndDrain(gitflowApp, "f")
	pressAndDrain(gitflowApp, "v")
	if text := overlayText(t, gitflowApp); !strings.Contains(text, git.ErrRebaseOrphanCombine.Error()) {
		t.Fatalf("T18 first-row fixup preview does not explain the block:\n%s", text)
	}
	pressAndDrain(gitflowApp, "esc")
	if available, reason := availability(gitflowApp, actionRebaseExecute); available || !strings.Contains(reason, git.ErrRebaseOrphanCombine.Error()) {
		t.Fatalf("T18 Execute with a first-row fixup: %v %q", available, reason)
	}

	pressAndDrain(gitflowApp, "d", "J", "J", "k", "K", "j", "f")
	if got := todoSubjects(planner); !slices.Equal(got, []string{"pick third", "fixup second", "drop first"}) {
		t.Fatalf("todo after edits = %v", got)
	}
	if available, reason := availability(gitflowApp, actionRebaseExecute); available || reason != reasonPreviewPlan {
		t.Fatalf("Execute before previewing the edited plan: %v %q", available, reason)
	}
	pressAndDrain(gitflowApp, "v")
	text := overlayText(t, gitflowApp)
	for _, want := range []string{"git rebase -i " + base, "pick", "fixup", "drop", "third", "second", "first", rebasePreviewReady} {
		if !strings.Contains(text, want) {
			t.Errorf("preview missing %q:\n%s", want, text)
		}
	}
	pressAndDrain(gitflowApp, "esc")
	if fixture.Snapshot() != before {
		t.Fatal("planning and preview changed Git state")
	}

	executeRebase(t, gitflowApp)
	if subjects := fixture.Git("log", "--format=%s", base+"..HEAD"); subjects != "third\n" {
		t.Fatalf("T18 history = %q, want third only", subjects)
	}
	if files := fixture.Git("ls-tree", "--name-only", "HEAD"); files != "base.txt\nsecond.txt\nthird.txt\n" {
		t.Fatalf("T18 tree = %q", files)
	}
	if gitflowApp.top().ID() != ScreenHistory || gitflowApp.errorText != "" || !strings.Contains(gitflowApp.notice, "not pushed") {
		t.Fatalf("after rebase top %s error %q notice %q", gitflowApp.top().ID(), gitflowApp.errorText, gitflowApp.notice)
	}
	history, _ := gitflowApp.historyOnTop()
	if rows := history.visible(); len(rows) == 0 || rows[0].Subject != "third" || rows[0].OID != strings.TrimSpace(fixture.Git("rev-parse", "HEAD")) {
		t.Fatalf("History was not reloaded after the rebase: %+v", rows)
	}
}

func TestRebasePlannerBlockedRanges(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, rebaseHistory)
	before := fixture.Snapshot()
	openRebasePlanner(t, gitflowApp, "third")
	if view := flattenedView(gitflowApp); !strings.Contains(view, git.ErrRebaseEmptyRange.Error()) {
		t.Fatalf("empty range not explained:\n%s", view)
	}
	for _, id := range []ActionID{actionRebasePreview, actionRebasePick, actionRebaseExecute} {
		if available, reason := availability(gitflowApp, id); available || reason != reasonPlanNotLoaded {
			t.Errorf("%s on an empty range: %v %q", id, available, reason)
		}
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenHistory || fixture.Snapshot() != before {
		t.Fatalf("Esc left %s or changed Git", gitflowApp.top().ID())
	}
}

func TestRebasePlannerPublishedNeedsAcknowledgement(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, func(fixture *gittest.Fixture) {
		fixture.WithRemote()
		rebaseHistory(fixture)
		fixture.Git("push", "-q", "origin", "main")
		fixture.Git("fetch", "-q", "origin")
	})
	head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	openRebasePlanner(t, gitflowApp, "base")
	pressAndDrain(gitflowApp, "j", "d", "v")
	if text := overlayText(t, gitflowApp); !strings.Contains(text, "origin/main") {
		t.Fatalf("preview does not warn about published commits:\n%s", text)
	}
	pressAndDrain(gitflowApp, "esc")
	runRegistered(gitflowApp, actionRebaseExecute)
	text := overlayText(t, gitflowApp)
	for _, want := range []string{rebasePublishedTitle, "origin/main", rebaseNoPush} {
		if !strings.Contains(text, want) {
			t.Errorf("published acknowledgement missing %q:\n%s", want, text)
		}
	}
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.overlay != nil || fixture.Snapshot() != before {
		t.Fatal("Esc on the published acknowledgement did not cancel cleanly")
	}
	runRegistered(gitflowApp, actionRebaseExecute)
	runRebaseHandoff(t, gitflowApp, press(gitflowApp, "c"))
	if subjects := fixture.Git("log", "--format=%s", "HEAD~2..HEAD"); subjects != "third\nfirst\n" {
		t.Fatalf("history after acknowledged rewrite = %q", subjects)
	}
	if remote := fixture.RemoteRefOID(fixture.Remote, "refs/heads/main"); remote != head {
		t.Fatalf("remote main = %s, want unchanged %s: rewritten history must never be pushed", remote, head)
	}
}

func TestRebasePlannerT19PauseAndAbort(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, rebaseHistory)
	head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	openRebasePlanner(t, gitflowApp, "base")
	pressAndDrain(gitflowApp, "j", "e", "v", "esc")
	executeRebase(t, gitflowApp)
	if gitflowApp.top().ID() != ScreenPaused {
		t.Fatalf("T19 edit stop left %s on top, want the Paused Operation Dashboard", gitflowApp.top().ID())
	}
	view := flattenedView(gitflowApp)
	for _, want := range []string{"Paused Operation Dashboard · rebase", "Reason: Git stopped at the edit step without conflicts", "c Continue"} {
		if !strings.Contains(view, want) {
			t.Errorf("T19 dashboard missing %q:\n%s", want, view)
		}
	}

	paused := fixture.Snapshot()
	pressAndDrain(gitflowApp, "A")
	if text := overlayText(t, gitflowApp); !strings.Contains(text, "git rebase --abort") {
		t.Fatalf("T20 Abort review does not name the command:\n%s", text)
	}
	pressAndDrain(gitflowApp, "esc")
	if fixture.Snapshot() != paused || !gitflowApp.repository.operation.Active() {
		t.Fatal("T20 Esc on the Abort review changed the paused rebase")
	}

	pressAndDrain(gitflowApp, "A", "c")
	if got := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); got != head || gitflowApp.repository.operation.Active() {
		t.Fatalf("T21 abort left HEAD %s (want %s), operation %+v", got, head, gitflowApp.repository.operation)
	}
	if branch := strings.TrimSpace(fixture.Git("branch", "--show-current")); branch != "main" {
		t.Fatalf("T21 abort left branch %q", branch)
	}
}

func TestRebasePlannerT19ConflictPause(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, func(fixture *gittest.Fixture) {
		fixture.Commit("base", gittest.ConflictBase)
		fixture.Commit("main", gittest.ConflictMain)
		fixture.Commit("feature", gittest.FeatureOne)
	})
	openRebasePlanner(t, gitflowApp, "base")
	pressAndDrain(gitflowApp, "J", "v", "esc")
	executeRebase(t, gitflowApp)
	if gitflowApp.top().ID() != ScreenPaused {
		t.Fatalf("T19 conflict left %s on top", gitflowApp.top().ID())
	}
	if view := flattenedView(gitflowApp); !strings.Contains(view, "Reason: conflicts in 1 files") || !strings.Contains(view, "conflict.txt") {
		t.Fatalf("T19 conflict dashboard:\n%s", view)
	}
	if unmerged := fixture.Git("diff", "--name-only", "--diff-filter=U"); unmerged != "conflict.txt\n" {
		t.Fatalf("unmerged = %q", unmerged)
	}
}

func TestRebasePlannerT25StaleHead(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, rebaseHistory)
	openRebasePlanner(t, gitflowApp, "base")
	pressAndDrain(gitflowApp, "d", "v", "esc")
	fixture.Commit("external", map[string]string{"external.txt": "external\n"})
	before := fixture.Snapshot()

	action, _ := gitflowApp.registry.ByID(actionRebaseExecute)
	drain(gitflowApp, action.Run(gitflowApp))
	if !strings.Contains(gitflowApp.errorText, git.ErrStaleRef.Error()) || fixture.Snapshot() != before {
		t.Fatalf("T25 Execute on a moved HEAD: error %q, Git changed %v", gitflowApp.errorText, fixture.Snapshot() != before)
	}
	if available, reason := availability(gitflowApp, actionRebaseExecute); available || reason != reasonPlanStale {
		t.Fatalf("T25 Execute after refresh: %v %q, want the stale plan reason", available, reason)
	}
}

func TestRebasePlannerTrackedChangesBlockExecute(t *testing.T) {
	gitflowApp, fixture := newHistoryApp(t, rebaseHistory)
	openRebasePlanner(t, gitflowApp, "base")
	pressAndDrain(gitflowApp, "d", "v", "esc")
	fixture.Dirty("first.txt")
	drain(gitflowApp, gitflowApp.refresh())
	if available, reason := availability(gitflowApp, actionRebaseExecute); available || !strings.Contains(reason, "first.txt") {
		t.Fatalf("Execute with tracked changes: %v %q", available, reason)
	}
}

func TestPullRebasePausesIntoDashboard(t *testing.T) {
	gitflowApp, fixture, _ := newSyncApp(t, func(fixture *gittest.Fixture) string {
		fixture.PushFromOtherClone("main", "remote", map[string]string{"shared.txt": "remote\n"})
		fixture.Git("fetch", "-q", "origin")
		return fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
	})
	pressAndDrain(gitflowApp, "g", "r")
	if gitflowApp.overlay != nil {
		t.Fatalf("g r opened %T; routine sync never asks for review", gitflowApp.overlay)
	}
	operation, err := gitflowApp.deps.Operations.Active(gitflowApp.ctx)
	if err != nil || operation.Kind != git.OperationRebase {
		t.Fatalf("T85 g r conflict: operation %+v, %v", operation, err)
	}
	if gitflowApp.top().ID() != ScreenPaused || gitflowApp.errorText == "" {
		t.Fatalf("T85 g r pause left %s on top, error %q", gitflowApp.top().ID(), gitflowApp.errorText)
	}
	if unmerged := fixture.Git("diff", "--name-only", "--diff-filter=U"); unmerged != "shared.txt\n" {
		t.Fatalf("unmerged = %q", unmerged)
	}
}
