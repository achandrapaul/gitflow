package recovery_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

func newRestore(fixture *gittest.Fixture) *recovery.CherryPickRestore {
	runner := fixture.Runner
	operations := git.NewOperationDetector(runner)
	return recovery.NewCherryPickRestore(git.NewStatusReader(runner), operations, git.NewDiffer(runner), git.NewStasher(runner), git.NewCherryPicker(runner, operations), recovery.NewStateStore(filepath.Join(fixture.Dir, ".git", "gitflow-state.json")))
}

func head(fixture *gittest.Fixture) string {
	return strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
}

func sideCommits(fixture *gittest.Fixture) []git.CommitSummary {
	base := fixture.Commit("base", map[string]string{"tracked.txt": "tracked\n"})
	fixture.Git("switch", "-q", "-c", "side")
	first := fixture.Commit("first", map[string]string{"first.txt": "first\n"})
	second := fixture.Commit("second", map[string]string{"second.txt": "second\n"})
	fixture.Git("switch", "-q", "main")
	return []git.CommitSummary{{OID: second, Parents: []string{first}}, {OID: first, Parents: []string{base}}}
}

func blockerKinds(result recovery.PreflightResult) []recovery.BlockerKind {
	var kinds []recovery.BlockerKind
	for _, blocker := range result.Blockers {
		kinds = append(kinds, blocker.Kind)
	}
	return kinds
}

func TestCherryPickRestorePreflight(t *testing.T) {
	testCases := []struct {
		name         string
		prepare      func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary
		moveHead     bool
		wantBlockers []recovery.BlockerKind
		wantDirty    []string
	}{
		{name: "clean target", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary { return commits }},
		{name: "T89 staged tracked file", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary {
			fixture.Dirty("tracked.txt")
			fixture.Git("add", "--", "tracked.txt")
			return commits
		}, wantDirty: []string{"tracked.txt"}},
		{name: "T91 non-colliding untracked file", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary {
			fixture.Write("notes.txt", "keep\n")
			return commits
		}},
		{name: "T92 untracked collision", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary {
			fixture.Write("first.txt", "mine\n")
			return commits
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockUntrackedCollision}},
		{name: "HEAD moved since review", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary { return commits }, moveHead: true, wantBlockers: []recovery.BlockerKind{recovery.BlockStale}},
		{name: "missing commit", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary {
			return append(commits, git.CommitSummary{OID: strings.Repeat("1", 40), Parents: []string{commits[0].OID}})
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockStale}},
		{name: "merge commit without mainline", prepare: func(fixture *gittest.Fixture, commits []git.CommitSummary) []git.CommitSummary {
			fixture.Git("switch", "-q", "side")
			fixture.Git("merge", "-q", "--no-ff", "-m", "merge main", "main")
			merge := git.CommitSummary{OID: head(fixture), Parents: []string{commits[0].OID, strings.TrimSpace(fixture.Git("rev-parse", "main"))}}
			fixture.Git("switch", "-q", "main")
			return append(commits, merge)
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockMergeCommit}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			commits := testCase.prepare(fixture, sideCommits(fixture))
			reviewedHead := head(fixture)
			if testCase.moveHead {
				fixture.Commit("moved", map[string]string{"moved.txt": "moved\n"})
			}
			before := fixture.Snapshot()
			result, err := newRestore(fixture).Preflight(context.Background(), commits, reviewedHead)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if !slices.Equal(blockerKinds(result), testCase.wantBlockers) || !slices.Equal(result.Status.TrackedDirtyPaths, testCase.wantDirty) || result.NeedsStash != (testCase.wantDirty != nil) {
				t.Fatalf("Preflight = blockers %v dirty %v needsStash %v; want %v %v", result.Blockers, result.Status.TrackedDirtyPaths, result.NeedsStash, testCase.wantBlockers, testCase.wantDirty)
			}
			if result.Target != "main" || len(result.Commits) != len(commits) {
				t.Fatalf("Preflight target %q commits %v", result.Target, result.Commits)
			}
			if fixture.Snapshot() != before {
				t.Fatal("Preflight changed Git state")
			}
		})
	}
	t.Run("T93 paused operation blocks the new plan", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		before := fixture.Snapshot()
		result, err := newRestore(fixture).Preflight(context.Background(), []git.CommitSummary{{OID: strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch)), Parents: []string{strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch+"~1"))}}}, head(fixture))
		if err != nil || !slices.Contains(blockerKinds(result), recovery.BlockSequencer) || !slices.Contains(blockerKinds(result), recovery.BlockUnmerged) || result.NeedsStash {
			t.Fatalf("Preflight with paused merge = %+v, %v", result.Blockers, err)
		}
		if fixture.Snapshot() != before {
			t.Fatal("Preflight changed paused state")
		}
	})
}

