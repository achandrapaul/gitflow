package app

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

const (
	historyPageSize          = 100
	historyPrefetchMargin    = 20
	compareSelectionSize     = 2
	historyLabel             = "History"
	historyTitle             = "%s · %s @ %s"
	historyUnbornTitle       = "%s · %s" + unbornLabel
	historyDetachedTitle     = "%s · " + detachedLabel + "%s"
	historyCounts            = "%d loaded · %d selected · %d hidden"
	historyMoreOnScroll      = " · more load on scroll"
	historyAllLoaded         = " · all loaded"
	historyFilterLabel       = "Filter loaded commits only (subject, author, OID)"
	historyFilterPrompt      = historyFilterLabel + ": "
	historyLoading           = "Loading history…"
	historyNoMatch           = "No loaded commits match"
	historyStaleNotice       = "Source moved since this history was loaded; pagination stopped · g h reloads"
	commitsPanelTitle        = "Commits"
	commitPanelTitle         = "Commit"
	commitLoading            = "Loading commit…"
	commitParentsLabel       = "Parents "
	commitRootLabel          = "Root commit"
	commitAuthorFormat       = "Author %s · %s"
	commitDateLayout         = "2006-01-02 15:04"
	commitFilesTitle         = "Commit %s · %s"
	commitFilesParent        = "Parent %s"
	commitFilesRootParent    = "Parent none · root commit"
	commitFilesSummary       = "%d files · read-only · Enter full-width diff · Esc back"
	historyFilesLine         = "Files %d selected · Space selects files · g c carries them into the Cherry-pick Explorer"
	unsupportedFileNote      = " · unsupported: %s"
	commitFilesLoading       = "Loading files…"
	commitFilesEmpty         = "No file changes"
	filesPanelTitle          = "Files"
	patchPanelTitle          = "Diff"
	patchLoading             = "Loading diff…"
	reasonHistoryClosed      = "Open History (g h) first"
	reasonSelectTwoCommits   = "Select exactly two commits"
	reasonCompareContractGap = "Compare output awaits a contract decision (OPEN_QUESTIONS Q4)"
	reasonMergeContractGap   = "Inspecting a merge commit awaits a contract decision on the parent (OPEN_QUESTIONS Q25)"
)

type orderedSelection[K comparable] struct {
	order []K
	index map[K]int
}

func (selection orderedSelection[K]) With(keys ...K) orderedSelection[K] {
	next := orderedSelection[K]{order: slices.Clone(selection.order), index: maps.Clone(selection.index)}
	if next.index == nil {
		next.index = map[K]int{}
	}
	for _, key := range keys {
		if _, found := next.index[key]; found {
			continue
		}
		next.index[key] = len(next.order)
		next.order = append(next.order, key)
	}
	return next
}

func (selection orderedSelection[K]) Without(key K) orderedSelection[K] {
	if !selection.Contains(key) {
		return selection
	}
	return orderedSelection[K]{}.With(slices.DeleteFunc(slices.Clone(selection.order), func(member K) bool { return member == key })...)
}

func (selection orderedSelection[K]) Move(key K, offset int) orderedSelection[K] {
	position, found := selection.index[key]
	target := position + offset
	if !found || target < 0 || target >= len(selection.order) {
		return selection
	}
	order := slices.Clone(selection.order)
	order[position], order[target] = order[target], order[position]
	return orderedSelection[K]{}.With(order...)
}

func (selection orderedSelection[K]) Contains(key K) bool {
	_, found := selection.index[key]
	return found
}

func (selection orderedSelection[K]) Ordered() []K { return slices.Clone(selection.order) }

func (selection orderedSelection[K]) Len() int { return len(selection.order) }

type historyLoadedMsg struct {
	history    *historyScreen
	generation int
	reset      bool
	page       git.HistoryPage
	stale      bool
	err        error
}

type commitDetailMsg struct {
	history *historyScreen
	oid     string
	detail  git.CommitDetail
	err     error
}

type commitFilesLoadedMsg struct {
	oid   string
	files []git.CommitFile
	err   error
}

type filePatchLoadedMsg struct {
	path  string
	patch string
	err   error
}

type selectedCommit struct {
	summary git.CommitSummary
	source  string
}

