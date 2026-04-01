package store

import (
	"context"
	"time"
)

type TargetType string

const (
	TargetTypeUser  TargetType = "user"
	TargetTypeGuild TargetType = "guild"
)

type Restriction struct {
	TargetType TargetType
	TargetID   uint64
	Reason     string
	CreatedBy  uint64
	CreatedAt  time.Time
}

type Warning struct {
	ID          string
	GuildID     uint64
	UserID      uint64
	ModeratorID uint64
	Reason      string
	CreatedAt   time.Time
}

type AuditEntry struct {
	GuildID    *uint64
	ActorID    *uint64
	Action     string
	TargetType *TargetType
	TargetID   *uint64
	CreatedAt  time.Time
	MetaJSON   string
}

type RestrictionStore interface {
	GetRestriction(ctx context.Context, targetType TargetType, targetID uint64) (Restriction, bool, error)
	PutRestriction(ctx context.Context, r Restriction) error
	DeleteRestriction(ctx context.Context, targetType TargetType, targetID uint64) error
}

type WarningStore interface {
	CountWarnings(ctx context.Context, guildID, userID uint64) (int, error)
	ListWarnings(ctx context.Context, guildID, userID uint64, limit int) ([]Warning, error)
	CreateWarning(ctx context.Context, w Warning) error
	DeleteWarning(ctx context.Context, id string) error
}

type AuditStore interface {
	Append(ctx context.Context, entry AuditEntry) error
}

type TrustedSigner struct {
	KeyID        string
	PublicKeyB64 string
	AddedAt      time.Time
}

type TrustedSignerStore interface {
	ListTrustedSigners(ctx context.Context) ([]TrustedSigner, error)
	PutTrustedSigner(ctx context.Context, signer TrustedSigner) error
	DeleteTrustedSigner(ctx context.Context, keyID string) error
}

type PluginKVStore interface {
	GetPluginKV(ctx context.Context, guildID uint64, pluginID, key string) (valueJSON string, ok bool, err error)
	PutPluginKV(ctx context.Context, guildID uint64, pluginID, key, valueJSON string) error
	DeletePluginKV(ctx context.Context, guildID uint64, pluginID, key string) error
}

type UserSeen struct {
	UserID      uint64
	CreatedAt   time.Time
	IsBot       bool
	IsSystem    bool
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

type UserStore interface {
	UpsertUserSeen(ctx context.Context, u UserSeen) error
	TouchUserSeen(ctx context.Context, userID uint64, seenAt time.Time) error
}

type GuildSeen struct {
	GuildID   uint64
	OwnerID   uint64
	CreatedAt time.Time
	JoinedAt  time.Time
	LeftAt    *time.Time
	Name      string
	UpdatedAt time.Time
}

type GuildStore interface {
	UpsertGuildSeen(ctx context.Context, g GuildSeen) error
	MarkGuildLeft(ctx context.Context, guildID uint64, leftAt time.Time) error
	UpdateGuildOwner(ctx context.Context, guildID uint64, ownerID uint64, updatedAt time.Time) error
}

type GuildMemberStore interface {
	MarkMemberJoined(ctx context.Context, guildID, userID uint64, joinedAt time.Time) error
	MarkMemberLeft(ctx context.Context, guildID, userID uint64, leftAt time.Time) error
}
