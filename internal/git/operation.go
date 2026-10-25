package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type OperationKind int

const (
	OperationNone OperationKind = iota
	OperationRebase
	OperationCherryPick
	OperationMerge
)

var ErrSkipUnsupported = errors.New("merge has no skip operation")

func (k OperationKind) Name() string {
	return [...]string{"none", "rebase", "cherry-pick", "merge"}[k]
}

type Operation struct {
	Kind          OperationKind
	HeadName      string
	Onto          string
	HasProgress   bool
	Step          int
	Total         int
	HasRemaining  bool
	Remaining     int
	CurrentCommit string
	EditStop      bool
	EmptyPick     bool
}

func (o Operation) Active() bool { return o.Kind != OperationNone }

func (o Operation) SupportsSkip() bool {
	return o.Kind == OperationRebase || o.Kind == OperationCherryPick
}

func (o Operation) ContinueInvocation() Invocation {
	return Invocation{Args: []string{o.Kind.Name(), "--continue"}}
}

func (o Operation) SkipInvocation() (Invocation, error) {
	if !o.SupportsSkip() {
		return Invocation{}, ErrSkipUnsupported
	}
	return Invocation{Args: []string{o.Kind.Name(), "--skip"}}, nil
}

func (o Operation) AbortInvocation() Invocation {
	return Invocation{Args: []string{o.Kind.Name(), "--abort"}}
}

func readTrimmed(path string) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(content)), true
}

func DetectOperation(ctx context.Context, runner *Runner, gitDir string) (Operation, error) {
	for _, rebaseDir := range []string{"rebase-merge", "rebase-apply"} {
		stateDir := filepath.Join(gitDir, rebaseDir)
		if _, err := os.Stat(stateDir); err != nil {
			continue
		}
		if _, isApplyMailbox := readTrimmed(filepath.Join(stateDir, "applying")); isApplyMailbox {
			continue
		}
		operation := Operation{Kind: OperationRebase}
		operation.HeadName, _ = readTrimmed(filepath.Join(stateDir, "head-name"))
		operation.HeadName = strings.TrimPrefix(operation.HeadName, "refs/heads/")
		operation.Onto, _ = readTrimmed(filepath.Join(stateDir, "onto"))
		stepFile, totalFile := "msgnum", "end"
		if rebaseDir == "rebase-apply" {
			stepFile, totalFile = "next", "last"
		}
		stepText, hasStep := readTrimmed(filepath.Join(stateDir, stepFile))
		totalText, hasTotal := readTrimmed(filepath.Join(stateDir, totalFile))
		step, stepErr := strconv.Atoi(stepText)
		total, totalErr := strconv.Atoi(totalText)
		if hasStep && hasTotal && stepErr == nil && totalErr == nil {
			operation.HasProgress, operation.Step, operation.Total = true, step, total
		}
		operation.CurrentCommit, _ = readTrimmed(filepath.Join(stateDir, "stopped-sha"))
		if operation.CurrentCommit == "" {
			operation.CurrentCommit, _ = readTrimmed(filepath.Join(gitDir, "REBASE_HEAD"))
		}
		_, operation.EditStop = readTrimmed(filepath.Join(stateDir, "amend"))
		return operation, nil
	}

	cherryPickHead, hasCherryPickHead := readTrimmed(filepath.Join(gitDir, "CHERRY_PICK_HEAD"))
	sequencerTodo, hasSequencer := readTrimmed(filepath.Join(gitDir, "sequencer", "todo"))
	if hasSequencer && !strings.HasPrefix(sequencerTodo, "pick") && !hasCherryPickHead {
		hasSequencer = false
	}
	if hasCherryPickHead || hasSequencer {
		operation := Operation{Kind: OperationCherryPick, CurrentCommit: cherryPickHead}
		if hasSequencer {
			operation.HasRemaining = true
			for _, line := range strings.Split(sequencerTodo, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") {
					operation.Remaining++
				}
			}
		}
		if hasCherryPickHead {
			unmerged, err := runner.Read(ctx, "ls-files", "--unmerged", "-z")
			if err != nil {
				return Operation{}, fmt.Errorf("detect cherry-pick state: %w", err)
			}
			if len(unmerged) == 0 {
				_, diffErr := runner.Read(ctx, "diff", "--cached", "--quiet", "--no-ext-diff")
				operation.EmptyPick = diffErr == nil
			}
		}
		return operation, nil
	}

	if mergeHead, ok := readTrimmed(filepath.Join(gitDir, "MERGE_HEAD")); ok {
		return Operation{Kind: OperationMerge, CurrentCommit: strings.Fields(mergeHead + " ")[0]}, nil
	}
	return Operation{}, nil
}
