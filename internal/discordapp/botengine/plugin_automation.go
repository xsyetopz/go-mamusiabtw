package botengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/robfig/cron/v3"

	"github.com/xsuetopz/go-mamusiabtw/internal/permissions"
	"github.com/xsuetopz/go-mamusiabtw/internal/plugins"
)

const (
	pluginEventMemberJoin  = "guild_member_join"
	pluginEventMemberLeave = "guild_member_leave"
	pluginEventGuildBan    = "guild_ban"
	pluginEventGuildUnban  = "guild_unban"
)

const (
	defaultPluginAutomationBurst      = 3
	defaultPluginAutomationRatePerSec = 1.0
	defaultPluginAutomationTimeout    = 2 * time.Second

	pluginCronStopTimeout = 3 * time.Second
)

type pluginAutomation struct {
	mu sync.Mutex

	logger *slog.Logger
	bot    *Bot

	cron *cron.Cron

	limiter *tokenBucketLimiter
}

func newPluginAutomation(b *Bot) *pluginAutomation {
	if b == nil {
		return nil
	}

	return &pluginAutomation{
		logger:  b.logger.With(slog.String("component", "plugin_automation")),
		bot:     b,
		limiter: newTokenBucketLimiter(defaultPluginAutomationRatePerSec, defaultPluginAutomationBurst),
	}
}

func (p *pluginAutomation) Start(ctx context.Context) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.bot == nil || p.bot.plugins == nil || p.bot.client == nil {
		return
	}
	if p.cron != nil {
		return
	}

	jobEntries := p.bot.plugins.Jobs()
	if len(jobEntries) == 0 {
		return
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))

	for _, job := range jobEntries {
		if strings.TrimSpace(job.PluginID) == "" ||
			strings.TrimSpace(job.JobID) == "" ||
			strings.TrimSpace(job.Schedule) == "" {
			continue
		}

		if _, err := c.AddFunc(job.Schedule, func() {
			p.runJob(context.Background(), job)
		}); err != nil {
			p.logger.WarnContext(
				ctx,
				"invalid plugin job schedule",
				slog.String("plugin", job.PluginID),
				slog.String("job", job.JobID),
				slog.String("err", err.Error()),
			)
			continue
		}
	}

	c.Start()
	p.cron = c

	go func() {
		<-ctx.Done()
		p.Stop()
	}()
}

func (p *pluginAutomation) Stop() {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cron == nil {
		return
	}

	ctx := p.cron.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(pluginCronStopTimeout):
	}
	p.cron = nil
}

func (p *pluginAutomation) Restart(ctx context.Context) {
	p.Stop()
	p.Start(ctx)
}

func (p *pluginAutomation) FireEvent(eventName string, payload plugins.Payload) {
	eventName = strings.ToLower(strings.TrimSpace(eventName))
	if eventName == "" {
		return
	}

	if p == nil || p.bot == nil || p.bot.plugins == nil {
		return
	}

	pluginIDs := p.bot.plugins.EventSubscribers(eventName)
	if len(pluginIDs) == 0 {
		return
	}

	go p.fireEvent(context.Background(), pluginIDs, eventName, payload)
}

func (p *pluginAutomation) fireEvent(
	ctx context.Context,
	pluginIDs []string,
	eventName string,
	payload plugins.Payload,
) {
	for _, pluginID := range pluginIDs {
		if strings.TrimSpace(pluginID) == "" {
			continue
		}
		p.runEventOne(ctx, pluginID, eventName, payload)
	}
}

func (p *pluginAutomation) runEventOne(ctx context.Context, pluginID, eventName string, payload plugins.Payload) {
	callCtx, cancel := context.WithTimeout(ctx, defaultPluginAutomationTimeout)
	defer cancel()

	perms, ok := p.bot.plugins.EffectivePermissions(pluginID)
	if !ok {
		return
	}

	if !eventAllowed(perms, eventName) {
		p.logger.WarnContext(
			callCtx,
			"plugin event denied by permissions",
			slog.String("plugin", pluginID),
			slog.String("event", eventName),
		)
		return
	}

	res, hasValue, err := p.bot.plugins.HandleEvent(callCtx, pluginID, eventName, payload)
	if err != nil {
		p.logger.WarnContext(
			callCtx,
			"plugin event failed",
			slog.String("plugin", pluginID),
			slog.String("event", eventName),
			slog.String("err", err.Error()),
		)
		return
	}
	if !hasValue {
		return
	}

	actions, parseErr := parseAutomationActions(res)
	if parseErr != nil {
		p.logger.WarnContext(
			callCtx,
			"plugin event response invalid",
			slog.String("plugin", pluginID),
			slog.String("event", eventName),
			slog.String("err", parseErr.Error()),
		)
		return
	}
	p.executeAutomationActions(callCtx, pluginID, perms, payload, actions)
}

