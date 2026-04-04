package discordplatform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/features"
	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

const (
	moduleSourceBuiltin = "builtin"
	moduleSourcePlugin  = "plugins"
)

var officialPluginCatalog = map[string]string{
	"fun":        "Fun",
	"info":       "Info",
	"wellness":   "Wellness",
	"moderation": "Moderation",
	"manager":    "Manager",
}

func (b *Bot) refreshRuntimeCatalog(ctx context.Context) error {
	states, err := b.loadModuleStates(ctx)
	if err != nil {
		return err
	}

	modules := map[string]commandapi.ModuleInfo{}
	commands := map[string]commandapi.SlashCommand{}
	order := []commandapi.SlashCommand{}
	pluginCommands := map[string]pluginCommandRoute{}
	pluginRoutes := map[string]pluginRoute{}

	for _, desc := range features.Catalog() {
		cmds := desc.Commands()
		defaultEnabled := builtinDefaultEnabled(desc, b.moduleSeed)
		enabled := resolveBuiltinModuleEnabled(desc, b.moduleSeed, states)
		info := commandapi.ModuleInfo{
			ID:             desc.ID,
			Name:           desc.Name,
			Kind:           commandapi.ModuleKindCoreBuiltin,
			Runtime:        commandapi.ModuleRuntimeGo,
			Enabled:        enabled,
			DefaultEnabled: defaultEnabled,
			Toggleable:     desc.Toggleable,
			Source:         moduleSourceBuiltin,
			Commands:       slashCommandNames(cmds),
		}
		modules[info.ID] = info
		if !enabled {
			continue
		}
		for _, cmd := range cmds {
			name := strings.TrimSpace(cmd.Name)
			if name == "" {
				continue
			}
			if _, exists := commands[name]; exists {
				b.logger.WarnContext(ctx, "duplicate builtin command, skipping", slog.String("command", name), slog.String("module", desc.ID))
				continue
			}
			order = append(order, cmd)
			commands[name] = cmd
		}
	}

	b.appendPluginModules(ctx, modules, pluginRoutes, pluginCommands, commands, b.pluginHost, states)

	b.modules = modules
	b.commands = commands
	b.order = order
	b.pluginCommands = pluginCommands
	b.pluginRoutes = pluginRoutes
	return nil
}

func builtinDefaultEnabled(desc features.ModuleDescriptor, seed config.ModulesFile) bool {
	if !desc.Toggleable {
		return true
	}
	if entry, ok := seed.Modules[desc.ID]; ok && entry.Enabled != nil {
		return *entry.Enabled
	}
	return desc.DefaultEnabled
}

func resolveBuiltinModuleEnabled(
	desc features.ModuleDescriptor,
	seed config.ModulesFile,
	states map[string]store.ModuleState,
) bool {
	if !desc.Toggleable {
		return true
	}
	if state, ok := states[desc.ID]; ok {
		return state.Enabled
	}
	return builtinDefaultEnabled(desc, seed)
}

func pluginDefaultEnabled(kind commandapi.ModuleKind, moduleID string, seed config.ModulesFile) bool {
	defaultEnabled := kind == commandapi.ModuleKindUserPlugin
	if kind == commandapi.ModuleKindOfficialPlugin && seed.Defaults.OfficialEnabled != nil {
		defaultEnabled = *seed.Defaults.OfficialEnabled
	}
	if kind == commandapi.ModuleKindUserPlugin && seed.Defaults.UserEnabled != nil {
		defaultEnabled = *seed.Defaults.UserEnabled
	}
	if entry, ok := seed.Modules[moduleID]; ok && entry.Enabled != nil {
		defaultEnabled = *entry.Enabled
	}
	return defaultEnabled
}

func (b *Bot) appendPluginModules(
	ctx context.Context,
	modules map[string]commandapi.ModuleInfo,
	pluginRoutes map[string]pluginRoute,
	pluginCommands map[string]pluginCommandRoute,
	builtinCommands map[string]commandapi.SlashCommand,
	host *pluginhost.Host,
	states map[string]store.ModuleState,
) {
	if host == nil {
		return
	}

	for _, info := range host.Infos() {
		pluginRoutes[info.ID] = pluginRoute{host: host}
		kind := moduleKindForPlugin(info.ID)

		defaultEnabled := pluginDefaultEnabled(kind, info.ID, b.moduleSeed)
		enabled := defaultEnabled
		if state, ok := states[info.ID]; ok {
			enabled = state.Enabled
		}

		moduleInfo := commandapi.ModuleInfo{
			ID:             info.ID,
			Name:           strings.TrimSpace(info.Name),
			Kind:           kind,
			Runtime:        commandapi.ModuleRuntimeLua,
			Enabled:        enabled,
			DefaultEnabled: defaultEnabled,
			Toggleable:     true,
			Signed:         info.Signed,
			Source:         moduleSourcePlugin,
			Commands:       pluginCommandNames(info.Commands),
		}
		if moduleInfo.Name == "" {
			moduleInfo.Name = info.ID
		}
		modules[info.ID] = moduleInfo
		if !enabled {
			continue
		}

		for _, cmd := range info.Commands {
			name := strings.TrimSpace(cmd.Name)
			if name == "" {
				continue
			}
			if _, exists := builtinCommands[name]; exists {
				b.logger.WarnContext(ctx, "plugin command conflicts with builtin command, skipping", slog.String("command", name), slog.String("module", info.ID))
				continue
			}
			if _, exists := pluginCommands[name]; exists {
				b.logger.WarnContext(ctx, "duplicate plugin command, skipping", slog.String("command", name), slog.String("module", info.ID))
				continue
			}
			pluginCommands[name] = pluginCommandRoute{host: host, pluginID: info.ID}
		}
	}
}

