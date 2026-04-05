package adminapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/config"
	"github.com/xsyetopz/go-mamusiabtw/internal/permissions"
)

const (
	sessionCookieName = "mamusiabtw_admin_session"
	stateCookieName   = "mamusiabtw_admin_state"
	sessionTTL        = 12 * time.Hour
)

type OAuthUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

type OAuthClient interface {
	ExchangeCode(ctx context.Context, code string) (OAuthToken, error)
	FetchUser(ctx context.Context, accessToken string) (OAuthUser, error)
	FetchGuilds(ctx context.Context, accessToken string) ([]OAuthGuild, error)
}

type Options struct {
	Addr           string
	Logger         *slog.Logger
	Service        Service
	AppOrigin      string
	OwnerAppOrigin string
	SessionSecret  string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	OwnerStatus    func() OwnerStatus
	OAuthClient    OAuthClient
}

type Server struct {
	logger *slog.Logger
	addr   string
	svc    Service

	appOrigin      string
	ownerAppOrigin string
	corsOrigins    map[string]struct{}
	clientID       string
	clientSecret   string
	ownerStatus    func() OwnerStatus
	oauth          OAuthClient
	secret         []byte
	cookieSecure   bool
	cookieSameSite http.SameSite

	sessionsMu sync.Mutex
	sessions   map[string]session

	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
}

type session struct {
	ID          string
	UserID      uint64
	Username    string
	Name        string
	AvatarURL   string
	CSRFToken   string
	AccessToken string
	IsOwner     bool
	ExpiresAt   int64
}

func New(opts Options) (*Server, error) {
	if strings.TrimSpace(opts.Addr) == "" {
		return nil, nil
	}
	if opts.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if opts.OAuthClient == nil {
		opts.OAuthClient = NewDiscordOAuthClient(opts.ClientID, opts.ClientSecret, opts.RedirectURL)
	}
	cookieSecure, cookieSameSite := cookiePolicy(opts.AppOrigin, opts.RedirectURL)
	return &Server{
		logger:         opts.Logger.With(slog.String("component", "admin_api")),
		addr:           strings.TrimSpace(opts.Addr),
		svc:            opts.Service,
		appOrigin:      strings.TrimSpace(opts.AppOrigin),
		ownerAppOrigin: strings.TrimSpace(opts.OwnerAppOrigin),
		corsOrigins:    buildAllowedCORSOrigins(opts.AppOrigin),
		clientID:       strings.TrimSpace(opts.ClientID),
		clientSecret:   strings.TrimSpace(opts.ClientSecret),
		ownerStatus:    opts.OwnerStatus,
		oauth:          opts.OAuthClient,
		secret:         []byte(opts.SessionSecret),
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
		sessions:       map[string]session{},
	}, nil
}

func (s *Server) Start() error {
	if s == nil || s.addr == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.listener = listener
	s.server = httpServer
	go func() {
		err := httpServer.Serve(listener)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}
		s.logger.Error("admin server stopped unexpectedly", slog.String("err", err.Error()))
	}()
	s.logger.Info("admin server listening", slog.String("addr", listener.Addr().String()))
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	server := s.server
	s.server = nil
	s.listener = nil
	s.mu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

