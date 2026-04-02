package features

import (
	cmdadmin "github.com/xsyetopz/go-mamusiabtw/internal/features/admin"
	cmdcore "github.com/xsyetopz/go-mamusiabtw/internal/features/corecmd"
	cmdfun "github.com/xsyetopz/go-mamusiabtw/internal/features/fun"
	cmdinfo "github.com/xsyetopz/go-mamusiabtw/internal/features/info"
	cmdmanager "github.com/xsyetopz/go-mamusiabtw/internal/features/manager"
	cmdemojis "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/emojis"
	cmdroles "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/roles"
	cmdstickers "github.com/xsyetopz/go-mamusiabtw/internal/features/manager/stickers"
	cmdmoderation "github.com/xsyetopz/go-mamusiabtw/internal/features/moderation"
	cmdwellness "github.com/xsyetopz/go-mamusiabtw/internal/features/wellness"

	"github.com/xsyetopz/go-mamusiabtw/internal/features/commandapi"
)

func All() []commandapi.SlashCommand {
	out := []commandapi.SlashCommand{}
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