func TestCherryPickRestoreExecute(t *testing.T) {
	t.Run("T99 reviewed whole commits land in reviewed order", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		commits := sideCommits(fixture)
		fixture.Write("notes.txt", "keep\n")
		reviewedHead := head(fixture)
		outcome, err := newRestore(fixture).Execute(context.Background(), commits, reviewedHead)
		if err != nil || outcome.Paused || len(outcome.Completed) != 2 {
			t.Fatalf("Execute = %+v, %v", outcome, err)
		}
		if subjects := fixture.Git("log", "--format=%s", "--reverse", reviewedHead+"..HEAD"); subjects != "second\nfirst\n" {
			t.Fatalf("subjects %q, want second then first", subjects)
		}
		if content, err := os.ReadFile(filepath.Join(fixture.Dir, "notes.txt")); err != nil || string(content) != "keep\n" {
			t.Fatalf("T91 untracked file = %q, %v", content, err)
		}
	})
	blockedCases := []struct {
		name    string
		prepare func(fixture *gittest.Fixture)
	}{
		{name: "T89 dirty tracked file is not stashed", prepare: func(fixture *gittest.Fixture) { fixture.Dirty("tracked.txt") }},
		{name: "T92 untracked collision keeps bytes", prepare: func(fixture *gittest.Fixture) { fixture.Write("second.txt", "mine\n") }},
		{name: "HEAD moved after review", prepare: func(fixture *gittest.Fixture) {
			fixture.Commit("moved", map[string]string{"moved.txt": "moved\n"})
		}},
	}
	for _, blockedCase := range blockedCases {
		t.Run(blockedCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			commits := sideCommits(fixture)
			reviewedHead := head(fixture)
			blockedCase.prepare(fixture)
			before := fixture.Snapshot()
			if _, err := newRestore(fixture).Execute(context.Background(), commits, reviewedHead); !errors.Is(err, recovery.ErrExecutionBlocked) {
				t.Fatalf("Execute = %v, want ErrExecutionBlocked", err)
			}
			if fixture.Snapshot() != before {
				t.Fatal("blocked Execute changed Git state")
			}
			if list := fixture.Git("stash", "list"); list != "" {
				t.Fatalf("blocked Execute created a stash: %q", list)
			}
		})
	}
}

const longMiddle = "2\n3\n4\n5\n6\n7\n8\n9\n"

func newSelectiveApply(fixture *gittest.Fixture) *recovery.SelectiveApply {
	runner := fixture.Runner
	return recovery.NewSelectiveApply(git.NewStatusReader(runner), git.NewOperationDetector(runner), git.NewDiffer(runner))
}

type fileSources struct {
	base, first, second string
}

