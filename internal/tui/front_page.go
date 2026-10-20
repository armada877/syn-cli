package tui

import (
	"context"
	"github.com/armada877/syn-cli/internal/config"
	"github.com/armada877/syn-cli/internal/images"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/charmbracelet/x/ansi"
)

const (
	MinimumIndexWidth   = 20
	MinimumPreviewWidth = 40
	IndexResizeStep     = 1
)

type Options struct {
	ConfigPath      string
	ImageConfig     *config.Config
	ImageOverride   string
	ImageService    *images.Service
	KittyCapability *bool
	ThemePath       string
	StatePath       string
	NoColor         bool
	Now             func() time.Time
}

func NewWithOptions(ctx context.Context, a *app.App, options Options) *Model {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	theme := loadTheme(options.ThemePath, options.NoColor)
	m := &Model{app: a, ctx: ctx, width: 80, height: 24, loading: true, theme: theme, styles: makeStyles(theme), themeFile: options.ThemePath, noColor: options.NoColor, stateFile: options.StatePath, preferredWidth: loadIndexWidth(options.StatePath), now: now}
	m.configureImages(options)
	return m
}

func (m *Model) mainWide() bool { return m.wide() && !m.picker && !m.help && !m.input && !m.confirm }
func (m *Model) busy() bool     { return m.loading || m.saving }
func (m *Model) indexWidth() int {
	width := m.preferredWidth
	if width == 0 {
		width = m.width * 3 / 10
	}
	return min(max(MinimumIndexWidth, width), m.width-MinimumPreviewWidth-1)
}
func (m *Model) resizeIndex(width int) {
	m.preferredWidth = min(max(MinimumIndexWidth, width), m.width-MinimumPreviewWidth-1)
	m.scroll = min(m.scroll, m.maxScroll())
}
func (m *Model) mastheadRows() []string {
	date := m.now().Format("2 January 2006")
	title := strings.ToUpper(singleLine(m.feedTitle(m.filter.FeedID)))
	title = fit(title, max(1, m.width-ansi.StringWidth(date)-1))
	row := padRow(title, m.width-ansi.StringWidth(date)) + date
	rows := []string{m.styles.Unread.Render(strings.Repeat("═", m.width)), m.styles.Focus.Render(row), m.styles.Unread.Render(strings.Repeat("═", m.width))}
	focus := "Index"
	if m.reader != nil {
		focus = "Preview"
	}
	status := "Focus: " + focus
	if m.filter.UnreadOnly {
		status += " | Unread only"
	}
	if operation := m.operationRow(); operation != "" {
		status += " | " + operation
	}
	if m.height <= 4 {
		if m.height == 1 {
			return nil
		}
		available := m.width - ansi.StringWidth(date) - 1
		compactTitle := fit(title, max(1, available-ansi.StringWidth(status)-3))
		compact := fit(compactTitle+" | "+status, available)
		compact = m.styles.Focus.Render(padRow(compact, m.width-ansi.StringWidth(date)) + date)
		if m.height == 2 {
			return []string{compact}
		}
		if m.height == 3 {
			return []string{compact, rows[2]}
		}
		rows[1] = compact
		return rows
	}

	return append(rows, status)
}
func (m *Model) previewArticle() *domain.Article {
	if m.reader != nil {
		return m.reader
	}
	items := m.visible()
	if m.articleIndex < 0 || m.articleIndex >= len(items) {
		return nil
	}
	article := items[m.articleIndex]
	return &article
}
func (m *Model) previewLines() []string { lines, _ := m.previewLayout(); return lines }
func (m *Model) previewLayout() ([]string, []imageSpan) {
	article := m.previewArticle()
	if article == nil {
		return []string{"No article matches"}, nil
	}
	panelWidth := max(1, m.width-m.indexWidth()-1)
	width := max(1, min(NewspaperColumnWidth, panelWidth-4))
	indent := strings.Repeat(" ", (panelWidth-width)/2)
	lines := []string{}
	var spans []imageSpan
	centered := func(text string, render func(...string) string) {
		for _, line := range wrapWords(text, width) {
			lines = append(lines, strings.Repeat(" ", (panelWidth-width)/2+(width-ansi.StringWidth(line))/2)+render(line))
		}
	}
	centered(strings.ToUpper(app.SafeText(article.Title)), m.styles.Emphasis.Render)
	lines = append(lines, m.styles.Unread.Render(strings.Repeat("─", panelWidth)))
	lead, leadSpans := m.leadImageLayout(article, width)
	appendImageLayout(&lines, &spans, lead, leadSpans, indent)
	centered(app.ArticleSubtitle(article.Subtitle, article.Content), m.styles.Read.Render)
	centered(app.ArticleByline(article.Author), m.styles.Read.Render)
	lines = append(lines, "")
	body, bodySpans := m.articleBodyLayout(article, width)
	appendImageLayout(&lines, &spans, body, bodySpans, indent)
	return lines, spans
}
func (m *Model) indexRows() []string {
	items := m.visible()
	if len(items) == 0 {
		return []string{"No articles"}
	}
	rows := []string{}
	selectedStart, selectedEnd := 0, 0
	for i, article := range items {
		start := len(rows)
		prefix, state := "  ", "* "
		if i == m.articleIndex {
			prefix = "> "
		}
		if article.Read {
			state = "  "
		}
		headline := wrapWords(singleLine(article.Title), m.indexWidth()-4)
		if len(headline) == 0 {
			headline = []string{""}
		}
		for j, line := range headline[:min(2, len(headline))] {
			marker := "    "
			if j == 0 {
				marker = prefix + state
			}
			row := marker + line
			if i == m.articleIndex {
				row = m.styles.Selected.Render(row)
			} else if article.Read {
				row = m.styles.Read.Render(row)
			} else if j == 0 {
				row = prefix + m.styles.Unread.Render(state) + line
			}
			rows = append(rows, row)
		}
		date := ""
		if !article.PublishedAt.IsZero() {
			date = article.PublishedAt.Format("2006-01-02")
		}
		rows = append(rows, m.styles.Read.Render("    "+date))
		if i == m.articleIndex {
			selectedStart, selectedEnd = start, len(rows)
		}
	}
	height := m.listHeight()
	start := max(0, selectedEnd-height)
	start = min(start, selectedStart)
	return rows[start:min(len(rows), start+height)]
}

func (m *Model) queueWidthSave() tea.Cmd {
	m.pendingWidth = m.preferredWidth
	return m.startWidthSave()
}
func (m *Model) startWidthSave() tea.Cmd {
	if m.busy() || m.pendingWidth == 0 {
		return nil
	}
	width, path := m.pendingWidth, m.stateFile
	m.pendingWidth = 0
	if path == "" {
		return nil
	}
	m.saving = true
	return func() tea.Msg { return widthSavedMsg{saveIndexWidth(path, width)} }
}
