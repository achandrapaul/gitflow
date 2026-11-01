package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	notAncestorExit       = 1
	rangeFieldsPerCommit  = 3
	rangeFormat           = "--format=%H%x00%P%x00%s"
	todoFilePattern       = "gitflow-rebase-todo-*"
	sequenceEditorFormat  = "GIT_SEQUENCE_EDITOR=cp -- '%s'"
	sequenceEditorQuote   = "'"
	remoteShortNameFormat = "--format=%(if)%(symref)%(then)%(else)%(refname:short)%(end)"
)

var (
	ErrRebaseDetached        = errors.New("HEAD is not on a local branch")
	ErrRebaseBaseOffBranch   = errors.New("base is not an ancestor of the checked-out branch")
	ErrRebaseEmptyRange      = errors.New("no commits between base and HEAD")
	ErrRebaseMergeInRange    = errors.New("the range contains a merge commit")
	ErrRebaseOrphanCombine   = errors.New("squash or fixup has no earlier kept commit to combine with")
	ErrRebaseTrackedChanges  = errors.New("tracked files have uncommitted changes")
	ErrRebaseOperationActive = errors.New("a Git operation is paused")
	ErrRebasePlanMismatch    = errors.New("the plan does not list exactly the commits in the range")
)

type TodoAction int

const (
	TodoPick TodoAction = iota
	TodoReword
	TodoEdit
	TodoSquash
	TodoFixup
	TodoDrop
)

var todoCommands = map[TodoAction]string{
	TodoPick:   "pick",
	TodoReword: "reword",
	TodoEdit:   "edit",
	TodoSquash: "squash",
	TodoFixup:  "fixup",
	TodoDrop:   "drop",
}

func (a TodoAction) String() string { return todoCommands[a] }

type TodoStep struct {
	Action  TodoAction
	OID     string
	Subject string
}

type RebaseRange struct {
	Branch string
	Head   string
	Base   string
	Steps  []TodoStep
}

type RebaseRun struct {
	Command  *exec.Cmd
	Base     string
	todoPath string
}

type Rebaser struct {
	runner     *Runner
	status     *StatusReader
	operations *OperationDetector
}

func NewRebaser(runner *Runner, status *StatusReader, operations *OperationDetector) *Rebaser {
	return &Rebaser{runner: runner, status: status, operations: operations}
}

func (r *Rebaser) Range(ctx context.Context, base string) (RebaseRange, error) {
	status, err := r.status.Snapshot(ctx)
	if err != nil {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, err)
	}
	if status.Unborn || status.Detached {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, ErrRebaseDetached)
	}
	_, err = r.runner.Read(ctx, "merge-base", "--is-ancestor", base, status.Head)
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == notAncestorExit {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, ErrRebaseBaseOffBranch)
	}
	if err != nil {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, err)
	}
	output, err := r.runner.Read(ctx, "log", "--reverse", "-z", rangeFormat, base+".."+status.Head)
	if err != nil {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, err)
	}
	if len(output) == 0 {
		return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w", base, ErrRebaseEmptyRange)
	}
	reviewed := RebaseRange{Branch: status.Branch, Head: status.Head, Base: base}
	for fields := range slices.Chunk(splitNulTerminated(output), rangeFieldsPerCommit) {
		if len(fields) != rangeFieldsPerCommit {
			return RebaseRange{}, fmt.Errorf("read rebase range from %s: malformed log record %q", base, fields)
		}
		if len(strings.Fields(fields[1])) > 1 {
			return RebaseRange{}, fmt.Errorf("read rebase range from %s: %w: %s", base, ErrRebaseMergeInRange, fields[0])
		}
		reviewed.Steps = append(reviewed.Steps, TodoStep{Action: TodoPick, OID: fields[0], Subject: fields[2]})
	}
	return reviewed, nil
}

func (r *Rebaser) Validate(steps []TodoStep) error {
	kept := false
	for position, step := range steps {
		combines := step.Action == TodoSquash || step.Action == TodoFixup
		if combines && !kept {
			return fmt.Errorf("validate todo row %d: %w", position+1, ErrRebaseOrphanCombine)
		}
		kept = kept || step.Action != TodoDrop
	}
	return nil
}

