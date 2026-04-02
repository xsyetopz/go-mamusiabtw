package cmdwellness

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"

	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/core"
	"github.com/xsyetopz/go-mamusiabtw/internal/discordapp/interactions"
	"github.com/xsyetopz/go-mamusiabtw/internal/i18n"
	"github.com/xsyetopz/go-mamusiabtw/internal/present"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
	"github.com/xsyetopz/go-mamusiabtw/internal/wellness"
)

const (
	wellnessListLimit          = 25
	wellnessCheckInHistorySize = 10
	remindDeleteLabelMaxLen    = 90
)

func Commands() []core.SlashCommand {
	return []core.SlashCommand{
		timezone(),
		remind(),
		checkin(),
	}
}

func timezone() core.SlashCommand {
	return core.SlashCommand{
		Name:          "timezone",
		NameID:        "cmd.timezone.name",
		DescID:        "cmd.timezone.desc",
		CreateCommand: timezoneCreateCommand,
		Handle:        timezoneHandle,
	}
}

func timezoneCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	maxLen := 64
	minLen := 1

	return discord.SlashCommandCreate{
		Name:                     "timezone",
		NameLocalizations:        loc("cmd.timezone.name"),
		Description:              t.S("cmd.timezone.desc", nil),
		DescriptionLocalizations: loc("cmd.timezone.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "set",
				NameLocalizations:        loc("cmd.timezone.sub.set.name"),
				Description:              t.S("cmd.timezone.sub.set.desc", nil),
				DescriptionLocalizations: loc("cmd.timezone.sub.set.desc"),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:                     "iana",
						NameLocalizations:        loc("cmd.timezone.opt.iana.name"),
						Description:              t.S("cmd.timezone.opt.iana.desc", nil),
						DescriptionLocalizations: loc("cmd.timezone.opt.iana.desc"),
						Required:                 true,
						MinLength:                &minLen,
						MaxLength:                &maxLen,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "show",
				NameLocalizations:        loc("cmd.timezone.sub.show.name"),
				Description:              t.S("cmd.timezone.sub.show.desc", nil),
				DescriptionLocalizations: loc("cmd.timezone.sub.show.desc"),
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:                     "clear",
				NameLocalizations:        loc("cmd.timezone.sub.clear.name"),
				Description:              t.S("cmd.timezone.sub.clear.desc", nil),
				DescriptionLocalizations: loc("cmd.timezone.sub.clear.desc"),
			},
		},
	}
}

func timezoneHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	s core.Services,
) (interactions.SlashAction, error) {
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true),
		}, nil
	}

	userID := uint64(e.User().ID)
	settings := s.Store.UserSettings()

	switch strings.ToLower(strings.TrimSpace(*sub)) {
	case "set":
		tzRaw := strings.TrimSpace(data.String("iana"))
		tzName, ok := normalizeTimezoneName(tzRaw)
		if !ok {
			return interactions.SlashMessage{
				Create: interactions.NoticeMessage(
					present.KindError,
					"",
					t.S("wellness.timezone.invalid", map[string]any{"Timezone": tzRaw}),
					true,
				),
			}, nil
		}
		if upsertErr := settings.UpsertUserTimezone(ctx, userID, tzName); upsertErr != nil {
			return nil, upsertErr
		}
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindSuccess,
				"",
				t.S("wellness.timezone.set", map[string]any{"Timezone": tzName}),
				true,
			),
		}, nil
	case "show":
		setting, ok, err := settings.GetUserSettings(ctx, userID)
		if err != nil {
			return nil, err
		}
		tz := ""
		if ok {
			tz = strings.TrimSpace(setting.Timezone)
		}
		if tz == "" {
			return interactions.SlashMessage{
				Create: interactions.NoticeMessage(present.KindInfo, "", t.S("wellness.timezone.unset", nil), true),
			}, nil
		}
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindInfo,
				"",
				t.S("wellness.timezone.show", map[string]any{"Timezone": tz}),
				true,
			),
		}, nil
	case "clear":
		if err := settings.ClearUserTimezone(ctx, userID); err != nil {
			return nil, err
		}
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindSuccess, "", t.S("wellness.timezone.cleared", nil), true),
		}, nil
	default:
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true),
		}, nil
	}
}

func checkin() core.SlashCommand {
	return core.SlashCommand{
		Name:          "checkin",
		NameID:        "cmd.checkin.name",
		DescID:        "cmd.checkin.desc",
		CreateCommand: checkinCreateCommand,
		Handle:        checkinHandle,
	}
}

func checkinCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	minMood := 1
	maxMood := 5

	return discord.SlashCommandCreate{
		Name:                     "checkin",
		NameLocalizations:        loc("cmd.checkin.name"),
		Description:              t.S("cmd.checkin.desc", nil),
		DescriptionLocalizations: loc("cmd.checkin.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:                     "mood",
				NameLocalizations:        loc("cmd.checkin.opt.mood.name"),
				Description:              t.S("cmd.checkin.opt.mood.desc", nil),
				DescriptionLocalizations: loc("cmd.checkin.opt.mood.desc"),
				Required:                 false,
				MinValue:                 &minMood,
				MaxValue:                 &maxMood,
			},
			discord.ApplicationCommandOptionBool{
				Name:                     "history",
				NameLocalizations:        loc("cmd.checkin.opt.history.name"),
				Description:              t.S("cmd.checkin.opt.history.desc", nil),
				DescriptionLocalizations: loc("cmd.checkin.opt.history.desc"),
				Required:                 false,
			},
		},
	}
}

func checkinHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	s core.Services,
) (interactions.SlashAction, error) {
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	mood, hasMood := data.OptInt("mood")
	history, _ := data.OptBool("history")

	userID := uint64(e.User().ID)

	if history && !hasMood {
		loc := mustUserLocation(ctx, s.Store, userID)
		items, listErr := s.Store.CheckIns().ListCheckIns(ctx, userID, wellnessCheckInHistorySize)
		if listErr != nil {
			return nil, listErr
		}
		if len(items) == 0 {
			return interactions.SlashMessage{
				Create: interactions.NoticeMessage(
					present.KindInfo,
					"",
					t.S("wellness.checkin.history.empty", nil),
					true,
				),
			}, nil
		}

		lines := make([]string, 0, len(items))
		for _, c := range items {
			lines = append(lines, fmt.Sprintf("- %s: %d/5", c.CreatedAt.In(loc).Format(time.RFC822), c.Mood))
		}

		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindInfo,
				"",
				t.S("wellness.checkin.history", map[string]any{"Lines": strings.Join(lines, "\n")}),
				true,
			),
		}, nil
	}

	if !hasMood {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindInfo, "", t.S("wellness.checkin.prompt", nil), true),
		}, nil
	}

	if mood < 1 || mood > 5 {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("wellness.checkin.invalid_mood", nil), true),
		}, nil
	}

	c := store.CheckIn{
		ID:        uuid.NewString(),
		UserID:    userID,
		Mood:      mood,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Store.CheckIns().CreateCheckIn(ctx, c); err != nil {
		return nil, err
	}

	return interactions.SlashMessage{
		Create: interactions.NoticeMessage(
			present.KindSuccess,
			"",
			t.S("wellness.checkin.saved", map[string]any{"Mood": mood}),
			true,
		),
	}, nil
}

func remind() core.SlashCommand {
	return core.SlashCommand{
		Name:          "remind",
		NameID:        "cmd.remind.name",
		DescID:        "cmd.remind.desc",
		CreateCommand: remindCreateCommand,
		Handle:        remindHandle,
	}
}

func remindCreateCommand(locales []string, t core.Translator) discord.ApplicationCommandCreate {
	loc := func(id string) map[discord.Locale]string {
		return core.LocalizeMap(locales, func(locale string) string {
			return t.Registry.MustLocalize(i18n.Config{Locale: locale, MessageID: id})
		})
	}

	return discord.SlashCommandCreate{
		Name:                     "remind",
		NameLocalizations:        loc("cmd.remind.name"),
		Description:              t.S("cmd.remind.desc", nil),
		DescriptionLocalizations: loc("cmd.remind.desc"),
		Options: []discord.ApplicationCommandOption{
			remindCreateSubCommand(loc, t),
			remindListSubCommand(loc, t),
			remindDeleteSubCommand(loc, t),
		},
	}
}

func remindCreateSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	minScheduleLen := 1
	maxScheduleLen := 128
	maxNoteLen := 120

	kinds := reminderKinds()
	kindChoices := make([]discord.ApplicationCommandOptionChoiceString, 0, len(kinds))
	for _, k := range kinds {
		kindChoices = append(kindChoices, discord.ApplicationCommandOptionChoiceString{Name: k, Value: k})
	}

	deliveryChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "dm", Value: "dm"},
		{Name: "channel", Value: "channel"},
	}

	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "create",
		NameLocalizations:        loc("cmd.remind.sub.create.name"),
		Description:              t.S("cmd.remind.sub.create.desc", nil),
		DescriptionLocalizations: loc("cmd.remind.sub.create.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "schedule",
				NameLocalizations:        loc("cmd.remind.opt.schedule.name"),
				Description:              t.S("cmd.remind.opt.schedule.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.schedule.desc"),
				Required:                 true,
				MinLength:                &minScheduleLen,
				MaxLength:                &maxScheduleLen,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "kind",
				NameLocalizations:        loc("cmd.remind.opt.kind.name"),
				Description:              t.S("cmd.remind.opt.kind.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.kind.desc"),
				Required:                 true,
				Choices:                  kindChoices,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "note",
				NameLocalizations:        loc("cmd.remind.opt.note.name"),
				Description:              t.S("cmd.remind.opt.note.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.note.desc"),
				Required:                 false,
				MaxLength:                &maxNoteLen,
			},
			discord.ApplicationCommandOptionString{
				Name:                     "delivery",
				NameLocalizations:        loc("cmd.remind.opt.delivery.name"),
				Description:              t.S("cmd.remind.opt.delivery.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.delivery.desc"),
				Required:                 false,
				Choices:                  deliveryChoices,
			},
			discord.ApplicationCommandOptionChannel{
				Name:                     "channel",
				NameLocalizations:        loc("cmd.remind.opt.channel.name"),
				Description:              t.S("cmd.remind.opt.channel.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.channel.desc"),
				Required:                 false,
			},
		},
	}
}

func remindListSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "list",
		NameLocalizations:        loc("cmd.remind.sub.list.name"),
		Description:              t.S("cmd.remind.sub.list.desc", nil),
		DescriptionLocalizations: loc("cmd.remind.sub.list.desc"),
	}
}

func remindDeleteSubCommand(
	loc func(id string) map[discord.Locale]string,
	t core.Translator,
) discord.ApplicationCommandOptionSubCommand {
	return discord.ApplicationCommandOptionSubCommand{
		Name:                     "delete",
		NameLocalizations:        loc("cmd.remind.sub.delete.name"),
		Description:              t.S("cmd.remind.sub.delete.desc", nil),
		DescriptionLocalizations: loc("cmd.remind.sub.delete.desc"),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:                     "id",
				NameLocalizations:        loc("cmd.remind.opt.id.name"),
				Description:              t.S("cmd.remind.opt.id.desc", nil),
				DescriptionLocalizations: loc("cmd.remind.opt.id.desc"),
				Required:                 false,
			},
		},
	}
}

func remindHandle(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	s core.Services,
) (interactions.SlashAction, error) {
	if s.Store == nil {
		return nil, errors.New("store not configured")
	}

	data := e.SlashCommandInteractionData()
	sub := data.SubCommandName
	if sub == nil {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true),
		}, nil
	}

	switch strings.ToLower(strings.TrimSpace(*sub)) {
	case "create":
		return remindCreate(ctx, e, t, s.Store)
	case "list":
		return remindList(ctx, e, t, s.Store)
	case "delete":
		return remindDelete(ctx, e, t, s.Store)
	default:
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindError, "", t.S("err.generic", nil), true),
		}, nil
	}
}

func remindCreate(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	st core.Store,
) (interactions.SlashAction, error) {
	data := e.SlashCommandInteractionData()
	spec := strings.TrimSpace(data.String("schedule"))
	kind := strings.TrimSpace(data.String("kind"))
	note, _ := data.OptString("note")
	note = strings.TrimSpace(note)

	sched, scheduleOK := parseScheduleSpec(spec)
	if !scheduleOK {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindError,
				"",
				t.S("wellness.remind.bad_schedule", map[string]any{"Schedule": spec}),
				true,
			),
		}, nil
	}

	userID := uint64(e.User().ID)
	loc := mustUserLocation(ctx, st, userID)
	now := time.Now().UTC()
	next := sched.Next(now, loc)

	target, action, ok := remindCreateDeliveryFromInput(e, data, t)
	if !ok {
		return action, nil
	}

	r := store.Reminder{
		ID:        uuid.NewString(),
		UserID:    userID,
		Schedule:  sched.Spec(),
		Kind:      kind,
		Note:      note,
		Delivery:  target.delivery,
		GuildID:   target.guildID,
		ChannelID: target.channelID,
		Enabled:   true,
		NextRunAt: next,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if createErr := st.Reminders().CreateReminder(ctx, r); createErr != nil {
		return nil, createErr
	}

	nextStr := next.In(loc).Format(time.RFC822)
	return interactions.SlashMessage{
		Create: interactions.NoticeMessage(
			present.KindSuccess,
			"",
			t.S("wellness.remind.created", map[string]any{
				"ID":       r.ID,
				"Kind":     kind,
				"NextRun":  nextStr,
				"Delivery": string(r.Delivery),
			}),
			true,
		),
	}, nil
}

