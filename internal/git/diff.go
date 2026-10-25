package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const untrackedPreviewLimit = 64 * 1024

var (
	ErrNothingSelected = errors.New("no changed lines selected")
	ErrStale           = errors.New("file changed since it was displayed")
	ErrUntrackedTarget = errors.New("untracked files are never discarded")
)

var diffArgs = []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "--no-relative", "--src-prefix=a/", "--dst-prefix=b/", "-U3"}

type DiffLine struct {
	Kind byte
	Text string
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Section  string
	Lines    []DiffLine
}

type FileDiff struct {
	Header     []string
	Hunks      []Hunk
	Binary     bool
	NewFile    bool
	Deleted    bool
	Renamed    bool
	ModeChange bool
}

type LineRef struct {
	Hunk int
	Line int
}

type LineSelection map[LineRef]bool

func (d FileDiff) Empty() bool { return len(d.Header) == 0 }

func (d FileDiff) PartialUnsupportedReason() string {
	switch {
	case d.Binary:
		return "binary file: file-level staging only"
	case d.NewFile:
		return "added file: file-level staging only"
	case d.Deleted:
		return "deleted file: file-level staging only"
	case d.Renamed:
		return "renamed file: file-level staging only"
	case d.ModeChange:
		return "mode change: file-level staging only"
	}
	return ""
}

func (d FileDiff) String() string {
	var text strings.Builder
	for _, line := range d.Header {
		text.WriteString(line + "\n")
	}
	for _, hunk := range d.Hunks {
		text.WriteString(hunkHeader(hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount, hunk.Section) + "\n")
		for _, line := range hunk.Lines {
			text.WriteString(string(line.Kind) + line.Text + "\n")
		}
	}
	return text.String()
}

func hunkHeader(oldStart, oldCount, newStart, newCount int, section string) string {
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
	if section != "" {
		header += " " + section
	}
	return header
}

func ParseDiff(raw string) (FileDiff, error) {
	var diff FileDiff
	if raw == "" {
		return diff, nil
	}
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	lineIndex := 0
	for ; lineIndex < len(lines) && !strings.HasPrefix(lines[lineIndex], "@@ "); lineIndex++ {
		line := lines[lineIndex]
		diff.Header = append(diff.Header, line)
		switch {
		case strings.HasPrefix(line, "Binary files ") || line == "GIT binary patch":
			diff.Binary = true
		case strings.HasPrefix(line, "new file mode"):
			diff.NewFile = true
		case strings.HasPrefix(line, "deleted file mode"):
			diff.Deleted = true
		case strings.HasPrefix(line, "rename from") || strings.HasPrefix(line, "copy from"):
			diff.Renamed = true
		case strings.HasPrefix(line, "old mode"):
			diff.ModeChange = true
		}
	}
	for ; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		if strings.HasPrefix(line, "@@ ") {
			closing := strings.Index(line[3:], " @@")
			if closing < 0 {
				return FileDiff{}, fmt.Errorf("parse hunk header %q", line)
			}
			ranges := strings.Fields(line[3 : 3+closing])
			if len(ranges) != 2 {
				return FileDiff{}, fmt.Errorf("parse hunk header %q", line)
			}
			hunk := Hunk{Section: strings.TrimSpace(line[3+closing+3:])}
			for rangeIndex, rangeText := range ranges {
				startText, countText, hasCount := strings.Cut(rangeText[1:], ",")
				start, startErr := strconv.Atoi(startText)
				count := 1
				var countErr error
				if hasCount {
					count, countErr = strconv.Atoi(countText)
				}
				if startErr != nil || countErr != nil {
					return FileDiff{}, fmt.Errorf("parse hunk range %q", rangeText)
				}
				if rangeIndex == 0 {
					hunk.OldStart, hunk.OldCount = start, count
				} else {
					hunk.NewStart, hunk.NewCount = start, count
				}
			}
			diff.Hunks = append(diff.Hunks, hunk)
			continue
		}
		if len(diff.Hunks) == 0 || line == "" {
			continue
		}
		current := &diff.Hunks[len(diff.Hunks)-1]
		current.Lines = append(current.Lines, DiffLine{Kind: line[0], Text: line[1:]})
	}
	return diff, nil
}

