package git_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

const (
	testPageSize      = 100
	largeHistorySize  = 1100
	largeHistoryMerge = 10
)

func readAllPages(t *testing.T, pager *git.HistoryPager, cursor git.HistoryCursor) ([]git.CommitSummary, int) {
	t.Helper()
	var commits []git.CommitSummary
	pages := 0
	for {
		page, err := pager.Page(context.Background(), cursor, testPageSize)
		if err != nil {
			t.Fatalf("Page at %d: %v", cursor.Offset, err)
		}
		pages++
		commits = append(commits, page.Commits...)
		cursor = page.Next
		if page.Done {
			return commits, pages
		}
	}
}

func TestHistoryPagination(t *testing.T) {
	fixture := gittest.NewFixture(t)
	tipOID := fixture.ImportHistory(largeHistorySize, largeHistoryMerge)
	before := fixture.Snapshot()
	pager := git.NewHistoryPager(fixture.Runner)

	cursor, err := pager.Open(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if cursor.TipOID != tipOID || cursor.Label != "main" || cursor.Detached {
		t.Fatalf("cursor = %+v, want main at %s", cursor, tipOID)
	}
	commits, pages := readAllPages(t, pager, cursor)

	want := strings.Fields(fixture.Git("rev-list", "--topo-order", "main"))
	got := make([]string, len(commits))
	for position, commit := range commits {
		got[position] = commit.OID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("T14 paged %d commits, rev-list has %d; order or membership differs", len(got), len(want))
	}
	if pages != len(want)/testPageSize+1 {
		t.Fatalf("pages = %d for %d commits", pages, len(want))
	}
	mergeRows := 0
	for _, commit := range commits {
		if strings.Count(commit.Graph, "*") != 1 {
			t.Fatalf("graph %q of %s has no single commit marker", commit.Graph, commit.OID)
		}
		if len(commit.Parents) > 1 {
			mergeRows++
		}
	}
	if mergeRows != largeHistorySize/largeHistoryMerge {
		t.Fatalf("merge rows = %d", mergeRows)
	}
	if fixture.Snapshot() != before {
		t.Fatal("reading history changed Git state")
	}
}

func TestHistoryGraphLanes(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"a.txt": "a"})
	fixture.Git("switch", "-q", "-c", "side")
	fixture.Commit("side work", map[string]string{"b.txt": "b"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main work", map[string]string{"c.txt": "c"})
	fixture.Git("merge", "-q", "--no-ff", "-m", "merge side", "side")
	pager := git.NewHistoryPager(fixture.Runner)
	cursor, err := pager.Open(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	page, err := pager.Page(context.Background(), cursor, testPageSize)
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	graphs := make([]string, len(page.Commits))
	for position, commit := range page.Commits {
		graphs[position] = commit.Graph
	}
	want := []string{"*", "| *", "* |", "*"}
	if !slices.Equal(graphs, want) || !page.Done {
		t.Fatalf("graphs = %q (done %v), want %q", graphs, page.Done, want)
	}
	firstPage, err := pager.Page(context.Background(), cursor, 2)
	if err != nil {
		t.Fatalf("Page first two: %v", err)
	}
	secondPage, err := pager.Page(context.Background(), firstPage.Next, 2)
	if err != nil {
		t.Fatalf("Page next two: %v", err)
	}
	if secondPage.Commits[0].Graph != want[2] || secondPage.Commits[1].Graph != want[3] {
		t.Fatalf("lanes across pages = %q, %q", secondPage.Commits[0].Graph, secondPage.Commits[1].Graph)
	}
}

func TestHistorySources(t *testing.T) {
	testCases := []struct {
		name         string
		setup        func(fixture *gittest.Fixture) string
		wantLabel    string
		wantDetached bool
		wantCommits  int
	}{
		{
			name:        "unborn branch",
			setup:       func(fixture *gittest.Fixture) string { return "" },
			wantLabel:   "main",
			wantCommits: 0,
		},
		{
			name: "detached HEAD",
			setup: func(fixture *gittest.Fixture) string {
				first := fixture.Commit("first", map[string]string{"a.txt": "1"})
				fixture.Commit("second", map[string]string{"a.txt": "2"})
				fixture.Git("switch", "-q", "--detach", first)
				return first
			},
			wantLabel:    "HEAD",
			wantDetached: true,
			wantCommits:  1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			wantTip := testCase.setup(fixture)
			pager := git.NewHistoryPager(fixture.Runner)
			cursor, err := pager.Open(context.Background(), "HEAD")
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if cursor.TipOID != wantTip || cursor.Label != testCase.wantLabel || cursor.Detached != testCase.wantDetached {
				t.Fatalf("cursor = %+v", cursor)
			}
			commits, _ := readAllPages(t, pager, cursor)
			if len(commits) != testCase.wantCommits {
				t.Fatalf("commits = %d, want %d", len(commits), testCase.wantCommits)
			}
		})
	}
}

func TestHistoryStale(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.ImportHistory(150, 0)
	pager := git.NewHistoryPager(fixture.Runner)
	cursor, err := pager.Open(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	first, err := pager.Page(context.Background(), cursor, testPageSize)
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if stale, err := pager.Stale(context.Background(), cursor); err != nil || stale {
		t.Fatalf("Stale before move = %v, %v", stale, err)
	}

	fixture.Commit("moved", map[string]string{"moved.txt": "x"})

	if stale, err := pager.Stale(context.Background(), cursor); err != nil || !stale {
		t.Fatalf("Stale after move = %v, %v", stale, err)
	}
	second, err := pager.Page(context.Background(), first.Next, testPageSize)
	if err != nil {
		t.Fatalf("Page after move: %v", err)
	}
	if len(second.Commits) != 50 || !second.Done || second.Commits[0].OID == first.Commits[0].OID {
		t.Fatalf("second page from fixed tip = %d commits, done %v", len(second.Commits), second.Done)
	}
}

func TestCommitDetail(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"a.txt": "old\n"})
	fixture.Write("a.txt", "new\n")
	fixture.Git("add", "a.txt")
	fixture.Git("commit", "-q", "-m", "change a", "-m", "explains why")
	oid := strings.TrimSpace(fixture.Git("rev-parse", "HEAD"))
	before := fixture.Snapshot()

	detail, err := git.NewHistoryPager(fixture.Runner).Detail(context.Background(), oid)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail.Summary.OID != oid || detail.Summary.Subject != "change a" || detail.Body != "explains why" || detail.Summary.Author != "Fixture User" {
		t.Fatalf("detail = %+v", detail)
	}
	for _, fragment := range []string{"a.txt | 2 +-", "-old", "+new"} {
		if !strings.Contains(detail.Patch, fragment) {
			t.Fatalf("patch lacks %q:\n%s", fragment, detail.Patch)
		}
	}
	if fixture.Snapshot() != before {
		t.Fatal("reading commit detail changed Git state")
	}
}

