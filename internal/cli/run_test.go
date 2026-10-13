package cli_test

import (
	"bytes"
	"strings"
	"syn-cli/internal/cli"
	"testing"
)

func TestHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"-h"}} {
		var stdout, stderr bytes.Buffer
		if code := cli.Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("exit code %d", code)
		}
		if !strings.Contains(stdout.String(), "Usage: syn") || stderr.Len() != 0 {
			t.Fatalf("stdout %q; stderr %q", stdout.String(), stderr.String())
		}
	}
}

func TestInvalidArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code %d", code)
	}
	if stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("stdout %q; stderr %q", stdout.String(), stderr.String())
	}
}
