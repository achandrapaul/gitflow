package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

type syncTerminal struct{}

func (syncTerminal) Run(command *exec.Cmd, stderr *bytes.Buffer, done func(error) tea.Msg) tea.Cmd {
	command.Stderr = stderr
	return func() tea.Msg { return done(command.Run()) }
}

type harness struct {
	t       *testing.T
	fixture *gittest.Fixture
	app     *App
	quit    bool
}

func newHarness(t *testing.T, fixture *gittest.Fixture) *harness {
	t.Helper()
	repository := git.Repository{WorkTree: fixture.Dir, GitDir: filepath.Join(fixture.Dir, ".git")}
	model, err := New(context.Background(), repository, fixture.Runner, syncTerminal{})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	h := &harness{t: t, fixture: fixture, app: model}
	h.run(model.Init())
	return h
}

func (h *harness) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	switch msg := msg.(type) {
	case nil:
	case tea.BatchMsg:
		for _, inner := range msg {
			h.run(inner)
		}
	case tea.QuitMsg:
		h.quit = true
	default:
		_, next := h.app.Update(msg)
		h.run(next)
	}
}

var specialKeys = map[string]tea.KeyType{
	"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "shift+tab": tea.KeyShiftTab,
	"up": tea.KeyUp, "down": tea.KeyDown, "ctrl+d": tea.KeyCtrlD, "ctrl+u": tea.KeyCtrlU, " ": tea.KeySpace,
}

func (h *harness) press(keys ...string) {
	h.t.Helper()
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if keyType, ok := specialKeys[key]; ok {
			msg = tea.KeyMsg{Type: keyType}
			if key == " " {
				msg.Runes = []rune(" ")
			}
		}
		_, cmd := h.app.Update(msg)
		h.run(cmd)
	}
}

func (h *harness) typeText(text string) {
	h.t.Helper()
	for _, character := range text {
		h.press(string(character))
	}
}

func (h *harness) screen() string {
	return ansi.Strip(h.app.View())
}

func (h *harness) expectScreen(fragments ...string) {
	h.t.Helper()
	view := h.screen()
	for _, fragment := range fragments {
		if !strings.Contains(view, fragment) {
			h.t.Fatalf("screen missing %q:\n%s", fragment, view)
		}
	}
}

func (h *harness) menuSelect(title string) {
	h.t.Helper()
	h.press("a")
	menu, ok := h.app.overlay.(*menuOverlay)
	if !ok {
		h.t.Fatalf("menu not open, overlay = %T", h.app.overlay)
	}
	for index, action := range menu.actions {
		if action.Title == title {
			menu.cursor = index
			h.press("enter")
			return
		}
	}
	h.t.Fatalf("menu has no %q", title)
}

func (h *harness) palette(title string) {
	h.t.Helper()
	h.press(":")
	h.typeText(title)
	h.press("enter")
}

func (h *harness) confirm() {
	h.t.Helper()
	dialog, ok := h.app.overlay.(*confirmOverlay)
	if !ok {
		h.t.Fatalf("no confirmation open, overlay = %T", h.app.overlay)
	}
	if dialog.focus != 0 && dialog.title != "Quit GitFlow?" {
		h.t.Fatalf("confirmation %q must default to Cancel", dialog.title)
	}
	h.press("tab", "enter")
}

func (h *harness) head() string { return strings.TrimSpace(h.fixture.Git("rev-parse", "HEAD")) }

func conflictRepo(t *testing.T) (*gittest.Fixture, string) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"shared.txt": "one\nshared\nthree\n"})
	fixture.Git("switch", "-q", "-c", "topic")
	topic := fixture.Commit("topic edit", map[string]string{"shared.txt": "one\ntopic\nthree\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main edit", map[string]string{"shared.txt": "one\nmain\nthree\n"})
	return fixture, topic
}