func (p *pluginAutomation) runJob(ctx context.Context, job plugins.PluginJob) {
	if p == nil || p.bot == nil || p.bot.plugins == nil || p.bot.client == nil {
		return
	}

	perms, ok := p.bot.plugins.EffectivePermissions(job.PluginID)
	if !ok || !perms.Automation.Jobs {
		return
	}

	for guild := range p.bot.client.Caches.Guilds() {
		guildID := uint64(guild.ID)
		if guildID == 0 {
			continue
		}

		locale := strings.TrimSpace(guild.PreferredLocale)
		if locale == "" {
			locale = discord.LocaleEnglishUS.Code()
		}

		callCtx, cancel := context.WithTimeout(ctx, defaultPluginAutomationTimeout)
		res, hasValue, err := p.bot.plugins.HandleJob(callCtx, job.PluginID, job.JobID, plugins.Payload{
			GuildID:   snowflake.ID(guildID).String(),
			ChannelID: "",
			UserID:    "",
			Locale:    locale,
			Options: map[string]any{
				"job_id": job.JobID,
			},
		})
		cancel()
		if err != nil || !hasValue {
			continue
		}

		actions, parseErr := parseAutomationActions(res)
		if parseErr != nil {
			p.logger.WarnContext(
				ctx,
				"plugin job response invalid",
				slog.String("plugin", job.PluginID),
				slog.String("job", job.JobID),
				slog.String("err", parseErr.Error()),
			)
			continue
		}

		p.executeAutomationActions(ctx, job.PluginID, perms, plugins.Payload{
			GuildID: snowflake.ID(guildID).String(),
			Locale:  locale,
		}, actions)
	}
}

func eventAllowed(perms permissions.Permissions, eventName string) bool {
	switch eventName {
	case pluginEventMemberJoin, pluginEventMemberLeave:
		return perms.Automation.Events.MemberJoinLeave
	case pluginEventGuildBan, pluginEventGuildUnban:
		return perms.Automation.Events.Moderation
	default:
		return false
	}
}

type automationAction struct {
	Type      string
	ChannelID string
	UserID    string
	Message   any
}

func parseAutomationActions(raw any) ([]automationAction, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("automation response must be an object")
	}
	actionsRaw, ok := m["actions"]
	if !ok {
		return nil, errors.New("automation response missing actions")
	}
	list, ok := actionsRaw.([]any)
	if !ok {
		return nil, errors.New("actions must be an array")
	}
	if len(list) == 0 {
		return nil, nil
	}

	out := make([]automationAction, 0, len(list))
	for _, item := range list {
		im, isMap := item.(map[string]any)
		if !isMap {
			return nil, errors.New("action must be an object")
		}
		typ, _ := im["type"].(string)
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ == "" {
			return nil, errors.New("action missing type")
		}
		ch, _ := im["channel_id"].(string)
		uid, _ := im["user_id"].(string)
		out = append(out, automationAction{
			Type:      typ,
			ChannelID: strings.TrimSpace(ch),
			UserID:    strings.TrimSpace(uid),
			Message:   im["message"],
		})
	}
	return out, nil
}

func (p *pluginAutomation) executeAutomationActions(
	ctx context.Context,
	pluginID string,
	perms permissions.Permissions,
	trigger plugins.Payload,
	actions []automationAction,
) {
	if p == nil || p.bot == nil || p.bot.client == nil {
		return
	}

	for _, a := range actions {
		if !p.allowAutomation(pluginID, trigger, a) {
			p.logger.WarnContext(
				ctx,
				"plugin automation rate-limited",
				slog.String("plugin", pluginID),
				slog.String("type", a.Type),
			)
			continue
		}

		p.executeAutomationAction(ctx, pluginID, perms, trigger, a)
	}
}

func (p *pluginAutomation) allowAutomation(pluginID string, trigger plugins.Payload, a automationAction) bool {
	if p == nil || p.limiter == nil {
		return false
	}
	key := strings.TrimSpace(pluginID) + ":" + strings.TrimSpace(trigger.GuildID) + ":" + strings.TrimSpace(a.Type)
	return p.limiter.Allow(key, time.Now())
}

func (p *pluginAutomation) executeAutomationAction(
	ctx context.Context,
	pluginID string,
	perms permissions.Permissions,
	trigger plugins.Payload,
	a automationAction,
) {
	switch a.Type {
	case "send_channel":
		p.executeSendChannel(ctx, pluginID, perms, trigger, a)
	case "send_dm":
		p.executeSendDM(ctx, pluginID, perms, trigger, a)
	default:
		p.logger.WarnContext(
			ctx,
			"plugin automation unsupported action",
			slog.String("plugin", pluginID),
			slog.String("type", a.Type),
		)
	}
}

