// Command hostlier-sandbox-browser contributes a Playwright browser sidecar.
package main

import (
	"fmt"
	"github.com/Usefused/hostlier-packages/packages/sandboxes"
	"os"
)

// main keeps runtime dependencies out of the package executable.
func main() {
	if err := sandboxes.Serve("browser", os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
