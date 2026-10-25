package git_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func detect(t *testing.T, fixture *gittest.Fixture) git.Operation {
	t.Helper()
	operation, err := git.DetectOperation(context.Background(), fixture.Runner, filepath.Join(fixture.Dir, ".git"))
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	return operation
}

func loadCommits(t *testing.T, fixture *gittest.Fixture, oids ...string) []git.Commit {
	t.Helper()
	commits, err := git.LoadCommits(context.Background(), fixture.Runner, oids)
	if err != nil {
		t.Fatalf("load commits: %v", err)
	}
	return commits
}

func subjects(fixture *gittest.Fixture, revisionRange string) string {
	return strings.TrimSpace(fixture.Git("log", "--reverse", "--format=%s", revisionRange))
}

func conflictFixture(t *testing.T) (*gittest.Fixture, string) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"shared.txt": "line one\nshared\nline three\n"})
	fixture.Git("switch", "-q", "-c", "topic")
	topic := fixture.Commit("topic edit", map[string]string{"shared.txt": "line one\ntopic\nline three\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main edit", map[string]string{"shared.txt": "line one\nmain\nline three\n"})
	return fixture, topic
}

func TestCherryPickScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("T15 noncontiguous commits in explicit queue order", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"base.txt": "b\n"})
		fixture.Git("switch", "-q", "-c", "source")
		first := fixture.Commit("one", map[string]string{"one.txt": "1\n"})
		fixture.Commit("two", map[string]string{"two.txt": "2\n"})
		third := fixture.Commit("three", map[string]string{"three.txt": "3\n"})
		fixture.Git("switch", "-q", "main")
		queue := loadCommits(t, fixture, third, first)
		invocation, err := git.CherryPickInvocation(queue)
		if err != nil {
			t.Fatalf("invocation: %v", err)
		}
		if _, err := fixture.Runner.Execute(ctx, invocation); err != nil {
			t.Fatalf("cherry-pick: %v", err)
		}
		if got := subjects(fixture, "HEAD~2..HEAD"); got != "three\none" {
			t.Fatalf("order = %q", got)
		}
		if detect(t, fixture).Active() {
			t.Fatal("operation should be complete")
		}
	})

	t.Run("T16 T17 conflict pauses with sequencer state detectable after reopen", func(t *testing.T) {
		fixture, topic := conflictFixture(t)
		fixture.Git("switch", "-q", "topic")
		extra := fixture.Commit("topic extra", map[string]string{"extra.txt": "e\n"})
		fixture.Git("switch", "-q", "main")
		invocation, _ := git.CherryPickInvocation(loadCommits(t, fixture, topic, extra))
		if _, err := fixture.Runner.Execute(ctx, invocation); err == nil {
			t.Fatal("expected conflict")
		}
		operation := detect(t, fixture)
		if operation.Kind != git.OperationCherryPick || operation.CurrentCommit != topic || !operation.HasRemaining || operation.Remaining != 2 {
			t.Fatalf("operation = %+v", operation)
		}
		if !operation.SupportsSkip() {
			t.Fatal("cherry-pick supports skip")
		}
		status, _ := git.LoadStatus(ctx, fixture.Runner)
		if !status.HasConflicts() {
			t.Fatal("expected unmerged entries")
		}
		if _, err := fixture.Runner.Execute(ctx, operation.AbortInvocation()); err != nil {
			t.Fatalf("abort: %v", err)
		}
		if detect(t, fixture).Active() {
			t.Fatal("abort should clear operation")
		}
	})

	t.Run("T27 empty cherry-pick pauses without fake success", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("switch", "-q", "-c", "topic")
		duplicate := fixture.Commit("same change", map[string]string{"a.txt": "b\n"})
		fixture.Git("switch", "-q", "main")
		fixture.Commit("same change on main", map[string]string{"a.txt": "b\n"})
		commits := loadCommits(t, fixture, duplicate)
		alreadyApplied, err := git.PatchAlreadyInHead(ctx, fixture.Runner, commits[0])
		if err != nil || !alreadyApplied {
			t.Fatalf("duplicate patch warning = %v err = %v", alreadyApplied, err)
		}
		invocation, _ := git.CherryPickInvocation(commits)
		if _, err := fixture.Runner.Execute(ctx, invocation); err == nil {
			t.Fatal("expected empty cherry-pick to stop")
		}
		operation := detect(t, fixture)
		if operation.Kind != git.OperationCherryPick || !operation.EmptyPick {
			t.Fatalf("operation = %+v", operation)
		}
		skip, _ := operation.SkipInvocation()
		if _, err := fixture.Runner.Execute(ctx, skip); err != nil {
			t.Fatalf("skip: %v", err)
		}
		if detect(t, fixture).Active() {
			t.Fatal("skip should finish")
		}
	})

	t.Run("merge commit requires mainline", func(t *testing.T) {
		fixture, topic := conflictFixture(t)
		_ = topic
		fixture.Git("switch", "-q", "-c", "side", "HEAD~1")
		fixture.Commit("side", map[string]string{"side.txt": "s\n"})
		fixture.Git("switch", "-q", "main")
		fixture.Git("merge", "-q", "--no-edit", "side")
		merge := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
		if _, err := git.CherryPickInvocation(loadCommits(t, fixture, merge)); !errors.Is(err, git.ErrMergeCommitNeedsMainline) {
			t.Fatalf("expected mainline error, got %v", err)
		}
	})
}

