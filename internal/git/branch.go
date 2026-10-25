package git

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var defaultBaseRefs = []string{"refs/remotes/origin/main", "refs/heads/main"}

var (
	ErrInvalidName = errors.New("invalid name")
	ErrNoUpstream  = errors.New("no upstream configured: set one with Set upstream in the branch inspector")
)

type Comparison struct {
	Ref         string
	Available   bool
	CountsKnown bool
	Ahead       int
	Behind      int
}

func (c Comparison) ShortRef() string {
	for _, prefix := range []string{"refs/remotes/", "refs/heads/"} {
		if strings.HasPrefix(c.Ref, prefix) {
			return strings.TrimPrefix(c.Ref, prefix)
		}
	}
	return c.Ref
}

func ResolveBase(ctx context.Context, runner *Runner, override string) (Comparison, error) {
	candidates := defaultBaseRefs
	if override != "" {
		if strings.HasPrefix(override, "-") {
			return Comparison{}, fmt.Errorf("resolve comparison base %q: %w", override, ErrInvalidName)
		}
		if _, err := runner.Read(ctx, "rev-parse", "--verify", "--quiet", "--end-of-options", override+"^{commit}"); err != nil {
			return Comparison{}, fmt.Errorf("resolve comparison base %q: %w", override, err)
		}
		candidates = []string{override}
	}
	for _, candidate := range candidates {
		if _, err := runner.Read(ctx, "rev-parse", "--verify", "--quiet", "--end-of-options", candidate+"^{commit}"); err != nil {
			continue
		}
		comparison := Comparison{Ref: candidate, Available: true}
		output, err := runner.Read(ctx, "rev-list", "--left-right", "--count", "HEAD..."+candidate, "--")
		if err != nil {
			return comparison, nil
		}
		counts := strings.Fields(string(output))
		if len(counts) != 2 {
			return Comparison{}, fmt.Errorf("compare HEAD with %s: unexpected output %q", candidate, output)
		}
		comparison.Ahead, _ = strconv.Atoi(counts[0])
		comparison.Behind, _ = strconv.Atoi(counts[1])
		comparison.CountsKnown = true
		return comparison, nil
	}
	return Comparison{}, nil
}

type Branch struct {
	Name     string
	Upstream string
	Current  bool
	OID      string
}

func ListBranches(ctx context.Context, runner *Runner) ([]Branch, error) {
	output, err := runner.Read(ctx, "for-each-ref", "--format=%(refname:short)%00%(upstream:short)%00%(HEAD)%00%(objectname)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	var branches []Branch
	for _, line := range strings.Split(strings.TrimRight(string(output), "\n"), "\n") {
		fields := strings.Split(line, "\x00")
		if len(fields) != 4 {
			continue
		}
		branches = append(branches, Branch{Name: fields[0], Upstream: fields[1], Current: fields[2] == "*", OID: fields[3]})
	}
	return branches, nil
}

func ListRemotes(ctx context.Context, runner *Runner) ([]string, error) {
	output, err := runner.Read(ctx, "remote")
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}
	return strings.Fields(string(output)), nil
}

func ValidateBranchName(ctx context.Context, runner *Runner, name string) error {
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("validate branch name %q: %w", name, ErrInvalidName)
	}
	if _, err := runner.Read(ctx, "check-ref-format", "--branch", name); err != nil {
		return fmt.Errorf("validate branch name %q: %w: %w", name, ErrInvalidName, err)
	}
	return nil
}

func CreateBranch(ctx context.Context, runner *Runner, name string) error {
	if err := ValidateBranchName(ctx, runner, name); err != nil {
		return err
	}
	if _, err := runner.Run(ctx, "branch", "--end-of-options", name); err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}
	return nil
}

