package cmdmoderation

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/xsyetopz/imotherbtw/internal/discordapp/core"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/interactions"
	"github.com/xsyetopz/imotherbtw/internal/i18n"
	"github.com/xsyetopz/imotherbtw/internal/present"
	"github.com/xsyetopz/imotherbtw/internal/store"
)

const (
	warnMaxWarnings       = 3
	unwarnListLimit       = 25
	unwarnVerifyListLimit = 100
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		warn(),
		unwarn(),
	}
}

func warn() core.SlashCommand {
	return core.SlashCommand{
		Name:          "warn",
		NameID:        "cmd.warn.name",
		DescID:        "cmd.warn.desc",
		CreateCommand: warnCreateCommand,
		Handle:        warnHandle,
	}
}

func warnCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	maxLen := 255
	minLen := 1
	perm := discord.PermissionModerateMembers

	return discord.SlashCommandCreate{
		Name: "warn",
		NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.name"})
		}),
		Description: t.S("cmd.warn.desc", nil),
		DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.desc"})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{
				Name: "user",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.opt.user.name"})
				}),
				Description: t.S("cmd.warn.opt.user.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.opt.user.desc"})
				}),
				Required: true,
			},
			discord.ApplicationCommandOptionString{
				Name: "reason",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.opt.reason.name"})
				}),
				Description: t.S("cmd.warn.opt.reason.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.warn.opt.reason.desc"})
				}),
				Required:  true,
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
	}
}

func warnHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	s core.Services,
) (interactions.SlashAction, error) {
	guildID := e.GuildID()
	if guildID == nil {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}, nil
	}
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	user := data.User("user")
	reason := strings.TrimSpace(data.String("reason"))

	actorID := uint64(e.User().ID)
	targetID := uint64(user.ID)

	if actorID == targetID {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("mod.warn.self", nil), true),
		}, nil
	}
	if user.Bot || user.System {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("mod.warn.bot", nil), true),
		}, nil
	}
	if reason == "" {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true),
		}, nil
	}

	warnings := s.Store.Warnings()
	guildIDU64 := uint64(*guildID)

	count, err := warnings.CountWarnings(ctx, guildIDU64, targetID)
	if err != nil {
		return nil, err
	}
	if count >= warnMaxWarnings {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindError,
				"",
				t.S("mod.warn.too_many", map[string]any{"User": user.Mention()}),
				true,
			),
		}, nil
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	if err = warnings.CreateWarning(ctx, store.Warning{
		ID:          id,
		GuildID:     guildIDU64,
		UserID:      targetID,
		ModeratorID: actorID,
		Reason:      reason,
		CreatedAt:   now,
	}); err != nil {
		return nil, err
	}

	warnAppendAudit(ctx, s.Store, guildIDU64, actorID, targetID, now)

	_ = sendDM(
		ctx,
		e.Client(),
		user.ID,
		interactions.NoticeMessage(
			present.KindWarning,
			"",
			t.S("mod.warn.dm", map[string]any{"Reason": reason}),
			false,
		),
	)

	return interactions.SlashMessage{
		Create: interactions.NoticeMessage(
			present.KindSuccess,
			"",
			t.S("mod.warn.success", map[string]any{"User": user.Mention(), "Reason": reason}),
			true,
		),
	}, nil
}

func warnAppendAudit(ctx context.Context, s core.Store, guildID, actorID, targetID uint64, now time.Time) {
	if s == nil {
		return
	}

	auditGuildID := guildID
	auditActorID := actorID
	auditTargetID := targetID
	auditTargetType := store.TargetTypeUser
	_ = s.Audit().Append(ctx, store.AuditEntry{
		GuildID:    &auditGuildID,
		ActorID:    &auditActorID,
		Action:     "warn.create",
		TargetType: &auditTargetType,
		TargetID:   &auditTargetID,
		CreatedAt:  now,
		MetaJSON:   "{}",
	})
}

