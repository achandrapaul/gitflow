package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/app"
	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/prefs"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

const (
	gitBinary          = "git"
	stateFileName      = "gitflow/restore.json"
	prefsDirectoryName = "gitflow"
	prefsFileName      = "prefs.json"
	exitCodeFailure    = 1
)

var gitEnvironment = []string{"GIT_TERMINAL_PROMPT=0"}

func main() {
	ctx := context.Background()
	startDir, err := os.Getwd()
	if err != nil {
		exitWith(fmt.Errorf("resolve working directory: %w", err))
	}
	repository, err := git.Discover(ctx, git.NewRunner(gitBinary, startDir, nil))
	if err != nil {
		exitWith(fmt.Errorf("start gitflow: %w", err))
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		exitWith(fmt.Errorf("resolve user config directory: %w", err))
	}

	runner := git.NewRunner(gitBinary, repository.WorkTree, gitEnvironment)
	statusReader := git.NewStatusReader(runner)
	stasher := git.NewStasher(runner)
	operations := git.NewOperationDetector(runner)
	brancher := git.NewBrancher(runner, statusReader, operations)
	cherryPicker := git.NewCherryPicker(runner, operations)
	differ := git.NewDiffer(runner)

	gitflowApp, err := app.NewApp(ctx, app.Dependencies{
		Runner:         runner,
		Status:         statusReader,
		Indexer:        git.NewIndexer(runner),
		Differ:         differ,
		History:        git.NewHistoryPager(runner),
		Branches:       brancher,
		Remote:         git.NewRemoteSync(runner, statusReader, operations, brancher),
		CherryPicker:   cherryPicker,
		Rebaser:        git.NewRebaser(runner, statusReader, operations),
		Conflicts:      git.NewConflictReader(runner),
		Stasher:        stasher,
		Operations:     operations,
		Controls:       git.NewOperationController(runner, statusReader, operations),
		Restore:        recovery.NewCherryPickRestore(statusReader, operations, differ, stasher, cherryPicker, recovery.NewStateStore(filepath.Join(repository.GitDir, stateFileName))),
		SelectiveApply: recovery.NewSelectiveApply(statusReader, operations, differ),
		StashSwitch:    recovery.NewStashSwitch(statusReader, stasher, brancher),
		Prefs:          prefs.NewStore(filepath.Join(configDir, prefsDirectoryName, prefsFileName)),
		Deleter:        git.NewFileDeleter(runner),
	})
	if err != nil {
		exitWith(fmt.Errorf("start gitflow: %w", err))
	}

	if _, err := tea.NewProgram(gitflowApp, tea.WithAltScreen()).Run(); err != nil {
		exitWith(fmt.Errorf("run gitflow: %w", err))
	}
}

func exitWith(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(exitCodeFailure)
}
