package catalog

import (
	"sort"
	"strings"

	"github.com/xsyetopz/go-mamusiabtw/internal/commands"
	commandapi "github.com/xsyetopz/go-mamusiabtw/internal/commands/api"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	pluginhost "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins"
	store "github.com/xsyetopz/go-mamusiabtw/internal/storage"
)

const (
	ModuleSourceBuiltin = "builtin"
	ModuleSourcePlugin  = "plugins"
)

var OfficialPluginCatalog = map[string]string{
	"fun":        "Fun",
	"info":       "Info",
	"wellness":   "Wellness",
	"moderation": "Moderation",
	"manager":    "Manager",
}

func BuiltinDefaultEnabled(desc commands.ModuleDescriptor, seed config.ModulesFile) bool {
	if !desc.Toggleable {
		return true
	}
	if entry, ok := seed.Modules[desc.ID]; ok && entry.Enabled != nil {
		return *entry.Enabled
	}
	return desc.DefaultEnabled
}

func ResolveBuiltinModuleEnabled(
	desc commands.ModuleDescriptor,
	seed config.ModulesFile,
	states map[string]store.ModuleState,
) bool {
	if !desc.Toggleable {
		return true
	}
	if state, ok := states[desc.ID]; ok {
		return state.Enabled
	}
	return BuiltinDefaultEnabled(desc, seed)
}

func PluginDefaultEnabled(kind commandapi.ModuleKind, moduleID string, seed config.ModulesFile) bool {
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

func ModuleKindForPlugin(pluginID string) commandapi.ModuleKind {
	if _, ok := OfficialPluginCatalog[strings.TrimSpace(pluginID)]; ok {
		return commandapi.ModuleKindOfficialPlugin
	}
	return commandapi.ModuleKindUserPlugin
}

func SlashCommandNames(commands []commandapi.SlashCommand) []string {
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if strings.TrimSpace(cmd.Name) != "" {
			out = append(out, cmd.Name)
		}
	}
	sort.Strings(out)
	return out
}

func PluginCommandNames(commands []pluginhost.Command) []string {
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if strings.TrimSpace(cmd.Name) != "" {
			out = append(out, cmd.Name)
		}
	}
	sort.Strings(out)
	return out
}
