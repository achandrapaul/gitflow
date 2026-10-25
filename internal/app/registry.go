package app

import (
	"fmt"
	"sort"
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
	ScopeViewer         Scope = "viewer"
	ScopeCherryPick     Scope = "cherry-pick"
	ScopeRebase         Scope = "rebase"
	ScopeOperation      Scope = "operation"
	ScopeConflict       Scope = "conflict"
	ScopeBranch         Scope = "branch"
	ScopeCommitFiles    Scope = "commit-files"
	ScopeSelectiveApply Scope = "selective-apply"
	ScopeSync           Scope = "sync"
)

const busyReason = "another Git command is running"

var navigationKeys = map[string]bool{
	"enter": true, " ": true, "esc": true, "j": true, "k": true, "up": true, "down": true,
	"h": true, "l": true, "left": true, "right": true, "tab": true, "shift+tab": true,
	"a": true, ":": true, "?": true, "/": true, "g": true, "G": true, "ctrl+d": true, "ctrl+u": true, "q": true, "S": true,
}

var scopeNavigationKeys = map[Scope][]string{
	ScopeChanges:  {"1", "2", "3", "4"},
	ScopeConflict: {"n", "N"},
}

type Action struct {
	ID        string
	Title     string
	Scope     Scope
	Keys      []string
	Mutates   bool
	Available func(a *App) string
	Run       func(a *App) tea.Cmd
}

type Registry struct {
	actions []*Action
	byKey   map[Scope]map[string]*Action
}

func NewRegistry(actions []Action) (*Registry, error) {
	registry := &Registry{byKey: map[Scope]map[string]*Action{}}
	seenIDs := map[string]bool{}
	for index := range actions {
		action := &actions[index]
		if seenIDs[action.ID] {
			return nil, fmt.Errorf("register action %s: duplicate ID", action.ID)
		}
		seenIDs[action.ID] = true
		registry.actions = append(registry.actions, action)
		if registry.byKey[action.Scope] == nil {
			registry.byKey[action.Scope] = map[string]*Action{}
		}
		for _, key := range action.Keys {
			firstKey := strings.Fields(key)[0]
			reserved := navigationKeys[key] || (key == firstKey && navigationKeys[firstKey])
			for _, scopeKey := range scopeNavigationKeys[action.Scope] {
				reserved = reserved || scopeKey == key
			}
			if reserved {
				return nil, fmt.Errorf("register action %s: key %q is reserved for navigation", action.ID, key)
			}
			if existing, ok := registry.byKey[action.Scope][key]; ok {
				return nil, fmt.Errorf("register action %s: key %q already bound to %s in scope %s", action.ID, key, existing.ID, action.Scope)
			}
			if existing, ok := registry.byKey[action.Scope][firstKey]; ok && firstKey != key {
				return nil, fmt.Errorf("register action %s: prefix %q already bound to %s", action.ID, firstKey, existing.ID)
			}
			registry.byKey[action.Scope][key] = action
		}
	}
	for scope, bindings := range registry.byKey {
		for key, action := range bindings {
			if global, ok := registry.byKey[ScopeGlobal][key]; ok && scope != ScopeGlobal {
				return nil, fmt.Errorf("register action %s: key %q shadows global %s", action.ID, key, global.ID)
			}
			if fields := strings.Fields(key); len(fields) > 1 {
				if existing, ok := bindings[fields[0]]; ok {
					return nil, fmt.Errorf("register scope %s: prefix %q already bound to %s", scope, fields[0], existing.ID)
				}
			}
		}
	}
	return registry, nil
}

func (r *Registry) Lookup(scope Scope, key string) (*Action, bool) {
	if action, ok := r.byKey[scope][key]; ok {
		return action, true
	}
	action, ok := r.byKey[ScopeGlobal][key]
	return action, ok
}

func (r *Registry) Completions(scope Scope, prefix string) []string {
	var completions []string
	for _, candidateScope := range []Scope{scope, ScopeGlobal} {
		for key := range r.byKey[candidateScope] {
			if fields := strings.Fields(key); len(fields) > 1 && fields[0] == prefix {
				completions = append(completions, key)
			}
		}
	}
	sort.Strings(completions)
	return completions
}

func (r *Registry) InScope(scopes ...Scope) []*Action {
	var matching []*Action
	for _, action := range r.actions {
		for _, scope := range scopes {
			if action.Scope == scope {
				matching = append(matching, action)
			}
		}
	}
	return matching
}

func (r *Registry) ByID(id string) *Action {
	for _, action := range r.actions {
		if action.ID == id {
			return action
		}
	}
	return nil
}

func (a *App) unavailableReason(action *Action) string {
	if action.Mutates && a.busy != "" {
		return busyReason
	}
	if action.Available == nil {
		return ""
	}
	return action.Available(a)
}

func (a *App) invoke(action *Action) tea.Cmd {
	if reason := a.unavailableReason(action); reason != "" {
		a.notice = action.Title + " unavailable: " + reason
		return nil
	}
	return action.Run(a)
}

func keyLabel(action *Action) string {
	return strings.Join(action.Keys, " / ")
}
