package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/achandrapaul/gitflow/internal/git"
)

type changesPanel int

const (
	panelHeader changesPanel = iota
	panelFiles
	panelDiff
	panelComposer
	panelCount
)

const (
	composerPanelHeight = 6
	statusColumnsWidth  = 14
	listChromeRows      = 5
	filePanelShare      = 45
	percent             = 100
)

type filePreview struct {
	staged    git.FileDiff
	unstaged  git.FileDiff
	untracked string
	binary    bool
	err       error
}

type changesScreen struct {
	filter      git.Filter
	focusPanel  changesPanel
	cursor      int
	focusedPath string
	offset      int
	diffOffset  int
	selected    map[string]bool
	previews    map[string]filePreview
	pathFilter  string
	searching   bool
	search      textinput.Model
}

func (c *changesScreen) scope() Scope { return ScopeChanges }

func displayPath(path string) string {
	if strings.ContainsAny(path, "\n\r\t\x1b") {
		return strconv.Quote(path)
	}
	return path
}

func (c *changesScreen) visible(a *App) []git.FileChange {
	var files []git.FileChange
	for _, file := range a.status.Files {
		if c.filter.Matches(file) && (c.pathFilter == "" || strings.Contains(strings.ToLower(file.Path), strings.ToLower(c.pathFilter))) {
			files = append(files, file)
		}
	}
	return files
}

func (c *changesScreen) focused(a *App) (git.FileChange, bool) {
	files := c.visible(a)
	if c.cursor < 0 || c.cursor >= len(files) {
		return git.FileChange{}, false
	}
	return files[c.cursor], true
}

func (c *changesScreen) targets(a *App) ([]git.FileChange, int) {
	if len(c.selected) == 0 {
		if file, ok := c.focused(a); ok {
			return []git.FileChange{file}, 0
		}
		return nil, 0
	}
	visiblePaths := map[string]bool{}
	for _, file := range c.visible(a) {
		visiblePaths[file.Path] = true
	}
	var targets []git.FileChange
	hidden := 0
	for _, file := range a.status.Files {
		if c.selected[file.Path] {
			targets = append(targets, file)
			if !visiblePaths[file.Path] {
				hidden++
			}
		}
	}
	return targets, hidden
}

func (c *changesScreen) targetScope(a *App) string {
	targets, hidden := c.targets(a)
	if len(c.selected) == 0 {
		if len(targets) == 0 {
			return "No file focused"
		}
		return "Target: focused file " + displayPath(targets[0].Path)
	}
	return fmt.Sprintf("Targets: %d selected file(s) (%d hidden by filter)", len(targets), hidden)
}

func (c *changesScreen) menuTitle(a *App) string {
	if len(c.selected) > 0 {
		return fmt.Sprintf("Actions for %d selected files", len(c.selected))
	}
	if file, ok := c.focused(a); ok {
		return "Actions for " + displayPath(file.Path)
	}
	return "Actions"
}

func (c *changesScreen) reposition(a *App) tea.Cmd {
	files := c.visible(a)
	c.cursor = min(c.cursor, max(len(files)-1, 0))
	for index, file := range files {
		if file.Path == c.focusedPath {
			c.cursor = index
		}
	}
	if file, ok := c.focused(a); ok {
		c.focusedPath = file.Path
	}
	c.diffOffset = 0
	return c.requestPreview(a)
}

func (c *changesScreen) requestPreview(a *App) tea.Cmd {
	file, ok := c.focused(a)
	if !ok {
		return nil
	}
	if _, cached := c.previews[file.Signature()]; cached {
		return nil
	}
	c.previews[file.Signature()] = filePreview{}
	runner := a.runner
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		preview := loadPreview(ctx, runner, file)
		return func(a *App) tea.Cmd {
			c.previews[file.Signature()] = preview
			return nil
		}
	})
}

