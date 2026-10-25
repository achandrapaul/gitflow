package git_test

import (
	"context"
	"errors"
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

func TestRemoteSync(t *testing.T) {
	ctx := context.Background()
	trackedFixture := func(t *testing.T) *gittest.Fixture {
		fixture := gittest.New(t)
		fixture.BareRemote("origin")
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		fixture.Git("push", "-q", "-u", "origin", "main")
		return fixture
	}
	invocation := func(t *testing.T, fixture *gittest.Fixture, kind git.SyncKind) git.Invocation {
		t.Helper()
		upstream, err := git.LoadUpstream(ctx, fixture.Runner, "main")
		if err != nil {
			t.Fatalf("load upstream: %v", err)
		}
		built, err := git.SyncInvocation(kind, upstream)
		if err != nil {
			t.Fatalf("build %v invocation: %v", kind, err)
		}
		return built
	}
	remoteMain := func(fixture *gittest.Fixture) string {
		return strings.Fields(fixture.Git("ls-remote", "origin", "refs/heads/main"))[0]
	}

	t.Run("upstream is read from tracking configuration", func(t *testing.T) {
		fixture := trackedFixture(t)
		upstream, err := git.LoadUpstream(ctx, fixture.Runner, "main")
		if err != nil || upstream.Remote != "origin" || upstream.RemoteRef != "refs/heads/main" {
			t.Fatalf("upstream = %+v err = %v", upstream, err)
		}
		pull := invocation(t, fixture, git.SyncPull)
		push := invocation(t, fixture, git.SyncPush)
		fetch := invocation(t, fixture, git.SyncFetch)
		if strings.Join(pull.Args, " ") != "pull --ff-only" || strings.Join(push.Args, " ") != "push origin HEAD:refs/heads/main" || strings.Join(fetch.Args, " ") != "fetch origin" {
			t.Fatalf("pull %v push %v fetch %v", pull.Args, push.Args, fetch.Args)
		}
	})

	t.Run("31 fast-forward-only pull advances to the upstream", func(t *testing.T) {
		fixture := trackedFixture(t)
		advanced := fixture.Commit("upstream advance", map[string]string{"b.txt": "b\n"})
		fixture.Git("push", "-q", "origin", "main")
		fixture.Git("reset", "-q", "--hard", "HEAD~1")
		if _, err := fixture.Runner.Execute(ctx, invocation(t, fixture, git.SyncPull)); err != nil {
			t.Fatalf("pull: %v", err)
		}
		if head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); head != advanced {
			t.Fatalf("HEAD = %s, want %s", head, advanced)
		}
	})

	t.Run("32 diverged pull is refused without merge or rebase", func(t *testing.T) {
		fixture := trackedFixture(t)
		fixture.Commit("upstream advance", map[string]string{"b.txt": "b\n"})
		fixture.Git("push", "-q", "origin", "main")
		fixture.Git("reset", "-q", "--hard", "HEAD~1")
		local := fixture.Commit("local work", map[string]string{"c.txt": "c\n"})
		workingState := func() string {
			return fixture.Git("status", "--porcelain=v2", "-z", "--untracked-files=all") + fixture.Git("ls-files", "--stage", "-z") + fixture.Git("for-each-ref", "refs/heads")
		}
		before := workingState()
		if _, err := fixture.Runner.Execute(ctx, invocation(t, fixture, git.SyncPull)); err == nil {
			t.Fatal("expected diverged ff-only pull to fail")
		}
		if head := strings.TrimSpace(fixture.Git("rev-parse", "HEAD")); head != local {
			t.Fatalf("HEAD moved to %s", head)
		}
		if workingState() != before {
			t.Fatal("refused pull changed the working state")
		}
	})

	t.Run("33 fast-forward push advances the remote ref", func(t *testing.T) {
		fixture := trackedFixture(t)
		local := fixture.Commit("local work", map[string]string{"c.txt": "c\n"})
		if _, err := fixture.Runner.Execute(ctx, invocation(t, fixture, git.SyncPush)); err != nil {
			t.Fatalf("push: %v", err)
		}
		if remote := remoteMain(fixture); remote != local {
			t.Fatalf("remote = %s, want %s", remote, local)
		}
	})

	t.Run("34 non-fast-forward push is rejected and never forced", func(t *testing.T) {
		fixture := trackedFixture(t)
		advanced := fixture.Commit("upstream advance", map[string]string{"b.txt": "b\n"})
		fixture.Git("push", "-q", "origin", "main")
		fixture.Git("reset", "-q", "--hard", "HEAD~1")
		fixture.Commit("local work", map[string]string{"c.txt": "c\n"})
		push := invocation(t, fixture, git.SyncPush)
		for _, argument := range push.Args {
			if strings.HasPrefix(argument, "-") || strings.HasPrefix(argument, "+") {
				t.Fatalf("push must not force: %v", push.Args)
			}
		}
		if _, err := fixture.Runner.Execute(ctx, push); err == nil {
			t.Fatal("expected non-fast-forward push to be rejected")
		}
		if remote := remoteMain(fixture); remote != advanced {
			t.Fatalf("remote moved to %s", remote)
		}
	})

	t.Run("35 no upstream blocks pull and push before Git runs", func(t *testing.T) {
		fixture := gittest.New(t)
		fixture.Commit("base", map[string]string{"a.txt": "a\n"})
		upstream, err := git.LoadUpstream(ctx, fixture.Runner, "main")
		if err != nil || upstream != (git.Upstream{}) {
			t.Fatalf("upstream = %+v err = %v", upstream, err)
		}
		for _, kind := range []git.SyncKind{git.SyncPull, git.SyncPush} {
			if _, err := git.SyncInvocation(kind, upstream); !errors.Is(err, git.ErrNoUpstream) {
				t.Fatalf("%v: expected ErrNoUpstream, got %v", kind, err)
			}
		}
		fetch, err := git.SyncInvocation(git.SyncFetch, upstream)
		if err != nil || strings.Join(fetch.Args, " ") != "fetch" {
			t.Fatalf("fetch = %v err = %v", fetch.Args, err)
		}
	})
}
