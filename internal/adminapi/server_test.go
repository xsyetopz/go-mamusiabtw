package adminapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xsyetopz/go-mamusiabtw/internal/buildinfo"
	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/ops"
)

func optionalUint64(value uint64) *uint64 {
	return &value
}

type fakeOAuthClient struct{}

func (fakeOAuthClient) ExchangeCode(context.Context, string) (OAuthToken, error) {
	return OAuthToken{AccessToken: "token", TokenType: "Bearer", Scope: "identify guilds"}, nil
}

func (fakeOAuthClient) FetchUser(context.Context, string) (OAuthUser, error) {
	return OAuthUser{
		ID:         "42",
		Username:   "owner",
		GlobalName: "Owner",
		Avatar:     "abc",
	}, nil
}

func (fakeOAuthClient) FetchGuilds(context.Context, string) ([]OAuthGuild, error) {
	return []OAuthGuild{
		{
			ID:          "1",
			Name:        "Guild",
			Owner:       true,
			Permissions: OAuthPermissions("8"),
		},
	}, nil
}

func TestHandleMeRequiresSession(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:          "127.0.0.1:0",
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service:       Service{},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{
				Configured:      true,
				Resolved:        true,
				Source:          "discord",
				EffectiveUserID: optionalUint64(42),
			}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	server.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestHandleModulesWithSession(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service: Service{
			Config: config.Config{},
			BuildInfo: func() buildinfo.Info {
				return buildinfo.Info{Version: "test"}
			},
			Snapshot: func() ops.Snapshot {
				return ops.Snapshot{Ready: true}
			},
		},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{
				Configured:      true,
				Resolved:        true,
				Source:          "discord",
				EffectiveUserID: optionalUint64(42),
			}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server.putSession(session{
		ID:        "session-token",
		UserID:    42,
		Username:  "owner",
		Name:      "Owner",
		CSRFToken: "csrf",
		IsOwner:   true,
		ExpiresAt: 4102444800,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/owner/status", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: server.signCookieValue(sessionCookieName, "session-token"),
	})
	server.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := payload["snapshot"]; !ok {
		t.Fatalf("expected snapshot in response")
	}
}

func TestHandleLoginReturns503WhenAuthNotConfigured(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:          "127.0.0.1:0",
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service:       Service{},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "",
		ClientSecret:  "",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{Resolved: false, Source: "unresolved"}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	server.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSetupWithoutSession(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service: Service{
			Config: config.Config{
				AdminAddr:               "127.0.0.1:8081",
				DashboardAppOrigin:      "http://127.0.0.1:5173",
				DashboardClientID:       "client-id",
				DashboardClientSecret:   "client-secret",
				DashboardRedirectURL:    "http://127.0.0.1:8081/api/auth/callback",
				DashboardSessionSecret:  strings.Repeat("x", 32),
				DashboardSigningKeyID:   "official",
				DashboardSigningKeyFile: "/tmp/key",
				OwnerUserID:             optionalUint64(42),
			},
			OwnerStatus: func() OwnerStatus {
				return OwnerStatus{
					Configured:      true,
					Resolved:        true,
					Source:          "discord",
					EffectiveUserID: optionalUint64(42),
				}
			},
		},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{
				Configured:      true,
				Resolved:        true,
				Source:          "discord",
				EffectiveUserID: optionalUint64(42),
			}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/setup", nil)
	server.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got, _ := payload["auth_configured"].(bool); !got {
		t.Fatalf("expected auth_configured=true, got %#v", payload["auth_configured"])
	}
	if got, _ := payload["login_ready"].(bool); !got {
		t.Fatalf("expected login_ready=true, got %#v", payload["login_ready"])
	}
	if got, _ := payload["owner_configured"].(bool); !got {
		t.Fatalf("expected owner_configured=true, got %#v", payload["owner_configured"])
	}
	if got, _ := payload["owner_resolved"].(bool); !got {
		t.Fatalf("expected owner_resolved=true, got %#v", payload["owner_resolved"])
	}
	if got, _ := payload["owner_source"].(string); got != "discord" {
		t.Fatalf("expected owner_source=discord, got %#v", payload["owner_source"])
	}
}

func TestCORSAllowsLocalhostWhenConfiguredFor127(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:          "127.0.0.1:0",
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service:       Service{},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{Resolved: false, Source: "unresolved"}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/setup", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	server.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allow-origin=%q, got %q", "http://localhost:5173", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected allow-credentials=true, got %q", got)
	}
}

func TestCORSDoesNotAllowUnknownOrigin(t *testing.T) {
	t.Parallel()

	server, err := New(Options{
		Addr:          "127.0.0.1:0",
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service:       Service{},
		AppOrigin:     "http://127.0.0.1:5173",
		SessionSecret: strings.Repeat("x", 32),
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "http://127.0.0.1:8081/api/auth/callback",
		OwnerStatus: func() OwnerStatus {
			return OwnerStatus{Resolved: false, Source: "unresolved"}
		},
		OAuthClient: fakeOAuthClient{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/setup", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	server.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected allow-origin empty, got %q", got)
	}
}
