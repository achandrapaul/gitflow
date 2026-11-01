package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
	"github.com/achandrapaul/gitflow/internal/prefs"
)

var explorerPaths = []string{"a.txt", "b.txt", "mixed.txt", "src/c.txt", "src/d.txt", "src/deep/e.txt", "src/new.txt"}

func newExplorerFixture(t *testing.T) *gittest.Fixture {
	t.Helper()
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n", "b.txt": "b\n", "mixed.txt": "base\n", "src/c.txt": "c\n", "src/d.txt": "d\n", "src/deep/e.txt": "e\n"})
	fixture.Dirty("a.txt", "src/c.txt", "src/d.txt")
	fixture.Write("b.txt", "b staged\n")
	fixture.Write("src/deep/e.txt", "e staged\n")
	fixture.Write("mixed.txt", "base\nstaged\n")
	fixture.Git("add", "--", "b.txt", "src/deep/e.txt", "mixed.txt")
	fixture.Dirty("mixed.txt")
	fixture.Write("src/new.txt", "new\n")
	return fixture
}

func newExplorerApp(t *testing.T) (*App, *gittest.Fixture, string) {
	t.Helper()
	fixture := newExplorerFixture(t)
	prefsPath := filepath.Join(t.TempDir(), "gitflow", "prefs.json")
	gitflowApp := newTestAppFor(t, fixture, prefsPath)
	drain(gitflowApp, gitflowApp.Init())
	return gitflowApp, fixture, prefsPath
}

func visibleRows(gitflowApp *App) (files, folders []string) {
	for _, row := range gitflowApp.changes.rows() {
		if row.folder {
			folders = append(folders, row.path)
			continue
		}
		files = append(files, row.path)
	}
	return files, folders
}

func selectedPaths(gitflowApp *App) []string {
	var selected []string
	for _, path := range append(gitflowApp.changes.checked.Ordered(), gitflowApp.changes.ranged.Ordered()...) {
		if !slices.Contains(selected, path) {
			selected = append(selected, path)
		}
	}
	slices.Sort(selected)
	return selected
}

func flattenedView(gitflowApp *App) string {
	return strings.Join(strings.Fields(gitflowApp.View()), " ")
}

func TestExplorerFiltersAndLayouts(t *testing.T) {
	testCases := []struct {
		filterKey string
		wantFiles []string
	}{
		{filterKey: "1", wantFiles: explorerPaths},
		{filterKey: "2", wantFiles: []string{"b.txt", "mixed.txt", "src/deep/e.txt"}},
		{filterKey: "3", wantFiles: []string{"a.txt", "mixed.txt", "src/c.txt", "src/d.txt", "src/new.txt"}},
		{filterKey: "4", wantFiles: []string{"mixed.txt"}},
	}
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	for _, layoutKeys := range [][]string{nil, {"v"}} {
		pressAndDrain(gitflowApp, layoutKeys...)
		for _, testCase := range testCases {
			pressAndDrain(gitflowApp, testCase.filterKey)
			files, folders := visibleRows(gitflowApp)
			if !slices.Equal(files, testCase.wantFiles) {
				t.Errorf("layout %v filter %s files = %v, want %v", gitflowApp.changes.layout, testCase.filterKey, files, testCase.wantFiles)
			}
			if gitflowApp.changes.layout == prefs.LayoutFlat && len(folders) != 0 {
				t.Errorf("flat layout shows folders %v", folders)
			}
		}
	}
	pressAndDrain(gitflowApp, "1")
	_, folders := visibleRows(gitflowApp)
	if !slices.Equal(folders, []string{"src", "src/deep"}) {
		t.Fatalf("folders = %v", folders)
	}
	view := flattenedView(gitflowApp)
	for _, want := range []string{"src/ (4)", "deep/ (1)", "MM mixed.txt", "?? new.txt"} {
		if !strings.Contains(view, want) {
			t.Errorf("folder view missing %q:\n%s", want, view)
		}
	}
	if fixture.Snapshot() != before {
		t.Fatal("filters and layouts changed Git state")
	}
}

