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

const conflictPath = "conflict.txt"

func loadConflict(t *testing.T, fixture *gittest.Fixture, path string) git.ConflictFile {
	t.Helper()
	file, err := git.NewConflictReader(fixture.Runner).Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return file
}

func activeOperation(t *testing.T, fixture *gittest.Fixture) git.Operation {
	t.Helper()
	operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
	if err != nil {
		t.Fatalf("detect operation: %v", err)
	}
	return operation
}

func TestConflictSides(t *testing.T) {
	testCases := []struct {
		name         string
		kind         git.OperationKind
		detach       bool
		wantOurs     func(*gittest.Fixture) git.Label
		wantTheirs   func(*gittest.Fixture) git.Label
		wantOursLn   string
		wantTheirsLn string
	}{
		{
			name:         "T118 merge",
			kind:         git.OperationMerge,
			wantOurs:     func(*gittest.Fixture) git.Label { return git.Label{Ref: "main", Role: git.RoleCurrent} },
			wantTheirs:   func(*gittest.Fixture) git.Label { return git.Label{Ref: gittest.FeatureBranch, Role: git.RoleIncoming} },
			wantOursLn:   "MAIN\n",
			wantTheirsLn: "FEATURE\n",
		},
		{
			name:     "T119 rebase reverses sides",
			kind:     git.OperationRebase,
			wantOurs: func(*gittest.Fixture) git.Label { return git.Label{Ref: "main", Role: git.RoleRebaseDestination} },
			wantTheirs: func(fixture *gittest.Fixture) git.Label {
				return git.Label{Ref: revParse(fixture, gittest.FeatureBranch+"~1"), Role: git.RoleReplayed}
			},
			wantOursLn:   "MAIN\n",
			wantTheirsLn: "FEATURE\n",
		},
		{
			name:     "cherry-pick on branch",
			kind:     git.OperationCherryPick,
			wantOurs: func(*gittest.Fixture) git.Label { return git.Label{Ref: "main", Role: git.RoleCurrent} },
			wantTheirs: func(fixture *gittest.Fixture) git.Label {
				return git.Label{Ref: revParse(fixture, gittest.FeatureBranch+"~1"), Role: git.RoleReplayed}
			},
			wantOursLn:   "MAIN\n",
			wantTheirsLn: "FEATURE\n",
		},
		{
			name:   "T120 detached cherry-pick",
			kind:   git.OperationCherryPick,
			detach: true,
			wantOurs: func(fixture *gittest.Fixture) git.Label {
				return git.Label{Ref: revParse(fixture, "main"), Role: git.RoleCurrent}
			},
			wantTheirs: func(fixture *gittest.Fixture) git.Label {
				return git.Label{Ref: revParse(fixture, gittest.FeatureBranch+"~1"), Role: git.RoleReplayed}
			},
			wantOursLn:   "MAIN\n",
			wantTheirsLn: "FEATURE\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.ConflictingBranches(gittest.ConflictBase, gittest.ConflictMain, gittest.FeatureOne, gittest.FeatureTwo)
			switch testCase.kind {
			case git.OperationMerge:
				fixture.GitFails("merge", gittest.FeatureBranch)
			case git.OperationRebase:
				fixture.Git("switch", "-q", gittest.FeatureBranch)
				fixture.GitFails("rebase", "main")
			case git.OperationCherryPick:
				if testCase.detach {
					fixture.Git("switch", "-q", "--detach", "main")
				}
				fixture.GitFails("cherry-pick", gittest.FeatureBranch+"~1")
			}
			before := fixture.Snapshot()
			labels, err := git.NewConflictReader(fixture.Runner).Sides(context.Background(), activeOperation(t, fixture))
			if err != nil {
				t.Fatalf("O10 Sides: %v", err)
			}
			if labels.Ours != testCase.wantOurs(fixture) || labels.Theirs != testCase.wantTheirs(fixture) || labels.Base.Role != git.RoleBase {
				t.Fatalf("O10 labels = %+v", labels)
			}
			file := loadConflict(t, fixture, conflictPath)
			if file.Fallback != git.FallbackNone || !file.HasBase() || len(file.Regions) != 1 || !file.Regions[0].HasBase {
				t.Fatalf("O10 conflict = %+v", file)
			}
			region := file.Regions[0]
			if !slices.Equal(region.Ours, []string{testCase.wantOursLn}) || !slices.Equal(region.Theirs, []string{testCase.wantTheirsLn}) || !slices.Equal(region.Base, []string{"b\n"}) {
				t.Fatalf("O10 stage mapping: ours %q base %q theirs %q", region.Ours, region.Base, region.Theirs)
			}
			if fixture.Snapshot() != before {
				t.Fatal("reading conflict sides changed Git state")
			}
		})
	}
}

