package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ModulesFile struct {
	Version  string                 `json:"version,omitempty"`
	Defaults ModuleDefaults         `json:"defaults,omitempty"`
	Modules  map[string]ModuleEntry `json:"modules,omitempty"`
}

type ModuleDefaults struct {
	OfficialEnabled *bool `json:"official_enabled,omitempty"`
	UserEnabled     *bool `json:"user_enabled,omitempty"`
}

type ModuleEntry struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func LoadModulesFile(path string) (ModulesFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ModulesFile{Modules: map[string]ModuleEntry{}}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ModulesFile{Modules: map[string]ModuleEntry{}}, nil
		}
		return ModulesFile{}, fmt.Errorf("read modules file %q: %w", path, err)
	}

	var file ModulesFile
	if err := json.Unmarshal(b, &file); err != nil {
		return ModulesFile{}, fmt.Errorf("parse modules file %q: %w", path, err)
	}
	if file.Modules == nil {
		file.Modules = map[string]ModuleEntry{}
	}
	return file, nil
}
