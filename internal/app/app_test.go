package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
	"github.com/achandrapaul/gitflow/internal/gittest"
	"github.com/achandrapaul/gitflow/internal/prefs"
	"github.com/achandrapaul/gitflow/internal/recovery"
)

const (
	testWidth  = 120
	testHeight = 36
)

var namedKeys = map[string]tea.KeyType{
	"esc":        tea.KeyEsc,
	"enter":      tea.KeyEnter,
	"tab":        tea.KeyTab,
	"shift+tab":  tea.KeyShiftTab,
	"ctrl+d":     tea.KeyCtrlD,
	"ctrl+u":     tea.KeyCtrlU,
	"up":         tea.KeyUp,
	"down":       tea.KeyDown,
	"left":       tea.KeyLeft,
	"right":      tea.KeyRight,
	"shift+up":   tea.KeyShiftUp,
	"shift+down": tea.KeyShiftDown,
	"space":      tea.KeySpace,
	"backspace":  tea.KeyBackspace,
}

func newTestApp(t *testing.T) (*App, *gittest.Fixture) {
	t.Helper()
	fixture := gittest.NewFixture(t)
	fixture.Commit("initial", map[string]string{"tracked.txt": "base\n", "staged.txt": "base\n"})
	fixture.Dirty("tracked.txt")
	fixture.Write("staged.txt", "staged\n")
	fixture.Git("add", "--", "staged.txt")
	fixture.Write("untracked.txt", "new\n")
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	drain(gitflowApp, gitflowApp.Init())
	return gitflowApp, fixture
}

func newTestAppFor(t *testing.T, fixture *gittest.Fixture, prefsPath string) *App {
	t.Helper()
	runner := fixture.Runner
	status := git.NewStatusReader(runner)
	operations := git.NewOperationDetector(runner)
	brancher := git.NewBrancher(runner, status, operations)
	differ := git.NewDiffer(runner)
	stasher := git.NewStasher(runner)
	cherryPicker := git.NewCherryPicker(runner, operations)
	gitflowApp, err := NewApp(context.Background(), Dependencies{
		Runner:         runner,
		Status:         status,
		Indexer:        git.NewIndexer(runner),
		Differ:         differ,
		History:        git.NewHistoryPager(runner),
		Branches:       brancher,
		Remote:         git.NewRemoteSync(runner, status, operations, brancher),
		CherryPicker:   cherryPicker,
		Operations:     operations,
		Controls:       git.NewOperationController(runner, status, operations),
		Conflicts:      git.NewConflictReader(runner),
		Stasher:        stasher,
		Restore:        recovery.NewCherryPickRestore(status, operations, differ, stasher, cherryPicker, recovery.NewStateStore(filepath.Join(fixture.Dir, ".git", "gitflow-state.json"))),
		SelectiveApply: recovery.NewSelectiveApply(status, operations, differ),
		Prefs:          prefs.NewStore(prefsPath),
	})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	gitflowApp.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	return gitflowApp
}

func drain(gitflowApp *App, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, batched := range msg {
			drain(gitflowApp, batched)
		}
	case nil:
	default:
		_, next := gitflowApp.Update(msg)
		drain(gitflowApp, next)
	}
}

func press(gitflowApp *App, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, key := range keys {
		message := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if keyType, named := namedKeys[key]; named {
			message = tea.KeyMsg{Type: keyType}
		}
		_, cmd = gitflowApp.Update(message)
	}
	return cmd
}

func pressAndDrain(gitflowApp *App, keys ...string) {
	for _, key := range keys {
		drain(gitflowApp, press(gitflowApp, key))
	}
}

func overlayID(gitflowApp *App) ScreenID {
	if gitflowApp.overlay == nil {
		return ""
	}
	return gitflowApp.overlay.ID()
}

