package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

const (
	prefetchMargin   = 20
	graphLaneLimit   = 8
	historyListShare = 55
	historyChrome    = 4
)

type commitSelection struct {
	order    []string
	selected map[string]bool
}

func (s *commitSelection) toggle(oid string) {
	if s.selected[oid] {
		delete(s.selected, oid)
		for index, selectedOID := range s.order {
			if selectedOID == oid {
				s.order = append(s.order[:index:index], s.order[index+1:]...)
				break
			}
		}
		return
	}
	s.selected[oid] = true
	s.order = append(s.order, oid)
}

type historyScreen struct {
	ref        string
	cursor     *git.HistoryCursor
	commits    []git.Commit
	graphs     []string
	lanes      []string
	loading    bool
	err        error
	generation int
	focus      int
	offset     int
	focusOID   string
	filter     string
	searching  bool
	search     textinput.Model
	details    map[string]string
}

func newHistoryScreen(ref string) *historyScreen {
	search := newTextInput()
	search.Prompt = "/"
	return &historyScreen{ref: ref, search: search, details: map[string]string{}}
}

func (h *historyScreen) scope() Scope { return ScopeHistory }

func (h *historyScreen) opened(a *App) tea.Cmd {
	if h.cursor != nil || h.loading {
		return nil
	}
	return h.reset(a)
}

func (h *historyScreen) reset(a *App) tea.Cmd {
	if h.cursor != nil {
		h.cursor.Close()
	}
	h.generation++
	h.cursor, h.commits, h.graphs, h.lanes, h.err = nil, nil, nil, nil, nil
	h.loading = true
	generation, runner, ref := h.generation, a.runner, h.ref
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		cursor, err := git.OpenHistory(ctx, runner, ref)
		return func(a *App) tea.Cmd {
			if generation != h.generation {
				if cursor != nil {
					cursor.Close()
				}
				return nil
			}
			h.loading = false
			h.cursor, h.err = cursor, err
			if err != nil {
				return nil
			}
			return h.loadPage(a)
		}
	})
}

func (h *historyScreen) loadPage(a *App) tea.Cmd {
	if h.cursor == nil || h.loading || h.cursor.Done() {
		return nil
	}
	h.loading = true
	generation, cursor := h.generation, h.cursor
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		commits, err := cursor.Next(git.HistoryPageSize)
		return func(a *App) tea.Cmd {
			if generation != h.generation {
				return nil
			}
			h.loading = false
			h.err = err
			for _, commit := range commits {
				column := -1
				for laneIndex, lane := range h.lanes {
					if lane == commit.OID && column < 0 {
						column = laneIndex
					}
				}
				if column < 0 {
					column = len(h.lanes)
					h.lanes = append(h.lanes, commit.OID)
				}
				var graph strings.Builder
				for laneIndex, lane := range h.lanes {
					switch {
					case laneIndex == column:
						graph.WriteString("●")
					case lane == "":
						graph.WriteString(" ")
					default:
						graph.WriteString("│")
					}
				}
				for laneIndex, lane := range h.lanes {
					if laneIndex != column && lane == commit.OID {
						h.lanes[laneIndex] = ""
					}
				}
				h.lanes[column] = ""
				if len(commit.Parents) > 0 {
					h.lanes[column] = commit.Parents[0]
				}
				for _, parent := range commit.Parents[min(1, len(commit.Parents)):] {
					placed := false
					for laneIndex, lane := range h.lanes {
						if lane == parent {
							placed = true
						} else if lane == "" && !placed {
							h.lanes[laneIndex] = parent
							placed = true
						}
					}
					if !placed {
						h.lanes = append(h.lanes, parent)
					}
				}
				for len(h.lanes) > 0 && h.lanes[len(h.lanes)-1] == "" {
					h.lanes = h.lanes[:len(h.lanes)-1]
				}
				h.commits = append(h.commits, commit)
				h.graphs = append(h.graphs, graph.String())
			}
			visible := h.visible()
			for index, commitIndex := range visible {
				if h.commits[commitIndex].OID == h.focusOID {
					h.focus = index
				}
			}
			return h.requestDetail(a)
		}
	})
}

func (h *historyScreen) statusChanged(a *App) tea.Cmd {
	if h.cursor == nil {
		return nil
	}
	tip, runner, ref := h.cursor.TipOID, a.runner, h.ref
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		current, err := git.ResolveCommit(ctx, runner, ref)
		return func(a *App) tea.Cmd {
			if err == nil && current == tip {
				return nil
			}
			a.notice = "History refreshed: " + ref + " changed"
			return h.reset(a)
		}
	})
}

