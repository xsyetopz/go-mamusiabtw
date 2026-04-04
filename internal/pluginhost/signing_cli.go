package pluginhost

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const SignatureSchemaURL = "https://raw.githubusercontent.com/xsyetopz/go-mamusiabtw/refs/heads/main/schemas/signature.schema.v1.json"

func DecodeEd25519PrivateKey(raw string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	switch len(decoded) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(decoded), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	default:
		return nil, fmt.Errorf("unexpected private key size %d", len(decoded))
	}
}

func ReadEd25519PrivateKeyFile(path string) (ed25519.PrivateKey, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(bytes))
	if raw == "" {
		return nil, errors.New("private key file is empty")
	}

	if strings.HasPrefix(raw, "{") {
		var payload struct {
			PrivateKeyB64 string `json:"private_key_b64"`
		}
		if err := json.Unmarshal(bytes, &payload); err != nil {
			return nil, fmt.Errorf("parse private key file: %w", err)
		}
		raw = strings.TrimSpace(payload.PrivateKeyB64)
	}

	return DecodeEd25519PrivateKey(raw)
}

func SignDir(dir string, keyID string, privateKey ed25519.PrivateKey) (Signature, ed25519.PublicKey, error) {
	if strings.TrimSpace(dir) == "" {
		return Signature{}, nil, errors.New("plugin dir is required")
	}
	if strings.TrimSpace(keyID) == "" {
		return Signature{}, nil, errors.New("key id is required")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return Signature{}, nil, errors.New("invalid ed25519 private key")
	}

	hash, err := HashDir(dir)
	if err != nil {
		return Signature{}, nil, err
	}

	sig := Signature{
		KeyID:        strings.TrimSpace(keyID),
		HashB64:      base64.StdEncoding.EncodeToString(hash[:]),
		SignatureB64: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, hash[:])),
		Algorithm:    "ed25519-sha256",
	}
	return sig, privateKey.Public().(ed25519.PublicKey), nil
}
