package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const stderrExcerptLimit = 2000

type CommandError struct {
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *CommandError) Error() string {
	stderr := strings.TrimSpace(e.Stderr)
	if len(stderr) > stderrExcerptLimit {
		stderr = stderr[:stderrExcerptLimit] + "…"
	}
	if stderr == "" {
		return fmt.Sprintf("git %s: %v", strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), stderr)
}

func (e *CommandError) Unwrap() error { return e.Err }

type Invocation struct {
	Args  []string
	Stdin []byte
	Env   []string
}

type Runner struct {
	gitPath  string
	workTree string
	extraEnv []string
}

func NewRunner(gitPath, workTree string, extraEnv []string) *Runner {
	return &Runner{gitPath: gitPath, workTree: workTree, extraEnv: extraEnv}
}

func (r *Runner) WorkTree() string { return r.workTree }

func (r *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return r.Execute(ctx, Invocation{Args: args})
}

func (r *Runner) Read(ctx context.Context, args ...string) ([]byte, error) {
	return r.Execute(ctx, Invocation{Args: append([]string{"--no-optional-locks"}, args...)})
}

func (r *Runner) Execute(ctx context.Context, invocation Invocation) ([]byte, error) {
	command := r.Command(ctx, invocation.Env, invocation.Args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if invocation.Stdin != nil {
		command.Stdin = bytes.NewReader(invocation.Stdin)
	}
	if err := command.Run(); err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return stdout.Bytes(), &CommandError{Args: invocation.Args, ExitCode: exitCode, Stderr: stderr.String(), Err: err}
	}
	return stdout.Bytes(), nil
}

func (r *Runner) Command(ctx context.Context, env []string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, r.gitPath, args...)
	command.Dir = r.workTree
	command.Env = append(append(os.Environ(), r.extraEnv...), env...)
	return command
}
