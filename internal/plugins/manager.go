package plugins

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/disgoorg/disgo/discord"

	"github.com/xsyetopz/jagpda/internal/i18n"
	"github.com/xsyetopz/jagpda/internal/luavm"
	"github.com/xsyetopz/jagpda/internal/permissions"
	"github.com/xsyetopz/jagpda/internal/store"
)

type Manager struct {
	mu sync.RWMutex

	logger *slog.Logger
	dir    string

	prodMode             bool
	allowUnsignedPlugins bool
	trustedKeysFile      string
	permissionsFile      string

	store  Store
	policy permissions.Policy
	i18n   *i18n.Registry

	plugins  map[string]*Plugin
	commands map[string]PluginCommand
}

type Store interface {
	TrustedSigners() store.TrustedSignerStore
	PluginKV() store.PluginKVStore
}

type Options struct {
	Dir                 string
	ProdMode            bool
	AllowUnsignedPlugin bool
	TrustedKeysFile     string
	PermissionsFile     string
	Store               Store
	Logger              *slog.Logger
	I18n                *i18n.Registry
}

type Plugin struct {
	ID  string
	Dir string

	Manifest  Manifest
	Signature *Signature
	Effective permissions.Permissions

	VM *luavm.VM
}

type PluginCommand struct {
	PluginID string
	Command  Command
}

type Payload struct {
	GuildID   string
	ChannelID string
	UserID    string
	Locale    string
	Options   map[string]any
}

func NewManager(opts Options) (*Manager, error) {
	if strings.TrimSpace(opts.Dir) == "" {
		return nil, errors.New("plugins dir is required")
	}
	if opts.Logger == nil {
		return nil, errors.New("logger is required")
	}

	policy, err := permissions.LoadPolicyFile(opts.PermissionsFile)
	if err != nil {
		return nil, err
	}

	return &Manager{
		logger:               opts.Logger.With(slog.String("component", "plugins")),
		dir:                  opts.Dir,
		prodMode:             opts.ProdMode,
		allowUnsignedPlugins: opts.AllowUnsignedPlugin,
		trustedKeysFile:      opts.TrustedKeysFile,
		permissionsFile:      opts.PermissionsFile,
		store:                opts.Store,
		policy:               policy,
		i18n:                 opts.I18n,
		plugins:              map[string]*Plugin{},
		commands:             map[string]PluginCommand{},
	}, nil
}

func (m *Manager) LoadAll(ctx context.Context) error {
	entries, err := m.readPluginDirEntries()
	if err != nil || entries == nil {
		return err
	}

	policy, err := permissions.LoadPolicyFile(m.permissionsFile)
	if err != nil {
		return err
	}

	m.resetPluginLocales()

	keys, err := LoadTrustedKeys(ctx, m.trustedKeysFile, m.store)
	if err != nil {
		return err
	}

	nextPlugins, nextCommands := m.loadPluginsFromEntries(ctx, entries, keys, policy)
	oldPlugins := m.swapState(nextPlugins, nextCommands, policy)
	closePlugins(oldPlugins)
	return nil
}

func (m *Manager) readPluginDirEntries() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}
	return entries, nil
}

func (m *Manager) resetPluginLocales() {
	if m.i18n != nil {
		m.i18n.ResetPluginLocales()
	}
}

func (m *Manager) loadPluginsFromEntries(
	ctx context.Context,
	entries []os.DirEntry,
	keys map[string]ed25519.PublicKey,
	policy permissions.Policy,
) (map[string]*Plugin, map[string]PluginCommand) {
	nextPlugins := map[string]*Plugin{}
	nextCommands := map[string]PluginCommand{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(m.dir, entry.Name())
		pl, cmds, err := m.loadOne(ctx, pluginDir, keys, policy)
		if err != nil {
			m.logger.WarnContext(
				ctx,
				"failed to load plugin",
				slog.String("dir", pluginDir),
				slog.String("err", err.Error()),
			)
			continue
		}
		if pl == nil {
			continue
		}

		if _, exists := nextPlugins[pl.ID]; exists {
			m.logger.WarnContext(ctx, "duplicate plugin id, skipping", slog.String("plugin", pl.ID))
			if pl.VM != nil {
				pl.VM.Close()
			}
			continue
		}

		nextPlugins[pl.ID] = pl
		m.loadPluginLocales(ctx, pl.ID, pluginDir)
		addCommands(ctx, m.logger, nextCommands, pl.ID, cmds)
	}

	return nextPlugins, nextCommands
}

