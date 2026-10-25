package app

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/achandrapaul/gitflow/internal/git"
)

type composeMode int

const (
	composeCommit composeMode = iota
	composeAmend
)

type draft struct {
	subject string
	body    string
}

type composerField int

const (
	fieldSubject composerField = iota
	fieldBody
	fieldReview
	fieldCancel
	composerFieldCount
)

const (
	composerBodyHeight  = 6
	composerChromeWidth = 4
	summaryLineLimit    = 12
)

type composerScreen struct {
	mode            composeMode
	subject         textinput.Model
	body            textarea.Model
	field           composerField
	initial         draft
	summary         string
	summaryErr      error
	publishedRefs   []string
	publishedLoaded bool
}

func newComposerScreen(a *App, mode composeMode) *composerScreen {
	subject := newTextInput()
	subject.Placeholder = "Subject"
	subject.Prompt = ""
	body := textarea.New()
	body.Cursor.SetMode(cursor.CursorStatic)
	body.Placeholder = "Body (optional)"
	body.ShowLineNumbers = false
	body.SetHeight(composerBodyHeight)
	composer := &composerScreen{mode: mode, subject: subject, body: body}
	if saved, ok := a.drafts[mode]; ok {
		composer.subject.SetValue(saved.subject)
		composer.body.SetValue(saved.body)
		composer.initial = saved
	}
	composer.subject.Focus()
	return composer
}

func (c *composerScreen) scope() Scope { return ScopeComposer }

func (c *composerScreen) current() draft {
	return draft{subject: strings.TrimSpace(c.subject.Value()), body: strings.TrimRight(c.body.Value(), "\n ")}
}

func (c *composerScreen) message() string {
	current := c.current()
	if current.body == "" {
		return current.subject + "\n"
	}
	return current.subject + "\n\n" + current.body + "\n"
}

func (c *composerScreen) opened(a *App) tea.Cmd {
	runner := a.runner
	_, hasDraft := a.drafts[c.mode]
	mode := c.mode
	return a.load(func(ctx context.Context) func(a *App) tea.Cmd {
		summary, summaryErr := git.StagedSummary(ctx, runner)
		var published []string
		var headMessage string
		if mode == composeAmend {
			published, _ = git.RemoteRefsContaining(ctx, runner, "HEAD")
			if !hasDraft {
				headMessage, _ = git.HeadMessage(ctx, runner)
			}
		}
		return func(a *App) tea.Cmd {
			c.summary, c.summaryErr = summary, summaryErr
			c.publishedRefs, c.publishedLoaded = published, true
			if headMessage != "" {
				subject, body, _ := strings.Cut(strings.TrimRight(headMessage, "\n"), "\n")
				c.subject.SetValue(subject)
				c.body.SetValue(strings.TrimLeft(body, "\n"))
				c.initial = c.current()
			}
			return nil
		}
	})
}

func (c *composerScreen) statusChanged(a *App) tea.Cmd { return c.opened(a) }

func (c *composerScreen) capturingText() bool {
	return c.field == fieldSubject || c.field == fieldBody
}

func (c *composerScreen) focusField(field composerField) tea.Cmd {
	c.field = field
	c.subject.Blur()
	c.body.Blur()
	switch field {
	case fieldSubject:
		return c.subject.Focus()
	case fieldBody:
		return c.body.Focus()
	}
	return nil
}

func (c *composerScreen) handleText(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		return c.focusField(fieldReview)
	case "tab":
		return c.focusField((c.field + 1) % composerFieldCount)
	case "shift+tab":
		return c.focusField((c.field + composerFieldCount - 1) % composerFieldCount)
	case "enter":
		if c.field == fieldSubject {
			return c.focusField(fieldBody)
		}
	}
	var cmd tea.Cmd
	if c.field == fieldSubject {
		c.subject, cmd = c.subject.Update(msg)
	} else {
		c.body, cmd = c.body.Update(msg)
	}
	a.drafts[c.mode] = c.current()
	return cmd
}

func (c *composerScreen) handleKey(a *App, key string) (tea.Cmd, bool) {
	switch key {
	case "tab":
		return c.focusField((c.field + 1) % composerFieldCount), true
	case "shift+tab":
		return c.focusField((c.field + composerFieldCount - 1) % composerFieldCount), true
	case "enter":
		if c.field == fieldReview {
			return a.invoke(a.registry.ByID("composer.review")), true
		}
		a.back()
		return nil, true
	}
	return nil, false
}

func (c *composerScreen) confirmLeave(a *App) bool {
	current := c.current()
	if current == c.initial || (current == draft{}) {
		if c.mode == composeAmend || current == (draft{}) {
			delete(a.drafts, c.mode)
		}
		return true
	}
	a.overlay = &confirmOverlay{
		title: "Leave composer?",
		lines: []string{"Your message is not committed. No Git command runs from this dialog."},
		buttons: []confirmButton{
			{label: "Cancel"},
			{label: "Keep draft", run: func(a *App) tea.Cmd {
				a.drafts[c.mode] = current
				a.pop()
				return nil
			}},
			{label: "Discard draft", run: func(a *App) tea.Cmd {
				delete(a.drafts, c.mode)
				a.pop()
				return nil
			}},
		},
	}
	return false
}