func moduleKindForPlugin(pluginID string) commandapi.ModuleKind {
	if _, ok := officialPluginCatalog[strings.TrimSpace(pluginID)]; ok {
		return commandapi.ModuleKindOfficialPlugin
	}
	return commandapi.ModuleKindUserPlugin
}

func (b *Bot) enabledPluginIDsForHost(host *pluginhost.Host) map[string]struct{} {
	if host == nil {
		return nil
	}

	out := map[string]struct{}{}
	for moduleID, route := range b.pluginRoutes {
		if route.host != host {
			continue
		}
		info, ok := b.modules[moduleID]
		if !ok || !info.Enabled {
			continue
		}
		out[moduleID] = struct{}{}
	}
	return out
}

func (b *Bot) loadModuleStates(ctx context.Context) (map[string]store.ModuleState, error) {
	if b.store == nil {
		return nil, errors.New("store not configured")
	}

	states, err := b.store.ModuleStates().ListModuleStates(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string]store.ModuleState, len(states))
	for _, state := range states {
		if strings.TrimSpace(state.ModuleID) == "" {
			continue
		}
		out[state.ModuleID] = state
	}
	return out, nil
}

func slashCommandNames(commands []commandapi.SlashCommand) []string {
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if strings.TrimSpace(cmd.Name) != "" {
			out = append(out, cmd.Name)
		}
	}
	sort.Strings(out)
	return out
}

func pluginCommandNames(commands []pluginhost.Command) []string {
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if strings.TrimSpace(cmd.Name) != "" {
			out = append(out, cmd.Name)
		}
	}
	sort.Strings(out)
	return out
}

func (b *Bot) moduleInfos() []commandapi.ModuleInfo {
	out := make([]commandapi.ModuleInfo, 0, len(b.modules))
	for _, info := range b.modules {
		info.Commands = append([]string(nil), info.Commands...)
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (b *Bot) moduleInfo(moduleID string) (commandapi.ModuleInfo, bool) {
	info, ok := b.modules[strings.TrimSpace(moduleID)]
	return info, ok
}

func (b *Bot) moduleEnabled(moduleID string) bool {
	info, ok := b.moduleInfo(moduleID)
	return ok && info.Enabled
}

func (b *Bot) setModuleEnabled(ctx context.Context, moduleID string, enabled bool, actorID uint64) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return errors.New("module id is required")
	}

	info, ok := b.moduleInfo(moduleID)
	if !ok {
		return fmt.Errorf("unknown module %q", moduleID)
	}
	if !info.Toggleable {
		return fmt.Errorf("module %q is required and cannot be disabled", moduleID)
	}

	state := store.ModuleState{
		ModuleID:  moduleID,
		Enabled:   enabled,
		UpdatedAt: time.Now().UTC(),
	}
	if actorID != 0 {
		state.UpdatedBy = &actorID
	}
	if err := b.store.ModuleStates().PutModuleState(ctx, state); err != nil {
		return err
	}
	return b.reloadModules(ctx)
}

func (b *Bot) resetModule(ctx context.Context, moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return errors.New("module id is required")
	}

	info, ok := b.moduleInfo(moduleID)
	if !ok {
		return fmt.Errorf("unknown module %q", moduleID)
	}
	if !info.Toggleable {
		return fmt.Errorf("module %q is required and cannot be reset", moduleID)
	}

	if err := b.store.ModuleStates().DeleteModuleState(ctx, moduleID); err != nil {
		return err
	}
	return b.reloadModules(ctx)
}

func (b *Bot) enabledPluginJobs() []pluginhost.PluginJob {
	out := []pluginhost.PluginJob{}
	if b.pluginHost != nil {
		for _, job := range b.pluginHost.Jobs() {
			if b.moduleEnabled(job.PluginID) {
				out = append(out, job)
			}
		}
	}
	return out
}

func (b *Bot) enabledPluginEventSubscribers(eventName string) []pluginCommandRoute {
	out := []pluginCommandRoute{}
	if b.pluginHost != nil {
		for _, pluginID := range b.pluginHost.EventSubscribers(eventName) {
			if !b.moduleEnabled(pluginID) {
				continue
			}
			out = append(out, pluginCommandRoute{host: b.pluginHost, pluginID: pluginID})
		}
	}
	return out
}

func (b *Bot) reloadModules(ctx context.Context) error {
	if b.pluginHost != nil {
		if err := b.pluginHost.LoadAll(ctx); err != nil {
			return err
		}
	}
	if err := b.refreshRuntimeCatalog(ctx); err != nil {
		return err
	}
	if err := b.registerCommands(ctx); err != nil {
		return err
	}
	if b.commandRegisterAllGuilds && b.devGuildID == nil {
		if err := b.registerCommandsInCachedGuilds(ctx); err != nil {
			return err
		}
	}
	if b.pluginAuto != nil {
		b.pluginAuto.Restart(ctx)
	}
	return nil
}
