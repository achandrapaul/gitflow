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
	prefetchMargin       = 20
	graphLaneLimit       = 8
	historyListShare     = 55
	historyChrome        = 4
	commitFilesListShare = 40
	commitFilesHeader    = 2
)

type orderedSelection[K comparable] struct {
	order    []K
	selected map[K]bool
}

func (s *orderedSelection[K]) toggle(key K) {
	if s.selected[key] {
		delete(s.selected, key)
		for index, selectedKey := range s.order {
			if selectedKey == key {
				s.order = append(s.order[:index:index], s.order[index+1:]...)
				break
			}
		}
		return
	}
	s.selected[key] = true
	s.order = append(s.order, key)
}

type fileKey struct {
	commitOID string
	parentOID string
	file      git.CommitFile
}

func (a *App) selectedFileCommits() int {
	commits := map[string]bool{}
	for _, key := range a.fileSelection.order {
		commits[key.commitOID] = true
	}
	return len(commits)
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
		if !commit.IsMerge() {
			return a.openScreen(&commitFilesScreen{commit: commit, parentIndex: 0, diffs: map[string]git.FileDiff{}}), true
		}
		buttons := []confirmButton{{label: "Cancel"}}
		for parentIndex, parent := range commit.Parents {
			buttons = append(buttons, confirmButton{label: fmt.Sprintf("Parent %d %s", parentIndex+1, shortOID(parent)), run: func(a *App) tea.Cmd {
				return a.openScreen(&commitFilesScreen{commit: commit, parentIndex: parentIndex, diffs: map[string]git.FileDiff{}})
			}})
		}
		a.overlay = &confirmOverlay{
			title:   "Choose the parent to compare merge " + commit.Short() + " against",
			lines:   []string{"A merge commit has no single change set. Files and patches are computed against the parent you choose.", "GitFlow never guesses the parent."},
			buttons: buttons,
		}
		return nil, true
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
	if count := len(a.fileSelection.order); count > 0 {
		body = append(body, styleAccent.Render(fmt.Sprintf("Files: %d selected in %d commit(s) · c f Apply files", count, a.selectedFileCommits())))
	}
	listPanel := panel("HISTORY · "+h.ref, body, listWidth, listHeight, true)

	var detail []string
	if commit, ok := h.focused(); ok {
		detail = strings.Split(h.details[commit.OID], "\n")
	}
	detailHeight := height
	if width < stackedLayoutBelow {
		detailHeight = height - listHeight
	}
	detailPanel := panel("COMMIT DETAILS · Enter opens files", detail, detailWidth, detailHeight, false)
	if width < stackedLayoutBelow {
		return lipgloss.JoinVertical(lipgloss.Left, listPanel, detailPanel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)
}

func (h *historyScreen) footer() string {
	return "j/k Move  Space Select  Enter Files  / Filter  c p Cherry-pick  c f Apply files  r i Rebase onto focused  d c Compare  a Actions  Esc Back"
}

func (h *historyScreen) menuActions(a *App) []*Action {
	actions := a.registry.InScope(ScopeHistory)
	if len(a.fileSelection.order) > 0 {
		actions = append(actions, a.registry.ByID("files.apply"))
	}
	return actions
}

type commitFilesScreen struct {
	commit      git.Commit
	parentIndex int
	parentOID   string
	files       []git.CommitFile
	loaded      bool
	err         error
	cursor      int
	offset      int
	diffFocused bool
	expanded    bool
	diffOffset  int
	diffs       map[string]git.FileDiff
}

func (c *commitFilesScreen) scope() Scope { return ScopeCommitFiles }

func (c *commitFilesScreen) opened(a *App) tea.Cmd {
	runner, commit, parentIndex := a.runner, c.commit, c.parentIndex
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		var parent string
		var err error
		if len(commit.Parents) == 0 {
			parent, err = git.EmptyTreeOID(ctx, runner)
		} else {
			parent = commit.Parents[parentIndex]
		}
		var files []git.CommitFile
		if err == nil {
			files, err = git.ListCommitFiles(ctx, runner, parent, commit.OID)
		}
		return func(a *App) tea.Cmd {
			c.parentOID, c.files, c.err, c.loaded = parent, files, err, true
			return c.requestDiff(a)
		}
	})
}

func (c *commitFilesScreen) focused() (git.CommitFile, bool) {
	if c.cursor < 0 || c.cursor >= len(c.files) {
		return git.CommitFile{}, false
	}
	return c.files[c.cursor], true
}

func (c *commitFilesScreen) key(file git.CommitFile) fileKey {
	return fileKey{commitOID: c.commit.OID, parentOID: c.parentOID, file: file}
}

func (c *commitFilesScreen) selectedHere(a *App) int {
	count := 0
	for _, file := range c.files {
		if a.fileSelection.selected[c.key(file)] {
			count++
		}
	}
	return count
}

func (c *commitFilesScreen) requestDiff(a *App) tea.Cmd {
	file, ok := c.focused()
	if !ok {
		return nil
	}
	if _, cached := c.diffs[file.Path]; cached {
		return nil
	}
	c.diffs[file.Path] = git.FileDiff{}
	runner, parent, commit := a.runner, c.parentOID, c.commit.OID
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		patch, err := git.CommitFilePatch(ctx, runner, parent, commit, file)
		var diff git.FileDiff
		if err == nil {
			diff, err = git.ParseDiff(patch)
		}
		return func(a *App) tea.Cmd {
			if err != nil {
				diff = git.FileDiff{Header: []string{err.Error()}}
			}
			c.diffs[file.Path] = diff
			return nil
		}
	})
}

