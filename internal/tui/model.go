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
	"time"
)

const WideLayoutMinColumns = 81

type dataMsg struct {
	feeds    []domain.Feed
	articles []domain.Article
	err      error
	status   string
}
type Model struct {
	app            *app.App
	ctx            context.Context
	width, height  int
	feeds          []domain.Feed
	articles       []domain.Article
	filter         domain.ArticleFilter
	selected       int64
	feedIndex      int
	articleIndex   int
	picker         bool
	input          bool
	url            string
	confirm        bool
	help           bool
	reader         *domain.Article
	scroll         int
	loading        bool
	theme          Theme
	styles         Styles
	themeFile      string
	noColor        bool
	status         string
	err            error
	now            func() time.Time
	stateFile      string
	preferredWidth int
	dragging       bool
	saving         bool
	pendingWidth   int
}

func New(ctx context.Context, a *app.App) *Model {
	home, _ := os.UserHomeDir()
	return NewWithOptions(ctx, a, Options{ThemePath: themePath(home), StatePath: statePath(os.Getenv("XDG_STATE_HOME"), home), NoColor: os.Getenv("NO_COLOR") != "", Now: time.Now})
}
func NewWithTheme(ctx context.Context, a *app.App, path string, noColor bool) *Model {
	return NewWithOptions(ctx, a, Options{ThemePath: path, NoColor: noColor, Now: time.Now})
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
	case widthSavedMsg:
		m.saving = false
		m.err = msg.err
		if msg.err == nil {
			m.status = "Index width saved"
		}
		return m, m.startWidthSave()
	case tea.MouseClickMsg:
		if m.mainWide() && msg.Button == tea.MouseLeft && msg.X == m.indexWidth() && msg.Y >= len(m.headerRows()) && msg.Y < m.height-1 {
			m.dragging = true
		}
		return m, nil
	case tea.MouseMotionMsg:
		if m.dragging && m.mainWide() && msg.Button == tea.MouseLeft {
			m.resizeIndex(msg.X)
		}
		return m, nil
	case tea.MouseReleaseMsg:
		if m.dragging && msg.Button == tea.MouseLeft {
			m.dragging = false
			return m, m.queueWidthSave()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.dragging = false
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
		m.scroll = min(m.scroll, m.maxScroll())
		return m, m.startWidthSave()
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
			if !m.busy() {
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
		m.dragging = false
		m.help = true
		return m, nil
	}
	if m.confirm {
		if key == "esc" || key == "n" {
			m.confirm = false
		}
		if (key == "y" || key == "enter") && !m.busy() {
			id := m.feedID()
			m.confirm = false
			return m, m.work(func() (string, error) { return "Feed removed", m.app.RemoveFeed(m.ctx, id) })
		}
		return m, nil
	}
	if key == "f" && !m.picker && (m.reader == nil || m.wide()) {
		m.dragging = false
		m.picker = true
		m.feedIndex = 0
		for i, f := range m.feeds {
			if f.ID == m.filter.FeedID {
				m.feedIndex = i + 1
			}
		}
		return m, nil
	}
	if m.picker && key == "esc" {
		m.picker = false
		return m, nil
	}
	if (key == "<" || key == ">") && m.mainWide() {
		delta := IndexResizeStep
		if key == "<" {
			delta = -delta
		}
		before := m.indexWidth()
		m.resizeIndex(before + delta)
		if before != m.indexWidth() {
			return m, m.queueWidthSave()
		}
		return m, nil
	}
	if m.reader != nil && !m.picker {
		switch key {
		case "u":
			m.filter.UnreadOnly = !m.filter.UnreadOnly
			m.reconcile()
		case "R":
			if !m.busy() {
				return m, m.refresh(0)
			}
		case "tab":
			if !m.wide() {
				return m, nil
			}
			m.reader = nil
			m.scroll = 0
		case "esc":
			m.reader = nil
			m.scroll = 0
		case "down", "j":
			m.scroll = min(m.scroll+1, m.maxScroll())
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "pgdown":
			m.scroll = min(m.scroll+max(1, m.readerHeight()), m.maxScroll())
		case "pgup":
			m.scroll = max(0, m.scroll-max(1, m.readerHeight()))
		case "m":
			if !m.busy() {
				id, read := m.reader.ID, !m.reader.Read
				return m, m.work(func() (string, error) { return "Read state saved", m.app.SetRead(m.ctx, id, read) })
			}
		}
		return m, nil
	}
	switch key {
	case "down", "j", "up", "k":
		delta := 1
		if key == "up" || key == "k" {
			delta = -1
		}
		if m.picker {
			m.feedIndex = min(max(0, m.feedIndex+delta), len(m.feeds))
		} else {
			items := m.visible()
			m.articleIndex = min(max(0, m.articleIndex+delta), max(0, len(items)-1))
			m.selected = 0
			m.scroll = 0
			m.reconcile()
		}
	case "enter":
		if m.picker {
			m.chooseFeed()
			m.picker = false
			m.reader = nil
			m.scroll = 0
		} else {
			items := m.visible()
			if len(items) > 0 && !m.busy() {
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
		if len(items) > 0 && !m.busy() {
			article := items[m.articleIndex]
			return m, m.work(func() (string, error) { return "Read state saved", m.app.SetRead(m.ctx, article.ID, !article.Read) })
		}
	case "a":
		if m.picker && !m.busy() {
			m.input = true
			m.url = ""
		}
	case "d":
		if m.picker && m.feedID() != 0 && !m.busy() {
			m.confirm = true
		}
	case "r":
		if !m.busy() && m.picker {
			return m, m.refresh(m.feedID())
		}
	case "R":
		if !m.busy() {
			return m, m.refresh(0)
		}
	}
	return m, nil
}

const NewspaperColumnWidth = 60

func wrapWords(text string, width int) []string {
	width = max(1, width)
	lines := []string{}
	line := ""
	flush := func() {
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
	}
	for _, word := range strings.Fields(text) {
		if ansi.StringWidth(word) > width {
			flush()
			piece := ""
			for word != "" {
				cluster, cells := ansi.FirstGraphemeCluster(word, ansi.GraphemeWidth)
				word = word[len(cluster):]
				if cells > width {
					continue
				}
				if ansi.StringWidth(piece)+cells > width {
					lines = append(lines, piece)
					piece = ""
				}
				piece += cluster
			}
			line = piece
			continue
		}
		if line == "" {
			line = word
		} else if ansi.StringWidth(line)+1+ansi.StringWidth(word) <= width {
			line += " " + word
		} else {
			flush()
			line = word
		}
	}
	flush()
	return lines
}

func justifyText(text string, width int) []string {
	lines := []string{}
	for _, paragraph := range strings.Split(text, "\n\n") {
		wrapped := wrapWords(paragraph, width)
		if len(wrapped) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for i, line := range wrapped {
			words := strings.Fields(line)
			if i < len(wrapped)-1 && len(words) > 1 {
				extra := width - ansi.StringWidth(line)
				gaps := len(words) - 1
				line = words[0]
				for j, word := range words[1:] {
					spaces := 1 + extra/gaps
					if j < extra%gaps {
						spaces++
					}
					line += strings.Repeat(" ", spaces) + word
				}
			}
			lines = append(lines, line)
		}
	}
	return lines
}

func (m *Model) readerLines() []string {
	if m.reader == nil {
		return nil
	}
	width := max(1, min(NewspaperColumnWidth, m.width-4))
	indent := strings.Repeat(" ", max(0, (m.width-width)/2))
	lines := []string{}
	rule := func(glyph string) { lines = append(lines, m.styles.Unread.Render(strings.Repeat(glyph, m.width))) }
	centered := func(text string, layoutWidth int, render func(...string) string) {
		for _, line := range wrapWords(text, layoutWidth) {
			lines = append(lines, strings.Repeat(" ", max(0, (m.width-layoutWidth)/2+(layoutWidth-ansi.StringWidth(line))/2))+render(line))
		}
	}
	rule("═")
	centered(strings.ToUpper(m.feedTitle(m.reader.FeedID)), m.width, m.styles.Unread.Render)
	if !m.reader.PublishedAt.IsZero() {
		centered(m.reader.PublishedAt.Format("2 January 2006"), m.width, m.styles.Unread.Render)
	}
	rule("═")
	centered(strings.ToUpper(app.SafeText(m.reader.Title)), width, m.styles.Emphasis.Render)
	rule("─")
	centered(app.ArticleSubtitle(m.reader.Subtitle, m.reader.Content), width, m.styles.Read.Render)
	centered(app.ArticleByline(m.reader.Author), width, m.styles.Read.Render)
	lines = append(lines, "")
	for _, line := range justifyText(app.PlainText(m.reader.Content), width) {
		lines = append(lines, indent+line)
	}
	return lines
}
func (m *Model) readerHeight() int {
	if m.wide() {
		return max(0, m.height-len(m.mastheadRows())-1)
	}
	height := max(0, m.height-1)
	if m.operationRow() != "" {
		height = max(0, height-1)
	}
	return height
}
func (m *Model) maxScroll() int {
	if m.wide() {
		return max(0, len(m.previewLines())-m.readerHeight())
	}
	return max(0, len(m.readerLines())-m.readerHeight())
}
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
			return singleLine(f.URL)
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
	return window(rows, m.feedIndex, m.listHeight())
}
func window(rows []string, selected, height int) []string {
	height = max(0, height)
	start := min(len(rows), max(0, selected-height+1))
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
	return window(rows, m.articleIndex, m.listHeight())
}
func (m *Model) operationRow() string {
	if m.loading {
		return "Loading..."
	}
	if m.err != nil {
		return m.styles.Error.Render("Error: " + singleLine(m.err.Error()))
	}
	return singleLine(m.status)
}

func singleLine(text string) string { return strings.Join(strings.Fields(app.SafeText(text)), " ") }

func (m *Model) headerRows() []string {
	if m.mainWide() {
		return m.mastheadRows()
	}
	text := fmt.Sprintf("syn | %s | unread: %t", singleLine(m.feedTitle(m.filter.FeedID)), m.filter.UnreadOnly)
	if m.picker {
		text += " | Feed picker"
	}
	rows := []string{m.styles.Focus.Render(text)}
	if status := m.operationRow(); status != "" {
		if m.height >= 4 {
			rows = append(rows, status)
		} else {
			rows[0] += " | " + status
		}
	}
	return rows
}

func (m *Model) listHeight() int { return max(0, m.height-len(m.headerRows())-1) }

func padRow(text string, width int) string {
	text = fit(text, width)
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text)))
}