func (d FileDiff) BuildPatch(selection LineSelection, reverse bool) (string, error) {
	if reason := d.PartialUnsupportedReason(); reason != "" {
		return "", errors.New(reason)
	}
	var patch strings.Builder
	for _, line := range d.Header {
		patch.WriteString(line + "\n")
	}
	emittedHunks := 0
	appliedDelta := 0
	for hunkIndex, hunk := range d.Hunks {
		var body strings.Builder
		oldCount, newCount := 0, 0
		hasSelectedChange := false
		previousKept := false
		for lineIndex, line := range hunk.Lines {
			selected := selection[LineRef{Hunk: hunkIndex, Line: lineIndex}]
			emitted := line.Kind
			switch line.Kind {
			case '+':
				switch {
				case selected:
					hasSelectedChange = true
				case reverse:
					emitted = ' '
				default:
					emitted = 0
				}
			case '-':
				switch {
				case selected:
					hasSelectedChange = true
				case reverse:
					emitted = 0
				default:
					emitted = ' '
				}
			case '\\':
				if previousKept {
					body.WriteString(string(line.Kind) + line.Text + "\n")
				}
				continue
			}
			previousKept = emitted != 0
			if emitted == 0 {
				continue
			}
			if emitted != '+' {
				oldCount++
			}
			if emitted != '-' {
				newCount++
			}
			body.WriteString(string(emitted) + line.Text + "\n")
		}
		if !hasSelectedChange {
			continue
		}
		oldStart, newStart := hunk.OldStart, hunk.OldStart+appliedDelta
		if reverse {
			oldStart, newStart = hunk.NewStart-appliedDelta, hunk.NewStart
		}
		patch.WriteString(hunkHeader(oldStart, oldCount, newStart, newCount, hunk.Section) + "\n")
		patch.WriteString(body.String())
		appliedDelta += newCount - oldCount
		emittedHunks++
	}
	if emittedHunks == 0 {
		return "", ErrNothingSelected
	}
	return patch.String(), nil
}

func diffPaths(file FileChange) []string {
	if file.OrigPath != "" {
		return []string{file.OrigPath, file.Path}
	}
	return []string{file.Path}
}

func LoadDiff(ctx context.Context, runner *Runner, file FileChange, staged bool) (FileDiff, error) {
	args := append([]string{}, diffArgs...)
	if staged {
		args = append(args, "--cached", "-M")
	}
	args = append(append(args, "--"), diffPaths(file)...)
	output, err := runner.Read(ctx, args...)
	if err != nil {
		return FileDiff{}, fmt.Errorf("load diff for %s: %w", file.Path, err)
	}
	return ParseDiff(string(output))
}

func ReadUntrackedPreview(workTree string, path string) (string, bool, error) {
	fullPath := filepath.Join(workTree, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", false, fmt.Errorf("read untracked %s: %w", path, err)
	}
	if info.IsDir() {
		return "untracked directory", false, nil
	}
	handle, err := os.Open(fullPath)
	if err != nil {
		return "", false, fmt.Errorf("read untracked %s: %w", path, err)
	}
	defer handle.Close()
	buffer := make([]byte, untrackedPreviewLimit)
	readCount, _ := handle.Read(buffer)
	content := buffer[:readCount]
	if bytes.IndexByte(content, 0) >= 0 {
		return "", true, nil
	}
	return string(content), false, nil
}

func ApplyToIndex(ctx context.Context, runner *Runner, patch string, reverse bool) error {
	args := []string{"apply", "--cached", "--whitespace=nowarn"}
	if reverse {
		args = append(args, "--reverse")
	}
	if _, err := runner.Execute(ctx, Invocation{Args: append(append([]string{}, args...), "--check", "-"), Stdin: []byte(patch)}); err != nil {
		return fmt.Errorf("validate patch against index: %w", err)
	}
	if _, err := runner.Execute(ctx, Invocation{Args: append(args, "-"), Stdin: []byte(patch)}); err != nil {
		return fmt.Errorf("apply patch to index: %w", err)
	}
	return nil
}

func Revalidate(ctx context.Context, runner *Runner, targets []FileChange) (Status, error) {
	status, err := LoadStatus(ctx, runner)
	if err != nil {
		return Status{}, err
	}
	current := make(map[string]bool, len(status.Files))
	for _, file := range status.Files {
		current[file.Signature()] = true
	}
	for _, target := range targets {
		if !current[target.Signature()] {
			return status, fmt.Errorf("revalidate %s: %w", target.Path, ErrStale)
		}
	}
	return status, nil
}

func StageFiles(ctx context.Context, runner *Runner, files []FileChange) error {
	args := []string{"add", "-A", "--"}
	for _, file := range files {
		args = append(args, diffPaths(file)...)
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("stage %d file(s): %w", len(files), err)
	}
	return nil
}

func UnstageFiles(ctx context.Context, runner *Runner, files []FileChange, unborn bool) error {
	args := []string{"restore", "--staged", "--"}
	if unborn {
		args = []string{"rm", "--cached", "-r", "-q", "--"}
	}
	for _, file := range files {
		args = append(args, diffPaths(file)...)
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("unstage %d file(s): %w", len(files), err)
	}
	return nil
}

func DiscardWorktreeChanges(ctx context.Context, runner *Runner, files []FileChange) error {
	args := []string{"restore", "--worktree", "--"}
	for _, file := range files {
		if file.Kind == KindUntracked {
			return fmt.Errorf("discard %s: %w", file.Path, ErrUntrackedTarget)
		}
		args = append(args, file.Path)
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("discard worktree changes in %d file(s): %w", len(files), err)
	}
	return nil
}
