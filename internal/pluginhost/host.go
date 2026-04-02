package pluginhost

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

	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost/lua"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

type Host struct {
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

	eventSubs map[string][]string
	jobs      []PluginJob
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
	Commands  []Command
	Events    []string
	Jobs      []Job

	VM *luaplugin.VM
}

type PluginCommand struct {
	PluginID string
	Command  Command
}

type PluginJob struct {
	PluginID string
	JobID    string
	Schedule string
}

type Payload struct {
	GuildID   string
	ChannelID string
	UserID    string
	Locale    string
	Options   map[string]any
}

func NewHost(opts Options) (*Host, error) {
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

	return &Host{
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
		eventSubs:            map[string][]string{},
	}, nil
}

func (m *Host) LoadAll(ctx context.Context) error {
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
	nextEvents, nextJobs := buildSubscriptions(nextPlugins)
	oldPlugins := m.swapState(nextPlugins, nextCommands, nextEvents, nextJobs, policy)
	closePlugins(oldPlugins)
	return nil
}

func (m *Host) readPluginDirEntries() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}
	return entries, nil
}

func (m *Host) resetPluginLocales() {
	if m.i18n != nil {
		m.i18n.ResetPluginLocales()
	}
}

func (m *Host) loadPluginsFromEntries(
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

func (m *Host) loadPluginLocales(ctx context.Context, pluginID string, pluginDir string) {
	if m.i18n == nil {
		return
	}

	localesDir := filepath.Join(pluginDir, "locales")
	fi, statErr := os.Stat(localesDir)
	if statErr != nil || !fi.IsDir() {
		return
	}

	if entries, err := os.ReadDir(localesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			locale := strings.TrimSpace(entry.Name())
			if locale == "" || i18n.IsSupportedDiscordLocale(locale) {
				continue
			}

			path := filepath.Join(localesDir, locale, "messages.json")
			if _, msgFileErr := os.Stat(path); msgFileErr != nil {
				continue
			}

			m.logger.WarnContext(
				ctx,
				"unknown plugin locale, ignoring",
				slog.String("plugin", pluginID),
				slog.String("locale", locale),
				slog.String("path", path),
			)
		}
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

func (m *Host) swapState(
	nextPlugins map[string]*Plugin,
	nextCommands map[string]PluginCommand,
	nextEvents map[string][]string,
	nextJobs []PluginJob,
	policy permissions.Policy,
) map[string]*Plugin {
	m.mu.Lock()
	oldPlugins := m.plugins
	m.plugins = nextPlugins
	m.commands = nextCommands
	m.eventSubs = nextEvents
	m.jobs = nextJobs
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

func buildSubscriptions(pls map[string]*Plugin) (map[string][]string, []PluginJob) {
	ev := map[string][]string{}
	var jobs []PluginJob

	for _, pl := range pls {
		if pl == nil {
			continue
		}

		for _, raw := range pl.Events {
			name := strings.ToLower(strings.TrimSpace(raw))
			if name == "" {
				continue
			}
			ev[name] = append(ev[name], pl.ID)
		}

		for _, job := range pl.Jobs {
			id := strings.TrimSpace(job.ID)
			spec := strings.TrimSpace(job.Schedule)
			if id == "" || spec == "" {
				continue
			}
			jobs = append(jobs, PluginJob{
				PluginID: pl.ID,
				JobID:    id,
				Schedule: spec,
			})
		}
	}

	for name := range ev {
		sort.Strings(ev[name])
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].PluginID != jobs[j].PluginID {
			return jobs[i].PluginID < jobs[j].PluginID
		}
		return jobs[i].JobID < jobs[j].JobID
	})

	return ev, jobs
}

func defaultEphemeralForCommand(cmd Command, opts map[string]any) bool {
	if opts == nil {
		return cmd.Ephemeral
	}

	sub := readPayloadString(opts, "__subcommand")
	if sub == "" {
		return cmd.Ephemeral
	}

	group := readPayloadString(opts, "__group")

	if group != "" {
		return defaultEphemeralFromGroups(cmd.Groups, group, sub, cmd.Ephemeral)
	}

	return defaultEphemeralFromSubcommands(cmd.Subcommands, sub, cmd.Ephemeral)
}

func readPayloadString(opts map[string]any, key string) string {
	if opts == nil {
		return ""
	}
	v, ok := opts[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func defaultEphemeralFromGroups(groups []CommandGroup, group, sub string, fallback bool) bool {
	for _, g := range groups {
		if strings.TrimSpace(g.Name) != group {
			continue
		}
		return defaultEphemeralFromSubcommands(g.Subcommands, sub, fallback)
	}
	return fallback
}

func defaultEphemeralFromSubcommands(subs []Subcommand, sub string, fallback bool) bool {
	for _, sc := range subs {
		if strings.TrimSpace(sc.Name) != sub {
			continue
		}
		if sc.Ephemeral != nil {
			return *sc.Ephemeral
		}
		return fallback
	}
	return fallback
}

func (m *Host) Commands() map[string]PluginCommand {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]PluginCommand, len(m.commands))
	maps.Copy(out, m.commands)
	return out
}

func (m *Host) Jobs() []PluginJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]PluginJob(nil), m.jobs...)
}