func unwarn() core.SlashCommand {
	return core.SlashCommand{
		Name:          "unwarn",
		NameID:        "cmd.unwarn.name",
		DescID:        "cmd.unwarn.desc",
		CreateCommand: unwarnCreateCommand,
		Handle:        unwarnHandle,
	}
}

func unwarnCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	perm := discord.PermissionModerateMembers

	return discord.SlashCommandCreate{
		Name: "unwarn",
		NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.unwarn.name"})
		}),
		Description: t.S("cmd.unwarn.desc", nil),
		DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.unwarn.desc"})
		}),
		DefaultMemberPermissions: omit.NewPtr(perm),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{
				Name: "user",
				NameLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.unwarn.opt.user.name"})
				}),
				Description: t.S("cmd.unwarn.opt.user.desc", nil),
				DescriptionLocalizations: core.LocalizeMap(locales, func(locale string) string {
					return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: "cmd.unwarn.opt.user.desc"})
				}),
				Required: true,
			},
		},
	}
}

func unwarnHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	s core.Services,
) (interactions.SlashAction, error) {
	guildID := e.GuildID()
	if guildID == nil {
		return interactions.SlashMessage{
			Create: discord.NewMessageCreate().WithEphemeral(true).WithContent(t.S("err.not_in_guild", nil)),
		}, nil
	}
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	user := data.User("user")

	actorID := uint64(e.User().ID)
	targetID := uint64(user.ID)

	if actorID == targetID {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("mod.unwarn.self", nil), true),
		}, nil
	}
	if user.Bot || user.System {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("mod.warn.bot", nil), true),
		}, nil
	}

	warnings := s.Store.Warnings()
	list, err := warnings.ListWarnings(ctx, uint64(*guildID), targetID, unwarnListLimit)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindError,
				"",
				t.S("mod.unwarn.none", map[string]any{"User": user.Mention()}),
				true,
			),
		}, nil
	}

	customID := buildUnwarnCustomID(actorID, uint64(*guildID), targetID, time.Now().UTC().Unix())

	options := make([]discord.StringSelectMenuOption, 0, len(list))
	for _, w := range list {
		label := fmtWarningLabel(w)
		options = append(options, discord.NewStringSelectMenuOption(label, w.ID))
	}

	menu := discord.NewStringSelectMenu(customID, t.S("mod.unwarn.placeholder", nil), options...).
		WithMinValues(1).
		WithMaxValues(1)

	embed := interactions.NoticeEmbed(present.KindInfo, "", t.S("mod.unwarn.prompt", nil))
	msg := discord.MessageCreate{
		Flags:           discord.MessageFlagEphemeral,
		Embeds:          []discord.Embed{embed},
		Components:      []discord.LayoutComponent{discord.NewActionRow(menu)},
		AllowedMentions: &discord.AllowedMentions{},
	}

	return interactions.SlashMessage{Create: msg}, nil
}

