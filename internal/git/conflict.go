package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ConflictMode int

const (
	ConflictCherryPick ConflictMode = iota
	ConflictRebase
	ConflictMerge
	ConflictStashRestore
)

type ConflictSide int

const (
	SideBase ConflictSide = iota
	SideOurs
	SideTheirs
)

const (
	stashRestoreOursRole   = "current branch after cherry-pick"
	stashRestoreTheirsRole = "GitFlow stash"
)

const (
	RoleCurrent           = "current"
	RoleIncoming          = "incoming"
	RoleRebaseDestination = "rebase destination"
	RoleReplayed          = "replayed commit"
	RoleBase              = "common ancestor"
	baseLabelRef          = "base"
	resolveTempSuffix     = ".gitflow.tmp"
)

const (
	binaryProbeBytes      = 8000
	symlinkMode           = "120000"
	gitlinkMode           = "160000"
	markerOursStart       = "<<<<<<<"
	markerBaseStart       = "|||||||"
	markerSeparator       = "======="
	markerTheirsEnd       = ">>>>>>>"
	conflictStyleKey      = "merge.conflictStyle"
	diff3Style            = "diff3"
	zdiff3Style           = "zdiff3"
	stageRecordFields     = 3
	stageCount            = 3
	mergeFileMaxConflicts = 127
	remoteHeadSuffix      = "/HEAD"
	editorShellScript     = ` "$@"`
)

var baseConflictStyles = []string{diff3Style, zdiff3Style}

var (
	ErrNotConflicted = errors.New("file is not conflicted")
	ErrNoBase        = errors.New("Git has no common-ancestor version for this file")
	ErrEditorOnly    = errors.New("this conflict can only be resolved in the editor")
	ErrNoRegion      = errors.New("conflict region no longer exists")
	ErrMarkersRemain = errors.New("conflict markers remain in the file")
)

type ConflictFallback int

const (
	FallbackNone ConflictFallback = iota
	FallbackBinary
	FallbackDeleted
	FallbackSpecialFile
	FallbackMissingFile
	FallbackMarkersChanged
)

type Label struct {
	Ref  string
	Role string
}

func (l Label) String() string { return l.Ref + " (" + l.Role + ")" }

type SideLabels struct {
	Base   Label
	Ours   Label
	Theirs Label
}

func (l SideLabels) Of(side ConflictSide) Label {
	return [stageCount]Label{l.Base, l.Ours, l.Theirs}[side]
}

type StageEntry struct {
	Mode string
	OID  string
}

type ConflictRegion struct {
	StartLine int
	EndLine   int
	Base      []string
	Ours      []string
	Theirs    []string
	HasBase   bool
}

func (r ConflictRegion) Side(side ConflictSide) []string {
	return [stageCount][]string{r.Base, r.Ours, r.Theirs}[side]
}

type ConflictFile struct {
	Path     string
	Stages   [stageCount]StageEntry
	Fallback ConflictFallback
	Regions  []ConflictRegion
	lines    []string
}

func (f ConflictFile) HasBase() bool { return f.Stages[SideBase].OID != "" }

