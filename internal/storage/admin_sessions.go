package storage

import "context"

type AdminSession struct {
	ID          string
	UserID      uint64
	Username    string
	Name        string
	AvatarURL   string
	CSRFToken   string
	AccessToken string
	IsOwner     bool
	ExpiresAt   int64 // unix seconds
}

type AdminSessionStore interface {
	GetAdminSession(ctx context.Context, id string) (AdminSession, bool, error)
	PutAdminSession(ctx context.Context, sess AdminSession) error
	DeleteAdminSession(ctx context.Context, id string) error
	DeleteExpiredAdminSessions(ctx context.Context, nowUnix int64) (int64, error)
}
