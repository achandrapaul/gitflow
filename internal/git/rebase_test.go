package git_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func newRebaser(fixture *gittest.Fixture) *git.Rebaser {
	return git.NewRebaser(fixture.Runner, git.NewStatusReader(fixture.Runner), git.NewOperationDetector(fixture.Runner))
}

func linearHistory(fixture *gittest.Fixture) (base string) {
	base = fixture.Commit("base", map[string]string{"base.txt": "base\n"})
	fixture.Commit("first", map[string]string{"first.txt": "first\n"})
	fixture.Commit("second", map[string]string{"second.txt": "second\n"})
	fixture.Commit("third", map[string]string{"third.txt": "third\n"})
	return base
}

func stepSubjects(steps []git.TodoStep) []string {
	var subjects []string
	for _, step := range steps {
		subjects = append(subjects, step.Subject)
	}
	return subjects
}

func runRebase(t *testing.T, rebaser *git.Rebaser, reviewed git.RebaseRange) git.SequenceOutcome {
	t.Helper()
	run, err := rebaser.Prepare(context.Background(), reviewed)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	commandErr := run.Command.Run()
	outcome, err := rebaser.Finish(context.Background(), run)
	if err != nil {
		t.Fatalf("Finish after %v: %v", commandErr, err)
	}
	return outcome
}