func TestFocusNeverSelects(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "space", "j", "space")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt", "b.txt"}) {
		t.Fatalf("after checking A and B: %v", got)
	}
	pressAndDrain(gitflowApp, "j", "j", "down", "up")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt", "b.txt"}) {
		t.Fatalf("j/k changed selection: %v", got)
	}
	pressAndDrain(gitflowApp, "k", "space")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt", "b.txt", "mixed.txt"}) {
		t.Fatalf("Space did not toggle only C: %v", got)
	}
	pressAndDrain(gitflowApp, "k", "k", "j", "k")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt", "b.txt", "mixed.txt"}) {
		t.Fatalf("revisit changed selection: %v", got)
	}
	if !strings.Contains(flattenedView(gitflowApp), "3 selected · 0 hidden") {
		t.Errorf("orientation counts wrong:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc")
	if len(selectedPaths(gitflowApp)) != 0 || gitflowApp.changes.focusPath != "a.txt" {
		t.Fatalf("Esc left %v focused %q", selectedPaths(gitflowApp), gitflowApp.changes.focusPath)
	}
	if fixture.Snapshot() != before {
		t.Fatal("focus and selection changed Git state")
	}
}

func TestShiftRangeKeepsAnchorAndCheckedRows(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	steps := []struct {
		keys []string
		want []string
	}{
		{keys: []string{"space", "j", "j"}, want: []string{"a.txt"}},
		{keys: []string{"J", "J"}, want: []string{"a.txt", "mixed.txt", "src/c.txt", "src/d.txt"}},
		{keys: []string{"K"}, want: []string{"a.txt", "mixed.txt", "src/c.txt"}},
		{keys: []string{"shift+up"}, want: []string{"a.txt", "mixed.txt"}},
		{keys: []string{"K"}, want: []string{"a.txt", "b.txt", "mixed.txt"}},
		{keys: []string{"shift+down"}, want: []string{"a.txt", "mixed.txt"}},
		{keys: []string{"j", "j", "k"}, want: []string{"a.txt", "mixed.txt"}},
		{keys: []string{"j", "j", "K", "K"}, want: []string{"a.txt", "mixed.txt", "src/c.txt", "src/d.txt", "src/deep/e.txt"}},
		{keys: []string{"J"}, want: []string{"a.txt", "mixed.txt", "src/d.txt", "src/deep/e.txt"}},
	}
	for _, step := range steps {
		pressAndDrain(gitflowApp, step.keys...)
		if got := selectedPaths(gitflowApp); !slices.Equal(got, step.want) {
			t.Fatalf("after %v selected = %v, want %v", step.keys, got, step.want)
		}
	}
	if gitflowApp.changes.rangeAnchor != "src/deep/e.txt" {
		t.Fatalf("anchor = %q, want the file focused when the second range started", gitflowApp.changes.rangeAnchor)
	}
	pressAndDrain(gitflowApp, "j")
	if gitflowApp.changes.rangeAnchor != "" || len(selectedPaths(gitflowApp)) != 4 {
		t.Fatalf("j kept anchor %q or changed selection %v", gitflowApp.changes.rangeAnchor, selectedPaths(gitflowApp))
	}

	pressAndDrain(gitflowApp, "esc", "v", "1")
	for range explorerPaths {
		pressAndDrain(gitflowApp, "k")
	}
	pressAndDrain(gitflowApp, "j", "j", "J", "J", "J")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"mixed.txt", "src/c.txt", "src/d.txt", "src/deep/e.txt"}) {
		t.Fatalf("folder range = %v, want files only", got)
	}
	if fixture.Snapshot() != before {
		t.Fatal("range selection changed Git state")
	}
}

func TestShiftRangeAtListEdges(t *testing.T) {
	gitflowApp, _, _ := newExplorerApp(t)
	pressAndDrain(gitflowApp, "K")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt"}) {
		t.Fatalf("K on the first file selected %v", got)
	}
	pressAndDrain(gitflowApp, "esc", "G", "J")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"src/new.txt"}) {
		t.Fatalf("J on the last file selected %v", got)
	}
}

