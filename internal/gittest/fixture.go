package gittest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/gitflow/internal/git"
)

type Fixture struct {
	t        *testing.T
	Dir      string
	Env      []string
	Runner   *git.Runner
	gitPath  string
	isolated string
}

func New(t *testing.T) *Fixture {
	t.Helper()
	root := t.TempDir()
	workTree := filepath.Join(root, "repo")
	if err := os.Mkdir(workTree, 0o755); err != nil {
		t.Fatalf("create work tree: %v", err)
	}
	globalConfig := filepath.Join(root, "gitconfig")
	configText := "[user]\n\tname = Fixture User\n\temail = fixture@example.invalid\n[init]\n\tdefaultBranch = main\n[commit]\n\tgpgsign = false\n"
	if err := os.WriteFile(globalConfig, []byte(configText), 0o644); err != nil {
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
	fixture := &Fixture{t: t, Dir: workTree, Env: env, gitPath: gitPath, isolated: root}
	fixture.Runner = git.NewRunner(gitPath, workTree, env)
	fixture.Git("init", "-q", "-b", "main")
	return fixture
}

func (f *Fixture) Git(args ...string) string {
	f.t.Helper()
	output, err := f.Runner.Run(context.Background(), args...)
	if err != nil {
		f.t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(output)
}

func (f *Fixture) GitFails(args ...string) string {
	f.t.Helper()
	_, err := f.Runner.Run(context.Background(), args...)
	if err == nil {
		f.t.Fatalf("git %s: expected failure", strings.Join(args, " "))
	}
	return err.Error()
}

func (f *Fixture) Write(path, content string) {
	f.t.Helper()
	fullPath := filepath.Join(f.Dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		f.t.Fatalf("create parent of %s: %v", path, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", path, err)
	}
}

func (f *Fixture) ReadFile(path string) string {
	f.t.Helper()
	content, err := os.ReadFile(filepath.Join(f.Dir, path))
	if err != nil {
		f.t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func (f *Fixture) Commit(message string, files map[string]string) string {
	f.t.Helper()
	for path, content := range files {
		f.Write(path, content)
		f.Git("add", "--", path)
	}
	f.Git("commit", "-q", "--allow-empty", "-m", message)
	return strings.TrimSpace(f.Git("rev-parse", "HEAD"))
}

func (f *Fixture) BareRemote(name string) string {
	f.t.Helper()
	remoteDir := filepath.Join(f.isolated, name+".git")
	f.Git("init", "-q", "--bare", remoteDir)
	f.Git("remote", "add", name, remoteDir)
	return remoteDir
}

func (f *Fixture) Snapshot() string {
	f.t.Helper()
	var snapshot strings.Builder
	snapshot.WriteString(f.Git("status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all"))
	snapshot.WriteString(f.Git("ls-files", "--stage", "-z"))
	snapshot.WriteString(f.Git("for-each-ref", "--format=%(refname) %(objectname)"))
	for _, operationPath := range []string{"rebase-merge", "rebase-apply", "sequencer", "CHERRY_PICK_HEAD", "MERGE_HEAD"} {
		if _, err := os.Stat(filepath.Join(f.Dir, ".git", operationPath)); err == nil {
			snapshot.WriteString("operation:" + operationPath + "\n")
		}
	}
	walkErr := filepath.WalkDir(f.Dir, func(path string, entry os.DirEntry, err error) error {
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
		snapshot.WriteString("file:" + path + "\n" + string(content) + "\n")
		return nil
	})
	if walkErr != nil {
		f.t.Fatalf("snapshot work tree: %v", walkErr)
	}
	return snapshot.String()
}
