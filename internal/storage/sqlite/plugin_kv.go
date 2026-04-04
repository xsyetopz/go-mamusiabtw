package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type pluginKVStore struct {
	db  *sql.DB
	now func() time.Time
}

func (s pluginKVStore) GetPluginKV(ctx context.Context, guildID uint64, pluginID, key string) (string, bool, error) {
	pluginID = strings.TrimSpace(pluginID)
	key = strings.TrimSpace(key)
	if pluginID == "" || key == "" {
		return "", false, nil
	}

	const query = `
SELECT value_json
FROM plugin_kv
WHERE guild_id = ? AND plugin_id = ? AND key = ?`

	guildIDDB, err := toInt64(guildID, "guild_id")
	if err != nil {
		return "", false, err
	}

	var value string
	err = s.db.QueryRowContext(ctx, query, guildIDDB, pluginID, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("get plugin kv: %w", err)
	}
	return value, true, nil
}

func (s pluginKVStore) PutPluginKV(ctx context.Context, guildID uint64, pluginID, key, valueJSON string) error {
	pluginID = strings.TrimSpace(pluginID)
	key = strings.TrimSpace(key)
	if pluginID == "" || key == "" {
		return errors.New("plugin_id and key are required")
	}

	const query = `
INSERT INTO plugin_kv(guild_id, plugin_id, key, value_json, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(guild_id, plugin_id, key)
DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at`

	guildIDDB, err := toInt64(guildID, "guild_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, query, guildIDDB, pluginID, key, valueJSON, s.now().Unix())
	if err != nil {
		return fmt.Errorf("put plugin kv: %w", err)
	}
	return nil
}

func (s pluginKVStore) DeletePluginKV(ctx context.Context, guildID uint64, pluginID, key string) error {
	pluginID = strings.TrimSpace(pluginID)
	key = strings.TrimSpace(key)
	if pluginID == "" || key == "" {
		return errors.New("plugin_id and key are required")
	}

	guildIDDB, err := toInt64(guildID, "guild_id")
	if err != nil {
		return err
	}

	const query = `DELETE FROM plugin_kv WHERE guild_id = ? AND plugin_id = ? AND key = ?`
	_, err = s.db.ExecContext(ctx, query, guildIDDB, pluginID, key)
	if err != nil {
		return fmt.Errorf("delete plugin kv: %w", err)
	}
	return nil
}