func (s *Server) Close(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/setup", s.handleSetup)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/callback", s.handleCallback)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/guilds", s.withAuth(s.handleGuilds))
	mux.HandleFunc("/api/guilds/dashboard", s.withAuth(s.handleGuildDashboard))
	mux.HandleFunc("/api/guilds/config", s.withAuth(s.withCSRF(s.handleGuildConfig)))
	mux.HandleFunc("/api/guilds/channels", s.withAuth(s.handleGuildChannels))
	mux.HandleFunc("/api/guilds/roles", s.withAuth(s.handleGuildRoles))
	mux.HandleFunc("/api/guilds/members", s.withAuth(s.handleGuildMembers))
	mux.HandleFunc("/api/guilds/emojis", s.withAuth(s.handleGuildEmojis))
	mux.HandleFunc("/api/guilds/stickers", s.withAuth(s.handleGuildStickers))
	mux.HandleFunc("/api/guilds/moderation/warnings", s.withAuth(s.handleGuildWarnings))
	mux.HandleFunc("/api/guilds/moderation/warn", s.withAuth(s.withCSRF(s.handleGuildWarn)))
	mux.HandleFunc("/api/guilds/moderation/unwarn", s.withAuth(s.withCSRF(s.handleGuildUnwarn)))
	mux.HandleFunc("/api/guilds/manager/slowmode", s.withAuth(s.withCSRF(s.handleGuildSlowmode)))
	mux.HandleFunc("/api/guilds/manager/nick", s.withAuth(s.withCSRF(s.handleGuildNickname)))
	mux.HandleFunc("/api/guilds/manager/roles/create", s.withAuth(s.withCSRF(s.handleGuildRoleCreate)))
	mux.HandleFunc("/api/guilds/manager/roles/edit", s.withAuth(s.withCSRF(s.handleGuildRoleEdit)))
	mux.HandleFunc("/api/guilds/manager/roles/delete", s.withAuth(s.withCSRF(s.handleGuildRoleDelete)))
	mux.HandleFunc("/api/guilds/manager/roles/member", s.withAuth(s.withCSRF(s.handleGuildRoleMember)))
	mux.HandleFunc("/api/guilds/manager/purge", s.withAuth(s.withCSRF(s.handleGuildPurge)))
	mux.HandleFunc("/api/guilds/manager/emojis/create", s.withAuth(s.withCSRF(s.handleGuildEmojiCreate)))
	mux.HandleFunc("/api/guilds/manager/emojis/edit", s.withAuth(s.withCSRF(s.handleGuildEmojiEdit)))
	mux.HandleFunc("/api/guilds/manager/emojis/delete", s.withAuth(s.withCSRF(s.handleGuildEmojiDelete)))
	mux.HandleFunc("/api/guilds/manager/stickers/create", s.withAuth(s.withCSRF(s.handleGuildStickerCreate)))
	mux.HandleFunc("/api/guilds/manager/stickers/edit", s.withAuth(s.withCSRF(s.handleGuildStickerEdit)))
	mux.HandleFunc("/api/guilds/manager/stickers/delete", s.withAuth(s.withCSRF(s.handleGuildStickerDelete)))
	mux.HandleFunc("/api/install/start", s.withAuth(s.handleInstallStart))
	mux.HandleFunc("/api/install/callback", s.withAuth(s.handleInstallCallback))

	mux.HandleFunc("/api/owner/status", s.withOwner(s.handleStatus))
	mux.HandleFunc("/api/owner/modules", s.withOwner(s.handleModules))
	mux.HandleFunc("/api/owner/modules/set", s.withOwner(s.withCSRF(s.handleSetModule)))
	mux.HandleFunc("/api/owner/modules/reset", s.withOwner(s.withCSRF(s.handleResetModule)))
	mux.HandleFunc("/api/owner/modules/reload", s.withOwner(s.withCSRF(s.handleReloadModules)))

	mux.HandleFunc("/api/owner/plugins", s.withOwner(s.handlePlugins))
	mux.HandleFunc("/api/owner/plugins/reload", s.withOwner(s.withCSRF(s.handleReloadPlugins)))
	mux.HandleFunc("/api/owner/plugins/scaffold", s.withOwner(s.withCSRF(s.handleScaffoldPlugin)))
	mux.HandleFunc("/api/owner/plugins/sign", s.withOwner(s.withCSRF(s.handleSignPlugin)))

	mux.HandleFunc("/api/owner/config/modules", s.withOwner(s.handleModulesConfig))
	mux.HandleFunc("/api/owner/config/permissions", s.withOwner(s.handlePermissionsConfig))
	mux.HandleFunc("/api/owner/config/trusted-keys", s.withOwner(s.handleTrustedKeys))

	mux.HandleFunc("/api/owner/migrations/status", s.withOwner(s.handleMigrationStatus))
	mux.HandleFunc("/api/owner/migrations/backup", s.withOwner(s.withCSRF(s.handleMigrationBackup)))
	mux.HandleFunc("/api/owner/migrations/up", s.withOwner(s.withCSRF(s.handleMigrationUp)))

	return s.withCORS(mux)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Setup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && s.isAllowedCORSOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAllowedCORSOrigin(origin string) bool {
	if s == nil {
		return false
	}
	origin = normalizeOrigin(origin)
	if origin == "" {
		return false
	}
	if _, ok := s.corsOrigins[origin]; ok {
		return true
	}
	return false
}

