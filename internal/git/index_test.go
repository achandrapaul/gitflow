package git_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func TestIndexerStageUnstage(t *testing.T) {
	type wantCodes struct {
		staged   byte
		unstaged byte
	}
	testCases := []struct {
		name      string
		unborn    bool
		setup     func(fixture *gittest.Fixture)
		unstage   bool
		paths     []string
		want      map[string]wantCodes
		worktree  map[string]string
		untracked []string
	}{
		{
			name: "T06 stage mixed file keeps worktree content",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("file.txt", strings.Replace(twoHunkFile, "alpha", "ALPHA", 1))
				fixture.Git("add", "--", "file.txt")
				fixture.Write("file.txt", strings.Replace(strings.Replace(twoHunkFile, "alpha", "ALPHA", 1), "omega", "OMEGA", 1))
			},
			paths:    []string{"file.txt"},
			want:     map[string]wantCodes{"file.txt": {'M', '.'}},
			worktree: map[string]string{"file.txt": strings.Replace(strings.Replace(twoHunkFile, "alpha", "ALPHA", 1), "omega", "OMEGA", 1)},
		},
		{
			name:     "stage untracked file",
			setup:    func(fixture *gittest.Fixture) { fixture.Write("new.txt", "new\n") },
			paths:    []string{"new.txt"},
			want:     map[string]wantCodes{"new.txt": {'A', '.'}},
			worktree: map[string]string{"new.txt": "new\n"},
		},
		{
			name: "stage deleted file",
			setup: func(fixture *gittest.Fixture) {
				if err := os.Remove(filepath.Join(fixture.Dir, "file.txt")); err != nil {
					t.Fatalf("remove file.txt: %v", err)
				}
			},
			paths: []string{"file.txt"},
			want:  map[string]wantCodes{"file.txt": {'D', '.'}},
		},
		{
			name: "stage only the named paths with glob and space characters",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("star*.txt", "glob\n")
				fixture.Write("starry.txt", "other\n")
				fixture.Write("with space.txt", "space\n")
			},
			paths:     []string{"star*.txt", "with space.txt"},
			want:      map[string]wantCodes{"star*.txt": {'A', '.'}, "with space.txt": {'A', '.'}},
			untracked: []string{"starry.txt"},
		},
		{
			name: "unstage mixed file keeps worktree content",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("file.txt", strings.Replace(twoHunkFile, "alpha", "ALPHA", 1))
				fixture.Git("add", "--", "file.txt")
				fixture.Write("file.txt", strings.Replace(strings.Replace(twoHunkFile, "alpha", "ALPHA", 1), "omega", "OMEGA", 1))
			},
			unstage:  true,
			paths:    []string{"file.txt"},
			want:     map[string]wantCodes{"file.txt": {'.', 'M'}},
			worktree: map[string]string{"file.txt": strings.Replace(strings.Replace(twoHunkFile, "alpha", "ALPHA", 1), "omega", "OMEGA", 1)},
		},
		{
			name:      "unstage added file leaves it untracked",
			setup:     func(fixture *gittest.Fixture) { fixture.Write("new.txt", "new\n"); fixture.Git("add", "--", "new.txt") },
			unstage:   true,
			paths:     []string{"new.txt"},
			worktree:  map[string]string{"new.txt": "new\n"},
			untracked: []string{"new.txt"},
		},
		{
			name:      "unstage staged rename restores both paths",
			setup:     func(fixture *gittest.Fixture) { fixture.Git("mv", "file.txt", "moved.txt") },
			unstage:   true,
			paths:     []string{"moved.txt", "file.txt"},
			want:      map[string]wantCodes{"file.txt": {'.', 'D'}},
			untracked: []string{"moved.txt"},
		},
		{
			name:   "unstage on unborn HEAD keeps worktree content",
			unborn: true,
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("first.txt", "staged\n")
				fixture.Git("add", "--", "first.txt")
				fixture.Write("first.txt", "staged\nthen edited\n")
			},
			unstage:   true,
			paths:     []string{"first.txt"},
			worktree:  map[string]string{"first.txt": "staged\nthen edited\n"},
			untracked: []string{"first.txt"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			if !testCase.unborn {
				fixture.Commit("base", map[string]string{"file.txt": twoHunkFile})
			}
			testCase.setup(fixture)
			indexer := git.NewIndexer(fixture.Runner)
			action := indexer.Stage
			if testCase.unstage {
				action = indexer.Unstage
			}
			if err := action(context.Background(), testCase.paths); err != nil {
				t.Fatalf("index action on %v: %v", testCase.paths, err)
			}
			status := snapshotStatus(t, fixture)
			var untracked []string
			for _, change := range status.Changes {
				if change.Kind == git.KindUntracked {
					untracked = append(untracked, change.Path)
					continue
				}
				want, expected := testCase.want[change.Path]
				if !expected {
					t.Errorf("unexpected change %+v", change)
					continue
				}
				if change.Staged != want.staged || change.Unstaged != want.unstaged {
					t.Errorf("%s codes = %c%c, want %c%c", change.Path, change.Staged, change.Unstaged, want.staged, want.unstaged)
				}
			}
			if strings.Join(untracked, ",") != strings.Join(testCase.untracked, ",") {
				t.Errorf("untracked = %v, want %v", untracked, testCase.untracked)
			}
			for path, content := range testCase.worktree {
				onDisk, err := os.ReadFile(filepath.Join(fixture.Dir, path))
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				if string(onDisk) != content {
					t.Errorf("%s worktree = %q, want %q", path, onDisk, content)
				}
			}
		})
	}
}