func remindList(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	st core.Store,
) (interactions.SlashAction, error) {
	userID := uint64(e.User().ID)
	loc := mustUserLocation(ctx, st, userID)
	items, err := st.Reminders().ListReminders(ctx, userID, wellnessListLimit)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindInfo, "", t.S("wellness.remind.list.empty", nil), true),
		}, nil
	}

	lines := make([]string, 0, len(items))
	for _, r := range items {
		next := r.NextRunAt.In(loc).Format(time.RFC822)
		lines = append(lines, fmt.Sprintf("- `%s` %s • %s", r.ID, r.Kind, next))
	}

	return interactions.SlashMessage{
		Create: interactions.NoticeMessage(
			present.KindInfo,
			"",
			t.S("wellness.remind.list", map[string]any{"Lines": strings.Join(lines, "\n")}),
			true,
		),
	}, nil
}

type remindCreateDelivery struct {
	delivery  store.ReminderDelivery
	guildID   *uint64
	channelID *uint64
}

func remindCreateDeliveryFromInput(
	e *events.ApplicationCommandInteractionCreate,
	data discord.SlashCommandInteractionData,
	t core.Translator,
) (remindCreateDelivery, interactions.SlashAction, bool) {
	delivery := store.ReminderDeliveryDM
	if raw, hasDelivery := data.OptString("delivery"); hasDelivery {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "channel":
			delivery = store.ReminderDeliveryChannel
		case "dm":
			delivery = store.ReminderDeliveryDM
		}
	}

	var guildIDPtr *uint64
	var channelIDPtr *uint64
	if delivery == store.ReminderDeliveryChannel {
		guildID := e.GuildID()
		if guildID == nil {
			return remindCreateDelivery{}, interactions.SlashMessage{
				Create: interactions.NoticeMessage(present.KindError, "", t.S("err.not_in_guild", nil), true),
			}, false
		}
		ch, hasChannel := data.OptChannel("channel")
		if !hasChannel || ch.ID == 0 {
			return remindCreateDelivery{}, interactions.SlashMessage{
				Create: interactions.NoticeMessage(
					present.KindError,
					"",
					t.S("wellness.remind.channel_required", nil),
					true,
				),
			}, false
		}
		g := uint64(*guildID)
		c := uint64(ch.ID)
		guildIDPtr = &g
		channelIDPtr = &c
	}

	return remindCreateDelivery{
		delivery:  delivery,
		guildID:   guildIDPtr,
		channelID: channelIDPtr,
	}, nil, true
}

func remindDelete(
	ctx context.Context,
	e *events.ApplicationCommandInteractionCreate,
	t core.Translator,
	st core.Store,
) (interactions.SlashAction, error) {
	data := e.SlashCommandInteractionData()
	userID := uint64(e.User().ID)

	if id, ok := data.OptString("id"); ok && strings.TrimSpace(id) != "" {
		deleted, err := st.Reminders().DeleteReminder(ctx, userID, strings.TrimSpace(id))
		if err != nil {
			return nil, err
		}
		if !deleted {
			return interactions.SlashMessage{
				Create: interactions.NoticeMessage(
					present.KindError,
					"",
					t.S("wellness.remind.delete.not_found", nil),
					true,
				),
			}, nil
		}
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(
				present.KindSuccess,
				"",
				t.S("wellness.remind.delete.success", nil),
				true,
			),
		}, nil
	}

	items, err := st.Reminders().ListReminders(ctx, userID, wellnessListLimit)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return interactions.SlashMessage{
			Create: interactions.NoticeMessage(present.KindInfo, "", t.S("wellness.remind.list.empty", nil), true),
		}, nil
	}

	customID := buildRemindDeleteCustomID(userID, time.Now().UTC().Unix())
	options := make([]discord.StringSelectMenuOption, 0, len(items))
	for _, r := range items {
		label := strings.TrimSpace(r.Kind)
		if label == "" {
			label = r.ID
		}
		if len(label) > remindDeleteLabelMaxLen {
			label = label[:remindDeleteLabelMaxLen]
		}
		options = append(options, discord.NewStringSelectMenuOption(label, r.ID))
	}

	menu := discord.NewStringSelectMenu(customID, t.S("wellness.remind.delete.placeholder", nil), options...).
		WithMinValues(1).
		WithMaxValues(1)

	embed := interactions.NoticeEmbed(present.KindInfo, "", t.S("wellness.remind.delete.prompt", nil))
	msg := discord.MessageCreate{
		Flags:           discord.MessageFlagEphemeral,
		Embeds:          []discord.Embed{embed},
		Components:      []discord.LayoutComponent{discord.NewActionRow(menu)},
		AllowedMentions: &discord.AllowedMentions{},
	}

	return interactions.SlashMessage{Create: msg}, nil
}

