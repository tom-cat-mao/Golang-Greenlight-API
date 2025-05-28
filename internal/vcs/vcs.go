package vcs

import "runtime/debug"

func Version() string {
	// Read build information from the executable.
	bi, ok := debug.ReadBuildInfo()
	if ok {
		// Return the main module's version if available.
		return bi.Main.Version
	}

	// Return an empty string if build information is not available.
	return ""
}
