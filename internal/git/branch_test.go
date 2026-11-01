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

func TestResolveBase(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(fixture *gittest.Fixture)
		wantRef string
	}{
		{
			name: "T13 missing base",
			setup: func(fixture *gittest.Fixture) {
				fixture.Git("switch", "-q", "-c", "trunk")
				fixture.Git("branch", "-q", "-D", "main")
			},
			wantRef: "",
		},
		{name: "local main only", setup: func(*gittest.Fixture) {}, wantRef: "refs/heads/main"},
		{
			name: "origin/main preferred over local main",
			setup: func(fixture *gittest.Fixture) {
				fixture.WithRemote()
				fixture.Git("push", "-q", "origin", "main")
				fixture.Git("fetch", "-q", "origin")
			},
			wantRef: "refs/remotes/origin/main",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.Commit("base", nil)
			testCase.setup(fixture)
			ref, err := newTestBrancher(fixture).ResolveBase(context.Background())
			if err != nil || ref != testCase.wantRef {
				t.Fatalf("ResolveBase = %q, %v; want %q", ref, err, testCase.wantRef)
			}
		})
	}
}

func TestUpstreamAndBaseAreIndependent(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	fixture.Commit("base", nil)
	fixture.Git("push", "-q", "origin", "main")
	fixture.Git("switch", "-q", "-c", "feature")
	fixture.Commit("feature one", nil)
	fixture.Git("push", "-q", "-u", "origin", "feature")
	fixture.Commit("feature two", nil)
	fixture.Commit("feature three", nil)
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main one", nil)
	fixture.Git("push", "-q", "origin", "main")
	fixture.Git("switch", "-q", "feature")
	fixture.Git("fetch", "-q", "origin")

	ctx := context.Background()
	status := snapshotStatus(t, fixture)
	if status.Upstream != "origin/feature" || status.Tracking != (git.AheadBehind{Ahead: 2}) {
		t.Fatalf("upstream %q tracking %+v", status.Upstream, status.Tracking)
	}
	brancher := newTestBrancher(fixture)
	base, err := brancher.ResolveBase(ctx)
	if err != nil || base != "refs/remotes/origin/main" {
		t.Fatalf("ResolveBase = %q, %v", base, err)
	}
	counts, err := brancher.AheadBehind(ctx, "HEAD", base)
	if err != nil || counts != (git.AheadBehind{Ahead: 3, Behind: 1}) {
		t.Fatalf("AheadBehind = %+v, %v", counts, err)
	}
}

func TestFreshness(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	fixture.Commit("base", nil)
	brancher := newTestBrancher(fixture)
	ctx := context.Background()

	before, err := brancher.Freshness(ctx)
	if err != nil || before.Known {
		t.Fatalf("before fetch = %+v, %v; want unknown", before, err)
	}
	fixture.Git("push", "-q", "origin", "main")
	fixture.Git("fetch", "-q", "origin")
	after, err := brancher.Freshness(ctx)
	if err != nil || !after.Known || after.LastFetch.IsZero() {
		t.Fatalf("after fetch = %+v, %v; want known", after, err)
	}
}

func newTestBrancher(fixture *gittest.Fixture) *git.Brancher {
	return git.NewBrancher(fixture.Runner, git.NewStatusReader(fixture.Runner), git.NewOperationDetector(fixture.Runner))
}

func refOID(fixture *gittest.Fixture, ref string) string {
	return strings.TrimSpace(fixture.Git("rev-parse", ref))
}

func currentBranch(fixture *gittest.Fixture) string {
	return strings.TrimSpace(fixture.Git("branch", "--show-current"))
}

func TestListBranches(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	base := fixture.Commit("base", nil)
	fixture.Git("push", "-q", "-u", "origin", "main")
	fixture.Git("remote", "set-head", "origin", "main")
	fixture.Git("branch", "feature")
	brancher := newTestBrancher(fixture)

	branches, err := brancher.List(context.Background(), true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []git.BranchInfo{
		{Name: "feature", Ref: "refs/heads/feature", OID: base},
		{Name: "main", Ref: "refs/heads/main", OID: base, Current: true, Upstream: "refs/remotes/origin/main"},
		{Name: "origin/main", Ref: "refs/remotes/origin/main", OID: base, Remote: true},
	}
	if !slices.Equal(branches, want) {
		t.Fatalf("branches = %+v\nwant %+v", branches, want)
	}

	localOnly, err := brancher.List(context.Background(), false)
	if err != nil || len(localOnly) != 2 {
		t.Fatalf("local-only List = %+v, %v", localOnly, err)
	}

	fixture.Git("switch", "-q", "--detach")
	detached, err := brancher.List(context.Background(), false)
	if err != nil || slices.ContainsFunc(detached, func(branch git.BranchInfo) bool { return branch.Current }) {
		t.Fatalf("detached HEAD List = %+v, %v; no branch may be current", detached, err)
	}
}

func TestValidateBranchName(t *testing.T) {
	testCases := []struct {
		name    string
		branch  string
		wantErr error
	}{
		{name: "nested name", branch: "feature/login"},
		{name: "double dot", branch: "bad..name", wantErr: git.ErrInvalidBranchName},
		{name: "previous-branch shorthand", branch: "@{-1}", wantErr: git.ErrInvalidBranchName},
		{name: "leading dash", branch: "-x", wantErr: git.ErrInvalidBranchName},
		{name: "HEAD", branch: "HEAD", wantErr: git.ErrInvalidBranchName},
		{name: "empty", branch: "", wantErr: git.ErrInvalidBranchName},
		{name: "existing branch", branch: "main", wantErr: git.ErrBranchExists},
	}
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", nil)
	fixture.Git("switch", "-q", "-c", "previous")
	fixture.Git("switch", "-q", "main")
	brancher := newTestBrancher(fixture)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := brancher.ValidateName(context.Background(), testCase.branch)
			if testCase.wantErr == nil && err != nil || testCase.wantErr != nil && !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateName(%q) = %v, want %v", testCase.branch, err, testCase.wantErr)
			}
		})
	}
}

