package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	stderrExcerptLimit = 2000
	redactedUserInfo   = "${1}***@"
	nulSeparator       = "\x00"
)

var (
	errNotImplemented = errors.New("git: not implemented")
	urlUserInfo       = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/@\s]+@`)
)

func redactCredentials(text string) string {
	return urlUserInfo.ReplaceAllString(text, redactedUserInfo)
}

type CommandError struct {
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *CommandError) Error() string {
	stderr := redactCredentials(strings.TrimSpace(e.Stderr))
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
	Args          []string
	Stdin         []byte
	Env           []string
	CombineOutput bool
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

func splitNulTerminated(output []byte) []string {
	return strings.Split(strings.TrimSuffix(string(output), nulSeparator), nulSeparator)
}

func (r *Runner) Execute(ctx context.Context, invocation Invocation) ([]byte, error) {
	command := exec.CommandContext(ctx, r.gitPath, invocation.Args...)
	command.Dir = r.workTree
	command.Env = append(append(os.Environ(), r.extraEnv...), invocation.Env...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if invocation.CombineOutput {
		command.Stderr = &stdout
	}
	if invocation.Stdin != nil {
		command.Stdin = bytes.NewReader(invocation.Stdin)
	}
	if err := command.Run(); err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		failureOutput := stderr.String()
		if invocation.CombineOutput {
			failureOutput = stdout.String()
		}
		return stdout.Bytes(), &CommandError{Args: invocation.Args, ExitCode: exitCode, Stderr: failureOutput, Err: err}
	}
	return stdout.Bytes(), nil
}
