package recovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/achandrapaul/gitflow/internal/git"
)

type BlockerKind int

const (
	BlockUnmerged BlockerKind = iota
	BlockSequencer
	BlockUntrackedCollision
	BlockStale
	BlockMergeCommit
	BlockDirtyTracked
	BlockUnsupportedChange
	BlockPatchDoesNotApply
)

const (
	reviewedHeadLabel = "HEAD"
	mergeParentCount  = 2
	reasonSeparator   = "; "
	pathSeparator     = ", "
	unsupportedPath   = "%s (%s)"
	trackedChanges    = "tracked changes: "
)

var ErrExecutionBlocked = errors.New("execution is blocked")

var blockerReasons = map[BlockerKind]string{
	BlockUnmerged:           "unmerged files",
	BlockSequencer:          "a paused operation is in progress",
	BlockUntrackedCollision: "untracked files would be overwritten",
	BlockStale:              "changed since preview",
	BlockMergeCommit:        "merge commits need a mainline parent, which GitFlow does not support",
	BlockDirtyTracked:       "selected files apply only to a clean tracked index and worktree; tracked changes",
	BlockUnsupportedChange:  "unsupported file changes",
	BlockPatchDoesNotApply:  "patch does not apply in the planned order",
}

type Blocker struct {
	Kind  BlockerKind
	Paths []string
}

func (blocker Blocker) Reason() string {
	if len(blocker.Paths) == 0 {
		return blockerReasons[blocker.Kind]
	}
	return blockerReasons[blocker.Kind] + ": " + strings.Join(blocker.Paths, pathSeparator)
}

type PreflightResult struct {
	Status       git.Status
	Commits      []string
	Target       string
	Blockers     []Blocker
	NeedsStash   bool
	StashOffered bool
}

func (result PreflightResult) reasons() []string {
	var reasons []string
	for _, blocker := range result.Blockers {
		reasons = append(reasons, blocker.Reason())
	}
	if result.NeedsStash {
		reasons = append(reasons, trackedChanges+strings.Join(result.Status.TrackedDirtyPaths, pathSeparator))
	}
	return reasons
}

type RestoreResult struct {
	Outcome  git.ApplyOutcome
	Cleaned  bool
	Retained *git.StashRef
}

type RollbackVerdict struct {
	Allowed bool
	Reason  string
}

type CherryPickRestore struct {
	statusReader *git.StatusReader
	operations   *git.OperationDetector
	differ       *git.Differ
	stasher      *git.Stasher
	cherryPicker *git.CherryPicker
	store        *StateStore
}

func NewCherryPickRestore(statusReader *git.StatusReader, operations *git.OperationDetector, differ *git.Differ, stasher *git.Stasher, cherryPicker *git.CherryPicker, store *StateStore) *CherryPickRestore {
	return &CherryPickRestore{statusReader: statusReader, operations: operations, differ: differ, stasher: stasher, cherryPicker: cherryPicker, store: store}
}

func (restore *CherryPickRestore) Preflight(ctx context.Context, commits []git.CommitSummary, reviewedHead string) (PreflightResult, error) {
	status, err := restore.statusReader.Snapshot(ctx)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("check cherry-pick target: %w", err)
	}
	operation, err := restore.operations.Active(ctx)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("check cherry-pick target: %w", err)
	}
	result := PreflightResult{Status: status, Target: status.Branch, NeedsStash: len(status.Unmerged) == 0 && !status.TrackedClean()}
	if len(status.Unmerged) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockUnmerged, Paths: status.Unmerged})
	}
	if operation.Active() {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockSequencer})
	}
	var stale, merges, incomingPaths []string
	if status.Head != reviewedHead {
		stale = append(stale, reviewedHeadLabel)
	}
	for _, commit := range commits {
		result.Commits = append(result.Commits, commit.OID)
		if len(commit.Parents) >= mergeParentCount {
			merges = append(merges, commit.OID)
			continue
		}
		parent := ""
		if len(commit.Parents) > 0 {
			parent = commit.Parents[0]
		}
		files, err := restore.differ.CommitFiles(ctx, commit.OID, parent)
		var commandErr *git.CommandError
		if errors.As(err, &commandErr) {
			stale = append(stale, commit.OID)
			continue
		}
		if err != nil {
			return PreflightResult{}, fmt.Errorf("check commit %s: %w", commit.OID, err)
		}
		for _, file := range files {
			incomingPaths = append(incomingPaths, file.Path)
		}
	}
	if collisions := status.UntrackedCollisions(incomingPaths); len(collisions) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockUntrackedCollision, Paths: collisions})
	}
	if len(stale) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockStale, Paths: stale})
	}
	if len(merges) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockMergeCommit, Paths: merges})
	}
	return result, nil
}

func (restore *CherryPickRestore) Execute(ctx context.Context, commits []git.CommitSummary, reviewedHead string) (git.SequenceOutcome, error) {
	result, err := restore.Preflight(ctx, commits, reviewedHead)
	if err != nil {
		return git.SequenceOutcome{}, fmt.Errorf("cherry-pick selected commits: %w", err)
	}
	if reasons := result.reasons(); len(reasons) > 0 {
		return git.SequenceOutcome{}, fmt.Errorf("cherry-pick selected commits: %w: %s", ErrExecutionBlocked, strings.Join(reasons, reasonSeparator))
	}
	outcome, err := restore.cherryPicker.Pick(ctx, result.Commits)
	if err != nil {
		return outcome, fmt.Errorf("cherry-pick selected commits onto %s: %w", result.Target, err)
	}
	return outcome, nil
}

