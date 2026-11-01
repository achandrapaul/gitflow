package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Scope string

const (
	ScopeGlobal         Scope = "global"
	ScopeChanges        Scope = "changes"
	ScopeDiff           Scope = "diff"
	ScopeComposer       Scope = "composer"
	ScopeHistory        Scope = "history"
	ScopeCommitFiles    Scope = "commit-files"
	ScopeCherryPick     Scope = "cherry-pick"
	ScopeRebase         Scope = "rebase"
	ScopeSelectiveApply Scope = "selective-apply"
	ScopeOperation      Scope = "operation"
	ScopeConflict       Scope = "conflict"
	ScopeBranch         Scope = "branch"
	ScopeStash          Scope = "stash"
	ScopeStashRestore   Scope = "stash-restore"
	ScopeOverlay        Scope = "overlay"
)

type ActionID string

type ReviewRule int

const (
	ReviewNone ReviewRule = iota
	ReviewConfirm
	ReviewPlan
)

type RefreshPolicy int

const (
	RefreshNone RefreshPolicy = iota
	RefreshStatus
	RefreshAll
)

const reasonNotAvailableYet = "Not available yet"

var reservedKeySequences = map[string]bool{"a e": true, "gg": true, "g g": true, "H": true, "r i": true}

type Binding struct {
	Scope Scope
	Keys  []string
}

type Action struct {
	ID        ActionID
	Title     string
	Scope     Scope
	Keys      []string
	Aliases   []Binding
	Target    func(*App) bool
	Available func(*App) (bool, string)
	Review    ReviewRule
	Run       func(*App) tea.Cmd
	Refresh   RefreshPolicy
}

type Registry struct {
	actions []*Action
	byKey   map[Scope]map[string]*Action
	byID    map[ActionID]*Action
}

func (action *Action) bindings() []Binding {
	return append([]Binding{{Scope: action.Scope, Keys: action.Keys}}, action.Aliases...)
}

func (action *Action) Availability(app *App) (bool, string) {
	if action.Available != nil {
		if available, reason := action.Available(app); !available {
			return false, reason
		}
	}
	if action.Run == nil {
		return false, reasonNotAvailableYet
	}
	return true, ""
}

func NewRegistry(actions []Action) (*Registry, error) {
	registry := &Registry{byKey: map[Scope]map[string]*Action{}, byID: map[ActionID]*Action{}}
	for _, declared := range actions {
		action := &declared
		if _, exists := registry.byID[action.ID]; exists {
			return nil, fmt.Errorf("register action %q: duplicate id", action.ID)
		}
		for _, binding := range action.bindings() {
			if registry.byKey[binding.Scope] == nil {
				registry.byKey[binding.Scope] = map[string]*Action{}
			}
			for _, keys := range binding.Keys {
				if reservedKeySequences[keys] {
					return nil, fmt.Errorf("register action %q: key %q is reserved", action.ID, keys)
				}
				if bound, exists := registry.byKey[binding.Scope][keys]; exists {
					return nil, fmt.Errorf("register action %q: key %q in scope %s already bound to %q", action.ID, keys, binding.Scope, bound.ID)
				}
				registry.byKey[binding.Scope][keys] = action
			}
		}
		registry.byID[action.ID] = action
		registry.actions = append(registry.actions, action)
	}
	return registry, nil
}

func (registry *Registry) Lookup(scope Scope, keys string) (*Action, bool) {
	action, found := registry.byKey[scope][keys]
	return action, found
}

func (registry *Registry) InScope(scope Scope) []*Action {
	var scoped []*Action
	for _, action := range registry.actions {
		for _, binding := range action.bindings() {
			if binding.Scope == scope {
				scoped = append(scoped, action)
				break
			}
		}
	}
	return scoped
}

func (registry *Registry) ByID(id ActionID) (*Action, bool) {
	action, found := registry.byID[id]
	return action, found
}

func (registry *Registry) Search(query string) []*Action {
	needle := strings.ToLower(query)
	var matches []*Action
	for _, action := range registry.actions {
		if strings.Contains(strings.ToLower(action.Title), needle) || strings.Contains(string(action.ID), needle) {
			matches = append(matches, action)
		}
	}
	return matches
}