func TestConflictResolveRegions(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.ConflictingBranches(
		map[string]string{conflictPath: separatedLines("2", "9")},
		map[string]string{conflictPath: separatedLines("MAIN2", "MAIN9")},
		map[string]string{conflictPath: separatedLines("FEAT2", "FEAT9")},
	)
	fixture.GitFails("merge", gittest.FeatureBranch)
	reader := git.NewConflictReader(fixture.Runner)
	if regions := loadConflict(t, fixture, conflictPath).Regions; len(regions) != 2 {
		t.Fatalf("regions = %+v", regions)
	}
	if err := reader.Resolve(context.Background(), conflictPath, 1, git.SideTheirs); err != nil {
		t.Fatalf("resolve second region: %v", err)
	}
	remaining := loadConflict(t, fixture, conflictPath)
	if remaining.Fallback != git.FallbackNone || len(remaining.Regions) != 1 || !slices.Equal(remaining.Regions[0].Ours, []string{"MAIN2\n"}) {
		t.Fatalf("after one choice = %+v", remaining)
	}
	content, _ := os.ReadFile(filepath.Join(fixture.Dir, conflictPath))
	if !strings.HasSuffix(string(content), "8\nFEAT9\n10\n") {
		t.Fatalf("choice rewrote more than its region:\n%s", content)
	}
	if !strings.Contains(fixture.Git("ls-files", "-u", "--", conflictPath), conflictPath) {
		t.Fatal("choosing a side staged the file")
	}
	if err := reader.MarkResolved(context.Background(), conflictPath); !errors.Is(err, git.ErrMarkersRemain) {
		t.Fatalf("MarkResolved with markers = %v, want ErrMarkersRemain", err)
	}
	if err := reader.Resolve(context.Background(), conflictPath, 0, git.SideBase); err != nil {
		t.Fatalf("resolve with base: %v", err)
	}
	content, _ = os.ReadFile(filepath.Join(fixture.Dir, conflictPath))
	if string(content) != separatedLines("2", "FEAT9") {
		t.Fatalf("resolved content = %q", content)
	}
	if err := reader.MarkResolved(context.Background(), conflictPath); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}
	if fixture.Git("ls-files", "-u") != "" || activeOperation(t, fixture).Kind != git.OperationMerge {
		t.Fatal("MarkResolved did not stage only, or continued the merge")
	}
}

func separatedLines(second, ninth string) string {
	return "1\n" + second + "\n3\n4\n5\n6\n7\n8\n" + ninth + "\n10\n"
}

