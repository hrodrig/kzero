package cli

// Build metadata (overridden via -ldflags from GNUmakefile / GoReleaser).
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	Branch    = "unknown"
)
