package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrMergeCommitNeedsMainline = errors.New("merge commits need an explicitly chosen mainline parent")
	ErrMergeInRange             = errors.New("range contains merge commits")
	ErrNotAncestor              = errors.New("base is not an ancestor of HEAD")
	ErrSquashWithoutPick        = errors.New("squash/fixup needs a preceding picked commit")
	ErrEmptyPlan                = errors.New("plan has no commits")
)

func CherryPickInvocation(commits []Commit) (Invocation, error) {
	if len(commits) == 0 {
		return Invocation{}, ErrEmptyPlan
	}
	args := []string{"cherry-pick"}
	for _, commit := range commits {
		if commit.IsMerge() {
			return Invocation{}, fmt.Errorf("cherry-pick %s: %w", commit.Short(), ErrMergeCommitNeedsMainline)
		}
		args = append(args, commit.OID)
	}
	if err := ValidateOIDs(args[1:]); err != nil {
		return Invocation{}, err
	}
	return Invocation{Args: args}, nil
}

func PatchAlreadyInHead(ctx context.Context, runner *Runner, commit Commit) (bool, error) {
	if len(commit.Parents) != 1 {
		return false, nil
	}
	output, err := runner.Read(ctx, "cherry", "HEAD", commit.OID, commit.Parents[0])
	if err != nil {
		return false, fmt.Errorf("check patch equivalence of %s: %w", commit.Short(), err)
	}
	return strings.HasPrefix(strings.TrimSpace(string(output)), "-"), nil
}

func ChangedFiles(ctx context.Context, runner *Runner, oid string) (string, error) {
	if err := ValidateOIDs([]string{oid}); err != nil {
		return "", err
	}
	output, err := runner.Read(ctx, "show", "--no-color", "--stat", "--format=", oid, "--")
	if err != nil {
		return "", fmt.Errorf("list files changed by %s: %w", oid, err)
	}
	return string(output), nil
}

type TodoAction string

const (
	TodoPick   TodoAction = "pick"
	TodoReword TodoAction = "reword"
	TodoEdit   TodoAction = "edit"
	TodoSquash TodoAction = "squash"
	TodoFixup  TodoAction = "fixup"
	TodoDrop   TodoAction = "drop"
)

type TodoRow struct {
	Action TodoAction
	Commit Commit
}

func RebaseRange(ctx context.Context, runner *Runner, onto string) ([]Commit, error) {
	if err := ValidateOIDs([]string{onto}); err != nil {
		return nil, err
	}
	if _, err := runner.Read(ctx, "merge-base", "--is-ancestor", onto, "HEAD"); err != nil {
		return nil, fmt.Errorf("check rebase base %s: %w", onto[:7], ErrNotAncestor)
	}
	commits, err := ListRange(ctx, runner, onto, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("list rebase range: %w", err)
	}
	for _, commit := range commits {
		if commit.IsMerge() {
			return nil, fmt.Errorf("list rebase range: %s: %w", commit.Short(), ErrMergeInRange)
		}
	}
	return commits, nil
}

func ListRange(ctx context.Context, runner *Runner, from, to string) ([]Commit, error) {
	if strings.HasPrefix(from, "-") || strings.HasPrefix(to, "-") {
		return nil, fmt.Errorf("list %s..%s: %w", from, to, ErrInvalidName)
	}
	output, err := runner.Read(ctx, "log", "--reverse", "--topo-order", "-z", "--no-color", commitFormat, from+".."+to, "--")
	if err != nil {
		return nil, fmt.Errorf("list %s..%s: %w", from, to, err)
	}
	return parseCommitRecords(output)
}

func ValidateTodo(rows []TodoRow) error {
	if len(rows) == 0 {
		return ErrEmptyPlan
	}
	for _, row := range rows {
		if row.Action == TodoDrop {
			continue
		}
		if row.Action == TodoSquash || row.Action == TodoFixup {
			return fmt.Errorf("%s %s: %w", row.Action, row.Commit.Short(), ErrSquashWithoutPick)
		}
		return nil
	}
	return nil
}

func StartRebaseInvocation(onto string, rows []TodoRow) (Invocation, func(), error) {
	if err := ValidateTodo(rows); err != nil {
		return Invocation{}, nil, err
	}
	oids := []string{onto}
	var todo strings.Builder
	for _, row := range rows {
		oids = append(oids, row.Commit.OID)
		fmt.Fprintf(&todo, "%s %s %s\n", row.Action, row.Commit.OID, row.Commit.Subject)
	}
	if err := ValidateOIDs(oids); err != nil {
		return Invocation{}, nil, err
	}
	handle, err := os.CreateTemp("", "gitflow-todo-")
	if err != nil {
		return Invocation{}, nil, fmt.Errorf("create rebase todo: %w", err)
	}
	defer handle.Close()
	cleanup := func() { os.Remove(handle.Name()) }
	if _, err := handle.WriteString(todo.String()); err != nil {
		cleanup()
		return Invocation{}, nil, fmt.Errorf("write rebase todo: %w", err)
	}
	return Invocation{
		Args: []string{"rebase", "--interactive", "--no-update-refs", onto},
		Env:  []string{"GIT_SEQUENCE_EDITOR=cp -- '" + strings.ReplaceAll(handle.Name(), "'", `'\''`) + "'"},
	}, cleanup, nil
}