func loadPreview(ctx context.Context, runner *git.Runner, file git.FileChange) filePreview {
	var preview filePreview
	if file.Conflicted() {
		return preview
	}
	if file.Kind == git.KindUntracked {
		preview.untracked, preview.binary, preview.err = git.ReadUntrackedPreview(runner.WorkTree(), file.Path)
		return preview
	}
	if file.Staged() {
		preview.staged, preview.err = git.LoadDiff(ctx, runner, file, true)
	}
	if file.Unstaged() && preview.err == nil {
		preview.unstaged, preview.err = git.LoadDiff(ctx, runner, file, false)
	}
	return preview
}

func (c *changesScreen) statusChanged(a *App) tea.Cmd {
	present := map[string]bool{}
	for _, file := range a.status.Files {
		present[file.Path] = true
	}
	for path := range c.selected {
		if !present[path] {
			delete(c.selected, path)
		}
	}
	c.previews = map[string]filePreview{}
	return c.reposition(a)
}

func (c *changesScreen) capturingText() bool { return c.searching }

func (c *changesScreen) handleText(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		c.searching, c.pathFilter = false, ""
		c.search.SetValue("")
		c.search.Blur()
		return c.reposition(a)
	case "enter":
		c.searching = false
		c.search.Blur()
		return c.reposition(a)
	}
	var cmd tea.Cmd
	c.search, cmd = c.search.Update(msg)
	c.pathFilter = c.search.Value()
	return tea.Batch(cmd, c.reposition(a))
}

func (c *changesScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "1", "2", "3", "4":
		c.filter = git.Filters[key[0]-'1']
		return c.reposition(a), true
	case "/":
		c.searching = true
		c.search.SetValue(c.pathFilter)
		return c.search.Focus(), true
	case "tab":
		c.focusPanel = (c.focusPanel + 1) % panelCount
		return nil, true
	case "shift+tab":
		c.focusPanel = (c.focusPanel + panelCount - 1) % panelCount
		return nil, true
	}
	switch c.focusPanel {
	case panelHeader:
		if key == "enter" {
			return a.openScreen(newBranchScreen()), true
		}
	case panelFiles:
		files := c.visible(a)
		if next, moved := moveCursor(key, c.cursor, len(files), a.height-listChromeRows); moved {
			c.cursor = next
			if file, ok := c.focused(a); ok {
				c.focusedPath = file.Path
			}
			c.diffOffset = 0
			return c.requestPreview(a), true
		}
		switch key {
		case " ":
			file, ok := c.focused(a)
			if !ok {
				a.notice = "No file to select"
				return nil, true
			}
			if c.selected[file.Path] {
				delete(c.selected, file.Path)
			} else {
				c.selected[file.Path] = true
			}
			return nil, true
		case "S":
			files := c.visible(a)
			if len(files) == 0 {
				a.notice = "No files to select"
				return nil, true
			}
			allSelected := true
			for _, file := range files {
				allSelected = allSelected && c.selected[file.Path]
			}
			for _, file := range files {
				if allSelected {
					delete(c.selected, file.Path)
				} else {
					c.selected[file.Path] = true
				}
			}
			return nil, true
		case "enter":
			file, ok := c.focused(a)
			if !ok {
				a.notice = "Working tree clean: nothing to open"
				return nil, true
			}
			return a.openScreen(newDiffScreen(file)), true
		}
	case panelDiff:
		switch key {
		case "j", "down":
			c.diffOffset++
			return nil, true
		case "k", "up":
			c.diffOffset = max(c.diffOffset-1, 0)
			return nil, true
		case "enter":
			if file, ok := c.focused(a); ok {
				return a.openScreen(newDiffScreen(file)), true
			}
		}
	case panelComposer:
		if key == "enter" {
			return a.invoke(a.registry.ByID("changes.compose")), true
		}
	}
	return nil, false
}