func (p *pluginAutomation) executeSendChannel(
	ctx context.Context,
	pluginID string,
	perms permissions.Permissions,
	trigger plugins.Payload,
	a automationAction,
) {
	if !perms.Discord.SendChannel {
		p.logger.WarnContext(ctx, "plugin send_channel denied", slog.String("plugin", pluginID))
		return
	}

	channelID := strings.TrimSpace(a.ChannelID)
	if channelID == "" {
		channelID = strings.TrimSpace(trigger.ChannelID)
	}
	if channelID == "" {
		p.logger.WarnContext(ctx, "plugin send_channel missing channel_id", slog.String("plugin", pluginID))
		return
	}

	chID, err := snowflake.Parse(channelID)
	if err != nil {
		p.logger.WarnContext(ctx, "plugin send_channel invalid channel_id", slog.String("plugin", pluginID))
		return
	}

	msg, err := parseAutomationMessage(pluginID, a.Message)
	if err != nil {
		p.logger.WarnContext(
			ctx,
			"plugin send_channel invalid message",
			slog.String("plugin", pluginID),
			slog.String("err", err.Error()),
		)
		return
	}

	if _, sendErr := p.bot.client.Rest.CreateMessage(chID, msg); sendErr != nil {
		p.logger.WarnContext(
			ctx,
			"plugin send_channel failed",
			slog.String("plugin", pluginID),
			slog.String("err", sendErr.Error()),
		)
	}
}

func (p *pluginAutomation) executeSendDM(
	ctx context.Context,
	pluginID string,
	perms permissions.Permissions,
	trigger plugins.Payload,
	a automationAction,
) {
	if !perms.Discord.SendDM {
		p.logger.WarnContext(ctx, "plugin send_dm denied", slog.String("plugin", pluginID))
		return
	}

	userID := strings.TrimSpace(a.UserID)
	if userID == "" {
		userID = strings.TrimSpace(trigger.UserID)
	}
	if userID == "" {
		p.logger.WarnContext(ctx, "plugin send_dm missing user_id", slog.String("plugin", pluginID))
		return
	}

	uid, err := snowflake.Parse(userID)
	if err != nil {
		p.logger.WarnContext(ctx, "plugin send_dm invalid user_id", slog.String("plugin", pluginID))
		return
	}

	msg, err := parseAutomationMessage(pluginID, a.Message)
	if err != nil {
		p.logger.WarnContext(
			ctx,
			"plugin send_dm invalid message",
			slog.String("plugin", pluginID),
			slog.String("err", err.Error()),
		)
		return
	}

	dmID, dmErr := p.bot.ensureDMChannel(ctx, uint64(uid))
	if dmErr != nil {
		p.logger.WarnContext(
			ctx,
			"plugin send_dm failed to create dm",
			slog.String("plugin", pluginID),
			slog.String("err", dmErr.Error()),
		)
		return
	}

	if _, sendErr := p.bot.client.Rest.CreateMessage(snowflake.ID(dmID), msg); sendErr != nil {
		p.logger.WarnContext(
			ctx,
			"plugin send_dm failed",
			slog.String("plugin", pluginID),
			slog.String("err", sendErr.Error()),
		)
	}
}

func parseAutomationMessage(pluginID string, raw any) (discord.MessageCreate, error) {
	switch v := raw.(type) {
	case nil:
		return discord.MessageCreate{}, errors.New("missing message")
	case string:
		act, err := pluginActionFromString(pluginID, v, false, pluginResponseSlash)
		if err != nil {
			return discord.MessageCreate{}, err
		}
		if act.Kind != pluginActionMessage {
			return discord.MessageCreate{}, errors.New("unsupported message type")
		}
		if emptyMessageCreate(act.Create) {
			return discord.MessageCreate{}, errors.New("message is empty")
		}
		return act.Create, nil
	case map[string]any:
		act, err := pluginActionFromMap(pluginID, v, false, pluginResponseSlash)
		if err == nil {
			if act.Kind != pluginActionMessage {
				return discord.MessageCreate{}, errors.New("unsupported message type")
			}
			if emptyMessageCreate(act.Create) {
				return discord.MessageCreate{}, errors.New("message is empty")
			}
			return act.Create, nil
		}
		msg, err := parseMessageCreate(pluginID, v)
		if err != nil {
			return discord.MessageCreate{}, err
		}
		if emptyMessageCreate(msg) {
			return discord.MessageCreate{}, errors.New("message is empty")
		}
		return msg, nil
	default:
		return discord.MessageCreate{}, fmt.Errorf("unsupported message type %T", raw)
	}
}

func emptyMessageCreate(msg discord.MessageCreate) bool {
	return strings.TrimSpace(msg.Content) == "" && len(msg.Embeds) == 0 && len(msg.Components) == 0
}

type tokenBucketLimiter struct {
	mu sync.Mutex

	ratePerSec float64
	burst      float64
	state      map[string]tokenBucket
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func newTokenBucketLimiter(ratePerSec float64, burst int) *tokenBucketLimiter {
	if ratePerSec <= 0 {
		ratePerSec = defaultPluginAutomationRatePerSec
	}
	if burst <= 0 {
		burst = defaultPluginAutomationBurst
	}
	return &tokenBucketLimiter{
		ratePerSec: ratePerSec,
		burst:      float64(burst),
		state:      map[string]tokenBucket{},
	}
}

func (l *tokenBucketLimiter) Allow(key string, now time.Time) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.state[key]
	if b.last.IsZero() {
		b = tokenBucket{tokens: l.burst - 1, last: now}
		l.state[key] = b
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	b.tokens = minFloat(l.burst, b.tokens+elapsed*l.ratePerSec)
	b.last = now
	if b.tokens < 1 {
		l.state[key] = b
		return false
	}
	b.tokens--
	l.state[key] = b
	return true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