func sideFileCommits(fixture *gittest.Fixture) fileSources {
	base := fixture.Commit("base", map[string]string{"tracked.txt": "tracked\n", "shared.txt": "top\n" + longMiddle + "bottom\n", "image.bin": "\x00\x01"})
	fixture.Git("switch", "-q", "-c", "side")
	first := fixture.Commit("first", map[string]string{"shared.txt": "ONE\n" + longMiddle + "bottom\n", "added.txt": "added\n", "image.bin": "\x00\x02"})
	second := fixture.Commit("second", map[string]string{"shared.txt": "TWO\n" + longMiddle + "bottom\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", map[string]string{"main.txt": "main\n"})
	return fileSources{base: base, first: first, second: second}
}

func selectiveSteps(t *testing.T, fixture *gittest.Fixture, selections ...git.FileSelection) []git.ApplyStep {
	t.Helper()
	var steps []git.ApplyStep
	for _, selection := range selections {
		patch, err := git.NewDiffer(fixture.Runner).FilePatch(context.Background(), selection)
		if err != nil {
			t.Fatalf("FilePatch %s: %v", selection.Path, err)
		}
		steps = append(steps, git.ApplyStep{Selection: selection, Patch: patch})
	}
	return steps
}

func TestSelectiveApplyPreflight(t *testing.T) {
	testCases := []struct {
		name         string
		prepare      func(fixture *gittest.Fixture)
		selections   func(sources fileSources) []git.FileSelection
		moveHead     bool
		wantBlockers []recovery.BlockerKind
	}{
		{name: "T115 clean tracked state with a noncolliding untracked file", prepare: func(fixture *gittest.Fixture) { fixture.Write("notes.txt", "keep\n") }},
		{name: "T170 dirty tracked state blocks with no stash offer", prepare: func(fixture *gittest.Fixture) { fixture.Dirty("tracked.txt") }, wantBlockers: []recovery.BlockerKind{recovery.BlockDirtyTracked}},
		{name: "T148 staged tracked change blocks", prepare: func(fixture *gittest.Fixture) {
			fixture.Dirty("tracked.txt")
			fixture.Git("add", "--", "tracked.txt")
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockDirtyTracked}},
		{name: "T117 untracked collision", prepare: func(fixture *gittest.Fixture) { fixture.Write("added.txt", "mine\n") }, wantBlockers: []recovery.BlockerKind{recovery.BlockUntrackedCollision}},
		{name: "T117 binary change is unsupported", selections: func(sources fileSources) []git.FileSelection {
			return []git.FileSelection{{OID: sources.first, Parent: sources.base, Path: "image.bin", Status: 'M'}}
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockUnsupportedChange}},
		{name: "HEAD moved since preview", moveHead: true, wantBlockers: []recovery.BlockerKind{recovery.BlockStale}},
		{name: "T116 overlapping patches apply in selection order", selections: func(sources fileSources) []git.FileSelection {
			return []git.FileSelection{{OID: sources.first, Parent: sources.base, Path: "shared.txt", Status: 'M'}, {OID: sources.second, Parent: sources.first, Path: "shared.txt", Status: 'M'}}
		}},
		{name: "T117 overlapping patches that do not apply in order", selections: func(sources fileSources) []git.FileSelection {
			return []git.FileSelection{{OID: sources.second, Parent: sources.first, Path: "shared.txt", Status: 'M'}, {OID: sources.first, Parent: sources.base, Path: "shared.txt", Status: 'M'}}
		}, wantBlockers: []recovery.BlockerKind{recovery.BlockPatchDoesNotApply}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			sources := sideFileCommits(fixture)
			selections := []git.FileSelection{{OID: sources.first, Parent: sources.base, Path: "added.txt", Status: 'A'}, {OID: sources.first, Parent: sources.base, Path: "shared.txt", Status: 'M'}}
			if testCase.selections != nil {
				selections = testCase.selections(sources)
			}
			steps := selectiveSteps(t, fixture, selections...)
			reviewedHead := head(fixture)
			if testCase.prepare != nil {
				testCase.prepare(fixture)
			}
			if testCase.moveHead {
				fixture.Commit("moved", map[string]string{"moved.txt": "moved\n"})
			}
			before := fixture.Snapshot()
			result, err := newSelectiveApply(fixture).Preflight(context.Background(), steps, reviewedHead)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if !slices.Equal(blockerKinds(result), testCase.wantBlockers) || result.NeedsStash || result.StashOffered || result.Target != "main" {
				t.Fatalf("Preflight = blockers %v needsStash %v stashOffered %v target %q; want %v", result.Blockers, result.NeedsStash, result.StashOffered, result.Target, testCase.wantBlockers)
			}
			if fixture.Snapshot() != before {
				t.Fatal("Preflight changed Git state")
			}
		})
	}
	t.Run("paused operation blocks selective apply", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		before := fixture.Snapshot()
		feature := strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch))
		steps := []git.ApplyStep{{Selection: git.FileSelection{OID: feature, Parent: strings.TrimSpace(fixture.Git("rev-parse", gittest.FeatureBranch+"~1")), Path: "conflict.txt", Status: 'M'}}}
		result, err := newSelectiveApply(fixture).Preflight(context.Background(), steps, head(fixture))
		if err != nil || !slices.Contains(blockerKinds(result), recovery.BlockSequencer) || !slices.Contains(blockerKinds(result), recovery.BlockUnmerged) {
			t.Fatalf("Preflight with paused merge = %+v, %v", result.Blockers, err)
		}
		if fixture.Snapshot() != before {
			t.Fatal("Preflight changed paused state")
		}
	})
}

