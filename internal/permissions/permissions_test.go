package permissions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
)

func TestEffective(t *testing.T) {
	t.Parallel()

	req := permissions.Permissions{Storage: permissions.StoragePermissions{KV: true}}
	grant := permissions.Permissions{Storage: permissions.StoragePermissions{KV: false}}
	eff := permissions.Effective(req, grant)
	if eff.Storage.KV {
		t.Fatalf("expected kv denied")
	}

	grant.Storage.KV = true
	eff = permissions.Effective(req, grant)
	if !eff.Storage.KV {
		t.Fatalf("expected kv allowed")
	}

	req.Storage.KV = false
	eff = permissions.Effective(req, grant)
	if eff.Storage.KV {
		t.Fatalf("expected kv denied when not requested")
	}

	req = permissions.Permissions{
		Discord: permissions.DiscordPermissions{SendChannel: true, SendDM: true},
		Network: permissions.NetworkPermissions{HTTP: true},
		Automation: permissions.AutomationPermissions{
			Jobs: true,
			Events: permissions.AutomationEventPermissions{
				MemberJoinLeave: true,
				Moderation:      true,
			},
		},
	}
	grant = permissions.Permissions{
		Discord: permissions.DiscordPermissions{SendChannel: true, SendDM: false},
		Network: permissions.NetworkPermissions{HTTP: false},
		Automation: permissions.AutomationPermissions{
			Jobs: false,
			Events: permissions.AutomationEventPermissions{
				MemberJoinLeave: true,
				Moderation:      false,
			},
		},
	}
	eff = permissions.Effective(req, grant)
	if !eff.Discord.SendChannel {
		t.Fatalf("expected send_channel allowed")
	}
	if eff.Discord.SendDM {
		t.Fatalf("expected send_dm denied")
	}
	if eff.Network.HTTP {
		t.Fatalf("expected http denied")
	}
	if eff.Automation.Jobs {
		t.Fatalf("expected jobs denied")
	}
	if !eff.Automation.Events.MemberJoinLeave {
		t.Fatalf("expected member_join_leave allowed")
	}
	if eff.Automation.Events.Moderation {
		t.Fatalf("expected moderation denied")
	}
}

func TestPolicyGranted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")

	if err := os.WriteFile(p, []byte(`{
  "defaults": { "storage": { "kv": false }, "discord": { "send_channel": false } },
  "plugins": {
    "a": { "storage": { "kv": true }, "discord": { "send_channel": true }, "network": { "http": true } }
  }
}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	pol, err := permissions.LoadPolicyFile(p)
	if err != nil {
		t.Fatalf("LoadPolicyFile: %v", err)
	}

	if pol.Granted("x").Storage.KV {
		t.Fatalf("expected default kv denied")
	}
	if !pol.Granted("a").Storage.KV {
		t.Fatalf("expected plugin override kv allowed")
	}
	if !pol.Granted("a").Discord.SendChannel {
		t.Fatalf("expected plugin override send_channel allowed")
	}
	if !pol.Granted("a").Network.HTTP {
		t.Fatalf("expected plugin override http allowed")
	}
}
