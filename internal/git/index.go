package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	literalPathspecs = "--literal-pathspecs"
	unbornHeadExit   = 1
	headRevision     = "HEAD"
	messageFormat    = "--format=%B"
	parentsFormat    = "--format=%P"
)

var (
	ErrStaleReview    = errors.New("staged index or HEAD changed since review")
	ErrUnexpectedHead = errors.New("HEAD moved to a commit this action did not create")
)

type CommitRequest struct {
	Message      string
	ReviewedTree string
	ReviewedHead string
	NoVerify     bool
	Amend        bool
}

type CommitResult struct {
	OID            string
	Tree           string
	Message        string
	TreeChanges    []string
	MessageChanged bool
}

func (r CommitResult) MatchesReview() bool {
	return len(r.TreeChanges) == 0 && !r.MessageChanged
}

type Indexer struct {
	runner *Runner
}

func NewIndexer(runner *Runner) *Indexer {
	return &Indexer{runner: runner}
}

func (i *Indexer) Stage(ctx context.Context, paths []string) error {
	if _, err := i.runner.Run(ctx, append([]string{literalPathspecs, "add", "-A", "--"}, paths...)...); err != nil {
		return fmt.Errorf("stage %s: %w", strings.Join(paths, ", "), err)
	}
	return nil
}

func (i *Indexer) Head(ctx context.Context) (string, error) {
	output, err := i.runner.Read(ctx, "rev-parse", "--verify", "-q", headRevision)
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == unbornHeadExit {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (i *Indexer) Unstage(ctx context.Context, paths []string) error {
	command := []string{literalPathspecs, "restore", "--staged", "--"}
	head, err := i.Head(ctx)
	if err != nil {
		return fmt.Errorf("unstage %s: %w", strings.Join(paths, ", "), err)
	}
	if head == "" {
		command = []string{literalPathspecs, "rm", "--cached", "-f", "-q", "--"}
	}
	if _, err := i.runner.Run(ctx, append(command, paths...)...); err != nil {
		return fmt.Errorf("unstage %s: %w", strings.Join(paths, ", "), err)
	}
	return nil
}

func (i *Indexer) StagePatch(ctx context.Context, patch []byte) error {
	return errNotImplemented
}

func (i *Indexer) UnstagePatch(ctx context.Context, patch []byte) error {
	return errNotImplemented
}

func (i *Indexer) Discard(ctx context.Context, paths []string) error {
	return errNotImplemented
}

func (i *Indexer) IndexTree(ctx context.Context) (string, error) {
	output, err := i.runner.Run(ctx, "write-tree")
	if err != nil {
		return "", fmt.Errorf("write index tree: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (i *Indexer) Commit(ctx context.Context, request CommitRequest) (CommitResult, error) {
	head, err := i.Head(ctx)
	if err != nil {
		return CommitResult{}, fmt.Errorf("commit: %w", err)
	}
	tree, err := i.IndexTree(ctx)
	if err != nil {
		return CommitResult{}, fmt.Errorf("commit: %w", err)
	}
	if head != request.ReviewedHead || tree != request.ReviewedTree {
		return CommitResult{}, fmt.Errorf("commit: %w", ErrStaleReview)
	}
	wantParents := head
	if request.Amend {
		parents, err := i.runner.Read(ctx, "log", "-1", parentsFormat, head)
		if err != nil {
			return CommitResult{}, fmt.Errorf("commit: read parents of %s: %w", head, err)
		}
		wantParents = strings.TrimSpace(string(parents))
	}
	args := []string{"commit", "-q", "-F", "-"}
	if request.NoVerify {
		args = append(args, "--no-verify")
	}
	if request.Amend {
		args = append(args, "--amend")
	}
	if _, err := i.runner.Execute(ctx, Invocation{Args: args, Stdin: []byte(request.Message)}); err != nil {
		return CommitResult{}, fmt.Errorf("commit: %w", err)
	}
	result := CommitResult{}
	if result.OID, err = i.Head(ctx); err != nil {
		return CommitResult{}, fmt.Errorf("commit: verify: %w", err)
	}
	parents, err := i.runner.Read(ctx, "log", "-1", parentsFormat, result.OID)
	if err != nil {
		return result, fmt.Errorf("commit: verify parents of %s: %w", result.OID, err)
	}
	if result.OID == head || strings.TrimSpace(string(parents)) != wantParents {
		return result, fmt.Errorf("commit: HEAD is %s: %w", result.OID, ErrUnexpectedHead)
	}
	actualTree, err := i.runner.Read(ctx, "rev-parse", result.OID+"^{tree}")
	if err != nil {
		return result, fmt.Errorf("commit: verify tree of %s: %w", result.OID, err)
	}
	result.Tree = strings.TrimSpace(string(actualTree))
	message, err := i.runner.Read(ctx, "log", "-1", messageFormat, result.OID)
	if err != nil {
		return result, fmt.Errorf("commit: verify message of %s: %w", result.OID, err)
	}
	result.Message = strings.TrimSuffix(string(message), "\n")
	reviewedMessage, err := i.runner.Execute(ctx, Invocation{Args: []string{"stripspace"}, Stdin: []byte(request.Message)})
	if err != nil {
		return result, fmt.Errorf("commit: normalise reviewed message: %w", err)
	}
	result.MessageChanged = result.Message != string(reviewedMessage)
	if result.Tree == request.ReviewedTree {
		return result, nil
	}
	changes, err := i.runner.Read(ctx, "diff-tree", "-r", "--name-status", request.ReviewedTree, result.Tree)
	if err != nil {
		return result, fmt.Errorf("commit: compare reviewed tree with %s: %w", result.Tree, err)
	}
	result.TreeChanges = strings.Split(strings.TrimSuffix(string(changes), "\n"), "\n")
	return result, nil
}
