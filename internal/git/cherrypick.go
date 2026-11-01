package git

import (
	"context"
	"fmt"
	"strings"
)

type SequenceOutcome struct {
	Completed []string
	Paused    bool
}

type CherryPicker struct {
	runner     *Runner
	operations *OperationDetector
}

func NewCherryPicker(runner *Runner, operations *OperationDetector) *CherryPicker {
	return &CherryPicker{runner: runner, operations: operations}
}

func (c *CherryPicker) Pick(ctx context.Context, oids []string) (SequenceOutcome, error) {
	head, err := c.runner.Read(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return SequenceOutcome{}, fmt.Errorf("read target before cherry-pick: %w", err)
	}
	_, pickErr := c.runner.Execute(ctx, Invocation{Args: append([]string{"cherry-pick"}, oids...), Env: []string{nonInteractiveGitEditor}, CombineOutput: true})
	created, err := c.runner.Read(ctx, "rev-list", "--reverse", strings.TrimSpace(string(head))+"..HEAD")
	if err != nil {
		return SequenceOutcome{}, fmt.Errorf("read commits created by cherry-pick: %w", err)
	}
	outcome := SequenceOutcome{Completed: strings.Fields(string(created))}
	if pickErr == nil {
		return outcome, nil
	}
	operation, err := c.operations.Active(ctx)
	if err != nil {
		return outcome, fmt.Errorf("read cherry-pick state after %v: %w", pickErr, err)
	}
	if operation.Kind != OperationCherryPick {
		return outcome, fmt.Errorf("cherry-pick %d commits: %w", len(oids), pickErr)
	}
	outcome.Paused = true
	return outcome, nil
}
