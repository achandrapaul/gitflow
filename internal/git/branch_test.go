package git_test

import (
	"context"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func TestBranchIntelligence(t *testing.T) {
	ctx := context.Background()

	t.Run("T11 no upstream reports no counts", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		status, err := git.LoadStatus(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if status.Branch.Upstream != "" || status.Branch.HasAheadBehind {
			t.Fatalf("branch = %+v", status.Branch)
		}
	})

	t.Run("T12 upstream and base comparisons are independent", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.BareRemote("origin")
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("push", "-q", "origin", "main")
		fixture.Commit("main only 1", map[string]string{"m1.txt": "1\n"})
		fixture.Commit("main only 2", map[string]string{"m2.txt": "2\n"})
		fixture.Commit("main only 3", map[string]string{"m3.txt": "3\n"})
		fixture.Git("push", "-q", "origin", "main")
		fixture.Git("switch", "-q", "-c", "feature/auth", "HEAD~3")
		fixture.Commit("feature 1", map[string]string{"f1.txt": "1\n"})
		fixture.Git("push", "-q", "-u", "origin", "feature/auth")
		for index := range 7 {
			fixture.Commit("feature extra", map[string]string{"f2.txt": strings.Repeat("x", index+1)})
		}
		fixture.Git("fetch", "-q", "origin")

		status, err := git.LoadStatus(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if status.Branch.Upstream != "origin/feature/auth" || status.Branch.Ahead != 7 || status.Branch.Behind != 0 {
			t.Fatalf("upstream = %+v", status.Branch)
		}
		base, err := git.ResolveBase(ctx, fixture.Runner, "")
		if err != nil {
			t.Fatalf("base: %v", err)
		}
		if base.Ref != "refs/remotes/origin/main" || !base.CountsKnown || base.Ahead != 8 || base.Behind != 3 {
			t.Fatalf("base = %+v", base)
		}
	})

	t.Run("T13 detached HEAD and missing base", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Git("switch", "-q", "-c", "trunk")
		fixture.Commit("one", map[string]string{"a.txt": "a\n"})
		fixture.Commit("two", map[string]string{"a.txt": "b\n"})
		fixture.Git("switch", "-q", "--detach", "HEAD~1")
		status, err := git.LoadStatus(ctx, fixture.Runner)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if !status.Branch.Detached || status.Branch.Head != "" || status.Branch.OID == "" {
			t.Fatalf("branch = %+v", status.Branch)
		}
		base, err := git.ResolveBase(ctx, fixture.Runner, "")
		if err != nil {
			t.Fatalf("base: %v", err)
		}
		if base.Available {
			t.Fatalf("base should be unavailable, got %+v", base)
		}
	})

	t.Run("local main fallback and explicit override", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("one", map[string]string{"a.txt": "a\n"})
		fixture.Git("switch", "-q", "-c", "topic")
		fixture.Commit("two", map[string]string{"a.txt": "b\n"})
		base, err := git.ResolveBase(ctx, fixture.Runner, "")
		if err != nil || base.Ref != "refs/heads/main" || base.Ahead != 1 || base.Behind != 0 {
			t.Fatalf("base = %+v err = %v", base, err)
		}
		override, err := git.ResolveBase(ctx, fixture.Runner, "topic")
		if err != nil || override.Ref != "topic" || override.Ahead != 0 {
			t.Fatalf("override = %+v err = %v", override, err)
		}
		if _, err := git.ResolveBase(ctx, fixture.Runner, "does-not-exist"); err == nil {
			t.Fatal("expected error for unknown override")
		}
	})

	t.Run("branch create switch delete", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("one", map[string]string{"a.txt": "a\n"})
		if err := git.CreateBranch(ctx, fixture.Runner, "-evil"); err == nil {
			t.Fatal("expected leading dash branch name to be rejected")
		}
		if err := git.CreateBranch(ctx, fixture.Runner, "topic"); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := git.SwitchBranch(ctx, fixture.Runner, "topic"); err != nil {
			t.Fatalf("switch: %v", err)
		}
		branches, err := git.ListBranches(ctx, fixture.Runner)
		if err != nil || len(branches) != 2 {
			t.Fatalf("branches = %+v err = %v", branches, err)
		}
		if err := git.DeleteBranch(ctx, fixture.Runner, "topic"); err == nil {
			t.Fatal("expected deleting the checked-out branch to fail")
		}
		if err := git.DeleteBranch(ctx, fixture.Runner, "main"); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})
}
