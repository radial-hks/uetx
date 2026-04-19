package version

import "fmt"

// Version and Commit are set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
)

func Info() string {
	return fmt.Sprintf("uetx %s (%s)", Version, Commit)
}
