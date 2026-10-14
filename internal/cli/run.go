package cli

import (
	"context"
	"flag"
	"fmt"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/armada877/syn-cli/internal/feed"
	"github.com/armada877/syn-cli/internal/setup"
	"github.com/armada877/syn-cli/internal/store"
	"github.com/armada877/syn-cli/internal/tui"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

const help = `syn - RSS reader

Usage: syn [--db PATH] COMMAND
  syn, syn tui
  syn feed add URL
  syn feed list
  syn feed remove ID
  syn refresh [ID]
  syn article list [--feed ID] [--unread]
  syn article show ID
  syn article read ID
  syn article unread ID
  syn setup omarchy
  syn --help
`

type Dependencies struct {
	SetupPath string
	Open      func(string) (domain.Repository, error)
	Fetcher   domain.Fetcher
	TUI       func(context.Context, *app.App, io.Writer) error
}

func DefaultDatabase() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "syn", "articles.db"), nil
}
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWith(args, stdout, stderr, Dependencies{Open: func(path string) (domain.Repository, error) { return store.Open(path) }, Fetcher: feed.New(nil), TUI: tui.Run})
}
func RunWith(args []string, stdout, stderr io.Writer, deps Dependencies) int {
	flags := flag.NewFlagSet("syn", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "Database path")
	h := flags.Bool("help", false, "Show help")
	flags.BoolVar(h, "h", false, "Show help")
	invalid := func(err error) int { fmt.Fprintln(stderr, app.SafeText(err.Error())); return 2 }
	failed := func(err error) int { fmt.Fprintln(stderr, app.SafeText(err.Error())); return 1 }
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			_, err = io.WriteString(stdout, help)
			if err != nil {
				return failed(err)
			}
			return 0
		}
		return invalid(err)
	}
	if *h {
		if _, err := io.WriteString(stdout, help); err != nil {
			return failed(err)
		}
		return 0
	}
	command, err := parse(flags.Args())
	if err != nil {
		return invalid(err)
	}
	if command.name == "setup omarchy" {
		target := deps.SetupPath
		if target == "" {
			home, e := os.UserHomeDir()
			if e != nil {
				return failed(e)
			}
			target = setup.OmarchyPath(home)
		}
		if e := setup.InstallOmarchy(target); e != nil {
			return failed(e)
		}
		if _, e := fmt.Fprintf(stdout, "Installed %s\nOmarchy renders this template on the next theme change.\n", app.SafeText(target)); e != nil {
			return failed(e)
		}
		return 0
	}
	if *path == "" {
		*path, err = DefaultDatabase()
		if err != nil {
			return failed(err)
		}
	}
	repository, err := deps.Open(*path)
	if err != nil {
		return failed(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	a := app.New(repository, deps.Fetcher)
	code := execute(ctx, a, command, stdout, stderr, deps.TUI)
	cancel()
	if err := repository.Close(); err != nil {
		return failed(err)
	}
	return code
}

type command struct {
	name   string
	value  string
	id     int64
	filter domain.ArticleFilter
}

func parse(args []string) (command, error) {
	c := command{name: "tui"}
	bad := fmt.Errorf("Invalid arguments. Run syn --help.")
	if len(args) == 0 {
		return c, nil
	}
	if len(args) == 1 && args[0] == "tui" {
		return c, nil
	}
	if len(args) == 2 && args[0] == "setup" && args[1] == "omarchy" {
		return command{name: "setup omarchy"}, nil
	}
	if args[0] == "refresh" && len(args) <= 2 {
		c.name = "refresh"
		if len(args) == 2 {
			c.value = args[1]
		}
	} else if len(args) >= 2 && args[0] == "feed" {
		c.name = "feed " + args[1]
		switch c.name {
		case "feed list":
			if len(args) != 2 {
				return c, bad
			}
		case "feed add", "feed remove":
			if len(args) != 3 {
				return c, bad
			}
			c.value = args[2]
		default:
			return c, bad
		}
	} else if len(args) >= 2 && args[0] == "article" {
		c.name = "article " + args[1]
		switch c.name {
		case "article list":
			f := flag.NewFlagSet("article list", flag.ContinueOnError)
			f.SetOutput(io.Discard)
			f.Int64Var(&c.filter.FeedID, "feed", 0, "Feed ID")
			f.BoolVar(&c.filter.UnreadOnly, "unread", false, "Unread only")
			if err := f.Parse(args[2:]); err != nil || f.NArg() != 0 || c.filter.FeedID < 0 {
				return c, bad
			}
		case "article show", "article read", "article unread":
			if len(args) != 3 {
				return c, bad
			}
			c.value = args[2]
		default:
			return c, bad
		}
	} else {
		return c, bad
	}
	if c.name == "feed add" {
		if _, err := domain.NormalizeFeedURL(c.value); err != nil {
			return c, err
		}
	} else if c.value != "" {
		id, err := strconv.ParseInt(c.value, 10, 64)
		if err != nil || id <= 0 {
			return c, bad
		}
		c.id = id
	} else if c.name == "feed remove" || strings.HasPrefix(c.name, "article ") && c.name != "article list" || c.name == "refresh" && len(args) == 2 {
		return c, bad
	}
	return c, nil
}
func column(s string) string { return strings.Join(strings.Fields(app.SafeText(s)), " ") }
func execute(ctx context.Context, a *app.App, c command, out, errors io.Writer, runTUI func(context.Context, *app.App, io.Writer) error) int {
	var err error
	switch c.name {
	case "tui":
		err = runTUI(ctx, a, out)
	case "feed add":
		var f domain.Feed
		f, err = a.AddFeed(ctx, c.value)
		if err == nil {
			_, err = fmt.Fprintf(out, "%d\t%s\n", f.ID, column(f.URL))
		}
	case "feed list":
		var feeds []domain.Feed
		feeds, err = a.ListFeeds(ctx)
		if err == nil {
			for _, f := range feeds {
				if _, err = fmt.Fprintf(out, "%d\t%s\t%s\n", f.ID, column(f.Title), column(f.URL)); err != nil {
					break
				}
			}
		}
	case "feed remove":
		err = a.RemoveFeed(ctx, c.id)
	case "refresh":
		var results []app.RefreshResult
		results, err = a.Refresh(ctx, c.id)
		code := 0
		for _, r := range results {
			if r.Err != nil {
				code = 1
				if _, e := fmt.Fprintf(errors, "%d\t%s\n", r.Feed.ID, column(r.Err.Error())); e != nil {
					err = e
				}
			} else {
				state := "updated"
				if r.NotModified {
					state = "not modified"
				}
				if _, e := fmt.Fprintf(out, "%d\t%s\t%d\n", r.Feed.ID, state, r.Count); e != nil {
					err = e
				}
			}
		}
		if err == nil {
			return code
		}
	case "article list":
		var articles []domain.Article
		articles, err = a.ListArticles(ctx, c.filter)
		if err == nil {
			for _, article := range articles {
				state := "unread"
				if article.Read {
					state = "read"
				}
				if _, err = fmt.Fprintf(out, "%d\t%d\t%s\t%s\t%s\n", article.ID, article.FeedID, state, column(article.Title), column(article.URL)); err != nil {
					break
				}
			}
		}
	case "article show":
		var article domain.Article
		article, err = a.Article(ctx, c.id)
		if err == nil {
			_, err = fmt.Fprintln(out, app.ArticleDisplayText(article.Title, article.URL, article.Subtitle, article.Author, article.Content))
		}
	case "article read", "article unread":
		err = a.SetRead(ctx, c.id, c.name == "article read")
	}
	if err != nil {
		fmt.Fprintln(errors, app.SafeText(err.Error()))
		return 1
	}
	return 0
}
