package cli

import (
	"fmt"
	"io"
)

const help = "syn - RSS reader\n\nUsage: syn [--help]\n\nOptions:\n  -h, --help  Show help\n"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		if _, err := io.WriteString(stdout, help); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(stderr, "Invalid arguments. Run syn --help.")
	return 2
}