func TestStartupRendersS01ReadOnly(t *testing.T) {
	fixture := gittest.NewFixture(t)
	head := fixture.Commit("initial", map[string]string{"tracked.txt": "base\n"})
	before := fixture.Snapshot()
	gitflowApp := newTestAppFor(t, fixture, filepath.Join(t.TempDir(), "prefs.json"))
	firstFrame := gitflowApp.View()
	for _, want := range []string{"GitFlow · S01 · repo", statusLoading, branchLoading, "▶ Files", "a Actions", "g Git menu", ": Command palette", "? Help", "q Back / Quit"} {
		if !strings.Contains(firstFrame, want) {
			t.Errorf("first frame missing %q:\n%s", want, firstFrame)
		}
	}
	drain(gitflowApp, gitflowApp.Init())
	view := gitflowApp.View()
	for _, want := range []string{"Branch main @ " + head[:shortOIDLength], "No upstream", "Base main ↑0 ↓0", "Local refs · last fetched unknown", "No changes · working tree clean", "Commit: No staged changes"} {
		if !strings.Contains(view, want) {
			t.Errorf("T01 loaded frame missing %q:\n%s", want, view)
		}
	}
	if after := fixture.Snapshot(); after != before {
		t.Fatalf("startup changed Git state")
	}
}