func TestRebaseScenarios(t *testing.T) {
	ctx := context.Background()

	linearFixture := func(t *testing.T) (*gittest.Fixture, string, []git.Commit) {
		fixture := gittest.New(t)
		base := fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Commit("one", map[string]string{"one.txt": "1\n"})
		fixture.Commit("two", map[string]string{"two.txt": "2\n"})
		fixture.Commit("fix one", map[string]string{"one.txt": "1 fixed\n"})
		commits, err := git.RebaseRange(ctx, fixture.Runner, base)
		if err != nil || len(commits) != 3 {
			t.Fatalf("range = %+v err = %v", commits, err)
		}
		return fixture, base, commits
	}

	t.Run("T18 reorder and fixup produce planned history", func(t *testing.T) {
		fixture, base, commits := linearFixture(t)
		rows := []git.TodoRow{{Action: git.TodoPick, Commit: commits[0]}, {Action: git.TodoFixup, Commit: commits[2]}, {Action: git.TodoPick, Commit: commits[1]}}
		invocation, cleanup, err := git.StartRebaseInvocation(base, rows)
		if err != nil {
			t.Fatalf("invocation: %v", err)
		}
		defer cleanup()
		if _, err := fixture.Runner.Execute(ctx, invocation); err != nil {
			t.Fatalf("rebase: %v", err)
		}
		if got := subjects(fixture, base+"..HEAD"); got != "one\ntwo" {
			t.Fatalf("history = %q", got)
		}
		if fixture.Git("show", "HEAD~1:one.txt") != "1 fixed\n" {
			t.Fatal("fixup content not folded into first commit")
		}
	})

	t.Run("T18 invalid first-row fixup rejected", func(t *testing.T) {
		fixture, base, commits := linearFixture(t)
		head := fixture.Git("rev-parse", "HEAD")
		rows := []git.TodoRow{{Action: git.TodoDrop, Commit: commits[0]}, {Action: git.TodoFixup, Commit: commits[1]}, {Action: git.TodoPick, Commit: commits[2]}}
		if _, _, err := git.StartRebaseInvocation(base, rows); !errors.Is(err, git.ErrSquashWithoutPick) {
			t.Fatalf("expected ErrSquashWithoutPick, got %v", err)
		}
		if fixture.Git("rev-parse", "HEAD") != head {
			t.Fatal("HEAD changed")
		}
	})

	t.Run("T19 edit stop reports edit without fake conflicts", func(t *testing.T) {
		fixture, base, commits := linearFixture(t)
		rows := []git.TodoRow{{Action: git.TodoEdit, Commit: commits[0]}, {Action: git.TodoPick, Commit: commits[1]}, {Action: git.TodoPick, Commit: commits[2]}}
		invocation, cleanup, _ := git.StartRebaseInvocation(base, rows)
		defer cleanup()
		if _, err := fixture.Runner.Execute(ctx, invocation); err != nil {
			t.Fatalf("rebase: %v", err)
		}
		operation := detect(t, fixture)
		if operation.Kind != git.OperationRebase || !operation.EditStop || !operation.HasProgress || operation.Step != 1 || operation.Total != 3 {
			t.Fatalf("operation = %+v", operation)
		}
		status, _ := git.LoadStatus(ctx, fixture.Runner)
		if status.HasConflicts() {
			t.Fatal("edit stop must not report conflicts")
		}
		if _, err := fixture.Runner.Execute(ctx, operation.ContinueInvocation()); err != nil {
			t.Fatalf("continue: %v", err)
		}
		if detect(t, fixture).Active() {
			t.Fatal("rebase should finish")
		}
	})

	t.Run("T19 T21 conflict stop then confirmed abort restores branch", func(t *testing.T) {
		fixture, _ := conflictFixture(t)
		fixture.Git("switch", "-q", "topic")
		head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
		mainTip := strings.TrimSpace(fixture.Git("rev-parse", "main"))
		commits, err := git.RebaseRange(ctx, fixture.Runner, strings.TrimSpace(fixture.Git("merge-base", "main", "topic")))
		if err != nil {
			t.Fatalf("range: %v", err)
		}
		invocation, cleanup, _ := git.StartRebaseInvocation(mainTip, []git.TodoRow{{Action: git.TodoPick, Commit: commits[0]}})
		defer cleanup()
		if _, err := fixture.Runner.Execute(ctx, invocation); err == nil {
			t.Fatal("expected conflict")
		}
		operation := detect(t, fixture)
		if operation.Kind != git.OperationRebase || operation.EditStop || operation.HeadName != "topic" {
			t.Fatalf("operation = %+v", operation)
		}
		if _, err := fixture.Runner.Execute(ctx, operation.AbortInvocation()); err != nil {
			t.Fatalf("abort: %v", err)
		}
		if detect(t, fixture).Active() || strings.TrimSpace(fixture.Git("rev-parse", "HEAD")) != head {
			t.Fatal("abort did not restore topic")
		}
	})

	t.Run("base must be ancestor and range must be linear", func(t *testing.T) {
		fixture, topic := conflictFixture(t)
		if _, err := git.RebaseRange(ctx, fixture.Runner, topic); !errors.Is(err, git.ErrNotAncestor) {
			t.Fatalf("expected ErrNotAncestor, got %v", err)
		}
		base := strings.TrimSpace(fixture.Git("rev-parse", "HEAD~1"))
		fixture.Git("switch", "-q", "-c", "side", base)
		fixture.Commit("side", map[string]string{"side.txt": "s\n"})
		fixture.Git("switch", "-q", "main")
		fixture.Git("merge", "-q", "--no-edit", "side")
		if _, err := git.RebaseRange(ctx, fixture.Runner, base); !errors.Is(err, git.ErrMergeInRange) {
			t.Fatalf("expected ErrMergeInRange, got %v", err)
		}
	})
}

