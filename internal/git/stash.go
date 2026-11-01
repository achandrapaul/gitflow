package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	stashListFormat     = "--format=%gd%x00%H%x00%cI%x00%gs"
	stashListFieldCount = 4
	stashRefPrefix      = "stash@{"
	stashRefSuffix      = "}"
)

type StashRef struct {
	OID     string
	Index   int
	Message string
	Date    time.Time
}

func (stash StashRef) Ref() string {
	return stashRefPrefix + strconv.Itoa(stash.Index) + stashRefSuffix
}

type ApplyOutcome struct {
	Applied    bool
	Conflicted []string
	IndexKept  bool
	Diagnostic string
}

type Stasher struct {
	runner *Runner
}

func NewStasher(runner *Runner) *Stasher {
	return &Stasher{runner: runner}
}

func (s *Stasher) PushTracked(ctx context.Context, message string) (StashRef, error) {
	return StashRef{}, errNotImplemented
}

func (s *Stasher) ApplyIndex(ctx context.Context, oid string) (ApplyOutcome, error) {
	return ApplyOutcome{}, errNotImplemented
}

func (s *Stasher) DropByOID(ctx context.Context, oid string) error {
	return errNotImplemented
}

func (s *Stasher) ResolveRef(ctx context.Context, oid string) (StashRef, bool, error) {
	return StashRef{}, false, errNotImplemented
}

func (s *Stasher) List(ctx context.Context) ([]StashRef, error) {
	output, err := s.runner.Read(ctx, "stash", "list", "-z", "--no-color", stashListFormat)
	if err != nil {
		return nil, fmt.Errorf("read stash list: %w", err)
	}
	if len(output) == 0 {
		return nil, nil
	}
	fields := splitNulTerminated(output)
	if len(fields)%stashListFieldCount != 0 {
		return nil, fmt.Errorf("read stash list: unexpected field count %d", len(fields))
	}
	stashes := make([]StashRef, 0, len(fields)/stashListFieldCount)
	for start := 0; start < len(fields); start += stashListFieldCount {
		ref := fields[start]
		indexText, isStashRef := strings.CutPrefix(ref, stashRefPrefix)
		indexText, closed := strings.CutSuffix(indexText, stashRefSuffix)
		index, err := strconv.Atoi(indexText)
		if !isStashRef || !closed || err != nil || index != len(stashes) {
			return nil, fmt.Errorf("read stash list: unexpected stash reference %q at position %d", ref, len(stashes))
		}
		date, err := time.Parse(time.RFC3339, fields[start+2])
		if err != nil {
			return nil, fmt.Errorf("read stash list: parse date of %s: %w", ref, err)
		}
		stashes = append(stashes, StashRef{OID: fields[start+1], Index: index, Message: fields[start+3], Date: date})
	}
	return stashes, nil
}

func (s *Stasher) Paths(ctx context.Context, oid string) ([]string, error) {
	return nil, errNotImplemented
}
