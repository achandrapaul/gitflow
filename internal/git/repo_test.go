package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
)

func TestDiscover(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locate git: %v", err)
	}
	testCases := []struct {
		name      string
		repoName  string
		startPath string
	}{
		{name: "plain path", repoName: "repo", startPath: "."},
		{name: "path ending in a space", repoName: "repo ", startPath: "."},
		{name: "path with leading space and unicode", repoName: " répo 文件", startPath: "."},
		{name: "nested start directory", repoName: "repo", startPath: "nested/deeper"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatalf("resolve temp dir: %v", err)
			}
			workTree := filepath.Join(root, testCase.repoName)
			startDir := filepath.Join(workTree, testCase.startPath)
			if err := os.MkdirAll(startDir, 0o755); err != nil {
				t.Fatalf("create start dir: %v", err)
			}
			if output, err := exec.Command(gitPath, "init", "-q", workTree).CombinedOutput(); err != nil {
				t.Fatalf("git init: %v: %s", err, output)
			}

			repository, err := git.Discover(context.Background(), git.NewRunner(gitPath, startDir, nil))
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			want := git.Repository{WorkTree: workTree, GitDir: filepath.Join(workTree, ".git")}
			if repository != want {
				t.Fatalf("repository = %#v, want %#v", repository, want)
			}
		})
	}

	t.Run("outside a work tree", func(t *testing.T) {
		_, err := git.Discover(context.Background(), git.NewRunner(gitPath, t.TempDir(), []string{"GIT_CEILING_DIRECTORIES=" + os.TempDir()}))
		if !errors.Is(err, git.ErrNotWorkTree) {
			t.Fatalf("error = %v, want ErrNotWorkTree", err)
		}
	})
}