func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	return raw
}

func canonicalLoopbackOrigin(origin string) string {
	origin = normalizeOrigin(origin)
	if origin == "" {
		return ""
	}

	u, err := url.Parse(origin)
	if err != nil {
		return origin
	}
	if strings.TrimSpace(u.Scheme) == "" {
		return origin
	}

	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return origin
	}

	// Prefer localhost for redirects. It "just works" in browsers even when the
	// dev server is bound to localhost and the config uses 127.0.0.1.
	if host != "127.0.0.1" && host != "::1" {
		return origin
	}

	port := strings.TrimSpace(u.Port())
	targetHost := "localhost"
	if port != "" {
		targetHost = net.JoinHostPort(targetHost, port)
	}
	u.Host = targetHost
	return normalizeOrigin(u.String())
}

func buildAllowedCORSOrigins(appOrigin string) map[string]struct{} {
	appOrigin = normalizeOrigin(appOrigin)
	if appOrigin == "" {
		// Local dev defaults: allow Vite's default origin even if the config is
		// missing. Production remains strict via config validation.
		return map[string]struct{}{
			"http://localhost:5173":   {},
			"http://127.0.0.1:5173":   {},
			"http://[::1]:5173":       {},
		}
	}

	out := map[string]struct{}{appOrigin: {}}

	u, err := url.Parse(appOrigin)
	if err != nil {
		return out
	}

	host := strings.TrimSpace(u.Hostname())
	port := strings.TrimSpace(u.Port())
	scheme := strings.TrimSpace(u.Scheme)
	if host == "" || scheme == "" {
		return out
	}

	// Reduce dev friction: treat localhost / 127.0.0.1 / ::1 as equivalent for the
	// same port when the configured origin is loopback. This avoids confusing CORS
	// failures when Vite uses localhost but the config uses 127.0.0.1.
	isLoopback := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if !isLoopback {
		return out
	}

	variants := []string{"localhost", "127.0.0.1", "::1"}
	for _, v := range variants {
		if v == host {
			continue
		}
		targetHost := v
		if port != "" {
			targetHost = net.JoinHostPort(v, port)
		}
		origin := scheme + "://" + targetHost
		out[normalizeOrigin(origin)] = struct{}{}
	}

	return out
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.readSession(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r, sess)
	}
}

func (s *Server) withOwner(next func(http.ResponseWriter, *http.Request, session)) http.HandlerFunc {
	return s.withAuth(func(w http.ResponseWriter, r *http.Request, sess session) {
		if !s.isOwnerUser(sess.UserID) {
			writeError(w, http.StatusForbidden, "owner access required")
			return
		}
		next(w, r, sess)
	})
}

