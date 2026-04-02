package commands

import (
	cmdadmin "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/admin"
	cmdcore "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/core"
	cmdfun "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/fun"
	cmdinfo "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/info"
	cmdmanager "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/manager"
	cmdemojis "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/manager/emojis"
	cmdroles "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/manager/roles"
	cmdstickers "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/manager/stickers"
	cmdmoderation "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/moderation"
	cmdwellness "github.com/xsyetopz/imotherbtw/internal/discordapp/commands/wellness"

	"github.com/xsyetopz/imotherbtw/internal/discordapp/core"
)

func All() []core.SlashCommand {
	out := []core.SlashCommand{}
	out = append(out, cmdcore.Commands()...)
	out = append(out, cmdinfo.Commands()...)
	out = append(out, cmdfun.Commands()...)
	out = append(out, cmdmoderation.Commands()...)
	out = append(out, cmdadmin.Commands()...)
	out = append(out, cmdmanager.Commands()...)
	out = append(out, cmdroles.Commands()...)
	out = append(out, cmdemojis.Commands()...)
	out = append(out, cmdstickers.Commands()...)
	out = append(out, cmdwellness.Commands()...)
	return out
}
