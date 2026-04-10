package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorCommand_PrintsSigningDiagnostics(t *testing.T) {
	resetMainEnv(t)
	t.Setenv("DISCORD_TOKEN", "discord-token")
	t.Setenv("MAMUSIABTW_PROD_MODE", "1")
	t.Setenv("MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS", "0")
	t.Setenv("MAMUSIABTW_TRUSTED_KEYS_FILE", "/tmp/trusted_keys.json")
	t.Setenv("MAMUSIABTW_DASHBOARD_SIGNING_KEY_ID", "owner")
	t.Setenv("MAMUSIABTW_DASHBOARD_SIGNING_KEY_FILE", "./data/keys/owner.key")
	t.Setenv("MAMUSIABTW_LOADED_ENV_FILE", ".env.prod")
	t.Setenv("MAMUSIABTW_LOADED_ENV_SOURCE", "working_dir")

	output := captureStdout(t, func() {
		if code := runDoctorCommand(nil); code != 0 {
			t.Fatalf("runDoctorCommand returned %d", code)
		}
	})

	for _, want := range []string{
		"env_file_loaded: .env.prod",
		"discord_token: true",
		"prod_mode: true",
		"allow_unsigned_plugins: false",
		"trusted_keys_file: /tmp/trusted_keys.json",
		"trusted_keys_file_exists: false",
		"trusted_keys_count_file: 0",
		"dashboard_signing_configured: true",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q\n%s", want, output)
		}
	}
}

func TestRunGenSigningKeyCommand_CreatesFiles(t *testing.T) {
	resetMainEnv(t)

	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "data", "keys", "owner.key")
	trustedKeysPath := filepath.Join(dir, "config", "trusted_keys.json")

	output := captureStdout(t, func() {
		code := runGenSigningKeyCommand([]string{
			"--key-id", "owner",
			"--private-key-file", privateKeyPath,
			"--trusted-keys-file", trustedKeysPath,
		})
		if code != 0 {
			t.Fatalf("runGenSigningKeyCommand returned %d", code)
		}
	})

	if _, err := os.Stat(privateKeyPath); err != nil {
		t.Fatalf("private key missing: %v", err)
	}
	if _, err := os.Stat(trustedKeysPath); err != nil {
		t.Fatalf("trusted keys file missing: %v", err)
	}

	for _, want := range []string{
		"key_id: owner",
		"private_key_file: " + privateKeyPath,
		"trusted_keys_file: " + trustedKeysPath,
		"public_key_b64: ",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("keygen output missing %q\n%s", want, output)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()

	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()
	fn()
	_ = w.Close()

	bytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(bytes)
}

func resetMainEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"DISCORD_TOKEN",
		"MAMUSIABTW_PROD_MODE",
		"MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS",
		"MAMUSIABTW_TRUSTED_KEYS_FILE",
		"MAMUSIABTW_DASHBOARD_SIGNING_KEY_ID",
		"MAMUSIABTW_DASHBOARD_SIGNING_KEY_FILE",
		"MAMUSIABTW_DASHBOARD_CLIENT_ID",
		"MAMUSIABTW_DASHBOARD_CLIENT_SECRET",
		"MAMUSIABTW_DASHBOARD_SESSION_SECRET",
		"MAMUSIABTW_ADMIN_ADDR",
		"MAMUSIABTW_LOADED_ENV_FILE",
		"MAMUSIABTW_LOADED_ENV_SOURCE",
	} {
		t.Setenv(name, "")
	}
}
