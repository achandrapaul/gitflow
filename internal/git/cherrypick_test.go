package git_test

import (
	"context"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
)

func newCherryPicker(fixture *gittest.Fixture) *git.CherryPicker {
	return git.NewCherryPicker(fixture.Runner, git.NewOperationDetector(fixture.Runner))
}

func sidePicks(fixture *gittest.Fixture) (first, second, conflicting, third string) {
	fixture.Commit("base", gittest.ConflictBase)
	fixture.Git("switch", "-q", "-c", "side")
	first = fixture.Commit("first", map[string]string{"first.txt": "first\n"})
	second = fixture.Commit("second", map[string]string{"second.txt": "second\n"})
	conflicting = fixture.Commit("conflicting", gittest.FeatureOne)
	third = fixture.Commit("third", map[string]string{"third.txt": "third\n"})
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", gittest.ConflictMain)
	return first, second, conflicting, third
}

func TestCherryPickerPick(t *testing.T) {
	t.Run("T15 noncontiguous commits land in queue order", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		first, _, _, third := sidePicks(fixture)
		head := revParse(fixture, "HEAD")
		outcome, err := newCherryPicker(fixture).Pick(context.Background(), []string{third, first})
		if err != nil || outcome.Paused || len(outcome.Completed) != 2 {
			t.Fatalf("Pick = %+v, %v", outcome, err)
		}
		if subjects := fixture.Git("log", "--format=%s", "--reverse", head+"..HEAD"); subjects != "third\nfirst\n" {
			t.Fatalf("picked subjects %q, want third then first", subjects)
		}
		if strings.Join(outcome.Completed, " ") != strings.TrimSpace(strings.ReplaceAll(fixture.Git("rev-list", "--reverse", head+"..HEAD"), "\n", " ")) {
			t.Fatalf("Completed %v does not match new commits", outcome.Completed)
		}
	})
	t.Run("T127 conflict partway pauses with real sequencer progress", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		first, _, conflicting, third := sidePicks(fixture)
		outcome, err := newCherryPicker(fixture).Pick(context.Background(), []string{first, conflicting, third})
		if err != nil || !outcome.Paused || len(outcome.Completed) != 1 {
			t.Fatalf("Pick = %+v, %v; want paused after one commit", outcome, err)
		}
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Kind != git.OperationCherryPick || operation.Commit != conflicting || operation.Completed != 1 {
			t.Fatalf("operation after restart = %+v, %v", operation, err)
		}
		if len(operation.Remaining) != 1 || !strings.HasPrefix(third, operation.Remaining[0]) {
			t.Fatalf("remaining = %v, want %s", operation.Remaining, third)
		}
	})
	t.Run("T27 empty pick pauses natively without a commit", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		fixture.Commit("base", gittest.ConflictBase)
		fixture.Git("switch", "-q", "-c", "side")
		duplicate := fixture.Commit("duplicate", gittest.ConflictMain)
		fixture.Git("switch", "-q", "main")
		fixture.Commit("main", gittest.ConflictMain)
		head := revParse(fixture, "HEAD")
		outcome, err := newCherryPicker(fixture).Pick(context.Background(), []string{duplicate})
		if err != nil || !outcome.Paused || len(outcome.Completed) != 0 || revParse(fixture, "HEAD") != head {
			t.Fatalf("empty Pick = %+v, %v; want paused, no commit", outcome, err)
		}
		operation, err := git.NewOperationDetector(fixture.Runner).Active(context.Background())
		if err != nil || operation.Kind != git.OperationCherryPick || !operation.SkipAllowed {
			t.Fatalf("operation = %+v, %v", operation, err)
		}
	})
	t.Run("unknown commit fails without pausing", func(t *testing.T) {
		fixture := gittest.NewFixture(t)
		sidePicks(fixture)
		before := fixture.Snapshot()
		if _, err := newCherryPicker(fixture).Pick(context.Background(), []string{strings.Repeat("0", 40)}); err == nil {
			t.Fatal("Pick of missing commit succeeded")
		}
		if fixture.Snapshot() != before {
			t.Fatal("failed Pick changed Git state")
		}
	})
}