func TestSelectiveApplyExecute(t *testing.T) {
	t.Run("T115 stages the selected files without a commit", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		sources := sideFileCommits(fixture)
		fixture.Write("notes.txt", "keep\n")
		reviewedHead := head(fixture)
		steps := selectiveSteps(t, fixture, git.FileSelection{OID: sources.first, Parent: sources.base, Path: "added.txt", Status: 'A'})
		report, err := newSelectiveApply(fixture).Execute(context.Background(), steps, reviewedHead)
		if err != nil || len(report.Applied) != 1 {
			t.Fatalf("Execute = %+v, %v", report, err)
		}
		if head(fixture) != reviewedHead {
			t.Fatal("Execute created a commit")
		}
		if staged := fixture.Git("diff", "--cached", "--name-status"); staged != "A\tadded.txt\n" {
			t.Fatalf("staged = %q", staged)
		}
		if content, err := os.ReadFile(filepath.Join(fixture.Dir, "notes.txt")); err != nil || string(content) != "keep\n" {
			t.Fatalf("untracked notes.txt = %q, %v", content, err)
		}
	})
	blockedCases := []struct {
		name    string
		prepare func(fixture *gittest.Fixture)
	}{
		{name: "T170 dirty tracked file is not stashed", prepare: func(fixture *gittest.Fixture) { fixture.Dirty("tracked.txt") }},
		{name: "T117 untracked collision keeps bytes", prepare: func(fixture *gittest.Fixture) { fixture.Write("added.txt", "mine\n") }},
	}
	for _, blockedCase := range blockedCases {
		t.Run(blockedCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			sources := sideFileCommits(fixture)
			reviewedHead := head(fixture)
			steps := selectiveSteps(t, fixture, git.FileSelection{OID: sources.first, Parent: sources.base, Path: "added.txt", Status: 'A'})
			blockedCase.prepare(fixture)
			before := fixture.Snapshot()
			if _, err := newSelectiveApply(fixture).Execute(context.Background(), steps, reviewedHead); !errors.Is(err, recovery.ErrExecutionBlocked) {
				t.Fatalf("Execute = %v, want ErrExecutionBlocked", err)
			}
			if fixture.Snapshot() != before {
				t.Fatal("blocked Execute changed Git state")
			}
			if list := fixture.Git("stash", "list"); list != "" {
				t.Fatalf("blocked Execute created a stash: %q", list)
			}
		})
	}
}
