package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Images        string `toml:"images"`
	ImageFallback string `toml:"image_fallback"`
}

func Default() Config { return Config{Images: "auto", ImageFallback: "blocks"} }

func Path(xdg, home string) string {
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "syn", "config.toml")
}

func Load(path string) (Config, string) {
	defaults := Default()
	if path == "" {
		return defaults, ""
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return defaults, ""
	}
	if err != nil {
		return defaults, "Cannot read image settings"
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 {
		return defaults, "Cannot read image settings"
	}
	parsed := defaults
	if toml.Unmarshal(data, &parsed) != nil {
		return defaults, "Invalid image settings; using defaults"
	}
	status := ""
	if parsed.Images != "auto" && parsed.Images != "off" {
		parsed.Images = defaults.Images
		status = "Invalid image settings; using defaults"
	}
	if parsed.ImageFallback != "blocks" && parsed.ImageFallback != "placeholder" {
		parsed.ImageFallback = defaults.ImageFallback
		status = "Invalid image settings; using defaults"
	}
	return parsed, status
}