func SwitchBranch(ctx context.Context, runner *Runner, name string) error {
	if err := ValidateBranchName(ctx, runner, name); err != nil {
		return err
	}
	if _, err := runner.Run(ctx, "switch", "--end-of-options", name); err != nil {
		return fmt.Errorf("switch to %s: %w", name, err)
	}
	return nil
}

func DeleteBranch(ctx context.Context, runner *Runner, name string) error {
	if err := ValidateBranchName(ctx, runner, name); err != nil {
		return err
	}
	if _, err := runner.Run(ctx, "branch", "-d", "--end-of-options", name); err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	return nil
}

func SetUpstream(ctx context.Context, runner *Runner, upstreamRef string) error {
	if upstreamRef == "" || strings.HasPrefix(upstreamRef, "-") {
		return fmt.Errorf("set upstream %q: %w", upstreamRef, ErrInvalidName)
	}
	if _, err := runner.Run(ctx, "branch", "--set-upstream-to="+upstreamRef); err != nil {
		return fmt.Errorf("set upstream to %s: %w", upstreamRef, err)
	}
	return nil
}

func RemoteRefsContaining(ctx context.Context, runner *Runner, oid string) ([]string, error) {
	output, err := runner.Read(ctx, "for-each-ref", "--contains", oid, "--format=%(refname:short)", "refs/remotes")
	if err != nil {
		return nil, fmt.Errorf("find remote refs containing %s: %w", oid, err)
	}
	return strings.Fields(string(output)), nil
}

func PushSetUpstreamInvocation(remote, branch string) (Invocation, error) {
	if remote == "" || strings.HasPrefix(remote, "-") || branch == "" || strings.HasPrefix(branch, "-") {
		return Invocation{}, fmt.Errorf("push %s to %s: %w", branch, remote, ErrInvalidName)
	}
	return Invocation{Args: []string{"push", "--set-upstream", remote, branch}}, nil
}

type Upstream struct {
	Remote    string
	RemoteRef string
}

func LoadUpstream(ctx context.Context, runner *Runner, branch string) (Upstream, error) {
	if branch == "" || strings.HasPrefix(branch, "-") {
		return Upstream{}, fmt.Errorf("load upstream of %q: %w", branch, ErrInvalidName)
	}
	output, err := runner.Read(ctx, "for-each-ref", "--format=%(upstream:remotename)%00%(upstream:remoteref)", "refs/heads/"+branch)
	if err != nil {
		return Upstream{}, fmt.Errorf("load upstream of %s: %w", branch, err)
	}
	remote, remoteRef, _ := strings.Cut(strings.TrimRight(string(output), "\n"), "\x00")
	if remote == "" || remoteRef == "" {
		return Upstream{}, nil
	}
	return Upstream{Remote: remote, RemoteRef: remoteRef}, nil
}

type SyncKind int

const (
	SyncFetch SyncKind = iota
	SyncPull
	SyncPush
)

func (k SyncKind) Name() string {
	return [...]string{"Fetch", "Pull", "Push"}[k]
}

func SyncInvocation(kind SyncKind, upstream Upstream) (Invocation, error) {
	if strings.HasPrefix(upstream.Remote, "-") || strings.HasPrefix(upstream.RemoteRef, "-") {
		return Invocation{}, fmt.Errorf("%s %s %s: %w", kind.Name(), upstream.Remote, upstream.RemoteRef, ErrInvalidName)
	}
	switch {
	case kind == SyncFetch && upstream.Remote == "":
		return Invocation{Args: []string{"fetch"}}, nil
	case kind == SyncFetch:
		return Invocation{Args: []string{"fetch", upstream.Remote}}, nil
	case upstream.Remote == "":
		return Invocation{}, fmt.Errorf("%s: %w", kind.Name(), ErrNoUpstream)
	case kind == SyncPull:
		return Invocation{Args: []string{"pull", "--ff-only"}}, nil
	}
	return Invocation{Args: []string{"push", upstream.Remote, "HEAD:" + upstream.RemoteRef}}, nil
}
