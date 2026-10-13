package setup

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed syn.toml.tpl
var template []byte

func OmarchyPath(home string) string {
	return filepath.Join(home, ".config", "omarchy", "themed", "syn.toml.tpl")
}

func InstallOmarchy(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		info, e := os.Lstat(path)
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Refuse nonregular template: %s", path)
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if bytes.Equal(data, template) {
			return nil
		}
		return fmt.Errorf("Preserve existing template: %s", path)
	}
	if err != nil {
		return err
	}
	_, err = file.Write(template)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
