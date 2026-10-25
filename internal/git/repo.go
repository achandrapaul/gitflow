package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNotWorkTree = errors.New("not inside a Git working tree")

type Repository struct {
	WorkTree string
	GitDir   string
}

func Discover(ctx context.Context, startDir string) (Repository, string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return Repository{}, "", fmt.Errorf("locate git executable: %w", err)
	}
	probe := NewRunner(gitPath, startDir, nil)
	output, err := probe.Read(ctx, "rev-parse", "--is-inside-work-tree", "--path-format=absolute", "--show-toplevel", "--git-dir")
	if err != nil {
		return Repository{}, "", fmt.Errorf("discover repository from %s: %w: %w", startDir, ErrNotWorkTree, err)
	}
	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "true" {
		return Repository{}, "", fmt.Errorf("discover repository from %s: %w", startDir, ErrNotWorkTree)
	}
	return Repository{WorkTree: lines[1], GitDir: lines[2]}, gitPath, nil
}
