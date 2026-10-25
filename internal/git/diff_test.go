package git_test

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

func commitPatchSteps(t *testing.T, fixture *gittest.Fixture, parent, commit string, paths ...string) []git.PatchStep {
	t.Helper()
	ctx := context.Background()
	files, err := git.ListCommitFiles(ctx, fixture.Runner, parent, commit)
	if err != nil {
		t.Fatalf("list commit files: %v", err)
	}
	var steps []git.PatchStep
	for _, path := range paths {
		index := slices.IndexFunc(files, func(file git.CommitFile) bool { return file.Path == path })
		if index < 0 {
			t.Fatalf("%s not in commit files %+v", path, files)
		}
		patch, err := git.CommitFilePatch(ctx, fixture.Runner, parent, commit, files[index])
		if err != nil {
			t.Fatalf("patch for %s: %v", path, err)
		}
		steps = append(steps, git.PatchStep{CommitOID: commit, ParentOID: parent, File: files[index], Patch: patch})
	}
	return steps
}

func applyAll(t *testing.T, fixture *gittest.Fixture, steps []git.PatchStep) {
	t.Helper()
	ctx := context.Background()
	status, err := git.LoadStatus(ctx, fixture.Runner)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if err := git.ValidateSelectiveApply(status, git.Operation{}, steps); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := git.PreflightPatches(ctx, fixture.Runner, filepath.Join(fixture.Dir, ".git"), steps); err != nil {
		t.Fatalf("preflight: %v", err)
	}
	completed, err := git.ApplyPatchSteps(ctx, fixture.Runner, steps)
	if err != nil || completed != len(steps) {
		t.Fatalf("apply: completed %d of %d: %v", completed, len(steps), err)
	}
}

func sourceBranchCommit(fixture *gittest.Fixture, message string, files map[string]string) (string, string) {
	fixture.Git("switch", "-q", "-c", "source")
	parent := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	commit := fixture.Commit(message, files)
	fixture.Git("switch", "-q", "main")
	return parent, commit
}

