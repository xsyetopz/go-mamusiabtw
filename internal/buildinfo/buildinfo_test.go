package buildinfo

import "testing"

func TestCurrentNormalizesUnknownValues(t *testing.T) {
	t.Parallel()

	prevVersion := Version
	prevRepository := Repository
	prevDescription := Description
	prevDeveloperURL := DeveloperURL
	prevSupportServerURL := SupportServerURL
	prevMascotImageURL := MascotImageURL
	defer func() {
		Version = prevVersion
		Repository = prevRepository
		Description = prevDescription
		DeveloperURL = prevDeveloperURL
		SupportServerURL = prevSupportServerURL
		MascotImageURL = prevMascotImageURL
	}()

	Version = "1.2.3"
	Repository = " UNKNOWN "
	Description = " test build "
	DeveloperURL = "https://example.com/dev"
	SupportServerURL = "UNKNOWN"
	MascotImageURL = " https://example.com/mascot.png "

	info := Current()
	if info.Version != "1.2.3" {
		t.Fatalf("unexpected version: %q", info.Version)
	}
	if info.Repository != "" {
		t.Fatalf("expected empty repository, got %q", info.Repository)
	}
	if info.Description != "test build" {
		t.Fatalf("unexpected description: %q", info.Description)
	}
	if info.DeveloperURL != "https://example.com/dev" {
		t.Fatalf("unexpected developer url: %q", info.DeveloperURL)
	}
	if info.SupportServerURL != "" {
		t.Fatalf("expected empty support url, got %q", info.SupportServerURL)
	}
	if info.MascotImageURL != "https://example.com/mascot.png" {
		t.Fatalf("unexpected mascot image url: %q", info.MascotImageURL)
	}
}