type historyScreen struct {
	label         string
	sourceRef     string
	explorer      *cherryPickScreen
	cursor        git.HistoryCursor
	commits       []git.CommitSummary
	generation    int
	loading       bool
	done          bool
	stale         bool
	err           error
	focusOID      string
	search        textinput.Model
	searching     bool
	selection     orderedSelection[string]
	selected      map[string]selectedCommit
	files         orderedSelection[git.FileSelection]
	fileSources   map[string]selectedCommit
	detailOID     string
	detail        git.CommitDetail
	detailErr     error
	detailFocused bool
	detailScroll  int
}

func (app *App) historyOnTop() (*historyScreen, bool) {
	if len(app.stack) == 0 {
		return nil, false
	}
	history, open := app.top().(*historyScreen)
	return history, open
}

func (history *historyScreen) orderedFiles() []git.FileSelection {
	selections := history.files.Ordered()
	var ordered []git.FileSelection
	grouped := map[string]bool{}
	for _, first := range selections {
		if grouped[first.OID] {
			continue
		}
		grouped[first.OID] = true
		for _, selection := range selections {
			if selection.OID == first.OID {
				ordered = append(ordered, selection)
			}
		}
	}
	return ordered
}

func (history *historyScreen) reload(app *App) tea.Cmd {
	history.generation++
	history.loading, history.stale, history.err = true, false, nil
	generation, sourceRef := history.generation, history.sourceRef
	return func() tea.Msg {
		cursor, err := app.deps.History.Open(app.ctx, sourceRef)
		if err != nil {
			return historyLoadedMsg{history: history, generation: generation, reset: true, err: err}
		}
		page, err := app.deps.History.Page(app.ctx, cursor, historyPageSize)
		return historyLoadedMsg{history: history, generation: generation, reset: true, page: page, err: err}
	}
}

func (history *historyScreen) loadNext(app *App) tea.Cmd {
	history.loading = true
	cursor, generation, done := history.cursor, history.generation, history.done
	return func() tea.Msg {
		stale, err := app.deps.History.Stale(app.ctx, cursor)
		if err != nil || stale || done {
			return historyLoadedMsg{history: history, generation: generation, stale: stale, err: err, page: git.HistoryPage{Next: cursor, Done: done}}
		}
		page, err := app.deps.History.Page(app.ctx, cursor, historyPageSize)
		return historyLoadedMsg{history: history, generation: generation, page: page, err: err}
	}
}

func (history *historyScreen) visible() []git.CommitSummary {
	query := strings.ToLower(history.search.Value())
	if query == "" {
		return history.commits
	}
	return slices.DeleteFunc(slices.Clone(history.commits), func(commit git.CommitSummary) bool {
		return !strings.Contains(strings.ToLower(commit.Subject), query) && !strings.Contains(strings.ToLower(commit.Author), query) && !strings.HasPrefix(commit.OID, query)
	})
}

func (history *historyScreen) focusedIndex(rows []git.CommitSummary) int {
	return slices.IndexFunc(rows, func(commit git.CommitSummary) bool { return commit.OID == history.focusOID })
}

