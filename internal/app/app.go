package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

const (
	defaultWidth       = 120
	defaultHeight      = 36
	headerHeight       = 2
	footerHeight       = 1
	errorLineLimit     = 6
	stackedLayoutBelow = 100
	prefixHintTitle    = "pending "
)

type Terminal interface {
	Run(command *exec.Cmd, stderr *bytes.Buffer, done func(error) tea.Msg) tea.Cmd
}

type TTYTerminal struct{}

func (TTYTerminal) Run(command *exec.Cmd, stderr *bytes.Buffer, done func(error) tea.Msg) tea.Cmd {
	command.Stderr = io.MultiWriter(os.Stderr, stderr)
	return tea.ExecProcess(command, func(err error) tea.Msg { return done(err) })
}

type screen interface {
	scope() Scope
	handleKey(a *App, key string) (tea.Cmd, bool)
	view(a *App, width, height int) string
	footer() string
	menuTitle(a *App) string
}

type textCapturer interface {
	capturingText() bool
	handleText(a *App, msg tea.KeyMsg) tea.Cmd
}

type leaveGuard interface {
	confirmLeave(a *App) bool
}

type statusListener interface {
	statusChanged(a *App) tea.Cmd
}

type menuProvider interface {
	menuActions(a *App) []*Action
}

type overlay interface {
	handleKey(a *App, msg tea.KeyMsg) tea.Cmd
	view(a *App, width, height int) string
}

type loadedMsg struct {
	apply func(a *App) tea.Cmd
}

type mutationMsg struct {
	label string
	err   error
	after func(a *App, err error) tea.Cmd
}

type App struct {
	ctx      context.Context
	repo     git.Repository
	runner   *git.Runner
	terminal Terminal
	registry *Registry
	width    int
	height   int

	status          git.Status
	statusLoaded    bool
	statusErr       error
	base            git.Comparison
	baseLoaded      bool
	baseErr         error
	baseOverride    string
	operation       git.Operation
	operationLoaded bool

	changes         *changesScreen
	history         *historyScreen
	commitSelection *orderedSelection[string]
	fileSelection   *orderedSelection[fileKey]
	drafts          map[composeMode]draft
	stack           []screen
	overlay         overlay
	pendingPrefix   string
	notice          string
	errorText       string
	busy            string
}

func New(ctx context.Context, repo git.Repository, runner *git.Runner, terminal Terminal) (*App, error) {
	registry, err := NewRegistry(actionDefinitions())
	if err != nil {
		return nil, fmt.Errorf("build action registry: %w", err)
	}
	app := &App{
		ctx:             ctx,
		repo:            repo,
		runner:          runner,
		terminal:        terminal,
		registry:        registry,
		width:           defaultWidth,
		height:          defaultHeight,
		commitSelection: &orderedSelection[string]{selected: map[string]bool{}},
		fileSelection:   &orderedSelection[fileKey]{selected: map[fileKey]bool{}},
		drafts:          map[composeMode]draft{},
	}
	search := newTextInput()
	search.Prompt = "/"
	app.changes = &changesScreen{filter: git.FilterAll, focusPanel: panelFiles, selected: map[string]bool{}, previews: map[string]filePreview{}, search: search}
	app.stack = []screen{app.changes}
	return app, nil
}

func (a *App) Init() tea.Cmd { return a.refresh() }

func (a *App) top() screen { return a.stack[len(a.stack)-1] }

func (a *App) push(next screen) { a.stack = append(a.stack, next) }

func (a *App) pop() {
	if len(a.stack) > 1 {
		a.stack = a.stack[:len(a.stack)-1]
	}
}

func (a *App) popTo(keep int) {
	if keep >= 1 && keep < len(a.stack) {
		a.stack = a.stack[:keep]
	}
}

func (a *App) load(work func(ctx context.Context) func(a *App) tea.Cmd) tea.Cmd {
	ctx := a.ctx
	return func() tea.Msg { return loadedMsg{apply: work(ctx)} }
}