func TestRegistryInvariants(t *testing.T) {
	registry, err := NewRegistry(actionDefinitions())
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, action := range registry.actions {
		if action.Title == "" || action.Run == nil {
			t.Errorf("%s lacks title or handler", action.ID)
		}
	}
	shortcut, _ := registry.Lookup(ScopeChanges, "s")
	if shortcut != registry.ByID("changes.stage") {
		t.Fatal("K04: shortcut s must resolve to the same action as the menu entry")
	}
	_, err = NewRegistry([]Action{{ID: "one", Scope: ScopeChanges, Keys: []string{"x"}}, {ID: "two", Scope: ScopeChanges, Keys: []string{"x"}}})
	if err == nil {
		t.Fatal("duplicate key in one scope must be rejected")
	}
	if _, err := NewRegistry([]Action{{ID: "one", Scope: ScopeChanges, Keys: []string{"c"}}, {ID: "two", Scope: ScopeChanges, Keys: []string{"c x"}}}); err == nil {
		t.Fatal("prefix collision must be rejected")
	}
	if _, err := NewRegistry([]Action{{ID: "one", Scope: ScopeChanges, Keys: []string{"j"}}}); err == nil {
		t.Fatal("navigation key collision must be rejected")
	}
	if _, err := NewRegistry([]Action{{ID: "one", Scope: ScopeConflict, Keys: []string{"n"}}}); err == nil {
		t.Fatal("scope navigation key collision must be rejected")
	}
}

func TestStartupCleanRepository(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Git("switch", "-q", "-c", "trunk")
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	h := newHarness(t, fixture)
	h.expectScreen("CHANGES", "DIFF INSPECTOR", "COMMIT COMPOSER", "Working tree clean", "● trunk @", "No upstream", "Base unavailable", "operation: none", "Nothing staged", "[c c Compose]")
	if strings.Contains(h.screen(), "↑0 ↓0") {
		t.Fatal("B02: no upstream must not show zero counts")
	}
	h.press("c", "c")
	h.expectScreen("review unavailable: nothing staged")
	h.press("esc", "esc")
	if _, ok := h.app.top().(*changesScreen); !ok {
		t.Fatalf("expected root after Esc, got %T", h.app.top())
	}
}

func TestStartupBranchIntelligence(t *testing.T) {
	fixture := gittest.New(t)
	fixture.BareRemote("origin")
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	fixture.Git("push", "-q", "-u", "origin", "main")
	fixture.Commit("local", map[string]string{"b.txt": "b\n"})
	h := newHarness(t, fixture)
	h.expectScreen("→ origin/main ↑1 ↓0", "[local refs · fetch time unknown]", "vs origin/main ↑1 ↓0")
	fixture.Git("switch", "-q", "--detach", "HEAD")
	h.run(h.app.refresh())
	h.expectScreen("DETACHED @ ")
}

func TestNavigationNeverMutatesGit(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"mixed.txt": tenLinesForApp, "dirty.txt": "a\n"})
	fixture.Write("mixed.txt", strings.Replace(tenLinesForApp, "l2\n", "L2\n", 1))
	fixture.Git("add", "--", "mixed.txt")
	fixture.Write("mixed.txt", strings.Replace(strings.Replace(tenLinesForApp, "l2\n", "L2\n", 1), "l9\n", "L9\n", 1))
	fixture.Write("dirty.txt", "b\n")
	fixture.Write("new file.txt", "n\n")
	before := fixture.Snapshot()
	h := newHarness(t, fixture)

	h.expectScreen("[1 All:3]", "[2 Staged:1]", "[3 Not staged:2]", "[4 Mixed:1]", "MIXED")
	h.press("j", "k", "enter")
	view, ok := h.app.top().(*diffScreen)
	if !ok {
		t.Fatalf("Enter must open S02, got %T", h.app.top())
	}
	h.expectScreen("STAGED · HEAD → INDEX", "UNSTAGED · INDEX → WORKTREE")
	h.press("tab", " ", "j", " ", "enter", "esc")
	if view == nil || h.app.top() != h.app.changes {
		t.Fatal("Esc from S02 must return to S01")
	}
	h.press("j", " ")
	h.press("3")
	h.expectScreen("1 selected (1 hidden by filter)")
	h.press("a")
	h.expectScreen("Targets: 1 selected file(s) (1 hidden by filter)")
	h.press("esc", "2", "4", "1", ":", "s", "u", "esc", "?", "esc", "tab", "tab", "tab", "tab", "o", "/", "s", "u", "esc", "esc", "q", "esc")
	if h.quit {
		t.Fatal("Esc on quit control must cancel")
	}
	if after := fixture.Snapshot(); after != before {
		t.Fatalf("K01-K03/T08: navigation mutated Git state\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if len(h.app.changes.selected) != 1 {
		t.Fatalf("C09: selection must survive filters, got %v", h.app.changes.selected)
	}
}

