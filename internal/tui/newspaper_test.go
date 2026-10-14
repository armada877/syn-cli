package tui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestJustification(t *testing.T) {
	got := justifyText("one two three four five\n\nlast words", 13)
	want := []string{"one two three", "four five", "", "last words"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatal(got)
	}
	got = justifyText("aa bb cc dd ee", 10)
	if strings.Join(got, "\n") != "aa  bb  cc\ndd ee" {
		t.Fatal(got)
	}
	if got := strings.Join(justifyText("aa bb ccc ddd", 12), "\n"); got != "aa   bb  ccc\nddd" {
		t.Fatal(got)
	}
	for _, width := range []int{1, 2, 3, 7, 60} {
		for _, text := range []string{"abcdefghijk next", "界界界 next", "e\u0301e\u0301e\u0301 next", "👩🏽‍💻👩🏽‍💻 next", "", "one"} {
			for _, line := range justifyText(text, width) {
				if !utf8.ValidString(line) || ansi.StringWidth(line) > width || strings.HasPrefix(line, "\u0301") {
					t.Fatal(width, line)
				}
			}
		}
	}
	if got := strings.Join(justifyText("界", 1), ""); got != "" {
		t.Fatal(got)
	}
	if got := strings.Join(justifyText("e\u0301e\u0301", 1), "\n"); got != "e\u0301\ne\u0301" {
		t.Fatal(got)
	}
}

func TestNewspaperLayout(t *testing.T) {
	m, _, _ := fixture(t)
	m.noColor = true
	m.styles = makeStyles(Theme{NoColor: true})
	article := m.articles[0]
	article.Title, article.Subtitle, article.Author = "Headline", "Summary", "Alice"
	article.Content = "<p>Body</p>"
	article.PublishedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	m.reader = &article
	m.filter.FeedID = 1
	lines := m.readerLines()
	want := []string{strings.Repeat("═", 80), "TWO", "2 January 2026", strings.Repeat("═", 80), "HEADLINE", strings.Repeat("─", 80), "Summary", "By Alice", "", "Body"}
	if len(lines) != len(want) {
		t.Fatal(lines)
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != want[i] {
			t.Fatal(i, line)
		}
		if want[i] != "" {
			left := len(line) - len(strings.TrimLeft(line, " "))
			expected := 10
			if i == 0 || i == 3 || i == 5 {
				expected = 0
			} else if i != 9 {
				expected += (60 - ansi.StringWidth(want[i])) / 2
			}
			if left != expected {
				t.Fatal(i, left, expected)
			}
		}
	}
	if strings.Contains(m.View().Content, "syn |") {
		t.Fatal(m.View().Content)
	}
	article.Author, article.Subtitle = "", "<p>Body</p>"
	article.PublishedAt = time.Time{}
	text := strings.Join(m.readerLines(), "\n")
	if strings.Contains(text, "By ") || strings.Contains(text, "January") || strings.Count(text, "Body") != 1 {
		t.Fatal(text)
	}
	m.feeds[1].Title = strings.Repeat("Long feed 界e\u0301 ", 20)
	article.Title = strings.Repeat("Long headline 界e\u0301 ", 20)
	article.Author = "\x1b]0;attack\aAlice"
	article.Subtitle = "safe\x1b[31m subtitle\u202e"
	for _, width := range []int{1, 2, 3, 4, 10, 40, 79, 80, 81, 120} {
		resize(m, width)
		for _, line := range m.readerLines() {
			if ansi.StringWidth(line) > width || strings.ContainsAny(line, "\x1b\u202e") || strings.Contains(line, "attack") {
				t.Fatal(width, line)
			}
		}
	}
}

func TestReaderViewport(t *testing.T) {
	m, _, _ := fixture(t)
	article := m.articles[0]
	m.reader = &article
	resize(m, 40)
	press(m, "pgdown")
	if m.scroll != m.readerHeight() {
		t.Fatal(m.scroll)
	}
	for range 100 {
		press(m, "j")
	}
	if m.scroll != m.maxScroll() {
		t.Fatal(m.scroll)
	}
	m.status = "Read state saved"
	view := strings.Split(m.View().Content, "\n")
	if !strings.Contains(view[len(view)-1], "Esc: return") {
		t.Fatal(view)
	}
	resize(m, 120)
	if m.scroll > m.maxScroll() || m.reader.ID != article.ID {
		t.Fatal(m.scroll)
	}
}

func TestNewspaperStyles(t *testing.T) {
	m, _, _ := fixture(t)
	article := m.articles[0]
	article.Title, article.Subtitle, article.Author, article.Content = "Headline", "Summary", "Alice", "Body"
	m.reader = &article
	m.Update(themeMsg{Theme{Accent: lipgloss.Color("#abc"), Muted: lipgloss.Color("#123"), Emphasis: lipgloss.Color("#fed")}})
	lines := m.readerLines()
	for _, check := range []struct {
		index  int
		styled string
	}{
		{0, m.styles.Unread.Render(strings.Repeat("═", 80))},
		{1, m.styles.Unread.Render("TWO")},
		{3, m.styles.Emphasis.Render("HEADLINE")},
		{4, m.styles.Unread.Render(strings.Repeat("─", 80))},
		{5, m.styles.Read.Render("Summary")},
		{6, m.styles.Read.Render("By Alice")},
	} {
		if !strings.HasSuffix(lines[check.index], check.styled) || !strings.Contains(lines[check.index], "\x1b[") {
			t.Fatal(check.index, lines[check.index])
		}
	}
	if strings.Contains(lines[len(lines)-1], "\x1b[") {
		t.Fatal("Body has generated styles")
	}
	m.Update(themeMsg{Theme{NoColor: true}})
	if strings.Contains(strings.Join(m.readerLines(), "\n"), "\x1b[") {
		t.Fatal("NO_COLOR has generated styles")
	}
}