func TestSelectiveApply(t *testing.T) {
	ctx := context.Background()
	threeFiles := map[string]string{"a.txt": "a1\n", "b.txt": "b1\n", "c.txt": "c1\n"}

	t.Run("36 one file of a three-file commit applies alone and creates no commit", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", threeFiles)
		parent, commit := sourceBranchCommit(fixture, "three", map[string]string{"a.txt": "a2\n", "b.txt": "b2\n", "c.txt": "c2\n"})
		head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
		applyAll(t, fixture, commitPatchSteps(t, fixture, parent, commit, "a.txt"))
		if strings.TrimSpace(fixture.Git("rev-parse", "HEAD")) != head {
			t.Fatal("a commit was created")
		}
		if staged := fixture.Git("diff", "--cached", "--name-only"); staged != "a.txt\n" {
			t.Fatalf("staged = %q", staged)
		}
		if fixture.ReadFile("a.txt") != "a2\n" || fixture.ReadFile("b.txt") != "b1\n" || fixture.ReadFile("c.txt") != "c1\n" {
			t.Fatal("unselected files changed")
		}
	})

	t.Run("37 two files of one commit leave the third untouched and land staged", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", threeFiles)
		parent, commit := sourceBranchCommit(fixture, "three", map[string]string{"a.txt": "a2\n", "b.txt": "b2\n", "c.txt": "c2\n"})
		applyAll(t, fixture, commitPatchSteps(t, fixture, parent, commit, "a.txt", "b.txt"))
		if status := fixture.Git("status", "--porcelain"); status != "M  a.txt\nM  b.txt\n" {
			t.Fatalf("status = %q", status)
		}
		if fixture.ReadFile("c.txt") != "c1\n" {
			t.Fatal("third file changed")
		}
	})

	t.Run("38 conflicting patch fails closed and reports completed steps", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", threeFiles)
		parent, commit := sourceBranchCommit(fixture, "three", map[string]string{"a.txt": "a2\n", "b.txt": "b2\n"})
		fixture.Commit("main edits b", map[string]string{"b.txt": "main\n"})
		steps := commitPatchSteps(t, fixture, parent, commit, "a.txt", "b.txt")
		before := fixture.Snapshot()
		err := git.PreflightPatches(ctx, fixture.Runner, filepath.Join(fixture.Dir, ".git"), steps)
		if err == nil || !strings.Contains(err.Error(), "step 2") {
			t.Fatalf("expected preflight failure at step 2, got %v", err)
		}
		if fixture.Snapshot() != before {
			t.Fatal("preflight changed the repository")
		}
		completed, err := git.ApplyPatchSteps(ctx, fixture.Runner, steps)
		if err == nil || completed != 1 {
			t.Fatalf("completed = %d err = %v", completed, err)
		}
		if status := fixture.Git("status", "--porcelain"); status != "M  a.txt\n" || fixture.ReadFile("b.txt") != "main\n" {
			t.Fatalf("state after partial failure: %q b=%q", status, fixture.ReadFile("b.txt"))
		}
	})

	t.Run("39 same path across two commits applies in explicit order", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"a.txt": "v1\n"})
		fixture.Git("switch", "-q", "-c", "source")
		first := fixture.Commit("v2", map[string]string{"a.txt": "v2\n"})
		second := fixture.Commit("v3", map[string]string{"a.txt": "v3\n"})
		fixture.Git("switch", "-q", "main")
		firstStep := commitPatchSteps(t, fixture, strings.TrimSpace(fixture.Git("rev-parse", first+"~1")), first, "a.txt")
		secondStep := commitPatchSteps(t, fixture, first, second, "a.txt")
		reversed := append(append([]git.PatchStep{}, secondStep...), firstStep...)
		if err := git.PreflightPatches(ctx, fixture.Runner, filepath.Join(fixture.Dir, ".git"), reversed); err == nil || !strings.Contains(err.Error(), "step 1") {
			t.Fatalf("expected reversed order to fail at step 1, got %v", err)
		}
		applyAll(t, fixture, append(firstStep, secondStep...))
		if fixture.ReadFile("a.txt") != "v3\n" {
			t.Fatalf("a.txt = %q", fixture.ReadFile("a.txt"))
		}
	})

	t.Run("40 add delete rename binary mode symlink and unusual paths apply correctly", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"old.txt": "old\n", "r.txt": "rename me\nline two\nline three\n", "bin.dat": "\x00\x01\x02", "run.sh": "echo hi\n", "tc.txt": "regular\n"})
		fixture.Git("switch", "-q", "-c", "source")
		parent := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
		fixture.Write("new file ü.txt", "new\n")
		fixture.Write("bin.dat", "\x00\x09\x08\x07")
		fixture.Git("rm", "-q", "--", "old.txt")
		if err := os.Mkdir(filepath.Join(fixture.Dir, "moved"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		fixture.Git("mv", "--", "r.txt", "moved/r2.txt")
		if err := os.Chmod(filepath.Join(fixture.Dir, "run.sh"), 0o755); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		fixture.Git("add", "--", "run.sh")
		if err := os.Symlink("run.sh", filepath.Join(fixture.Dir, "link")); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		os.Remove(filepath.Join(fixture.Dir, "tc.txt"))
		if err := os.Symlink("run.sh", filepath.Join(fixture.Dir, "tc.txt")); err != nil {
			t.Fatalf("typechange symlink: %v", err)
		}
		fixture.Git("add", "--", "link", "tc.txt")
		commit := fixture.Commit("all kinds", map[string]string{"new file ü.txt": "new\n", "bin.dat": "\x00\x09\x08\x07"})
		fixture.Git("switch", "-q", "main")

		files, err := git.ListCommitFiles(ctx, fixture.Runner, parent, commit)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		statuses := map[string]string{}
		for _, file := range files {
			statuses[file.Path] = string(file.Status) + " " + file.OrigPath + " " + file.NewMode
		}
		expected := map[string]string{"new file ü.txt": "A  100644", "old.txt": "D  000000", "moved/r2.txt": "R r.txt 100644", "bin.dat": "M  100644", "run.sh": "M  100755", "link": "A  120000", "tc.txt": "T  120000"}
		for path, want := range expected {
			if statuses[path] != want {
				t.Fatalf("%s = %q, want %q (all %+v)", path, statuses[path], want, statuses)
			}
		}
		applyAll(t, fixture, commitPatchSteps(t, fixture, parent, commit, "new file ü.txt", "old.txt", "moved/r2.txt", "bin.dat", "run.sh", "link", "tc.txt"))
		if fixture.ReadFile("new file ü.txt") != "new\n" || fixture.ReadFile("bin.dat") != "\x00\x09\x08\x07" || fixture.ReadFile("moved/r2.txt") != "rename me\nline two\nline three\n" {
			t.Fatal("content not applied")
		}
		if _, err := os.Stat(filepath.Join(fixture.Dir, "old.txt")); !os.IsNotExist(err) {
			t.Fatal("old.txt still exists")
		}
		if _, err := os.Stat(filepath.Join(fixture.Dir, "r.txt")); !os.IsNotExist(err) {
			t.Fatal("rename source still exists")
		}
		if target, err := os.Readlink(filepath.Join(fixture.Dir, "link")); err != nil || target != "run.sh" {
			t.Fatalf("link = %q err = %v", target, err)
		}
		if target, err := os.Readlink(filepath.Join(fixture.Dir, "tc.txt")); err != nil || target != "run.sh" {
			t.Fatalf("typechange tc.txt = %q err = %v", target, err)
		}
		stage := fixture.Git("ls-files", "-s", "--", "run.sh", "link")
		if !strings.Contains(stage, "100755") || !strings.Contains(stage, "120000") {
			t.Fatalf("modes = %q", stage)
		}
		if worktree := fixture.Git("diff", "--name-only"); worktree != "" {
			t.Fatalf("unstaged leftovers = %q", worktree)
		}
	})

	t.Run("gitlink changes are rejected", func(t *testing.T) {
		fixture := gittest.New(t)
		base := fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("update-index", "--add", "--cacheinfo", "160000,"+base+",sub")
		commit := fixture.Commit("add gitlink", nil)
		files, err := git.ListCommitFiles(ctx, fixture.Runner, base, commit)
		if err != nil || len(files) != 1 || files[0].UnsupportedReason() == "" {
			t.Fatalf("files = %+v err = %v", files, err)
		}
		err = git.ValidateSelectiveApply(git.Status{}, git.Operation{}, []git.PatchStep{{CommitOID: commit, ParentOID: base, File: files[0]}})
		if !errors.Is(err, git.ErrUnsupportedChange) {
			t.Fatalf("expected ErrUnsupportedChange, got %v", err)
		}
	})

	t.Run("41 merge commit files depend on the explicitly chosen parent", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("switch", "-q", "-c", "topic")
		fixture.Commit("topic", map[string]string{"t.txt": "t\n"})
		fixture.Git("switch", "-q", "main")
		fixture.Commit("main", map[string]string{"m.txt": "m\n"})
		fixture.Git("merge", "-q", "--no-ff", "-m", "merge", "topic")
		merge := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
		first := strings.TrimSpace(fixture.Git("rev-parse", "HEAD^1"))
		second := strings.TrimSpace(fixture.Git("rev-parse", "HEAD^2"))
		againstFirst, err := git.ListCommitFiles(ctx, fixture.Runner, first, merge)
		if err != nil || len(againstFirst) != 1 || againstFirst[0].Path != "t.txt" {
			t.Fatalf("against first parent = %+v err = %v", againstFirst, err)
		}
		againstSecond, err := git.ListCommitFiles(ctx, fixture.Runner, second, merge)
		if err != nil || len(againstSecond) != 1 || againstSecond[0].Path != "m.txt" {
			t.Fatalf("against second parent = %+v err = %v", againstSecond, err)
		}
	})

	t.Run("root commit diffs against the empty tree", func(t *testing.T) {
		fixture := gittest.New(t)
		root := fixture.Commit("root", map[string]string{"a.txt": "a\n", "b.txt": "b\n"})
		emptyTree, err := git.EmptyTreeOID(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("empty tree: %v", err)
		}
		files, err := git.ListCommitFiles(ctx, fixture.Runner, emptyTree, root)
		if err != nil || len(files) != 2 || files[0].Status != 'A' || files[1].Status != 'A' {
			t.Fatalf("files = %+v err = %v", files, err)
		}
	})

	t.Run("prerequisites block dirty trees, operations and untracked collisions", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", threeFiles)
		parent, commit := sourceBranchCommit(fixture, "add", map[string]string{"added.txt": "x\n", "dir/inner.txt": "y\n"})
		steps := commitPatchSteps(t, fixture, parent, commit, "added.txt", "dir/inner.txt")
		check := func(want error) {
			t.Helper()
			status, err := git.LoadStatus(ctx, fixture.Runner)
			if err != nil {
				t.Fatalf("status: %v", err)
			}
			if err := git.ValidateSelectiveApply(status, git.Operation{}, steps); !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
		}
		fixture.Write("unrelated.txt", "u\n")
		if status, _ := git.LoadStatus(ctx, fixture.Runner); git.ValidateSelectiveApply(status, git.Operation{}, steps) != nil {
			t.Fatal("unrelated untracked file must be allowed")
		}
		fixture.Write("added.txt", "collide\n")
		check(git.ErrUntrackedCollision)
		os.Remove(filepath.Join(fixture.Dir, "added.txt"))
		fixture.Write("dir/other.txt", "collide dir\n")
		check(git.ErrUntrackedCollision)
		os.RemoveAll(filepath.Join(fixture.Dir, "dir"))
		fixture.Write("a.txt", "dirty\n")
		check(git.ErrDirtyTree)
		fixture.Git("add", "--", "a.txt")
		check(git.ErrDirtyTree)
		fixture.Git("reset", "-q", "--hard")
		if err := git.ValidateSelectiveApply(git.Status{}, git.Operation{Kind: git.OperationRebase}, steps); !errors.Is(err, git.ErrOperationActive) {
			t.Fatalf("expected ErrOperationActive, got %v", err)
		}
	})
}
