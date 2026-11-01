package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var operationMarkers = []struct {
	name string
	kind OperationKind
}{
	{rebaseMergeDir, OperationRebase},
	{rebaseApplyDir, OperationRebase},
	{"CHERRY_PICK_HEAD", OperationCherryPick},
	{"REVERT_HEAD", OperationRevert},
	{"MERGE_HEAD", OperationMerge},
}

type OperationKind int

const (
	OperationNone OperationKind = iota
	OperationCherryPick
	OperationRebase
	OperationMerge
	OperationRevert
)

type Operation struct {
	Kind        OperationKind
	Commit      string
	Branch      string
	Onto        string
	Step        int
	Total       int
	Completed   int
	Remaining   []string
	StopAction  string
	SkipAllowed bool
}

func (o Operation) Active() bool { return o.Kind != OperationNone }

const (
	rebaseMergeDir          = "rebase-merge"
	rebaseApplyDir          = "rebase-apply"
	rebaseHeadFile          = "REBASE_HEAD"
	sequencerTodo           = "sequencer/todo"
	sequencerHead           = "sequencer/head"
	rebaseOntoFile          = "onto"
	rebaseHeadNameFile      = "head-name"
	rebaseDoneFile          = "done"
	detailCommit            = "commit"
	detailStep              = "step"
	detailTotal             = "total"
	detailOnto              = "onto"
	detailBranch            = "branch"
	detailTodo              = "todo"
	detailDone              = "done"
	detailSequenceStart     = "sequence start"
	todoComment             = "#"
	todoMinFields           = 2
	nonInteractiveGitEditor = "GIT_EDITOR=true"
)

var rebaseProgressFiles = map[string][2]string{
	rebaseMergeDir: {"msgnum", "end"},
	rebaseApplyDir: {"next", "last"},
}

var (
	ErrOperationChanged     = errors.New("the paused operation changed since review")
	ErrUnresolvedConflicts  = errors.New("conflicted files remain unresolved")
	ErrSkipUnsupported      = errors.New("this operation has no native skip")
	ErrOperationUnsupported = errors.New("this operation is not supported by GitFlow")
)

var operationCommands = map[OperationKind]string{
	OperationCherryPick: "cherry-pick",
	OperationRebase:     "rebase",
	OperationMerge:      "merge",
}

type OperationDetector struct {
	runner *Runner
}

func NewOperationDetector(runner *Runner) *OperationDetector {
	return &OperationDetector{runner: runner}
}

func (o *OperationDetector) Active(ctx context.Context) (Operation, error) {
	names := []string{rebaseHeadFile, sequencerTodo, sequencerHead}
	for _, marker := range operationMarkers {
		names = append(names, marker.name)
	}
	paths, err := o.runner.GitPaths(ctx, names...)
	if err != nil {
		return Operation{}, fmt.Errorf("detect paused operation: %w", err)
	}
	rebaseHeadPath, todoPath, sequenceStartPath, markerPaths := paths[0], paths[1], paths[2], paths[3:]
	for index, path := range markerPaths {
		marker := operationMarkers[index]
		_, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Operation{}, fmt.Errorf("detect paused operation from %s: %w", marker.name, err)
		}
		operation := Operation{Kind: marker.kind, SkipAllowed: marker.kind == OperationRebase || marker.kind == OperationCherryPick}
		var details map[string]string
		switch marker.kind {
		case OperationRebase:
			progress := rebaseProgressFiles[marker.name]
			details = map[string]string{detailCommit: rebaseHeadPath, detailStep: filepath.Join(path, progress[0]), detailTotal: filepath.Join(path, progress[1]), detailOnto: filepath.Join(path, rebaseOntoFile), detailBranch: filepath.Join(path, rebaseHeadNameFile), detailDone: filepath.Join(path, rebaseDoneFile)}
		case OperationCherryPick:
			details = map[string]string{detailCommit: path, detailTodo: todoPath, detailSequenceStart: sequenceStartPath}
		default:
			details = map[string]string{detailCommit: path}
		}
		values := map[string]string{}
		for key, detailPath := range details {
			content, err := os.ReadFile(detailPath)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return Operation{}, fmt.Errorf("read %s details from %s: %w", marker.name, detailPath, err)
			}
			values[key] = strings.TrimSpace(string(content))
		}
		operation.Commit, _, _ = strings.Cut(values[detailCommit], "\n")
		operation.Onto = values[detailOnto]
		operation.Branch = strings.TrimPrefix(values[detailBranch], localRefPrefix)
		operation.Step, _ = strconv.Atoi(values[detailStep])
		operation.Total, _ = strconv.Atoi(values[detailTotal])
		if sequenceStart := values[detailSequenceStart]; sequenceStart != "" {
			completed, err := o.runner.Read(ctx, "rev-list", "--count", sequenceStart+"..HEAD")
			if err != nil {
				return Operation{}, fmt.Errorf("count completed cherry-picks since %s: %w", sequenceStart, err)
			}
			operation.Completed, _ = strconv.Atoi(strings.TrimSpace(string(completed)))
		}
		for _, line := range strings.Split(values[detailDone], "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && !strings.HasPrefix(fields[0], todoComment) {
				operation.StopAction = fields[0]
			}
		}
		todoLines := strings.Split(values[detailTodo], "\n")
		for _, line := range todoLines[min(1, len(todoLines)):] {
			fields := strings.Fields(line)
			if len(fields) >= todoMinFields && !strings.HasPrefix(fields[0], todoComment) {
				operation.Remaining = append(operation.Remaining, fields[1])
			}
		}
		return operation, nil
	}
	return Operation{}, nil
}

type OperationController struct {
	runner     *Runner
	status     *StatusReader
	operations *OperationDetector
}

func NewOperationController(runner *Runner, status *StatusReader, operations *OperationDetector) *OperationController {
	return &OperationController{runner: runner, status: status, operations: operations}
}

func (c *OperationController) run(ctx context.Context, kind OperationKind, control string) error {
	operation, err := c.operations.Active(ctx)
	if err != nil {
		return fmt.Errorf("%s paused operation: %w", control, err)
	}
	if operation.Kind != kind {
		return fmt.Errorf("%s paused operation: %w", control, ErrOperationChanged)
	}
	command, supported := operationCommands[kind]
	if !supported {
		return fmt.Errorf("%s paused operation: %w", control, ErrOperationUnsupported)
	}
	args := []string{command, "--" + control}
	if _, err := c.runner.Execute(ctx, Invocation{Args: args, Env: []string{nonInteractiveGitEditor}, CombineOutput: true}); err != nil {
		return fmt.Errorf("%s %s: %w", control, command, err)
	}
	return nil
}

func (c *OperationController) Continue(ctx context.Context, kind OperationKind) error {
	status, err := c.status.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("continue paused operation: %w", err)
	}
	if len(status.Unmerged) > 0 {
		return fmt.Errorf("continue paused operation: %w: %s", ErrUnresolvedConflicts, strings.Join(status.Unmerged, ", "))
	}
	return c.run(ctx, kind, "continue")
}

func (c *OperationController) Skip(ctx context.Context, kind OperationKind) error {
	if kind == OperationMerge {
		return fmt.Errorf("skip paused merge: %w", ErrSkipUnsupported)
	}
	return c.run(ctx, kind, "skip")
}

func (c *OperationController) Abort(ctx context.Context, kind OperationKind) error {
	return c.run(ctx, kind, "abort")
}