const tenLinesForApp = "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\n"

func TestStageShortcutAndMenuShareAction(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n", "b.txt": "b\n"})
	fixture.Write("a.txt", "A\n")
	fixture.Write("b.txt", "B\n")
	h := newHarness(t, fixture)
	h.press(" ", "s")
	if !strings.HasPrefix(fixture.Git("status", "--porcelain", "--", "a.txt"), "M ") {
		t.Fatalf("s must stage selected a.txt: %q", fixture.Git("status", "--porcelain"))
	}
	if len(h.app.changes.selected) != 0 {
		t.Fatal("selection of staged target should be cleared visibly")
	}
	h.press("1")
	for h.app.changes.focusedPath != "b.txt" {
		h.press("j")
	}
	h.menuSelect("Stage")
	if !strings.HasPrefix(fixture.Git("status", "--porcelain", "--", "b.txt"), "M ") {
		t.Fatal("menu Stage must stage focused b.txt")
	}
	h.menuSelect("Unstage")
	if !strings.HasPrefix(fixture.Git("status", "--porcelain", "--", "b.txt"), " M") {
		t.Fatal("menu Unstage must unstage b.txt")
	}
}

func TestStageHunkFromDiffInspector(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"f.txt": tenLinesForApp})
	changed := strings.Replace(strings.Replace(tenLinesForApp, "l2\n", "L2\n", 1), "l18\n", "L18\n", 1)
	fixture.Write("f.txt", changed)
	h := newHarness(t, fixture)
	h.press("enter", " ", "s")
	if !strings.Contains(fixture.Git("diff", "--cached"), "+L2") || strings.Contains(fixture.Git("diff", "--cached"), "L18") {
		t.Fatalf("J03: only the first hunk should be staged:\n%s", fixture.Git("diff", "--cached"))
	}
	if fixture.ReadFile("f.txt") != changed {
		t.Fatal("worktree must be preserved")
	}
	h.expectScreen("+L2", "+L18")
	h.press("esc")
	h.expectScreen("[4 Mixed:1]")
}

func TestComposerTypingAndCommit(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"f.txt": tenLinesForApp})
	staged := strings.Replace(tenLinesForApp, "l2\n", "L2\n", 1)
	fixture.Write("f.txt", staged)
	fixture.Git("add", "--", "f.txt")
	worktree := strings.Replace(staged, "l18\n", "L18\n", 1)
	fixture.Write("f.txt", worktree)
	h := newHarness(t, fixture)
	before := fixture.Snapshot()
	h.press("c", "c")
	h.typeText("s u a 1 c : q d A ?")
	if fixture.Snapshot() != before {
		t.Fatal("T29/K06: typing in the composer mutated Git")
	}
	h.press("enter")
	h.typeText("body j k")
	h.press("esc")
	if fixture.Snapshot() != before {
		t.Fatal("J10: Esc in a text field must not mutate Git")
	}
	h.press("enter")
	h.expectScreen("Review commit", "s u a 1 c : q d A ?", "body j k", "index only")
	h.confirm()
	if subject := strings.TrimSpace(fixture.Git("log", "-1", "--format=%s")); subject != "s u a 1 c : q d A ?" {
		t.Fatalf("subject = %q", subject)
	}
	if fixture.Git("show", "HEAD:f.txt") != staged || fixture.ReadFile("f.txt") != worktree {
		t.Fatal("C10/T09: commit must contain index only")
	}
	if h.app.top() != h.app.changes {
		t.Fatalf("successful commit should return to S01, got %T", h.app.top())
	}
	if _, kept := h.app.drafts[composeCommit]; kept {
		t.Fatal("draft should be cleared after commit")
	}
}