func reminderKinds() []string {
	return []string{
		"hydrate",
		"stretch",
		"breathe",
		"meds",
		"sleep",
		"checkin",
	}
}

func mustUserLocation(ctx context.Context, st core.Store, userID uint64) *time.Location {
	settings, ok, err := st.UserSettings().GetUserSettings(ctx, userID)
	if err == nil && ok && strings.TrimSpace(settings.Timezone) != "" {
		if loc, _, loadErr := wellness.LoadLocation(settings.Timezone); loadErr == nil {
			return loc
		}
	}
	return time.UTC
}

func normalizeTimezoneName(tzRaw string) (string, bool) {
	_, tzName, err := wellness.LoadLocation(tzRaw)
	if err != nil {
		return "", false
	}
	return tzName, true
}

func parseScheduleSpec(spec string) (wellness.Schedule, bool) {
	s, err := wellness.ParseSchedule(spec)
	if err != nil {
		return wellness.Schedule{}, false
	}
	return s, true
}

func buildRemindDeleteCustomID(userID uint64, issuedAt int64) string {
	return "mamusiabtw:reminddel:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatInt(issuedAt, 10)
}

func parseRemindDeleteCustomID(customID string) (uint64, int64, bool) {
	const partsCount = 4
	parts := strings.Split(strings.TrimSpace(customID), ":")
	if len(parts) != partsCount {
		return 0, 0, false
	}
	if parts[0] != "mamusiabtw" || parts[1] != "reminddel" {
		return 0, 0, false
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	issuedAt, err := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return userID, issuedAt, true
}

const remindDeleteMaxAge = 10 * time.Minute

func HandleRemindDeleteSelection(
	ctx context.Context,
	e *events.ComponentInteractionCreate,
	t core.Translator,
	st core.Store,
	customID string,
	values []string,
) (interactions.ComponentAction, error) {
	if st == nil {
		return nil, errors.New("store not configured")
	}

	actorID := uint64(e.User().ID)
	ownerID, issuedAt, ok := parseRemindDeleteCustomID(customID)
	if !ok || ownerID == 0 {
		return nil, errors.New("invalid remind delete custom id")
	}
	if ownerID != actorID {
		return interactions.ComponentUpdate{
			Update: interactions.NoticeUpdate(present.KindError, "", t.S("wellness.remind.delete.not_owner", nil)),
		}, nil
	}
	if time.Since(time.Unix(issuedAt, 0).UTC()) > remindDeleteMaxAge {
		embeds := []discord.Embed{
			interactions.NoticeEmbed(present.KindError, "", t.S("wellness.remind.delete.expired", nil)),
		}
		components := []discord.LayoutComponent{}
		return interactions.ComponentUpdate{Update: discord.MessageUpdate{
			Embeds:          &embeds,
			Components:      &components,
			AllowedMentions: &discord.AllowedMentions{},
		}}, nil
	}

	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return interactions.ComponentUpdate{
			Update: interactions.NoticeUpdate(present.KindError, "", t.S("err.generic", nil)),
		}, nil
	}

	deleted, err := st.Reminders().DeleteReminder(ctx, actorID, strings.TrimSpace(values[0]))
	if err != nil {
		return nil, err
	}
	if !deleted {
		return interactions.ComponentUpdate{
			Update: interactions.NoticeUpdate(present.KindError, "", t.S("wellness.remind.delete.not_found", nil)),
		}, nil
	}

	embeds := []discord.Embed{
		interactions.NoticeEmbed(present.KindSuccess, "", t.S("wellness.remind.delete.success", nil)),
	}
	components := []discord.LayoutComponent{}
	return interactions.ComponentUpdate{Update: discord.MessageUpdate{
		Embeds:          &embeds,
		Components:      &components,
		AllowedMentions: &discord.AllowedMentions{},
	}}, nil
}