func TestKeyDispatch(t *testing.T) {
	testCases := []struct {
		name        string
		keys        []string
		wantOverlay ScreenID
		wantNotice  string
		wantTop     ScreenID
		check       func(t *testing.T, gitflowApp *App)
	}{
		{name: "g opens S15 without running Git", keys: []string{"g"}, wantOverlay: ScreenSync},
		{name: "g g is unbound", keys: []string{"g", "g"}, wantNotice: "g g is not assigned"},
		{name: "g p is configured pull", keys: []string{"g", "p"}, wantNotice: "Pull: " + reasonNoUpstream},
		{
			name: "g P is push", keys: []string{"g", "P"},
			check: func(t *testing.T, gitflowApp *App) {
				if !gitflowApp.busy {
					t.Fatalf("g P did not dispatch the push")
				}
			},
		},
		{name: "g f and g F differ", keys: []string{"g", "F"}, wantNotice: "Force push with lease: " + reasonNoUpstream},
		{name: "g Enter has no generic choice", keys: []string{"g", "enter"}, wantNotice: "g enter is not assigned"},
		{name: "g Esc closes without action", keys: []string{"g", "esc"}},
		{name: "a opens S05", keys: []string{"a"}, wantOverlay: ScreenActions},
		{name: "a e is unassigned", keys: []string{"a", "e"}, wantNotice: "a e is not assigned"},
		{name: "a menu Enter runs highlighted entry", keys: []string{"a", "enter"}, wantTop: ScreenBranches},
		{
			name: "a menu G goes to last row", keys: []string{"a", "G"}, wantOverlay: ScreenActions,
			check: func(t *testing.T, gitflowApp *App) {
				menu, isMenu := gitflowApp.overlay.(*actionsOverlay)
				if !isMenu || menu.cursor.index != len(menu.entries)-1 {
					t.Fatalf("cursor not on last row: %#v", gitflowApp.overlay)
				}
			},
		},
		{
			name: "a menu j k move focus only", keys: []string{"a", "j", "j", "k"}, wantOverlay: ScreenActions,
			check: func(t *testing.T, gitflowApp *App) {
				menu, isMenu := gitflowApp.overlay.(*actionsOverlay)
				if !isMenu || menu.cursor.index != 1 {
					t.Fatalf("cursor = %#v", gitflowApp.overlay)
				}
			},
		},
		{
			name: "a menu ctrl+d moves half a page", keys: []string{"a", "ctrl+d"}, wantOverlay: ScreenActions,
			check: func(t *testing.T, gitflowApp *App) {
				menu, isMenu := gitflowApp.overlay.(*actionsOverlay)
				if !isMenu || menu.cursor.index != min(testHeight/2, len(menu.entries)-1) {
					t.Fatalf("cursor = %#v", gitflowApp.overlay)
				}
			},
		},
		{
			name: "a menu ctrl+u returns half a page", keys: []string{"a", "ctrl+d", "ctrl+u"}, wantOverlay: ScreenActions,
			check: func(t *testing.T, gitflowApp *App) {
				menu, isMenu := gitflowApp.overlay.(*actionsOverlay)
				if !isMenu || menu.cursor.index != 0 {
					t.Fatalf("cursor = %#v", gitflowApp.overlay)
				}
			},
		},
		{name: "root Esc is a no-op", keys: []string{"esc"}},
		{name: "overlay-first Esc", keys: []string{"a", "esc", "esc"}},
		{name: "S01 stage key reports an ineligible focused file", keys: []string{"s"}, wantNotice: "Stage: staged.txt: nothing to stage"},
		{name: "S01 b is the g b alias", keys: []string{"b"}, wantTop: ScreenBranches},
		{name: "retired H does nothing", keys: []string{"H"}},
		{name: "second key is consumed once", keys: []string{"g", "h", "h"}, wantTop: ScreenHistory},
		{name: ": opens S12 palette", keys: []string{":"}, wantOverlay: ScreenPalette},
		{
			name: "palette letters are typed, not dispatched", keys: []string{":", "p", "u", "s", "h", "j", "S"}, wantOverlay: ScreenPalette,
			check: func(t *testing.T, gitflowApp *App) {
				palette, isPalette := gitflowApp.overlay.(*paletteOverlay)
				if !isPalette || palette.query.Value() != "pushjS" || len(palette.entries) != 0 || gitflowApp.changes.checked.Len() != 0 {
					t.Fatalf("palette = %#v", gitflowApp.overlay)
				}
			},
		},
		{
			name: "palette Enter runs the shared handler", keys: []string{":", "f", "e", "t", "c", "h", "enter"},
			check: func(t *testing.T, gitflowApp *App) {
				if !gitflowApp.busy {
					t.Fatalf("palette Enter did not dispatch the fetch")
				}
			},
		},
		{
			name: "? opens scoped help", keys: []string{"?"}, wantOverlay: ScreenPalette,
			check: func(t *testing.T, gitflowApp *App) {
				help, isHelp := gitflowApp.overlay.(*paletteOverlay)
				if !isHelp || help.helpScope != ScopeChanges {
					t.Fatalf("help = %#v", gitflowApp.overlay)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gitflowApp, fixture := newTestApp(t)
			before := fixture.Snapshot()
			press(gitflowApp, testCase.keys...)
			if got := overlayID(gitflowApp); got != testCase.wantOverlay {
				t.Errorf("overlay = %q, want %q", got, testCase.wantOverlay)
			}
			if gitflowApp.notice != testCase.wantNotice {
				t.Errorf("notice = %q, want %q", gitflowApp.notice, testCase.wantNotice)
			}
			if testCase.wantTop != "" {
				if gitflowApp.top().ID() != testCase.wantTop {
					t.Errorf("top = %s, want %s", gitflowApp.top().ID(), testCase.wantTop)
				}
			} else if len(gitflowApp.stack) != 1 || gitflowApp.top().ID() != ScreenChanges {
				t.Errorf("navigation left S01: %d screens", len(gitflowApp.stack))
			}
			if testCase.check != nil {
				testCase.check(t, gitflowApp)
			}
			if after := fixture.Snapshot(); after != before {
				t.Fatalf("keys %q changed Git state", testCase.keys)
			}
		})
	}
}

func TestComposerCapturesTypedKeys(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	before := fixture.Snapshot()

	press(gitflowApp, "c")
	if gitflowApp.changes.focus != focusComposer {
		t.Fatalf("c did not focus the composer")
	}
	press(gitflowApp, "s", "u", "a", "c", "g", "1", "2", "3", "4", "?", ":", "q", "H", "S", "v", "J")
	if got := gitflowApp.changes.draft.Value(); got != "suacg1234?:qHSvJ" {
		t.Fatalf("draft = %q", got)
	}
	if gitflowApp.overlay != nil || gitflowApp.notice != "" || gitflowApp.changes.checked.Len() != 0 || gitflowApp.changes.layout != prefs.LayoutFlat {
		t.Fatalf("typed keys dispatched: overlay %v notice %q", gitflowApp.overlay, gitflowApp.notice)
	}

	press(gitflowApp, "esc")
	if gitflowApp.top().ID() != ScreenComposer || gitflowApp.changes.draft.Focused() {
		t.Fatalf("Esc did not leave the editor for the S03 review")
	}
	press(gitflowApp, "g")
	if overlayID(gitflowApp) != ScreenSync {
		t.Fatalf("g after leaving the editor did not open S15")
	}
	if got := gitflowApp.changes.draft.Value(); got != "suacg1234?:qHSvJ" {
		t.Fatalf("draft lost on navigation: %q", got)
	}
	if after := fixture.Snapshot(); after != before {
		t.Fatal("typing changed Git state")
	}
}

func TestPanelFocusCycle(t *testing.T) {
	gitflowApp, _ := newTestApp(t)
	wantFocus := []changesFocus{focusDiff, focusComposer, focusFiles}
	for _, want := range wantFocus {
		press(gitflowApp, "tab")
		if gitflowApp.changes.focus != want {
			t.Fatalf("tab focus = %d, want %d", gitflowApp.changes.focus, want)
		}
	}
	press(gitflowApp, "shift+tab")
	if gitflowApp.changes.focus != focusComposer {
		t.Fatalf("shift+tab focus = %d", gitflowApp.changes.focus)
	}
}

func TestRootQuit(t *testing.T) {
	gitflowApp, fixture := newTestApp(t)
	before := fixture.Snapshot()
	cmd := press(gitflowApp, "q")
	if cmd == nil {
		t.Fatal("q at root returned no command")
	}
	if _, isQuit := cmd().(tea.QuitMsg); !isQuit {
		t.Fatal("q at root did not quit")
	}
	if after := fixture.Snapshot(); after != before {
		t.Fatal("quit changed Git state")
	}
}

func TestResponsiveRender(t *testing.T) {
	testCases := []struct {
		width, height int
		tooSmall      bool
	}{
		{width: minimumWidth, height: minimumHeight},
		{width: 60, height: 20},
		{width: 100, height: 24},
		{width: testWidth, height: testHeight},
		{width: 200, height: 60},
		{width: minimumWidth - 1, height: minimumHeight, tooSmall: true},
		{width: minimumWidth, height: minimumHeight - 1, tooSmall: true},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%dx%d", testCase.width, testCase.height), func(t *testing.T) {
			gitflowApp, fixture := newTestApp(t)
			before := fixture.Snapshot()
			for _, overlayKeys := range [][]string{nil, {"a"}, {"esc", "g"}, {"esc", ":"}, {"esc", "?"}} {
				press(gitflowApp, overlayKeys...)
				gitflowApp.Update(tea.WindowSizeMsg{Width: testCase.width, Height: testCase.height})
				view := gitflowApp.View()
				if lipgloss.Height(view) > testCase.height || lipgloss.Width(view) > testCase.width {
					t.Fatalf("view %dx%d exceeds terminal after %q", lipgloss.Width(view), lipgloss.Height(view), overlayKeys)
				}
				flattened := strings.Join(strings.Fields(view), " ")
				wantText := []string{"? Help", "q Back / Quit", "a Actions", "g Git menu", "1 staged · 0 mixed · 2 unstaged"}
				if testCase.tooSmall {
					wantText = []string{fmt.Sprintf("at least %d×%d", minimumWidth, minimumHeight), "? help", "q quit"}
				}
				for _, want := range wantText {
					if !strings.Contains(flattened, want) {
						t.Errorf("after %q missing %q:\n%s", overlayKeys, want, view)
					}
				}
			}
			press(gitflowApp, "esc", "g")
			if overlayID(gitflowApp) != ScreenSync {
				t.Fatal("g menu not reachable after resize")
			}
			if after := fixture.Snapshot(); after != before {
				t.Fatal("resize changed Git state")
			}
		})
	}
}