func TestCommitHookFailureKeepsDraft(t *testing.T) {
	fixture := gittest.New(t)
	head := fixture.Commit("base", map[string]string{"f.txt": "a\n"})
	if err := os.WriteFile(filepath.Join(fixture.Dir, ".git", "hooks", "commit-msg"), []byte("#!/bin/sh\necho 'message rejected by hook' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("hook: %v", err)
	}
	fixture.Write("f.txt", "b\n")
	fixture.Git("add", "--", "f.txt")
	h := newHarness(t, fixture)
	h.press("c", "c")
	h.typeText("rejected subject")
	h.press("esc", "enter")
	h.confirm()
	if h.head() != head {
		t.Fatal("T10: HEAD must not move")
	}
	if _, ok := h.app.top().(*composerScreen); !ok {
		t.Fatalf("C11: failed commit must stay in composer, got %T", h.app.top())
	}
	h.expectScreen("message rejected by hook", "rejected subject")
	if h.app.drafts[composeCommit].subject != "rejected subject" {
		t.Fatal("draft must be retained")
	}
	h.press("esc")
	dialog, ok := h.app.overlay.(*confirmOverlay)
	if !ok || dialog.title != "Leave composer?" {
		t.Fatalf("dirty composer Esc must ask Keep/Discard/Cancel, got %T", h.app.overlay)
	}
	h.press("tab", "enter")
	if h.app.top() != h.app.changes || h.app.drafts[composeCommit].subject != "rejected subject" {
		t.Fatal("Keep draft must return to S01 with the draft")
	}
}

func TestDiscardRequiresConfirmation(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"f.txt": "a\n"})
	fixture.Write("f.txt", "changed\n")
	h := newHarness(t, fixture)
	h.press("d")
	h.expectScreen("Discard worktree changes in 1 file(s)?", "git restore --worktree -- f.txt")
	h.press("esc")
	if fixture.ReadFile("f.txt") != "changed\n" {
		t.Fatal("Esc on discard confirmation must not discard")
	}
	h.press("d")
	h.confirm()
	if fixture.ReadFile("f.txt") != "a\n" {
		t.Fatal("confirmed discard should restore the file")
	}
}

func TestPausedCherryPickEscAbortAndQuit(t *testing.T) {
	fixture, topic := conflictRepo(t)
	fixture.GitFails("cherry-pick", topic)
	before := fixture.Snapshot()
	h := newHarness(t, fixture)
	h.expectScreen("operation: CHERRY-PICK PAUSED (o)", "CONFLICT")
	h.press("o")
	h.expectScreen("OPERATIONS · CHERRY-PICK PAUSED", "1 conflict(s) need resolution", "shared.txt", "UNRESOLVED", "[disabled: 1 conflict(s) need resolution]")
	h.press("c")
	h.press("esc")
	h.expectScreen("CHERRY-PICK PAUSED (o)")
	h.press("o", "A")
	h.expectScreen("Abort cherry-pick?", "git cherry-pick --abort")
	h.press("esc", "esc")
	if fixture.Snapshot() != before {
		t.Fatal("T16/T20/O05: Esc must leave the paused operation intact")
	}
	h.press("q", "enter")
	if !h.quit || fixture.Snapshot() != before {
		t.Fatal("T28/O09: quitting must leave Git untouched")
	}
	reopened := newHarness(t, fixture)
	reopened.press("o", "A")
	reopened.confirm()
	if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", "CHERRY_PICK_HEAD")); err == nil {
		t.Fatal("confirmed abort should end the cherry-pick")
	}
	if reopened.app.operation.Active() || reopened.app.top() != reopened.app.changes {
		t.Fatalf("after abort expected root with no operation, got %T active=%v", reopened.app.top(), reopened.app.operation.Active())
	}
}

