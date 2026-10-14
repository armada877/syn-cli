package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/charmbracelet/x/ansi"
)

func frameRows(t *testing.T, m *Model) []string {
	t.Helper()
	rows := strings.Split(m.View().Content, "\n")
	if len(rows) != m.height {
		t.Fatalf("Rows: %d, want %d", len(rows), m.height)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) != m.width {
			t.Fatalf("Width: %d, want %d: %q", ansi.StringWidth(row), m.width, row)
		}
		if strings.ContainsAny(ansi.Strip(row), "\x1b\a\u202e\t") {
			t.Fatalf("Unsafe row: %q", row)
		}
	}
	return rows
}

func TestEveryViewFillsFrame(t *testing.T) {
	modes := []string{"list", "picker", "reader", "help", "input", "confirmation"}
	for _, width := range []int{1, 2, 10, 40, 80, 81, 120} {
		for _, height := range []int{1, 2, 3, 4, 10, 24} {
			for _, mode := range modes {
				for _, status := range []string{"none", "loading", "error", "status"} {
					t.Run(fmt.Sprintf("%d/%d/%s/%s", width, height, mode, status), func(t *testing.T) {
						m, _, _ := fixture(t)
						m.Update(tea.WindowSizeMsg{Width: width, Height: height})
						m.feeds[0].Title = "Feed\x1b]0;attack\a\nName\u202e"
						m.articles[0].Title = "Title\x1b[31m\tSafe"
						bar := "f: feeds | Enter: read | u: unread | ?: help"
						if m.wide() {
							bar = "Tab: focus | Enter: read | a/d/r: manage | ?: help"
						}
						switch mode {
						case "picker":
							m.picker = true
							bar = "Enter: filter | Esc: cancel | a/d/r: manage"
						case "reader":
							article := m.articles[0]
							m.reader = &article
							bar = "Esc: return | arrows/PageUp/PageDown: scroll"
						case "help":
							m.help = true
							bar = "Esc/?: return | q/Ctrl+C: quit"
						case "input":
							m.input = true
							m.url = "URL\n\t\x1b]0;attack\a"
							bar = "Enter: add | Esc: cancel"
						case "confirmation":
							m.confirm = true
							m.feedIndex = 1
							bar = "y/Enter: remove | n/Esc: cancel"
						}
						switch status {
						case "loading":
							m.loading = true
						case "error":
							m.err = errors.New("failure\n\x1b]0;attack\a")
						case "status":
							m.status = "Operation\ncomplete"
						}
						rows := frameRows(t, m)
						if rows[len(rows)-1] != padRow(bar, width) {
							t.Fatal("Key bar is not on final row")
						}
						if strings.Contains(m.View().Content, "attack") {
							t.Fatal("Source controls survived")
						}
						m.Update(themeMsg{Theme{NoColor: true}})
						frameRows(t, m)
						if strings.Contains(m.View().Content, "\x1b[") {
							t.Fatal("NO_COLOR generated styles")
						}
					})
				}
			}
		}
	}
}

func TestListUsesAvailableRows(t *testing.T) {
	m, _, _ := fixture(t)
	m.articles = nil
	m.feeds = nil
	for i := int64(1); i <= 40; i++ {
		m.feeds = append(m.feeds, domain.Feed{ID: i, Title: fmt.Sprintf("Feed %d", i)})
		m.articles = append(m.articles, domain.Article{ID: i, FeedID: i, Title: fmt.Sprintf("Article %d", i)})
	}
	m.articleIndex, m.feedIndex = 39, 40
	m.selected = 40
	for _, width := range []int{80, 81, 120, 80, 81} {
		for _, height := range []int{3, 4, 8, 24, 5} {
			for _, status := range []string{"", "Saved"} {
				m.status = status
				m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				for _, picker := range []bool{false, true} {
					m.picker = picker
					rows := frameRows(t, m)
					available := m.listHeight()
					if len(m.feedRows()) != available || len(m.articleRows()) != available {
						t.Fatal("List does not use available rows")
					}
					body := rows[len(m.headerRows()) : len(rows)-1]
					text := ansi.Strip(strings.Join(body, "\n"))
					if !strings.Contains(text, "40") || !strings.Contains(text, "> ") {
						t.Fatal("Selection is not visible", text)
					}
					if !picker && m.wide() {
						for _, row := range body {
							if ansi.Cut(ansi.Strip(row), (width-1)/3, (width-1)/3+1) != "|" {
								t.Fatal("Missing divider", row)
							}
						}
					}
				}
			}
		}
	}
}