func (a *App) refresh() tea.Cmd {
	runner, gitDir, override := a.runner, a.repo.GitDir, a.baseOverride
	return tea.Batch(
		a.load(func(ctx context.Context) func(a *App) tea.Cmd {
			status, err := git.LoadStatus(ctx, runner)
			return func(a *App) tea.Cmd {
				a.status, a.statusErr, a.statusLoaded = status, err, true
				var cmds []tea.Cmd
				for _, stacked := range a.stack {
					if listener, ok := stacked.(statusListener); ok {
						cmds = append(cmds, listener.statusChanged(a))
					}
				}
				return tea.Batch(cmds...)
			}
		}),
		a.load(func(ctx context.Context) func(a *App) tea.Cmd {
			base, err := git.ResolveBase(ctx, runner, override)
			return func(a *App) tea.Cmd {
				a.base, a.baseErr, a.baseLoaded = base, err, true
				return nil
			}
		}),
		a.load(func(ctx context.Context) func(a *App) tea.Cmd {
			operation, err := git.DetectOperation(ctx, runner, gitDir)
			return func(a *App) tea.Cmd {
				a.operation, a.operationLoaded = operation, err == nil
				if err != nil {
					a.errorText = err.Error()
				}
				return nil
			}
		}),
	)
}

func (a *App) mutate(label string, work func(ctx context.Context) error, after func(a *App, err error) tea.Cmd) tea.Cmd {
	if a.busy != "" {
		a.notice = label + " unavailable: " + busyReason
		return nil
	}
	a.busy = label
	ctx := a.ctx
	return func() tea.Msg { return mutationMsg{label: label, err: work(ctx), after: after} }
}

func (a *App) handoff(label string, command *exec.Cmd, after func(a *App, err error) tea.Cmd) tea.Cmd {
	if a.busy != "" {
		a.notice = label + " unavailable: " + busyReason
		return nil
	}
	a.busy = label
	var stderr bytes.Buffer
	return a.terminal.Run(command, &stderr, func(err error) tea.Msg {
		if err != nil {
			exitCode := -1
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			}
			err = &git.CommandError{Args: command.Args[1:], ExitCode: exitCode, Stderr: stderr.String(), Err: err}
		}
		return mutationMsg{label: label, err: err, after: after}
	})
}