func parseConflictMarkers(content []byte) ([]string, []ConflictRegion, bool) {
	lines := strings.SplitAfter(string(content), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var regions []ConflictRegion
	var current *[]string
	for number, line := range lines {
		switch {
		case strings.HasPrefix(line, markerOursStart):
			if current != nil {
				return lines, nil, false
			}
			regions = append(regions, ConflictRegion{StartLine: number})
			current = &regions[len(regions)-1].Ours
		case current != nil && strings.HasPrefix(line, markerBaseStart):
			current = &regions[len(regions)-1].Base
		case current != nil && strings.TrimRight(line, "\r\n") == markerSeparator:
			current = &regions[len(regions)-1].Theirs
		case current != nil && strings.HasPrefix(line, markerTheirsEnd):
			regions[len(regions)-1].EndLine = number
			current = nil
		case current != nil:
			*current = append(*current, line)
		}
	}
	return lines, regions, current == nil
}

type ConflictReader struct {
	runner *Runner
}

func NewConflictReader(runner *Runner) *ConflictReader {
	return &ConflictReader{runner: runner}
}

func (c *ConflictReader) uniqueRef(ctx context.Context, oid string) (string, error) {
	output, err := c.runner.Read(ctx, "for-each-ref", "--points-at", oid, "--format=%(refname)", localRefPrefix, remoteRefPrefix)
	if err != nil {
		return "", fmt.Errorf("find refs at %s: %w", oid, err)
	}
	var local, remote []string
	for _, ref := range strings.Fields(string(output)) {
		switch {
		case strings.HasPrefix(ref, localRefPrefix):
			local = append(local, strings.TrimPrefix(ref, localRefPrefix))
		case !strings.HasSuffix(ref, remoteHeadSuffix):
			remote = append(remote, strings.TrimPrefix(ref, remoteRefPrefix))
		}
	}
	switch {
	case len(local) == 1:
		return local[0], nil
	case len(local) == 0 && len(remote) == 1:
		return remote[0], nil
	}
	return oid, nil
}

func (c *ConflictReader) Sides(ctx context.Context, operation Operation) (SideLabels, error) {
	labels := SideLabels{Base: Label{Ref: baseLabelRef, Role: RoleBase}}
	head, err := c.runner.Read(ctx, "rev-parse", "HEAD")
	if err != nil {
		return SideLabels{}, fmt.Errorf("label conflict sides: %w", err)
	}
	current := strings.TrimSpace(string(head))
	branch, err := c.runner.Read(ctx, "symbolic-ref", "-q", "--short", "HEAD")
	var commandErr *CommandError
	switch {
	case err == nil:
		current = strings.TrimSpace(string(branch))
	case !errors.As(err, &commandErr) || commandErr.ExitCode != 1:
		return SideLabels{}, fmt.Errorf("label conflict sides: %w", err)
	}
	switch operation.Kind {
	case OperationMerge:
		incoming, err := c.uniqueRef(ctx, operation.Commit)
		if err != nil {
			return SideLabels{}, fmt.Errorf("label merge sides: %w", err)
		}
		labels.Ours = Label{Ref: current, Role: RoleCurrent}
		labels.Theirs = Label{Ref: incoming, Role: RoleIncoming}
	case OperationRebase:
		destination, err := c.uniqueRef(ctx, operation.Onto)
		if err != nil {
			return SideLabels{}, fmt.Errorf("label rebase sides: %w", err)
		}
		labels.Ours = Label{Ref: destination, Role: RoleRebaseDestination}
		labels.Theirs = Label{Ref: operation.Commit, Role: RoleReplayed}
	case OperationCherryPick:
		labels.Ours = Label{Ref: current, Role: RoleCurrent}
		labels.Theirs = Label{Ref: operation.Commit, Role: RoleReplayed}
	default:
		return SideLabels{}, fmt.Errorf("label conflict sides: %w", ErrOperationUnsupported)
	}
	return labels, nil
}

func (c *ConflictReader) Load(ctx context.Context, path string) (ConflictFile, error) {
	output, err := c.runner.Read(ctx, "ls-files", "-u", "-z", "--", path)
	if err != nil {
		return ConflictFile{}, fmt.Errorf("read conflict stages of %s: %w", path, err)
	}
	if len(output) == 0 {
		return ConflictFile{}, fmt.Errorf("load %s: %w", path, ErrNotConflicted)
	}
	file := ConflictFile{Path: path}
	for _, record := range splitNulTerminated(output) {
		info, _, _ := strings.Cut(record, "\t")
		fields := strings.Fields(info)
		if len(fields) != stageRecordFields || len(fields[2]) != 1 || fields[2][0] < '1' || fields[2][0] > '3' {
			return ConflictFile{}, fmt.Errorf("read conflict stages of %s: unexpected record %q", path, record)
		}
		file.Stages[fields[2][0]-'1'] = StageEntry{Mode: fields[0], OID: fields[1]}
	}
	if file.Stages[SideOurs].OID == "" || file.Stages[SideTheirs].OID == "" {
		file.Fallback = FallbackDeleted
		return file, nil
	}
	for _, stage := range file.Stages {
		if stage.Mode == symlinkMode || stage.Mode == gitlinkMode {
			file.Fallback = FallbackSpecialFile
			return file, nil
		}
		if stage.OID == "" {
			continue
		}
		blob, err := c.runner.Read(ctx, "cat-file", "blob", stage.OID)
		if err != nil {
			return ConflictFile{}, fmt.Errorf("read conflict stage of %s: %w", path, err)
		}
		if bytes.IndexByte(blob[:min(len(blob), binaryProbeBytes)], 0) >= 0 {
			file.Fallback = FallbackBinary
			return file, nil
		}
	}
	content, err := os.ReadFile(filepath.Join(c.runner.workTree, path))
	if errors.Is(err, os.ErrNotExist) {
		file.Fallback = FallbackMissingFile
		return file, nil
	}
	if err != nil {
		return ConflictFile{}, fmt.Errorf("read conflicted %s: %w", path, err)
	}
	lines, worktreeRegions, closed := parseConflictMarkers(content)
	file.lines = lines
	if !closed || bytes.IndexByte(content[:min(len(content), binaryProbeBytes)], 0) >= 0 {
		file.Fallback = FallbackMarkersChanged
		return file, nil
	}
	if len(worktreeRegions) == 0 {
		return file, nil
	}
	style, err := c.runner.Read(ctx, "config", "--default", "", "--get", conflictStyleKey)
	if err != nil {
		return ConflictFile{}, fmt.Errorf("read %s: %w", conflictStyleKey, err)
	}
	base := file.Stages[SideBase].OID
	if base == "" {
		emptyBlob, err := c.runner.Execute(ctx, Invocation{Args: []string{"hash-object", "-t", "blob", "--stdin"}, Stdin: []byte{}})
		if err != nil {
			return ConflictFile{}, fmt.Errorf("hash empty base for %s: %w", path, err)
		}
		base = strings.TrimSpace(string(emptyBlob))
	}
	configuredStyle := strings.TrimSpace(string(style))
	baseStyle := zdiff3Style
	if slices.Contains(baseConflictStyles, configuredStyle) {
		baseStyle = configuredStyle
	}
	styleRegions := map[string][]ConflictRegion{}
	for _, styleName := range []string{configuredStyle, baseStyle} {
		if _, merged := styleRegions[styleName]; merged {
			continue
		}
		args := []string{"merge-file", "-p", "--object-id", file.Stages[SideOurs].OID, base, file.Stages[SideTheirs].OID}
		if slices.Contains(baseConflictStyles, styleName) {
			args = append(args, "--"+styleName)
		}
		merged, err := c.runner.Read(ctx, args...)
		var commandErr *CommandError
		if err != nil && (!errors.As(err, &commandErr) || commandErr.ExitCode < 1 || commandErr.ExitCode > mergeFileMaxConflicts) {
			return ConflictFile{}, fmt.Errorf("merge stages of %s: %w", path, err)
		}
		_, styleRegions[styleName], _ = parseConflictMarkers(merged)
	}
	sameSides := func(left, right ConflictRegion) bool {
		return slices.Equal(left.Ours, right.Ours) && slices.Equal(left.Theirs, right.Theirs)
	}
	expected := styleRegions[configuredStyle]
	next := 0
	for _, region := range worktreeRegions {
		for next < len(expected) && !sameSides(expected[next], region) {
			next++
		}
		if next == len(expected) {
			file.Fallback = FallbackMarkersChanged
			file.Regions = nil
			return file, nil
		}
		next++
		baseIndex := slices.IndexFunc(styleRegions[baseStyle], func(candidate ConflictRegion) bool { return sameSides(candidate, region) })
		region.Base = nil
		if baseIndex >= 0 && file.HasBase() {
			region.Base, region.HasBase = styleRegions[baseStyle][baseIndex].Base, true
		}
		file.Regions = append(file.Regions, region)
	}
	return file, nil
}

func (c *ConflictReader) Resolve(ctx context.Context, path string, region int, side ConflictSide) error {
	file, err := c.Load(ctx, path)
	if err != nil {
		return fmt.Errorf("choose a side in %s: %w", path, err)
	}
	if file.Fallback != FallbackNone {
		return fmt.Errorf("choose a side in %s: %w", path, ErrEditorOnly)
	}
	if region < 0 || region >= len(file.Regions) {
		return fmt.Errorf("choose a side in %s: %w", path, ErrNoRegion)
	}
	if side == SideBase && !file.Regions[region].HasBase {
		return fmt.Errorf("choose a side in %s: %w", path, ErrNoBase)
	}
	chosen := file.Regions[region]
	rewritten := append(append(slices.Clone(file.lines[:chosen.StartLine]), chosen.Side(side)...), file.lines[chosen.EndLine+1:]...)
	fullPath := filepath.Join(c.runner.workTree, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("choose a side in %s: %w", path, err)
	}
	temporary := fullPath + resolveTempSuffix
	if err := os.WriteFile(temporary, []byte(strings.Join(rewritten, "")), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write chosen side of %s: %w", path, err)
	}
	if err := os.Rename(temporary, fullPath); err != nil {
		return fmt.Errorf("replace %s with chosen side: %w", path, errors.Join(err, os.Remove(temporary)))
	}
	return nil
}

func (c *ConflictReader) MarkResolved(ctx context.Context, path string) error {
	file, err := c.Load(ctx, path)
	if err != nil {
		return fmt.Errorf("mark %s resolved: %w", path, err)
	}
	if file.Fallback == FallbackNone && len(file.Regions) > 0 {
		return fmt.Errorf("mark %s resolved: %w", path, ErrMarkersRemain)
	}
	if _, err := c.runner.Run(ctx, "add", "--", path); err != nil {
		return fmt.Errorf("mark %s resolved: %w", path, err)
	}
	return nil
}

func (c *ConflictReader) EditorCommand(ctx context.Context, path string) (*exec.Cmd, error) {
	output, err := c.runner.Read(ctx, "var", "GIT_EDITOR")
	if err != nil {
		return nil, fmt.Errorf("find configured editor: %w", err)
	}
	editor := strings.TrimSpace(string(output))
	command := exec.CommandContext(ctx, "sh", "-c", editor+editorShellScript, editor, filepath.Join(c.runner.workTree, path))
	command.Dir = c.runner.workTree
	command.Env = append(os.Environ(), c.runner.extraEnv...)
	return command, nil
}