func diffLines(diff git.FileDiff) []string {
	var lines []string
	for _, hunk := range diff.Hunks {
		lines = append(lines, styleAccent.Render(fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount, hunk.Section)))
		for _, line := range hunk.Lines {
			text := string(line.Kind) + line.Text
			switch line.Kind {
			case '+':
				text = styleStaged.Render(text)
			case '-':
				text = styleError.Render(text)
			case '\\':
				text = styleMuted.Render(text)
			}
			lines = append(lines, text)
		}
	}
	if diff.Binary {
		lines = append(lines, styleMuted.Render("Binary file: no text diff"))
	} else if len(diff.Hunks) == 0 && !diff.Empty() {
		lines = append(lines, styleMuted.Render(strings.Join(diff.Header, " · ")))
	}
	return lines
}

func statusCode(code byte, untracked bool) string {
	switch {
	case untracked:
		return "??"
	case code == '.':
		return "-"
	}
	return string(code)
}

func (c *changesScreen) view(a *App, width, height int) string {
	topHeight := height - composerPanelHeight
	filesWidth, diffWidth := width/2, width-width/2
	filesHeight, diffHeight := topHeight, topHeight
	stacked := width < stackedLayoutBelow
	if stacked {
		filesWidth, diffWidth = width, width
		filesHeight = topHeight * filePanelShare / percent
		diffHeight = topHeight - filesHeight
	}

	var counts [4]int
	for _, file := range a.status.Files {
		for filterIndex, filter := range git.Filters {
			if filter.Matches(file) {
				counts[filterIndex]++
			}
		}
	}
	var tabs []string
	for filterIndex, filter := range git.Filters {
		label := fmt.Sprintf("[%d %s:%d]", filterIndex+1, filter.Label(), counts[filterIndex])
		if filter == c.filter {
			label = styleFocus.Render(label)
		}
		tabs = append(tabs, label)
	}
	fileBody := []string{strings.Join(tabs, " ")}
	if c.searching || c.pathFilter != "" {
		if c.searching {
			fileBody = append(fileBody, c.search.View())
		} else {
			fileBody = append(fileBody, styleMuted.Render("path filter: "+c.pathFilter+" (/ to edit)"))
		}
	}
	files := c.visible(a)
	innerWidth := filesWidth - panelBorderSize - 2
	fileBody = append(fileBody, styleMuted.Render(fmt.Sprintf("SEL IDX WT  %s", "PATH")))
	switch {
	case !a.statusLoaded:
		fileBody = append(fileBody, styleMuted.Render("Loading status…"))
	case a.statusErr != nil:
		fileBody = append(fileBody, styleError.Render(a.statusErr.Error()))
	case len(a.status.Files) == 0:
		fileBody = append(fileBody, "Working tree clean (0 files)")
	case len(files) == 0:
		fileBody = append(fileBody, styleMuted.Render("No files in "+c.filter.Label()))
	}
	rows := make([]string, len(files))
	for index, file := range files {
		mark := "[ ]"
		if c.selected[file.Path] {
			mark = styleAccent.Render("[x]")
		}
		indexCode := statusCode(file.Index, false)
		worktreeCode := statusCode(file.Worktree, file.Kind == git.KindUntracked)
		if file.Kind == git.KindUntracked {
			indexCode = "-"
		}
		indexCell := fmt.Sprintf("%-3s", indexCode)
		worktreeCell := fmt.Sprintf("%-3s", worktreeCode)
		if file.Staged() {
			indexCell = styleStaged.Render(indexCell)
		}
		if file.Unstaged() {
			worktreeCell = styleDirty.Render(worktreeCell)
		}
		path := displayPath(file.Path)
		if file.OrigPath != "" {
			path = displayPath(file.OrigPath) + " → " + path
		}
		suffix := ""
		switch {
		case file.Conflicted():
			suffix = " " + styleConflict.Render("CONFLICT")
			indexCell, worktreeCell = styleConflict.Render(fmt.Sprintf("%-3s", string(file.Index))), styleConflict.Render(fmt.Sprintf("%-3s", string(file.Worktree)))
		case file.Mixed():
			suffix = " " + styleWarning.Render("MIXED")
		}
		pathWidth := max(innerWidth-statusColumnsWidth-lipgloss.Width(suffix), 8)
		rows[index] = fmt.Sprintf("%s %s %s %-*s", mark, indexCell, worktreeCell, pathWidth, clip(path, pathWidth)) + suffix
	}
	listRows := filesHeight - panelBorderSize - len(fileBody) - 2
	renderedRows, offset := renderRows(rows, c.cursor, c.offset, listRows, c.focusPanel == panelFiles)
	c.offset = offset
	fileBody = append(fileBody, renderedRows...)
	_, hidden := c.targets(a)
	if len(c.selected) > 0 {
		fileBody = append(fileBody, styleAccent.Render(fmt.Sprintf("%d selected (%d hidden by filter)", len(c.selected), hidden)))
	}
	filesPanel := panel("CHANGES", fileBody, filesWidth, filesHeight, c.focusPanel == panelFiles)

	diffTitle := "DIFF INSPECTOR"
	var diffBody []string
	if file, ok := c.focused(a); ok {
		diffTitle += " · " + displayPath(file.Path)
		preview := c.previews[file.Signature()]
		switch {
		case preview.err != nil:
			diffBody = append(diffBody, styleError.Render(preview.err.Error()))
		case file.Conflicted():
			diffBody = append(diffBody, styleConflict.Render("Unresolved conflict."), "Open the operation dashboard with o to resolve it.")
		case file.Kind == git.KindUntracked:
			diffBody = append(diffBody, styleDirty.Render("UNTRACKED · not in index"))
			if preview.binary {
				diffBody = append(diffBody, styleMuted.Render("Binary content"))
			}
			for _, line := range strings.Split(preview.untracked, "\n") {
				diffBody = append(diffBody, styleDirty.Render("+"+line))
			}
		default:
			if file.Staged() {
				diffBody = append(diffBody, styleStaged.Render("┌ STAGED · HEAD → INDEX"))
				diffBody = append(diffBody, diffLines(preview.staged)...)
			}
			if file.Unstaged() {
				diffBody = append(diffBody, styleDirty.Render("┌ UNSTAGED · INDEX → WORKTREE"))
				diffBody = append(diffBody, diffLines(preview.unstaged)...)
			}
		}
		c.diffOffset = min(c.diffOffset, max(len(diffBody)-1, 0))
		diffBody = append(diffBody[c.diffOffset:], "")
		diffBody = append(diffBody, styleMuted.Render("Enter: full diff  s Stage  u Unstage  d Discard"))
	} else {
		diffBody = append(diffBody, styleMuted.Render("No file focused"))
	}
	diffPanel := panel(diffTitle, diffBody, diffWidth, diffHeight, c.focusPanel == panelDiff)

	subject := a.drafts[composeCommit].subject
	if subject == "" {
		subject = styleMuted.Render("(no draft)")
	}
	stagedCount := 0
	for _, file := range a.status.Files {
		if file.Staged() {
			stagedCount++
		}
	}
	stagedLine := fmt.Sprintf("Staged: %d file(s) · Only index content will be committed", stagedCount)
	if stagedCount == 0 {
		stagedLine = styleMuted.Render("Nothing staged · commit unavailable (stage with s)")
	}
	composerPanel := panel("COMMIT COMPOSER", []string{"Subject: " + subject, stagedLine, "[c c Compose] [c a Amend] [a Actions]"}, width, composerPanelHeight, c.focusPanel == panelComposer)

	if stacked {
		return lipgloss.JoinVertical(lipgloss.Left, filesPanel, diffPanel, composerPanel)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, filesPanel, diffPanel), composerPanel)
}

func (c *changesScreen) footer() string {
	return "j/k Move  Tab Panel  Enter Open  Space Select  S Select all  1-4 Filter  / Search  s/u Stage/Unstage  d Discard  c c Commit  c a Amend  a Actions  o Ops  q Quit"
}
