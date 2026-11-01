package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	nullDevicePath         = "/dev/null"
	noIndexDifferencesExit = 1
	renameStatus           = 'R'
	copyStatus             = 'C'
	addedStatus            = 'A'
	modifiedStatus         = 'M'
	deletedStatus          = 'D'
	typeChangeStatus       = 'T'
	rawEntryPrefix         = ":"
	rawStatusField         = 4
	rawFieldCount          = 5
	numstatSeparator       = "\t"
	binaryNumstatPrefix    = "-" + numstatSeparator + "-" + numstatSeparator
	numstatFieldCount      = 3
	submoduleMode          = "160000"
	indexName              = "index"
	dryRunIndexPattern     = "gitflow-apply-check-*"
	indexFileVariable      = "GIT_INDEX_FILE="
	indexFileMode          = 0o600
	reasonRename           = "rename"
	reasonCopy             = "copy"
	reasonTypeChange       = "file type change"
	reasonSubmodule        = "submodule change"
	reasonSymlink          = "symbolic link"
	reasonBinary           = "binary file"
	reasonUnknownChange    = "unsupported change type"
)

var (
	applyToIndexArgs     = []string{"apply", "--index", "--whitespace=nowarn"}
	applyCheckIndexArgs  = []string{"apply", "--cached", "--whitespace=nowarn"}
	statusUnsupportedFor = map[byte]string{renameStatus: reasonRename, copyStatus: reasonCopy, typeChangeStatus: reasonTypeChange}
)

var readOnlyDiffArgs = []string{"diff", "--no-ext-diff", "--no-color"}

type StagedUnstaged struct {
	Staged   string
	Unstaged string
}

type CommitFile struct {
	Path              string
	OriginalPath      string
	Status            byte
	UnsupportedReason string
}

type FileSelection struct {
	OID          string
	Parent       string
	Path         string
	OriginalPath string
	Status       byte
}

type HunkSelection struct {
	Path   string
	Staged bool
	Hunk   int
	Lines  []int
}

type ApplyStep struct {
	Selection FileSelection
	Patch     string
}

type ApplyReport struct {
	Applied []FileSelection
	Failed  []FileSelection
	Skipped []FileSelection
}

type Differ struct {
	runner *Runner
}

func NewDiffer(runner *Runner) *Differ {
	return &Differ{runner: runner}
}

func (d *Differ) WorktreeDiff(ctx context.Context, file FileChange) (StagedUnstaged, error) {
	var diff StagedUnstaged
	if file.HasStaged() {
		paths := []string{file.Path}
		if file.OriginalPath != "" {
			paths = []string{file.OriginalPath, file.Path}
		}
		output, err := d.runner.Read(ctx, append(append(append([]string{}, readOnlyDiffArgs...), "--cached", "-M", "--"), paths...)...)
		if err != nil {
			return StagedUnstaged{}, fmt.Errorf("read staged diff of %s: %w", file.Path, err)
		}
		diff.Staged = string(output)
	}
	if file.Kind == KindUntracked {
		output, err := d.runner.Read(ctx, append(append([]string{}, readOnlyDiffArgs...), "--no-index", "--", nullDevicePath, file.Path)...)
		var commandErr *CommandError
		if err != nil && !(errors.As(err, &commandErr) && commandErr.ExitCode == noIndexDifferencesExit) {
			return StagedUnstaged{}, fmt.Errorf("read untracked diff of %s: %w", file.Path, err)
		}
		diff.Unstaged = string(output)
		return diff, nil
	}
	if file.HasUnstaged() {
		output, err := d.runner.Read(ctx, append(append([]string{}, readOnlyDiffArgs...), "--", file.Path)...)
		if err != nil {
			return StagedUnstaged{}, fmt.Errorf("read unstaged diff of %s: %w", file.Path, err)
		}
		diff.Unstaged = string(output)
	}
	return diff, nil
}

func commitTreeArgs(oid, parent string) []string {
	if parent == "" {
		return []string{"--root", oid}
	}
	return []string{parent, oid}
}

