package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		data    string
		want    Config
		warning bool
	}{
		{"", Default(), false},
		{"images = 'off'\nimage_fallback = 'placeholder'", Config{"off", "placeholder"}, false},
		{"unknown = true", Default(), false},
		{"images = [", Default(), true},
		{"images = 'bad'\nimage_fallback = 'bad'", Default(), true},
		{"images = 3", Default(), true},
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
			t.Fatal(err)
		}
		got, status := Load(path)
		if got != test.want || (status != "") != test.warning {
			t.Fatalf("%q: %+v %q", test.data, got, status)
		}
		after, _ := os.ReadFile(path)
		if string(after) != test.data {
			t.Fatal("configuration changed")
		}
	}
	got, status := Load(filepath.Join(t.TempDir(), "missing"))
	if got != Default() || status != "" {
		t.Fatal(got, status)
	}
	if Path("/xdg", "/home/test") != "/xdg/syn/config.toml" || Path("", "/home/test") != "/home/test/.config/syn/config.toml" {
		t.Fatal("incorrect paths")
	}
}

func TestImageModesAndOverride(t *testing.T) {
	for _, mode := range []string{"auto", "kitty", "blocks", "placeholder", "off"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte("images = '"+mode+"'"), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, warning := Load(path)
			if cfg.Images != mode || warning != "" {
				t.Fatal(cfg, warning)
			}
			for _, override := range []string{"", "auto", "kitty", "blocks", "placeholder", "off", "bad\x1b[31m", " KITTY", "KITTY"} {
				got, warning := ApplyImageOverride(cfg, override)
				want := mode
				if ValidImageMode(override) {
					want = override
				}
				invalid := override != "" && !ValidImageMode(override)
				if got.Images != want || got.ImageFallback != cfg.ImageFallback || (warning != "") != invalid || strings.ContainsAny(warning, "\x1b\n\r") {
					t.Fatal(got, warning)
				}
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "images = '"+mode+"'" {
				t.Fatal("configuration changed", err)
			}
		})
	}
}