func (a *App) gitHandoff(label string, invocation git.Invocation, after func(a *App, err error) tea.Cmd) tea.Cmd {
	return a.handoff(label, a.runner.Command(a.ctx, invocation.Env, invocation.Args...), after)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 && msg.Height > 0 {
			a.width, a.height = msg.Width, msg.Height
		}
		return a, nil
	case tea.FocusMsg:
		return a, a.refresh()
	case loadedMsg:
		return a, msg.apply(a)
	case mutationMsg:
		a.busy = ""
		var commandErr *git.CommandError
		switch {
		case errors.As(msg.err, &commandErr):
			stderr := strings.TrimSpace(commandErr.Stderr)
			if stderr == "" {
				stderr = commandErr.Err.Error()
			}
			a.errorText = fmt.Sprintf("%s failed (exit %d): git %s\n%s\nGit state was re-read; nothing is reported as done.", msg.label, commandErr.ExitCode, strings.Join(commandErr.Args, " "), stderr)
		case msg.err != nil:
			a.errorText = msg.label + " failed: " + msg.err.Error()
		default:
			a.errorText = ""
			a.notice = msg.label + ": done"
		}
		cmds := []tea.Cmd{a.refresh()}
		if msg.after != nil {
			cmds = append(cmds, msg.after(a, msg.err))
		}
		return a, tea.Batch(cmds...)
	case tea.KeyMsg:
		a.notice = ""
		if a.overlay != nil {
			return a, a.overlay.handleKey(a, msg)
		}
		if capturer, ok := a.top().(textCapturer); ok && capturer.capturingText() {
			return a, capturer.handleText(a, msg)
		}
		if msg.Paste {
			a.notice = "Paste ignored outside text input"
			return a, nil
		}
		a.errorText = ""
		key := msg.String()
		current := a.top()

		if a.pendingPrefix != "" {
			sequence := a.pendingPrefix + " " + key
			prefix := a.pendingPrefix
			a.pendingPrefix = ""
			if key == "esc" {
				return a, nil
			}
			if sequence == "g g" {
				cmd, _ := current.handleKey(a, sequence)
				return a, cmd
			}
			if action, ok := a.registry.Lookup(current.scope(), sequence); ok {
				return a, a.invoke(action)
			}
			a.notice = "No command for " + prefix + " " + key
			return a, nil
		}

		switch key {
		case "esc":
			a.back()
			return a, nil
		case "ctrl+c":
			a.openQuit()
			return a, nil
		}

		if action, ok := a.registry.Lookup(current.scope(), key); ok {
			return a, a.invoke(action)
		}
		if key == "g" || len(a.registry.Completions(current.scope(), key)) > 0 {
			a.pendingPrefix = key
			return a, nil
		}
		if cmd, handled := current.handleKey(a, key); handled {
			return a, cmd
		}

		switch key {
		case "q":
			if len(a.stack) > 1 {
				a.back()
			} else {
				a.openQuit()
			}
		case ":":
			input := newTextInput()
			input.Placeholder = "type to filter commands"
			input.Prompt = ": "
			input.Focus()
			a.overlay = &paletteOverlay{input: input, actions: a.registry.InScope(current.scope(), ScopeGlobal)}
		case "?":
			a.overlay = &helpOverlay{scope: current.scope()}
		case "a":
			menu := &menuOverlay{title: current.menuTitle(a), actions: a.registry.InScope(current.scope())}
			if provider, ok := current.(menuProvider); ok {
				menu.actions = provider.menuActions(a)
			}
			if scoped, ok := current.(interface{ targetScope(a *App) string }); ok {
				menu.target = scoped.targetScope(a)
			}
			a.overlay = menu
		case " ", "S":
			a.notice = "Nothing selectable here"
		}
		return a, nil
	}
	return a, nil
}

func (a *App) back() {
	if guard, ok := a.top().(leaveGuard); ok && !guard.confirmLeave(a) {
		return
	}
	a.pop()
}

func (a *App) openQuit() {
	lines := []string{"Leaving the interface does not change Git state."}
	if a.operation.Active() {
		lines = append(lines, "The "+a.operation.Kind.Name()+" stays paused; resume later with o.")
	}
	a.overlay = &confirmOverlay{
		title:   "Quit GitFlow?",
		lines:   lines,
		buttons: []confirmButton{{label: "Cancel"}, {label: "Quit", run: func(a *App) tea.Cmd { return tea.Quit }}},
		focus:   1,
	}
}

