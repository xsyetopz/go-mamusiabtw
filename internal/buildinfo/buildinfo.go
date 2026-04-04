package buildinfo

import "strings"

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

type Info struct {
	Version          string
	Repository       string
	Description      string
	DeveloperURL     string
	SupportServerURL string
	MascotImageURL   string
}

func Current() Info {
	return Info{
		Version:          strings.TrimSpace(Version),
		Repository:       normalizeMetadata(Repository),
		Description:      strings.TrimSpace(Description),
		DeveloperURL:     normalizeMetadata(DeveloperURL),
		SupportServerURL: normalizeMetadata(SupportServerURL),
		MascotImageURL:   normalizeMetadata(MascotImageURL),
	}
}

func normalizeMetadata(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "UNKNOWN") {
		return ""
	}
	return value
}