func TestFileListKeepsFocusCentred(t *testing.T) {
	const fileCount = 60
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", nil)
	for index := range fileCount {
		fixture.Write(fmt.Sprintf("f%02d.txt", index), "new\n")
	}
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	pressAndDrain(gitflowApp, "j", "j", "j")
	if !strings.Contains(gitflowApp.View(), "f00.txt") {
		t.Fatalf("list scrolled before focus reached the middle:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")
	if strings.Contains(gitflowApp.View(), "f00.txt") {
		t.Fatalf("list did not scroll once focus passed the middle:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "G", "k", "k")
	if !strings.Contains(gitflowApp.View(), fmt.Sprintf("f%02d.txt", fileCount-1)) {
		t.Fatalf("k at the end scrolled away from the last file:\n%s", gitflowApp.View())
	}
}

func TestSelectAllIncludesHiddenFiles(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "3", "v", "j", "j")
	if gitflowApp.changes.focusPath != "src" {
		t.Fatalf("focus = %q, want the src folder", gitflowApp.changes.focusPath)
	}
	pressAndDrain(gitflowApp, "h", "space")
	if files, _ := visibleRows(gitflowApp); !slices.Equal(files, []string{"a.txt", "mixed.txt"}) || len(selectedPaths(gitflowApp)) != 0 {
		t.Fatalf("collapse or folder Space wrong: files %v selected %v", files, selectedPaths(gitflowApp))
	}
	pressAndDrain(gitflowApp, "S", "S")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, explorerPaths) {
		t.Fatalf("Shift+S selected %v", got)
	}
	if !strings.Contains(flattenedView(gitflowApp), "7 selected · 5 hidden") {
		t.Errorf("counts wrong:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "1", "k", "space")
	if got := selectedPaths(gitflowApp); len(got) != len(explorerPaths)-1 || slices.Contains(got, "mixed.txt") {
		t.Fatalf("Space after filter change = %v", got)
	}
	if !strings.Contains(flattenedView(gitflowApp), "6 selected · 4 hidden") {
		t.Errorf("counts after filter wrong:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "j", "l")
	if files, _ := visibleRows(gitflowApp); len(files) != len(explorerPaths) {
		t.Fatalf("expand showed %v", files)
	}
	if fixture.Snapshot() != before {
		t.Fatal("select all changed Git state")
	}
}

func TestOverlayEscBeforeSelectionClear(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "space", "2", "J")
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"a.txt", "b.txt", "mixed.txt"}) {
		t.Fatalf("setup selected %v", got)
	}
	if !strings.Contains(flattenedView(gitflowApp), "3 selected · 1 hidden") {
		t.Errorf("hidden count wrong:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "a", "esc")
	if gitflowApp.overlay != nil || len(selectedPaths(gitflowApp)) != 3 {
		t.Fatalf("first Esc: overlay %v selected %v", gitflowApp.overlay, selectedPaths(gitflowApp))
	}
	pressAndDrain(gitflowApp, "esc")
	if len(selectedPaths(gitflowApp)) != 0 || gitflowApp.changes.rangeAnchor != "" {
		t.Fatalf("second Esc left %v anchor %q", selectedPaths(gitflowApp), gitflowApp.changes.rangeAnchor)
	}
	if gitflowApp.changes.focusPath != "mixed.txt" || gitflowApp.changes.filter != git.FilterStaged {
		t.Fatalf("Esc changed focus %q or filter %v", gitflowApp.changes.focusPath, gitflowApp.changes.filter)
	}
	pressAndDrain(gitflowApp, "esc")
	if len(gitflowApp.stack) != 1 || gitflowApp.changes.focusPath != "mixed.txt" {
		t.Fatal("root Esc without selection was not a no-op")
	}
	if fixture.Snapshot() != before {
		t.Fatal("Esc changed Git state")
	}
}

func TestLayoutPreference(t *testing.T) {
	gitflowApp, fixture, prefsPath := newExplorerApp(t)
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "v")
	if _, folders := visibleRows(gitflowApp); gitflowApp.changes.layout != prefs.LayoutFolder || len(folders) == 0 {
		t.Fatalf("v did not switch to folders immediately: %v", folders)
	}
	if saved, err := prefs.NewStore(prefsPath).Load(); err != nil || saved.Layout != prefs.LayoutFolder {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	restarted := newTestAppFor(t, fixture, prefsPath)
	otherRepository := newTestAppFor(t, newExplorerFixture(t), prefsPath)
	if restarted.changes.layout != prefs.LayoutFolder || otherRepository.changes.layout != prefs.LayoutFolder {
		t.Fatal("folder layout did not survive restart or repository switch")
	}
	pressAndDrain(otherRepository, "v")
	if saved, _ := prefs.NewStore(prefsPath).Load(); saved.Layout != prefs.LayoutFlat {
		t.Fatalf("flat not saved: %+v", saved)
	}
	if fixture.Snapshot() != before {
		t.Fatal("layout toggle changed Git state")
	}
}

func TestLayoutPreferenceFailures(t *testing.T) {
	fixture := newExplorerFixture(t)
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	unwritable := newTestAppFor(t, fixture, filepath.Join(blocker, "prefs.json"))
	pressAndDrain(unwritable, "v")
	if unwritable.changes.layout != prefs.LayoutFolder || !strings.Contains(unwritable.errorText, layoutNotSaved) {
		t.Fatalf("layout %v error %q", unwritable.changes.layout, unwritable.errorText)
	}

	corruptPath := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(corruptPath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	corrupt := newTestAppFor(t, fixture, corruptPath)
	if corrupt.changes.layout != prefs.LayoutFlat || !strings.Contains(corrupt.errorText, layoutNotLoaded) {
		t.Fatalf("layout %v error %q", corrupt.changes.layout, corrupt.errorText)
	}
}

func TestDiffInspection(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "S", "j", "j")
	preview := flattenedView(gitflowApp)
	for _, want := range []string{stagedHeading, unstagedHeading, "+staged"} {
		if !strings.Contains(preview, want) {
			t.Errorf("S01 preview missing %q:\n%s", want, gitflowApp.View())
		}
	}
	pressAndDrain(gitflowApp, "enter")
	if gitflowApp.top().ID() != ScreenDiff {
		t.Fatalf("Enter opened %s", gitflowApp.top().ID())
	}
	view := flattenedView(gitflowApp)
	for _, want := range []string{"S02", "mixed.txt", stagedHeading, unstagedHeading, "+staged"} {
		if !strings.Contains(view, want) {
			t.Errorf("S02 missing %q:\n%s", want, gitflowApp.View())
		}
	}
	pressAndDrain(gitflowApp, "j", "G", ":", "esc", "esc")
	if gitflowApp.top().ID() != ScreenChanges || gitflowApp.changes.focusPath != "mixed.txt" || len(selectedPaths(gitflowApp)) != len(explorerPaths) {
		t.Fatalf("return from S02: top %s focus %q selected %v", gitflowApp.top().ID(), gitflowApp.changes.focusPath, selectedPaths(gitflowApp))
	}

	pressAndDrain(gitflowApp, "esc", "G", "enter")
	view = flattenedView(gitflowApp)
	if !strings.Contains(view, "+new") || !strings.Contains(view, noStagedChanges) {
		t.Errorf("untracked S02 wrong:\n%s", gitflowApp.View())
	}
	if fixture.Snapshot() != before {
		t.Fatal("diff inspection changed Git state")
	}
}

func TestHeaderBranchIntelligence(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(fixture *gittest.Fixture)
		want  []string
	}{
		{
			name: "T12 upstream and base are independent",
			setup: func(fixture *gittest.Fixture) {
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
			},
			want: []string{"Branch feature @", "origin/feature ↑2 ↓0", "Base origin/main ↑3 ↓1", "Local refs · last fetched 20"},
		},
		{
			name: "T11 no upstream",
			setup: func(fixture *gittest.Fixture) {
				fixture.Commit("base", nil)
			},
			want: []string{"No upstream", "Base main ↑0 ↓0", "Local refs · last fetched unknown"},
		},
		{
			name: "T13 detached HEAD and missing base",
			setup: func(fixture *gittest.Fixture) {
				head := fixture.Commit("base", nil)
				fixture.Git("switch", "-q", "--detach", head)
				fixture.Git("branch", "-q", "-D", "main")
			},
			want: []string{"DETACHED @", "No upstream", "Base unavailable"},
		},
		{
			name:  "unborn branch",
			setup: func(*gittest.Fixture) {},
			want:  []string{"Branch main · no commits yet", "Base unavailable"},
		},
		{
			name: "paused merge badge",
			setup: func(fixture *gittest.Fixture) {
				fixture.Commit("base", map[string]string{"conflict.txt": "base\n"})
				fixture.Git("switch", "-q", "-c", "other")
				fixture.Commit("other", map[string]string{"conflict.txt": "other\n"})
				fixture.Git("switch", "-q", "main")
				fixture.Commit("main", map[string]string{"conflict.txt": "main\n"})
				if _, err := fixture.Runner.Run(context.Background(), "merge", "-q", "other"); err == nil {
					t.Fatal("merge unexpectedly succeeded")
				}
			},
			want: []string{"Paused: merge", "UU conflict.txt"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			testCase.setup(fixture)
			before := fixture.Snapshot()
			gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
			drain(gitflowApp, gitflowApp.Init())
			view := flattenedView(gitflowApp)
			for _, want := range testCase.want {
				if !strings.Contains(view, want) {
					t.Errorf("header missing %q:\n%s", want, gitflowApp.View())
				}
			}
			if gitflowApp.errorText != "" {
				t.Errorf("unexpected error %q", gitflowApp.errorText)
			}
			if fixture.Snapshot() != before {
				t.Fatal("loading the header changed Git state")
			}
		})
	}
}

func indexCodes(t *testing.T, fixture *gittest.Fixture) map[string]string {
	t.Helper()
	status, err := git.NewStatusReader(fixture.Runner).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	codes := map[string]string{}
	for _, change := range status.Changes {
		codes[change.Path] = string([]byte{change.Staged, change.Unstaged})
	}
	return codes
}

func TestStageTargetsCheckedMembersOnly(t *testing.T) {
	testCases := []struct {
		name        string
		keys        []string
		action      string
		wantFooter  string
		wantCounts  string
		wantNotice  []string
		wantChanged map[string]string
	}{
		{
			name:        "T109 three checked, two hidden, focused unchecked row untouched",
			keys:        []string{"space", "j", "j", "space", "j", "space", "2", "k", "k", "k"},
			action:      "s",
			wantFooter:  "s Stage 3 files (2 hidden)",
			wantCounts:  "3 selected · 2 hidden",
			wantNotice:  []string{"Staged 3 files (2 hidden)"},
			wantChanged: map[string]string{"a.txt": "M.", "mixed.txt": "M.", "src/c.txt": "M.", "b.txt": "MM"},
		},
		{
			name:        "T147 ineligible checked member is reported, not replaced",
			keys:        []string{"space", "j", "j", "space", "j", "space", "j", "j", "space", "2", "k", "k", "k"},
			action:      "s",
			wantFooter:  "s Stage 3 files (2 hidden)",
			wantCounts:  "4 selected · 2 hidden",
			wantNotice:  []string{"Staged 3 files (2 hidden)", "skipped src/deep/e.txt: nothing to stage"},
			wantChanged: map[string]string{"a.txt": "M.", "mixed.txt": "M.", "src/c.txt": "M.", "b.txt": "MM", "src/deep/e.txt": "M."},
		},
		{
			name:        "focused file is the sole target without a checked set",
			keys:        []string{"j"},
			action:      "u",
			wantFooter:  "u Unstage 1 file",
			wantCounts:  "0 selected · 0 hidden",
			wantNotice:  []string{"Unstaged 1 file"},
			wantChanged: map[string]string{"b.txt": ".M", "mixed.txt": "MM"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newExplorerFixture(t)
			fixture.Dirty("b.txt")
			gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
			drain(gitflowApp, gitflowApp.Init())
			pressAndDrain(gitflowApp, testCase.keys...)
			focused := gitflowApp.changes.focusPath
			view := flattenedView(gitflowApp)
			for _, want := range []string{testCase.wantFooter, testCase.wantCounts} {
				if !strings.Contains(view, want) {
					t.Errorf("before %s, view missing %q:\n%s", testCase.action, want, gitflowApp.View())
				}
			}
			pressAndDrain(gitflowApp, testCase.action)
			for _, want := range testCase.wantNotice {
				if !strings.Contains(gitflowApp.notice, want) {
					t.Errorf("notice = %q, want %q", gitflowApp.notice, want)
				}
			}
			codes := indexCodes(t, fixture)
			for path, want := range testCase.wantChanged {
				if codes[path] != want {
					t.Errorf("%s = %q, want %q", path, codes[path], want)
				}
			}
			if gitflowApp.changes.focusPath != focused || gitflowApp.busy {
				t.Errorf("focus %q → %q, busy %v", focused, gitflowApp.changes.focusPath, gitflowApp.busy)
			}
		})
	}
}

func TestStageBlocksStaleTargets(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	pressAndDrain(gitflowApp, "space", "j", "j", "space")
	fixture.Git("checkout", "--", "a.txt")
	before := fixture.Snapshot()
	pressAndDrain(gitflowApp, "s")
	if fixture.Snapshot() != before {
		t.Fatal("stale selection still mutated Git")
	}
	if !strings.Contains(gitflowApp.errorText, "a.txt") || !strings.Contains(gitflowApp.errorText, "changed since") {
		t.Errorf("stale blocker = %q", gitflowApp.errorText)
	}
	if got := selectedPaths(gitflowApp); !slices.Equal(got, []string{"mixed.txt"}) {
		t.Errorf("refresh kept vanished identities: %v", got)
	}
}

func TestStageFailureShowsRealState(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	if err := os.WriteFile(filepath.Join(fixture.Dir, ".git", "index.lock"), nil, 0o644); err != nil {
		t.Fatalf("lock index: %v", err)
	}
	pressAndDrain(gitflowApp, "s")
	if !strings.Contains(gitflowApp.errorText, "stage a.txt") || gitflowApp.busy {
		t.Errorf("failure error = %q busy %v", gitflowApp.errorText, gitflowApp.busy)
	}
	if codes := indexCodes(t, fixture); codes["a.txt"] != ".M" {
		t.Errorf("a.txt = %q after failed stage", codes["a.txt"])
	}
}

func TestStageIsSerialised(t *testing.T) {
	gitflowApp, _, _ := newExplorerApp(t)
	pending := press(gitflowApp, "s")
	if pending == nil || !gitflowApp.busy {
		t.Fatal("stage did not start a mutation")
	}
	if press(gitflowApp, "s") != nil || gitflowApp.notice != actionBusy {
		t.Errorf("second action while busy: notice %q", gitflowApp.notice)
	}
	drain(gitflowApp, pending)
	if gitflowApp.busy {
		t.Error("busy not cleared after mutation")
	}
}

func TestDiffScreenStagesInspectedFile(t *testing.T) {
	gitflowApp, fixture, _ := newExplorerApp(t)
	pressAndDrain(gitflowApp, "space", "j", "j", "enter", "s")
	if codes := indexCodes(t, fixture); codes["mixed.txt"] != "M." || codes["a.txt"] != ".M" {
		t.Fatalf("S02 stage codes = %v", codes)
	}
	if view := flattenedView(gitflowApp); gitflowApp.top().ID() != ScreenDiff || !strings.Contains(view, noUnstagedChanges) {
		t.Errorf("S02 not refreshed:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "u")
	if codes := indexCodes(t, fixture); codes["mixed.txt"] != ".M" {
		t.Fatalf("S02 unstage codes = %v", codes)
	}
}

func TestFocusMovesToNearestSurvivor(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n", "b.txt": "b\n", "c.txt": "c\n"})
	fixture.Dirty("a.txt", "b.txt", "c.txt")
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	pressAndDrain(gitflowApp, "3", "j", "s")
	if gitflowApp.changes.focusPath != "c.txt" {
		t.Errorf("focus after b.txt left the filter = %q, want c.txt", gitflowApp.changes.focusPath)
	}
	pressAndDrain(gitflowApp, "s")
	if gitflowApp.changes.focusPath != "a.txt" {
		t.Errorf("focus after last row left = %q, want a.txt", gitflowApp.changes.focusPath)
	}
}

func runComposerAction(t *testing.T, gitflowApp *App, id ActionID) {
	t.Helper()
	press(gitflowApp, "a")
	overlay, isActions := gitflowApp.overlay.(*actionsOverlay)
	if !isActions {
		t.Fatalf("a did not open the actions menu, overlay = %v", gitflowApp.overlay)
	}
	index := slices.IndexFunc(overlay.entries, func(entry menuEntry) bool { return entry.action.ID == id })
	if index < 0 {
		t.Fatalf("%s missing from the %s actions menu", id, overlay.scope)
	}
	for range index {
		press(gitflowApp, "j")
	}
	pressAndDrain(gitflowApp, "enter")
}

func reviewDraft(t *testing.T, gitflowApp *App, message string) {
	t.Helper()
	press(gitflowApp, "c")
	for _, letter := range message {
		press(gitflowApp, string(letter))
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenComposer || !gitflowApp.changes.review.loaded {
		t.Fatalf("Esc from the editor did not open the S03 review: %s, review %+v", gitflowApp.top().ID(), gitflowApp.changes.review)
	}
}

func headSubject(fixture *gittest.Fixture) string {
	return strings.TrimSpace(fixture.Git("log", "-1", "--format=%s"))
}

func TestCommitFromSingleComposer(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	reviewDraft(t, gitflowApp, "Fix")
	view := flattenedView(gitflowApp)
	if !strings.Contains(view, "GitFlow · S03") || !strings.Contains(view, "staged.txt") || strings.Contains(view, "untracked.txt") {
		t.Fatalf("T57 S03 review must list only the staged index:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenChanges || gitflowApp.changes.focus != focusFiles {
		t.Fatalf("second Esc did not return to S01 files")
	}
	press(gitflowApp, "c")
	if gitflowApp.changes.draft.Value() != "Fix" || len(gitflowApp.stack) != 1 {
		t.Fatalf("T102 draft %q stack %d, want the same single draft", gitflowApp.changes.draft.Value(), len(gitflowApp.stack))
	}
	pressAndDrain(gitflowApp, "esc")
	runComposerAction(t, gitflowApp, actionCommit)
	if headSubject(fixture) != "Fix" || !strings.Contains(gitflowApp.notice, "Committed") {
		t.Fatalf("commit subject %q notice %q error %q", headSubject(fixture), gitflowApp.notice, gitflowApp.errorText)
	}
	if fixture.Git("show", "--name-only", "--format=", "HEAD") != "staged.txt\n" {
		t.Error("T09 commit included more than the staged index")
	}
	if codes := indexCodes(t, fixture); codes["tracked.txt"] != ".M" {
		t.Errorf("unstaged worktree change lost: %v", codes)
	}
	if gitflowApp.changes.draft.Value() != "" {
		t.Error("draft not cleared after verified success")
	}
}

func TestCommitRequiresMessageAndStagedChanges(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	reviewDraft(t, gitflowApp, "")
	before := fixture.Snapshot()
	runComposerAction(t, gitflowApp, actionCommit)
	if fixture.Snapshot() != before || !strings.Contains(gitflowApp.notice, emptyMessage) {
		t.Fatalf("empty draft commit notice = %q", gitflowApp.notice)
	}
	fixture.Git("restore", "--staged", "staged.txt")
	reviewDraft(t, gitflowApp, "Nothing staged")
	runComposerAction(t, gitflowApp, actionCommit)
	if !strings.Contains(gitflowApp.notice, noStagedChanges) {
		t.Fatalf("C11 empty-stage commit notice = %q", gitflowApp.notice)
	}
}

func TestCommitRefusesStaleReview(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	reviewDraft(t, gitflowApp, "Fix")
	reviewedTree := gitflowApp.changes.review.tree
	head := fixture.Git("rev-parse", "HEAD")
	fixture.Git("add", "--", "tracked.txt")
	runComposerAction(t, gitflowApp, actionCommit)
	if fixture.Git("rev-parse", "HEAD") != head || !strings.Contains(gitflowApp.errorText, "changed since review") {
		t.Fatalf("T103 stale review error = %q", gitflowApp.errorText)
	}
	if gitflowApp.changes.draft.Value() != "Fix" || gitflowApp.changes.review.tree == reviewedTree {
		t.Errorf("T103 draft %q, review not re-run", gitflowApp.changes.draft.Value())
	}
}

func TestCommitHookFailureKeepsDraftAndRealIndex(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	fixture.Hook("pre-commit", "echo hooked > hooked.txt\ngit add hooked.txt\nexit 1\n")
	head := fixture.Git("rev-parse", "HEAD")
	reviewDraft(t, gitflowApp, "Fix")
	reviewedTree := gitflowApp.changes.review.tree
	runComposerAction(t, gitflowApp, actionCommit)
	if fixture.Git("rev-parse", "HEAD") != head || gitflowApp.errorText == "" {
		t.Fatalf("T10 hook failure reported as %q / %q", gitflowApp.notice, gitflowApp.errorText)
	}
	if !strings.Contains(gitflowApp.errorText, "a v") {
		t.Errorf("T134 failure should point at the guarded bypass: %q", gitflowApp.errorText)
	}
	if gitflowApp.changes.draft.Value() != "Fix" {
		t.Errorf("T104 draft lost: %q", gitflowApp.changes.draft.Value())
	}
	if gitflowApp.changes.review.tree == reviewedTree || !strings.Contains(flattenedView(gitflowApp), "hooked.txt") {
		t.Errorf("T121 review not re-read after hook changed the index:\n%s", gitflowApp.View())
	}
	if gitflowApp.overlay != nil {
		t.Error("T134 a failure must not open the bypass automatically")
	}
}

func TestCommitDisclosesHookChanges(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	fixture.Hook("pre-commit", "echo hooked > hooked.txt\ngit add hooked.txt\n")
	reviewDraft(t, gitflowApp, "Fix")
	runComposerAction(t, gitflowApp, actionCommit)
	view := flattenedView(gitflowApp)
	if !strings.Contains(gitflowApp.notice, differsFromReview) || !strings.Contains(view, "A hooked.txt") {
		t.Fatalf("T122 hook change not disclosed: notice %q\n%s", gitflowApp.notice, gitflowApp.View())
	}
}

func TestCommitWithoutVerificationIsOneShot(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	fixture.Hook("pre-commit", "exit 1\n")
	head := fixture.Git("rev-parse", "HEAD")
	reviewDraft(t, gitflowApp, "Bypass")
	press(gitflowApp, "a", "v")
	view := flattenedView(gitflowApp)
	if overlayID(gitflowApp) != ScreenConfirm || !strings.Contains(view, "pre-commit") || !strings.Contains(view, "commit-msg") {
		t.Fatalf("K43 a v did not open the warned S10 review:\n%s", gitflowApp.View())
	}
	for _, cancel := range []string{"esc", "enter"} {
		pressAndDrain(gitflowApp, cancel)
		if gitflowApp.overlay != nil || fixture.Git("rev-parse", "HEAD") != head {
			t.Fatalf("T125 %s did not cancel without committing", cancel)
		}
		press(gitflowApp, "a", "v")
	}
	pressAndDrain(gitflowApp, "y")
	if headSubject(fixture) != "Bypass" {
		t.Fatalf("T143 confirmed bypass did not commit: %q", gitflowApp.errorText)
	}
	fixture.Write("next.txt", "next\n")
	fixture.Git("add", "--", "next.txt")
	pressAndDrain(gitflowApp, "esc")
	reviewDraft(t, gitflowApp, "Normal")
	runComposerAction(t, gitflowApp, actionCommit)
	if headSubject(fixture) != "Bypass" || gitflowApp.errorText == "" {
		t.Fatalf("T125 next normal commit skipped hooks: subject %q", headSubject(fixture))
	}
}

func TestAmendWarnsAboutPublishedHistory(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	fixture.WithRemote()
	fixture.Git("push", "-q", "origin", "main")
	published := fixture.Git("rev-parse", "HEAD")
	parents := fixture.Git("log", "-1", "--format=%P")
	pressAndDrain(gitflowApp, "esc")
	reviewDraft(t, gitflowApp, "Amended")
	runComposerAction(t, gitflowApp, actionAmend)
	if overlayID(gitflowApp) != ScreenConfirm || !strings.Contains(flattenedView(gitflowApp), publishedWarning) {
		t.Fatalf("T129 amend did not warn about published history:\n%s", gitflowApp.View())
	}
	pressAndDrain(gitflowApp, "esc")
	if fixture.Git("rev-parse", "HEAD") != published {
		t.Fatal("T129 cancelled amend changed HEAD")
	}
	runComposerAction(t, gitflowApp, actionAmend)
	pressAndDrain(gitflowApp, "y")
	if headSubject(fixture) != "Amended" || fixture.Git("log", "-1", "--format=%P") != parents {
		t.Fatalf("T129 amend result subject %q error %q", headSubject(fixture), gitflowApp.errorText)
	}
	if fixture.Git("rev-parse", "origin/main") != published {
		t.Error("T129 amend pushed")
	}
}

func TestHeaderSummaryAndRowTags(t *testing.T) {
	gitflowApp, _, _ := newExplorerApp(t)
	view := gitflowApp.View()
	if !strings.Contains(view, "2 staged · 1 mixed · 4 unstaged") || strings.Contains(view, "Read-only") {
		t.Fatalf("header summary missing:\n%s", view)
	}
	for path, want := range map[string]string{"b.txt": "[Staged]", "mixed.txt": "[Mixed]", "a.txt": ""} {
		line := ""
		for _, candidate := range strings.Split(view, "\n") {
			if strings.Contains(candidate, " "+path) {
				line = candidate
			}
		}
		hasTag := strings.Contains(line, "[Staged]") || strings.Contains(line, "[Mixed]")
		if (want == "" && hasTag) || (want != "" && !strings.Contains(line, want)) {
			t.Errorf("%s row = %q, want tag %q", path, line, want)
		}
	}
}