func (m *Manager) loadPluginLocales(ctx context.Context, pluginID string, pluginDir string) {
	if m.i18n == nil {
		return
	}

	localesDir := filepath.Join(pluginDir, "locales")
	fi, statErr := os.Stat(localesDir)
	if statErr != nil || !fi.IsDir() {
		return
	}
	if err := m.i18n.LoadPluginLocales(pluginID, localesDir); err != nil {
		m.logger.WarnContext(
			ctx,
			"failed to load plugin locales",
			slog.String("plugin", pluginID),
			slog.String("err", err.Error()),
		)
	}
}

func addCommands(
	ctx context.Context,
	logger *slog.Logger,
	nextCommands map[string]PluginCommand,
	pluginID string,
	cmds []PluginCommand,
) {
	for _, cmd := range cmds {
		if cmd.Command.Name == "" {
			continue
		}
		if _, exists := nextCommands[cmd.Command.Name]; exists {
			logger.WarnContext(
				ctx,
				"duplicate command name, skipping",
				slog.String("command", cmd.Command.Name),
				slog.String("plugin", pluginID),
			)
			continue
		}
		nextCommands[cmd.Command.Name] = cmd
	}
}

func (m *Manager) swapState(
	nextPlugins map[string]*Plugin,
	nextCommands map[string]PluginCommand,
	policy permissions.Policy,
) map[string]*Plugin {
	m.mu.Lock()
	oldPlugins := m.plugins
	m.plugins = nextPlugins
	m.commands = nextCommands
	m.policy = policy
	m.mu.Unlock()
	return oldPlugins
}

func closePlugins(oldPlugins map[string]*Plugin) {
	for _, pl := range oldPlugins {
		if pl != nil && pl.VM != nil {
			pl.VM.Close()
		}
	}
}

func (m *Manager) Commands() map[string]PluginCommand {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]PluginCommand, len(m.commands))
	maps.Copy(out, m.commands)
	return out
}

func (m *Manager) CommandCreates() []discord.ApplicationCommandCreate {
	return m.CommandCreatesWithLocalizations(nil, nil)
}

type CommandLocalizer func(pluginID, locale, messageID string) (string, bool)

func (m *Manager) CommandCreatesWithLocalizations(
	locales []string,
	localize CommandLocalizer,
) []discord.ApplicationCommandCreate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.commands) == 0 {
		return nil
	}

	names := make([]string, 0, len(m.commands))
	for name := range m.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]discord.ApplicationCommandCreate, 0, len(names))
	for _, name := range names {
		cmd := m.commands[name]
		out = append(out, commandToCreate(name, cmd.PluginID, cmd.Command, locales, localize))
	}
	return out
}

