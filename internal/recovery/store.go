package recovery

import "errors"

var errNotImplemented = errors.New("recovery: not implemented")

type Phase string

const (
	PhaseStashed   Phase = "stashed"
	PhasePicking   Phase = "picking"
	PhaseRestoring Phase = "restoring"
)

type Baseline struct {
	Head         string
	Branch       string
	IndexTree    string
	WorktreeTree string
	Untracked    []string
}

type PendingRestore struct {
	StashOID string
	Commits  []string
	Branch   string
	Baseline Baseline
	Phase    Phase
}

type StateStore struct {
	path string
}

func NewStateStore(path string) *StateStore {
	return &StateStore{path: path}
}

func (store *StateStore) Load() (PendingRestore, bool, error) {
	return PendingRestore{}, false, errNotImplemented
}

func (store *StateStore) Save(pending PendingRestore) error {
	return errNotImplemented
}

func (store *StateStore) Clear() error {
	return errNotImplemented
}