func (r *Rebaser) Published(ctx context.Context, reviewed RebaseRange) ([]string, error) {
	args := []string{"for-each-ref", remoteShortNameFormat}
	for _, step := range reviewed.Steps {
		args = append(args, "--contains", step.OID)
	}
	output, err := r.runner.Read(ctx, append(args, remoteRefPrefix)...)
	if err != nil {
		return nil, fmt.Errorf("find remote refs containing commits after %s: %w", reviewed.Base, err)
	}
	return strings.Fields(string(output)), nil
}

func (r *Rebaser) Prepare(ctx context.Context, reviewed RebaseRange) (RebaseRun, error) {
	operation, err := r.operations.Active(ctx)
	if err != nil {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, err)
	}
	if operation.Active() {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, ErrRebaseOperationActive)
	}
	status, err := r.status.Snapshot(ctx)
	if err != nil {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, err)
	}
	if !status.TrackedClean() {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w: %s", reviewed.Base, ErrRebaseTrackedChanges, strings.Join(slices.Concat(status.TrackedDirtyPaths, status.Unmerged), ", "))
	}
	current, err := r.Range(ctx, reviewed.Base)
	if err != nil {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, err)
	}
	if current.Head != reviewed.Head || current.Branch != reviewed.Branch {
		return RebaseRun{}, fmt.Errorf("prepare rebase of %s onto %s: %w", reviewed.Branch, reviewed.Base, ErrStaleRef)
	}
	var currentOIDs, plannedOIDs []string
	for _, step := range current.Steps {
		currentOIDs = append(currentOIDs, step.OID)
	}
	for _, step := range reviewed.Steps {
		plannedOIDs = append(plannedOIDs, step.OID)
	}
	slices.Sort(currentOIDs)
	slices.Sort(plannedOIDs)
	if !slices.Equal(currentOIDs, plannedOIDs) {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, ErrRebasePlanMismatch)
	}
	if err := r.Validate(reviewed.Steps); err != nil {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: %w", reviewed.Base, err)
	}
	var todo strings.Builder
	for _, step := range reviewed.Steps {
		fmt.Fprintf(&todo, "%s %s\n", step.Action, step.OID)
	}
	todoFile, err := os.CreateTemp("", todoFilePattern)
	if err != nil {
		return RebaseRun{}, fmt.Errorf("prepare rebase onto %s: create todo file: %w", reviewed.Base, err)
	}
	todoPath := todoFile.Name()
	_, writeErr := todoFile.WriteString(todo.String())
	if err := errors.Join(writeErr, todoFile.Close()); err != nil {
		return RebaseRun{}, errors.Join(fmt.Errorf("prepare rebase onto %s: write todo file %s: %w", reviewed.Base, todoPath, err), os.Remove(todoPath))
	}
	if strings.Contains(todoPath, sequenceEditorQuote) {
		return RebaseRun{}, errors.Join(fmt.Errorf("prepare rebase onto %s: todo path %s cannot be quoted for Git's sequence editor", reviewed.Base, todoPath), os.Remove(todoPath))
	}
	command := exec.CommandContext(ctx, r.runner.gitPath, "rebase", "-i", reviewed.Base)
	command.Dir = r.runner.workTree
	command.Env = append(append(os.Environ(), r.runner.extraEnv...), fmt.Sprintf(sequenceEditorFormat, todoPath))
	return RebaseRun{Command: command, Base: reviewed.Base, todoPath: todoPath}, nil
}

func (r *Rebaser) Finish(ctx context.Context, run RebaseRun) (SequenceOutcome, error) {
	if err := os.Remove(run.todoPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return SequenceOutcome{}, fmt.Errorf("remove rebase todo file %s: %w", run.todoPath, err)
	}
	created, err := r.runner.Read(ctx, "rev-list", "--reverse", run.Base+"..HEAD")
	if err != nil {
		return SequenceOutcome{}, fmt.Errorf("read commits rewritten onto %s: %w", run.Base, err)
	}
	operation, err := r.operations.Active(ctx)
	if err != nil {
		return SequenceOutcome{}, fmt.Errorf("read rebase state after rewriting onto %s: %w", run.Base, err)
	}
	return SequenceOutcome{Completed: strings.Fields(string(created)), Paused: operation.Kind == OperationRebase}, nil
}
