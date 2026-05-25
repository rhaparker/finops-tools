package main

import (
	"os"

	"github.com/openshift-online/finops-tools/cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