func (m *Manager) HandleSlash(ctx context.Context, cmdName string, payload Payload) (any, bool, string, error) {
	m.mu.RLock()
	cmd, ok := m.commands[cmdName]
	if !ok {
		m.mu.RUnlock()
		return nil, false, "", fmt.Errorf("unknown plugin command %q", cmdName)
	}

	pl, ok := m.plugins[cmd.PluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, "", fmt.Errorf("plugin %q not loaded", cmd.PluginID)
	}
	m.mu.RUnlock()

	res, hasValue, err := pl.VM.CallHandle(ctx, "Handle", cmdName, luavm.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
	if err != nil {
		return nil, false, pl.ID, err
	}

	if !hasValue {
		return nil, cmd.Command.Ephemeral, pl.ID, nil
	}
	return res, cmd.Command.Ephemeral, pl.ID, nil
}

func (m *Manager) HandleComponent(ctx context.Context, pluginID, localID string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if !vm.HasFunc("HandleComponent") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleComponent", pluginID)
	}

	return vm.CallHandle(ctx, "HandleComponent", localID, luavm.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
}

func (m *Manager) HandleModal(ctx context.Context, pluginID, localID string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if !vm.HasFunc("HandleModal") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleModal", pluginID)
	}

	return vm.CallHandle(ctx, "HandleModal", localID, luavm.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
}

type PluginInfo struct {
	ID        string
	Name      string
	Version   string
	Dir       string
	Signed    bool
	Effective permissions.Permissions
	Commands  []Command
}

func (m *Manager) Infos() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]PluginInfo, 0, len(m.plugins))
	for _, pl := range m.plugins {
		if pl == nil {
			continue
		}
		out = append(out, PluginInfo{
			ID:        pl.ID,
			Name:      pl.Manifest.Name,
			Version:   pl.Manifest.Version,
			Dir:       pl.Dir,
			Signed:    pl.Signature != nil,
			Effective: pl.Effective,
			Commands:  append([]Command(nil), pl.Manifest.Commands...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) loadOne(
	_ context.Context,
	pluginDir string,
	keys map[string]ed25519.PublicKey,
	policy permissions.Policy,
) (*Plugin, []PluginCommand, error) {
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	if permErr := manifest.Permissions.Validate(); permErr != nil {
		return nil, nil, fmt.Errorf("permissions: %w", permErr)
	}

	signaturePath := filepath.Join(pluginDir, "signature.json")
	var sig *Signature
	if s, sigErr := ReadSignature(signaturePath); sigErr == nil {
		sig = &s
	} else if !os.IsNotExist(sigErr) {
		return nil, nil, sigErr
	}

	if m.prodMode && !m.allowUnsignedPlugins {
		if sig == nil {
			return nil, nil, errors.New("missing signature.json")
		}

		if verifyErr := VerifyDirSignature(pluginDir, *sig, keys); verifyErr != nil {
			return nil, nil, verifyErr
		}
	}

	script := filepath.Join(pluginDir, "plugin.lua")
	granted := policy.Granted(manifest.ID)
	effective := permissions.Effective(manifest.Permissions, granted)

	vm, err := luavm.NewFromFile(script, luavm.Options{
		Logger:      m.logger,
		PluginID:    manifest.ID,
		PluginDir:   pluginDir,
		Permissions: effective,
		I18n:        m.i18n,
		Store: func() store.PluginKVStore {
			if m.store == nil {
				return nil
			}
			return m.store.PluginKV()
		}(),
	})
	if err != nil {
		return nil, nil, err
	}

	pl := &Plugin{
		ID:        manifest.ID,
		Dir:       pluginDir,
		Manifest:  manifest,
		Signature: sig,
		Effective: effective,
		VM:        vm,
	}

	var cmds []PluginCommand
	for _, cmd := range manifest.Commands {
		if cmd.Name == "" {
			continue
		}
		cmds = append(cmds, PluginCommand{
			PluginID: pl.ID,
			Command:  cmd,
		})
	}

	return pl, cmds, nil
}

func commandToCreate(
	name string,
	pluginID string,
	cmd Command,
	locales []string,
	localize CommandLocalizer,
) discord.ApplicationCommandCreate {
	var options []discord.ApplicationCommandOption
	for _, opt := range cmd.Options {
		switch opt.Type {
		case "string":
			options = append(options, discord.ApplicationCommandOptionString{
				Name:        opt.Name,
				Description: opt.Description,
				Required:    opt.Required,
			})
		case "bool":
			options = append(options, discord.ApplicationCommandOptionBool{
				Name:        opt.Name,
				Description: opt.Description,
				Required:    opt.Required,
			})
		case "int":
			options = append(options, discord.ApplicationCommandOptionInt{
				Name:        opt.Name,
				Description: opt.Description,
				Required:    opt.Required,
			})
		case "user":
			options = append(options, discord.ApplicationCommandOptionUser{
				Name:        opt.Name,
				Description: opt.Description,
				Required:    opt.Required,
			})
		default:
		}
	}

	create := discord.SlashCommandCreate{
		Name:        name,
		Description: cmd.Description,
		Options:     options,
	}

	descID := strings.TrimSpace(cmd.DescriptionID)
	if descID != "" && len(locales) != 0 && localize != nil {
		locs := map[discord.Locale]string{}
		for _, locale := range locales {
			s, ok := localize(pluginID, locale, descID)
			if !ok {
				continue
			}
			locs[discord.Locale(locale)] = s
		}
		if len(locs) != 0 {
			create.DescriptionLocalizations = locs
		}
	}

	return create
}
