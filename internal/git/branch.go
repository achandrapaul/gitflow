package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	remoteMainRef        = "refs/remotes/origin/main"
	localMainRef         = "refs/heads/main"
	fetchHeadName        = "FETCH_HEAD"
	countFieldSize       = 2
	localRefPrefix       = "refs/heads/"
	remoteRefPrefix      = "refs/remotes/"
	currentRefMarker     = "*"
	branchListFormat     = "--format=%(refname)%00%(objectname)%00%(HEAD)%00%(upstream)%00%(symref)"
	branchListFieldCount = 5
	upstreamFormat       = "--format=%(refname)%00%(upstream:remotename)%00%(upstream:remoteref)%00%(upstream)"
	upstreamFieldCount   = 4
	missingRefExit       = 1
)

var (
	ErrInvalidBranchName = errors.New("invalid branch name")
	ErrBranchExists      = errors.New("branch already exists")
	ErrStaleRef          = errors.New("ref moved since review")
	ErrCurrentBranch     = errors.New("branch is checked out")
	ErrNotLocalBranch    = errors.New("not a local branch")
)

type SwitchBlocker int

const (
	BlockedCurrent SwitchBlocker = iota + 1
	BlockedNotLocal
	BlockedOperation
	BlockedUnmerged
	BlockedUntracked
	BlockedDirty
)

type SwitchBlocked struct {
	Branch  string
	Blocker SwitchBlocker
	Paths   []string
}

func (s *SwitchBlocked) Error() string {
	return fmt.Sprintf("switch to %s blocked: %d", s.Branch, s.Blocker)
}

type BranchInfo struct {
	Name     string
	Ref      string
	Remote   bool
	Current  bool
	Upstream string
	OID      string
}

type Upstream struct {
	Remote      string
	Ref         string
	TrackingRef string
	Set         bool
}

type FetchFreshness struct {
	LastFetch time.Time
	Known     bool
}

type AheadBehind struct {
	Ahead  int
	Behind int
}

type MainUpdate struct {
	Fetched       bool
	FastForwarded bool
	Reason        string
}

type Brancher struct {
	runner     *Runner
	status     *StatusReader
	operations *OperationDetector
}

func NewBrancher(runner *Runner, status *StatusReader, operations *OperationDetector) *Brancher {
	return &Brancher{runner: runner, status: status, operations: operations}
}

func (b *Brancher) List(ctx context.Context, includeRemote bool) ([]BranchInfo, error) {
	patterns := []string{localRefPrefix}
	if includeRemote {
		patterns = append(patterns, remoteRefPrefix)
	}
	output, err := b.runner.Read(ctx, append([]string{"for-each-ref", branchListFormat}, patterns...)...)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	var branches []BranchInfo
	for _, line := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x00")
		if len(fields) != branchListFieldCount {
			return nil, fmt.Errorf("list branches: unexpected record %q", line)
		}
		ref, symref := fields[0], fields[4]
		if symref != "" {
			continue
		}
		remote := strings.HasPrefix(ref, remoteRefPrefix)
		branches = append(branches, BranchInfo{
			Name:     strings.TrimPrefix(strings.TrimPrefix(ref, localRefPrefix), remoteRefPrefix),
			Ref:      ref,
			Remote:   remote,
			Current:  fields[2] == currentRefMarker,
			Upstream: fields[3],
			OID:      fields[1],
		})
	}
	return branches, nil
}

func (b *Brancher) Current(ctx context.Context) (BranchInfo, error) {
	return BranchInfo{}, errNotImplemented
}

