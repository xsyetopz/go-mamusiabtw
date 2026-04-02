package botengine

import (
	"strings"
	"time"

	"github.com/xsuetopz/go-mamusiabtw/internal/plugins"
)

func (b *Bot) commandCooldown(cmdName string) time.Duration {
	if b == nil {
		return 0
	}
	key := strings.ToLower(strings.TrimSpace(cmdName))
	if key == "" {
		return 0
	}

	root := key
	if idx := strings.IndexByte(root, ':'); idx >= 0 {
		root = root[:idx]
	}

	if _, ok := b.slashBypass[root]; ok {
		return 0
	}

	if d, ok := b.slashCooldownOverrides[key]; ok {
		return d
	}
	if root != key {
		if d, ok := b.slashCooldownOverrides[root]; ok {
			return d
		}
	}

	if b.slashCooldown <= 0 {
		return 0
	}
	return b.slashCooldown
}

func (b *Bot) componentCooldown(_ string) time.Duration {
	if b == nil || b.componentCooldownDur <= 0 {
		return 0
	}
	return b.componentCooldownDur
}

func (b *Bot) modalCooldown(_ string) time.Duration {
	if b == nil || b.modalCooldownDur <= 0 {
		return 0
	}
	return b.modalCooldownDur
}

func cooldownSecs(remaining time.Duration) int {
	secs := int(remaining.Round(time.Second).Seconds())
	return max(1, secs)
}

func componentCooldownKey(customID string) string {
	cid := strings.TrimSpace(customID)
	if cid == "" {
		return "component"
	}
	if pid, _, ok := plugins.ParseCustomID(cid); ok {
		return "component:" + pid
	}
	if strings.HasPrefix(cid, "mamusiabtw:") {
		return "component:mamusiabtw"
	}
	return "component:other"
}

func modalCooldownKey(customID string) string {
	cid := strings.TrimSpace(customID)
	if cid == "" {
		return "modal"
	}
	if pid, _, ok := plugins.ParseCustomID(cid); ok {
		return "modal:" + pid
	}
	if strings.HasPrefix(cid, "mamusiabtw:") {
		return "modal:mamusiabtw"
	}
	return "modal:other"
}
