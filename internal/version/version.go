// Package version provides version information for HELM.
package version

import "fmt"

// These variables are set during build using ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns the full version string.
func String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