func (b *Brancher) Upstream(ctx context.Context, branch string) (Upstream, error) {
	output, err := b.runner.Read(ctx, "for-each-ref", upstreamFormat, localRefPrefix+branch)
	if err != nil {
		return Upstream{}, fmt.Errorf("read upstream of %s: %w", branch, err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x00")
		if len(fields) != upstreamFieldCount {
			return Upstream{}, fmt.Errorf("read upstream of %s: unexpected record %q", branch, line)
		}
		if fields[0] != localRefPrefix+branch || fields[1] == "" {
			continue
		}
		return Upstream{Remote: fields[1], Ref: fields[2], TrackingRef: fields[3], Set: true}, nil
	}
	return Upstream{}, nil
}

func (b *Brancher) Freshness(ctx context.Context) (FetchFreshness, error) {
	paths, err := b.runner.GitPaths(ctx, fetchHeadName)
	if err != nil {
		return FetchFreshness{}, fmt.Errorf("read last fetch time: %w", err)
	}
	info, err := os.Stat(paths[0])
	if errors.Is(err, fs.ErrNotExist) {
		return FetchFreshness{}, nil
	}
	if err != nil {
		return FetchFreshness{}, fmt.Errorf("read %s time: %w", fetchHeadName, err)
	}
	return FetchFreshness{LastFetch: info.ModTime(), Known: true}, nil
}

func (b *Brancher) AheadBehind(ctx context.Context, local, other string) (AheadBehind, error) {
	output, err := b.runner.Read(ctx, "rev-list", "--left-right", "--count", local+"..."+other, "--")
	if err != nil {
		return AheadBehind{}, fmt.Errorf("compare %s with %s: %w", local, other, err)
	}
	counts := strings.Fields(string(output))
	if len(counts) != countFieldSize {
		return AheadBehind{}, fmt.Errorf("compare %s with %s: unexpected output %q", local, other, output)
	}
	ahead, aheadErr := strconv.Atoi(counts[0])
	behind, behindErr := strconv.Atoi(counts[1])
	if aheadErr != nil || behindErr != nil {
		return AheadBehind{}, fmt.Errorf("compare %s with %s: unexpected output %q", local, other, output)
	}
	return AheadBehind{Ahead: ahead, Behind: behind}, nil
}

func (b *Brancher) ResolveBase(ctx context.Context) (string, error) {
	output, err := b.runner.Read(ctx, "for-each-ref", "--format=%(refname)", remoteMainRef, localMainRef)
	if err != nil {
		return "", fmt.Errorf("resolve comparison base: %w", err)
	}
	refs := strings.Split(string(output), "\n")
	for _, candidate := range []string{remoteMainRef, localMainRef} {
		if slices.Contains(refs, candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func (b *Brancher) HeadPublished(ctx context.Context) (bool, error) {
	output, err := b.runner.Read(ctx, "for-each-ref", "--contains", headRevision, "--count=1", "--format=%(refname)", "refs/remotes")
	if err != nil {
		return false, fmt.Errorf("check whether HEAD is published: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (b *Brancher) resolveRef(ctx context.Context, ref string) (string, bool, error) {
	output, err := b.runner.Read(ctx, "rev-parse", "--verify", "-q", ref)
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == missingRefExit {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("resolve %s: %w", ref, err)
	}
	return strings.TrimSpace(string(output)), true, nil
}

func (b *Brancher) ValidateName(ctx context.Context, name string) error {
	output, err := b.runner.Read(ctx, "check-ref-format", "--branch", name)
	var commandErr *CommandError
	if errors.As(err, &commandErr) {
		return fmt.Errorf("validate branch name %q: %w: %w", name, ErrInvalidBranchName, err)
	}
	if err != nil {
		return fmt.Errorf("validate branch name %q: %w", name, err)
	}
	if strings.TrimSuffix(string(output), "\n") != name {
		return fmt.Errorf("validate branch name %q: %w: expands to %q", name, ErrInvalidBranchName, strings.TrimSpace(string(output)))
	}
	_, exists, err := b.resolveRef(ctx, localRefPrefix+name)
	if err != nil {
		return fmt.Errorf("validate branch name %q: %w", name, err)
	}
	if exists {
		return fmt.Errorf("validate branch name %q: %w", name, ErrBranchExists)
	}
	return nil
}

func (b *Brancher) Create(ctx context.Context, name, startRef, startOID string) error {
	if err := b.ValidateName(ctx, name); err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}
	currentOID, found, err := b.resolveRef(ctx, startRef)
	if err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}
	if !found || currentOID != startOID {
		return fmt.Errorf("create branch %s from %s: %w", name, startRef, ErrStaleRef)
	}
	if _, err := b.runner.Run(ctx, "branch", "--no-track", name, startOID); err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}
	return nil
}

func (b *Brancher) Switch(ctx context.Context, name, reviewedOID string) error {
	targetOID, found, err := b.resolveRef(ctx, localRefPrefix+name)
	if err != nil {
		return fmt.Errorf("switch to %s: %w", name, err)
	}
	if !found {
		return &SwitchBlocked{Branch: name, Blocker: BlockedNotLocal}
	}
	if targetOID != reviewedOID {
		return fmt.Errorf("switch to %s: %w", name, ErrStaleRef)
	}
	status, err := b.status.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("switch to %s: %w", name, err)
	}
	if status.Branch == name {
		return &SwitchBlocked{Branch: name, Blocker: BlockedCurrent}
	}
	operation, err := b.operations.Active(ctx)
	if err != nil {
		return fmt.Errorf("switch to %s: %w", name, err)
	}
	if operation.Active() {
		return &SwitchBlocked{Branch: name, Blocker: BlockedOperation}
	}
	if len(status.Unmerged) > 0 {
		return &SwitchBlocked{Branch: name, Blocker: BlockedUnmerged, Paths: status.Unmerged}
	}
	changedArgs := []string{"diff", "--name-only", "-z", "--no-renames", headRevision, targetOID, "--"}
	if status.Unborn {
		changedArgs = []string{"ls-tree", "-r", "-z", "--name-only", targetOID}
	}
	changedOutput, err := b.runner.Read(ctx, changedArgs...)
	if err != nil {
		return fmt.Errorf("switch to %s: list paths that differ: %w", name, err)
	}
	changedPaths := splitNulTerminated(changedOutput)
	if collisions := status.UntrackedCollisions(changedPaths); len(collisions) > 0 {
		return &SwitchBlocked{Branch: name, Blocker: BlockedUntracked, Paths: collisions}
	}
	var dirtyPaths []string
	for _, path := range status.TrackedDirtyPaths {
		if slices.Contains(changedPaths, path) {
			dirtyPaths = append(dirtyPaths, path)
		}
	}
	if len(dirtyPaths) > 0 {
		return &SwitchBlocked{Branch: name, Blocker: BlockedDirty, Paths: dirtyPaths}
	}
	if _, err := b.runner.Run(ctx, "switch", "--no-guess", "-q", name); err != nil {
		return fmt.Errorf("switch to %s: %w", name, err)
	}
	return nil
}

func (b *Brancher) Delete(ctx context.Context, name, reviewedOID string) error {
	currentOID, found, err := b.resolveRef(ctx, localRefPrefix+name)
	if err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	if !found {
		return fmt.Errorf("delete branch %s: %w", name, ErrNotLocalBranch)
	}
	if currentOID != reviewedOID {
		return fmt.Errorf("delete branch %s: %w", name, ErrStaleRef)
	}
	status, err := b.status.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	if status.Branch == name {
		return fmt.Errorf("delete branch %s: %w", name, ErrCurrentBranch)
	}
	if _, err := b.runner.Run(ctx, "branch", "-d", name); err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	return nil
}

func (b *Brancher) UpdateMainFastForward(ctx context.Context) (MainUpdate, error) {
	return MainUpdate{}, errNotImplemented
}