func TestMergeConflictResolveAndContinue(t *testing.T) {
	fixture, _ := conflictRepo(t)
	fixture.GitFails("merge", "topic")
	h := newHarness(t, fixture)
	h.press("o", "s")
	h.expectScreen("Merge does not support skip")
	h.press("a")
	h.expectScreen("Skip current commit", "unavailable: Merge does not support skip")
	h.press("esc", "enter")
	if _, ok := h.app.top().(*conflictScreen); !ok {
		t.Fatalf("O06: Enter on conflicted file must open S09, got %T", h.app.top())
	}
	h.expectScreen("ours = current branch", "theirs = the branch being merged", "Conflict region 1 of 1")
	h.press("2")
	if !strings.Contains(fixture.ReadFile("shared.txt"), "<<<<<<<") {
		t.Fatal("O07: choosing a side must not write or stage")
	}
	h.menuSelect("Mark resolved (stage result)")
	if fixture.ReadFile("shared.txt") != "one\ntopic\nthree\n" || fixture.Git("show", ":shared.txt") != "one\ntopic\nthree\n" {
		t.Fatal("T23: resolved content must be in worktree and index")
	}
	if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", "MERGE_HEAD")); err != nil {
		t.Fatal("O07: mark resolved must not continue the merge")
	}
	if _, ok := h.app.top().(*operationScreen); !ok {
		t.Fatalf("after mark resolved expected S08, got %T", h.app.top())
	}
	h.press("c")
	if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", "MERGE_HEAD")); err == nil {
		t.Fatal("O03: continue should conclude the merge")
	}
	if parents := strings.Fields(fixture.Git("log", "-1", "--format=%P")); len(parents) != 2 {
		t.Fatal("merge commit expected")
	}
}

func TestCherryPickPlannerNoncontiguousOrder(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"base.txt": "b\n"})
	fixture.Git("switch", "-q", "-c", "side")
	fixture.Commit("one", map[string]string{"one.txt": "1\n"})
	fixture.Commit("two", map[string]string{"two.txt": "2\n"})
	fixture.Commit("three", map[string]string{"three.txt": "3\n"})
	fixture.Git("switch", "-q", "main")
	h := newHarness(t, fixture)
	h.palette("branch inspector")
	branches := h.app.top().(*branchScreen)
	for index, branch := range branches.branches {
		if branch.Name == "side" {
			branches.cursor = index
		}
	}
	h.press("enter")
	h.expectScreen("HISTORY · side", "three", "one")
	h.press(" ", "j", "j", " ")
	h.menuSelect("Cherry-pick selected")
	planner, ok := h.app.top().(*cherryPickScreen)
	if !ok {
		t.Fatalf("expected S06, got %T", h.app.top())
	}
	firstOIDs := append([]string{}, planner.oids...)
	h.expectScreen("1. [✓]", "three", "2. [✓]", "one", "Target: main @")
	h.press("esc")
	if len(h.app.commitSelection.order) != 2 {
		t.Fatal("J04: selection must survive leaving the planner")
	}
	h.press("c", "p")
	planner = h.app.top().(*cherryPickScreen)
	if strings.Join(planner.oids, " ") != strings.Join(firstOIDs, " ") {
		t.Fatal("J04: c p must use the same selected OIDs as the menu")
	}
	before := fixture.Snapshot()
	h.press("j", "K")
	if fixture.Snapshot() != before {
		t.Fatal("reorder must not touch Git")
	}
	h.press("esc")
	h.expectScreen("Discard this unexecuted plan?")
	h.press("esc")
	if _, ok := h.app.top().(*cherryPickScreen); !ok {
		t.Fatal("H08: cancelling the discard dialog keeps the plan")
	}
	h.press("p")
	h.expectScreen("Cherry-pick preview", "1. ", "one", "Conflicts cannot be predicted")
	h.press("esc")
	h.menuSelect("Execute plan")
	h.expectScreen("Execute cherry-pick of 2 commit(s)?", "not an atomic transaction")
	h.confirm()
	if got := strings.TrimSpace(fixture.Git("log", "--reverse", "--format=%s", "HEAD~2..HEAD")); got != "one\nthree" {
		t.Fatalf("T15: applied order = %q", got)
	}
	h.expectScreen("Cherry-pick finished", "→")
}