func TestIndexerRejectsUnknownPath(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"file.txt": twoHunkFile})
	before := fixture.Snapshot()
	indexer := git.NewIndexer(fixture.Runner)
	for name, action := range map[string]func(context.Context, []string) error{"stage": indexer.Stage, "unstage": indexer.Unstage} {
		if err := action(context.Background(), []string{"missing*.txt"}); err == nil || !strings.Contains(err.Error(), "missing*.txt") {
			t.Errorf("%s missing path error = %v, want an error naming the path", name, err)
		}
	}
	if fixture.Snapshot() != before {
		t.Error("a rejected index action changed Git state")
	}
}

func reviewCommit(t *testing.T, fixture *gittest.Fixture, indexer *git.Indexer, message string) git.CommitRequest {
	t.Helper()
	tree, err := indexer.IndexTree(context.Background())
	if err != nil {
		t.Fatalf("IndexTree: %v", err)
	}
	head, err := fixture.Runner.Run(context.Background(), "rev-parse", "--verify", "-q", "HEAD")
	if err != nil {
		head = nil
	}
	return git.CommitRequest{Message: message, ReviewedTree: tree, ReviewedHead: strings.TrimSpace(string(head))}
}

func TestIndexerCommit(t *testing.T) {
	const (
		baseMessage  = "base"
		draftMessage = "Add feature\n\nBody line\n"
	)
	mixedContent := strings.Replace(twoHunkFile, "alpha", "ALPHA", 1)
	mixedWorktree := strings.Replace(mixedContent, "omega", "OMEGA", 1)
	testCases := []struct {
		name        string
		unborn      bool
		setup       func(fixture *gittest.Fixture)
		afterReview func(fixture *gittest.Fixture)
		noVerify    bool
		amend       bool
		wantErr     error
		wantFailure bool
		check       func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult)
	}{
		{
			name: "T09 commits only the staged part of a mixed file",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("file.txt", mixedContent)
				fixture.Git("add", "--", "file.txt")
				fixture.Write("file.txt", mixedWorktree)
			},
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if committed := fixture.Git("show", "HEAD:file.txt"); committed != mixedContent {
					t.Errorf("committed file = %q, want staged content", committed)
				}
				if onDisk, _ := os.ReadFile(filepath.Join(fixture.Dir, "file.txt")); string(onDisk) != mixedWorktree {
					t.Errorf("worktree lost its unstaged change: %q", onDisk)
				}
				if !result.MatchesReview() || result.Message != draftMessage {
					t.Errorf("result = %+v, want match with draft message", result)
				}
			},
		},
		{
			name:   "first commit on unborn HEAD",
			unborn: true,
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("first.txt", "first\n")
				fixture.Git("add", "--", "first.txt")
			},
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if !result.MatchesReview() || fixture.Git("rev-list", "--count", "HEAD") != "1\n" {
					t.Errorf("result = %+v, want one matching root commit", result)
				}
			},
		},
		{
			name:  "T103 index staged after review is refused",
			setup: func(fixture *gittest.Fixture) { fixture.Write("new.txt", "new\n"); fixture.Git("add", "--", "new.txt") },
			afterReview: func(fixture *gittest.Fixture) {
				fixture.Write("late.txt", "late\n")
				fixture.Git("add", "--", "late.txt")
			},
			wantErr: git.ErrStaleReview,
		},
		{
			name:        "T103 HEAD moved after review is refused",
			setup:       func(fixture *gittest.Fixture) { fixture.Write("new.txt", "new\n"); fixture.Git("add", "--", "new.txt") },
			afterReview: func(fixture *gittest.Fixture) { fixture.Git("commit", "-q", "--allow-empty", "-m", "elsewhere") },
			wantErr:     git.ErrStaleReview,
		},
		{
			name: "T10 T121 pre-commit hook changes index then fails",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("pre-commit", "echo hooked > hooked.txt\ngit add hooked.txt\nexit 1\n")
			},
			wantFailure: true,
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); head != base {
					t.Errorf("HEAD = %s, want unchanged %s", head, base)
				}
				if staged := fixture.Git("diff", "--cached", "--name-only"); staged != "hooked.txt\nnew.txt\n" {
					t.Errorf("staged = %q, want hook-modified index kept", staged)
				}
			},
		},
		{
			name: "T122 pre-commit hook stages more and succeeds",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("pre-commit", "echo hooked > hooked.txt\ngit add hooked.txt\n")
			},
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if result.MatchesReview() || strings.Join(result.TreeChanges, ",") != "A\thooked.txt" {
					t.Errorf("tree changes = %q, want the hook's addition disclosed", result.TreeChanges)
				}
			},
		},
		{
			name: "T123 commit-msg hook rewrites message",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("commit-msg", "echo 'Signed-off-by: Hook' >> \"$1\"\n")
			},
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if !result.MessageChanged || !strings.Contains(result.Message, "Signed-off-by: Hook") || len(result.TreeChanges) != 0 {
					t.Errorf("result = %+v, want the actual rewritten message disclosed", result)
				}
			},
		},
		{
			name: "T124 normal commit runs pre-commit and commit-msg hooks",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("pre-commit", "exit 1\n")
			},
			wantFailure: true,
		},
		{
			name: "T124 no-verify skips pre-commit and commit-msg but runs prepare-commit-msg",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("pre-commit", "exit 1\n")
				fixture.Hook("commit-msg", "exit 1\n")
				fixture.Hook("prepare-commit-msg", "touch .git/prepared\n")
			},
			noVerify: true,
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", "prepared")); err != nil {
					t.Errorf("prepare-commit-msg did not run: %v", err)
				}
				if !result.MatchesReview() {
					t.Errorf("result = %+v, want match", result)
				}
			},
		},
		{
			name: "T126 no-verify still honours signing failure",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Git("config", "commit.gpgsign", "true")
				fixture.Git("config", "gpg.program", "false")
			},
			noVerify:    true,
			wantFailure: true,
		},
		{
			name: "T132 hook moves HEAD to an unrelated commit",
			setup: func(fixture *gittest.Fixture) {
				fixture.Write("new.txt", "new\n")
				fixture.Git("add", "--", "new.txt")
				fixture.Hook("post-commit", "[ -f .git/moved ] && exit 0\ntouch .git/moved\ngit commit -q --allow-empty --no-verify -m unrelated\n")
			},
			wantErr: git.ErrUnexpectedHead,
		},
		{
			name:  "T129 amend replaces HEAD with reviewed index and draft",
			setup: func(fixture *gittest.Fixture) { fixture.Write("new.txt", "new\n"); fixture.Git("add", "--", "new.txt") },
			amend: true,
			check: func(t *testing.T, fixture *gittest.Fixture, base string, result git.CommitResult) {
				if head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); head == base {
					t.Error("amend left HEAD unchanged")
				}
				if fixture.Git("rev-list", "--count", "HEAD") != "1\n" || !result.MatchesReview() {
					t.Errorf("result = %+v, want a single amended commit matching review", result)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			var base string
			if !testCase.unborn {
				base = fixture.Commit(baseMessage, map[string]string{"file.txt": twoHunkFile})
			}
			testCase.setup(fixture)
			indexer := git.NewIndexer(fixture.Runner)
			request := reviewCommit(t, fixture, indexer, draftMessage)
			request.NoVerify, request.Amend = testCase.noVerify, testCase.amend
			if testCase.afterReview != nil {
				testCase.afterReview(fixture)
			}
			before := fixture.Snapshot()
			result, err := indexer.Commit(context.Background(), request)
			switch {
			case testCase.wantErr != nil:
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("Commit error = %v, want %v", err, testCase.wantErr)
				}
				if errors.Is(testCase.wantErr, git.ErrStaleReview) && fixture.Snapshot() != before {
					t.Error("a stale review changed Git state")
				}
			case testCase.wantFailure:
				var commandErr *git.CommandError
				if !errors.As(err, &commandErr) {
					t.Fatalf("Commit error = %v, want the Git failure", err)
				}
			case err != nil:
				t.Fatalf("Commit: %v", err)
			default:
				if head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); result.OID != head {
					t.Errorf("result OID = %s, want HEAD %s", result.OID, head)
				}
			}
			if testCase.check != nil {
				testCase.check(t, fixture, base, result)
			}
		})
	}
}