func (s *Server) withCSRF(next func(http.ResponseWriter, *http.Request, session)) func(http.ResponseWriter, *http.Request, session) {
	return func(w http.ResponseWriter, r *http.Request, sess session) {
		if r.Method == http.MethodGet {
			next(w, r, sess)
			return
		}
		if subtleTokenCompare(r.Header.Get("X-CSRF-Token"), sess.CSRFToken) == false {
			writeError(w, http.StatusForbidden, "csrf validation failed")
			return
		}
		next(w, r, sess)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authConfigured() {
		writeError(w, http.StatusServiceUnavailable, "dashboard auth is not configured")
		return
	}
	state, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start login")
		return
	}
	http.SetCookie(w, s.cookie(stateCookieName, state, 10*time.Minute, true))

	values := url.Values{}
	values.Set("client_id", s.clientID)
	values.Set("response_type", "code")
	values.Set("scope", "identify guilds")
	values.Set("redirect_uri", s.oauthRedirectURL())
	values.Set("state", state)
	http.Redirect(w, r, "https://discord.com/oauth2/authorize?"+values.Encode(), http.StatusFound)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	queryState := strings.TrimSpace(r.URL.Query().Get("state"))
	if !s.authConfigured() {
		writeError(w, http.StatusServiceUnavailable, "dashboard auth is not configured")
		return
	}
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	state, ok := s.unsignCookieValue(stateCookieName, cookie.Value)
	if !ok || !subtleTokenCompare(state, queryState) {
		writeError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing oauth code")
		return
	}
	token, err := s.oauth.ExchangeCode(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oauth token exchange failed")
		return
	}
	user, err := s.oauth.FetchUser(r.Context(), token.AccessToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oauth user lookup failed")
		return
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(user.ID), 10, 64)
	if err != nil {
		writeError(w, http.StatusForbidden, "invalid oauth user")
		return
	}
	csrfToken, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	displayName := strings.TrimSpace(user.GlobalName)
	if displayName == "" {
		displayName = strings.TrimSpace(user.Username)
	}
	sessionID, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	isOwner := s.isOwnerUser(userID)
	sess := session{
		ID:          sessionID,
		UserID:      userID,
		Username:    strings.TrimSpace(user.Username),
		Name:        displayName,
		AvatarURL:   avatarURL(user),
		CSRFToken:   csrfToken,
		AccessToken: token.AccessToken,
		IsOwner:     isOwner,
		ExpiresAt:   time.Now().Add(sessionTTL).Unix(),
	}
	s.putSession(sess)
	http.SetCookie(w, s.cookie(sessionCookieName, sessionID, sessionTTL, true))
	http.SetCookie(w, s.cookie(stateCookieName, "", -time.Hour, true))
	redirectOrigin := canonicalLoopbackOrigin(s.appOrigin)
	ownerOrigin := canonicalLoopbackOrigin(s.ownerAppOrigin)
	redirectTarget := strings.TrimRight(redirectOrigin, "/") + "/#/servers"
	if isOwner && strings.TrimSpace(s.ownerAppOrigin) != "" {
		redirectTarget = strings.TrimRight(ownerOrigin, "/") + "/#/owner"
	}
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	sess, err := s.readSession(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp := SessionResponse{
		Authenticated: true,
		IsOwner:       s.isOwnerUser(sess.UserID),
		CSRFToken:     sess.CSRFToken,
	}
	resp.User.ID = sess.UserID
	resp.User.Username = sess.Username
	resp.User.Name = sess.Name
	resp.User.AvatarURL = sess.AvatarURL
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if sessionID, ok := s.unsignCookieValue(sessionCookieName, cookie.Value); ok {
			s.deleteSession(sessionID)
		}
	}
	http.SetCookie(w, s.cookie(sessionCookieName, "", -time.Hour, true))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuilds(w http.ResponseWriter, r *http.Request, sess session) {
	guilds, err := s.svc.UserGuilds(r.Context(), sess.AccessToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Guilds []UserGuildSummary `json:"guilds"`
	}{Guilds: guilds})
}

