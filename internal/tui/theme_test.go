package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func writeTheme(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestThemeOverlays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme", "syn.toml")
	base := loadTheme(path, false)
	if base.Accent != lipgloss.BrightBlue || base.Error != lipgloss.Red || base.Muted != nil || base.Selection != nil {
		t.Fatal(base)
	}
	cases := []struct {
		data                               string
		accent, muted, emphasis, selection string
	}{
		{`accent = '#abc' # color`, "#abc", "", "", ""},
		{`accent = "{{ accent }}"`, "", "", "", ""},
		{`accent = "#ggg"`, "", "", "", ""},
		{"accent = \"#fff\"\nforeground = '#fff'\nmuted = '#123'", "", "#123", "#fff", ""},
		{"accent = '#222'\nbackground = '#000'\nbright_blue = '#fff'", "", "", "", ""},
		{"accent = '#fff'\nbackground = '#000'\nselection = '#000'\nforeground = '#123'\nbright_foreground = '#456'", "#fff", "", "#456", "#000"},
		{"accent = '#fff'\nselection = '#eee'", "#fff", "", "", ""},
		{"accent = '#888'\nbackground = '#000'\nbright_blue = '#777'", "#888", "", "", ""},
		{"accent = '#888'\nbackground = '#000'\nblue = '#777'", "#888", "", "", ""},
	}
	for _, tc := range cases {
		writeTheme(t, path, tc.data)
		got := loadTheme(path, false)
		want := base
		if tc.accent != "" {
			want.Accent = lipgloss.Color(tc.accent)
		}
		if tc.muted != "" {
			want.Muted = lipgloss.Color(tc.muted)
		}
		if tc.emphasis != "" {
			want.Emphasis = lipgloss.Color(tc.emphasis)
		}
		if tc.selection != "" {
			want.Selection = lipgloss.Color(tc.selection)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: %+v != %+v", tc.data, got, want)
		}
	}
	writeTheme(t, path, "error = '#f00'\nmode = 'light'\nunknown = '#123'")
	got := loadTheme(path, false)
	if got.Error != lipgloss.Color("#f00") || got.Mode != "light" || got.Accent != base.Accent {
		t.Fatal(got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadTheme(path, false), base) {
		t.Fatal("Removal did not restore defaults")
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadTheme(path, false), base) {
		t.Fatal("Unreadable file did not restore defaults")
	}
}

func TestSemanticStyles(t *testing.T) {
	m, _, _ := fixture(t)
	if m.styles.Selected.GetForeground() != lipgloss.BrightBlue || !m.styles.Selected.GetBold() || !m.styles.Focus.GetBold() || !m.styles.Read.GetFaint() || m.styles.Error.GetForeground() != lipgloss.Red {
		t.Fatal(m.styles)
	}
	m.articles[1].Read = true
	rows := m.articleRows()
	if !strings.Contains(rows[0], m.styles.Selected.Render("> * ")) && !strings.Contains(rows[0], "\x1b[") {
		t.Fatal(rows)
	}
	if !strings.Contains(rows[1], "\x1b[2m") {
		t.Fatal(rows[1])
	}
	m.articleIndex = 1
	if strings.Contains(m.articleRows()[1], "\x1b[2m") {
		t.Fatal("Selection used faint text")
	}
	m.articleIndex = -1
	if !strings.Contains(m.articleRows()[0], m.styles.Unread.Render("* ")) {
		t.Fatal("Unread marker has no accent")
	}
	m.err = errors.New("bad\x1b]0;attack\a")
	if !strings.Contains(m.View().Content, m.styles.Error.Render("Error: bad")) {
		t.Fatal(m.View())
	}
	path := filepath.Join(t.TempDir(), "syn.toml")
	writeTheme(t, path, "muted = '#123'\naccent = '#abc'")
	theme := loadTheme(path, false)
	if makeStyles(theme).Read.GetForeground() != lipgloss.Color("#123") || makeStyles(theme).Read.GetFaint() {
		t.Fatal("Muted overlay")
	}
	m.theme = loadTheme(path, true)
	m.styles = makeStyles(m.theme)
	if strings.Contains(m.View().Content, "\x1b") {
		t.Fatal("NO_COLOR emitted styles")
	}
}

func startWatch(t *testing.T, ctx context.Context, path string) <-chan any {
	t.Helper()
	result := make(chan any, 1)
	go func() { result <- watchTheme(ctx, path, false)() }()
	time.Sleep(40 * time.Millisecond)
	return result
}
func receiveTheme(t *testing.T, result <-chan any) Theme {
	t.Helper()
	select {
	case msg := <-result:
		got, ok := msg.(themeMsg)
		if !ok {
			t.Fatalf("Unexpected message: %T", msg)
		}
		return got.theme
	case <-time.After(3 * time.Second):
		t.Fatal("Theme watcher timed out")
	}
	return Theme{}
}

func TestLiveThemeSmoke(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "current", "theme", "syn.toml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, r, f := fixture(t)
	m.ctx = ctx
	m.themeFile = path
	reader := m.articles[0]
	m.reader = &reader
	m.scroll = 4
	m.picker = true
	m.filter.UnreadOnly = true
	before := *m
	beforeView := m.View().Content
	calls := r.calls
	result := startWatch(t, ctx, path)
	writeTheme(t, path, "accent = '#abc'")
	theme := receiveTheme(t, result)
	_, cmd := m.Update(themeMsg{theme})
	if m.View().Content == beforeView {
		t.Fatal("Theme did not redraw the open reader")
	}
	if m.theme.Accent != lipgloss.Color("#abc") || cmd == nil {
		t.Fatal(m.theme)
	}
	if m.filter != before.filter || m.selected != before.selected || m.picker != before.picker || m.reader != before.reader || m.scroll != before.scroll {
		t.Fatal("Theme changed reader state")
	}
	if r.calls != calls || r.readCalls != 0 || f.calls != 0 {
		t.Fatal("Theme accessed data")
	}
	result = startWatch(t, ctx, path)
	writeTheme(t, path, "accent = '#fed'")
	m.Update(themeMsg{receiveTheme(t, result)})
	if m.theme.Accent != lipgloss.Color("#fed") {
		t.Fatal("In-place write")
	}
	result = startWatch(t, ctx, path)
	next := filepath.Join(root, "next", "syn.toml")
	writeTheme(t, next, "accent = '#123'")
	if err := os.Rename(filepath.Dir(path), filepath.Join(root, "old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Dir(next), filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	m.Update(themeMsg{receiveTheme(t, result)})
	if m.theme.Accent != lipgloss.Color("#123") {
		t.Fatal("Directory replacement")
	}
	result = startWatch(t, ctx, path)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	m.Update(themeMsg{receiveTheme(t, result)})
	if m.theme.Accent != lipgloss.BrightBlue {
		t.Fatal("Removal")
	}
	result = startWatch(t, ctx, path)
	writeTheme(t, path, "accent = '#abc'")
	m.Update(themeMsg{receiveTheme(t, result)})
	if m.theme.Accent != lipgloss.Color("#abc") {
		t.Fatal("Later creation")
	}
	result = startWatch(t, ctx, path)
	cancel()
	select {
	case msg := <-result:
		if msg != nil {
			t.Fatal(msg)
		}
	case <-time.After(time.Second):
		t.Fatal("Cancellation blocked")
	}
	for _, line := range strings.Split(m.View().Content, "\n") {
		if ansi.StringWidth(line) > m.width {
			t.Fatal(line)
		}
	}
}

func TestWatchFileReplacementAndDirectoryRecovery(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "current", "theme", "syn.toml")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writeTheme(t, path, "accent = '#abc'")
	result := startWatch(t, ctx, path)
	next := filepath.Join(filepath.Dir(path), "next")
	writeTheme(t, next, "accent = '#fed'")
	if err := os.Rename(next, path); err != nil {
		t.Fatal(err)
	}
	if receiveTheme(t, result).Accent != lipgloss.Color("#fed") {
		t.Fatal("File replacement")
	}
	result = startWatch(t, ctx, path)
	if err := os.RemoveAll(filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	if receiveTheme(t, result).Accent != lipgloss.BrightBlue {
		t.Fatal("Directory removal")
	}
	result = startWatch(t, ctx, path)
	writeTheme(t, path, "accent = '#abc'")
	if receiveTheme(t, result).Accent != lipgloss.Color("#abc") {
		t.Fatal("Directory recovery")
	}
	result = startWatch(t, ctx, path)
	writeTheme(t, path, "accent = '#fed'")
	cancel()
	select {
	case msg := <-result:
		if msg != nil {
			t.Fatal(msg)
		}
	case <-time.After(time.Second):
		t.Fatal("Debounce cancellation blocked")
	}
}

func TestThemePathAndIndependentModels(t *testing.T) {
	home := t.TempDir()
	path := themePath(home)
	if path != filepath.Join(home, ".local", "state", "omarchy", "current", "theme", "syn.toml") {
		t.Fatal(path)
	}
	writeTheme(t, path, "accent = '#abc'\nmuted = '{{ muted }}'\nerror = '#123456'")
	m := NewWithTheme(context.Background(), nil, path, false)
	other := NewWithTheme(context.Background(), nil, "", false)
	if m.theme.Accent != lipgloss.Color("#abc") || other.theme.Accent != lipgloss.BrightBlue || m.theme.Muted != nil {
		t.Fatal(m.theme, other.theme)
	}
	if !reflect.DeepEqual(m.styles.Emphasis.GetForeground(), lipgloss.NoColor{}) {
		t.Fatal("Fixed default foreground")
	}
	if cmd := watchTheme(context.Background(), path, true); cmd != nil {
		t.Fatal("NO_COLOR started a watcher")
	}
}
