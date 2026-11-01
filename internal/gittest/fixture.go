package gittest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
)

const (
	directoryMode      = 0o755
	fileMode           = 0o644
	hookMode           = 0o755
	remoteName         = "origin"
	dirtyLineSuffix    = "dirty\n"
	detachedHeadExit   = 1
	detachedHeadMarker = "detached HEAD\n"
	importEpoch        = 1767225600
	importIdentity     = "Fixture User <fixture@example.invalid>"
	importedFile       = "history.txt"
	importedSideFile   = "side.txt"
	importSideRef      = "refs/heads/side"
	importMainRef      = "refs/heads/main"
	FeatureBranch      = "feature/payments"
	fixtureConfig      = "[user]\n\tname = Fixture User\n\temail = fixture@example.invalid\n[init]\n\tdefaultBranch = main\n[commit]\n\tgpgsign = false\n"
)

var (
	ConflictBase = map[string]string{"conflict.txt": "a\nb\nc\n"}
	ConflictMain = map[string]string{"conflict.txt": "a\nMAIN\nc\n"}
	FeatureOne   = map[string]string{"conflict.txt": "a\nFEATURE\nc\n"}
	FeatureTwo   = map[string]string{"conflict.txt": "a\nFEATURE\nc\n", "second.txt": "second\n"}
)

var operationMarkers = []string{"rebase-merge", "rebase-apply", "sequencer", "CHERRY_PICK_HEAD", "MERGE_HEAD", "REVERT_HEAD"}

type Fixture struct {
	t      testing.TB
	Dir    string
	Remote string
	Env    []string
	Runner *git.Runner
	root   string
}

func NewFixture(t testing.TB) *Fixture {
	t.Helper()
	root := t.TempDir()
	workTree := filepath.Join(root, "repo")
	if err := os.Mkdir(workTree, directoryMode); err != nil {
		t.Fatalf("create work tree: %v", err)
	}
	globalConfig := filepath.Join(root, "gitconfig")
	if err := os.WriteFile(globalConfig, []byte(fixtureConfig), fileMode); err != nil {
		t.Fatalf("write fixture git config: %v", err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locate git: %v", err)
	}
	env := []string{
		"GIT_CONFIG_GLOBAL=" + globalConfig,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_EDITOR=true",
		"GIT_SEQUENCE_EDITOR=true",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	}
	fixture := &Fixture{t: t, Dir: workTree, Env: env, Runner: git.NewRunner(gitPath, workTree, env), root: root}
	fixture.Git("init", "-q", "-b", "main")
	return fixture
}

func (fixture *Fixture) Git(args ...string) string {
	fixture.t.Helper()
	output, err := fixture.Runner.Run(context.Background(), args...)
	if err != nil {
		fixture.t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(output)
}

func (fixture *Fixture) GitFails(args ...string) {
	fixture.t.Helper()
	if _, err := fixture.Runner.Run(context.Background(), args...); err == nil {
		fixture.t.Fatalf("git %s succeeded, want failure", strings.Join(args, " "))
	}
}

func (fixture *Fixture) Write(path, content string) {
	fixture.t.Helper()
	fullPath := filepath.Join(fixture.Dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), directoryMode); err != nil {
		fixture.t.Fatalf("create parent of %s: %v", path, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), fileMode); err != nil {
		fixture.t.Fatalf("write %s: %v", path, err)
	}
}

func (fixture *Fixture) Commit(message string, files map[string]string) string {
	fixture.t.Helper()
	for path, content := range files {
		fixture.Write(path, content)
		fixture.Git("add", "--", path)
	}
	fixture.Git("commit", "-q", "--allow-empty", "-m", message)
	return strings.TrimSuffix(fixture.Git("rev-parse", "HEAD"), "\n")
}

func (fixture *Fixture) ImportHistory(commits, mergeEvery int) string {
	fixture.t.Helper()
	var stream strings.Builder
	mark := 0
	mainMark := 0
	for number := 1; number <= commits; number++ {
		sideMark := 0
		if mergeEvery > 0 && number%mergeEvery == 0 && mainMark > 0 {
			mark++
			sideMark = mark
			fmt.Fprintf(&stream, "commit %s\nmark :%d\ncommitter %s %d +0000\ndata <<END\nside %d\nEND\nfrom :%d\nM 644 inline %s\ndata <<END\n%d\nEND\n\n", importSideRef, sideMark, importIdentity, importEpoch+mark, number, mainMark, importedSideFile, number)
		}
		mark++
		fmt.Fprintf(&stream, "commit %s\nmark :%d\ncommitter %s %d +0000\ndata <<END\ncommit %d\nEND\n", importMainRef, mark, importIdentity, importEpoch+mark, number)
		if mainMark > 0 {
			fmt.Fprintf(&stream, "from :%d\n", mainMark)
		}
		if sideMark > 0 {
			fmt.Fprintf(&stream, "merge :%d\n", sideMark)
		}
		fmt.Fprintf(&stream, "M 644 inline %s\ndata <<END\n%d\nEND\n\n", importedFile, number)
		mainMark = mark
	}
	if _, err := fixture.Runner.Execute(context.Background(), git.Invocation{Args: []string{"fast-import", "--quiet"}, Stdin: []byte(stream.String())}); err != nil {
		fixture.t.Fatalf("import %d commits: %v", commits, err)
	}
	fixture.Git("reset", "-q", "--hard", importMainRef)
	fixture.Git("update-ref", "-d", importSideRef)
	return strings.TrimSuffix(fixture.Git("rev-parse", importMainRef), "\n")
}

func (fixture *Fixture) ConflictingBranches(base, main map[string]string, featureCommits ...map[string]string) {
	fixture.t.Helper()
	fixture.Commit("base", base)
	fixture.Git("switch", "-q", "-c", FeatureBranch)
	for number, files := range featureCommits {
		fixture.Commit(fmt.Sprintf("feature %d", number+1), files)
	}
	fixture.Git("switch", "-q", "main")
	fixture.Commit("main", main)
}

func NewPausedFixture(t testing.TB, kind git.OperationKind) *Fixture {
	t.Helper()
	fixture := NewFixture(t)
	fixture.ConflictingBranches(ConflictBase, ConflictMain, FeatureOne, FeatureTwo)
	switch kind {
	case git.OperationMerge:
		fixture.GitFails("merge", FeatureBranch)
	case git.OperationRebase:
		fixture.Git("switch", "-q", FeatureBranch)
		fixture.GitFails("rebase", "main")
	case git.OperationCherryPick:
		fixture.GitFails("cherry-pick", FeatureBranch+"~1", FeatureBranch)
	}
	return fixture
}

func (fixture *Fixture) Hook(name, script string) {
	fixture.t.Helper()
	hookPath := filepath.Join(fixture.Dir, ".git", "hooks", name)
	if err := os.MkdirAll(filepath.Dir(hookPath), directoryMode); err != nil {
		fixture.t.Fatalf("create hooks directory: %v", err)
	}
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\n"+script), hookMode); err != nil {
		fixture.t.Fatalf("write %s hook: %v", name, err)
	}
}

