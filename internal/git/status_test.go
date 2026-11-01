package git_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

var allFilters = []git.StatusFilter{git.FilterAll, git.FilterStaged, git.FilterNotStaged, git.FilterMixed}

func snapshotStatus(t *testing.T, fixture *gittest.Fixture) git.Status {
	t.Helper()
	status, err := git.NewStatusReader(fixture.Runner).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	return status
}

func TestStatusFilterTruthTable(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(fixture *gittest.Fixture)
		path        string
		wantKind    git.ChangeKind
		wantFilters []git.StatusFilter
	}{
		{
			name:        "T02 modified tracked file not staged",
			setup:       func(fixture *gittest.Fixture) { fixture.Dirty("tracked.txt") },
			path:        "tracked.txt",
			wantKind:    git.KindOrdinary,
			wantFilters: []git.StatusFilter{git.FilterAll, git.FilterNotStaged},
		},
		{
			name: "T03 staged tracked file",
			setup: func(fixture *gittest.Fixture) {
				fixture.Dirty("tracked.txt")
				fixture.Git("add", "--", "tracked.txt")
			},
			path:        "tracked.txt",
			wantKind:    git.KindOrdinary,
			wantFilters: []git.StatusFilter{git.FilterAll, git.FilterStaged},
		},
		{
			name: "mixed file appears in all four filters",
			setup: func(fixture *gittest.Fixture) {
				fixture.Dirty("tracked.txt")
				fixture.Git("add", "--", "tracked.txt")
				fixture.Dirty("tracked.txt")
			},
			path:        "tracked.txt",
			wantKind:    git.KindOrdinary,
			wantFilters: allFilters,
		},
		{
			name:        "T05 untracked file inside a new folder",
			setup:       func(fixture *gittest.Fixture) { fixture.Write("fresh/nested/new.txt", "new\n") },
			path:        "fresh/nested/new.txt",
			wantKind:    git.KindUntracked,
			wantFilters: []git.StatusFilter{git.FilterAll, git.FilterNotStaged},
		},
		{
			name:        "staged deletion",
			setup:       func(fixture *gittest.Fixture) { fixture.Git("rm", "-q", "--", "tracked.txt") },
			path:        "tracked.txt",
			wantKind:    git.KindOrdinary,
			wantFilters: []git.StatusFilter{git.FilterAll, git.FilterStaged},
		},
		{
			name:        "staged rename",
			setup:       func(fixture *gittest.Fixture) { fixture.Git("mv", "--", "tracked.txt", "renamed.txt") },
			path:        "renamed.txt",
			wantKind:    git.KindRenamed,
			wantFilters: []git.StatusFilter{git.FilterAll, git.FilterStaged},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.Commit("base", map[string]string{"tracked.txt": "base\n"})
			testCase.setup(fixture)
			status := snapshotStatus(t, fixture)
			if len(status.Changes) != 1 {
				t.Fatalf("changes = %+v, want exactly one row", status.Changes)
			}
			change := status.Changes[0]
			if change.Path != testCase.path || change.Kind != testCase.wantKind {
				t.Fatalf("change = %+v, want path %q kind %v", change, testCase.path, testCase.wantKind)
			}
			for _, filter := range allFilters {
				want := false
				for _, wanted := range testCase.wantFilters {
					want = want || wanted == filter
				}
				if change.Matches(filter) != want {
					t.Errorf("filter %v: matches = %v, want %v", filter, !want, want)
				}
			}
		})
	}
}

func TestStatusRenameKeepsOriginalPath(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"old name.txt": "same content\n"})
	fixture.Git("mv", "--", "old name.txt", "new name.txt")
	status := snapshotStatus(t, fixture)
	if len(status.Changes) != 1 || status.Changes[0].OriginalPath != "old name.txt" || status.Changes[0].Path != "new name.txt" {
		t.Fatalf("changes = %+v", status.Changes)
	}
}

func TestStatusUnmergedPath(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"conflict.txt": "base\n"})
	fixture.Git("switch", "-q", "-c", "other")
	fixture.Commit("other", map[string]string{"conflict.txt": "other\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", map[string]string{"conflict.txt": "main\n"})
	if _, err := fixture.Runner.Run(context.Background(), "merge", "-q", "other"); err == nil {
		t.Fatal("merge unexpectedly succeeded")
	}
	status := snapshotStatus(t, fixture)
	if len(status.Changes) != 1 || status.Changes[0].Kind != git.KindUnmerged {
		t.Fatalf("changes = %+v", status.Changes)
	}
	if status.Changes[0].Matches(git.FilterStaged) || !status.Changes[0].Matches(git.FilterNotStaged) {
		t.Fatalf("unmerged filters wrong: %+v", status.Changes[0])
	}
}

