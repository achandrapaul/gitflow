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

const sharedMiddle = "2\n3\n4\n5\n6\n7\n8\n9\n"

const twoHunkFile = "alpha\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\nomega\n"

func worktreeDiff(t *testing.T, fixture *gittest.Fixture) git.StagedUnstaged {
	t.Helper()
	status := snapshotStatus(t, fixture)
	if len(status.Changes) != 1 {
		t.Fatalf("changes = %+v, want one", status.Changes)
	}
	before := fixture.Snapshot()
	diff, err := git.NewDiffer(fixture.Runner).WorktreeDiff(context.Background(), status.Changes[0])
	if err != nil {
		t.Fatalf("WorktreeDiff: %v", err)
	}
	if fixture.Snapshot() != before {
		t.Fatal("reading a diff changed Git state")
	}
	return diff
}

func TestWorktreeDiff(t *testing.T) {
	testCases := []struct {
		name            string
		setup           func(fixture *gittest.Fixture)
		stagedHas       []string
		stagedLacks     []string
		unstagedHas     []string
		unstagedLacks   []string
		wantStagedEmpty bool
	}{
		{
			name: "T04 stage hunk A then edit hunk B",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("file.txt", strings.Replace(twoHunkFile, "alpha", "ALPHA", 1))
				fixture.Git("add", "--", "file.txt")
				fixture.Write("file.txt", strings.Replace(strings.Replace(twoHunkFile, "alpha", "ALPHA", 1), "omega", "OMEGA", 1))
			},
			stagedHas:     []string{"+ALPHA"},
			stagedLacks:   []string{"OMEGA"},
			unstagedHas:   []string{"+OMEGA"},
			unstagedLacks: []string{"+ALPHA"},
		},
		{
			name:            "unstaged binary file",
			setup:           func(fixture *gittest.Fixture) { fixture.Write("image.bin", "\x00\x01changed") },
			unstagedHas:     []string{"Binary files"},
			wantStagedEmpty: true,
		},
		{
			name:            "untracked text file",
			setup:           func(fixture *gittest.Fixture) { fixture.Write("new file.txt", "hello\n") },
			unstagedHas:     []string{"+hello", "new file.txt"},
			wantStagedEmpty: true,
		},
		{
			name:            "untracked binary file",
			setup:           func(fixture *gittest.Fixture) { fixture.Write("new.bin", "\x00\x02") },
			unstagedHas:     []string{"Binary files"},
			wantStagedEmpty: true,
		},
		{
			name:        "staged rename",
			setup:       func(fixture *gittest.Fixture) { fixture.Git("mv", "--", "file.txt", "moved.txt") },
			stagedHas:   []string{"rename from file.txt", "rename to moved.txt"},
			unstagedHas: []string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.Commit("base", map[string]string{"file.txt": twoHunkFile, "image.bin": "\x00\x01"})
			testCase.setup(fixture)
			diff := worktreeDiff(t, fixture)
			if testCase.wantStagedEmpty && diff.Staged != "" {
				t.Errorf("staged = %q, want empty", diff.Staged)
			}
			for _, want := range testCase.stagedHas {
				if !strings.Contains(diff.Staged, want) {
					t.Errorf("staged lacks %q:\n%s", want, diff.Staged)
				}
			}
			for _, unwanted := range testCase.stagedLacks {
				if strings.Contains(diff.Staged, unwanted) {
					t.Errorf("staged contains %q:\n%s", unwanted, diff.Staged)
				}
			}
			for _, want := range testCase.unstagedHas {
				if !strings.Contains(diff.Unstaged, want) {
					t.Errorf("unstaged lacks %q:\n%s", want, diff.Unstaged)
				}
			}
			for _, unwanted := range testCase.unstagedLacks {
				if strings.Contains(diff.Unstaged, unwanted) {
					t.Errorf("unstaged contains %q:\n%s", unwanted, diff.Unstaged)
				}
			}
		})
	}
}