func (m *Host) EventSubscribers(eventName string) []string {
	eventName = strings.ToLower(strings.TrimSpace(eventName))
	if eventName == "" {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]string(nil), m.eventSubs[eventName]...)
}

func (m *Host) EffectivePermissions(pluginID string) (permissions.Permissions, bool) {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return permissions.Permissions{}, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pl, ok := m.plugins[pluginID]
	if !ok || pl == nil {
		return permissions.Permissions{}, false
	}
	return pl.Effective, true
}

func (m *Host) CommandCreates() []discord.ApplicationCommandCreate {
	return m.CommandCreatesWithLocalizations(nil, nil)
}

type CommandLocalizer func(pluginID, locale, messageID string) (string, bool)

func (m *Host) CommandCreatesWithLocalizations(
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

func (m *Host) HandleSlash(ctx context.Context, cmdName string, payload Payload) (any, bool, string, error) {
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

	var (
		res      any
		hasValue bool
		err      error
	)
	if pl.VM.HasDefinition() {
		res, hasValue, err = pl.VM.CallRoute(ctx, luaplugin.RouteCommand, cmdName, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	} else {
		res, hasValue, err = pl.VM.CallHandle(ctx, "Handle", cmdName, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	}
	if err != nil {
		return nil, false, pl.ID, err
	}

	defaultEphemeral := defaultEphemeralForCommand(cmd.Command, payload.Options)
	if !hasValue {
		return nil, defaultEphemeral, pl.ID, nil
	}
	return res, defaultEphemeral, pl.ID, nil
}

func (m *Host) HandleComponent(ctx context.Context, pluginID, localID string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if vm.HasDefinition() {
		return vm.CallRoute(ctx, luaplugin.RouteComponent, localID, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	}

	if !vm.HasFunc("HandleComponent") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleComponent", pluginID)
	}

	return vm.CallHandle(ctx, "HandleComponent", localID, luaplugin.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
}

func (m *Host) HandleModal(ctx context.Context, pluginID, localID string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if vm.HasDefinition() {
		return vm.CallRoute(ctx, luaplugin.RouteModal, localID, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	}

	if !vm.HasFunc("HandleModal") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleModal", pluginID)
	}

	return vm.CallHandle(ctx, "HandleModal", localID, luaplugin.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
}

func (m *Host) HandleEvent(ctx context.Context, pluginID, eventName string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if vm.HasDefinition() {
		return vm.CallRoute(ctx, luaplugin.RouteEvent, eventName, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	}

	if !vm.HasFunc("HandleEvent") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleEvent", pluginID)
	}

	return vm.CallHandle(ctx, "HandleEvent", eventName, luaplugin.Payload{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		Locale:    payload.Locale,
		Options:   payload.Options,
	})
}

func (m *Host) HandleJob(ctx context.Context, pluginID, jobID string, payload Payload) (any, bool, error) {
	m.mu.RLock()
	pl, ok := m.plugins[pluginID]
	if !ok || pl.VM == nil {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("plugin %q not loaded", pluginID)
	}
	vm := pl.VM
	m.mu.RUnlock()

	if vm.HasDefinition() {
		return vm.CallRoute(ctx, luaplugin.RouteJob, jobID, luaplugin.Payload{
			GuildID:   payload.GuildID,
			ChannelID: payload.ChannelID,
			UserID:    payload.UserID,
			Locale:    payload.Locale,
			Options:   payload.Options,
		})
	}

	if !vm.HasFunc("HandleJob") {
		return nil, false, fmt.Errorf("plugin %q does not implement HandleJob", pluginID)
	}

	return vm.CallHandle(ctx, "HandleJob", jobID, luaplugin.Payload{
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

func (m *Host) Infos() []PluginInfo {
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
			Commands:  append([]Command(nil), pl.Commands...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Host) loadOne(
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

	vm, err := luaplugin.NewFromFile(script, luaplugin.Options{
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

	descriptor, hasDescriptor := vm.Definition()

	commands := append([]Command(nil), manifest.Commands...)
	events := append([]string(nil), manifest.Events...)
	jobs := append([]Job(nil), manifest.Jobs...)
	if hasDescriptor {
		commands = commandsFromDefinition(descriptor)
		events = append([]string(nil), descriptor.Events...)
		jobs = jobsFromDefinition(descriptor)
	}

	pl := &Plugin{
		ID:        manifest.ID,
		Dir:       pluginDir,
		Manifest:  manifest,
		Signature: sig,
		Effective: effective,
		Commands:  commands,
		Events:    events,
		Jobs:      jobs,
		VM:        vm,
	}

	var cmds []PluginCommand
	for _, cmd := range pl.Commands {
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
	if len(cmd.Subcommands) > 0 || len(cmd.Groups) > 0 {
		options = append(options, buildSubcommands(pluginID, cmd.Subcommands, locales, localize)...)
		options = append(options, buildGroups(pluginID, cmd.Groups, locales, localize)...)
	} else {
		options = append(options, buildOptions(pluginID, cmd.Options, locales, localize)...)
	}

	create := discord.SlashCommandCreate{
		Name:        name,
		Description: cmd.Description,
		Options:     options,
	}

	if locs := descriptionLocalizations(pluginID, cmd.DescriptionID, locales, localize); len(locs) != 0 {
		create.DescriptionLocalizations = locs
	}

	return create
}

func descriptionLocalizations(
	pluginID string,
	descriptionID string,
	locales []string,
	localize CommandLocalizer,
) map[discord.Locale]string {
	descID := strings.TrimSpace(descriptionID)
	if descID == "" || len(locales) == 0 || localize == nil {
		return nil
	}

	locs := map[discord.Locale]string{}
	for _, locale := range locales {
		s, ok := localize(pluginID, locale, descID)
		if !ok {
			continue
		}
		locs[discord.Locale(locale)] = s
	}
	return locs
}

func buildSubcommands(
	pluginID string,
	cmds []Subcommand,
	locales []string,
	localize CommandLocalizer,
) []discord.ApplicationCommandOption {
	out := make([]discord.ApplicationCommandOption, 0, len(cmds))
	for _, sc := range cmds {
		opt := discord.ApplicationCommandOptionSubCommand{
			Name:        sc.Name,
			Description: sc.Description,
			Options:     buildOptions(pluginID, sc.Options, locales, localize),
		}
		if locs := descriptionLocalizations(pluginID, sc.DescriptionID, locales, localize); len(locs) != 0 {
			opt.DescriptionLocalizations = locs
		}
		out = append(out, opt)
	}
	return out
}

func buildGroups(
	pluginID string,
	groups []CommandGroup,
	locales []string,
	localize CommandLocalizer,
) []discord.ApplicationCommandOption {
	out := make([]discord.ApplicationCommandOption, 0, len(groups))
	for _, g := range groups {
		opt := discord.ApplicationCommandOptionSubCommandGroup{
			Name:        g.Name,
			Description: g.Description,
		}
		if locs := descriptionLocalizations(pluginID, g.DescriptionID, locales, localize); len(locs) != 0 {
			opt.DescriptionLocalizations = locs
		}

		subs := make([]discord.ApplicationCommandOptionSubCommand, 0, len(g.Subcommands))
		for _, sc := range g.Subcommands {
			sub := discord.ApplicationCommandOptionSubCommand{
				Name:        sc.Name,
				Description: sc.Description,
				Options:     buildOptions(pluginID, sc.Options, locales, localize),
			}
			if locs := descriptionLocalizations(pluginID, sc.DescriptionID, locales, localize); len(locs) != 0 {
				sub.DescriptionLocalizations = locs
			}
			subs = append(subs, sub)
		}
		opt.Options = subs

		out = append(out, opt)
	}
	return out
}

func buildOptions(
	pluginID string,
	opts []CommandOption,
	locales []string,
	localize CommandLocalizer,
) []discord.ApplicationCommandOption {
	out := make([]discord.ApplicationCommandOption, 0, len(opts))
	for _, opt := range opts {
		if o, ok := buildOption(pluginID, opt, locales, localize); ok {
			out = append(out, o)
		}
	}
	return out
}

func buildOption(
	pluginID string,
	opt CommandOption,
	locales []string,
	localize CommandLocalizer,
) (discord.ApplicationCommandOption, bool) {
	typ := strings.ToLower(strings.TrimSpace(opt.Type))
	descLocs := descriptionLocalizations(pluginID, opt.DescriptionID, locales, localize)
	switch typ {
	case "string":
		return buildStringOption(opt, descLocs), true
	case "bool":
		return buildBoolOption(opt, descLocs), true
	case "int":
		return buildIntOption(opt, descLocs), true
	case "float":
		return buildFloatOption(opt, descLocs), true
	case "user":
		return buildUserOption(opt, descLocs), true
	case "channel":
		return buildChannelOption(opt, descLocs), true
	case "role":
		return buildRoleOption(opt, descLocs), true
	case "mentionable":
		return buildMentionableOption(opt, descLocs), true
	case "attachment":
		return buildAttachmentOption(opt, descLocs), true
	default:
		return nil, false
	}
}

func buildStringOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionString {
	o := discord.ApplicationCommandOptionString{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
		MinLength:   opt.MinLength,
		MaxLength:   opt.MaxLength,
		Choices:     buildStringChoices(opt.Choices),
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildBoolOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionBool {
	o := discord.ApplicationCommandOptionBool{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildIntOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionInt {
	o := discord.ApplicationCommandOptionInt{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
		Choices:     buildIntChoices(opt.Choices),
		MinValue:    floatToIntPtr(opt.MinValue),
		MaxValue:    floatToIntPtr(opt.MaxValue),
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildFloatOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionFloat {
	o := discord.ApplicationCommandOptionFloat{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
		Choices:     buildFloatChoices(opt.Choices),
		MinValue:    opt.MinValue,
		MaxValue:    opt.MaxValue,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildUserOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionUser {
	o := discord.ApplicationCommandOptionUser{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildChannelOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionChannel {
	o := discord.ApplicationCommandOptionChannel{
		Name:         opt.Name,
		Description:  opt.Description,
		Required:     opt.Required,
		ChannelTypes: buildChannelTypes(opt.ChannelTypes),
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildRoleOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionRole {
	o := discord.ApplicationCommandOptionRole{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildMentionableOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionMentionable {
	o := discord.ApplicationCommandOptionMentionable{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildAttachmentOption(
	opt CommandOption,
	descLocs map[discord.Locale]string,
) discord.ApplicationCommandOptionAttachment {
	o := discord.ApplicationCommandOptionAttachment{
		Name:        opt.Name,
		Description: opt.Description,
		Required:    opt.Required,
	}
	if len(descLocs) != 0 {
		o.DescriptionLocalizations = descLocs
	}
	return o
}

func buildStringChoices(in []OptionChoice) []discord.ApplicationCommandOptionChoiceString {
	out := make([]discord.ApplicationCommandOptionChoiceString, 0, len(in))
	for _, c := range in {
		v, ok := c.Value.(string)
		if !ok {
			continue
		}
		out = append(out, discord.ApplicationCommandOptionChoiceString{Name: c.Name, Value: v})
	}
	return out
}

func buildIntChoices(in []OptionChoice) []discord.ApplicationCommandOptionChoiceInt {
	out := make([]discord.ApplicationCommandOptionChoiceInt, 0, len(in))
	for _, c := range in {
		v, ok := floatToInt(c.Value)
		if !ok {
			continue
		}
		out = append(out, discord.ApplicationCommandOptionChoiceInt{Name: c.Name, Value: v})
	}
	return out
}

func buildFloatChoices(in []OptionChoice) []discord.ApplicationCommandOptionChoiceFloat {
	out := make([]discord.ApplicationCommandOptionChoiceFloat, 0, len(in))
	for _, c := range in {
		switch v := c.Value.(type) {
		case float64:
			out = append(out, discord.ApplicationCommandOptionChoiceFloat{Name: c.Name, Value: v})
		case int:
			out = append(out, discord.ApplicationCommandOptionChoiceFloat{Name: c.Name, Value: float64(v)})
		}
	}
	return out
}

func floatToIntPtr(v *float64) *int {
	if v == nil {
		return nil
	}
	if i, ok := floatToInt(*v); ok {
		return &i
	}
	return nil
}

func floatToInt(v any) (int, bool) {
	switch vv := v.(type) {
	case float64:
		if vv != float64(int(vv)) {
			return 0, false
		}
		return int(vv), true
	case int:
		return vv, true
	default:
		return 0, false
	}
}

func buildChannelTypes(in []int) []discord.ChannelType {
	if len(in) == 0 {
		return nil
	}
	out := make([]discord.ChannelType, 0, len(in))
	for _, v := range in {
		out = append(out, discord.ChannelType(v))
	}
	return out
}