func (m *Model) frame(rows []string, bar string) tea.View {
	height := max(1, m.height)
	rows = rows[:min(len(rows), height-1)]
	for len(rows) < height-1 {
		rows = append(rows, "")
	}
	rows = append(rows, bar)
	for i := range rows {
		rows[i] = padRow(rows[i], m.width)
	}
	view := tea.NewView(strings.Join(rows, "\n"))
	view.AltScreen = true
	if m.mainWide() {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
}

func (m *Model) View() tea.View {
	rows := m.headerRows()
	bar := ""
	switch {
	case m.input:
		rows = append(rows, "Feed URL: "+singleLine(m.url))
		bar = "Enter: add | Esc: cancel"
	case m.confirm:
		rows = append(rows, "Remove "+singleLine(m.feedTitle(m.feedID()))+" and its articles?")
		bar = "y/Enter: remove | n/Esc: cancel"
	case m.help:
		rows = append(rows, "j/k or arrows: move | Enter: read | Esc: return", "m: read state | u: unread filter | R: refresh all", "Wide: Tab: return to index | </>: resize index", "Main layouts and wide preview: f: feed picker", "Narrow reader: f and Tab: no action", "Picker: Enter: filter | a: add | d: remove | r: refresh", "Reader: arrows, PageUp, PageDown: scroll")
		bar = "Esc/?: return | q/Ctrl+C: quit"
	case m.reader != nil && !m.wide() && !m.picker:
		content := m.readerLines()
		start := min(m.scroll, len(content))
		rows = append([]string{}, content[start:min(len(content), start+m.readerHeight())]...)
		for len(rows) < m.readerHeight() {
			rows = append(rows, "")
		}
		if status := m.operationRow(); status != "" && m.height > 1 {
			rows = append(rows, status)
		}
		bar = "Esc: return | arrows/PageUp/PageDown: scroll"
	case m.picker:
		rows = append(rows, m.feedRows()...)
		bar = "Enter: filter | Esc: cancel | a/d/r: manage"
	case m.wide():
		left, right := m.indexRows(), m.previewLines()
		start := min(m.scroll, len(right))
		right = right[start:]
		leftWidth := m.indexWidth()
		rightWidth := m.width - leftWidth - 1
		for i := 0; i < m.listHeight(); i++ {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			rows = append(rows, padRow(l, leftWidth)+m.styles.Unread.Render("│")+padRow(r, rightWidth))
		}
		bar = "f: feeds | Enter: read | </>: width | ?: help"
		if m.reader != nil {
			bar = "Tab/Esc: index | arrows/PageUp/PageDown: scroll | f: feeds"
		}

	default:
		rows = append(rows, m.articleRows()...)
		bar = "f: feeds | Enter: read | u: unread | ?: help"
	}
	return m.frame(rows, bar)
}