func (h *historyScreen) visible() []int {
	var indexes []int
	query := strings.ToLower(h.filter)
	for index, commit := range h.commits {
		if query == "" || strings.Contains(strings.ToLower(commit.Subject), query) || strings.Contains(strings.ToLower(commit.Author), query) || strings.HasPrefix(commit.OID, query) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (h *historyScreen) focused() (git.Commit, bool) {
	visible := h.visible()
	if h.focus < 0 || h.focus >= len(visible) {
		return git.Commit{}, false
	}
	return h.commits[visible[h.focus]], true
}

func (h *historyScreen) requestDetail(a *App) tea.Cmd {
	commit, ok := h.focused()
	if !ok {
		return nil
	}
	h.focusOID = commit.OID
	if _, cached := h.details[commit.OID]; cached {
		return nil
	}
	h.details[commit.OID] = "Loading…"
	runner := a.runner
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		text, err := git.ShowCommit(ctx, runner, commit.OID)
		return func(a *App) tea.Cmd {
			if err != nil {
				text = err.Error()
			}
			h.details[commit.OID] = text
			return nil
		}
	})
}

func (h *historyScreen) hiddenSelected(a *App) int {
	visibleOIDs := map[string]bool{}
	for _, index := range h.visible() {
		visibleOIDs[h.commits[index].OID] = true
	}
	hidden := 0
	for _, oid := range a.commitSelection.order {
		if !visibleOIDs[oid] {
			hidden++
		}
	}
	return hidden
}

func (h *historyScreen) targets(a *App) []string {
	if len(a.commitSelection.order) > 0 {
		return append([]string{}, a.commitSelection.order...)
	}
	if commit, ok := h.focused(); ok {
		return []string{commit.OID}
	}
	return nil
}

func (h *historyScreen) targetScope(a *App) string {
	if len(a.commitSelection.order) == 0 {
		if commit, ok := h.focused(); ok {
			return "Target: focused commit " + commit.Short()
		}
		return "No commit focused"
	}
	return fmt.Sprintf("Targets: %d selected commit(s) (%d hidden: filtered, other ref or not loaded)", len(a.commitSelection.order), h.hiddenSelected(a))
}

func (h *historyScreen) menuTitle(a *App) string {
	if count := len(a.commitSelection.order); count > 0 {
		return fmt.Sprintf("Actions for %d selected commits", count)
	}
	if commit, ok := h.focused(); ok {
		return "Actions for " + commit.Short() + " " + commit.Subject
	}
	return "Actions"
}

func (h *historyScreen) capturingText() bool { return h.searching }

func (h *historyScreen) handleText(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.searching, h.filter = false, ""
		h.search.SetValue("")
	case "enter":
		h.searching = false
	default:
		var cmd tea.Cmd
		h.search, cmd = h.search.Update(msg)
		h.filter = h.search.Value()
		h.focus = 0
		return tea.Batch(cmd, h.requestDetail(a))
	}
	h.search.Blur()
	h.focus = 0
	for index, commitIndex := range h.visible() {
		if h.commits[commitIndex].OID == h.focusOID {
			h.focus = index
		}
	}
	return h.requestDetail(a)
}

func (h *historyScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	visible := h.visible()
	if next, moved := moveCursor(key, h.focus, len(visible), a.height-historyChrome); moved {
		h.focus = next
		cmds := []tea.Cmd{h.requestDetail(a)}
		if h.focus >= len(visible)-prefetchMargin {
			cmds = append(cmds, h.loadPage(a))
		}
		return tea.Batch(cmds...), true
	}
	switch key {
	case " ":
		commit, ok := h.focused()
		if !ok {
			a.notice = "No commit to select"
			return nil, true
		}
		a.commitSelection.toggle(commit.OID)
		return nil, true
	case "enter":
		commit, ok := h.focused()
		if !ok {
			return nil, true
		}
		runner := a.runner
		return a.openViewer("commit "+commit.Short(), func(ctx context.Context) (string, error) {
			return git.ShowCommit(ctx, runner, commit.OID)
		}), true
	case "/":
		h.searching = true
		h.search.SetValue(h.filter)
		return h.search.Focus(), true
	}
	return nil, false
}

