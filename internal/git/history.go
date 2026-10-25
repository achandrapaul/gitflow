package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	HistoryPageSize = 100
	fieldSeparator  = "\x1f"
	commitFormat    = "--format=%H%x1f%P%x1f%an%x1f%as%x1f%s"
	commitFields    = 5
)

var ErrInvalidOID = errors.New("invalid object ID")

var fullOIDPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

type Commit struct {
	OID     string
	Parents []string
	Author  string
	Date    string
	Subject string
}

func (c Commit) Short() string {
	if len(c.OID) < 7 {
		return c.OID
	}
	return c.OID[:7]
}

func (c Commit) IsMerge() bool { return len(c.Parents) > 1 }

func ValidateOIDs(oids []string) error {
	for _, oid := range oids {
		if !fullOIDPattern.MatchString(oid) {
			return fmt.Errorf("validate %q: %w", oid, ErrInvalidOID)
		}
	}
	return nil
}

func parseCommitRecord(record string) (Commit, error) {
	fields := strings.SplitN(record, fieldSeparator, commitFields)
	if len(fields) != commitFields {
		return Commit{}, fmt.Errorf("parse commit record %q", record)
	}
	return Commit{OID: fields[0], Parents: strings.Fields(fields[1]), Author: fields[2], Date: fields[3], Subject: fields[4]}, nil
}

func parseCommitRecords(output []byte) ([]Commit, error) {
	var commits []Commit
	for _, record := range strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00") {
		record = strings.TrimPrefix(record, "\n")
		if record == "" {
			continue
		}
		commit, err := parseCommitRecord(record)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

func ResolveCommit(ctx context.Context, runner *Runner, ref string) (string, error) {
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("resolve %q: %w", ref, ErrInvalidName)
	}
	output, err := runner.Read(ctx, "rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", ref, err)
	}
	return strings.TrimSpace(string(output)), nil
}

type HistoryCursor struct {
	Ref     string
	TipOID  string
	command *exec.Cmd
	reader  *bufio.Reader
	cancel  context.CancelFunc
	done    bool
}

func OpenHistory(ctx context.Context, runner *Runner, ref string) (*HistoryCursor, error) {
	tip, err := ResolveCommit(ctx, runner, ref)
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	streamContext, cancel := context.WithCancel(context.Background())
	command := runner.Command(streamContext, []string{"GIT_OPTIONAL_LOCKS=0"}, "log", "--topo-order", "-z", "--no-color", commitFormat, tip, "--")
	command.Stderr = io.Discard
	stdout, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open history pipe: %w", err)
	}
	if err := command.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start history traversal: %w", err)
	}
	return &HistoryCursor{Ref: ref, TipOID: tip, command: command, reader: bufio.NewReader(stdout), cancel: cancel}, nil
}

func (c *HistoryCursor) Done() bool { return c.done }

func (c *HistoryCursor) Next(limit int) ([]Commit, error) {
	var commits []Commit
	for len(commits) < limit && !c.done {
		record, err := c.reader.ReadString(0)
		record = strings.TrimSuffix(strings.TrimPrefix(record, "\n"), "\x00")
		if record != "" {
			commit, parseErr := parseCommitRecord(record)
			if parseErr != nil {
				return commits, parseErr
			}
			commits = append(commits, commit)
		}
		if errors.Is(err, io.EOF) {
			c.done = true
			if waitErr := c.command.Wait(); waitErr != nil {
				return commits, fmt.Errorf("history traversal of %s: %w", c.Ref, waitErr)
			}
			break
		}
		if err != nil {
			return commits, fmt.Errorf("read history page: %w", err)
		}
	}
	return commits, nil
}

func (c *HistoryCursor) Close() {
	c.cancel()
	if !c.done {
		c.done = true
		_ = c.command.Wait()
	}
}

func LoadCommits(ctx context.Context, runner *Runner, oids []string) ([]Commit, error) {
	if err := ValidateOIDs(oids); err != nil {
		return nil, err
	}
	args := append([]string{"show", "-s", "-z", "--no-color", commitFormat}, oids...)
	output, err := runner.Read(ctx, append(args, "--")...)
	if err != nil {
		return nil, fmt.Errorf("load %d commit(s): %w", len(oids), err)
	}
	return parseCommitRecords(output)
}

func ShowCommit(ctx context.Context, runner *Runner, oid string) (string, error) {
	if err := ValidateOIDs([]string{oid}); err != nil {
		return "", err
	}
	output, err := runner.Read(ctx, "show", "--no-color", "--no-ext-diff", "--stat", "--patch", "--format=fuller", oid, "--")
	if err != nil {
		return "", fmt.Errorf("show commit %s: %w", oid, err)
	}
	return string(output), nil
}

func CompareCommits(ctx context.Context, runner *Runner, older, newer string) (string, error) {
	if err := ValidateOIDs([]string{older, newer}); err != nil {
		return "", err
	}
	output, err := runner.Read(ctx, "diff", "--no-color", "--no-ext-diff", "--stat", "--patch", older, newer, "--")
	if err != nil {
		return "", fmt.Errorf("compare %s with %s: %w", older, newer, err)
	}
	return string(output), nil
}

func HeadMessage(ctx context.Context, runner *Runner) (string, error) {
	output, err := runner.Read(ctx, "log", "-1", "--format=%B", "HEAD", "--")
	if err != nil {
		return "", fmt.Errorf("read HEAD message: %w", err)
	}
	return string(output), nil
}

func StagedSummary(ctx context.Context, runner *Runner) (string, error) {
	output, err := runner.Read(ctx, "diff", "--cached", "--no-color", "--no-ext-diff", "--stat", "--summary")
	if err != nil {
		return "", fmt.Errorf("summarize staged changes: %w", err)
	}
	return string(output), nil
}

func WriteMessageFile(message string) (string, func(), error) {
	handle, err := os.CreateTemp("", "gitflow-message-")
	if err != nil {
		return "", nil, fmt.Errorf("create commit message file: %w", err)
	}
	defer handle.Close()
	if _, err := handle.WriteString(message); err != nil {
		os.Remove(handle.Name())
		return "", nil, fmt.Errorf("write commit message file: %w", err)
	}
	return handle.Name(), func() { os.Remove(handle.Name()) }, nil
}

func CommitInvocation(messagePath string, amend bool) Invocation {
	args := []string{"commit", "--file=" + messagePath}
	if amend {
		args = append(args, "--amend")
	}
	return Invocation{Args: args}
}
