package botengine

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/xsyetopz/imotherbtw/internal/discordapp/core"
	"github.com/xsyetopz/imotherbtw/internal/discordapp/interactions"
	"github.com/xsyetopz/imotherbtw/internal/present"
	"github.com/xsyetopz/imotherbtw/internal/store"
	"github.com/xsyetopz/imotherbtw/internal/wellness"
)

const (
	reminderClaimLimit     = 10
	reminderLeaseDuration  = 30 * time.Second
	reminderPollInterval   = 5 * time.Second
	reminderMaxFailures    = 3
	reminderDefaultLocale  = discord.LocaleEnglishUS
	reminderMessageMaxNote = 120
)

func (b *Bot) startReminderScheduler(ctx context.Context) {
	if b == nil || b.client == nil || b.store == nil {
		return
	}
	go b.runReminderScheduler(ctx, uuid.NewString())
}

func (b *Bot) runReminderScheduler(ctx context.Context, leaseID string) {
	ticker := time.NewTicker(reminderPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.tickReminders(ctx, leaseID)
		}
	}
}

func (b *Bot) tickReminders(ctx context.Context, leaseID string) {
	now := time.Now().UTC()
	reminders, err := b.store.Reminders().ClaimDueReminders(
		ctx,
		now,
		leaseID,
		reminderLeaseDuration,
		reminderClaimLimit,
	)
	if err != nil {
		b.logger.ErrorContext(ctx, "claim due reminders failed", slog.String("err", err.Error()))
		return
	}

	for _, r := range reminders {
		b.runOneReminder(ctx, leaseID, now, r)
	}
}

func (b *Bot) runOneReminder(ctx context.Context, leaseID string, now time.Time, r store.Reminder) {
	t := core.Translator{
		Registry: b.i18n,
		Locale:   reminderDefaultLocale,
		UserID:   r.UserID,
	}

	loc := b.userLocation(ctx, r.UserID)
	sched, err := wellness.ParseSchedule(r.Schedule)
	if err != nil {
		b.logger.ErrorContext(
			ctx,
			"invalid reminder schedule",
			slog.String("err", err.Error()),
			slog.String("id", r.ID),
		)
		_ = b.finishReminder(ctx, leaseID, r, now, now.Add(365*24*time.Hour), r.FailureCount+1, false)
		return
	}

	next := sched.Next(now, loc)
	if next.IsZero() || !next.After(now) {
		next = now.Add(time.Hour)
	}

	failureCount := r.FailureCount
	enabled := r.Enabled
	if enabled {
		if sendErr := b.sendReminder(ctx, t, r); sendErr != nil {
			failureCount++
			if failureCount >= reminderMaxFailures {
				enabled = false
			}
			b.logger.WarnContext(
				ctx,
				"send reminder failed",
				slog.String("id", r.ID),
				slog.String("err", sendErr.Error()),
				slog.Int("failures", failureCount),
				slog.Bool("enabled", enabled),
			)
		} else {
			failureCount = 0
		}
	}

	if !enabled {
		next = now.Add(365 * 24 * time.Hour)
	}

	_ = b.finishReminder(ctx, leaseID, r, now, next, failureCount, enabled)
}

func (b *Bot) finishReminder(
	ctx context.Context,
	leaseID string,
	r store.Reminder,
	lastRunAt time.Time,
	nextRunAt time.Time,
	failureCount int,
	enabled bool,
) error {
	if b == nil || b.store == nil {
		return errors.New("store not configured")
	}
	return b.store.Reminders().FinishReminderRun(ctx, r.ID, leaseID, lastRunAt, nextRunAt, failureCount, enabled)
}

func (b *Bot) userLocation(ctx context.Context, userID uint64) *time.Location {
	if b == nil || b.store == nil {
		return time.UTC
	}
	settings, ok, err := b.store.UserSettings().GetUserSettings(ctx, userID)
	if err == nil && ok && strings.TrimSpace(settings.Timezone) != "" {
		if loc, _, loadErr := wellness.LoadLocation(settings.Timezone); loadErr == nil {
			return loc
		}
	}
	return time.UTC
}

func (b *Bot) sendReminder(ctx context.Context, t core.Translator, r store.Reminder) error {
	if b == nil || b.client == nil {
		return errors.New("discord client not configured")
	}

	kindText := reminderKindText(t, r.Kind)
	note := strings.TrimSpace(r.Note)
	if len(note) > reminderMessageMaxNote {
		note = note[:reminderMessageMaxNote]
	}

	body := t.S("wellness.reminder.fire", map[string]any{
		"Kind": kindText,
		"Note": note,
	})

	msg := interactions.NoticeMessage(present.KindInfo, "", body, false)

	switch r.Delivery {
	case store.ReminderDeliveryChannel:
		if r.ChannelID == nil || *r.ChannelID == 0 {
			return errors.New("missing channel_id for channel delivery")
		}
		_, err := b.client.Rest.CreateMessage(snowflake.ID(*r.ChannelID), msg)
		return err
	case store.ReminderDeliveryDM:
		fallthrough
	default:
		chID, err := b.ensureDMChannel(ctx, r.UserID)
		if err != nil {
			return err
		}
		_, err = b.client.Rest.CreateMessage(snowflake.ID(chID), msg)
		return err
	}
}

func reminderKindText(t core.Translator, kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "hydrate":
		return t.S("wellness.reminder.kind.hydrate", nil)
	case "stretch":
		return t.S("wellness.reminder.kind.stretch", nil)
	case "breathe":
		return t.S("wellness.reminder.kind.breathe", nil)
	case "meds":
		return t.S("wellness.reminder.kind.meds", nil)
	case "sleep":
		return t.S("wellness.reminder.kind.sleep", nil)
	case "checkin":
		return t.S("wellness.reminder.kind.checkin", nil)
	default:
		return strings.TrimSpace(kind)
	}
}

func (b *Bot) ensureDMChannel(ctx context.Context, userID uint64) (uint64, error) {
	settingsStore := b.store.UserSettings()
	setting, ok, err := settingsStore.GetUserSettings(ctx, userID)
	if err != nil {
		return 0, err
	}
	if ok && setting.DMChannelID != nil && *setting.DMChannelID != 0 {
		return *setting.DMChannelID, nil
	}

	dm, err := b.client.Rest.CreateDMChannel(snowflake.ID(userID))
	if err != nil {
		return 0, err
	}
	chID := uint64(dm.ID())
	if chID != 0 {
		_ = settingsStore.UpsertUserDMChannelID(ctx, userID, chID)
	}
	return chID, nil
}
