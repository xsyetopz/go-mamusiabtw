package permissions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xsyetopz/imotherbtw/internal/permissions"
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
}

func TestPolicyGranted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")

	if err := os.WriteFile(p, []byte(`{
  "defaults": { "storage": { "kv": false } },
  "plugins": {
    "a": { "storage": { "kv": true } }
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
}
