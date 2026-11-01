package git_test

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

const leakedSecret = "s3cr3t-token"

func newTestRemoteSync(fixture *gittest.Fixture) *git.RemoteSync {
	status := git.NewStatusReader(fixture.Runner)
	operations := git.NewOperationDetector(fixture.Runner)
	return git.NewRemoteSync(fixture.Runner, status, operations, git.NewBrancher(fixture.Runner, status, operations))
}

func newPublishedFixture(t *testing.T) *gittest.Fixture {
	t.Helper()
	fixture := gittest.NewFixture(t)
	fixture.WithRemote()
	fixture.Commit("base", map[string]string{"shared.txt": "base\n", "other.txt": "base\n"})
	fixture.Git("push", "-q", "-u", "origin", "main")
	return fixture
}

func wantSyncBlocked(t *testing.T, err error, blocker git.SyncBlocker, paths []string) {
	t.Helper()
	var blocked *git.SyncBlocked
	if !errors.As(err, &blocked) || blocked.Blocker != blocker || !slices.Equal(blocked.Paths, paths) {
		t.Fatalf("err = %v; want SyncBlocked %d with paths %v", err, blocker, paths)
	}
}

func TestUpstream(t *testing.T) {
	fixture := newPublishedFixture(t)
	fixture.Git("switch", "-q", "-c", "feature/nested")
	fixture.Git("push", "-q", "-u", "origin", "feature/nested")
	brancher := newTestBrancher(fixture)
	ctx := context.Background()

	upstream, err := brancher.Upstream(ctx, "main")
	want := git.Upstream{Remote: "origin", Ref: "refs/heads/main", TrackingRef: "refs/remotes/origin/main", Set: true}
	if err != nil || upstream != want {
		t.Fatalf("Upstream(main) = %+v, %v; want %+v", upstream, err, want)
	}
	prefix, err := brancher.Upstream(ctx, "feature")
	if err != nil || prefix != (git.Upstream{}) {
		t.Fatalf("Upstream(feature) = %+v, %v; want unset", prefix, err)
	}
}

func TestFetch(t *testing.T) {
	ctx := context.Background()
	t.Run("no remote configured", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", nil)
		_, err := newTestRemoteSync(fixture).Fetch(ctx)
		wantSyncBlocked(t, err, git.SyncNoRemote, nil)
	})
	t.Run("T86 fetch updates the remote-tracking ref only", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		head := refOID(fixture, "HEAD")
		remoteOID := fixture.PushFromOtherClone("main", "remote change", map[string]string{"shared.txt": "remote\n"})
		summary, err := newTestRemoteSync(fixture).Fetch(ctx)
		if err != nil || summary == "" {
			t.Fatalf("Fetch = %q, %v", summary, err)
		}
		if got := refOID(fixture, "refs/remotes/origin/main"); got != remoteOID {
			t.Fatalf("origin/main = %s; want %s", got, remoteOID)
		}
		if got := refOID(fixture, "HEAD"); got != head {
			t.Fatalf("HEAD moved to %s", got)
		}
	})
}

func TestPull(t *testing.T) {
	testCases := []struct {
		name        string
		mode        git.PullMode
		setup       func(t *testing.T, fixture *gittest.Fixture)
		wantBlocker git.SyncBlocker
		wantPaths   []string
	}{
		{
			name: "T84 no upstream",
			mode: git.PullConfigured,
			setup: func(t *testing.T, fixture *gittest.Fixture) {
				fixture.Git("switch", "-q", "-c", "feature")
			},
			wantBlocker: git.SyncNoUpstream,
		},
		{
			name: "detached HEAD",
			mode: git.PullFastForwardOnly,
			setup: func(t *testing.T, fixture *gittest.Fixture) {
				fixture.Git("switch", "-q", "--detach")
			},
			wantBlocker: git.SyncDetached,
		},
		{
			name: "tracked dirty paths",
			mode: git.PullFastForwardOnly,
			setup: func(t *testing.T, fixture *gittest.Fixture) {
				fixture.Dirty("other.txt")
			},
			wantBlocker: git.SyncDirty,
			wantPaths:   []string{"other.txt"},
		},
		{
			name: "T110 paused operation",
			mode: git.PullConfigured,
			setup: func(t *testing.T, fixture *gittest.Fixture) {
				fixture.Git("switch", "-q", "-c", "side")
				fixture.Commit("side", map[string]string{"shared.txt": "side\n"})
				fixture.Git("switch", "-q", "main")
				fixture.Commit("main", map[string]string{"shared.txt": "main\n"})
				fixture.GitFails("merge", "-q", "side")
			},
			wantBlocker: git.SyncOperation,
		},
		{
			name: "T84 unresolved divergent strategy",
			mode: git.PullConfigured,
			setup: func(t *testing.T, fixture *gittest.Fixture) {
				fixture.PushFromOtherClone("main", "remote", map[string]string{"other.txt": "remote\n"})
				fixture.Git("fetch", "-q", "origin")
				fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
			},
			wantBlocker: git.SyncUnresolvedStrategy,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newPublishedFixture(t)
			testCase.setup(t, fixture)
			before := fixture.Snapshot()
			_, err := newTestRemoteSync(fixture).Pull(context.Background(), testCase.mode)
			wantSyncBlocked(t, err, testCase.wantBlocker, testCase.wantPaths)
			if after := fixture.Snapshot(); after != before {
				t.Fatalf("blocked pull changed the repository")
			}
		})
	}
}