func TestCherryPickConflictRoutesToDashboard(t *testing.T) {
	fixture, topic := conflictRepo(t)
	h := newHarness(t, fixture)
	h.app.commitSelection.toggle(topic)
	h.palette("open commit history")
	h.press("c", "p")
	h.menuSelect("Execute plan")
	h.confirm()
	if _, ok := h.app.top().(*operationScreen); !ok {
		t.Fatalf("conflict must open S08, got %T", h.app.top())
	}
	h.expectScreen("CHERRY-PICK PAUSED", "shared.txt")
}

func TestPlannerInvalidatedWhenHeadMoves(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	fixture.Git("switch", "-q", "-c", "side")
	fixture.Commit("pick me", map[string]string{"p.txt": "p\n"})
	fixture.Git("switch", "-q", "main")
	h := newHarness(t, fixture)
	h.palette("branch inspector")
	branches := h.app.top().(*branchScreen)
	for index, branch := range branches.branches {
		if branch.Name == "side" {
			branches.cursor = index
		}
	}
	h.press("enter", "c", "p")
	fixture.Commit("external commit", map[string]string{"x.txt": "x\n"})
	head := h.head()
	_, cmd := h.app.Update(tea.FocusMsg{})
	h.run(cmd)
	h.menuSelect("Execute plan")
	if h.head() != head {
		t.Fatal("T25: stale plan must not execute")
	}
	h.expectScreen("HEAD moved since the plan was created")
}

func TestRebasePlannerValidationAndExecution(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	fixture.Commit("one", map[string]string{"one.txt": "1\n"})
	fixture.Commit("two", map[string]string{"two.txt": "2\n"})
	fixture.Commit("fix one", map[string]string{"one.txt": "1 fixed\n"})
	h := newHarness(t, fixture)
	h.palette("open commit history")
	h.press("G", "r", "i")
	planner, ok := h.app.top().(*rebaseScreen)
	if !ok {
		t.Fatalf("expected S07, got %T", h.app.top())
	}
	h.expectScreen("Onto (explicit base): ", "base", "3 commit(s)", "pick", "one", "two", "fix one")
	h.press("f")
	h.expectScreen("first commit cannot be squash/fixup")
	h.press("a")
	h.expectScreen("unavailable: first commit cannot be squash/fixup")
	h.press("esc", "p", "G", "K", "f", "v")
	h.expectScreen("Rebase plan preview", "fixup")
	h.press("esc")
	if planner.rows[1].Action != git.TodoFixup || planner.rows[1].Commit.Subject != "fix one" {
		t.Fatalf("unexpected plan %+v", planner.rows)
	}
	h.menuSelect("Execute plan")
	h.expectScreen("Execute interactive rebase?", "Commit IDs in the range will change")
	h.confirm()
	if got := strings.TrimSpace(fixture.Git("log", "--reverse", "--format=%s", "HEAD~2..HEAD")); got != "one\ntwo" {
		t.Fatalf("T18: history = %q", got)
	}
	if fixture.Git("show", "HEAD~1:one.txt") != "1 fixed\n" {
		t.Fatal("fixup must fold into first commit")
	}
	h.expectScreen("Rebase finished")
}

