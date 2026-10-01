package version

// Version is set at build time via -ldflags.
var Version = "dev"

// String returns the version banner fragment.
func String() string {
	return Version
}
