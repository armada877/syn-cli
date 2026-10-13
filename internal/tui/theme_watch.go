package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"time"
)

type themeMsg struct{ theme Theme }

func watchTheme(ctx context.Context, path string, noColor bool) tea.Cmd {
	if path == "" || noColor {
		return nil
	}
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}
		defer watcher.Close()
		attach := func() {
			for _, dir := range watcher.WatchList() {
				_ = watcher.Remove(dir)
			}
			dir := filepath.Dir(path)
			count := 0
			for {
				if info, err := os.Stat(dir); err == nil && info.IsDir() {
					_ = watcher.Add(dir)
					count++
					if count == 3 {
						break
					}
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
		attach()
		timer := time.NewTimer(time.Hour)
		if !timer.Stop() {
			<-timer.C
		}
		defer timer.Stop()
		var settled <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return nil
			case _, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				attach()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(250 * time.Millisecond)
				settled = timer.C
			case <-settled:
				return themeMsg{loadTheme(path, noColor)}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				attach()
			}
		}
	}
}
