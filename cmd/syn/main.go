package main

import (
	"github.com/armada877/syn-cli/internal/cli"
	"os"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