func TestRebaserRange(t *testing.T) {
	t.Run("lists base..HEAD oldest first on the checked-out branch", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		base := linearHistory(fixture)
		reviewed, err := newRebaser(fixture).Range(context.Background(), base)
		if err != nil {
			t.Fatalf("Range: %v", err)
		}
		if reviewed.Branch != "main" || reviewed.Base != base || reviewed.Head != revParse(fixture, "HEAD") {
			t.Fatalf("range = %+v", reviewed)
		}
		if subjects := stepSubjects(reviewed.Steps); !slices.Equal(subjects, []string{"first", "second", "third"}) {
			t.Fatalf("subjects = %v", subjects)
		}
		for _, step := range reviewed.Steps {
			if step.Action != git.TodoPick || len(step.OID) != len(base) {
				t.Fatalf("step = %+v, want pick with full OID", step)
			}
		}
	})
	blocked := []struct {
		name  string
		setup func(fixture *gittest.Fixture) string
		want  error
	}{
		{name: "base equal to HEAD", want: git.ErrRebaseEmptyRange, setup: func(fixture *gittest.Fixture) string {
			linearHistory(fixture)
			return revParse(fixture, "HEAD")
		}},
		{name: "base off the checked-out branch", want: git.ErrRebaseBaseOffBranch, setup: func(fixture *gittest.Fixture) string {
			linearHistory(fixture)
			fixture.Git("switch", "-q", "-c", "side", "HEAD~1")
			side := fixture.Commit("side", map[string]string{"side.txt": "side\n"})
			fixture.Git("switch", "-q", "main")
			return side
		}},
		{name: "merge commit in range", want: git.ErrRebaseMergeInRange, setup: func(fixture *gittest.Fixture) string {
			base := linearHistory(fixture)
			fixture.Git("switch", "-q", "-c", "side", base)
			fixture.Commit("side", map[string]string{"side.txt": "side\n"})
			fixture.Git("switch", "-q", "main")
			fixture.Git("merge", "-q", "--no-ff", "-m", "merge side", "side")
			return base
		}},
		{name: "detached HEAD", want: git.ErrRebaseDetached, setup: func(fixture *gittest.Fixture) string {
			base := linearHistory(fixture)
			fixture.Git("switch", "-q", "--detach", "HEAD")
			return base
		}},
	}
	for _, testCase := range blocked {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			base := testCase.setup(fixture)
			if _, err := newRebaser(fixture).Range(context.Background(), base); !errors.Is(err, testCase.want) {
				t.Fatalf("Range error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestRebaserValidate(t *testing.T) {
	steps := func(actions ...git.TodoAction) []git.TodoStep {
		var todo []git.TodoStep
		for _, action := range actions {
			todo = append(todo, git.TodoStep{Action: action})
		}
		return todo
	}
	cases := []struct {
		name    string
		steps   []git.TodoStep
		wantErr bool
	}{
		{name: "T18 first-row fixup", steps: steps(git.TodoFixup, git.TodoPick), wantErr: true},
		{name: "first-row squash", steps: steps(git.TodoSquash, git.TodoPick), wantErr: true},
		{name: "fixup after a dropped first row", steps: steps(git.TodoDrop, git.TodoFixup), wantErr: true},
		{name: "fixup after pick", steps: steps(git.TodoPick, git.TodoFixup, git.TodoSquash)},
		{name: "all actions", steps: steps(git.TodoReword, git.TodoEdit, git.TodoDrop, git.TodoPick)},
	}
	rebaser := git.NewRebaser(nil, nil, nil)
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := rebaser.Validate(testCase.steps)
			if testCase.wantErr != errors.Is(err, git.ErrRebaseOrphanCombine) || (!testCase.wantErr && err != nil) {
				t.Fatalf("Validate = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestRebaserPublished(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	base := linearHistory(fixture)
	rebaser := newRebaser(fixture)
	reviewed, err := rebaser.Range(context.Background(), base)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if refs, err := rebaser.Published(context.Background(), reviewed); err != nil || len(refs) != 0 {
		t.Fatalf("unpushed Published = %v, %v", refs, err)
	}
	fixture.Git("push", "-q", "origin", "HEAD~2:refs/heads/main")
	fixture.Git("fetch", "-q", "origin")
	if refs, err := rebaser.Published(context.Background(), reviewed); err != nil || !slices.Equal(refs, []string{"origin/main"}) {
		t.Fatalf("pushed Published = %v, %v", refs, err)
	}
}

func TestRebaserExecute(t *testing.T) {
	t.Run("T18 reorder, fixup and drop produce the planned history", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		base := linearHistory(fixture)
		rebaser := newRebaser(fixture)
		reviewed, err := rebaser.Range(context.Background(), base)
		if err != nil {
			t.Fatalf("Range: %v", err)
		}
		first, second, third := reviewed.Steps[0], reviewed.Steps[1], reviewed.Steps[2]
		second.Action, first.Action = git.TodoFixup, git.TodoDrop
		reviewed.Steps = []git.TodoStep{third, second, first}
		outcome := runRebase(t, rebaser, reviewed)
		if outcome.Paused || len(outcome.Completed) != 1 {
			t.Fatalf("outcome = %+v", outcome)
		}
		if subjects := fixture.Git("log", "--format=%s", base+"..HEAD"); subjects != "third\n" {
			t.Fatalf("subjects = %q, want third only", subjects)
		}
		if files := fixture.Git("ls-tree", "--name-only", "HEAD"); files != "base.txt\nsecond.txt\nthird.txt\n" {
			t.Fatalf("tree = %q, want second folded into third and first dropped", files)
		}
		if branch := strings.TrimSpace(fixture.Git("branch", "--show-current")); branch != "main" {
			t.Fatalf("branch = %q", branch)
		}
	})
	t.Run("T19 edit stop pauses without conflicts and names the stop action", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		base := linearHistory(fixture)
		rebaser := newRebaser(fixture)
		reviewed, _ := rebaser.Range(context.Background(), base)
		reviewed.Steps[1].Action = git.TodoEdit
		outcome := runRebase(t, rebaser, reviewed)
		if !outcome.Paused || len(outcome.Completed) != 2 {
			t.Fatalf("outcome = %+v, want paused after two commits", outcome)
		}
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Kind != git.OperationRebase || operation.StopAction != "edit" {
			t.Fatalf("operation = %+v, %v", operation, err)
		}
		if unmerged := fixture.Git("diff", "--name-only", "--diff-filter=U"); unmerged != "" {
			t.Fatalf("edit stop reports conflicts %q", unmerged)
		}
	})
	t.Run("T19 conflicting reorder pauses on the conflicting pick", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		base := fixture.Commit("base", gittest.ConflictBase)
		fixture.Commit("main", gittest.ConflictMain)
		fixture.Commit("feature", gittest.FeatureOne)
		rebaser := newRebaser(fixture)
		reviewed, _ := rebaser.Range(context.Background(), base)
		reviewed.Steps = []git.TodoStep{reviewed.Steps[1], reviewed.Steps[0]}
		outcome := runRebase(t, rebaser, reviewed)
		if !outcome.Paused {
			t.Fatalf("outcome = %+v, want paused", outcome)
		}
		operation, _ := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if operation.StopAction != "pick" {
			t.Fatalf("stop action = %q, want pick", operation.StopAction)
		}
		if unmerged := fixture.Git("diff", "--name-only", "--diff-filter=U"); unmerged != "conflict.txt\n" {
			t.Fatalf("unmerged = %q", unmerged)
		}
	})
}

func TestRebaserPrepareRechecks(t *testing.T) {
	cases := []struct {
		name   string
		change func(fixture *gittest.Fixture)
		edit   func(reviewed *git.RebaseRange)
		want   error
	}{
		{name: "T25 HEAD moved since review", want: git.ErrStaleRef, change: func(fixture *gittest.Fixture) {
			fixture.Commit("external", map[string]string{"external.txt": "external\n"})
		}},
		{name: "tracked changes", want: git.ErrRebaseTrackedChanges, change: func(fixture *gittest.Fixture) {
			fixture.Dirty("first.txt")
		}},
		{name: "invalid plan", want: git.ErrRebaseOrphanCombine, edit: func(reviewed *git.RebaseRange) {
			reviewed.Steps[0].Action = git.TodoFixup
		}},
		{name: "plan lost a commit", want: git.ErrRebasePlanMismatch, edit: func(reviewed *git.RebaseRange) {
			reviewed.Steps = reviewed.Steps[1:]
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			base := linearHistory(fixture)
			rebaser := newRebaser(fixture)
			reviewed, _ := rebaser.Range(context.Background(), base)
			if testCase.change != nil {
				testCase.change(fixture)
			}
			if testCase.edit != nil {
				testCase.edit(&reviewed)
			}
			before := fixture.Snapshot()
			if _, err := rebaser.Prepare(context.Background(), reviewed); !errors.Is(err, testCase.want) {
				t.Fatalf("Prepare error = %v, want %v", err, testCase.want)
			}
			if after := fixture.Snapshot(); after != before {
				t.Fatalf("Prepare mutated Git:\n%s\n---\n%s", before, after)
			}
		})
	}
	t.Run("paused operation", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		rebaser := newRebaser(fixture)
		reviewed := git.RebaseRange{Branch: "main", Head: revParse(fixture, "HEAD"), Base: revParse(fixture, "HEAD~1")}
		if _, err := rebaser.Prepare(context.Background(), reviewed); !errors.Is(err, git.ErrRebaseOperationActive) {
			t.Fatalf("Prepare error = %v", err)
		}
	})
}
