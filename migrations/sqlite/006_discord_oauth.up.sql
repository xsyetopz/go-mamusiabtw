-- migrate:kind=normal
CREATE TABLE IF NOT EXISTS discord_oauth_tokens (
  user_id INTEGER PRIMARY KEY,
  access_token_enc TEXT NOT NULL,
  refresh_token_enc TEXT NOT NULL,
  scope TEXT NOT NULL,
  expires_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS plugin_oauth_grants (
  user_id INTEGER NOT NULL,
  plugin_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (user_id, plugin_id)
);