func TestShortListsPadWithoutRepeatedEntries(t *testing.T) {
	for _, width := range []int{80, 81, 120} {
		m, _, _ := fixture(t)
		m.Update(tea.WindowSizeMsg{Width: width, Height: 12})
		rows := frameRows(t, m)
		body := rows[len(m.headerRows()) : len(rows)-1]
		text := ansi.Strip(strings.Join(body, "\n"))
		if strings.Count(text, "Newest") != 1 || strings.Count(text, "Older") != 1 {
			t.Fatal(text)
		}
		for _, row := range body[3:] {
			want := ""
			if m.wide() {
				want = "|"
			}
			if strings.TrimSpace(ansi.Strip(row)) != want {
				t.Fatal("Padding contains list entries", row)
			}
		}
	}
}

func TestBoundaryControlsAndResize(t *testing.T) {
	m, _, _ := fixture(t)
	m.filter.FeedID, m.selected = 1, 10
	m.reconcile()
	for _, width := range []int{80, 81, 80, 120, 80} {
		resize(m, width)
		m.picker = false
		before := m.focusFeeds
		press(m, "tab")
		if m.wide() != (width >= 81) || (m.focusFeeds != before) != (width >= 81) {
			t.Fatal(width, m.focusFeeds)
		}
		rows := frameRows(t, m)
		if width <= 80 && strings.Contains(ansi.Strip(rows[len(m.headerRows())]), "All feeds") {
			t.Fatal("Narrow stream has a feed pane")
		}
		if m.filter.FeedID != 1 || m.selected != 10 {
			t.Fatal("Resize changed selection")
		}
	}
}

func TestFullWidthRulesAndCenteredBody(t *testing.T) {
	m, _, _ := fixture(t)
	article := m.articles[0]
	article.Title, article.Content = "Headline", "<p>"+strings.Repeat("word ", 30)+"</p>"
	m.reader = &article
	for _, width := range []int{1, 2, 40, 80, 81, 120} {
		resize(m, width)
		lines := m.readerLines()
		for _, line := range lines {
			plain := ansi.Strip(line)
			if strings.ContainsAny(plain, "═─") {
				if ansi.StringWidth(plain) != width || strings.HasPrefix(plain, " ") {
					t.Fatal(width, plain)
				}
			}
		}
		column := max(1, min(60, width-4))
		for _, line := range lines[len(lines)-len(justifyText("word "+strings.Repeat("word ", 29), column)):] {
			if ansi.StringWidth(strings.TrimSpace(line)) > column {
				t.Fatal(width, line)
			}
			if !strings.HasPrefix(line, strings.Repeat(" ", (width-column)/2)) {
				t.Fatal("Body margin changed", line)
			}
		}
		frameRows(t, m)
	}
}

func TestFramePreservesStyles(t *testing.T) {
	m, _, _ := fixture(t)
	m.articles[0].Title = strings.Repeat("Safe\x1b]0;attack\a title ", 40)
	for _, width := range []int{40, 80, 81, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 10})
		rows := frameRows(t, m)
		if !strings.Contains(rows[0], "\x1b[") || !strings.Contains(rows[len(m.headerRows())], "\x1b[") {
			t.Fatal("Frame lost generated styles")
		}
		if strings.Contains(m.View().Content, "attack") {
			t.Fatal("Frame retained source controls")
		}
	}
}