func TestIndexerCommitRetainsHookChangedIndexInReview(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"file.txt": twoHunkFile})
	fixture.Write("new.txt", "new\n")
	fixture.Git("add", "--", "new.txt")
	fixture.Hook("pre-commit", "echo hooked > hooked.txt\ngit add hooked.txt\nexit 1\n")
	indexer := git.NewIndexer(fixture.Runner)
	request := reviewCommit(t, fixture, indexer, "draft\n")
	if _, err := indexer.Commit(context.Background(), request); err == nil {
		t.Fatal("Commit succeeded, want hook failure")
	}
	if _, err := indexer.Commit(context.Background(), request); !errors.Is(err, git.ErrStaleReview) {
		t.Fatalf("retry with old review = %v, want %v", err, git.ErrStaleReview)
	}
}

func TestBrancherHeadPublished(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	fixture.Commit("base", map[string]string{"file.txt": twoHunkFile})
	brancher := newTestBrancher(fixture)
	published, err := brancher.HeadPublished(context.Background())
	if err != nil || published {
		t.Fatalf("local-only HEAD published = %v, %v", published, err)
	}
	fixture.Git("push", "-q", "origin", "main")
	published, err = brancher.HeadPublished(context.Background())
	if err != nil || !published {
		t.Fatalf("pushed HEAD published = %v, %v", published, err)
	}
}