func (h *historyScreen) view(a *App, width, height int) string {
	listWidth := width * historyListShare / percent
	detailWidth := width - listWidth
	if width < stackedLayoutBelow {
		listWidth, detailWidth = width, width
	}
	body := []string{}
	if h.searching {
		body = append(body, h.search.View()+styleMuted.Render("  (loaded commits only)"))
	} else if h.filter != "" {
		body = append(body, styleMuted.Render(fmt.Sprintf("filter (loaded commits only, %d loaded): %s", len(h.commits), h.filter)))
	}
	if h.err != nil {
		body = append(body, styleError.Render(h.err.Error()))
	}
	visible := h.visible()
	rows := make([]string, len(visible))
	for rowIndex, commitIndex := range visible {
		commit := h.commits[commitIndex]
		mark := "[ ]"
		if a.commitSelection.selected[commit.OID] {
			mark = styleAccent.Render("[x]")
		}
		graph := h.graphs[commitIndex]
		if len([]rune(graph)) > graphLaneLimit {
			graph = string([]rune(graph)[:graphLaneLimit])
		}
		rows[rowIndex] = fmt.Sprintf("%s %-*s %s %s %s", mark, graphLaneLimit, graph, styleDirty.Render(commit.Short()), commit.Subject, styleMuted.Render(commit.Author+" "+commit.Date))
	}
	listHeight := height
	if width < stackedLayoutBelow {
		listHeight = height / 2
	}
	rendered, offset := renderRows(rows, h.focus, h.offset, listHeight-panelBorderSize-len(body)-3, true)
	h.offset = offset
	body = append(body, rendered...)
	progress := "end of history"
	switch {
	case h.loading:
		progress = "loading next page…"
	case h.cursor != nil && !h.cursor.Done():
		progress = "more on scroll"
	}
	if len(h.commits) == 0 && !h.loading && h.err == nil {
		body = append(body, styleMuted.Render("No commits"))
	}
	body = append(body, "", fmt.Sprintf("Selected: %d (%d hidden) · %d loaded · %s", len(a.commitSelection.order), h.hiddenSelected(a), len(h.commits), progress))
	listPanel := panel("HISTORY · "+h.ref, body, listWidth, listHeight, true)

	var detail []string
	if commit, ok := h.focused(); ok {
		detail = strings.Split(h.details[commit.OID], "\n")
	}
	detailHeight := height
	if width < stackedLayoutBelow {
		detailHeight = height - listHeight
	}
	detailPanel := panel("COMMIT DETAILS · Enter opens", detail, detailWidth, detailHeight, false)
	if width < stackedLayoutBelow {
		return lipgloss.JoinVertical(lipgloss.Left, listPanel, detailPanel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)
}

func (h *historyScreen) footer() string {
	return "j/k Move  Space Select  Enter Open  / Filter  c p Cherry-pick  r i Rebase onto focused  d c Compare  a Actions  Esc Back"
}

type viewerScreen struct {
	title   string
	lines   []string
	loading bool
	offset  int
}

func (a *App) openViewer(title string, work func(ctx context.Context) (string, error)) tea.Cmd {
	viewer := &viewerScreen{title: title, loading: true}
	a.push(viewer)
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		text, err := work(ctx)
		return func(a *App) tea.Cmd {
			viewer.loading = false
			if err != nil {
				viewer.lines = []string{styleError.Render(err.Error())}
				return nil
			}
			viewer.lines = strings.Split(strings.TrimRight(text, "\n"), "\n")
			return nil
		}
	})
}

func (v *viewerScreen) scope() Scope { return ScopeViewer }

func (v *viewerScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	next, moved := moveCursor(key, v.offset, len(v.lines), a.height)
	v.offset = next
	return nil, moved
}

func (v *viewerScreen) view(a *App, width, height int) string {
	if v.loading {
		return panel(v.title, []string{styleMuted.Render("Loading…")}, width, height, true)
	}
	var body []string
	for _, line := range v.lines[min(v.offset, max(len(v.lines)-1, 0)):] {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			line = styleStaged.Render(line)
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			line = styleError.Render(line)
		case strings.HasPrefix(line, "@@"):
			line = styleAccent.Render(line)
		}
		body = append(body, line)
	}
	return panel(fmt.Sprintf("%s · line %d/%d", v.title, v.offset+1, len(v.lines)), body, width, height, true)
}

func (v *viewerScreen) footer() string {
	return "j/k Scroll  Ctrl+D/U Half page  gg/G Top/bottom  Esc Back"
}

func (v *viewerScreen) menuTitle(a *App) string { return "Actions for " + v.title }