func (a *App) View() string {
	width, height := a.width, a.height
	current := a.top()
	headerFocused := current == a.changes && a.changes.focusPanel == panelHeader
	var messageLines []string
	if a.errorText != "" {
		errorLines := strings.Split(strings.TrimSpace(a.errorText), "\n")
		if len(errorLines) > errorLineLimit {
			errorLines = append(errorLines[:errorLineLimit], "…")
		}
		for _, line := range errorLines {
			messageLines = append(messageLines, styleError.Render(clip(line, width)))
		}
	}
	if a.notice != "" {
		messageLines = append(messageLines, styleDirty.Render(clip(a.notice, width)))
	}
	if a.pendingPrefix != "" {
		completions := a.registry.Completions(current.scope(), a.pendingPrefix)
		if a.pendingPrefix == "g" {
			completions = append(completions, "g g")
		}
		var hints []string
		for _, completion := range completions {
			if action, ok := a.registry.Lookup(current.scope(), completion); ok {
				hints = append(hints, completion+" "+action.Title)
			} else {
				hints = append(hints, completion+" First loaded row")
			}
		}
		messageLines = append(messageLines, styleAccent.Render(clip(prefixHintTitle+a.pendingPrefix+": "+strings.Join(hints, " · ")+" · Esc cancel", width)))
	}
	if a.busy != "" {
		messageLines = append(messageLines, styleMuted.Render("Running: "+a.busy+"…"))
	}
	branch := a.status.Branch
	var first, second strings.Builder
	switch {
	case !a.statusLoaded:
		first.WriteString(styleMuted.Render("Loading branch status…"))
	case a.statusErr != nil:
		first.WriteString(styleError.Render("Status unavailable: " + a.statusErr.Error()))
	case branch.Detached:
		first.WriteString(styleWarning.Render("DETACHED @ "+shortOID(branch.OID)) + styleWarning.Render("  ⚠ not on a branch"))
	case branch.Unborn:
		first.WriteString(styleBold.Render("● "+branch.Head) + styleMuted.Render(" (no commits yet)"))
	default:
		first.WriteString(styleBold.Render("● "+branch.Head) + " @ " + shortOID(branch.OID))
	}
	if a.statusLoaded && a.statusErr == nil && !branch.Detached {
		switch {
		case branch.Upstream == "":
			first.WriteString("  " + styleMuted.Render("No upstream"))
		case branch.HasAheadBehind:
			first.WriteString(fmt.Sprintf("  → %s ↑%d ↓%d", branch.Upstream, branch.Ahead, branch.Behind))
		default:
			first.WriteString("  → " + branch.Upstream + styleWarning.Render(" (upstream ref missing locally)"))
		}
		if branch.Upstream != "" {
			first.WriteString(styleMuted.Render("  [local refs · fetch time unknown]"))
		}
	}

	switch {
	case !a.baseLoaded:
		second.WriteString(styleMuted.Render("  base: loading…"))
	case a.baseErr != nil:
		second.WriteString(styleError.Render("  base error: " + a.baseErr.Error()))
	case !a.base.Available:
		second.WriteString(styleMuted.Render("  Base unavailable"))
	case !a.base.CountsKnown:
		second.WriteString("  vs " + a.base.ShortRef() + styleMuted.Render(" (counts unknown)"))
	default:
		second.WriteString(fmt.Sprintf("  vs %s ↑%d ↓%d", a.base.ShortRef(), a.base.Ahead, a.base.Behind))
	}
	if a.statusLoaded && a.statusErr == nil {
		if len(a.status.Files) == 0 {
			second.WriteString(" · working tree: clean")
		} else {
			second.WriteString(fmt.Sprintf(" · working tree: %d changed", len(a.status.Files)))
		}
	}
	switch {
	case !a.operationLoaded:
		second.WriteString(styleMuted.Render(" · operation: checking…"))
	case a.operation.Active():
		second.WriteString(" · " + styleConflict.Render("operation: "+strings.ToUpper(a.operation.Kind.Name())+" PAUSED (o)"))
	default:
		second.WriteString(" · operation: none")
	}
	if current == a.changes {
		second.WriteString(styleMuted.Render(" · g h History"))
	}
	marker := "  "
	if headerFocused {
		marker = styleFocus.Render("▶") + " "
	}
	header := clip(marker+first.String(), width) + "\n" + clip(second.String(), width)
	bodyHeight := max(height-headerHeight-footerHeight-len(messageLines), 3)
	body := current.view(a, width, bodyHeight)
	if a.overlay != nil {
		body = lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, a.overlay.view(a, width, bodyHeight))
	}
	footer := styleMuted.Render(clip("? Help  : Palette  "+current.footer(), width))
	sections := []string{header, body}
	sections = append(sections, messageLines...)
	sections = append(sections, footer)
	return strings.Join(sections, "\n")
}

func shortOID(oid string) string {
	if len(oid) < 7 {
		return oid
	}
	return oid[:7]
}
