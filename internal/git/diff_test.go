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

const tenLines = "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\n"

func findFile(t *testing.T, fixture *gittest.Fixture, path string) git.FileChange {
	t.Helper()
	status, err := git.LoadStatus(context.Background(), fixture.Runner)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	index := slices.IndexFunc(status.Files, func(file git.FileChange) bool { return file.Path == path })
	if index < 0 {
		t.Fatalf("%s not in status %+v", path, status.Files)
	}
	return status.Files[index]
}

func selectHunk(diff git.FileDiff, hunkIndex int) git.LineSelection {
	selection := git.LineSelection{}
	for lineIndex, line := range diff.Hunks[hunkIndex].Lines {
		if line.Kind == '+' || line.Kind == '-' {
			selection[git.LineRef{Hunk: hunkIndex, Line: lineIndex}] = true
		}
	}
	return selection
}

func TestParseDiffAndBuildPatch(t *testing.T) {
	raw := "diff --git a/f.txt b/f.txt\nindex 1111111..2222222 100644\n--- a/f.txt\n+++ b/f.txt\n" +
		"@@ -1,4 +1,4 @@ section\n ctx\n-old1\n-old2\n+new1\n+new2\n ctx2\n"
	diff, err := git.ParseDiff(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(diff.Hunks) != 1 || len(diff.Hunks[0].Lines) != 6 || diff.PartialUnsupportedReason() != "" {
		t.Fatalf("diff = %+v", diff)
	}
	selection := git.LineSelection{{Hunk: 0, Line: 1}: true, {Hunk: 0, Line: 3}: true}

	forward, err := diff.BuildPatch(selection, false)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if !strings.Contains(forward, "@@ -1,4 +1,4 @@ section\n ctx\n-old1\n old2\n+new1\n ctx2\n") {
		t.Fatalf("forward patch:\n%s", forward)
	}
	reverse, err := diff.BuildPatch(selection, true)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if !strings.Contains(reverse, "@@ -1,4 +1,4 @@ section\n ctx\n-old1\n+new1\n new2\n ctx2\n") {
		t.Fatalf("reverse patch:\n%s", reverse)
	}
	if _, err := diff.BuildPatch(git.LineSelection{{Hunk: 0, Line: 0}: true}, false); !errors.Is(err, git.ErrNothingSelected) {
		t.Fatalf("expected ErrNothingSelected, got %v", err)
	}
}

func TestPartialSupportReasons(t *testing.T) {
	cases := map[string]string{
		"binary":   "diff --git a/b.bin b/b.bin\nindex 1..2 100644\nBinary files a/b.bin and b/b.bin differ\n",
		"new file": "diff --git a/n.txt b/n.txt\nnew file mode 100644\nindex 0000000..1111111\n--- /dev/null\n+++ b/n.txt\n@@ -0,0 +1 @@\n+x\n",
		"rename":   "diff --git a/a.txt b/b.txt\nsimilarity index 90%\nrename from a.txt\nrename to b.txt\n",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			diff, err := git.ParseDiff(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if diff.PartialUnsupportedReason() == "" {
				t.Fatal("expected partial staging to be unsupported")
			}
		})
	}
}

func TestIndexOperationsOnRealRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("T04 stage hunk A then edit hunk B yields one mixed row", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": tenLines})
		fixture.Write("f.txt", strings.Replace(strings.Replace(tenLines, "l2\n", "L2\n", 1), "l18\n", "L18\n", 1))
		file := findFile(t, fixture, "f.txt")
		unstaged, err := git.LoadDiff(ctx, fixture.Runner, file, false)
		if err != nil || len(unstaged.Hunks) != 2 {
			t.Fatalf("unstaged = %+v err = %v", unstaged, err)
		}
		patch, err := unstaged.BuildPatch(selectHunk(unstaged, 0), false)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if err := git.ApplyToIndex(ctx, fixture.Runner, patch, false); err != nil {
			t.Fatalf("apply: %v", err)
		}
		file = findFile(t, fixture, "f.txt")
		if !file.Mixed() {
			t.Fatalf("expected mixed, got %+v", file)
		}
		staged, _ := git.LoadDiff(ctx, fixture.Runner, file, true)
		unstaged, _ = git.LoadDiff(ctx, fixture.Runner, file, false)
		if !strings.Contains(staged.String(), "+L2") || strings.Contains(staged.String(), "L18") {
			t.Fatalf("staged diff:\n%s", staged.String())
		}
		if !strings.Contains(unstaged.String(), "+L18") || strings.Contains(unstaged.String(), "L2\n") {
			t.Fatalf("unstaged diff:\n%s", unstaged.String())
		}
		if fixture.ReadFile("f.txt") != strings.Replace(strings.Replace(tenLines, "l2\n", "L2\n", 1), "l18\n", "L18\n", 1) {
			t.Fatal("worktree content changed")
		}
	})

	t.Run("line staging keeps unselected lines unstaged", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": "a\nb\nc\n"})
		fixture.Write("f.txt", "a\nfirst\nsecond\nb\nc\n")
		file := findFile(t, fixture, "f.txt")
		unstaged, _ := git.LoadDiff(ctx, fixture.Runner, file, false)
		secondLine := slices.IndexFunc(unstaged.Hunks[0].Lines, func(line git.DiffLine) bool { return line.Text == "second" })
		patch, err := unstaged.BuildPatch(git.LineSelection{{Hunk: 0, Line: secondLine}: true}, false)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if err := git.ApplyToIndex(ctx, fixture.Runner, patch, false); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if indexContent := fixture.Git("show", ":f.txt"); indexContent != "a\nsecond\nb\nc\n" {
			t.Fatalf("index content = %q", indexContent)
		}
		if fixture.ReadFile("f.txt") != "a\nfirst\nsecond\nb\nc\n" {
			t.Fatal("worktree content changed")
		}
	})

	t.Run("T06 stage remaining worktree changes of mixed file", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": tenLines})
		fixture.Write("f.txt", strings.Replace(tenLines, "l2\n", "L2\n", 1))
		fixture.Git("add", "--", "f.txt")
		final := strings.Replace(strings.Replace(tenLines, "l2\n", "L2\n", 1), "l18\n", "L18\n", 1)
		fixture.Write("f.txt", final)
		file := findFile(t, fixture, "f.txt")
		if err := git.StageFiles(ctx, fixture.Runner, []git.FileChange{file}); err != nil {
			t.Fatalf("stage: %v", err)
		}
		file = findFile(t, fixture, "f.txt")
		if !file.Staged() || file.Unstaged() {
			t.Fatalf("expected staged only, got %+v", file)
		}
		if fixture.Git("show", ":f.txt") != final || fixture.ReadFile("f.txt") != final {
			t.Fatal("index or worktree content wrong")
		}
	})

	t.Run("T07 unstage staged hunk preserves working copy", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": tenLines})
		changed := strings.Replace(strings.Replace(tenLines, "l2\n", "L2\n", 1), "l18\n", "L18\n", 1)
		fixture.Write("f.txt", changed)
		fixture.Git("add", "--", "f.txt")
		file := findFile(t, fixture, "f.txt")
		staged, _ := git.LoadDiff(ctx, fixture.Runner, file, true)
		patch, err := staged.BuildPatch(selectHunk(staged, 1), true)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if err := git.ApplyToIndex(ctx, fixture.Runner, patch, true); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if fixture.Git("show", ":f.txt") != strings.Replace(tenLines, "l2\n", "L2\n", 1) {
			t.Fatalf("index = %q", fixture.Git("show", ":f.txt"))
		}
		if fixture.ReadFile("f.txt") != changed {
			t.Fatal("worktree changed")
		}
	})

	t.Run("T26 unusual filenames stage and unstage safely", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"keep.txt": "k\n"})
		names := []string{"with space.txt", "ünïcødé.txt", "-rf.txt", "tab\tname.txt", "quote\"name.txt", "*glob?.txt"}
		for _, name := range names {
			fixture.Write(name, "x\n")
		}
		status, err := git.LoadStatus(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if len(status.Files) != len(names) {
			t.Fatalf("files = %+v", status.Files)
		}
		if err := git.StageFiles(ctx, fixture.Runner, status.Files); err != nil {
			t.Fatalf("stage: %v", err)
		}
		staged, err := git.LoadStatus(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		for _, file := range staged.Files {
			if !file.Staged() || file.Index != 'A' {
				t.Errorf("%q not staged: %+v", file.Path, file)
			}
		}
		if err := git.UnstageFiles(ctx, fixture.Runner, staged.Files, false); err != nil {
			t.Fatalf("unstage: %v", err)
		}
		unstaged, _ := git.LoadStatus(ctx, fixture.Runner)
		for _, file := range unstaged.Files {
			if file.Kind != git.KindUntracked {
				t.Errorf("%q should be untracked again: %+v", file.Path, file)
			}
		}
	})

	t.Run("unstage in unborn repository", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Write("first.txt", "1\n")
		fixture.Git("add", "--", "first.txt")
		file := findFile(t, fixture, "first.txt")
		if err := git.UnstageFiles(ctx, fixture.Runner, []git.FileChange{file}, true); err != nil {
			t.Fatalf("unstage: %v", err)
		}
		if findFile(t, fixture, "first.txt").Kind != git.KindUntracked {
			t.Fatal("expected untracked")
		}
	})

	t.Run("revalidate detects stale targets", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": "a\n"})
		fixture.Write("f.txt", "b\n")
		file := findFile(t, fixture, "f.txt")
		fixture.Git("add", "--", "f.txt")
		if _, err := git.Revalidate(context.Background(), fixture.Runner, []git.FileChange{file}); !errors.Is(err, git.ErrStale) {
			t.Fatalf("expected ErrStale, got %v", err)
		}
	})

	t.Run("discard restores tracked worktree changes and refuses untracked", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": "a\n"})
		fixture.Write("f.txt", "staged\n")
		fixture.Git("add", "--", "f.txt")
		fixture.Write("f.txt", "dirty\n")
		fixture.Write("new.txt", "n\n")
		if err := git.DiscardWorktreeChanges(ctx, fixture.Runner, []git.FileChange{findFile(t, fixture, "new.txt")}); err == nil {
			t.Fatal("expected untracked discard to be refused")
		}
		if err := git.DiscardWorktreeChanges(ctx, fixture.Runner, []git.FileChange{findFile(t, fixture, "f.txt")}); err != nil {
			t.Fatalf("discard: %v", err)
		}
		if fixture.ReadFile("f.txt") != "staged\n" || fixture.ReadFile("new.txt") != "n\n" {
			t.Fatal("discard touched the wrong content")
		}
	})
}