type sourceCommit struct {
	oid    string
	parent string
}

type SelectiveApply struct {
	statusReader *git.StatusReader
	operations   *git.OperationDetector
	differ       *git.Differ
}

func NewSelectiveApply(statusReader *git.StatusReader, operations *git.OperationDetector, differ *git.Differ) *SelectiveApply {
	return &SelectiveApply{statusReader: statusReader, operations: operations, differ: differ}
}

func (selective *SelectiveApply) Preflight(ctx context.Context, steps []git.ApplyStep, reviewedHead string) (PreflightResult, error) {
	status, err := selective.statusReader.Snapshot(ctx)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("check selected files target: %w", err)
	}
	operation, err := selective.operations.Active(ctx)
	if err != nil {
		return PreflightResult{}, fmt.Errorf("check selected files target: %w", err)
	}
	result := PreflightResult{Status: status, Target: status.Branch}
	if len(status.Unmerged) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockUnmerged, Paths: status.Unmerged})
	}
	if operation.Active() {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockSequencer})
	}
	if len(status.TrackedDirtyPaths) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockDirtyTracked, Paths: status.TrackedDirtyPaths})
	}
	var stale, unsupported, incomingPaths []string
	if status.Head != reviewedHead {
		stale = append(stale, reviewedHeadLabel)
	}
	commitFiles := map[sourceCommit][]git.CommitFile{}
	for _, step := range steps {
		selection := step.Selection
		incomingPaths = append(incomingPaths, selection.Path)
		source := sourceCommit{oid: selection.OID, parent: selection.Parent}
		if _, loaded := commitFiles[source]; !loaded {
			files, err := selective.differ.CommitFiles(ctx, selection.OID, selection.Parent)
			var commandErr *git.CommandError
			if errors.As(err, &commandErr) {
				stale = append(stale, selection.OID)
				continue
			}
			if err != nil {
				return PreflightResult{}, fmt.Errorf("check commit %s: %w", selection.OID, err)
			}
			commitFiles[source] = files
		}
		position := slices.IndexFunc(commitFiles[source], func(file git.CommitFile) bool { return file.Path == selection.Path })
		switch {
		case position < 0:
			stale = append(stale, selection.Path)
		case commitFiles[source][position].UnsupportedReason != "":
			unsupported = append(unsupported, fmt.Sprintf(unsupportedPath, selection.Path, commitFiles[source][position].UnsupportedReason))
		}
	}
	if collisions := status.UntrackedCollisions(incomingPaths); len(collisions) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockUntrackedCollision, Paths: collisions})
	}
	if len(stale) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockStale, Paths: stale})
	}
	if len(unsupported) > 0 {
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockUnsupportedChange, Paths: unsupported})
	}
	if len(result.Blockers) > 0 {
		return result, nil
	}
	report, err := selective.differ.ApplyQueue(ctx, steps, true)
	var commandErr *git.CommandError
	switch {
	case errors.As(err, &commandErr):
		result.Blockers = append(result.Blockers, Blocker{Kind: BlockPatchDoesNotApply, Paths: []string{report.Failed[0].Path}})
	case err != nil:
		return PreflightResult{}, fmt.Errorf("check selected files apply in order: %w", err)
	}
	return result, nil
}

func (selective *SelectiveApply) Execute(ctx context.Context, steps []git.ApplyStep, reviewedHead string) (git.ApplyReport, error) {
	result, err := selective.Preflight(ctx, steps, reviewedHead)
	if err != nil {
		return git.ApplyReport{}, fmt.Errorf("apply selected files: %w", err)
	}
	if reasons := result.reasons(); len(reasons) > 0 {
		return git.ApplyReport{}, fmt.Errorf("apply selected files: %w: %s", ErrExecutionBlocked, strings.Join(reasons, reasonSeparator))
	}
	report, err := selective.differ.ApplyQueue(ctx, steps, false)
	if err != nil {
		return report, fmt.Errorf("apply selected files onto %s: %w", result.Target, err)
	}
	return report, nil
}

func (restore *CherryPickRestore) StashAndProceed(ctx context.Context, commits []git.CommitSummary) (git.SequenceOutcome, error) {
	return git.SequenceOutcome{}, errNotImplemented
}

func (restore *CherryPickRestore) Resume(ctx context.Context) (PendingRestore, bool, error) {
	return PendingRestore{}, false, errNotImplemented
}

func (restore *CherryPickRestore) Restore(ctx context.Context, pending PendingRestore) (RestoreResult, error) {
	return RestoreResult{}, errNotImplemented
}

func (restore *CherryPickRestore) CanAbortUnstash(ctx context.Context, pending PendingRestore) (RollbackVerdict, error) {
	return RollbackVerdict{}, errNotImplemented
}

func (restore *CherryPickRestore) AbortUnstash(ctx context.Context, pending PendingRestore) (RollbackVerdict, error) {
	return RollbackVerdict{}, errNotImplemented
}

func (restore *CherryPickRestore) Drifted(ctx context.Context, baseline Baseline) (bool, error) {
	return false, errNotImplemented
}
