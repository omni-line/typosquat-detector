// Package version reports the build version.
package version

import "runtime/debug"

// Version, Commit and Date are set at build time via -ldflags -X.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// String returns the version banner fragment. Binaries built with
// `go install module@version` fall back to the module version.
func String() string {
	if Version != "dev" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return Version
}

// Long returns the version with commit and build date when known.
func Long() string {
	s := String()
	if Commit != "" {
		s += " (" + Commit
		if Date != "" {
			s += ", " + Date
		}
		s += ")"
	}
	return s
}