func TestConflictFallbacks(t *testing.T) {
	t.Run("adjacent hunks without a truthful base", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.ConflictingBranches(
			map[string]string{conflictPath: "1\n2\n3\n4\n5\n6\n7\n"},
			map[string]string{conflictPath: "1\nMAIN2\n3\n4\n5\nMAIN6\n7\n"},
			map[string]string{conflictPath: "1\nFEAT2\n3\n4\n5\nFEAT6\n7\n"},
		)
		fixture.GitFails("merge", gittest.FeatureBranch)
		file := loadConflict(t, fixture, conflictPath)
		if file.Fallback != git.FallbackNone || len(file.Regions) != 1 || file.Regions[0].HasBase || len(file.Regions[0].Base) != 0 {
			t.Fatalf("combined region = %+v", file)
		}
		if err := git.NewConflictReader(fixture.Runner).Resolve(context.Background(), conflictPath, 0, git.SideBase); !errors.Is(err, git.ErrNoBase) {
			t.Fatalf("base for combined region = %v", err)
		}
	})
	t.Run("T23 no base", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.ConflictingBranches(map[string]string{"other.txt": "x\n"}, map[string]string{"added.txt": "main\n"}, map[string]string{"added.txt": "feature\n"})
		fixture.GitFails("merge", gittest.FeatureBranch)
		file := loadConflict(t, fixture, "added.txt")
		if file.HasBase() || file.Fallback != git.FallbackNone || len(file.Regions) != 1 || file.Regions[0].HasBase {
			t.Fatalf("add/add conflict = %+v", file)
		}
		before := fixture.Snapshot()
		if err := git.NewConflictReader(fixture.Runner).Resolve(context.Background(), "added.txt", 0, git.SideBase); !errors.Is(err, git.ErrNoBase) || fixture.Snapshot() != before {
			t.Fatalf("T23 base choice without base = %v or changed state", err)
		}
	})
	t.Run("T24 binary", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.ConflictingBranches(map[string]string{"image.bin": "\x00base"}, map[string]string{"image.bin": "\x00main"}, map[string]string{"image.bin": "\x00feature"})
		fixture.GitFails("merge", gittest.FeatureBranch)
		file := loadConflict(t, fixture, "image.bin")
		if file.Fallback != git.FallbackBinary || len(file.Regions) != 0 {
			t.Fatalf("T24 binary = %+v", file)
		}
		if err := git.NewConflictReader(fixture.Runner).Resolve(context.Background(), "image.bin", 0, git.SideOurs); !errors.Is(err, git.ErrEditorOnly) {
			t.Fatalf("T24 binary choice = %v", err)
		}
	})
	t.Run("modify/delete", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", gittest.ConflictBase)
		fixture.Git("switch", "-q", "-c", gittest.FeatureBranch)
		fixture.Commit("feature", gittest.FeatureOne)
		fixture.Git("switch", "-q", "main")
		fixture.Git("rm", "-q", "--", conflictPath)
		fixture.Commit("delete", nil)
		fixture.GitFails("merge", gittest.FeatureBranch)
		file := loadConflict(t, fixture, conflictPath)
		if file.Fallback != git.FallbackDeleted || file.Stages[git.SideOurs].OID != "" || file.Stages[git.SideTheirs].OID == "" {
			t.Fatalf("modify/delete = %+v", file)
		}
		if err := git.NewConflictReader(fixture.Runner).MarkResolved(context.Background(), conflictPath); err != nil {
			t.Fatalf("mark resolved keeps the file as is: %v", err)
		}
	})
	t.Run("edited markers", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		content, _ := os.ReadFile(filepath.Join(fixture.Dir, conflictPath))
		fixture.Write(conflictPath, strings.Replace(string(content), "MAIN\n", "EDITED\n", 1))
		if file := loadConflict(t, fixture, conflictPath); file.Fallback != git.FallbackMarkersChanged || len(file.Regions) != 0 {
			t.Fatalf("edited markers = %+v", file)
		}
	})
	t.Run("diff3 style", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Git("config", "merge.conflictStyle", "diff3")
		fixture.ConflictingBranches(gittest.ConflictBase, gittest.ConflictMain, gittest.FeatureOne)
		fixture.GitFails("merge", gittest.FeatureBranch)
		file := loadConflict(t, fixture, conflictPath)
		if len(file.Regions) != 1 || !slices.Equal(file.Regions[0].Base, []string{"b\n"}) {
			t.Fatalf("diff3 conflict = %+v", file)
		}
	})
	t.Run("not conflicted", func(t *testing.T) {
		fixture := gittest.NewPausedFixture(t, git.OperationMerge)
		resolveConflict(fixture)
		if _, err := git.NewConflictReader(fixture.Runner).Load(context.Background(), conflictPath); !errors.Is(err, git.ErrNotConflicted) {
			t.Fatalf("load resolved file = %v", err)
		}
		if err := git.NewConflictReader(fixture.Runner).MarkResolved(context.Background(), conflictPath); !errors.Is(err, git.ErrNotConflicted) {
			t.Fatalf("mark resolved twice = %v", err)
		}
	})
}

func TestConflictEditorCommand(t *testing.T) {
	fixture := gittest.NewPausedFixture(t, git.OperationMerge)
	command, err := git.NewConflictReader(fixture.Runner).EditorCommand(context.Background(), conflictPath)
	if err != nil {
		t.Fatalf("editor command: %v", err)
	}
	want := []string{"sh", "-c", `true "$@"`, "true", filepath.Join(fixture.Dir, conflictPath)}
	if !slices.Equal(command.Args, want) || command.Dir != fixture.Dir {
		t.Fatalf("editor command = %q in %s, want %q", command.Args, command.Dir, want)
	}
}