func TestPullRuns(t *testing.T) {
	ctx := context.Background()
	t.Run("T85 ff-only fast-forwards", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		remoteOID := fixture.PushFromOtherClone("main", "remote", map[string]string{"other.txt": "remote\n"})
		if _, err := newTestRemoteSync(fixture).Pull(ctx, git.PullFastForwardOnly); err != nil {
			t.Fatalf("Pull: %v", err)
		}
		if got := refOID(fixture, "HEAD"); got != remoteOID {
			t.Fatalf("HEAD = %s; want %s", got, remoteOID)
		}
	})
	t.Run("T85 ff-only refuses divergence natively", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.PushFromOtherClone("main", "remote", map[string]string{"other.txt": "remote\n"})
		head := fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
		_, err := newTestRemoteSync(fixture).Pull(ctx, git.PullFastForwardOnly)
		var commandErr *git.CommandError
		if !errors.As(err, &commandErr) || refOID(fixture, "HEAD") != head {
			t.Fatalf("Pull = %v, HEAD %s; want native refusal at %s", err, refOID(fixture, "HEAD"), head)
		}
	})
	t.Run("T84 configured merge strategy is honored", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("config", "pull.rebase", "false")
		remoteOID := fixture.PushFromOtherClone("main", "remote", map[string]string{"other.txt": "remote\n"})
		fixture.Git("fetch", "-q", "origin")
		head := fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
		if _, err := newTestRemoteSync(fixture).Pull(ctx, git.PullConfigured); err != nil {
			t.Fatalf("Pull: %v", err)
		}
		parents := strings.Fields(fixture.Git("rev-list", "--parents", "-n", "1", "HEAD"))
		if !slices.Equal(parents[1:], []string{head, remoteOID}) {
			t.Fatalf("HEAD parents = %v; want merge of %s and %s", parents[1:], head, remoteOID)
		}
	})
	t.Run("T85 rebase pull replays local commits on the upstream", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("config", "pull.rebase", "false")
		remoteOID := fixture.PushFromOtherClone("main", "remote", map[string]string{"other.txt": "remote\n"})
		fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
		if _, err := newTestRemoteSync(fixture).Pull(ctx, git.PullRebase); err != nil {
			t.Fatalf("Pull: %v", err)
		}
		parents := strings.Fields(fixture.Git("rev-list", "--parents", "-n", "1", "HEAD"))
		if !slices.Equal(parents[1:], []string{remoteOID}) || fixture.Git("log", "-1", "--format=%s") != "local\n" {
			t.Fatalf("HEAD parents = %v; want local replayed on %s", parents[1:], remoteOID)
		}
	})
	t.Run("T85 rebase pull conflict pauses a native rebase", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.PushFromOtherClone("main", "remote", map[string]string{"shared.txt": "remote\n"})
		fixture.Commit("local", map[string]string{"shared.txt": "local\n"})
		_, err := newTestRemoteSync(fixture).Pull(ctx, git.PullRebase)
		operation, detectErr := git.NewOperationDetector(fixture.Runner).Active(ctx)
		if err == nil || detectErr != nil || operation.Kind != git.OperationRebase {
			t.Fatalf("Pull = %v, operation %+v %v; want native rebase pause", err, operation, detectErr)
		}
	})
}

