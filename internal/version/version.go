// Package version provides version information for tailnginx.
package version

import (
	"fmt"
	"runtime"
)

// Version information. These are set via -ldflags during build.
var (
	Version   = "1.4.0"        // Application version
	GitCommit = "dev"          // Git commit hash
	BuildDate = "unknown"      // Build date
	GoVersion = runtime.Version() // Go version used to build
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf(
		"tailnginx version %s\n"+
			"Git commit: %s\n"+
			"Build date: %s\n"+
			"Go version: %s\n"+
			"OS/Arch: %s/%s",
		Version,
		GitCommit,
		BuildDate,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Short returns a short version string.
func Short() string {
	if GitCommit != "dev" {
		return fmt.Sprintf("%s (%s)", Version, GitCommit[:7])
	}
	return Version
}
