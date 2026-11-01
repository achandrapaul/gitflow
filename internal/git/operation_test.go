package git_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func TestOperationDetectorActive(t *testing.T) {
	testCases := []struct {
		name     string
		pausedBy []string
		wantKind git.OperationKind
	}{
		{name: "no operation", wantKind: git.OperationNone},
		{name: "cherry-pick conflict", pausedBy: []string{"cherry-pick", "other"}, wantKind: git.OperationCherryPick},
		{name: "merge conflict", pausedBy: []string{"merge", "other"}, wantKind: git.OperationMerge},
		{name: "rebase conflict", pausedBy: []string{"rebase", "other"}, wantKind: git.OperationRebase},
		{name: "revert conflict", pausedBy: []string{"revert", "other"}, wantKind: git.OperationRevert},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.Commit("base", map[string]string{"conflict.txt": "base\n"})
			fixture.Git("switch", "-q", "-c", "other")
			fixture.Commit("other", map[string]string{"conflict.txt": "other\n"})
			fixture.Git("switch", "-q", "main")
			fixture.Commit("main", map[string]string{"conflict.txt": "main\n"})
			if testCase.wantKind == git.OperationRevert {
				fixture.Commit("main again", map[string]string{"conflict.txt": "main again\n"})
				testCase.pausedBy = []string{"revert", "HEAD~1"}
			}
			if testCase.pausedBy != nil {
				if _, err := fixture.Runner.Run(context.Background(), testCase.pausedBy...); err == nil {
					t.Fatalf("git %v unexpectedly succeeded", testCase.pausedBy)
				}
			}
			before := fixture.Snapshot()
			operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
			if err != nil || operation.Kind != testCase.wantKind {
				t.Fatalf("Active = %+v, %v; want kind %v", operation, err, testCase.wantKind)
			}
			if fixture.Snapshot() != before {
				t.Fatal("detection changed Git state")
			}
		})
	}
}

func revParse(fixture *gittest.Fixture, revision string) string {
	return strings.TrimSpace(fixture.Git("rev-parse", revision))
}

func TestOperationDetectorDetails(t *testing.T) {
	t.Run("merge", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Commit != revParse(fixture, gittest.FeatureBranch) || operation.SkipAllowed || operation.Total != 0 {
			t.Fatalf("merge = %+v, %v", operation, err)
		}
	})
	t.Run("rebase", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationRebase)
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Commit != revParse(fixture, gittest.FeatureBranch+"~1") || !operation.SkipAllowed {
			t.Fatalf("rebase = %+v, %v", operation, err)
		}
		if operation.Step != 1 || operation.Total != 2 || operation.Branch != gittest.FeatureBranch || operation.Onto != revParse(fixture, "main") {
			t.Fatalf("rebase progress = %+v", operation)
		}
	})
	t.Run("cherry-pick sequence", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationCherryPick)
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Commit != revParse(fixture, gittest.FeatureBranch+"~1") || !operation.SkipAllowed {
			t.Fatalf("cherry-pick = %+v, %v", operation, err)
		}
		if len(operation.Remaining) != 1 || !strings.HasPrefix(revParse(fixture, gittest.FeatureBranch), operation.Remaining[0]) {
			t.Fatalf("cherry-pick remaining = %v", operation.Remaining)
		}
	})
}

func resolveConflict(fixture *gittest.Fixture) {
	fixture.Write("conflict.txt", "a\nRESOLVED\nc\n")
	fixture.Git("add", "--", "conflict.txt")
}

func TestOperationControllerControls(t *testing.T) {
	kinds := []git.OperationKind{git.OperationMerge, git.OperationRebase, git.OperationCherryPick}
	for _, kind := range kinds {
		t.Run(fmt.Sprintf("continue %d", kind), func(t *testing.T) {
			fixture := gittest.NewPausedFixture(t, kind)
			controller := newController(fixture)
			before := fixture.Snapshot()
			if err := controller.Continue(context.Background(), kind); !errors.Is(err, git.ErrUnresolvedConflicts) {
				t.Fatalf("O03 Continue with conflicts = %v, want ErrUnresolvedConflicts", err)
			}
			if fixture.Snapshot() != before {
				t.Fatal("blocked Continue changed Git state")
			}
			resolveConflict(fixture)
			head := revParse(fixture, "HEAD")
			if err := controller.Continue(context.Background(), kind); err != nil {
				t.Fatalf("Continue: %v", err)
			}
			if revParse(fixture, "HEAD") == head {
				t.Fatal("Continue created no commit")
			}
			if got := fixture.Git("show", "HEAD:conflict.txt"); kind == git.OperationMerge && got != "a\nRESOLVED\nc\n" {
				t.Fatalf("merge commit content %q", got)
			}
		})
		t.Run(fmt.Sprintf("abort %d", kind), func(t *testing.T) {
			fixture := gittest.NewPausedFixture(t, kind)
			original := revParse(fixture, "HEAD")
			if kind == git.OperationRebase {
				original = revParse(fixture, gittest.FeatureBranch)
			}
			if err := newController(fixture).Abort(context.Background(), kind); err != nil {
				t.Fatalf("O05 Abort: %v", err)
			}
			operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
			if err != nil || operation.Active() || revParse(fixture, "HEAD") != original {
				t.Fatalf("after Abort operation %+v err %v HEAD %s want %s", operation, err, revParse(fixture, "HEAD"), original)
			}
		})
		t.Run(fmt.Sprintf("skip %d", kind), func(t *testing.T) {
			fixture := gittest.NewPausedFixture(t, kind)
			before := fixture.Snapshot()
			err := newController(fixture).Skip(context.Background(), kind)
			if kind == git.OperationMerge {
				if !errors.Is(err, git.ErrSkipUnsupported) || fixture.Snapshot() != before {
					t.Fatalf("O04 merge Skip = %v or changed state", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("O04 Skip: %v", err)
			}
			if got := fixture.Git("show", "HEAD:second.txt"); got != "second\n" {
				t.Fatalf("Skip did not apply the next commit: %q", got)
			}
		})
	}
	t.Run("changed operation", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		before := fixture.Snapshot()
		if err := newController(fixture).Abort(context.Background(), git.OperationRebase); !errors.Is(err, git.ErrOperationChanged) || fixture.Snapshot() != before {
			t.Fatalf("Abort of a different operation = %v or changed state", err)
		}
	})
	t.Run("failure keeps stderr", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		resolveConflict(fixture)
		fixture.Hook("commit-msg", "echo hook refused >&2\nexit 1\n")
		err := newController(fixture).Continue(context.Background(), git.OperationMerge)
		if err == nil || !strings.Contains(err.Error(), "hook refused") {
			t.Fatalf("failed Continue = %v, want Git output", err)
		}
		if operation, _ := git.NewOperationDetector(fixture.Runner).Active(context.Background()); operation.Kind != git.OperationMerge {
			t.Fatalf("failed Continue ended the merge: %+v", operation)
		}
	})
}

func newController(fixture *gittest.Fixture) *git.OperationController {
	return git.NewOperationController(fixture.Runner, git.NewStatusReader(fixture.Runner), git.NewOperationDetector(fixture.Runner))
}
