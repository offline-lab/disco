package cli

import (
	"fmt"
	"os"
)

type ExitCode int

const (
	ExitSuccess ExitCode = 0
	ExitError   ExitCode = 1
	ExitUsage   ExitCode = 2
)

func Error(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

func Fatal(msg string, err error, code ExitCode) {
	Error(msg, err)
	os.Exit(int(code))
}

func UsageError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(int(ExitUsage))
}