func TestRebaseEditStopAndPublishedWarning(t *testing.T) {
	fixture := gittest.New(t)
	fixture.BareRemote("origin")
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	fixture.Commit("one", map[string]string{"one.txt": "1\n"})
	fixture.Git("push", "-q", "-u", "origin", "main")
	h := newHarness(t, fixture)
	h.palette("open commit history")
	h.press("G", "r", "i")
	h.expectScreen("PUBLISHED on origin/main")
	h.press("e")
	h.menuSelect("Execute plan")
	h.expectScreen("Rebasing rewrites published history", "Acknowledge risk and execute")
	h.confirm()
	if _, ok := h.app.top().(*operationScreen); !ok {
		t.Fatalf("edit stop should open S08, got %T", h.app.top())
	}
	h.expectScreen("Stopped for edit at", "amend then Continue")
	if strings.Contains(h.screen(), "conflict(s) need resolution") {
		t.Fatal("T19/J08: edit stop must not report conflicts")
	}
	before := fixture.Snapshot()
	h.press("A")
	h.expectScreen("Abort interactive rebase?", "git rebase --abort", "Abort is not a universal")
	h.press("esc")
	if fixture.Snapshot() != before {
		t.Fatal("T20: Esc on abort confirmation must not mutate refs, worktree or sequencer")
	}
	h.press("c")
	if h.app.operation.Active() {
		t.Fatal("continue after edit should finish")
	}
}

func TestHistoryPaginationAndSelectionPersistence(t *testing.T) {
	fixture := gittest.New(t)
	var fastImport strings.Builder
	for index := range 250 {
		fmt.Fprintf(&fastImport, "commit refs/heads/main\ncommitter F <f@example.invalid> %d +0000\ndata 8\nc%06d\n", 1700000000+index, index)
		fmt.Fprintf(&fastImport, "M 100644 inline f.txt\ndata 7\nc%05d\n\n", index)
	}
	if _, err := fixture.Runner.Execute(context.Background(), git.Invocation{Args: []string{"fast-import", "--quiet"}, Stdin: []byte(fastImport.String())}); err != nil {
		t.Fatalf("fast-import: %v", err)
	}
	fixture.Git("reset", "-q", "--hard", "main")
	h := newHarness(t, fixture)
	h.palette("open commit history")
	if loaded := len(h.app.history.commits); loaded != git.HistoryPageSize {
		t.Fatalf("H01: first page should hold %d commits, got %d", git.HistoryPageSize, loaded)
	}
	h.press(" ")
	firstSelected := h.app.commitSelection.order[0]
	h.press("G")
	if loaded := len(h.app.history.commits); loaded != 2*git.HistoryPageSize {
		t.Fatalf("prefetch near the end should load a second page, got %d", loaded)
	}
	h.press(" ", "/")
	h.typeText("c000249")
	h.press("enter")
	h.expectScreen("loaded commits only", "Selected: 2 (1 hidden)")
	h.press("/", "esc", "esc")
	h.palette("open commit history")
	if len(h.app.commitSelection.order) != 2 || h.app.commitSelection.order[0] != firstSelected {
		t.Fatal("H03: selection must persist across filter, pages and revisits")
	}
	h.expectScreen("Selected: 2 (0 hidden)")
}

func TestSmallTerminalStacksPanels(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	fixture.Write("a.txt", "b\n")
	h := newHarness(t, fixture)
	_, cmd := h.app.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	h.run(cmd)
	view := h.screen()
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > 60 {
			t.Fatalf("line wider than terminal: %q", line)
		}
	}
	h.expectScreen("CHANGES", "DIFF INSPECTOR", "COMMIT COMPOSER", "? Help")
}