func TestCreateBranch(t *testing.T) {
	testCases := []struct {
		name         string
		remoteBranch bool
	}{
		{name: "T137 origin branch absent"},
		{name: "T137 origin branch present", remoteBranch: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.WithRemote()
			base := fixture.Commit("base", nil)
			fixture.Git("push", "-q", "origin", "main")
			fixture.Git("config", "branch.autoSetupMerge", "always")
			if testCase.remoteBranch {
				fixture.Git("push", "-q", "origin", "main:feature")
				fixture.Git("fetch", "-q", "origin")
			}
			remoteBefore := fixture.Git("--git-dir", fixture.Remote, "for-each-ref")

			err := newTestBrancher(fixture).Create(context.Background(), "feature", "refs/remotes/origin/main", base)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if refOID(fixture, "refs/heads/feature") != base || currentBranch(fixture) != "main" {
				t.Fatalf("feature %s, current %q; want created at %s while staying on main", refOID(fixture, "refs/heads/feature"), currentBranch(fixture), base)
			}
			if upstream := fixture.Git("for-each-ref", "--format=%(upstream)", "refs/heads/feature"); strings.TrimSpace(upstream) != "" {
				t.Fatalf("created branch has upstream %q, want none", upstream)
			}
			if remoteAfter := fixture.Git("--git-dir", fixture.Remote, "for-each-ref"); remoteAfter != remoteBefore {
				t.Fatalf("remote changed by create:\n%s\nwant\n%s", remoteAfter, remoteBefore)
			}
		})
	}
}

func TestCreateBranchBlocksInvalidState(t *testing.T) {
	fixture := gittest.NewFixture(t)
	base := fixture.Commit("base", nil)
	fixture.Git("branch", "existing")
	moved := fixture.Commit("moved", nil)
	brancher := newTestBrancher(fixture)
	ctx := context.Background()

	if err := brancher.Create(ctx, "existing", "refs/heads/main", moved); !errors.Is(err, git.ErrBranchExists) || refOID(fixture, "refs/heads/existing") != base {
		t.Fatalf("existing name = %v, ref %s; want %v and untouched ref", err, refOID(fixture, "refs/heads/existing"), git.ErrBranchExists)
	}
	if err := brancher.Create(ctx, "stale", "refs/heads/main", base); !errors.Is(err, git.ErrStaleRef) {
		t.Fatalf("stale start = %v, want %v", err, git.ErrStaleRef)
	}
	if err := brancher.Create(ctx, "missing", "refs/heads/gone", base); err == nil {
		t.Fatal("missing start ref created a branch")
	}
	if err := brancher.Create(ctx, "bad..name", "refs/heads/main", moved); !errors.Is(err, git.ErrInvalidBranchName) {
		t.Fatalf("invalid name = %v, want %v", err, git.ErrInvalidBranchName)
	}
	if refs := fixture.Git("for-each-ref", "--format=%(refname)", "refs/heads"); refs != "refs/heads/existing\nrefs/heads/main\n" {
		t.Fatalf("refs after blocked creates:\n%s", refs)
	}
}

