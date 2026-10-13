package cli

import (
	"bytes"
	"context"
	"errors"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/armada877/syn-cli/internal/feed"
	"github.com/armada877/syn-cli/internal/store"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func deps(path string) Dependencies {
	return Dependencies{Open: func(string) (domain.Repository, error) { return store.Open(path) }, Fetcher: feed.New(&http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "fail") {
			return nil, errors.New("fixture failure")
		}
		body := `<rss version="2.0"><channel><title>Local feed</title><link>http://local.test</link><description>Fixture</description><item><guid>one</guid><title>Article one</title><link>http://local.test/one</link><description>&lt;p&gt;Hello &amp;amp; world&lt;/p&gt;</description></item></channel></rss>`
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}), TUI: func(context.Context, *app.App, io.Writer) error { return nil }}
}
func run(t *testing.T, d Dependencies, args ...string) (int, string, string) {
	t.Helper()
	var out, err bytes.Buffer
	code := RunWith(args, &out, &err, d)
	return code, out.String(), err.String()
}
func TestHelpAndArguments(t *testing.T) {
	d := Dependencies{Open: func(string) (domain.Repository, error) { t.Fatal("opened database"); return nil, nil }}
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--db", "/bad/path", "--help"}} {
		code, out, err := run(t, d, args...)
		if code != 0 || !strings.Contains(out, "Usage: syn") || err != "" {
			t.Fatalf("%v: %d %q %q", args, code, out, err)
		}
	}
	for _, args := range [][]string{{"bad"}, {"--bad"}, {"feed"}, {"feed", "add"}, {"feed", "add", "file:///tmp/feed"}, {"feed", "remove", "0"}, {"refresh", "-1"}, {"refresh", "1", "2"}, {"article", "show", "bad"}, {"article", "list", "--feed", "-1"}, {"article", "list", "extra"}, {"tui", "extra"}} {
		code, out, err := run(t, d, args...)
		if code != 2 || out != "" || err == "" {
			t.Fatalf("%v: %d %q %q", args, code, out, err)
		}
	}
}
func TestCommandsAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "articles.db")
	d := deps(path)
	steps := []struct {
		args     []string
		code     int
		out, err string
	}{
		{[]string{"feed", "add", "http://local.test/feed"}, 0, "1\thttp://local.test/feed", ""},
		{[]string{"feed", "list"}, 0, "1\t\thttp://local.test/feed", ""},
		{[]string{"article", "list"}, 0, "", ""},
		{[]string{"refresh", "1"}, 0, "1\tupdated\t1", ""},
		{[]string{"feed", "list"}, 0, "Local feed", ""},
		{[]string{"article", "list", "--feed", "1", "--unread"}, 0, "1\t1\tunread\tArticle one", ""},
		{[]string{"article", "show", "1"}, 0, "Hello & world", ""},
		{[]string{"article", "list", "--unread"}, 0, "unread", ""},
		{[]string{"article", "read", "1"}, 0, "", ""},
		{[]string{"article", "list", "--unread"}, 0, "", ""},
		{[]string{"article", "list"}, 0, "1\t1\tread", ""},
		{[]string{"refresh"}, 0, "updated", ""},
		{[]string{"article", "list"}, 0, "1\t1\tread", ""},
		{[]string{"article", "unread", "1"}, 0, "", ""},
		{[]string{"article", "list", "--unread"}, 0, "unread", ""},
		{[]string{"article", "show", "99"}, 1, "", "not found"},
		{[]string{"feed", "add", "http://local.test/fail"}, 0, "2\t", ""},
		{[]string{"refresh"}, 1, "1\tupdated", "fixture failure"},
		{[]string{"feed", "remove", "1"}, 0, "", ""},
		{[]string{"article", "list"}, 0, "", ""},
		{[]string{"feed", "remove", "1"}, 1, "", "not found"},
		{[]string{"article", "read", "1"}, 1, "", "not found"},
		{[]string{"article", "unread", "1"}, 1, "", "not found"},
		{[]string{"refresh", "99"}, 1, "", "not found"},
		{nil, 0, "", ""}, {[]string{"tui"}, 0, "", ""},
	}
	for _, step := range steps {
		code, out, err := run(t, d, step.args...)
		if code != step.code || !strings.Contains(out, step.out) || !strings.Contains(err, step.err) || (step.out == "" && out != "") || (step.err == "" && err != "") {
			t.Fatalf("%v: %d %q %q", step.args, code, out, err)
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestOutputAndOpenErrors(t *testing.T) {
	var err bytes.Buffer
	if code := RunWith([]string{"--help"}, brokenWriter{}, &err, Dependencies{}); code != 1 || !strings.Contains(err.String(), "write failed") {
		t.Fatalf("%d %s", code, &err)
	}
	d := deps(filepath.Join(t.TempDir(), "data.db"))
	for _, args := range [][]string{{"feed", "add", "http://local.test/feed"}, {"feed", "list"}, {"refresh"}, {"article", "list"}, {"article", "show", "1"}} {
		err.Reset()
		if code := RunWith(args, brokenWriter{}, &err, d); code != 1 {
			t.Fatalf("%v: %d", args, code)
		}
	}
	d.Open = func(string) (domain.Repository, error) { return nil, errors.New("open failed") }
	code, out, message := run(t, d, "feed", "list")
	if code != 1 || out != "" || !strings.Contains(message, "open failed") {
		t.Fatalf("%d %q %q", code, out, message)
	}
}
func TestDatabasePathAndColumns(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/data")
	path, err := DefaultDatabase()
	if err != nil || path != "/tmp/data/syn/articles.db" {
		t.Fatal(path, err)
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/home")
	path, err = DefaultDatabase()
	if err != nil || path != "/tmp/home/.local/share/syn/articles.db" {
		t.Fatal(path, err)
	}
	d := deps(filepath.Join(t.TempDir(), "data.db"))
	open := d.Open
	d.Open = func(path string) (domain.Repository, error) {
		if path != "explicit.db" {
			t.Fatal(path)
		}
		return open(path)
	}
	if code, _, _ := run(t, d, "--db", "explicit.db", "feed", "list"); code != 0 {
		t.Fatal(code)
	}
	if got := column("a\tb\nc\r\x1b[31md"); got != "a b cd" {
		t.Fatal(got)
	}
}

func TestOmarchySetupWithoutStorage(t *testing.T) {
	d := Dependencies{SetupPath: filepath.Join(t.TempDir(), "themed", "syn.toml.tpl"), Open: func(string) (domain.Repository, error) { t.Fatal("Opened database"); return nil, nil }}
	for range 2 {
		code, out, message := run(t, d, "setup", "omarchy")
		if code != 0 || !strings.Contains(out, d.SetupPath) || !strings.Contains(out, "next theme change") || message != "" {
			t.Fatal(code, out, message)
		}
	}
	for _, args := range [][]string{{"setup"}, {"setup", "other"}, {"setup", "omarchy", "extra"}} {
		code, _, _ := run(t, d, args...)
		if code != 2 {
			t.Fatal(args, code)
		}
	}
	code := RunWith([]string{"setup", "omarchy"}, brokenWriter{}, io.Discard, d)
	if code != 1 {
		t.Fatal(code)
	}
	d.SetupPath = filepath.Join(d.SetupPath, "bad")
	code, _, message := run(t, d, "setup", "omarchy")
	if code != 1 || message == "" {
		t.Fatal(code, message)
	}
}