func TestMergeAndConflictResolution(t *testing.T) {
	ctx := context.Background()

	t.Run("T22 merge conflict offers continue and abort, not skip", func(t *testing.T) {
		fixture, _ := conflictFixture(t)
		fixture.GitFails("merge", "topic")
		operation := detect(t, fixture)
		if operation.Kind != git.OperationMerge || operation.SupportsSkip() {
			t.Fatalf("operation = %+v", operation)
		}
		if _, err := operation.SkipInvocation(); !errors.Is(err, git.ErrSkipUnsupported) {
			t.Fatalf("expected ErrSkipUnsupported, got %v", err)
		}
		if strings.Join(operation.ContinueInvocation().Args, " ") != "merge --continue" || strings.Join(operation.AbortInvocation().Args, " ") != "merge --abort" {
			t.Fatal("unexpected merge invocations")
		}
	})

	for _, testCase := range []struct {
		name   string
		choice git.Choice
		want   string
	}{
		{"ours", git.ChoiceOurs, "line one\nmain\nline three\n"},
		{"theirs", git.ChoiceTheirs, "line one\ntopic\nline three\n"},
		{"both", git.ChoiceBoth, "line one\nmain\ntopic\nline three\n"},
	} {
		t.Run("T23 resolve with "+testCase.name, func(t *testing.T) {
			fixture, _ := conflictFixture(t)
			fixture.GitFails("merge", "topic")
			conflict, err := git.LoadConflict(ctx, fixture.Runner, "shared.txt")
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if conflict.TextSidesSupported() != nil || len(conflict.Regions) != 1 {
				t.Fatalf("conflict = %+v", conflict)
			}
			if _, err := conflict.Resolve(map[int]git.Choice{}); !errors.Is(err, git.ErrUnresolvedRegions) {
				t.Fatalf("expected unresolved error, got %v", err)
			}
			if err := git.MarkResolved(ctx, fixture.Runner, "shared.txt", nil); !errors.Is(err, git.ErrMarkersRemain) {
				t.Fatalf("expected markers error, got %v", err)
			}
			resolved, err := conflict.Resolve(map[int]git.Choice{0: testCase.choice})
			if err != nil || resolved != testCase.want {
				t.Fatalf("resolved = %q err = %v", resolved, err)
			}
			if !strings.Contains(fixture.ReadFile("shared.txt"), "<<<<<<<") {
				t.Fatal("choosing a side must not write the file before Mark resolved")
			}
			if err := git.MarkResolved(ctx, fixture.Runner, "shared.txt", &resolved); err != nil {
				t.Fatalf("mark resolved: %v", err)
			}
			if fixture.ReadFile("shared.txt") != testCase.want || fixture.Git("show", ":shared.txt") != testCase.want {
				t.Fatal("worktree or index content wrong")
			}
			if status, _ := git.LoadStatus(ctx, fixture.Runner); status.HasConflicts() {
				t.Fatal("conflict should be resolved in index")
			}
			if detect(t, fixture).Kind != git.OperationMerge {
				t.Fatal("marking resolved must not continue the merge")
			}
		})
	}

	t.Run("T24 binary conflict has no text sides", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"blob.bin": "\x00base"})
		fixture.Git("switch", "-q", "-c", "topic")
		fixture.Commit("topic", map[string]string{"blob.bin": "\x00topic"})
		fixture.Git("switch", "-q", "main")
		fixture.Commit("main", map[string]string{"blob.bin": "\x00main"})
		fixture.GitFails("merge", "topic")
		conflict, err := git.LoadConflict(ctx, fixture.Runner, "blob.bin")
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if !conflict.Binary || !errors.Is(conflict.TextSidesSupported(), git.ErrNoTextSides) {
			t.Fatalf("conflict = %+v", conflict)
		}
		if _, err := conflict.Resolve(map[int]git.Choice{0: git.ChoiceOurs}); !errors.Is(err, git.ErrNoTextSides) {
			t.Fatalf("expected ErrNoTextSides, got %v", err)
		}
	})
}