func TestSwitchBranch(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(fixture *gittest.Fixture)
		target      string
		staleOID    bool
		wantBlocker git.SwitchBlocker
		wantPaths   []string
		wantErr     error
		wantBranch  string
	}{
		{name: "T105 clean switch", target: "feature", wantBranch: "feature"},
		{name: "dirty file unchanged between branches is carried", setup: func(fixture *gittest.Fixture) { fixture.Dirty("shared.txt") }, target: "feature", wantBranch: "feature"},
		{name: "current branch", target: "main", wantBlocker: git.BlockedCurrent, wantBranch: "main"},
		{name: "T105 remote-tracking ref is never checked out", target: "origin/main", wantBlocker: git.BlockedNotLocal, wantBranch: "main"},
		{name: "stale branch", target: "feature", staleOID: true, wantErr: git.ErrStaleRef, wantBranch: "main"},
		{name: "dirty tracked path", setup: func(fixture *gittest.Fixture) { fixture.Dirty("differs.txt") }, target: "feature", wantBlocker: git.BlockedDirty, wantPaths: []string{"differs.txt"}, wantBranch: "main"},
		{name: "staged tracked path", setup: func(fixture *gittest.Fixture) {
			fixture.Dirty("differs.txt")
			fixture.Git("add", "differs.txt")
		}, target: "feature", wantBlocker: git.BlockedDirty, wantPaths: []string{"differs.txt"}, wantBranch: "main"},
		{name: "untracked collision", setup: func(fixture *gittest.Fixture) { fixture.Write("added.txt", "local\n") }, target: "feature", wantBlocker: git.BlockedUntracked, wantPaths: []string{"added.txt"}, wantBranch: "main"},
		{name: "unmerged files", setup: func(fixture *gittest.Fixture) {
			fixture.Write("shared.txt", "stashed\n")
			fixture.Git("stash", "-q")
			fixture.Commit("conflicting", map[string]string{"shared.txt": "committed\n"})
			fixture.GitFails("stash", "pop", "-q")
		}, target: "feature", wantBlocker: git.BlockedUnmerged, wantPaths: []string{"shared.txt"}, wantBranch: "main"},
		{name: "paused operation", setup: func(fixture *gittest.Fixture) {
			fixture.Git("switch", "-q", "-c", "side")
			pick := fixture.Commit("side change", map[string]string{"shared.txt": "side\n"})
			fixture.Git("switch", "-q", "main")
			fixture.Commit("main change", map[string]string{"shared.txt": "main\n"})
			fixture.GitFails("cherry-pick", pick)
		}, target: "feature", wantBlocker: git.BlockedOperation, wantBranch: "main"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.WithRemote()
			fixture.Commit("base", map[string]string{"shared.txt": "base\n", "differs.txt": "base\n"})
			fixture.Git("push", "-q", "origin", "main")
			fixture.Git("switch", "-q", "-c", "feature")
			fixture.Commit("feature", map[string]string{"differs.txt": "feature\n", "added.txt": "feature\n"})
			fixture.Git("switch", "-q", "main")
			if testCase.setup != nil {
				testCase.setup(fixture)
			}
			reviewed := refOID(fixture, "refs/heads/main")
			if strings.Contains(testCase.target, "/") {
				reviewed = refOID(fixture, "refs/remotes/"+testCase.target)
			} else if testCase.target != "main" {
				reviewed = refOID(fixture, "refs/heads/"+testCase.target)
			}
			if testCase.staleOID {
				reviewed = refOID(fixture, "refs/heads/main")
			}

			err := newTestBrancher(fixture).Switch(context.Background(), testCase.target, reviewed)
			var blocked *git.SwitchBlocked
			switch {
			case testCase.wantErr != nil:
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("Switch = %v, want %v", err, testCase.wantErr)
				}
			case testCase.wantBlocker != 0:
				if !errors.As(err, &blocked) || blocked.Blocker != testCase.wantBlocker || !slices.Equal(blocked.Paths, testCase.wantPaths) {
					t.Fatalf("Switch = %v (%+v), want blocker %d paths %v", err, blocked, testCase.wantBlocker, testCase.wantPaths)
				}
			case err != nil:
				t.Fatalf("Switch: %v", err)
			}
			if got := currentBranch(fixture); got != testCase.wantBranch {
				t.Fatalf("current branch = %q, want %q", got, testCase.wantBranch)
			}
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	base := fixture.Commit("base", nil)
	fixture.Git("push", "-q", "origin", "main")
	fixture.Git("fetch", "-q", "origin")
	fixture.Git("branch", "merged")
	fixture.Git("switch", "-q", "-c", "unmerged")
	unmerged := fixture.Commit("unmerged work", nil)
	fixture.Git("switch", "-q", "main")
	brancher := newTestBrancher(fixture)
	ctx := context.Background()

	if err := brancher.Delete(ctx, "main", base); !errors.Is(err, git.ErrCurrentBranch) {
		t.Fatalf("delete current = %v, want %v", err, git.ErrCurrentBranch)
	}
	if err := brancher.Delete(ctx, "unmerged", unmerged); err == nil || refOID(fixture, "refs/heads/unmerged") != unmerged {
		t.Fatalf("delete unmerged = %v; native -d must refuse and keep the branch", err)
	}
	if err := brancher.Delete(ctx, "merged", unmerged); !errors.Is(err, git.ErrStaleRef) {
		t.Fatalf("delete stale = %v, want %v", err, git.ErrStaleRef)
	}
	if err := brancher.Delete(ctx, "origin/main", base); err == nil || refOID(fixture, "refs/remotes/origin/main") != base {
		t.Fatalf("delete remote-tracking ref = %v; must be refused", err)
	}
	if err := brancher.Delete(ctx, "merged", base); err != nil {
		t.Fatalf("delete merged: %v", err)
	}
	if refs := fixture.Git("for-each-ref", "--format=%(refname)", "refs/heads"); refs != "refs/heads/main\nrefs/heads/unmerged\n" {
		t.Fatalf("refs after delete:\n%s", refs)
	}
}
