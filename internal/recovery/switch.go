package recovery

import (
	"context"

	"github.com/achandrapaul/gitflow/internal/git"
)

type SwitchReview struct {
	Target   string
	Paths    []string
	Blockers []Blocker
}

type StashSwitch struct {
	statusReader *git.StatusReader
	stasher      *git.Stasher
	brancher     *git.Brancher
}

func NewStashSwitch(statusReader *git.StatusReader, stasher *git.Stasher, brancher *git.Brancher) *StashSwitch {
	return &StashSwitch{statusReader: statusReader, stasher: stasher, brancher: brancher}
}

func (stashSwitch *StashSwitch) Review(ctx context.Context, target string) (SwitchReview, error) {
	return SwitchReview{}, errNotImplemented
}

func (stashSwitch *StashSwitch) Execute(ctx context.Context, review SwitchReview) (git.StashRef, error) {
	return git.StashRef{}, errNotImplemented
}
