package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

const unchangedCode = '.'

type EntryKind int

const (
	KindOrdinary EntryKind = iota
	KindRenamed
	KindUnmerged
	KindUntracked
)

type FileChange struct {
	Path     string
	OrigPath string
	Index    byte
	Worktree byte
	Kind     EntryKind
}

func (f FileChange) Staged() bool {
	return (f.Kind == KindOrdinary || f.Kind == KindRenamed) && f.Index != unchangedCode
}

func (f FileChange) Unstaged() bool {
	return f.Kind == KindUntracked || ((f.Kind == KindOrdinary || f.Kind == KindRenamed) && f.Worktree != unchangedCode)
}

func (f FileChange) Conflicted() bool { return f.Kind == KindUnmerged }

func (f FileChange) Mixed() bool { return f.Staged() && f.Unstaged() && f.Kind != KindUntracked }

func (f FileChange) Signature() string {
	return string([]byte{f.Index, f.Worktree}) + "\x00" + f.OrigPath + "\x00" + f.Path
}

type Filter int

const (
	FilterAll Filter = iota
	FilterStaged
	FilterNotStaged
	FilterMixed
)

var Filters = []Filter{FilterAll, FilterStaged, FilterNotStaged, FilterMixed}

func (f Filter) Label() string {
	return [...]string{"All", "Staged", "Not staged", "Mixed"}[f]
}

func (f Filter) Matches(file FileChange) bool {
	switch f {
	case FilterStaged:
		return file.Staged()
	case FilterNotStaged:
		return file.Unstaged() && !file.Staged()
	case FilterMixed:
		return file.Mixed()
	default:
		return true
	}
}

type BranchStatus struct {
	OID            string
	Head           string
	Detached       bool
	Unborn         bool
	Upstream       string
	HasAheadBehind bool
	Ahead          int
	Behind         int
}

type Status struct {
	Branch BranchStatus
	Files  []FileChange
}

func (s Status) HasConflicts() bool {
	for _, file := range s.Files {
		if file.Conflicted() {
			return true
		}
	}
	return false
}

func (s Status) HasStaged() bool {
	for _, file := range s.Files {
		if file.Staged() {
			return true
		}
	}
	return false
}

func (s Status) HasTrackedChanges() bool {
	for _, file := range s.Files {
		if file.Kind != KindUntracked {
			return true
		}
	}
	return false
}

func LoadStatus(ctx context.Context, runner *Runner) (Status, error) {
	output, err := runner.Read(ctx, "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return Status{}, fmt.Errorf("read status: %w", err)
	}
	return ParseStatus(output)
}

func ParseStatus(raw []byte) (Status, error) {
	var status Status
	records := bytes.Split(bytes.TrimSuffix(raw, []byte{0}), []byte{0})
	for recordIndex := 0; recordIndex < len(records); recordIndex++ {
		record := string(records[recordIndex])
		if record == "" {
			continue
		}
		switch record[0] {
		case '#':
			header := strings.SplitN(record, " ", 3)
			if len(header) < 3 {
				continue
			}
			switch header[1] {
			case "branch.oid":
				status.Branch.Unborn = header[2] == "(initial)"
				if !status.Branch.Unborn {
					status.Branch.OID = header[2]
				}
			case "branch.head":
				status.Branch.Detached = header[2] == "(detached)"
				if !status.Branch.Detached {
					status.Branch.Head = header[2]
				}
			case "branch.upstream":
				status.Branch.Upstream = header[2]
			case "branch.ab":
				counts := strings.Fields(header[2])
				if len(counts) != 2 {
					return Status{}, fmt.Errorf("parse status branch.ab %q", header[2])
				}
				ahead, aheadErr := strconv.Atoi(strings.TrimPrefix(counts[0], "+"))
				behind, behindErr := strconv.Atoi(strings.TrimPrefix(counts[1], "-"))
				if aheadErr != nil || behindErr != nil {
					return Status{}, fmt.Errorf("parse status branch.ab %q", header[2])
				}
				status.Branch.HasAheadBehind = true
				status.Branch.Ahead = ahead
				status.Branch.Behind = behind
			}
		case '1', '2', 'u':
			fieldCounts := map[byte]int{'1': 9, '2': 10, 'u': 11}
			fields := strings.SplitN(record, " ", fieldCounts[record[0]])
			if len(fields) != fieldCounts[record[0]] || len(fields[1]) != 2 {
				return Status{}, fmt.Errorf("parse status record %q", record)
			}
			file := FileChange{Path: fields[len(fields)-1], Index: fields[1][0], Worktree: fields[1][1]}
			switch record[0] {
			case '2':
				file.Kind = KindRenamed
				recordIndex++
				if recordIndex >= len(records) {
					return Status{}, fmt.Errorf("parse status rename %q: missing original path", file.Path)
				}
				file.OrigPath = string(records[recordIndex])
			case 'u':
				file.Kind = KindUnmerged
			}
			status.Files = append(status.Files, file)
		case '?':
			status.Files = append(status.Files, FileChange{Path: record[2:], Index: unchangedCode, Worktree: '?', Kind: KindUntracked})
		}
	}
	return status, nil
}