func TestBranchInspectorActions(t *testing.T) {
	fixture := gittest.New(t)
	remoteDir := fixture.BareRemote("origin")
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	h := newHarness(t, fixture)
	h.palette("branch inspector")
	h.expectScreen("BRANCH INSPECTOR", "No upstream (ahead/behind unknown)", "Comparison base (independent of upstream): main")
	h.press("a")
	h.expectScreen("Push", "unavailable: No upstream: use Push and set upstream", "Pull", "unavailable: No upstream: set one first")
	h.press("esc")

	h.menuSelect("Push and set upstream")
	h.press("enter")
	if strings.TrimSpace(fixture.Git("rev-parse", "--abbrev-ref", "main@{upstream}")) != "origin/main" {
		t.Fatal("push --set-upstream should configure origin/main")
	}
	h.expectScreen("→ origin/main ↑0 ↓0")

	h.menuSelect("Create branch at HEAD")
	h.typeText("feature/x")
	h.press("enter")
	if !strings.Contains(fixture.Git("branch", "--list", "feature/x"), "feature/x") {
		t.Fatal("create branch failed")
	}
	branches := h.app.top().(*branchScreen)
	for index, branch := range branches.branches {
		if branch.Name == "feature/x" {
			branches.cursor = index
		}
	}
	h.menuSelect("Switch to focused branch")
	if strings.TrimSpace(fixture.Git("branch", "--show-current")) != "feature/x" {
		t.Fatal("switch failed")
	}
	h.menuSelect("Set upstream")
	h.press("ctrl+u")
	for range len("origin/feature/x") {
		h.app.overlay.handleKey(h.app, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	h.typeText("origin/main")
	h.press("enter")
	if strings.TrimSpace(fixture.Git("rev-parse", "--abbrev-ref", "feature/x@{upstream}")) != "origin/main" {
		t.Fatal("set upstream failed")
	}

	other := gittest.New(t)
	other.Git("remote", "add", "origin", remoteDir)
	other.Git("fetch", "-q", "origin")
	other.Git("switch", "-q", "-c", "main", "origin/main")
	other.Commit("remote change", map[string]string{"r.txt": "r\n"})
	other.Git("push", "-q", "origin", "main")
	h.menuSelect("Fetch")
	h.expectScreen("→ origin/main ↑0 ↓1")
	h.menuSelect("Pull")
	if fixture.ReadFile("r.txt") != "r\n" {
		t.Fatal("pull did not bring remote change")
	}

	h.menuSelect("Set comparison base")
	h.press("ctrl+u")
	for range len("main") {
		h.app.overlay.handleKey(h.app, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	h.typeText("feature/x")
	h.press("enter")
	h.expectScreen("vs feature/x ↑0 ↓0")

	for index, branch := range branches.branches {
		if branch.Name == "main" {
			branches.cursor = index
		}
	}
	h.menuSelect("Delete focused local branch")
	h.expectScreen("git branch -d main")
	h.confirm()
	if strings.Contains(fixture.Git("branch", "--list", "main"), "main") {
		t.Fatal("delete branch failed")
	}
}

func TestSelectAllVisibleFiles(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n", "b.txt": "b\n", "c.txt": "c\n"})
	fixture.Write("a.txt", "A\n")
	fixture.Write("b.txt", "B\n")
	fixture.Write("c.txt", "C\n")
	fixture.Git("add", "--", "c.txt")
	before := fixture.Snapshot()
	h := newHarness(t, fixture)

	h.press("3", "S")
	if len(h.app.changes.selected) != 2 || h.app.changes.selected["c.txt"] {
		t.Fatalf("S must select only visible files, got %v", h.app.changes.selected)
	}
	if fixture.Snapshot() != before {
		t.Fatal("S must not mutate Git")
	}
	h.press("S")
	if len(h.app.changes.selected) != 0 {
		t.Fatalf("S with all visible selected must clear them, got %v", h.app.changes.selected)
	}
	h.press("1", "S")
	h.expectScreen("3 selected (0 hidden by filter)")
	h.press("3", "s")
	h.press("tab", "enter")
	if status := fixture.Git("status", "--porcelain"); status != "M  a.txt\nM  b.txt\nM  c.txt\n" {
		t.Fatalf("s after S must stage all selected files, status:\n%s", status)
	}
}
