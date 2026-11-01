package git

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	originRemote       = "origin"
	missingConfigExit  = 1
	pullStrategyFormat = `^(pull\.(rebase|ff)|branch\.%s\.rebase)$`
)

var (
	ErrTrackingUnconfirmed = errors.New("push succeeded but upstream tracking was not confirmed")
	pullModeArgs           = map[PullMode][]string{PullConfigured: nil, PullFastForwardOnly: {"--ff-only"}}
)

type PullMode int

const (
	PullFastForwardOnly PullMode = iota
	PullConfigured
	PullRebase
)

type SyncBlocker int

const (
	SyncNoRemote SyncBlocker = iota + 1
	SyncNoOrigin
	SyncUnborn
	SyncDetached
	SyncNoUpstream
	SyncOperation
	SyncUnmerged
	SyncDirty
	SyncUnresolvedStrategy
	SyncLeaseUnknown
)

type SyncBlocked struct {
	Blocker SyncBlocker
	Paths   []string
}

func (s *SyncBlocked) Error() string {
	return fmt.Sprintf("remote sync blocked: %d", s.Blocker)
}

type DestinationState int

const (
	DestinationUnknown DestinationState = iota
	DestinationAbsent
	DestinationExists
)

type PushTarget struct {
	Branch      string
	LocalOID    string
	Remote      string
	RemoteRef   string
	TrackingRef string
	FirstPush   bool
	State       DestinationState
	CachedOID   string
	Tracking    AheadBehind
}

func (target PushTarget) PushArgs() []string {
	if target.FirstPush {
		return []string{"push", "-u", originRemote, target.Branch}
	}
	return []string{"push", target.Remote, localRefPrefix + target.Branch + ":" + target.RemoteRef}
}

func (target PushTarget) ForceArgs() []string {
	return []string{"push", "--force-with-lease=" + target.RemoteRef + ":" + target.CachedOID, target.Remote, localRefPrefix + target.Branch + ":" + target.RemoteRef}
}

func (target PushTarget) LeaseBlocked() error {
	if target.FirstPush {
		return &SyncBlocked{Blocker: SyncNoUpstream}
	}
	if target.State != DestinationExists {
		return &SyncBlocked{Blocker: SyncLeaseUnknown}
	}
	return nil
}

type RemoteSync struct {
	runner     *Runner
	status     *StatusReader
	operations *OperationDetector
	branches   *Brancher
}

func NewRemoteSync(runner *Runner, status *StatusReader, operations *OperationDetector, branches *Brancher) *RemoteSync {
	return &RemoteSync{runner: runner, status: status, operations: operations, branches: branches}
}

func (r *RemoteSync) remotes(ctx context.Context) ([]string, error) {
	output, err := r.runner.Read(ctx, "remote")
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}
	return strings.Fields(string(output)), nil
}

func outputSummary(output []byte) string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return redactCredentials(strings.TrimSpace(lines[len(lines)-1]))
}