func TestPushTarget(t *testing.T) {
	ctx := context.Background()
	t.Run("T141 missing origin blocks the first push", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", nil)
		before := fixture.Snapshot()
		_, err := newTestRemoteSync(fixture).PushTarget(ctx)
		wantSyncBlocked(t, err, git.SyncNoOrigin, nil)
		if fixture.Snapshot() != before {
			t.Fatalf("blocked push target changed the repository")
		}
	})
	t.Run("T140 first push destination state follows cached refs", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.WithRemote()
		fixture.Commit("base", nil)
		fixture.Git("switch", "-q", "-c", "feature")
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		want := git.PushTarget{Branch: "feature", LocalOID: refOID(fixture, "HEAD"), Remote: "origin", RemoteRef: "refs/heads/feature",
			TrackingRef: "refs/remotes/origin/feature", FirstPush: true, State: git.DestinationUnknown}
		if err != nil || target != want {
			t.Fatalf("PushTarget = %+v, %v; want %+v", target, err, want)
		}
		fixture.Git("fetch", "-q", "origin")
		target, err = remoteSync.PushTarget(ctx)
		if err != nil || target.State != git.DestinationAbsent {
			t.Fatalf("after fetch PushTarget = %+v, %v; want absent", target, err)
		}
	})
	t.Run("T141 divergent existing destination is reported", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("switch", "-q", "-c", "feature")
		fixture.PushFromOtherClone("main", "remote", nil)
		fixture.Git("fetch", "-q", "origin")
		fixture.Git("push", "-q", "origin", "refs/remotes/origin/main:refs/heads/feature")
		fixture.Git("fetch", "-q", "origin")
		fixture.Commit("local", nil)
		target, err := newTestRemoteSync(fixture).PushTarget(ctx)
		if err != nil || !target.FirstPush || target.State != git.DestinationExists || target.Tracking != (git.AheadBehind{Ahead: 1, Behind: 1}) {
			t.Fatalf("PushTarget = %+v, %v; want divergent existing destination", target, err)
		}
		if target.CachedOID != refOID(fixture, "refs/remotes/origin/feature") {
			t.Fatalf("CachedOID = %s", target.CachedOID)
		}
	})
	t.Run("detached and unborn are blocked", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.WithRemote()
		_, err := newTestRemoteSync(fixture).PushTarget(ctx)
		wantSyncBlocked(t, err, git.SyncUnborn, nil)
		fixture.Commit("base", nil)
		fixture.Git("switch", "-q", "--detach")
		_, err = newTestRemoteSync(fixture).PushTarget(ctx)
		wantSyncBlocked(t, err, git.SyncDetached, nil)
	})
}

func TestPush(t *testing.T) {
	ctx := context.Background()
	t.Run("T140 first push sets tracking and later push honors it", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("switch", "-q", "-c", "feature")
		local := fixture.Commit("feature", nil)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		if fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature") != "" {
			t.Fatalf("remote ref exists before approval")
		}
		if _, err := remoteSync.Push(ctx, target); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if got := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature"); got != local {
			t.Fatalf("remote feature = %s; want %s", got, local)
		}
		upstream, err := newTestBrancher(fixture).Upstream(ctx, "feature")
		if err != nil || upstream.Remote != "origin" || upstream.Ref != "refs/heads/feature" {
			t.Fatalf("Upstream = %+v, %v", upstream, err)
		}
		next := fixture.Commit("feature two", nil)
		target, err = remoteSync.PushTarget(ctx)
		if err != nil || target.FirstPush {
			t.Fatalf("second PushTarget = %+v, %v", target, err)
		}
		if _, err := remoteSync.Push(ctx, target); err != nil {
			t.Fatalf("second Push: %v", err)
		}
		if got := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature"); got != next {
			t.Fatalf("remote feature = %s; want %s", got, next)
		}
		if after, _ := newTestBrancher(fixture).Upstream(ctx, "feature"); after != upstream {
			t.Fatalf("upstream changed to %+v", after)
		}
	})
	t.Run("T141 rejected first push never configures tracking", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("switch", "-q", "-c", "feature")
		fixture.PushFromOtherClone("main", "remote", nil)
		fixture.Git("fetch", "-q", "origin")
		fixture.Git("push", "-q", "origin", "refs/remotes/origin/main:refs/heads/feature")
		fixture.Git("fetch", "-q", "origin")
		fixture.Commit("local", nil)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		_, err = remoteSync.Push(ctx, target)
		var commandErr *git.CommandError
		if !errors.As(err, &commandErr) {
			t.Fatalf("Push = %v; want native rejection", err)
		}
		if upstream, _ := newTestBrancher(fixture).Upstream(ctx, "feature"); upstream.Set {
			t.Fatalf("rejected push configured %+v", upstream)
		}
	})
	t.Run("T141 failed network push hides credentials and configures nothing", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", nil)
		fixture.Git("remote", "add", "origin", "https://user:"+leakedSecret+"@127.0.0.1:1/repo.git")
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		_, err = remoteSync.Push(ctx, target)
		if err == nil || strings.Contains(err.Error(), leakedSecret) {
			t.Fatalf("Push = %v; want failure without the secret", err)
		}
		if upstream, _ := newTestBrancher(fixture).Upstream(ctx, "main"); upstream.Set {
			t.Fatalf("failed push configured %+v", upstream)
		}
	})
	t.Run("T142 non-origin upstream is honored", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		mirror := filepath.Join(t.TempDir(), "mirror.git")
		fixture.Git("init", "-q", "--bare", mirror)
		fixture.Git("remote", "add", "mirror", mirror)
		fixture.Git("switch", "-q", "-c", "feature")
		fixture.Git("push", "-q", "-u", "mirror", "feature:refs/heads/review")
		local := fixture.Commit("feature", nil)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil || target.Remote != "mirror" || target.RemoteRef != "refs/heads/review" || target.FirstPush {
			t.Fatalf("PushTarget = %+v, %v", target, err)
		}
		if _, err := remoteSync.Push(ctx, target); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if got := fixture.RemoteRefOID(mirror, "refs/heads/review"); got != local {
			t.Fatalf("mirror review = %s; want %s", got, local)
		}
		if fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature") != "" {
			t.Fatalf("push created origin/feature")
		}
		if upstream, _ := newTestBrancher(fixture).Upstream(ctx, "feature"); upstream.Remote != "mirror" {
			t.Fatalf("upstream replaced with %+v", upstream)
		}
	})
	t.Run("stale reviewed target is refused", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		fixture.Commit("after review", nil)
		if _, err := remoteSync.Push(ctx, target); !errors.Is(err, git.ErrStaleRef) {
			t.Fatalf("Push = %v; want ErrStaleRef", err)
		}
	})
}

