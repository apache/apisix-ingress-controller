package utils

import (
	"fmt"
	"os"
	"strings"
)

// Dief renders the template string with provided args, outputs it to stderr and
// exits the program with code 1.
func Dief(template string, args ...interface{}) {
	if !strings.HasSuffix(template, "\n") {
		template += "\n"
	}
	fmt.Fprintf(os.Stderr, template, args...)
	os.Exit(1)
}

// Die outputs the provided args to stderr and exits the program with code 1.
func Die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}