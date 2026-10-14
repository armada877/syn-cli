package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestWidePreviewFocusAndWrites(t *testing.T) {
	m, r, _ := fixture(t)
	resize(m, 120)
	initialCalls := r.calls
	if m.reader != nil || !strings.Contains(ansi.Strip(m.View().Content), "NEWEST") {
		t.Fatal("missing selection preview")
	}
	press(m, "j")
	if m.selected != 10 || r.readCalls != 0 || r.calls != initialCalls || !strings.Contains(ansi.Strip(m.View().Content), "OLDER") {
		t.Fatal("selection wrote or failed to update preview")
	}
	press(m, "tab")
	if m.reader != nil {
		t.Fatal("index Tab opened reader")
	}
	finish(m, press(m, "m"))
	if r.readCalls != 1 || !m.visible()[m.articleIndex].Read || m.reader != nil {
		t.Fatal("index read toggle")
	}
	finish(m, press(m, "m"))
	press(m, "k")
	press(m, "u")
	finish(m, press(m, "enter"))
	if m.reader == nil || m.reader.ID != 20 || !m.reader.Read || m.selected != 10 || r.readCalls != 3 {
		t.Fatal("unread removal lost reader")
	}
	press(m, "pgdown")
	if m.scroll == 0 {
		t.Fatal("preview did not scroll")
	}
	savedScroll := m.scroll
	press(m, "f")
	if !m.picker || m.View().MouseMode != tea.MouseModeNone {
		t.Fatal("focused picker")
	}
	press(m, "j")
	press(m, "esc")
	if m.reader == nil || m.reader.ID != 20 || m.scroll != savedScroll || m.filter.FeedID != 0 {
		t.Fatal("picker cancel changed reading")
	}
	resize(m, 80)
	if m.reader == nil || m.reader.ID != 20 {
		t.Fatal("narrow transition lost reader")
	}
	resize(m, 81)
	if m.reader == nil || m.reader.ID != 20 {
		t.Fatal("wide transition lost reader")
	}
	finish(m, press(m, "m"))
	if m.reader.Read || r.readCalls != 4 {
		t.Fatal("focused toggle")
	}
	press(m, "tab")
	if m.reader != nil || m.scroll != 0 {
		t.Fatal("focus return")
	}
	resize(m, 80)
	if m.reader != nil || r.readCalls != 4 {
		t.Fatal("unfocused resize opened reader")
	}
}

func TestWidePickerManagement(t *testing.T) {
	m, r, f := fixture(t)
	resize(m, 120)
	for _, key := range []string{"a", "d", "r"} {
		if press(m, key) != nil || m.input || m.confirm {
			t.Fatal("direct feed management remains")
		}
	}
	press(m, "f")
	press(m, "j")
	press(m, "enter")
	if m.picker || m.filter.FeedID != 1 || m.selected != 10 {
		t.Fatal("wide picker filter")
	}
	finish(m, press(m, "enter"))
	press(m, "f")
	press(m, "k")
	press(m, "enter")
	if m.reader != nil || m.filter.FeedID != 0 {
		t.Fatal("picker did not return to index")
	}
	press(m, "f")
	press(m, "j")
	finish(m, press(m, "r"))
	if f.calls != 1 {
		t.Fatal("selected feed refresh")
	}
	press(m, "a")
	m.url = "https://example.test/feed"
	finish(m, press(m, "enter"))
	if len(r.feeds) != 3 || !m.picker {
		t.Fatal("picker add")
	}
	press(m, "d")
	press(m, "n")
	if m.confirm || len(r.feeds) != 3 {
		t.Fatal("remove cancel")
	}
	press(m, "d")
	finish(m, press(m, "enter"))
	if len(r.feeds) != 2 || m.confirm {
		t.Fatal("confirmed remove")
	}
}

