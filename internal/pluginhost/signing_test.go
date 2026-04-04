package pluginhost_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/pluginhost"
	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

func TestReadTrustedKeysFileAndLoadTrustedKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "trusted_keys.json")

	filePublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey(file): %v", err)
	}
	storePublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey(store): %v", err)
	}

	payload := pluginhost.TrustedKeys{
		Keys: []pluginhost.TrustedKey{
			{
				KeyID:        "file-key",
				PublicKeyB64: base64.StdEncoding.EncodeToString(filePublicKey),
			},
		},
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if err := os.WriteFile(filePath, bytes, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	readKeys, err := pluginhost.ReadTrustedKeysFile(filePath)
	if err != nil {
		t.Fatalf("ReadTrustedKeysFile: %v", err)
	}
	if !reflect.DeepEqual(readKeys["file-key"], filePublicKey) {
		t.Fatalf("unexpected file key bytes")
	}

	loadedKeys, err := pluginhost.LoadTrustedKeys(
		context.Background(),
		filePath,
		trustedSignerSourceStub{
			store: trustedSignerStoreStub{
				signers: []store.TrustedSigner{
					{
						KeyID:        "store-key",
						PublicKeyB64: base64.StdEncoding.EncodeToString(storePublicKey),
						AddedAt:      time.Unix(1700000000, 0).UTC(),
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("LoadTrustedKeys: %v", err)
	}
	if len(loadedKeys) != 2 {
		t.Fatalf("unexpected loaded key count: %d", len(loadedKeys))
	}
	if !reflect.DeepEqual(loadedKeys["store-key"], storePublicKey) {
		t.Fatalf("unexpected store key bytes")
	}
}

func TestVerifyDirSignatureAndHashDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "plugin.json"), []byte(`{"id":"example"}`))
	mustWriteFile(t, filepath.Join(dir, "plugin.lua"), []byte(`return "ok"`))

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	hashBefore, err := pluginhost.HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir(before): %v", err)
	}

	mustWriteFile(t, filepath.Join(dir, "signature.json"), []byte(`{"ignored":true}`))

	hashAfter, err := pluginhost.HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir(after): %v", err)
	}
	if hashBefore != hashAfter {
		t.Fatalf("expected signature.json to be ignored by HashDir")
	}

	signatureBytes := ed25519.Sign(privateKey, hashBefore[:])
	signature := pluginhost.Signature{
		KeyID:        "key-1",
		HashB64:      base64.StdEncoding.EncodeToString(hashBefore[:]),
		SignatureB64: base64.StdEncoding.EncodeToString(signatureBytes),
		Algorithm:    "ed25519-sha256",
	}

	keys := map[string]ed25519.PublicKey{"key-1": publicKey}
	if err := pluginhost.VerifyDirSignature(dir, signature, keys); err != nil {
		t.Fatalf("VerifyDirSignature(valid): %v", err)
	}

	if err := pluginhost.VerifyDirSignature(dir, signature, map[string]ed25519.PublicKey{}); err == nil {
		t.Fatalf("expected unknown signer error")
	}

	signature.HashB64 = base64.StdEncoding.EncodeToString([]byte("bad-hash"))
	if err := pluginhost.VerifyDirSignature(dir, signature, keys); err == nil {
		t.Fatalf("expected hash mismatch error")
	}
}

func TestSignDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "plugin.lua"), []byte("return {}"))

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	sig, publicKey, err := pluginhost.SignDir(dir, "key-1", privateKey)
	if err != nil {
		t.Fatalf("SignDir: %v", err)
	}

	if sig.KeyID != "key-1" {
		t.Fatalf("unexpected key id: %q", sig.KeyID)
	}
	if sig.Algorithm != "ed25519-sha256" {
		t.Fatalf("unexpected algorithm: %q", sig.Algorithm)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("unexpected public key size: %d", len(publicKey))
	}
	if err := pluginhost.VerifyDirSignature(dir, sig, map[string]ed25519.PublicKey{"key-1": publicKey}); err != nil {
		t.Fatalf("VerifyDirSignature: %v", err)
	}
}

func mustWriteFile(t *testing.T, path string, bytes []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

type trustedSignerSourceStub struct {
	store trustedSignerStoreStub
}

func (s trustedSignerSourceStub) TrustedSigners() store.TrustedSignerStore {
	return s.store
}

type trustedSignerStoreStub struct {
	signers []store.TrustedSigner
	err     error
}

func (s trustedSignerStoreStub) ListTrustedSigners(context.Context) ([]store.TrustedSigner, error) {
	return s.signers, s.err
}

func (trustedSignerStoreStub) PutTrustedSigner(context.Context, store.TrustedSigner) error {
	return nil
}

func (trustedSignerStoreStub) DeleteTrustedSigner(context.Context, string) error {
	return nil
}
