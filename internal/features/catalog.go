package features

import (
	cmdadmin "github.com/xsyetopz/go-mamusiabtw/internal/features/admin"
	cmdcore "github.com/xsyetopz/go-mamusiabtw/internal/features/corecmd"
	cmdinfo "github.com/xsyetopz/go-mamusiabtw/internal/features/info"
	cmdmanager "github.com/xsyetopz/go-mamusiabtw/internal/features/manager"
	cmdemojis "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/emojis"
	cmdroles "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/roles"
	cmdstickers "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/stickers"
	cmdmoderation "github.com/xsyetopz/go-mamusiabtw/internal/features/moderation"
	cmdwellness "github.com/xsyetopz/go-mamusiabtw/internal/features/wellness"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
)

type ModuleDescriptor struct {
	ID             string
	Name           string
	DefaultEnabled bool
	Toggleable     bool
	Commands       func() []commandapi.SlashCommand
}

func Catalog() []ModuleDescriptor {
	return []ModuleDescriptor{
		{
			ID:             "core",
			Name:           "Core",
			DefaultEnabled: true,
			Toggleable:     false,
			Commands:       cmdcore.Commands,
		},
		{
			ID:             "info",
			Name:           "Info",
			DefaultEnabled: true,
			Toggleable:     false,
			Commands:       cmdinfo.Commands,
		},
		{
			ID:             "admin",
			Name:           "Admin",
			DefaultEnabled: true,
			Toggleable:     false,
			Commands:       cmdadmin.Commands,
		},
		{
			ID:             "moderation",
			Name:           "Moderation",
			DefaultEnabled: true,
			Toggleable:     true,
			Commands:       cmdmoderation.Commands,
		},
		{
			ID:             "manager",
			Name:           "Manager",
			DefaultEnabled: true,
			Toggleable:     true,
			Commands: func() []commandapi.SlashCommand {
				out := []commandapi.SlashCommand{}
				out = append(out, cmdmanager.Commands()...)
				out = append(out, cmdroles.Commands()...)
				out = append(out, cmdemojis.Commands()...)
				out = append(out, cmdstickers.Commands()...)
				return out
			},
		},
		{
			ID:             "wellness",
			Name:           "Wellness",
			DefaultEnabled: true,
			Toggleable:     true,
			Commands:       cmdwellness.Commands,
		},
	}
}

func All() []commandapi.SlashCommand {
	out := []commandapi.SlashCommand{}
	for _, module := range Catalog() {
		out = append(out, module.Commands()...)
	}
	return out
}
