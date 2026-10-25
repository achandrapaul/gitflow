package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/app"
	"github.com/achandrapaul/gitflow/internal/git"
)

func main() {
	ctx := context.Background()
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitflow: read working directory: %v\n", err)
		os.Exit(1)
	}
	repository, gitPath, err := git.Discover(ctx, workingDir)
	if err != nil {
		if errors.Is(err, git.ErrNotWorkTree) {
			fmt.Fprintf(os.Stderr, "gitflow: %s is not inside a Git working tree. Run gitflow from a repository.\n", workingDir)
		} else {
			fmt.Fprintf(os.Stderr, "gitflow: %v\n", err)
		}
		os.Exit(1)
	}
	model, err := app.New(ctx, repository, git.NewRunner(gitPath, repository.WorkTree, nil), app.TTYTerminal{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitflow: %v\n", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gitflow: %v\n", err)
		os.Exit(1)
	}
}
