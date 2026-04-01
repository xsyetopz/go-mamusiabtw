package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xsyetopz/jagpda/internal/permissions"
)

type Manifest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`

	Permissions permissions.Permissions `json:"permissions"`

	Commands []Command `json:"commands"`
}

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// DescriptionID is an optional i18n key used for command description localization.
	DescriptionID string `json:"description_id,omitempty"`
	Ephemeral     bool   `json:"ephemeral"`

	Options []CommandOption `json:"options"`
}

type CommandOption struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func ReadManifest(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if unmarshalErr := json.Unmarshal(b, &m); unmarshalErr != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", unmarshalErr)
	}

	if m.ID == "" {
		return Manifest{}, errors.New("manifest missing id")
	}

	return m, nil
}