func (r *RemoteSync) Fetch(ctx context.Context) (string, error) {
	remotes, err := r.remotes(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	if len(remotes) == 0 {
		return "", &SyncBlocked{Blocker: SyncNoRemote}
	}
	output, err := r.runner.Execute(ctx, Invocation{Args: []string{"fetch"}, CombineOutput: true})
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	return outputSummary(output), nil
}

func (r *RemoteSync) Pull(ctx context.Context, mode PullMode) (string, error) {
	modeArgs, supported := pullModeArgs[mode]
	if !supported {
		return "", fmt.Errorf("pull mode %d: %w", mode, errNotImplemented)
	}
	status, err := r.status.Snapshot(ctx)
	if err != nil {
		return "", fmt.Errorf("pull: %w", err)
	}
	if status.Unborn {
		return "", &SyncBlocked{Blocker: SyncUnborn}
	}
	if status.Detached {
		return "", &SyncBlocked{Blocker: SyncDetached}
	}
	upstream, err := r.branches.Upstream(ctx, status.Branch)
	if err != nil {
		return "", fmt.Errorf("pull %s: %w", status.Branch, err)
	}
	if !upstream.Set {
		return "", &SyncBlocked{Blocker: SyncNoUpstream}
	}
	operation, err := r.operations.Active(ctx)
	if err != nil {
		return "", fmt.Errorf("pull %s: %w", status.Branch, err)
	}
	if operation.Active() {
		return "", &SyncBlocked{Blocker: SyncOperation}
	}
	if len(status.Unmerged) > 0 {
		return "", &SyncBlocked{Blocker: SyncUnmerged, Paths: status.Unmerged}
	}
	if len(status.TrackedDirtyPaths) > 0 {
		return "", &SyncBlocked{Blocker: SyncDirty, Paths: status.TrackedDirtyPaths}
	}
	if mode == PullConfigured && status.TrackingKnown && status.Tracking.Ahead > 0 && status.Tracking.Behind > 0 {
		_, err := r.runner.Read(ctx, "config", "--get-regexp", fmt.Sprintf(pullStrategyFormat, regexp.QuoteMeta(status.Branch)))
		var commandErr *CommandError
		if errors.As(err, &commandErr) && commandErr.ExitCode == missingConfigExit {
			return "", &SyncBlocked{Blocker: SyncUnresolvedStrategy}
		}
		if err != nil {
			return "", fmt.Errorf("pull %s: read reconciliation config: %w", status.Branch, err)
		}
	}
	output, err := r.runner.Execute(ctx, Invocation{Args: append([]string{"pull"}, modeArgs...), CombineOutput: true})
	if err != nil {
		return "", fmt.Errorf("pull %s from %s %s: %w", status.Branch, upstream.Remote, upstream.Ref, err)
	}
	return outputSummary(output), nil
}

func (r *RemoteSync) PushTarget(ctx context.Context) (PushTarget, error) {
	status, err := r.status.Snapshot(ctx)
	if err != nil {
		return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
	}
	if status.Unborn {
		return PushTarget{}, &SyncBlocked{Blocker: SyncUnborn}
	}
	if status.Detached {
		return PushTarget{}, &SyncBlocked{Blocker: SyncDetached}
	}
	upstream, err := r.branches.Upstream(ctx, status.Branch)
	if err != nil {
		return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
	}
	target := PushTarget{Branch: status.Branch, LocalOID: status.Head, Remote: upstream.Remote, RemoteRef: upstream.Ref, TrackingRef: upstream.TrackingRef}
	if !upstream.Set {
		remotes, err := r.remotes(ctx)
		if err != nil {
			return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
		}
		if !slices.Contains(remotes, originRemote) {
			return PushTarget{}, &SyncBlocked{Blocker: SyncNoOrigin}
		}
		target.Remote, target.RemoteRef, target.FirstPush = originRemote, localRefPrefix+status.Branch, true
		target.TrackingRef = remoteRefPrefix + originRemote + "/" + status.Branch
	}
	if target.TrackingRef == "" {
		return target, nil
	}
	cachedOID, cached, err := r.branches.resolveRef(ctx, target.TrackingRef)
	if err != nil {
		return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
	}
	if cached {
		tracking, err := r.branches.AheadBehind(ctx, localRefPrefix+status.Branch, target.TrackingRef)
		if err != nil {
			return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
		}
		target.State, target.CachedOID, target.Tracking = DestinationExists, cachedOID, tracking
		return target, nil
	}
	freshness, err := r.branches.Freshness(ctx)
	if err != nil {
		return PushTarget{}, fmt.Errorf("resolve push target: %w", err)
	}
	if freshness.Known {
		target.State = DestinationAbsent
	}
	return target, nil
}

func (r *RemoteSync) confirmTarget(ctx context.Context, reviewed PushTarget) error {
	current, err := r.PushTarget(ctx)
	if err != nil {
		return err
	}
	if current != reviewed {
		return fmt.Errorf("push %s to %s %s: %w", reviewed.Branch, reviewed.Remote, reviewed.RemoteRef, ErrStaleRef)
	}
	return nil
}

func (r *RemoteSync) Push(ctx context.Context, reviewed PushTarget) (string, error) {
	if err := r.confirmTarget(ctx, reviewed); err != nil {
		return "", err
	}
	output, err := r.runner.Execute(ctx, Invocation{Args: reviewed.PushArgs(), CombineOutput: true})
	if err != nil {
		return "", fmt.Errorf("push %s to %s %s: %w", reviewed.Branch, reviewed.Remote, reviewed.RemoteRef, err)
	}
	if !reviewed.FirstPush {
		return outputSummary(output), nil
	}
	upstream, err := r.branches.Upstream(ctx, reviewed.Branch)
	if err != nil {
		return "", fmt.Errorf("push %s: confirm tracking: %w", reviewed.Branch, err)
	}
	if upstream.Remote != originRemote || upstream.Ref != reviewed.RemoteRef {
		return "", fmt.Errorf("push %s to %s %s: %w", reviewed.Branch, originRemote, reviewed.RemoteRef, ErrTrackingUnconfirmed)
	}
	return outputSummary(output), nil
}

func (r *RemoteSync) ForceWithLease(ctx context.Context, reviewed PushTarget) (string, error) {
	if err := reviewed.LeaseBlocked(); err != nil {
		return "", err
	}
	if err := r.confirmTarget(ctx, reviewed); err != nil {
		return "", err
	}
	output, err := r.runner.Execute(ctx, Invocation{Args: reviewed.ForceArgs(), CombineOutput: true})
	if err != nil {
		return "", fmt.Errorf("force push %s to %s %s with lease %s: %w", reviewed.Branch, reviewed.Remote, reviewed.RemoteRef, reviewed.CachedOID, err)
	}
	return outputSummary(output), nil
}

func (r *RemoteSync) SetUpstream(ctx context.Context, branch, upstream string) error {
	return errNotImplemented
}