func TestCommitFiles(t *testing.T) {
	fixture := gittest.NewFixture(t)
	root := fixture.Commit("root", map[string]string{"keep.txt": "keep\n", "move.txt": "line one\nline two\nline three\n", "drop.txt": "drop\n"})
	fixture.Git("mv", "move.txt", "moved.txt")
	fixture.Git("rm", "-q", "drop.txt")
	child := fixture.Commit("change", map[string]string{"keep.txt": "kept\n", "added.txt": "added\n"})
	before := fixture.Snapshot()
	differ := git.NewDiffer(fixture.Runner)

	rootFiles, err := differ.CommitFiles(context.Background(), root, "")
	if err != nil {
		t.Fatalf("CommitFiles root: %v", err)
	}
	if len(rootFiles) != 3 || rootFiles[0].Status != 'A' {
		t.Fatalf("root files = %+v", rootFiles)
	}
	files, err := differ.CommitFiles(context.Background(), child, root)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	want := []git.CommitFile{
		{Path: "added.txt", Status: 'A'},
		{Path: "drop.txt", Status: 'D'},
		{Path: "keep.txt", Status: 'M'},
		{Path: "moved.txt", OriginalPath: "move.txt", Status: 'R', UnsupportedReason: "rename"},
	}
	if !slices.Equal(files, want) {
		t.Fatalf("files = %+v, want %+v", files, want)
	}

	renamePatch, err := differ.FilePatch(context.Background(), git.FileSelection{OID: child, Parent: root, Path: "moved.txt", OriginalPath: "move.txt"})
	if err != nil {
		t.Fatalf("FilePatch rename: %v", err)
	}
	if !strings.Contains(renamePatch, "rename from move.txt") || !strings.Contains(renamePatch, "rename to moved.txt") {
		t.Fatalf("rename patch = %q", renamePatch)
	}
	modifyPatch, err := differ.FilePatch(context.Background(), git.FileSelection{OID: child, Parent: root, Path: "keep.txt"})
	if err != nil {
		t.Fatalf("FilePatch modify: %v", err)
	}
	if !strings.Contains(modifyPatch, "+kept") || strings.Contains(modifyPatch, "added.txt") {
		t.Fatalf("modify patch = %q", modifyPatch)
	}
	rootPatch, err := differ.FilePatch(context.Background(), git.FileSelection{OID: root, Path: "keep.txt"})
	if err != nil {
		t.Fatalf("FilePatch root: %v", err)
	}
	if !strings.Contains(rootPatch, "+keep") {
		t.Fatalf("root patch = %q", rootPatch)
	}
	if fixture.Snapshot() != before {
		t.Fatal("reading commit files changed Git state")
	}
}
