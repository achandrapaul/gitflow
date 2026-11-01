package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	discoveryLineCount   = 3
	insideWorkTreeAnswer = "true"
)

var ErrNotWorkTree = errors.New("not inside a Git working tree")

type Repository struct {
	WorkTree string
	GitDir   string
}

func Discover(ctx context.Context, runner *Runner) (Repository, error) {
	output, err := runner.Read(ctx, "rev-parse", "--is-inside-work-tree", "--show-toplevel", "--absolute-git-dir")
	if err != nil {
		return Repository{}, fmt.Errorf("discover repository from %q: %w: %w", runner.WorkTree(), ErrNotWorkTree, err)
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != discoveryLineCount || lines[0] != insideWorkTreeAnswer {
		return Repository{}, fmt.Errorf("discover repository from %q: %w", runner.WorkTree(), ErrNotWorkTree)
	}
	return Repository{WorkTree: lines[1], GitDir: lines[2]}, nil
}

func (r *Runner) GitPaths(ctx context.Context, names ...string) ([]string, error) {
	args := []string{"rev-parse"}
	for _, name := range names {
		args = append(args, "--git-path", name)
	}
	output, err := r.Read(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("locate %s in the Git directory: %w", strings.Join(names, ", "), err)
	}
	paths := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(paths) != len(names) {
		return nil, fmt.Errorf("locate %s in the Git directory: unexpected output %q", strings.Join(names, ", "), output)
	}
	for index, path := range paths {
		if !filepath.IsAbs(path) {
			paths[index] = filepath.Join(r.workTree, path)
		}
	}
	return paths, nil
}