func TestStatusUnusualPaths(t *testing.T) {
	names := []string{"trailing space ", " leading space", "unicode ñ €.txt", "line\nbreak.txt", "tab\there.txt", "-dash.txt", "quote\"d.txt"}
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"tracked dir/inner file.txt": "base\n"})
	fixture.Dirty("tracked dir/inner file.txt")
	for _, name := range names {
		fixture.Write(name, "content\n")
	}
	status := snapshotStatus(t, fixture)
	found := map[string]bool{}
	for _, change := range status.Changes {
		found[change.Path] = true
	}
	for _, name := range append(names, "tracked dir/inner file.txt") {
		if !found[name] {
			t.Errorf("missing %q in %+v", name, status.Changes)
		}
	}
	if len(status.Changes) != len(names)+1 {
		t.Fatalf("got %d rows, want %d", len(status.Changes), len(names)+1)
	}
}

func TestStatusBranchHeader(t *testing.T) {
	t.Run("T01 clean repository on main", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		head := fixture.Commit("base", nil)
		status := snapshotStatus(t, fixture)
		if status.Branch != "main" || status.Head != head || status.Detached || status.Unborn || len(status.Changes) != 0 {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("T11 no upstream reports no tracking counts", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", nil)
		status := snapshotStatus(t, fixture)
		if status.Upstream != "" || status.TrackingKnown {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("upstream counts come from Git", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.WithRemote()
		fixture.Commit("base", nil)
		fixture.Git("push", "-q", "-u", "origin", "main")
		fixture.Commit("local one", nil)
		fixture.Commit("local two", nil)
		status := snapshotStatus(t, fixture)
		if status.Upstream != "origin/main" || !status.TrackingKnown || status.Tracking != (git.AheadBehind{Ahead: 2}) {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("T13 detached HEAD", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		head := fixture.Commit("base", nil)
		fixture.Git("switch", "-q", "--detach", head)
		status := snapshotStatus(t, fixture)
		if !status.Detached || status.Branch != "" || status.Head != head {
			t.Fatalf("status = %+v", status)
		}
	})
	t.Run("unborn branch", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		status := snapshotStatus(t, fixture)
		if !status.Unborn || status.Head != "" || status.Branch != "main" {
			t.Fatalf("status = %+v", status)
		}
	})
}

func TestStatusIsReadOnly(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"tracked.txt": "base\n"})
	fixture.Dirty("tracked.txt")
	fixture.Write("untracked.txt", "new\n")
	before := fixture.Snapshot()
	indexInfo, err := os.Stat(filepath.Join(fixture.Dir, ".git", "index"))
	if err != nil {
		t.Fatalf("stat index: %v", err)
	}
	snapshotStatus(t, fixture)
	afterInfo, err := os.Stat(filepath.Join(fixture.Dir, ".git", "index"))
	if err != nil {
		t.Fatalf("stat index: %v", err)
	}
	if fixture.Snapshot() != before || !afterInfo.ModTime().Equal(indexInfo.ModTime()) {
		t.Fatal("status reading changed Git state")
	}
}

func TestStatusPathGroups(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"edited.txt": "base\n", "old.txt": "rename me\n", "conflict.txt": "base\n"})
	fixture.Write("conflict.txt", "stashed\n")
	fixture.Git("stash", "-q")
	fixture.Commit("conflicting", map[string]string{"conflict.txt": "committed\n"})
	fixture.GitFails("stash", "pop", "-q")
	fixture.Dirty("edited.txt")
	fixture.Git("mv", "--", "old.txt", "new.txt")
	fixture.Write("loose.txt", "untracked\n")

	status := snapshotStatus(t, fixture)
	if want := []string{"edited.txt", "old.txt", "new.txt"}; !slices.Equal(status.TrackedDirtyPaths, want) {
		t.Fatalf("tracked dirty = %v, want %v", status.TrackedDirtyPaths, want)
	}
	if want := []string{"conflict.txt"}; !slices.Equal(status.Unmerged, want) {
		t.Fatalf("unmerged = %v, want %v", status.Unmerged, want)
	}
	if want := []string{"loose.txt"}; !slices.Equal(status.Untracked, want) {
		t.Fatalf("untracked = %v, want %v", status.Untracked, want)
	}
	if collisions := status.UntrackedCollisions([]string{"edited.txt", "loose.txt"}); !slices.Equal(collisions, []string{"loose.txt"}) {
		t.Fatalf("collisions = %v, want [loose.txt]", collisions)
	}
}
