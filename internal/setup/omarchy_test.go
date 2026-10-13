package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallOmarchy(t *testing.T) {
	path := OmarchyPath(t.TempDir())
	for range 2 {
		if err := InstallOmarchy(path); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, template) {
		t.Fatal(string(data), err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatal(info, err)
	}
	if err := os.WriteFile(path, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InstallOmarchy(path); err == nil {
		t.Fatal("Overwrote custom file")
	}
	data, _ = os.ReadFile(path)
	if string(data) != "custom" {
		t.Fatal(string(data))
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, template, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := InstallOmarchy(path); err == nil {
		t.Fatal("Accepted symlink")
	}
	if got, err := os.Readlink(path); err != nil || got != target {
		t.Fatal(got, err)
	}
	if err := InstallOmarchy(filepath.Join(target, "syn.toml.tpl")); err == nil {
		t.Fatal("Accepted file parent")
	}
}