func HandleUnwarnSelection(
	ctx context.Context,
	e *events.ComponentInteractionCreate,
	t core.Translator,
	s core.Services,
	customID string,
	values []string,
) (interactions.ComponentAction, error) {
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	actorID, guildID, targetID, issuedAt, ok := parseUnwarnCustomID(customID)
	if !ok {
		return interactions.ComponentAcknowledge{}, nil
	}

	if uint64(e.User().ID) != actorID {
		return interactions.ComponentAcknowledge{}, nil
	}

	now := time.Now().UTC()
	if issuedAt <= 0 || now.Sub(time.Unix(issuedAt, 0).UTC()) > 2*time.Minute {
		embeds := []discord.Embed{interactions.NoticeEmbed(present.KindError, "", t.S("mod.unwarn.expired", nil))}
		components := []discord.LayoutComponent{}
		return interactions.ComponentUpdate{Update: discord.MessageUpdate{
			Embeds:          &embeds,
			Components:      &components,
			AllowedMentions: &discord.AllowedMentions{},
		}}, nil
	}

	if len(values) < 1 {
		return interactions.ComponentAcknowledge{}, nil
	}
	selectedID := values[0]

	warnings := s.Store.Warnings()
	list, err := warnings.ListWarnings(ctx, guildID, targetID, unwarnVerifyListLimit)
	if err != nil {
		return nil, err
	}

	found := false
	for _, w := range list {
		if w.ID == selectedID {
			found = true
			break
		}
	}
	if !found {
		embeds := []discord.Embed{interactions.NoticeEmbed(present.KindError, "", t.S("err.generic", nil))}
		components := []discord.LayoutComponent{}
		return interactions.ComponentUpdate{Update: discord.MessageUpdate{
			Embeds:          &embeds,
			Components:      &components,
			AllowedMentions: &discord.AllowedMentions{},
		}}, nil
	}

	if deleteErr := warnings.DeleteWarning(ctx, selectedID); deleteErr != nil {
		return nil, deleteErr
	}

	auditGuildID := guildID
	auditActorID := actorID
	auditTargetID := targetID
	auditTargetType := store.TargetTypeUser
	_ = s.Store.Audit().Append(ctx, store.AuditEntry{
		GuildID:    &auditGuildID,
		ActorID:    &auditActorID,
		Action:     "warn.delete",
		TargetType: &auditTargetType,
		TargetID:   &auditTargetID,
		CreatedAt:  now,
		MetaJSON:   "{}",
	})

	embeds := []discord.Embed{
		interactions.NoticeEmbed(
			present.KindSuccess,
			"",
			t.S("mod.unwarn.success", map[string]any{"User": discord.UserMention(snowflake.ID(targetID))}),
		),
	}
	components := []discord.LayoutComponent{}
	return interactions.ComponentUpdate{Update: discord.MessageUpdate{
		Embeds:          &embeds,
		Components:      &components,
		AllowedMentions: &discord.AllowedMentions{},
	}}, nil
}

func sendDM(_ context.Context, client *bot.Client, userID snowflake.ID, create discord.MessageCreate) error {
	dm, err := client.Rest.CreateDMChannel(userID)
	if err != nil {
		return err
	}

	_, err = client.Rest.CreateMessage(dm.ID(), create)
	return err
}

func buildUnwarnCustomID(actorID, guildID, targetID uint64, issuedAt int64) string {
	return "imotherbtw:unwarn:" + strconv.FormatUint(
		actorID,
		10,
	) + ":" + strconv.FormatUint(
		guildID,
		10,
	) + ":" + strconv.FormatUint(
		targetID,
		10,
	) + ":" + strconv.FormatInt(
		issuedAt,
		10,
	)
}

func parseUnwarnCustomID(customID string) (uint64, uint64, uint64, int64, bool) {
	const unwarnCustomIDParts = 6

	parts := strings.Split(customID, ":")
	if len(parts) != unwarnCustomIDParts {
		return 0, 0, 0, 0, false
	}
	if parts[0] != "imotherbtw" || parts[1] != "unwarn" {
		return 0, 0, 0, 0, false
	}

	actorID, actorOK := parseUint64(parts[2])
	if !actorOK {
		return 0, 0, 0, 0, false
	}
	guildID, guildOK := parseUint64(parts[3])
	if !guildOK {
		return 0, 0, 0, 0, false
	}
	targetID, targetOK := parseUint64(parts[4])
	if !targetOK {
		return 0, 0, 0, 0, false
	}

	issuedAt, issuedOK := parseInt64(parts[5])
	if !issuedOK {
		return 0, 0, 0, 0, false
	}

	return actorID, guildID, targetID, issuedAt, true
}

func parseUint64(raw string) (uint64, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseInt64(raw string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func fmtWarningLabel(w store.Warning) string {
	created := w.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}

	// Keep labels short (Discord limit ~100 chars).
	return discord.UserMention(snowflake.ID(w.ModeratorID)) + " - " + created.Format("2006-01-02")
}
