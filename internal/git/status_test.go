package git_test

import (
	"context"
	"slices"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func TestParseStatusRecords(t *testing.T) {
	raw := "# branch.oid 1111111111111111111111111111111111111111\x00" +
		"# branch.head feature/auth\x00" +
		"# branch.upstream origin/feature/auth\x00" +
		"# branch.ab +2 -1\x00" +
		"1 M. N... 100644 100644 100644 aaaa bbbb src/staged only.go\x00" +
		"1 MM N... 100644 100644 100644 aaaa bbbb src/mixed.go\x00" +
		"1 .M N... 100644 100644 100644 aaaa bbbb src/dirty\nnewline.go\x00" +
		"2 R. N... 100644 100644 100644 aaaa bbbb R100 new name.go\x00old name.go\x00" +
		"u UU N... 100644 100644 100644 100644 aaaa bbbb cccc conflict.go\x00" +
		"? untracked dir/\x00"
	status, err := git.ParseStatus([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if status.Branch.Head != "feature/auth" || status.Branch.Upstream != "origin/feature/auth" {
		t.Fatalf("branch = %+v", status.Branch)
	}
	if !status.Branch.HasAheadBehind || status.Branch.Ahead != 2 || status.Branch.Behind != 1 {
		t.Fatalf("ahead/behind = %+v", status.Branch)
	}
	paths := make([]string, 0, len(status.Files))
	for _, file := range status.Files {
		paths = append(paths, file.Path)
	}
	expected := []string{"src/staged only.go", "src/mixed.go", "src/dirty\nnewline.go", "new name.go", "conflict.go", "untracked dir/"}
	if !slices.Equal(paths, expected) {
		t.Fatalf("paths = %q", paths)
	}
	if status.Files[3].OrigPath != "old name.go" {
		t.Fatalf("rename orig = %q", status.Files[3].OrigPath)
	}

	cases := []struct {
		path      string
		filter    git.Filter
		inFilter  bool
		forFilter string
	}{
		{"src/staged only.go", git.FilterStaged, true, "staged"},
		{"src/staged only.go", git.FilterNotStaged, false, "not staged"},
		{"src/staged only.go", git.FilterMixed, false, "mixed"},
		{"src/mixed.go", git.FilterStaged, true, "staged"},
		{"src/mixed.go", git.FilterNotStaged, false, "not staged"},
		{"src/mixed.go", git.FilterMixed, true, "mixed"},
		{"src/dirty\nnewline.go", git.FilterNotStaged, true, "not staged"},
		{"src/dirty\nnewline.go", git.FilterStaged, false, "staged"},
		{"untracked dir/", git.FilterNotStaged, true, "not staged"},
		{"untracked dir/", git.FilterStaged, false, "staged"},
		{"conflict.go", git.FilterAll, true, "all"},
		{"conflict.go", git.FilterStaged, false, "staged"},
		{"conflict.go", git.FilterMixed, false, "mixed"},
	}
	for _, testCase := range cases {
		t.Run(testCase.path+"/"+testCase.forFilter, func(t *testing.T) {
			index := slices.IndexFunc(status.Files, func(file git.FileChange) bool { return file.Path == testCase.path })
			if got := testCase.filter.Matches(status.Files[index]); got != testCase.inFilter {
				t.Fatalf("Matches = %v, want %v", got, testCase.inFilter)
			}
		})
	}
}

func TestParseStatusNoUpstreamAndDetached(t *testing.T) {
	status, err := git.ParseStatus([]byte("# branch.oid 2222222222222222222222222222222222222222\x00# branch.head (detached)\x00"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !status.Branch.Detached || status.Branch.Upstream != "" || status.Branch.HasAheadBehind {
		t.Fatalf("branch = %+v", status.Branch)
	}
}

func TestStatusFiltersAgainstRealRepository(t *testing.T) {
	fixture := gittest.New(t)
	fixture.Commit("base", map[string]string{"tracked.txt": "one\n", "staged.txt": "one\n", "mixed.txt": "a\nb\nc\nd\ne\nf\ng\nh\n"})
	fixture.Write("tracked.txt", "two\n")
	fixture.Write("staged.txt", "two\n")
	fixture.Git("add", "--", "staged.txt")
	fixture.Write("mixed.txt", "A\nb\nc\nd\ne\nf\ng\nh\n")
	fixture.Git("add", "--", "mixed.txt")
	fixture.Write("mixed.txt", "A\nb\nc\nd\ne\nf\ng\nH\n")
	fixture.Write("new file.txt", "fresh\n")

	status, err := git.LoadStatus(context.Background(), fixture.Runner)
	if err != nil {
		t.Fatalf("load status: %v", err)
	}
	membership := map[string][]git.Filter{}
	for _, file := range status.Files {
		for _, filter := range git.Filters {
			if filter.Matches(file) {
				membership[file.Path] = append(membership[file.Path], filter)
			}
		}
	}
	expected := map[string][]git.Filter{
		"tracked.txt":  {git.FilterAll, git.FilterNotStaged},
		"staged.txt":   {git.FilterAll, git.FilterStaged},
		"mixed.txt":    {git.FilterAll, git.FilterStaged, git.FilterMixed},
		"new file.txt": {git.FilterAll, git.FilterNotStaged},
	}
	if len(membership) != len(expected) {
		t.Fatalf("membership = %v", membership)
	}
	for path, filters := range expected {
		if !slices.Equal(membership[path], filters) {
			t.Errorf("%s filters = %v, want %v", path, membership[path], filters)
		}
	}
}