func TestDividerMouseAndModalGuards(t *testing.T) {
	m, _, _ := fixture(t)
	resize(m, 100)
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Fatal("mouse mode")
	}
	width, y := m.indexWidth(), len(m.headerRows())
	for _, click := range []tea.MouseClickMsg{{X: width - 1, Y: y, Button: tea.MouseLeft}, {X: width, Y: y - 1, Button: tea.MouseLeft}, {X: width, Y: m.height - 1, Button: tea.MouseLeft}, {X: width, Y: y, Button: tea.MouseRight}} {
		m.Update(click)
		if m.dragging {
			t.Fatal("unrelated click started drag")
		}
	}
	m.Update(tea.MouseMotionMsg{X: 50, Y: y, Button: tea.MouseLeft})
	if m.indexWidth() != width {
		t.Fatal("unrelated motion resized")
	}
	m.Update(tea.MouseClickMsg{X: width, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: 50, Y: y, Button: tea.MouseRight})
	if m.indexWidth() != width {
		t.Fatal("wrong button resized")
	}
	m.Update(tea.MouseMotionMsg{X: 500, Y: y, Button: tea.MouseLeft})
	if !m.dragging || m.indexWidth() != 59 {
		t.Fatal("drag maximum")
	}
	m.Update(tea.MouseMotionMsg{X: -20, Y: y, Button: tea.MouseLeft})
	if m.indexWidth() != MinimumIndexWidth {
		t.Fatal("drag minimum")
	}
	_, cmd := m.Update(tea.MouseReleaseMsg{X: 5, Y: -1, Button: tea.MouseLeft})
	finish(m, cmd)
	if m.dragging || loadIndexWidth(m.stateFile) != MinimumIndexWidth {
		t.Fatal("release outside divider")
	}
	for _, modal := range []string{"picker", "help", "input", "confirm"} {
		m.picker, m.help, m.input, m.confirm = false, false, false, false
		switch modal {
		case "picker":
			m.picker = true
		case "help":
			m.help = true
		case "input":
			m.input = true
		case "confirm":
			m.confirm = true
		}
		before := m.indexWidth()
		press(m, ">")
		m.Update(tea.MouseClickMsg{X: before, Y: y, Button: tea.MouseLeft})
		m.Update(tea.MouseMotionMsg{X: 55, Y: y, Button: tea.MouseLeft})
		if m.indexWidth() != before || m.dragging || m.View().MouseMode != tea.MouseModeNone {
			t.Fatal("modal divider active", modal)
		}
	}
	m.picker, m.help, m.input, m.confirm = false, false, false, false
	m.Update(tea.MouseClickMsg{X: m.indexWidth(), Y: y, Button: tea.MouseLeft})
	resize(m, 80)
	if m.dragging || m.View().MouseMode != tea.MouseModeNone {
		t.Fatal("resize retained drag")
	}
	resize(m, 100)
	m.Update(tea.MouseClickMsg{X: m.indexWidth(), Y: len(m.headerRows()), Button: tea.MouseLeft})
	press(m, "?")
	if m.dragging {
		t.Fatal("help retained drag")
	}
}

func TestWidthStateAndSerialization(t *testing.T) {
	m, r, _ := fixture(t)
	resize(m, 120)
	before := m.indexWidth()
	first := press(m, ">")
	if first == nil || !m.saving || m.indexWidth() != before+1 {
		t.Fatal("keyboard save")
	}
	if press(m, "m") != nil || r.readCalls != 0 {
		t.Fatal("save collided with read")
	}
	press(m, ">")
	press(m, ">")
	press(m, "j")
	if m.selected != 10 {
		t.Fatal("save blocked navigation")
	}
	_, next := m.Update(first())
	if next == nil || loadIndexWidth(m.stateFile) != before+1 {
		t.Fatal("save queue")
	}
	finish(m, next)
	if loadIndexWidth(m.stateFile) != before+3 || m.saving {
		t.Fatal("latest width lost")
	}
	other := NewWithOptions(context.Background(), nil, Options{ThemePath: m.themeFile, StatePath: m.stateFile, Now: m.now})
	resize(other, 120)
	if other.indexWidth() != before+3 {
		t.Fatal("saved width not loaded")
	}
	other.preferredWidth = 200
	resize(other, 81)
	if other.indexWidth() != 40 || other.preferredWidth != 200 {
		t.Fatal("resize lost preference")
	}
	resize(other, 300)
	if other.indexWidth() != 200 {
		t.Fatal("large window did not restore preference")
	}
	other.loading = false
	for range 250 {
		finish(other, press(other, "<"))
	}
	if other.indexWidth() != 20 {
		t.Fatal("keyboard minimum")
	}
	resize(other, 81)
	for range 80 {
		finish(other, press(other, ">"))
	}
	if other.indexWidth() != 40 {
		t.Fatal("keyboard maximum")
	}
	m.stateFile = filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(m.stateFile, 0700); err != nil {
		t.Fatal(err)
	}
	finish(m, press(m, ">"))
	if m.err == nil || m.saving || !strings.Contains(m.operationRow(), "Error:") {
		t.Fatal("save error not reported")
	}
	press(m, "k")
	if m.selected != 20 {
		t.Fatal("save error blocked navigation")
	}
	m.loading = true
	if press(m, ">") != nil || m.pendingWidth == 0 {
		t.Fatal("loading did not queue save")
	}
	_, cmd := m.Update(dataMsg{feeds: r.feeds, articles: r.articles})
	if cmd == nil || !m.saving {
		t.Fatal("data completion did not start queued save")
	}
	finish(m, cmd)
}

