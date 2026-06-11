// Command easy-infra is a CLI for managing a project's local/dev
// infrastructure through named profiles of services.
package main

import (
	"fmt"
	"os"

	"github.com/minhnc/easy-infra/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