func (s *Server) handleGuildDashboard(w http.ResponseWriter, r *http.Request, sess session) {
	guildIDRaw := strings.TrimSpace(r.URL.Query().Get("guild_id"))
	guildID, err := strconv.ParseUint(guildIDRaw, 10, 64)
	if err != nil || guildID == 0 {
		writeError(w, http.StatusBadRequest, "invalid guild_id")
		return
	}
	dashboard, err := s.svc.GuildDashboard(r.Context(), sess.AccessToken, guildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) handleInstallStart(w http.ResponseWriter, r *http.Request, sess session) {
	guildIDRaw := strings.TrimSpace(r.URL.Query().Get("guild_id"))
	var (
		url string
		err error
	)
	if guildIDRaw == "" {
		url, err = s.svc.InstallURLAnyGuild()
	} else {
		guildID, parseErr := strconv.ParseUint(guildIDRaw, 10, 64)
		if parseErr != nil || guildID == 0 {
			writeError(w, http.StatusBadRequest, "invalid guild_id")
			return
		}
		url, err = s.svc.InstallURL(guildID)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	attrs := []any{slog.Uint64("actor_id", sess.UserID)}
	if guildIDRaw != "" {
		if guildID, parseErr := strconv.ParseUint(guildIDRaw, 10, 64); parseErr == nil {
			attrs = append(attrs, slog.Uint64("guild_id", guildID))
		}
	}
	s.logger.Info("bot install started", attrs...)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleInstallCallback(w http.ResponseWriter, r *http.Request, sess session) {
	guildIDRaw := strings.TrimSpace(r.URL.Query().Get("guild_id"))
	guildID, err := strconv.ParseUint(guildIDRaw, 10, 64)
	if err != nil || guildID == 0 {
		writeError(w, http.StatusBadRequest, "invalid guild_id")
		return
	}
	if _, err := s.svc.GuildDashboard(r.Context(), sess.AccessToken, guildID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	http.Redirect(w, r, strings.TrimRight(s.appOrigin, "/")+"/#/servers/"+guildIDRaw, http.StatusFound)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request, _ session) {
	resp, err := s.svc.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleModules(w http.ResponseWriter, _ *http.Request, _ session) {
	writeJSON(w, http.StatusOK, map[string]any{"modules": s.svc.Modules()})
}

func (s *Server) handleSetModule(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		ModuleID string `json:"module_id"`
		Enabled  bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.SetModuleEnabled(r.Context(), req.ModuleID, req.Enabled, sess.UserID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logger.Info("admin module state updated", slog.Uint64("actor_id", sess.UserID), slog.String("module_id", req.ModuleID), slog.Bool("enabled", req.Enabled))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleResetModule(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		ModuleID string `json:"module_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ResetModule(r.Context(), req.ModuleID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logger.Info("admin module reset", slog.Uint64("actor_id", sess.UserID), slog.String("module_id", req.ModuleID))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleReloadModules(w http.ResponseWriter, r *http.Request, sess session) {
	if err := s.svc.ReloadModules(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.logger.Info("admin modules reloaded", slog.Uint64("actor_id", sess.UserID))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request, _ session) {
	plugins, err := s.svc.Plugins()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": plugins})
}

func (s *Server) handleReloadPlugins(w http.ResponseWriter, r *http.Request, sess session) {
	if err := s.svc.ReloadPlugins(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.logger.Info("admin plugins reloaded", slog.Uint64("actor_id", sess.UserID))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleScaffoldPlugin(w http.ResponseWriter, r *http.Request, sess session) {
	var req PluginScaffoldRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := s.svc.ScaffoldPlugin(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logger.Info("admin plugin scaffolded", slog.Uint64("actor_id", sess.UserID), slog.String("plugin_id", resp.ID))
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSignPlugin(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		PluginID string `json:"plugin_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	path, err := s.svc.SignPlugin(req.PluginID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logger.Info("admin plugin signed", slog.Uint64("actor_id", sess.UserID), slog.String("plugin_id", req.PluginID))
	writeJSON(w, http.StatusOK, map[string]any{"signature": path})
}

func (s *Server) handleModulesConfig(w http.ResponseWriter, r *http.Request, sess session) {
	switch r.Method {
	case http.MethodGet:
		file, err := s.svc.LoadModulesConfig()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, file)
	case http.MethodPut:
		var file config.ModulesFile
		if err := decodeJSON(r, &file); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.svc.SaveModulesConfig(file); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Info("admin modules config updated", slog.Uint64("actor_id", sess.UserID))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePermissionsConfig(w http.ResponseWriter, r *http.Request, sess session) {
	switch r.Method {
	case http.MethodGet:
		file, err := s.svc.LoadPermissionsConfig()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, file)
	case http.MethodPut:
		var file permissions.Policy
		if err := decodeJSON(r, &file); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.svc.SavePermissionsConfig(file); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Info("admin permissions config updated", slog.Uint64("actor_id", sess.UserID))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTrustedKeys(w http.ResponseWriter, r *http.Request, _ session) {
	resp, err := s.svc.TrustedKeys(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMigrationStatus(w http.ResponseWriter, r *http.Request, _ session) {
	status, err := s.svc.MigrationStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleMigrationBackup(w http.ResponseWriter, r *http.Request, sess session) {
	path, err := s.svc.BackupMigrations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.logger.Info("admin migrations backup created", slog.Uint64("actor_id", sess.UserID), slog.String("path", path))
	writeJSON(w, http.StatusOK, map[string]any{"path": path})
}

func (s *Server) handleMigrationUp(w http.ResponseWriter, r *http.Request, sess session) {
	status, err := s.svc.MigrateUp(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.logger.Info("admin migrations applied", slog.Uint64("actor_id", sess.UserID), slog.Int("version", status.CurrentVersion))
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) oauthRedirectURL() string {
	if impl, ok := s.oauth.(interface{ RedirectURL() string }); ok {
		return impl.RedirectURL()
	}
	return ""
}

func (s *Server) putSession(sess session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[sess.ID] = sess
}

func (s *Server) readSession(r *http.Request) (session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return session{}, err
	}
	sessionID, ok := s.unsignCookieValue(sessionCookieName, cookie.Value)
	if !ok {
		return session{}, errors.New("invalid session")
	}
	s.sessionsMu.Lock()
	sess, ok := s.sessions[sessionID]
	s.sessionsMu.Unlock()
	if !ok {
		return session{}, errors.New("invalid session")
	}
	if time.Now().Unix() >= sess.ExpiresAt {
		s.deleteSession(sessionID)
		return session{}, errors.New("session expired")
	}
	return sess, nil
}

func (s *Server) deleteSession(id string) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, id)
}

func (s *Server) cookie(name, value string, ttl time.Duration, httpOnly bool) *http.Cookie {
	if ttl > 0 && value != "" && (name == sessionCookieName || name == stateCookieName) {
		value = s.signCookieValue(name, value)
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: httpOnly,
		SameSite: s.cookieSameSite,
		Secure:   s.cookieSecure,
		MaxAge:   int(ttl.Seconds()),
	}
}

func (s *Server) currentOwnerStatus() OwnerStatus {
	if s == nil || s.ownerStatus == nil {
		return OwnerStatus{Source: "unresolved"}
	}
	return s.ownerStatus()
}

func (s *Server) isOwnerUser(userID uint64) bool {
	status := s.currentOwnerStatus()
	if !status.Resolved || status.EffectiveUserID == nil {
		return false
	}
	return *status.EffectiveUserID == userID
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func subtleTokenCompare(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	return hmac.Equal([]byte(a), []byte(b))
}

func (s *Server) authConfigured() bool {
	// Keep this strict: in production we want missing config to be obvious.
	if strings.TrimSpace(s.appOrigin) == "" {
		return false
	}
	if strings.TrimSpace(s.clientID) == "" || strings.TrimSpace(s.clientSecret) == "" {
		return false
	}
	if len(s.secret) < 32 {
		return false
	}
	return strings.TrimSpace(s.oauthRedirectURL()) != ""
}

func (s *Server) signCookieValue(name, value string) string {
	if len(s.secret) == 0 {
		return value
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(name))
	_, _ = mac.Write([]byte{':'})
	_, _ = mac.Write([]byte(value))
	sum := mac.Sum(nil)
	// Keep cookie value compact: 16 bytes is plenty for tamper detection here.
	sig := base64.RawURLEncoding.EncodeToString(sum[:16])
	return value + "." + sig
}

func (s *Server) unsignCookieValue(name, signed string) (string, bool) {
	if len(s.secret) == 0 {
		return "", false
	}
	parts := strings.Split(signed, ".")
	if len(parts) != 2 {
		return "", false
	}
	value := parts[0]
	want := s.signCookieValue(name, value)
	return value, subtleTokenCompare(want, signed)
}

func avatarURL(user OAuthUser) string {
	if strings.TrimSpace(user.Avatar) == "" || strings.TrimSpace(user.ID) == "" {
		return ""
	}
	return "https://cdn.discordapp.com/avatars/" + strings.TrimSpace(user.ID) + "/" + strings.TrimSpace(user.Avatar) + ".png"
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": strings.TrimSpace(message)})
}

func cookiePolicy(appOrigin, redirectURL string) (bool, http.SameSite) {
	app, err := url.Parse(strings.TrimSpace(appOrigin))
	if err != nil {
		return false, http.SameSiteLaxMode
	}
	redirect, err := url.Parse(strings.TrimSpace(redirectURL))
	if err != nil {
		return false, http.SameSiteLaxMode
	}
	secure := app.Scheme == "https" || redirect.Scheme == "https"
	if secure && !sameOrigin(app, redirect) {
		return true, http.SameSiteNoneMode
	}
	if secure {
		return true, http.SameSiteLaxMode
	}
	return false, http.SameSiteLaxMode
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}
