// Package version holds build-time metadata, set via -ldflags at build time.
package version

var (
	CommitSHA = "unknown"
	BuildTime = "unknown"
)