func (history *historyScreen) followUp(app *App) tea.Cmd {
	rows := history.visible()
	focused := history.focusedIndex(rows)
	if focused < 0 && len(rows) > 0 {
		focused, history.focusOID = 0, rows[0].OID
	}
	var cmds []tea.Cmd
	if !history.loading && !history.done && !history.stale && history.err == nil && history.search.Value() == "" && focused >= len(rows)-historyPrefetchMargin {
		cmds = append(cmds, history.loadNext(app))
	}
	if focused >= 0 && rows[focused].OID != history.detailOID {
		oid := rows[focused].OID
		history.detailOID, history.detail, history.detailErr, history.detailScroll = oid, git.CommitDetail{}, nil, 0
		cmds = append(cmds, func() tea.Msg {
			detail, err := app.deps.History.Detail(app.ctx, oid)
			return commitDetailMsg{history: history, oid: oid, detail: detail, err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (history *historyScreen) detailLines() []string {
	switch {
	case history.detailErr != nil:
		return []string{appStyles.conflict.Render(history.detailErr.Error())}
	case history.detailOID == "":
		return nil
	case history.detail.Summary.OID != history.detailOID:
		return []string{commitLoading}
	}
	summary := history.detail.Summary
	parents := commitRootLabel
	if len(summary.Parents) > 0 {
		parents = commitParentsLabel + strings.Join(summary.Parents, " ")
	}
	lines := []string{appStyles.header.Render(summary.OID), parents, fmt.Sprintf(commitAuthorFormat, summary.Author, summary.AuthorDate.Format(commitDateLayout)), "", appStyles.header.Render(summary.Subject)}
	if history.detail.Body != "" {
		lines = append(lines, strings.Split(history.detail.Body, "\n")...)
	}
	return append(append(lines, ""), strings.Split(strings.TrimSuffix(history.detail.Patch, "\n"), "\n")...)
}

func (history *historyScreen) ID() ScreenID { return ScreenHistory }

func (history *historyScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		if msg.generation != history.generation {
			return nil, true
		}
		history.loading = false
		switch {
		case msg.err != nil:
			history.err = msg.err
			return nil, true
		case msg.stale:
			history.stale, app.notice = true, historyStaleNotice
			return nil, true
		case msg.reset:
			history.commits, history.detailOID = nil, ""
		}
		history.commits = append(history.commits, msg.page.Commits...)
		history.cursor, history.done = msg.page.Next, msg.page.Done
		return history.followUp(app), true
	case commitDetailMsg:
		if msg.oid == history.detailOID {
			history.detail, history.detailErr = msg.detail, msg.err
		}
		return nil, true
	case tea.KeyMsg:
		keys := msg.String()
		rows := history.visible()
		cursor := listCursor{index: max(0, history.focusedIndex(rows))}
		detailCursor := listCursor{index: history.detailScroll}
		switch {
		case keys == keyTab:
			history.detailFocused = !history.detailFocused
		case keys == keyEnter:
			history.searching = false
			history.search.Blur()
			index := history.focusedIndex(rows)
			if index < 0 {
				return nil, true
			}
			commit := rows[index]
			if len(commit.Parents) > 1 {
				app.notice = reasonMergeContractGap
				return nil, true
			}
			parent := ""
			if len(commit.Parents) == 1 {
				parent = commit.Parents[0]
			}
			app.push(&commitFilesScreen{commit: commit, parent: parent, history: history})
			return func() tea.Msg {
				files, err := app.deps.Differ.CommitFiles(app.ctx, commit.OID, parent)
				return commitFilesLoadedMsg{oid: commit.OID, files: files, err: err}
			}, true
		case keys == keyEscape && (history.searching || history.search.Value() != ""):
			history.searching = false
			history.search.Blur()
			history.search.SetValue("")
			return history.followUp(app), true
		case keys == keyEscape:
			return nil, false
		case history.searching && msg.Type != tea.KeyUp && msg.Type != tea.KeyDown:
			var cmd tea.Cmd
			history.search, cmd = history.search.Update(msg)
			return tea.Batch(cmd, history.followUp(app)), true
		case history.detailFocused && detailCursor.navigate(keys, len(history.detailLines()), app.height/2):
			history.detailScroll = detailCursor.index
		case len(rows) > 0 && cursor.navigate(keys, len(rows), app.height/2):
			history.focusOID = rows[cursor.index].OID
			return history.followUp(app), true
		case keys == keySpace && history.focusOID != "":
			if history.selection.Contains(history.focusOID) {
				history.selection = history.selection.Without(history.focusOID)
				delete(history.selected, history.focusOID)
				return nil, true
			}
			history.selection = history.selection.With(history.focusOID)
			if history.selected == nil {
				history.selected = map[string]selectedCommit{}
			}
			if focused := history.focusedIndex(rows); focused >= 0 {
				history.selected[history.focusOID] = selectedCommit{summary: rows[focused], source: history.cursor.Label}
			}
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

func (history *historyScreen) View(width, height int) string {
	title := fmt.Sprintf(historyTitle, history.label, history.cursor.Label, shortOID(history.cursor.TipOID))
	switch {
	case history.cursor.Detached:
		title = fmt.Sprintf(historyDetachedTitle, history.label, shortOID(history.cursor.TipOID))
	case history.cursor.TipOID == "":
		title = fmt.Sprintf(historyUnbornTitle, history.label, history.cursor.Label)
	}
	rows := history.visible()
	hidden := 0
	for _, oid := range history.selection.Ordered() {
		if !slices.ContainsFunc(rows, func(commit git.CommitSummary) bool { return commit.OID == oid }) {
			hidden++
		}
	}
	counts := fmt.Sprintf(historyCounts, len(history.commits), history.selection.Len(), hidden) + historyMoreOnScroll
	if history.done {
		counts = fmt.Sprintf(historyCounts, len(history.commits), history.selection.Len(), hidden) + historyAllLoaded
	}
	lines := []string{appStyles.header.Render(title), counts, appStyles.disabled.Render("/ " + historyFilterLabel)}
	if history.searching || history.search.Value() != "" {
		lines[2] = history.search.View()
	}
	if history.stale {
		lines = append(lines, appStyles.warning.Render(historyStaleNotice))
	}
	if history.err != nil {
		lines = append(lines, appStyles.conflict.Render(history.err.Error()))
	}
	panelHeight := max(1, height-len(lines))
	listHeight := max(1, panelHeight-panelBorderSize-panelTitleHeight)
	var listLines []string
	switch {
	case len(history.commits) == 0 && history.loading:
		listLines = append(listLines, historyLoading)
	case len(rows) == 0 && history.search.Value() != "":
		listLines = append(listLines, appStyles.disabled.Render(historyNoMatch))
	}
	focusedIndex := history.focusedIndex(rows)
	firstRow := listCursor{index: focusedIndex}.windowStart(len(rows), listHeight)
	for row := firstRow; row < min(len(rows), firstRow+listHeight); row++ {
		commit := rows[row]
		mark := uncheckedMark
		if position, queued := history.selection.index[commit.OID]; queued && history.explorer != nil {
			mark = fmt.Sprintf(queuePositionMark, position+1)
		} else if queued {
			mark = checkedMark
		}
		line := fmt.Sprintf("%s%s %s %s", mark, commit.Graph, shortOID(commit.OID), commit.Subject)
		if row == focusedIndex {
			line = appStyles.selected.Render(line)
		}
		listLines = append(listLines, line)
	}
	detail := history.detailLines()
	listWidth := width / 2
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		renderPanel(commitsPanelTitle, !history.detailFocused, strings.Join(listLines, "\n"), listWidth, panelHeight),
		renderPanel(commitPanelTitle, history.detailFocused, strings.Join(detail[min(history.detailScroll, len(detail)):], "\n"), width-listWidth, panelHeight))
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(append(lines, panels), "\n"))
}

func (history *historyScreen) Scope() Scope { return ScopeHistory }

type commitFilesScreen struct {
	commit      git.CommitSummary
	parent      string
	files       []git.CommitFile
	loaded      bool
	err         error
	row         int
	patchPath   string
	patch       string
	patchErr    error
	patchLoaded bool
	patchScroll int
	patchFocus  bool
	fullWidth   bool
	history     *historyScreen
}

func (commitFiles *commitFilesScreen) selectionOf(file git.CommitFile) git.FileSelection {
	return git.FileSelection{OID: commitFiles.commit.OID, Parent: commitFiles.parent, Path: file.Path, OriginalPath: file.OriginalPath, Status: file.Status}
}

func (commitFiles *commitFilesScreen) loadPatch(app *App) tea.Cmd {
	if commitFiles.row >= len(commitFiles.files) || commitFiles.files[commitFiles.row].Path == commitFiles.patchPath {
		return nil
	}
	selection := commitFiles.selectionOf(commitFiles.files[commitFiles.row])
	commitFiles.patchPath, commitFiles.patch, commitFiles.patchErr, commitFiles.patchLoaded, commitFiles.patchScroll = selection.Path, "", nil, false, 0
	return func() tea.Msg {
		patch, err := app.deps.Differ.FilePatch(app.ctx, selection)
		return filePatchLoadedMsg{path: selection.Path, patch: patch, err: err}
	}
}

func (commitFiles *commitFilesScreen) patchLines() []string {
	switch {
	case commitFiles.patchErr != nil:
		return []string{appStyles.conflict.Render(commitFiles.patchErr.Error())}
	case commitFiles.patchPath == "":
		return nil
	case !commitFiles.patchLoaded:
		return []string{patchLoading}
	}
	return strings.Split(strings.TrimSuffix(commitFiles.patch, "\n"), "\n")
}

func (commitFiles *commitFilesScreen) ID() ScreenID { return ScreenCommitFiles }

func (commitFiles *commitFilesScreen) Update(app *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case commitFilesLoadedMsg:
		if msg.oid != commitFiles.commit.OID {
			return nil, true
		}
		commitFiles.files, commitFiles.err, commitFiles.loaded = msg.files, msg.err, true
		return commitFiles.loadPatch(app), true
	case filePatchLoadedMsg:
		if msg.path == commitFiles.patchPath {
			commitFiles.patch, commitFiles.patchErr, commitFiles.patchLoaded = msg.patch, msg.err, true
		}
		return nil, true
	case tea.KeyMsg:
		keys := msg.String()
		cursor := listCursor{index: commitFiles.row}
		patchCursor := listCursor{index: commitFiles.patchScroll}
		switch {
		case keys == keyEnter:
			commitFiles.fullWidth = !commitFiles.fullWidth
		case keys == keyTab:
			commitFiles.patchFocus = !commitFiles.patchFocus
		case keys == keySpace && commitFiles.row < len(commitFiles.files):
			history, selection := commitFiles.history, commitFiles.selectionOf(commitFiles.files[commitFiles.row])
			if history.files.Contains(selection) {
				history.files = history.files.Without(selection)
				return nil, true
			}
			history.files = history.files.With(selection)
			if history.fileSources == nil {
				history.fileSources = map[string]selectedCommit{}
			}
			history.fileSources[selection.OID] = selectedCommit{summary: commitFiles.commit, source: history.cursor.Label}
		case keys == keyEscape && commitFiles.fullWidth:
			commitFiles.fullWidth = false
		case keys == keyEscape:
			return nil, false
		case (commitFiles.patchFocus || commitFiles.fullWidth) && patchCursor.navigate(keys, len(commitFiles.patchLines()), app.height/2):
			commitFiles.patchScroll = patchCursor.index
		case len(commitFiles.files) > 0 && cursor.navigate(keys, len(commitFiles.files), app.height/2):
			commitFiles.row = cursor.index
			return commitFiles.loadPatch(app), true
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

func (commitFiles *commitFilesScreen) View(width, height int) string {
	parent := fmt.Sprintf(commitFilesParent, commitFiles.parent)
	if commitFiles.parent == "" {
		parent = commitFilesRootParent
	}
	lines := []string{appStyles.header.Render(fmt.Sprintf(commitFilesTitle, commitFiles.commit.OID, commitFiles.commit.Subject)), parent, fmt.Sprintf(commitFilesSummary, len(commitFiles.files))}
	history := commitFiles.history
	if history.explorer != nil {
		lines = append(lines, appStyles.warning.Render(fmt.Sprintf(fileModeLine, history.files.Len(), history.selection.Len())))
	} else {
		lines = append(lines, fmt.Sprintf(historyFilesLine, history.files.Len()))
	}
	panelHeight := max(1, height-len(lines))
	patch := commitFiles.patchLines()
	patchContent := strings.Join(patch[min(commitFiles.patchScroll, len(patch)):], "\n")
	if commitFiles.fullWidth {
		return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(append(lines, renderPanel(patchPanelTitle+" "+commitFiles.patchPath, true, patchContent, width, panelHeight)), "\n"))
	}
	var fileLines []string
	switch {
	case commitFiles.err != nil:
		fileLines = append(fileLines, appStyles.conflict.Render(commitFiles.err.Error()))
	case !commitFiles.loaded:
		fileLines = append(fileLines, commitFilesLoading)
	case len(commitFiles.files) == 0:
		fileLines = append(fileLines, appStyles.disabled.Render(commitFilesEmpty))
	}
	listHeight := max(1, panelHeight-panelBorderSize-panelTitleHeight)
	firstRow := listCursor{index: commitFiles.row}.windowStart(len(commitFiles.files), listHeight)
	for row := firstRow; row < min(len(commitFiles.files), firstRow+listHeight); row++ {
		file := commitFiles.files[row]
		line := fmt.Sprintf("%c %s", file.Status, file.Path)
		if file.OriginalPath != "" {
			line = fmt.Sprintf("%c %s%s%s", file.Status, file.OriginalPath, renameArrow, file.Path)
		}
		if file.UnsupportedReason != "" {
			line += fmt.Sprintf(unsupportedFileNote, file.UnsupportedReason)
		}
		mark := uncheckedMark
		if commitFiles.history.files.Contains(commitFiles.selectionOf(file)) {
			mark = checkedMark
		}
		line = mark + line
		if row == commitFiles.row {
			line = appStyles.selected.Render(line)
		}
		fileLines = append(fileLines, line)
	}
	filesWidth := width / 2
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		renderPanel(filesPanelTitle, !commitFiles.patchFocus, strings.Join(fileLines, "\n"), filesWidth, panelHeight),
		renderPanel(patchPanelTitle, commitFiles.patchFocus, patchContent, width-filesWidth, panelHeight))
	return lipgloss.NewStyle().Width(width).MaxHeight(max(1, height)).Render(strings.Join(append(lines, panels), "\n"))
}

func (commitFiles *commitFilesScreen) Scope() Scope { return ScopeCommitFiles }