func TestStatePathsAndInvalidFiles(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	if statePath(xdg, home) != filepath.Join(xdg, "syn", "front-page.json") || statePath("", home) != filepath.Join(home, ".local", "state", "syn", "front-page.json") || statePath("", "") != "" {
		t.Fatal("state path resolution")
	}
	path := filepath.Join(t.TempDir(), "state", "front-page.json")
	if loadIndexWidth(path) != 0 {
		t.Fatal("missing state")
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("load created directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{"", "invalid", "{}", `{"index_width":0}`, `{"index_width":-1}`, `{"index_width":1.5}`, `{"index_width":"30"}`, `{"index_width":9999999999999999999999}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if loadIndexWidth(path) != 0 {
			t.Fatal("invalid state accepted", data)
		}
	}
	if err := saveIndexWidth(path, 200); err != nil {
		t.Fatal(err)
	}
	m := NewWithOptions(context.Background(), nil, Options{ThemePath: filepath.Join(t.TempDir(), "theme"), StatePath: path, Now: func() time.Time { return time.Date(2026, 10, 13, 23, 0, 0, 0, time.FixedZone("local", -4*60*60)) }})
	resize(m, 81)
	if m.indexWidth() != 40 || m.preferredWidth != 200 || !strings.Contains(ansi.Strip(m.View().Content), "13 October 2026") {
		t.Fatal("width or date injection")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if loadIndexWidth(path) != 0 {
		t.Fatal("unreadable state")
	}
}

func TestIndexWrappingAndPreviewColumn(t *testing.T) {
	m, _, _ := fixture(t)
	resize(m, 120)
	m.preferredWidth = 24
	m.articles[0].Title = strings.Repeat("界e\u0301👩🏽‍💻 ", 12) + "\x1b]0;attack\a"
	m.articles[0].PublishedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	m.articles[0].Content = "<p>" + strings.Repeat("aa bb cc ", 30) + "final</p><p>last words</p>"
	rows := m.indexRows()
	if len(rows) != 5 || !strings.Contains(ansi.Strip(rows[2]), "2026-01-02") {
		t.Fatal("index entry rows", rows)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 24 || strings.Contains(row, "attack") {
			t.Fatal(row)
		}
	}
	lines := m.previewLines()
	panel := m.width - m.indexWidth() - 1
	column := min(60, panel-4)
	indent := strings.Repeat(" ", (panel-column)/2)
	if strings.Contains(strings.Join(lines, "\n"), "ALL FEEDS") {
		t.Fatal("repeated masthead")
	}
	last := lines[len(lines)-1]
	if last != indent+"last words" || strings.TrimSpace(lines[len(lines)-2]) != "" {
		t.Fatal("final paragraph layout", lines)
	}
	justified := justifyText(strings.Repeat("aa bb cc ", 30)+"final", column)
	for _, line := range lines[len(lines)-len(justified)-2 : len(lines)-2] {
		if !strings.HasPrefix(line, indent) || ansi.StringWidth(line) > panel {
			t.Fatal("body column", line)
		}
	}
	press(m, "j")
	if m.selected != 10 || !strings.Contains(ansi.Strip(strings.Join(m.indexRows(), "\n")), "> ") {
		t.Fatal("wrapped selection visibility")
	}
	m.articles = nil
	if strings.Join(m.previewLines(), "") != "No article matches" {
		t.Fatal("empty preview")
	}
}

func TestShortMastheadStatusAndThemeState(t *testing.T) {
	m, _, _ := fixture(t)
	resize(m, 81)
	m.feeds[0].Title = strings.Repeat("Long title ", 30) + "\x1b]0;attack\a"
	m.filter.FeedID = 1
	m.reconcile()
	m.err = errors.New("Save failed")
	for _, height := range []int{2, 3, 4, 5, 10} {
		m.Update(tea.WindowSizeMsg{Width: 81, Height: height})
		rows := frameRows(t, m)
		text := ansi.Strip(strings.Join(rows, "\n"))
		if !strings.Contains(text, "Save failed") || !strings.Contains(text, "2 January 2026") || strings.Contains(text, "attack") {
			t.Fatal("short masthead lost status or date", height, text)
		}
	}
	finish(m, press(m, "enter"))
	m.preferredWidth = 35
	press(m, "j")
	beforeReader, beforeScroll, beforeSelected := m.reader, m.scroll, m.selected
	m.Update(themeMsg{Theme{NoColor: true}})
	if m.preferredWidth != 35 || m.reader != beforeReader || m.scroll != beforeScroll || m.selected != beforeSelected || m.filter.FeedID != 1 {
		t.Fatal("theme changed front page state")
	}
	if strings.Contains(m.View().Content, "\x1b[") {
		t.Fatal("NO_COLOR styles")
	}
}

func TestFeedPickerAccessAcrossLayouts(t *testing.T) {
	for _, width := range []int{79, 80, 81, 120} {
		m, r, _ := fixture(t)
		resize(m, width)
		press(m, "f")
		if !m.picker {
			t.Fatal("main layout did not open picker", width)
		}
		press(m, "esc")
		finish(m, press(m, "enter"))
		press(m, "j")
		reader, scroll, selected, filter := m.reader, m.scroll, m.selected, m.filter
		calls, readCalls := r.calls, r.readCalls
		before := m.View().Content
		cmd := press(m, "f")
		if cmd != nil || m.picker != m.wide() {
			t.Fatal("reader picker access", width)
		}
		if !m.wide() && m.View().Content != before {
			t.Fatal("narrow reader f changed view", width)
		}
		if m.wide() {
			press(m, "esc")
		}
		if m.reader != reader || m.scroll != scroll || m.selected != selected || m.filter != filter || r.calls != calls || r.readCalls != readCalls {
			t.Fatal("picker key changed reader state", width)
		}
	}
}
