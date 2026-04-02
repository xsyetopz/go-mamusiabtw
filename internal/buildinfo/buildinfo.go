package buildinfo

// These variables can be overridden at build time with -ldflags.
//
//nolint:gochecknoglobals // Intentionally global to support simple -ldflags injection.
var (
	Version     = "dev"
	Repository  = "UNKNOWN"
	Description = "A nurturing and protective Discord app."

	DeveloperURL     = "UNKNOWN"
	SupportServerURL = "UNKNOWN"
	MascotImageURL   = "UNKNOWN"
)
