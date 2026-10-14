package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type frontPageState struct {
	IndexWidth int `json:"index_width"`
}
type widthSavedMsg struct{ err error }

func statePath(xdg, home string) string {
	if xdg != "" {
		return filepath.Join(xdg, "syn", "front-page.json")
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "syn", "front-page.json")
}
func loadIndexWidth(path string) int {
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var state frontPageState
	if json.Unmarshal(data, &state) != nil || state.IndexWidth <= 0 {
		return 0
	}
	return state.IndexWidth
}
func saveIndexWidth(path string, width int) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".front-page-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	data, err := json.Marshal(frontPageState{width})
	if err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}
