package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	markerOurs   = "<<<<<<<"
	markerBase   = "|||||||"
	markerSplit  = "======="
	markerTheirs = ">>>>>>>"
)

var (
	ErrUnresolvedRegions = errors.New("conflict regions remain unresolved")
	ErrMarkersRemain     = errors.New("conflict markers remain in file")
	ErrNoTextSides       = errors.New("conflict has no text sides to choose from")
)

type Choice int

const (
	ChoiceNone Choice = iota
	ChoiceOurs
	ChoiceTheirs
	ChoiceBoth
)

type ConflictRegion struct {
	Start       int
	End         int
	Ours        []string
	Base        []string
	HasBase     bool
	Theirs      []string
	OursLabel   string
	TheirsLabel string
}

type ConflictFile struct {
	Path            string
	Stages          map[int]bool
	Binary          bool
	Missing         bool
	Lines           []string
	TrailingNewline bool
	Regions         []ConflictRegion
}

func (c ConflictFile) TextSidesSupported() error {
	if c.Binary || c.Missing || !c.Stages[2] || !c.Stages[3] || len(c.Regions) == 0 {
		return ErrNoTextSides
	}
	return nil
}

func LoadConflict(ctx context.Context, runner *Runner, path string) (ConflictFile, error) {
	output, err := runner.Read(ctx, "ls-files", "--unmerged", "-z", "--", path)
	if err != nil {
		return ConflictFile{}, fmt.Errorf("read conflict stages of %s: %w", path, err)
	}
	conflict := ConflictFile{Path: path, Stages: map[int]bool{}}
	for _, record := range strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00") {
		metadata, _, found := strings.Cut(record, "\t")
		fields := strings.Fields(metadata)
		if !found || len(fields) != 3 || len(fields[2]) != 1 {
			continue
		}
		conflict.Stages[int(fields[2][0]-'0')] = true
	}
	content, err := os.ReadFile(filepath.Join(runner.WorkTree(), path))
	if errors.Is(err, os.ErrNotExist) {
		conflict.Missing = true
		return conflict, nil
	}
	if err != nil {
		return ConflictFile{}, fmt.Errorf("read conflicted file %s: %w", path, err)
	}
	if bytes.IndexByte(content, 0) >= 0 {
		conflict.Binary = true
		return conflict, nil
	}
	text := string(content)
	conflict.TrailingNewline = strings.HasSuffix(text, "\n")
	conflict.Lines = strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	var current *ConflictRegion
	section := ""
	for lineIndex, line := range conflict.Lines {
		switch {
		case strings.HasPrefix(line, markerOurs) && current == nil:
			current = &ConflictRegion{Start: lineIndex, OursLabel: strings.TrimSpace(strings.TrimPrefix(line, markerOurs))}
			section = "ours"
		case strings.HasPrefix(line, markerBase) && current != nil && section == "ours":
			current.HasBase = true
			section = "base"
		case line == markerSplit && current != nil && section != "theirs":
			section = "theirs"
		case strings.HasPrefix(line, markerTheirs) && current != nil && section == "theirs":
			current.End = lineIndex
			current.TheirsLabel = strings.TrimSpace(strings.TrimPrefix(line, markerTheirs))
			conflict.Regions = append(conflict.Regions, *current)
			current = nil
		case current != nil:
			switch section {
			case "ours":
				current.Ours = append(current.Ours, line)
			case "base":
				current.Base = append(current.Base, line)
			case "theirs":
				current.Theirs = append(current.Theirs, line)
			}
		}
	}
	return conflict, nil
}

func (c ConflictFile) Resolve(choices map[int]Choice) (string, error) {
	if err := c.TextSidesSupported(); err != nil {
		return "", err
	}
	var resolved []string
	cursor := 0
	for regionIndex, region := range c.Regions {
		resolved = append(resolved, c.Lines[cursor:region.Start]...)
		switch choices[regionIndex] {
		case ChoiceOurs:
			resolved = append(resolved, region.Ours...)
		case ChoiceTheirs:
			resolved = append(resolved, region.Theirs...)
		case ChoiceBoth:
			resolved = append(append(resolved, region.Ours...), region.Theirs...)
		default:
			return "", fmt.Errorf("resolve %s region %d: %w", c.Path, regionIndex+1, ErrUnresolvedRegions)
		}
		cursor = region.End + 1
	}
	resolved = append(resolved, c.Lines[cursor:]...)
	text := strings.Join(resolved, "\n")
	if c.TrailingNewline && len(resolved) > 0 {
		text += "\n"
	}
	return text, nil
}

func MarkResolved(ctx context.Context, runner *Runner, path string, resolvedContent *string) error {
	fullPath := filepath.Join(runner.WorkTree(), path)
	if resolvedContent != nil {
		info, err := os.Stat(fullPath)
		if err != nil {
			return fmt.Errorf("mark %s resolved: %w", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(*resolvedContent), info.Mode().Perm()); err != nil {
			return fmt.Errorf("write resolution for %s: %w", path, err)
		}
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("mark %s resolved: %w", path, err)
	}
	if bytes.IndexByte(content, 0) < 0 {
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, markerOurs) || strings.HasPrefix(line, markerTheirs) || line == markerSplit {
				return fmt.Errorf("mark %s resolved: %w", path, ErrMarkersRemain)
			}
		}
	}
	if _, err := runner.Run(ctx, "add", "--", path); err != nil {
		return fmt.Errorf("stage resolution of %s: %w", path, err)
	}
	return nil
}

func ConfiguredEditor(ctx context.Context, runner *Runner) (string, error) {
	output, err := runner.Read(ctx, "var", "GIT_EDITOR")
	if err != nil {
		return "", fmt.Errorf("read configured Git editor: %w", err)
	}
	editor := strings.TrimSpace(string(output))
	if editor == "" {
		return "", errors.New("read configured Git editor: no editor configured (set core.editor or GIT_EDITOR)")
	}
	return editor, nil
}

func EditorInvocation(editor, path string) []string {
	return []string{"sh", "-c", editor + ` "$@"`, editor, path}
}