func (d *Differ) CommitFiles(ctx context.Context, oid, parent string) ([]CommitFile, error) {
	output, err := d.runner.Read(ctx, append([]string{"diff-tree", "-r", "-z", "-M", "--no-commit-id", "--raw", "--numstat"}, commitTreeArgs(oid, parent)...)...)
	if err != nil {
		return nil, fmt.Errorf("list files of commit %s against %s: %w", oid, parent, err)
	}
	fields := splitNulTerminated(output)
	var files []CommitFile
	position := 0
	for len(output) > 0 && position < len(fields) && strings.HasPrefix(fields[position], rawEntryPrefix) {
		meta := strings.Fields(fields[position])
		if len(meta) != rawFieldCount || meta[rawStatusField] == "" || position+1 >= len(fields) {
			return nil, fmt.Errorf("list files of commit %s: malformed entry at %d", oid, position)
		}
		file := CommitFile{Status: meta[rawStatusField][0], Path: fields[position+1]}
		position += 2
		if file.Status == renameStatus || file.Status == copyStatus {
			if position >= len(fields) {
				return nil, fmt.Errorf("list files of commit %s: missing destination of %s", oid, file.Path)
			}
			file.OriginalPath, file.Path = file.Path, fields[position]
			position++
		}
		oldMode, newMode := strings.TrimPrefix(meta[0], rawEntryPrefix), meta[1]
		switch {
		case statusUnsupportedFor[file.Status] != "":
			file.UnsupportedReason = statusUnsupportedFor[file.Status]
		case file.Status != addedStatus && file.Status != modifiedStatus && file.Status != deletedStatus:
			file.UnsupportedReason = reasonUnknownChange
		case oldMode == submoduleMode || newMode == submoduleMode:
			file.UnsupportedReason = reasonSubmodule
		case oldMode == symlinkMode || newMode == symlinkMode:
			file.UnsupportedReason = reasonSymlink
		}
		files = append(files, file)
	}
	for index := range files {
		if position >= len(fields) || len(strings.SplitN(fields[position], numstatSeparator, numstatFieldCount)) != numstatFieldCount {
			return nil, fmt.Errorf("list files of commit %s: missing line counts of %s", oid, files[index].Path)
		}
		if strings.HasPrefix(fields[position], binaryNumstatPrefix) && files[index].UnsupportedReason == "" {
			files[index].UnsupportedReason = reasonBinary
		}
		if strings.HasSuffix(fields[position], numstatSeparator) {
			position += 2
		}
		position++
	}
	return files, nil
}

func (d *Differ) FilePatch(ctx context.Context, selection FileSelection) (string, error) {
	paths := []string{selection.Path}
	if selection.OriginalPath != "" {
		paths = []string{selection.OriginalPath, selection.Path}
	}
	args := append(append([]string{"diff-tree", "-r", "-p", "-M", "--no-commit-id", "--no-color", "--no-ext-diff"}, commitTreeArgs(selection.OID, selection.Parent)...), "--")
	output, err := d.runner.Read(ctx, append(args, paths...)...)
	if err != nil {
		return "", fmt.Errorf("read patch of %s in commit %s against %s: %w", selection.Path, selection.OID, selection.Parent, err)
	}
	return string(output), nil
}

func (d *Differ) ApplyQueue(ctx context.Context, steps []ApplyStep, dryRun bool) (ApplyReport, error) {
	applyArgs := applyToIndexArgs
	var env []string
	if dryRun {
		indexPaths, err := d.runner.GitPaths(ctx, indexName)
		if err != nil {
			return ApplyReport{}, fmt.Errorf("check selected files apply in order: %w", err)
		}
		indexPath := indexPaths[0]
		if !filepath.IsAbs(indexPath) {
			indexPath = filepath.Join(d.runner.WorkTree(), indexPath)
		}
		scratch, err := os.MkdirTemp("", dryRunIndexPattern)
		if err != nil {
			return ApplyReport{}, fmt.Errorf("check selected files apply in order: create scratch index: %w", err)
		}
		defer os.RemoveAll(scratch)
		scratchIndex := filepath.Join(scratch, indexName)
		content, err := os.ReadFile(indexPath)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return ApplyReport{}, fmt.Errorf("check selected files apply in order: read index: %w", err)
		default:
			if err := os.WriteFile(scratchIndex, content, indexFileMode); err != nil {
				return ApplyReport{}, fmt.Errorf("check selected files apply in order: copy index: %w", err)
			}
		}
		applyArgs, env = applyCheckIndexArgs, []string{indexFileVariable + scratchIndex}
	}
	var report ApplyReport
	for position, step := range steps {
		if _, err := d.runner.Execute(ctx, Invocation{Args: applyArgs, Stdin: []byte(step.Patch), Env: env}); err != nil {
			report.Failed = []FileSelection{step.Selection}
			for _, pending := range steps[position+1:] {
				report.Skipped = append(report.Skipped, pending.Selection)
			}
			return report, fmt.Errorf("apply %s from commit %s: %w", step.Selection.Path, step.Selection.OID, err)
		}
		report.Applied = append(report.Applied, step.Selection)
	}
	return report, nil
}

func (d *Differ) SelectionPatch(ctx context.Context, selection HunkSelection) ([]byte, error) {
	return nil, errNotImplemented
}
