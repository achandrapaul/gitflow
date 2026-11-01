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

const fatalExitCode = 128

func TestRunnerExecute(t *testing.T) {
	fixture := gittest.NewFixture(t)
	resolvedDir, err := filepath.EvalSymlinks(fixture.Dir)
	if err != nil {
		t.Fatalf("resolve fixture dir: %v", err)
	}

	t.Run("runs in the explicit work tree", func(t *testing.T) {
		output, err := fixture.Runner.Run(context.Background(), "rev-parse", "--show-toplevel")
		if err != nil {
			t.Fatalf("rev-parse: %v", err)
		}
		if got := strings.TrimSuffix(string(output), "\n"); got != resolvedDir {
			t.Fatalf("toplevel = %q, want %q", got, resolvedDir)
		}
	})

	t.Run("reports exit code and stderr", func(t *testing.T) {
		_, err := fixture.Runner.Run(context.Background(), "rev-parse", "--verify", "missing-ref")
		var commandErr *git.CommandError
		if !errors.As(err, &commandErr) {
			t.Fatalf("error = %v, want *CommandError", err)
		}
		if commandErr.ExitCode != fatalExitCode || !strings.Contains(commandErr.Stderr, "fatal") {
			t.Fatalf("exit %d stderr %q", commandErr.ExitCode, commandErr.Stderr)
		}
	})

	t.Run("read disables optional locks", func(t *testing.T) {
		_, err := fixture.Runner.Read(context.Background(), "rev-parse", "--verify", "missing-ref")
		var commandErr *git.CommandError
		if !errors.As(err, &commandErr) {
			t.Fatalf("error = %v, want *CommandError", err)
		}
		if commandErr.Args[0] != "--no-optional-locks" {
			t.Fatalf("args = %q", commandErr.Args)
		}
	})

	t.Run("honours cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := fixture.Runner.Run(ctx, "status")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})

	t.Run("passes stdin", func(t *testing.T) {
		output, err := fixture.Runner.Execute(context.Background(), git.Invocation{Args: []string{"hash-object", "--stdin"}, Stdin: []byte("payload\n")})
		if err != nil {
			t.Fatalf("hash-object: %v", err)
		}
		if len(strings.TrimSuffix(string(output), "\n")) == 0 {
			t.Fatal("hash-object returned no object id")
		}
	})
}

func TestRunnerUnusualPaths(t *testing.T) {
	unusualPaths := []string{
		"with space.txt",
		"trailing space ",
		"ünïcødé/文件.txt",
		"$(touch injected)",
		"semi;colon|pipe&amp.txt",
		"-leading-dash.txt",
		"quote\"and'apostrophe.txt",
		"tab\there.txt",
		"new\nline.txt",
	}
	for _, path := range unusualPaths {
		t.Run(path, func(t *testing.T) {
			fixture := gittest.NewFixture(t)
			fixture.Write(path, "content\n")
			fixture.Git("add", "--", path)

			output := fixture.Git("ls-files", "-z")
			listed := strings.Split(strings.TrimSuffix(output, "\x00"), "\x00")
			if !slices.Equal(listed, []string{path}) {
				t.Fatalf("ls-files -z = %q, want %q", listed, path)
			}
			if _, err := os.Stat(filepath.Join(fixture.Dir, "injected")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("argv was interpreted by a shell: %v", err)
			}
		})
	}
}
