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

const (
	missingStashObject = "1111111111111111111111111111111111111111\n"
	stashRefFileMode   = 0o644
)

func TestStashList(t *testing.T) {
	fixture := gittest.NewFixture(t)
	fixture.Commit("base", map[string]string{"a.txt": "a\n"})
	stasher := git.NewStasher(fixture.Runner)

	empty, err := stasher.List(context.Background())
	if err != nil || len(empty) != 0 {
		t.Fatalf("T146 empty stash list = %+v, %v", empty, err)
	}

	fixture.Write("a.txt", "first\n")
	fixture.Git("stash", "push", "-q")
	fixture.Write("a.txt", "second\n")
	fixture.Git("stash", "push", "-q", "-m", "named work")
	before := fixture.Snapshot()
	beforeList := fixture.Git("stash", "list")

	stashes, err := stasher.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	wantOIDs := []string{strings.TrimSpace(fixture.Git("rev-parse", "stash@{0}")), strings.TrimSpace(fixture.Git("rev-parse", "stash@{1}"))}
	wantMessages := []string{"On main: named work", "WIP on main: "}
	if len(stashes) != len(wantOIDs) {
		t.Fatalf("T145 listed %d stashes, want %d: %+v", len(stashes), len(wantOIDs), stashes)
	}
	for position, stash := range stashes {
		if stash.Index != position || stash.Ref() != fmt.Sprintf("stash@{%d}", position) || stash.OID != wantOIDs[position] {
			t.Fatalf("stash %d = %+v, want index %d OID %s", position, stash, position, wantOIDs[position])
		}
		if !strings.HasPrefix(stash.Message, wantMessages[position]) || stash.Date.IsZero() {
			t.Fatalf("stash %d message %q date %v", position, stash.Message, stash.Date)
		}
	}
	if fixture.Snapshot() != before || fixture.Git("stash", "list") != beforeList {
		t.Fatal("T145 listing stashes changed Git state")
	}

	if err := os.WriteFile(filepath.Join(fixture.Dir, ".git", "refs", "stash"), []byte(missingStashObject), stashRefFileMode); err != nil {
		t.Fatalf("corrupt refs/stash: %v", err)
	}
	broken, err := stasher.List(context.Background())
	if err == nil || len(broken) != 0 {
		t.Fatalf("T146 bad refs/stash = %+v, %v; want error and no entries", broken, err)
	}
}
