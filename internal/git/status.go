package git

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type StatusFilter int

const (
	FilterAll StatusFilter = iota
	FilterStaged
	FilterNotStaged
	FilterMixed
)

type ChangeKind int

const (
	KindOrdinary ChangeKind = iota
	KindRenamed
	KindUnmerged
	KindUntracked
)

const (
	unchangedCode       byte = '.'
	untrackedCode       byte = '?'
	headerPrefix             = '#'
	ordinaryPrefix           = '1'
	renamedPrefix            = '2'
	unmergedPrefix           = 'u'
	untrackedPrefix          = '?'
	headerFieldCount         = 3
	ordinaryFieldCount       = 9
	renamedFieldCount        = 10
	unmergedFieldCount       = 11
	untrackedPathOffset      = 2
	headerOID                = "branch.oid"
	headerHead               = "branch.head"
	headerUpstream           = "branch.upstream"
	headerAheadBehind        = "branch.ab"
	unbornOID                = "(initial)"
	detachedHead             = "(detached)"
)

type FileChange struct {
	Path         string
	OriginalPath string
	Staged       byte
	Unstaged     byte
	Kind         ChangeKind
}

func (f FileChange) HasStaged() bool {
	return (f.Kind == KindOrdinary || f.Kind == KindRenamed) && f.Staged != unchangedCode
}

func (f FileChange) HasUnstaged() bool {
	return f.Kind == KindUntracked || f.Kind == KindUnmerged || f.Unstaged != unchangedCode
}

func (f FileChange) Mixed() bool {
	return f.HasStaged() && f.HasUnstaged()
}

func (f FileChange) Matches(filter StatusFilter) bool {
	switch filter {
	case FilterStaged:
		return f.HasStaged()
	case FilterNotStaged:
		return f.HasUnstaged()
	case FilterMixed:
		return f.Mixed()
	default:
		return true
	}
}

type Status struct {
	Head              string
	Branch            string
	Detached          bool
	Unborn            bool
	Upstream          string
	Tracking          AheadBehind
	TrackingKnown     bool
	Changes           []FileChange
	TrackedDirtyPaths []string
	Unmerged          []string
	Untracked         []string
}

func (s Status) TrackedClean() bool {
	return len(s.TrackedDirtyPaths) == 0 && len(s.Unmerged) == 0
}

func (s Status) UntrackedCollisions(paths []string) []string {
	var collisions []string
	for _, path := range s.Untracked {
		if slices.Contains(paths, path) {
			collisions = append(collisions, path)
		}
	}
	return collisions
}

type StatusReader struct {
	runner *Runner
}

func NewStatusReader(runner *Runner) *StatusReader {
	return &StatusReader{runner: runner}
}

func (s *StatusReader) Snapshot(ctx context.Context) (Status, error) {
	output, err := s.runner.Read(ctx, "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all")
	if err != nil {
		return Status{}, fmt.Errorf("read status: %w", err)
	}
	var status Status
	records := bytes.Split(bytes.TrimSuffix(output, []byte{0}), []byte{0})
	for recordIndex := 0; recordIndex < len(records); recordIndex++ {
		record := string(records[recordIndex])
		if record == "" {
			continue
		}
		switch record[0] {
		case headerPrefix:
			header := strings.SplitN(record, " ", headerFieldCount)
			if len(header) != headerFieldCount {
				return Status{}, fmt.Errorf("parse status header %q", record)
			}
			switch header[1] {
			case headerOID:
				status.Unborn = header[2] == unbornOID
				if !status.Unborn {
					status.Head = header[2]
				}
			case headerHead:
				status.Detached = header[2] == detachedHead
				if !status.Detached {
					status.Branch = header[2]
				}
			case headerUpstream:
				status.Upstream = header[2]
			case headerAheadBehind:
				counts := strings.Fields(header[2])
				if len(counts) != 2 {
					return Status{}, fmt.Errorf("parse status ahead/behind %q", header[2])
				}
				ahead, aheadErr := strconv.Atoi(strings.TrimPrefix(counts[0], "+"))
				behind, behindErr := strconv.Atoi(strings.TrimPrefix(counts[1], "-"))
				if aheadErr != nil || behindErr != nil {
					return Status{}, fmt.Errorf("parse status ahead/behind %q", header[2])
				}
				status.Tracking = AheadBehind{Ahead: ahead, Behind: behind}
				status.TrackingKnown = true
			}
		case ordinaryPrefix, renamedPrefix, unmergedPrefix:
			fieldCount := map[byte]int{ordinaryPrefix: ordinaryFieldCount, renamedPrefix: renamedFieldCount, unmergedPrefix: unmergedFieldCount}[record[0]]
			fields := strings.SplitN(record, " ", fieldCount)
			if len(fields) != fieldCount || len(fields[1]) != 2 {
				return Status{}, fmt.Errorf("parse status record %q", record)
			}
			change := FileChange{Path: fields[fieldCount-1], Staged: fields[1][0], Unstaged: fields[1][1], Kind: KindOrdinary}
			switch record[0] {
			case renamedPrefix:
				recordIndex++
				if recordIndex >= len(records) {
					return Status{}, fmt.Errorf("parse status rename of %q: missing original path", change.Path)
				}
				change.Kind = KindRenamed
				change.OriginalPath = string(records[recordIndex])
				status.TrackedDirtyPaths = append(status.TrackedDirtyPaths, change.OriginalPath, change.Path)
			case unmergedPrefix:
				change.Kind = KindUnmerged
				status.Unmerged = append(status.Unmerged, change.Path)
			default:
				status.TrackedDirtyPaths = append(status.TrackedDirtyPaths, change.Path)
			}
			status.Changes = append(status.Changes, change)
		case untrackedPrefix:
			untrackedPath := record[untrackedPathOffset:]
			status.Untracked = append(status.Untracked, untrackedPath)
			status.Changes = append(status.Changes, FileChange{Path: untrackedPath, Staged: untrackedCode, Unstaged: untrackedCode, Kind: KindUntracked})
		}
	}
	return status, nil
}
