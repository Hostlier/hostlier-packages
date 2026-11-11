// Command hostlier-sandbox-computer contributes a Linux computer-use sidecar.
package main

import (
	"fmt"
	"github.com/Usefused/hostlier-packages/packages/sandboxes"
	"os"
)

// main keeps runtime dependencies out of the package executable.
func main() {
	if err := sandboxes.Serve("computer", os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
