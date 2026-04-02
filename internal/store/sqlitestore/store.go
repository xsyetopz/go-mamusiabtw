package sqlitestore

import (
	"database/sql"
	"errors"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/store"
)

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}

	return &Store{
		db:  db,
		now: time.Now,
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Restrictions() store.RestrictionStore {
	return restrictionStore{db: s.db, now: s.now}
}

func (s *Store) Warnings() store.WarningStore {
	return warningStore{db: s.db, now: s.now}
}

func (s *Store) Audit() store.AuditStore {
	return auditStore{db: s.db, now: s.now}
}

func (s *Store) TrustedSigners() store.TrustedSignerStore {
	return signerStore{db: s.db, now: s.now}
}

func (s *Store) PluginKV() store.PluginKVStore {
	return pluginKVStore{db: s.db, now: s.now}
}

func (s *Store) ModuleStates() store.ModuleStateStore {
	return moduleStateStore{db: s.db, now: s.now}
}

func (s *Store) Users() store.UserStore {
	return userStore{db: s.db, now: s.now}
}

func (s *Store) Guilds() store.GuildStore {
	return guildStore{db: s.db, now: s.now}
}

func (s *Store) GuildMembers() store.GuildMemberStore {
	return guildMemberStore{db: s.db, now: s.now}
}

func (s *Store) UserSettings() store.UserSettingsStore {
	return userSettingsStore{db: s.db, now: s.now}
}

func (s *Store) Reminders() store.ReminderStore {
	return reminderStore{db: s.db, now: s.now}
}

func (s *Store) CheckIns() store.CheckInStore {
	return checkInStore{db: s.db, now: s.now}
}
