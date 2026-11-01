package git

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	historyFormat      = "--format=%H%x00%P%x00%an%x00%aI%x00%s"
	historyFieldCount  = 5
	detailFormat       = "--format=%H%x00%P%x00%an%x00%aI%x00%s%x00%b"
	detailFieldCount   = 6
	graphCommit        = "*"
	graphLane          = "|"
	graphEmpty         = " "
	graphSeparator     = " "
	detachedSourceName = "HEAD"
)

type HistoryCursor struct {
	SourceRef string
	Label     string
	Detached  bool
	TipOID    string
	Offset    int
	Lanes     []string
}

type CommitSummary struct {
	OID        string
	Parents    []string
	Author     string
	AuthorDate time.Time
	Subject    string
	Graph      string
}

type CommitDetail struct {
	Summary CommitSummary
	Body    string
	Patch   string
}

type HistoryPage struct {
	Commits []CommitSummary
	Next    HistoryCursor
	Done    bool
}

type HistoryPager struct {
	runner *Runner
}

func NewHistoryPager(runner *Runner) *HistoryPager {
	return &HistoryPager{runner: runner}
}

func parseCommitFields(fields []string) (CommitSummary, error) {
	authorDate, err := time.Parse(time.RFC3339, fields[3])
	if err != nil {
		return CommitSummary{}, fmt.Errorf("parse author date of %s: %w", fields[0], err)
	}
	return CommitSummary{OID: fields[0], Parents: strings.Fields(fields[1]), Author: fields[2], AuthorDate: authorDate, Subject: fields[4]}, nil
}

func (h *HistoryPager) resolveSource(ctx context.Context, sourceRef string) (string, error) {
	output, err := h.runner.Read(ctx, "rev-parse", "--verify", "-q", sourceRef+"^{commit}")
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == missingRefExit {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", sourceRef, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (h *HistoryPager) Open(ctx context.Context, sourceRef string) (HistoryCursor, error) {
	cursor := HistoryCursor{SourceRef: sourceRef, Label: strings.TrimPrefix(sourceRef, localRefPrefix)}
	if sourceRef == detachedSourceName {
		output, err := h.runner.Read(ctx, "symbolic-ref", "-q", detachedSourceName)
		var commandErr *CommandError
		switch {
		case errors.As(err, &commandErr) && commandErr.ExitCode == missingRefExit:
			cursor.Detached = true
		case err != nil:
			return HistoryCursor{}, fmt.Errorf("open history: read HEAD: %w", err)
		default:
			cursor.SourceRef = strings.TrimSpace(string(output))
			cursor.Label = strings.TrimPrefix(cursor.SourceRef, localRefPrefix)
		}
	}
	tipOID, err := h.resolveSource(ctx, cursor.SourceRef)
	if err != nil {
		return HistoryCursor{}, fmt.Errorf("open history: %w", err)
	}
	cursor.TipOID = tipOID
	return cursor, nil
}

func (h *HistoryPager) Page(ctx context.Context, cursor HistoryCursor, limit int) (HistoryPage, error) {
	if cursor.TipOID == "" {
		return HistoryPage{Next: cursor, Done: true}, nil
	}
	output, err := h.runner.Read(ctx, "log", "--topo-order", "-z", "--no-color", historyFormat, fmt.Sprintf("--skip=%d", cursor.Offset), fmt.Sprintf("--max-count=%d", limit), cursor.TipOID, "--")
	if err != nil {
		return HistoryPage{}, fmt.Errorf("read history of %s at %s: %w", cursor.Label, cursor.TipOID, err)
	}
	fields := splitNulTerminated(output)
	if len(output) == 0 {
		fields = nil
	}
	if len(fields)%historyFieldCount != 0 {
		return HistoryPage{}, fmt.Errorf("read history of %s: unexpected field count %d", cursor.Label, len(fields))
	}
	page := HistoryPage{Next: cursor}
	lanes := slices.Clone(cursor.Lanes)
	for start := 0; start < len(fields); start += historyFieldCount {
		commit, err := parseCommitFields(fields[start : start+historyFieldCount])
		if err != nil {
			return HistoryPage{}, fmt.Errorf("read history of %s: %w", cursor.Label, err)
		}
		column := slices.Index(lanes, commit.OID)
		if column < 0 {
			column = slices.Index(lanes, "")
			if column < 0 {
				column = len(lanes)
				lanes = append(lanes, "")
			}
			lanes[column] = commit.OID
		}
		cells := make([]string, len(lanes))
		for lane, laneOID := range lanes {
			switch {
			case lane == column:
				cells[lane] = graphCommit
			case laneOID == "":
				cells[lane] = graphEmpty
			default:
				cells[lane] = graphLane
			}
		}
		commit.Graph = strings.Join(cells, graphSeparator)
		lanes[column] = ""
		for position, parent := range commit.Parents {
			existing := slices.Index(lanes, parent)
			switch {
			case position == 0 && existing > column:
				lanes[existing], lanes[column] = "", parent
			case existing >= 0:
			case position == 0:
				lanes[column] = parent
			case slices.Contains(lanes, ""):
				lanes[slices.Index(lanes, "")] = parent
			default:
				lanes = append(lanes, parent)
			}
		}
		for len(lanes) > 0 && lanes[len(lanes)-1] == "" {
			lanes = lanes[:len(lanes)-1]
		}
		page.Commits = append(page.Commits, commit)
	}
	page.Next.Offset += len(page.Commits)
	page.Next.Lanes = lanes
	page.Done = len(page.Commits) < limit
	return page, nil
}

func (h *HistoryPager) Stale(ctx context.Context, cursor HistoryCursor) (bool, error) {
	tipOID, err := h.resolveSource(ctx, cursor.SourceRef)
	if err != nil {
		return false, fmt.Errorf("check history of %s: %w", cursor.Label, err)
	}
	return tipOID != cursor.TipOID, nil
}

func (h *HistoryPager) Compare(ctx context.Context, leftRef, rightRef string) ([]CommitSummary, error) {
	return nil, errNotImplemented
}

func (h *HistoryPager) Detail(ctx context.Context, oid string) (CommitDetail, error) {
	header, err := h.runner.Read(ctx, "show", "-s", "--no-color", detailFormat, oid, "--")
	if err != nil {
		return CommitDetail{}, fmt.Errorf("read commit %s: %w", oid, err)
	}
	fields := strings.SplitN(strings.TrimSuffix(string(header), "\n"), "\x00", detailFieldCount)
	if len(fields) != detailFieldCount {
		return CommitDetail{}, fmt.Errorf("read commit %s: unexpected field count %d", oid, len(fields))
	}
	summary, err := parseCommitFields(fields[:historyFieldCount])
	if err != nil {
		return CommitDetail{}, fmt.Errorf("read commit %s: %w", oid, err)
	}
	patch, err := h.runner.Read(ctx, "show", "--format=", "--no-color", "--no-ext-diff", "-M", "--stat", "--patch", oid, "--")
	if err != nil {
		return CommitDetail{}, fmt.Errorf("read changes of commit %s: %w", oid, err)
	}
	return CommitDetail{Summary: summary, Body: strings.TrimSpace(fields[historyFieldCount]), Patch: string(patch)}, nil
}
