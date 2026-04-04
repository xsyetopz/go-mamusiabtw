package adminapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/xsyetopz/go-mamusiabtw/internal/guildconfig"
	pluginhostlua "github.com/xsyetopz/go-mamusiabtw/internal/runtime/plugins/lua"
)

func (s *Server) handleGuildConfig(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID  uint64                   `json:"guild_id"`
		PluginID string                   `json:"plugin_id"`
		Config   guildconfig.PluginConfig `json:"config"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := s.svc.PutGuildConfig(r.Context(), sess.AccessToken, req.GuildID, strings.TrimSpace(req.PluginID), req.Config)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": cfg})
}

func (s *Server) handleGuildChannels(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	items, err := s.svc.GuildChannels(r.Context(), sess.AccessToken, guildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": items})
}

func (s *Server) handleGuildRoles(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	items, err := s.svc.GuildRoles(r.Context(), sess.AccessToken, guildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": items})
}

func (s *Server) handleGuildMembers(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	items, err := s.svc.GuildMembers(r.Context(), sess.AccessToken, guildID, strings.TrimSpace(r.URL.Query().Get("query")), limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (s *Server) handleGuildEmojis(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	items, err := s.svc.GuildEmojis(r.Context(), sess.AccessToken, guildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emojis": items})
}

func (s *Server) handleGuildStickers(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	items, err := s.svc.GuildStickers(r.Context(), sess.AccessToken, guildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stickers": items})
}

func (s *Server) handleGuildWarnings(w http.ResponseWriter, r *http.Request, sess session) {
	guildID, ok := guildIDQuery(w, r)
	if !ok {
		return
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("user_id")), 10, 64)
	if err != nil || userID == 0 {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if limit <= 0 {
		limit = 25
	}
	items, err := s.svc.GuildWarnings(r.Context(), sess.AccessToken, guildID, userID, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"warnings": items})
}

func (s *Server) handleGuildWarn(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID uint64 `json:"guild_id"`
		UserID  uint64 `json:"user_id"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.svc.CreateWarning(r.Context(), sess.AccessToken, req.GuildID, sess.UserID, req.UserID, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGuildUnwarn(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID   uint64 `json:"guild_id"`
		WarningID string `json:"warning_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.DeleteWarning(r.Context(), sess.AccessToken, req.GuildID, sess.UserID, req.WarningID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildSlowmode(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID   uint64 `json:"guild_id"`
		ChannelID uint64 `json:"channel_id"`
		Seconds   int    `json:"seconds"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ManagerSlowmode(r.Context(), sess.AccessToken, req.GuildID, req.ChannelID, req.Seconds); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildNickname(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID  uint64 `json:"guild_id"`
		UserID   uint64 `json:"user_id"`
		Nickname string `json:"nickname"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var nickname *string
	if strings.TrimSpace(req.Nickname) != "" {
		value := strings.TrimSpace(req.Nickname)
		nickname = &value
	}
	if err := s.svc.ManagerNickname(r.Context(), sess.AccessToken, req.GuildID, req.UserID, nickname); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildRoleCreate(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.RoleCreateSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	role, err := s.svc.ManagerCreateRole(r.Context(), sess.AccessToken, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": role})
}

func (s *Server) handleGuildRoleEdit(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.RoleEditSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	role, err := s.svc.ManagerEditRole(r.Context(), sess.AccessToken, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": role})
}

func (s *Server) handleGuildRoleDelete(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID uint64 `json:"guild_id"`
		RoleID  uint64 `json:"role_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ManagerDeleteRole(r.Context(), sess.AccessToken, req.GuildID, req.RoleID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildRoleMember(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		Add bool `json:"add"`
		pluginhostlua.RoleMemberSpec
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ManagerMemberRole(r.Context(), sess.AccessToken, req.Add, req.RoleMemberSpec); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildPurge(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID uint64 `json:"guild_id"`
		pluginhostlua.PurgeSpec
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	deleted, err := s.svc.ManagerPurge(r.Context(), sess.AccessToken, req.GuildID, req.PurgeSpec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted_count": deleted})
}

func (s *Server) handleGuildEmojiCreate(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID    uint64 `json:"guild_id"`
		Name       string `json:"name"`
		Filename   string `json:"filename"`
		ContentB64 string `json:"content_b64"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	emoji, err := s.svc.ManagerCreateEmoji(r.Context(), sess.AccessToken, req.GuildID, req.Name, req.Filename, req.ContentB64, req.Width, req.Height)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emoji": emoji})
}

func (s *Server) handleGuildEmojiEdit(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.EmojiEditSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	emoji, err := s.svc.ManagerEditEmoji(r.Context(), sess.AccessToken, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emoji": emoji})
}

func (s *Server) handleGuildEmojiDelete(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.EmojiDeleteSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ManagerDeleteEmoji(r.Context(), sess.AccessToken, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGuildStickerCreate(w http.ResponseWriter, r *http.Request, sess session) {
	var req struct {
		GuildID     uint64 `json:"guild_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		EmojiTag    string `json:"emoji_tag"`
		Filename    string `json:"filename"`
		ContentB64  string `json:"content_b64"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sticker, err := s.svc.ManagerCreateSticker(r.Context(), sess.AccessToken, req.GuildID, req.Name, req.Description, req.EmojiTag, req.Filename, req.ContentB64, req.Width, req.Height)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sticker": sticker})
}

func (s *Server) handleGuildStickerEdit(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.StickerEditSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sticker, err := s.svc.ManagerEditSticker(r.Context(), sess.AccessToken, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sticker": sticker})
}

func (s *Server) handleGuildStickerDelete(w http.ResponseWriter, r *http.Request, sess session) {
	var req pluginhostlua.StickerDeleteSpec
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.svc.ManagerDeleteSticker(r.Context(), sess.AccessToken, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func guildIDQuery(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	guildIDRaw := strings.TrimSpace(r.URL.Query().Get("guild_id"))
	guildID, err := strconv.ParseUint(guildIDRaw, 10, 64)
	if err != nil || guildID == 0 {
		writeError(w, http.StatusBadRequest, "invalid guild_id")
		return 0, false
	}
	return guildID, true
}
