package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"fmt"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/charmbracelet/x/ansi"
	"io"
	"os"
	"strings"
)

const WideLayoutMinColumns = 80

type dataMsg struct {
	feeds    []domain.Feed
	articles []domain.Article
	err      error
	status   string
}
type Model struct {
	app           *app.App
	ctx           context.Context
	width, height int
	feeds         []domain.Feed
	articles      []domain.Article
	filter        domain.ArticleFilter
	selected      int64
	feedIndex     int
	articleIndex  int
	focusFeeds    bool
	picker        bool
	input         bool
	url           string
	confirm       bool
	help          bool
	reader        *domain.Article
	scroll        int
	loading       bool
	theme         Theme
	styles        Styles
	themeFile     string
	noColor       bool
	status        string
	err           error
}

func New(ctx context.Context, a *app.App) *Model {
	home, _ := os.UserHomeDir()
	return NewWithTheme(ctx, a, themePath(home), os.Getenv("NO_COLOR") != "")
}
func NewWithTheme(ctx context.Context, a *app.App, path string, noColor bool) *Model {
	theme := loadTheme(path, noColor)
	return &Model{app: a, ctx: ctx, width: 80, height: 24, loading: true, theme: theme, styles: makeStyles(theme), themeFile: path, noColor: noColor}
}
func Run(ctx context.Context, a *app.App, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, err := tea.NewProgram(New(ctx, a), tea.WithContext(ctx), tea.WithInput(os.Stdin), tea.WithOutput(out)).Run()
	return err
}
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.work(nil), watchTheme(m.ctx, m.themeFile, m.noColor))
}
func (m *Model) work(operation func() (string, error)) tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		status := ""
		var err error
		if operation != nil {
			status, err = operation()
		}
		feeds, e := m.app.ListFeeds(m.ctx)
		if err == nil {
			err = e
		}
		articles, e := m.app.ListArticles(m.ctx, domain.ArticleFilter{})
		if err == nil {
			err = e
		}
		return dataMsg{feeds, articles, err, status}
	}
}
func (m *Model) wide() bool { return m.width >= WideLayoutMinColumns }
func (m *Model) visible() []domain.Article {
	items := []domain.Article{}
	for _, a := range m.articles {
		if (m.filter.FeedID == 0 || a.FeedID == m.filter.FeedID) && (!m.filter.UnreadOnly || !a.Read) {
			items = append(items, a)
		}
	}
	return items
}
func (m *Model) reconcile() {
	if m.filter.FeedID != 0 {
		found := false
		for _, f := range m.feeds {
			found = found || f.ID == m.filter.FeedID
		}
		if !found {
			m.filter.FeedID = 0
		}
	}
	m.feedIndex = min(max(0, m.feedIndex), len(m.feeds))
	items := m.visible()
	m.articleIndex = min(max(0, m.articleIndex), max(0, len(items)-1))
	for i, a := range items {
		if a.ID == m.selected {
			m.articleIndex = i
			break
		}
	}
	m.selected = 0
	if len(items) > 0 {
		m.selected = items[m.articleIndex].ID
	}
}
func (m *Model) feedID() int64 {
	if m.feedIndex == 0 || m.feedIndex > len(m.feeds) {
		return 0
	}
	return m.feeds[m.feedIndex-1].ID
}
func (m *Model) chooseFeed() {
	m.filter.FeedID = m.feedID()
	m.articleIndex = 0
	m.selected = 0
	m.reconcile()
}
func (m *Model) refresh(id int64) tea.Cmd {
	return m.work(func() (string, error) {
		results, err := m.app.Refresh(m.ctx, id)
		lines := []string{}
		for _, r := range results {
			if r.Err != nil {
				lines = append(lines, fmt.Sprintf("Feed %d: %s", r.Feed.ID, app.SafeText(r.Err.Error())))
				err = errors.Join(err, fmt.Errorf("feed %d: %w", r.Feed.ID, r.Err))
			} else {
				lines = append(lines, fmt.Sprintf("Feed %d updated", r.Feed.ID))
			}
		}
		return strings.Join(lines, "; "), err
	})
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case themeMsg:
		m.theme = msg.theme
		m.styles = makeStyles(msg.theme)
		return m, watchTheme(m.ctx, m.themeFile, m.noColor)
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
		m.scroll = min(m.scroll, m.maxScroll())
		return m, nil
	case dataMsg:
		m.loading = false
		m.err = msg.err
		m.status = msg.status
		m.feeds = msg.feeds
		m.articles = msg.articles
		m.reconcile()
		if m.reader != nil {
			for _, a := range m.articles {
				if a.ID == m.reader.ID {
					copy := a
					m.reader = &copy
				}
			}
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.input && msg.Text != "" {
			m.url += app.SafeText(msg.Text)
			return m, nil
		}
		return m.key(msg.String())
	case tea.PasteMsg:
		if m.input {
			m.url += strings.Join(strings.Fields(app.SafeText(msg.Content)), " ")
		}
	}
	return m, nil
}
func (m *Model) key(key string) (tea.Model, tea.Cmd) {
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.input {
		switch key {
		case "esc":
			m.input = false
			m.url = ""
		case "enter":
			if !m.loading {
				url := m.url
				m.input = false
				m.url = ""
				return m, m.work(func() (string, error) { _, err := m.app.AddFeed(m.ctx, url); return "Feed added", err })
			}
		case "backspace":
			r := []rune(m.url)
			if len(r) > 0 {
				m.url = string(r[:len(r)-1])
			}
		default:
			if key == "space" {
				m.url += " "
			} else if len([]rune(key)) == 1 {
				m.url += key
			}
		}
		return m, nil
	}
	if key == "q" {
		return m, tea.Quit
	}
	if m.help {
		if key == "esc" || key == "?" {
			m.help = false
		}
		return m, nil
	}
	if key == "?" {
		m.help = true
		return m, nil
	}
	if m.confirm {
		if key == "esc" || key == "n" {
			m.confirm = false
		}
		if (key == "y" || key == "enter") && !m.loading {
			id := m.feedID()
			m.confirm = false
			return m, m.work(func() (string, error) { return "Feed removed", m.app.RemoveFeed(m.ctx, id) })
		}
		return m, nil
	}
	if m.reader != nil {
		switch key {
		case "u":
			m.filter.UnreadOnly = !m.filter.UnreadOnly
			m.reconcile()
		case "R":
			if !m.loading {
				return m, m.refresh(0)
			}
		case "esc":
			m.reader = nil
			m.scroll = 0
		case "down", "j":
			m.scroll = min(m.scroll+1, m.maxScroll())
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "pgdown":
			m.scroll = min(m.scroll+max(1, m.height-4), m.maxScroll())
		case "pgup":
			m.scroll = max(0, m.scroll-max(1, m.height-4))
		case "m":
			if !m.loading {
				id, read := m.reader.ID, !m.reader.Read
				return m, m.work(func() (string, error) { return "Read state saved", m.app.SetRead(m.ctx, id, read) })
			}
		}
		return m, nil
	}
	if key == "f" && !m.wide() {
		m.picker = true
		for i, f := range m.feeds {
			if f.ID == m.filter.FeedID {
				m.feedIndex = i + 1
			}
		}
		if m.filter.FeedID == 0 {
			m.feedIndex = 0
		}
		return m, nil
	}
	if key == "esc" && m.picker {
		m.picker = false
		return m, nil
	}
	feedFocus := m.picker || (m.wide() && m.focusFeeds)
	switch key {
	case "tab":
		if m.wide() {
			m.focusFeeds = !m.focusFeeds
		}
	case "down", "j", "up", "k":
		delta := 1
		if key == "up" || key == "k" {
			delta = -1
		}
		if feedFocus {
			m.feedIndex = min(max(0, m.feedIndex+delta), len(m.feeds))
			if !m.picker {
				m.chooseFeed()
			}
		} else {
			items := m.visible()
			m.articleIndex = min(max(0, m.articleIndex+delta), max(0, len(items)-1))
			m.selected = 0
			m.reconcile()
		}
	case "enter":
		if feedFocus {
			m.chooseFeed()
			m.picker = false
			m.focusFeeds = false
		} else {
			items := m.visible()
			if len(items) > 0 && !m.loading {
				article := items[m.articleIndex]
				m.reader = &article
				m.scroll = 0
				return m, m.work(func() (string, error) { return "Read state saved", m.app.SetRead(m.ctx, article.ID, true) })
			}
		}
	case "u":
		m.filter.UnreadOnly = !m.filter.UnreadOnly
		m.reconcile()
	case "m":
		items := m.visible()
		if len(items) > 0 && !m.loading {
			article := items[m.articleIndex]
			return m, m.work(func() (string, error) { return "Read state saved", m.app.SetRead(m.ctx, article.ID, !article.Read) })
		}
	case "a":
		if (m.wide() || m.picker) && !m.loading {
			m.input = true
			m.url = ""
		}
	case "d":
		if (m.wide() || m.picker) && m.feedID() != 0 && !m.loading {
			m.confirm = true
		}
	case "r":
		if !m.loading && (m.wide() || m.picker) {
			id := m.filter.FeedID
			if feedFocus {
				id = m.feedID()
			}
			return m, m.refresh(id)
		}
	case "R":
		if !m.loading {
			return m, m.refresh(0)
		}
	}
	return m, nil
}
func (m *Model) readerLines() []string {
	if m.reader == nil {
		return nil
	}
	text := m.styles.Emphasis.Render(app.SafeText(m.reader.Title)) + "\n" + app.SafeText(m.reader.URL) + "\n\n" + app.PlainText(m.reader.Content)
	return strings.Split(ansi.Hardwrap(ansi.Wrap(text, max(1, m.width-2), ""), max(1, m.width-2), false), "\n")
}
func (m *Model) maxScroll() int      { return max(0, len(m.readerLines())-max(1, m.height-4)) }
func fit(s string, width int) string { return ansi.Truncate(s, max(1, width), "") }
func (m *Model) feedTitle(id int64) string {
	if id == 0 {
		return "All feeds"
	}
	for _, f := range m.feeds {
		if f.ID == id {
			if f.Title != "" {
				return strings.Join(strings.Fields(app.SafeText(f.Title)), " ")
			}
			return app.SafeText(f.URL)
		}
	}
	return "Unknown feed"
}
func (m *Model) feedRows() []string {
	rows := []string{}
	for i := 0; i <= len(m.feeds); i++ {
		prefix := "  "
		if i == m.feedIndex {
			prefix = "> "
		}
		id := int64(0)
		if i > 0 {
			id = m.feeds[i-1].ID
		}
		row := prefix + m.feedTitle(id)
		if i == m.feedIndex {
			row = m.styles.Selected.Render(row)
		}
		rows = append(rows, row)
	}
	return window(rows, m.feedIndex, max(1, m.height-5))
}
func window(rows []string, selected, height int) []string {
	start := max(0, selected-height+1)
	end := min(len(rows), start+height)
	return rows[start:end]
}
func (m *Model) articleRows() []string {
	rows := []string{}
	for i, a := range m.visible() {
		prefix := "  "
		if i == m.articleIndex {
			prefix = "> "
		}
		state := "* "
		if a.Read {
			state = "  "
		}
		body := a.PublishedAt.Format("2006-01-02") + " " + m.feedTitle(a.FeedID) + " | " + strings.Join(strings.Fields(app.SafeText(a.Title)), " ")
		row := prefix + state + body
		if i == m.articleIndex {
			row = m.styles.Selected.Render(row)
		} else if a.Read {
			row = m.styles.Read.Render(row)
		} else {
			row = prefix + m.styles.Unread.Render(state) + body
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return []string{"No articles"}
	}
	return window(rows, m.articleIndex, max(1, m.height-5))
}
func (m *Model) View() tea.View {
	lines := []string{m.styles.Focus.Render(fmt.Sprintf("syn | %s | unread: %t", m.feedTitle(m.filter.FeedID), m.filter.UnreadOnly))}
	switch {
	case m.input:
		lines = append(lines, "Feed URL: "+app.SafeText(m.url), "Enter: add | Esc: cancel")
	case m.confirm:
		lines = append(lines, "Remove "+m.feedTitle(m.feedID())+" and its articles?", "y/Enter: remove | n/Esc: cancel")
	case m.help:
		lines = append(lines, "j/k or arrows: move | Enter: read | Esc: return", "m: read state | u: unread filter | R: refresh all", "Wide: Tab: focus | a: add | d: remove | r: refresh", "Narrow: f: feed picker | Tab: no action", "Picker: Enter: filter | a: add | d: remove | r: refresh", "Reader: arrows, PageUp, PageDown: scroll", "?: help | q/Ctrl+C: quit")
	case m.reader != nil:
		rows := m.readerLines()
		start := min(m.scroll, len(rows))
		end := min(len(rows), start+max(1, m.height-4))
		lines = append(lines, rows[start:end]...)
		lines = append(lines, "Esc: return | arrows/PageUp/PageDown: scroll")
	case m.picker:
		lines = append(lines, m.styles.Focus.Render("Feed picker"))
		lines = append(lines, m.feedRows()...)
		lines = append(lines, "Enter: filter | Esc: cancel | a/d/r: manage")
	case m.wide():
		left, right := m.feedRows(), m.articleRows()
		pane := max(20, m.width/3)
		focus := "Articles"
		if m.focusFeeds {
			focus = "Feeds"
		}
		lines = append(lines, m.styles.Focus.Render("Focus: "+focus))
		for i := 0; i < max(len(left), len(right)); i++ {
			l, r := "", ""
			if i < len(left) {
				l = fit(left[i], pane-2)
			}
			if i < len(right) {
				r = right[i]
			}
			lines = append(lines, l+strings.Repeat(" ", max(0, pane-ansi.StringWidth(l)))+"| "+r)
		}
		lines = append(lines, "Tab: focus | Enter: read | a/d/r: manage | ?: help")
	default:
		lines = append(lines, m.articleRows()...)
		lines = append(lines, "f: feeds | Enter: read | u: unread | ?: help")
	}
	if m.loading {
		lines = append(lines, "Loading...")
	} else if m.err != nil {
		lines = append(lines, m.styles.Error.Render("Error: "+app.SafeText(m.err.Error())))
	} else if m.status != "" {
		lines = append(lines, app.SafeText(m.status))
	}
	for i := range lines {
		lines[i] = fit(lines[i], m.width)
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	view := tea.NewView(strings.Join(lines, "\n"))
	view.AltScreen = true
	return view
}