func TestListCursorWindowStart(t *testing.T) {
	testCases := []struct {
		name                  string
		index, count, visible int
		wantStart             int
	}{
		{name: "top stays at zero", index: 2, count: 30, visible: 10, wantStart: 0},
		{name: "middle centres focus", index: 15, count: 30, visible: 10, wantStart: 10},
		{name: "end clamps", index: 28, count: 30, visible: 10, wantStart: 20},
		{name: "short list", index: 3, count: 5, visible: 10, wantStart: 0},
		{name: "no cursor", index: noCursor, count: 30, visible: 10, wantStart: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := (listCursor{index: testCase.index}).windowStart(testCase.count, testCase.visible); got != testCase.wantStart {
				t.Fatalf("windowStart = %d, want %d", got, testCase.wantStart)
			}
		})
	}
}

func TestMenuKeepsCursorCentred(t *testing.T) {
	const entryCount = 30
	entries := make([]menuEntry, 0, entryCount)
	for index := range entryCount {
		entries = append(entries, menuEntry{action: &Action{Title: fmt.Sprintf("Entry %02d", index)}, available: true})
	}
	menu := renderMenu("Menu", entries, 15, testWidth, menuChromeHeight+10)
	for _, want := range []string{"Entry 10", "Entry 15", "Entry 19"} {
		if !strings.Contains(menu, want) {
			t.Errorf("menu missing %q:\n%s", want, menu)
		}
	}
	for _, unwanted := range []string{"Entry 09", "Entry 20"} {
		if strings.Contains(menu, unwanted) {
			t.Errorf("menu shows %q:\n%s", unwanted, menu)
		}
	}
}