func TestForceWithLease(t *testing.T) {
	ctx := context.Background()
	rewrite := func(t *testing.T) *gittest.Fixture {
		fixture := newPublishedFixture(t)
		fixture.Git("switch", "-q", "-c", "feature")
		fixture.Commit("feature", nil)
		fixture.Git("push", "-q", "-u", "origin", "feature")
		fixture.Git("commit", "-q", "--amend", "--allow-empty", "-m", "rewritten")
		return fixture
	}
	t.Run("T87 lease on the reviewed OID rewrites the remote", func(t *testing.T) {
		fixture := rewrite(t)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil || target.LeaseBlocked() != nil {
			t.Fatalf("PushTarget = %+v, %v, lease %v", target, err, target.LeaseBlocked())
		}
		if _, err := remoteSync.ForceWithLease(ctx, target); err != nil {
			t.Fatalf("ForceWithLease: %v", err)
		}
		if got := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature"); got != refOID(fixture, "HEAD") {
			t.Fatalf("remote feature = %s; want HEAD", got)
		}
	})
	t.Run("T87 stale lease is rejected", func(t *testing.T) {
		fixture := rewrite(t)
		remoteSync := newTestRemoteSync(fixture)
		target, err := remoteSync.PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		othersOID := fixture.PushFromOtherClone("feature", "someone else", nil)
		if _, err := remoteSync.ForceWithLease(ctx, target); err == nil {
			t.Fatalf("ForceWithLease succeeded over a stale lease")
		}
		if got := fixture.RemoteRefOID(fixture.Remote, "refs/heads/feature"); got != othersOID {
			t.Fatalf("remote feature = %s; want untouched %s", got, othersOID)
		}
	})
	t.Run("no upstream has no lease", func(t *testing.T) {
		fixture := newPublishedFixture(t)
		fixture.Git("switch", "-q", "-c", "feature")
		target, err := newTestRemoteSync(fixture).PushTarget(ctx)
		if err != nil {
			t.Fatalf("PushTarget: %v", err)
		}
		wantSyncBlocked(t, target.LeaseBlocked(), git.SyncNoUpstream, nil)
		_, err = newTestRemoteSync(fixture).ForceWithLease(ctx, target)
		wantSyncBlocked(t, err, git.SyncNoUpstream, nil)
	})
}

func TestCommandErrorRedactsCredentials(t *testing.T) {
	commandErr := &git.CommandError{Args: []string{"push"}, Stderr: "fatal: unable to access 'https://user:" + leakedSecret + "@example.invalid/x.git/'"}
	if message := commandErr.Error(); strings.Contains(message, leakedSecret) || !strings.Contains(message, "example.invalid") {
		t.Fatalf("Error() = %q", message)
	}
}
