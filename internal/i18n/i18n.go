package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Registry struct {
	state *registryState
}

type registryState struct {
	mu sync.RWMutex

	core    *i18n.Bundle
	plugins map[string]*i18n.Bundle
	locales []string
}

func LoadCore(localesDir string) (Registry, error) {
	locales, err := listLocales(localesDir)
	if err != nil {
		return Registry{}, err
	}

	bundle := i18n.NewBundle(language.MustParse("en-GB"))
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	for _, locale := range locales {
		path := filepath.Join(localesDir, locale, "messages.json")
		if loadErr := loadMessages(bundle, locale, path); loadErr != nil {
			return Registry{}, fmt.Errorf("load %q: %w", path, loadErr)
		}
	}

	return Registry{
		state: &registryState{
			core:    bundle,
			plugins: map[string]*i18n.Bundle{},
			locales: locales,
		},
	}, nil
}

func (r *Registry) LoadPluginLocales(pluginID, pluginLocalesDir string) error {
	if strings.TrimSpace(pluginID) == "" {
		return errors.New("plugin id is required")
	}
	if r == nil || r.state == nil {
		return errors.New("i18n registry not initialized")
	}

	locales, err := listLocales(pluginLocalesDir)
	if err != nil {
		return err
	}

	bundle := i18n.NewBundle(language.MustParse("en-GB"))
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	for _, locale := range locales {
		path := filepath.Join(pluginLocalesDir, locale, "messages.json")
		if loadErr := loadMessages(bundle, locale, path); loadErr != nil {
			return fmt.Errorf("load %q: %w", path, loadErr)
		}
	}

	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	if r.state.plugins == nil {
		r.state.plugins = map[string]*i18n.Bundle{}
	}

	r.state.plugins[pluginID] = bundle
	return nil
}

func (r *Registry) SupportedLocales() []string {
	if r == nil || r.state == nil {
		return nil
	}
	r.state.mu.RLock()
	defer r.state.mu.RUnlock()
	return append([]string(nil), r.state.locales...)
}

func (r *Registry) ResetPluginLocales() {
	if r == nil || r.state == nil {
		return
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	r.state.plugins = map[string]*i18n.Bundle{}
}

func normalizeLocale(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return "en-GB"
	}
	return locale
}

type Config struct {
	Locale       string
	PluginID     string
	MessageID    string
	TemplateData map[string]any
	PluralCount  any
}

func (r *Registry) Localize(cfg Config) (string, error) {
	if r == nil || r.state == nil {
		return "", errors.New("i18n registry not initialized")
	}

	messageID := strings.TrimSpace(cfg.MessageID)
	if messageID == "" {
		return "", nil
	}

	locale := normalizeLocale(cfg.Locale)
	fallback := "en-GB"

	if cfg.PluginID != "" {
		r.state.mu.RLock()
		bundle, ok := r.state.plugins[cfg.PluginID]
		r.state.mu.RUnlock()
		if !ok || bundle == nil {
			return "", fmt.Errorf("missing plugin locale bundle for %q", cfg.PluginID)
		}

		localizer := i18n.NewLocalizer(bundle, locale, fallback)
		return localizer.Localize(&i18n.LocalizeConfig{
			MessageID:    messageID,
			TemplateData: cfg.TemplateData,
			PluralCount:  cfg.PluralCount,
		})
	}

	r.state.mu.RLock()
	core := r.state.core
	r.state.mu.RUnlock()
	localizer := i18n.NewLocalizer(core, locale, fallback)
	return localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: cfg.TemplateData,
		PluralCount:  cfg.PluralCount,
	})
}

func (r *Registry) MustLocalize(cfg Config) string {
	s, err := r.Localize(cfg)
	if err != nil {
		if cfg.MessageID == "" {
			return ""
		}
		return cfg.MessageID
	}
	return s
}

func (r *Registry) TryLocalize(cfg Config) (string, bool) {
	s, err := r.Localize(cfg)
	if err != nil || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

func listLocales(localesDir string) ([]string, error) {
	localesDir = strings.TrimSpace(localesDir)
	if localesDir == "" {
		return nil, errors.New("locales dir is required")
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, fmt.Errorf("read locales dir %q: %w", localesDir, err)
	}

	var locales []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		locale := entry.Name()
		path := filepath.Join(localesDir, locale, "messages.json")
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}

		locales = append(locales, locale)
	}

	sort.Strings(locales)
	return locales, nil
}

func loadMessages(bundle *i18n.Bundle, locale, filePath string) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// go-i18n infers language tags from file names, not directory names.
	// Keep our on-disk layout `/<locale>/messages.json`, but pass a synthetic
	// file name with the locale embedded so tags are parsed correctly.
	synthetic := "active." + strings.TrimSpace(locale) + ".json"
	_, err = bundle.ParseMessageFileBytes(b, synthetic)
	return err
}
