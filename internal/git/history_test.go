package git_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func TestHistoryCursorOnMergeHeavyGraph(t *testing.T) {
	fixture := gittest.New(t)
	var fastImport strings.Builder
	mark := 0
	branchTips := map[string]int{}
	for step := range 1200 {
		branch := []string{"main", "side-a", "side-b"}[step%3]
		mark++
		fmt.Fprintf(&fastImport, "commit refs/heads/%s\nmark :%d\ncommitter F <f@example.invalid> %d +0000\ndata 6\nc%04d\n", branch, mark, 1700000000+step, step)
		if tip, ok := branchTips[branch]; ok {
			fmt.Fprintf(&fastImport, "from :%d\n", tip)
		} else if mainTip, ok := branchTips["main"]; ok {
			fmt.Fprintf(&fastImport, "from :%d\n", mainTip)
		}
		if branch == "main" && step%9 == 0 && step > 0 {
			for _, side := range []string{"side-a", "side-b"} {
				if sideTip, ok := branchTips[side]; ok {
					fmt.Fprintf(&fastImport, "merge :%d\n", sideTip)
				}
			}
		}
		fmt.Fprintf(&fastImport, "M 100644 inline %s.txt\ndata 6\nc%04d\n\n", branch, step)
		branchTips[branch] = mark
	}
	if _, err := fixture.Runner.Execute(context.Background(), git.Invocation{Args: []string{"fast-import", "--quiet"}, Stdin: []byte(fastImport.String())}); err != nil {
		t.Fatalf("fast-import: %v", err)
	}
	fixture.Git("reset", "-q", "--hard", "main")

	expected := strings.Fields(fixture.Git("rev-list", "--topo-order", "main"))
	if len(expected) < 1000 {
		t.Fatalf("fixture too small: %d", len(expected))
	}
	cursor, err := git.OpenHistory(context.Background(), fixture.Runner, "main")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cursor.Close()
	var seen []string
	unique := map[string]bool{}
	for page := 0; !cursor.Done(); page++ {
		commits, err := cursor.Next(git.HistoryPageSize)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if len(commits) > git.HistoryPageSize {
			t.Fatalf("page %d too large: %d", page, len(commits))
		}
		for _, commit := range commits {
			if unique[commit.OID] {
				t.Fatalf("duplicate %s on page %d", commit.OID, page)
			}
			unique[commit.OID] = true
			seen = append(seen, commit.OID)
		}
	}
	if strings.Join(seen, " ") != strings.Join(expected, " ") {
		t.Fatalf("paged history differs from rev-list: got %d want %d", len(seen), len(expected))
	}
}

func TestHistoryUnbornAndCommitHelpers(t *testing.T) {
	ctx := context.Background()
	fixture := gittest.New(t)
	if _, err := git.OpenHistory(ctx, fixture.Runner, "HEAD"); err == nil {
		t.Fatal("expected unborn HEAD to have no history")
	}
	fixture.Commit("first subject\n\nbody line", map[string]string{"a.txt": "a\n"})
	message, err := git.HeadMessage(ctx, fixture.Runner)
	if err != nil || !strings.HasPrefix(message, "first subject\n\nbody line") {
		t.Fatalf("message = %q err = %v", message, err)
	}
}

func TestCommitOnlyIndex(t *testing.T) {
	ctx := context.Background()

	t.Run("T09 commit staged portion of mixed file", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"f.txt": tenLines})
		stagedContent := strings.Replace(tenLines, "l2\n", "L2\n", 1)
		fixture.Write("f.txt", stagedContent)
		fixture.Git("add", "--", "f.txt")
		worktreeContent := strings.Replace(stagedContent, "l18\n", "L18\n", 1)
		fixture.Write("f.txt", worktreeContent)
		messagePath, cleanup, err := git.WriteMessageFile("feat: staged part\n")
		if err != nil {
			t.Fatalf("message file: %v", err)
		}
		defer cleanup()
		if _, err := fixture.Runner.Execute(ctx, git.CommitInvocation(messagePath, false)); err != nil {
			t.Fatalf("commit: %v", err)
		}
		if fixture.Git("show", "HEAD:f.txt") != stagedContent || fixture.ReadFile("f.txt") != worktreeContent {
			t.Fatal("commit included worktree content or worktree changed")
		}
		if file := findFile(t, fixture, "f.txt"); file.Staged() || !file.Unstaged() {
			t.Fatalf("after commit = %+v", file)
		}
	})

	t.Run("T10 rejecting hook yields error and no commit", func(t *testing.T) {
		fixture := gittest.New(t)
		head := fixture.Commit("base", map[string]string{"f.txt": "a\n"})
		hookPath := filepath.Join(fixture.Dir, ".git", "hooks", "pre-commit")
		if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'hook says no' >&2\nexit 1\n"), 0o755); err != nil {
			t.Fatalf("hook: %v", err)
		}
		fixture.Write("f.txt", "b\n")
		fixture.Git("add", "--", "f.txt")
		messagePath, cleanup, _ := git.WriteMessageFile("blocked\n")
		defer cleanup()
		_, err := fixture.Runner.Execute(ctx, git.CommitInvocation(messagePath, false))
		if err == nil || !strings.Contains(err.Error(), "hook says no") {
			t.Fatalf("expected hook failure with stderr, got %v", err)
		}
		if strings.TrimSpace(fixture.Git("rev-parse", "HEAD")) != head {
			t.Fatal("HEAD moved despite rejected hook")
		}
		if !findFile(t, fixture, "f.txt").Staged() {
			t.Fatal("staged content lost")
		}
	})

	t.Run("amend rewrites HEAD message", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("original", map[string]string{"f.txt": "a\n"})
		messagePath, cleanup, _ := git.WriteMessageFile("amended\n")
		defer cleanup()
		if _, err := fixture.Runner.Execute(ctx, git.CommitInvocation(messagePath, true)); err != nil {
			t.Fatalf("amend: %v", err)
		}
		if strings.TrimSpace(fixture.Git("log", "-1", "--format=%s")) != "amended" || strings.TrimSpace(fixture.Git("rev-list", "--count", "HEAD")) != "1" {
			t.Fatal("amend did not rewrite HEAD")
		}
	})
}