func TestCommitFilesSupport(t *testing.T) {
	fixture := gittest.NewFixture(t)
	root := fixture.Commit("root", map[string]string{"text.txt": "one\n", "image.bin": "\x00\x01", "kind.txt": "file\n", "script.sh": "echo\n"})
	fixture.Write("text.txt", "two\n")
	fixture.Write("image.bin", "\x00\x02")
	fixture.Git("rm", "-q", "kind.txt")
	if err := os.Symlink("text.txt", filepath.Join(fixture.Dir, "kind.txt")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Symlink("text.txt", filepath.Join(fixture.Dir, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Chmod(filepath.Join(fixture.Dir, "script.sh"), 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	fixture.Git("add", "-A")
	fixture.Git("update-index", "--add", "--cacheinfo", "160000,"+root+",module")
	child := fixture.Commit("change", nil)

	files, err := git.NewDiffer(fixture.Runner).CommitFiles(context.Background(), child, root)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	want := []git.CommitFile{
		{Path: "image.bin", Status: 'M', UnsupportedReason: "binary file"},
		{Path: "kind.txt", Status: 'T', UnsupportedReason: "file type change"},
		{Path: "link", Status: 'A', UnsupportedReason: "symbolic link"},
		{Path: "module", Status: 'A', UnsupportedReason: "submodule change"},
		{Path: "script.sh", Status: 'M'},
		{Path: "text.txt", Status: 'M'},
	}
	if !slices.Equal(files, want) {
		t.Fatalf("files = %+v, want %+v", files, want)
	}
}

func applySteps(t *testing.T, fixture *gittest.Fixture, selections ...git.FileSelection) []git.ApplyStep {
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

func TestApplyQueue(t *testing.T) {
	fixture := gittest.NewFixture(t)
	base := fixture.Commit("base", map[string]string{"shared.txt": "top\n" + sharedMiddle + "bottom\n", "gone.txt": "gone\n", "other.txt": "other\n"})
	fixture.Git("switch", "-q", "-c", "side")
	fixture.Write("shared.txt", "top\n"+sharedMiddle+"SIDE\n")
	fixture.Write("added.txt", "added\n")
	fixture.Write("other.txt", "OTHER\n")
	fixture.Git("rm", "-q", "gone.txt")
	fixture.Git("add", "-A")
	source := fixture.Commit("side change", nil)
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main change", map[string]string{"shared.txt": "MAIN\n" + sharedMiddle + "bottom\n", "other.txt": "main other\n"})
	fixture.Write("notes.txt", "untracked\n")
	head := fixture.Git("rev-parse", "HEAD")
	differ := git.NewDiffer(fixture.Runner)
	selection := func(path string, status byte) git.FileSelection {
		return git.FileSelection{OID: source, Parent: base, Path: path, Status: status}
	}

	t.Run("T115 dry run leaves every byte unchanged", func(t *testing.T) {
		before := fixture.Snapshot()
		report, err := differ.ApplyQueue(context.Background(), applySteps(t, fixture, selection("shared.txt", 'M'), selection("added.txt", 'A'), selection("gone.txt", 'D')), true)
		if err != nil || len(report.Applied) != 3 || len(report.Failed) != 0 {
			t.Fatalf("dry run = %+v, %v", report, err)
		}
		if fixture.Snapshot() != before {
			t.Fatal("dry run changed Git state")
		}
	})

	t.Run("T117 dry run reports the first step that does not apply", func(t *testing.T) {
		before := fixture.Snapshot()
		report, err := differ.ApplyQueue(context.Background(), applySteps(t, fixture, selection("added.txt", 'A'), selection("other.txt", 'M'), selection("gone.txt", 'D')), true)
		var commandErr *git.CommandError
		if !errors.As(err, &commandErr) || len(report.Applied) != 1 || report.Failed[0].Path != "other.txt" || len(report.Skipped) != 1 || report.Skipped[0].Path != "gone.txt" {
			t.Fatalf("dry run = %+v, %v", report, err)
		}
		if fixture.Snapshot() != before {
			t.Fatal("failed dry run changed Git state")
		}
	})

	t.Run("T115 applies parent to commit patches to worktree and index", func(t *testing.T) {
		report, err := differ.ApplyQueue(context.Background(), applySteps(t, fixture, selection("shared.txt", 'M'), selection("added.txt", 'A'), selection("gone.txt", 'D')), false)
		if err != nil || len(report.Applied) != 3 {
			t.Fatalf("ApplyQueue = %+v, %v", report, err)
		}
		if fixture.Git("rev-parse", "HEAD") != head {
			t.Fatal("ApplyQueue moved HEAD")
		}
		if staged := fixture.Git("diff", "--cached", "--name-status"); staged != "A\tadded.txt\nD\tgone.txt\nM\tshared.txt\n" {
			t.Fatalf("staged = %q", staged)
		}
		if unstaged := fixture.Git("diff", "--name-only"); unstaged != "" {
			t.Fatalf("unstaged = %q, want worktree equal to index", unstaged)
		}
		if shared, _ := os.ReadFile(filepath.Join(fixture.Dir, "shared.txt")); string(shared) != "MAIN\n"+sharedMiddle+"SIDE\n" {
			t.Fatalf("shared.txt = %q, want the side patch applied over main, not the side snapshot", shared)
		}
		if notes, _ := os.ReadFile(filepath.Join(fixture.Dir, "notes.txt")); string(notes) != "untracked\n" {
			t.Fatalf("untracked notes.txt = %q", notes)
		}
	})
}