func (c *commitFilesScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "tab", "shift+tab":
		c.diffFocused = !c.diffFocused
		return nil, true
	case "enter":
		c.expanded = !c.expanded
		c.diffFocused = c.expanded
		return nil, true
	}
	if c.diffFocused {
		switch key {
		case "j", "down":
			c.diffOffset++
			return nil, true
		case "k", "up":
			c.diffOffset = max(c.diffOffset-1, 0)
			return nil, true
		}
		return nil, false
	}
	if next, moved := moveCursor(key, c.cursor, len(c.files), a.height-historyChrome); moved {
		c.cursor, c.diffOffset = next, 0
		return c.requestDiff(a), true
	}
	switch key {
	case " ":
		file, ok := c.focused()
		if !ok {
			a.notice = "No file to select"
			return nil, true
		}
		a.fileSelection.toggle(c.key(file))
		return nil, true
	case "S":
		if len(c.files) == 0 {
			a.notice = "No files to select"
			return nil, true
		}
		allSelected := c.selectedHere(a) == len(c.files)
		for _, file := range c.files {
			if a.fileSelection.selected[c.key(file)] == allSelected {
				a.fileSelection.toggle(c.key(file))
			}
		}
		return nil, true
	}
	return nil, false
}

func (c *commitFilesScreen) parentLabel() string {
	switch {
	case len(c.commit.Parents) == 0:
		return "empty tree (root commit)"
	case len(c.commit.Parents) == 1:
		return shortOID(c.parentOID)
	}
	return fmt.Sprintf("parent %d of %d %s", c.parentIndex+1, len(c.commit.Parents), shortOID(c.parentOID))
}

func (c *commitFilesScreen) view(a *App, width, height int) string {
	header := []string{
		fmt.Sprintf("commit %s · %s %s", c.commit.OID, c.commit.Short(), c.commit.Subject),
		fmt.Sprintf("Author: %s %s · parents: %d · compared against: %s · %d changed file(s)", c.commit.Author, c.commit.Date, len(c.commit.Parents), c.parentLabel(), len(c.files)),
	}
	headerPanel := panel("COMMIT INSPECTOR", header, width, len(header)+panelBorderSize+1, false)
	bodyHeight := max(height-lipgloss.Height(headerPanel), commitFilesHeader+panelBorderSize)

	body := []string{fmt.Sprintf("%d selected in this commit · %d in %d commit(s) overall", c.selectedHere(a), len(a.fileSelection.order), a.selectedFileCommits())}
	switch {
	case !c.loaded:
		body = append(body, styleMuted.Render("Loading…"))
	case c.err != nil:
		body = append(body, styleError.Render(c.err.Error()))
	case len(c.files) == 0:
		body = append(body, styleMuted.Render("No file changes against this parent"))
	}
	rows := make([]string, len(c.files))
	for index, file := range c.files {
		mark := "[ ]"
		if a.fileSelection.selected[c.key(file)] {
			mark = styleAccent.Render("[x]")
		}
		path := displayPath(file.Path)
		if file.OrigPath != "" {
			path = displayPath(file.OrigPath) + " → " + path
		}
		row := fmt.Sprintf("%s %c %s", mark, file.Status, path)
		if reason := file.UnsupportedReason(); reason != "" {
			row += " " + styleWarning.Render("("+reason+")")
		}
		rows[index] = row
	}
	listWidth := width * commitFilesListShare / percent
	rendered, offset := renderRows(rows, c.cursor, c.offset, bodyHeight-panelBorderSize-len(body)-1, !c.diffFocused)
	c.offset = offset
	listPanel := panel("FILES IN COMMIT", append(body, rendered...), listWidth, bodyHeight, !c.diffFocused)

	diffTitle := "COMMIT PATCH PREVIEW"
	var diffBody []string
	if file, ok := c.focused(); ok {
		diffTitle += " · " + displayPath(file.Path)
		diffBody = append([]string{styleMuted.Render("diff " + c.parentLabel() + " → " + c.commit.Short())}, patchLines(c.diffs[file.Path])...)
		c.diffOffset = min(c.diffOffset, max(len(diffBody)-1, 0))
		diffBody = diffBody[c.diffOffset:]
	}
	if c.expanded {
		return lipgloss.JoinVertical(lipgloss.Left, headerPanel, panel(diffTitle, diffBody, width, bodyHeight, true))
	}
	diffPanel := panel(diffTitle, diffBody, width-listWidth, bodyHeight, c.diffFocused)
	return lipgloss.JoinVertical(lipgloss.Left, headerPanel, lipgloss.JoinHorizontal(lipgloss.Top, listPanel, diffPanel))
}

func (c *commitFilesScreen) footer() string {
	return "j/k Move  Space Select file  S Select all  Enter Expand diff  Tab Panel  c f Apply selected files  a Actions  Esc Back"
}

func (c *commitFilesScreen) menuTitle(a *App) string {
	return fmt.Sprintf("Actions for files in %s (%d selected overall)", c.commit.Short(), len(a.fileSelection.order))
}

func (c *commitFilesScreen) targetScope(a *App) string {
	if len(a.fileSelection.order) > 0 {
		return fmt.Sprintf("Targets: %d selected file(s) in %d commit(s)", len(a.fileSelection.order), a.selectedFileCommits())
	}
	if file, ok := c.focused(); ok {
		return "Target: focused file only " + displayPath(file.Path)
	}
	return "No file focused"
}

func (c *commitFilesScreen) menuActions(a *App) []*Action {
	return append(a.registry.InScope(ScopeCommitFiles), a.registry.ByID("files.apply"))
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