func (c *composerScreen) reviewUnavailable(a *App) string {
	switch {
	case a.status.HasConflicts():
		return "unmerged files: resolve conflicts first (o)"
	case c.mode == composeCommit && !a.status.HasStaged():
		return "nothing staged"
	case c.mode == composeAmend && a.status.Branch.Unborn:
		return "no commit to amend"
	case c.current().subject == "":
		return "subject is empty"
	}
	return ""
}

func (c *composerScreen) review(a *App) tea.Cmd {
	title, confirmLabel := "Review commit", "Confirm commit"
	if c.mode == composeAmend {
		title, confirmLabel = "Review amend of HEAD", "Confirm amend"
	}
	lines := []string{styleBold.Render("Message")}
	lines = append(lines, strings.Split(strings.TrimRight(c.message(), "\n"), "\n")...)
	lines = append(lines, "", styleBold.Render("Staged content (index only; working tree is not included)"))
	summaryLines := strings.Split(strings.TrimRight(c.summary, "\n"), "\n")
	if strings.TrimSpace(c.summary) == "" {
		summaryLines = []string{styleMuted.Render("(no staged changes: message-only amend)")}
	}
	if len(summaryLines) > summaryLineLimit {
		summaryLines = append(summaryLines[:summaryLineLimit], "…")
	}
	lines = append(lines, summaryLines...)
	if c.mode == composeAmend && len(c.publishedRefs) > 0 {
		lines = append(lines, "", styleWarning.Render("HEAD exists on "+strings.Join(c.publishedRefs, ", ")+": amending rewrites published history. GitFlow never force-pushes."))
	}
	commandText := "git commit --file=<message>"
	if c.mode == composeAmend {
		commandText += " --amend"
	}
	a.overlay = &confirmOverlay{
		title:   title,
		lines:   lines,
		command: commandText,
		buttons: []confirmButton{{label: "Cancel"}, {label: confirmLabel, run: func(a *App) tea.Cmd { return c.commit(a) }}},
	}
	return nil
}

func (c *composerScreen) commit(a *App) tea.Cmd {
	status, err := git.LoadStatus(a.ctx, a.runner)
	if err != nil {
		a.errorText = "Revalidate before commit: " + err.Error()
		return nil
	}
	a.status = status
	if reason := c.reviewUnavailable(a); reason != "" {
		a.errorText = "Commit blocked after revalidation: " + reason
		return nil
	}
	messagePath, cleanup, err := git.WriteMessageFile(c.message())
	if err != nil {
		a.errorText = err.Error()
		return nil
	}
	label := "Commit"
	if c.mode == composeAmend {
		label = "Amend"
	}
	return a.gitHandoff(label, git.CommitInvocation(messagePath, c.mode == composeAmend), func(a *App, err error) tea.Cmd {
		cleanup()
		if err != nil {
			a.drafts[c.mode] = c.current()
			return nil
		}
		delete(a.drafts, c.mode)
		for index, stacked := range a.stack {
			if stacked == c {
				a.popTo(index)
			}
		}
		return nil
	})
}

func (c *composerScreen) view(a *App, width, height int) string {
	title := "COMMIT COMPOSER"
	if c.mode == composeAmend {
		title = "AMEND HEAD COMMIT"
	}
	c.subject.Width = max(width-composerChromeWidth-len("Subject: "), 10)
	c.body.SetWidth(max(width-composerChromeWidth, 10))
	body := []string{"Subject: " + c.subject.View(), "Body:"}
	body = append(body, strings.Split(c.body.View(), "\n")...)
	body = append(body, "", styleBold.Render("Staged summary (real index)"))
	switch {
	case c.summaryErr != nil:
		body = append(body, styleError.Render(c.summaryErr.Error()))
	case strings.TrimSpace(c.summary) == "":
		body = append(body, styleMuted.Render("Nothing staged"))
	default:
		summaryLines := strings.Split(strings.TrimRight(c.summary, "\n"), "\n")
		if len(summaryLines) > summaryLineLimit {
			summaryLines = append(summaryLines[:summaryLineLimit], "…")
		}
		body = append(body, summaryLines...)
	}
	if c.mode == composeAmend && c.publishedLoaded && len(c.publishedRefs) > 0 {
		body = append(body, styleWarning.Render("Published: HEAD exists on "+strings.Join(c.publishedRefs, ", ")+"; amend rewrites it (no force push)"))
	}
	review, cancel := "[ Review commit ]", "[ Cancel ]"
	if c.field == fieldReview {
		review = styleFocus.Render(review)
	}
	if c.field == fieldCancel {
		cancel = styleFocus.Render(cancel)
	}
	buttons := review + "  " + cancel
	if reason := c.reviewUnavailable(a); reason != "" {
		buttons += "  " + styleMuted.Render("review unavailable: "+reason)
	}
	body = append(body, "", buttons)
	return panel(title, body, width, height, true)
}

func (c *composerScreen) footer() string {
	return "Tab Next field  Esc Leave field / composer  Enter on Review opens review  (typed keys are text)"
}

func (c *composerScreen) menuTitle(a *App) string { return "Actions for commit draft" }