func (fixture *Fixture) WithRemote() {
	fixture.t.Helper()
	remoteDir := filepath.Join(fixture.root, remoteName+".git")
	fixture.Git("init", "-q", "--bare", remoteDir)
	fixture.Git("remote", "add", remoteName, remoteDir)
	fixture.Remote = remoteDir
}

func (fixture *Fixture) Dirty(paths ...string) {
	fixture.t.Helper()
	for _, path := range paths {
		fullPath := filepath.Join(fixture.Dir, path)
		file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fileMode)
		if err != nil {
			fixture.t.Fatalf("open %s: %v", path, err)
		}
		_, writeErr := file.WriteString(dirtyLineSuffix)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			fixture.t.Fatalf("append to %s: write %v, close %v", path, writeErr, closeErr)
		}
	}
}

func (fixture *Fixture) Snapshot() string {
	fixture.t.Helper()
	var snapshot strings.Builder
	snapshot.WriteString(fixture.Git("status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all"))
	snapshot.WriteString(fixture.Git("ls-files", "--stage", "-z"))
	snapshot.WriteString(fixture.Git("for-each-ref", "--format=%(refname) %(objectname)"))
	headRef, err := fixture.Runner.Run(context.Background(), "symbolic-ref", "-q", "HEAD")
	var commandErr *git.CommandError
	switch {
	case errors.As(err, &commandErr) && commandErr.ExitCode == detachedHeadExit:
		headRef = []byte(detachedHeadMarker)
	case err != nil:
		fixture.t.Fatalf("read HEAD reference: %v", err)
	}
	snapshot.Write(headRef)
	for _, marker := range operationMarkers {
		if _, err := os.Stat(filepath.Join(fixture.Dir, ".git", marker)); err == nil {
			fmt.Fprintf(&snapshot, "operation:%s\n", marker)
		}
	}
	walkErr := filepath.WalkDir(fixture.Dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&snapshot, "file:%s\n%s\n", path, content)
		return nil
	})
	if walkErr != nil {
		fixture.t.Fatalf("snapshot work tree: %v", walkErr)
	}
	return snapshot.String()
}

func (fixture *Fixture) PushFromOtherClone(branch, message string, files map[string]string) string {
	fixture.t.Helper()
	other := NewFixture(fixture.t)
	other.Git("remote", "add", remoteName, fixture.Remote)
	other.Git("fetch", "-q", remoteName, branch)
	other.Git("reset", "-q", "--hard", "FETCH_HEAD")
	oid := other.Commit(message, files)
	other.Git("push", "-q", remoteName, "HEAD:refs/heads/"+branch)
	return oid
}

func (fixture *Fixture) RemoteRefOID(remote, ref string) string {
	fixture.t.Helper()
	fields := strings.Fields(fixture.Git("ls-remote", remote, ref))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
