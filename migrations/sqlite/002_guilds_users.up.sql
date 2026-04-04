-- migrate:kind=normal
CREATE TABLE IF NOT EXISTS users (
    user_id INTEGER PRIMARY KEY,
    created_at INTEGER NOT NULL,
    is_bot INTEGER NOT NULL,
    is_system INTEGER NOT NULL,
    first_seen_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_last_seen
    ON users(last_seen_at DESC);

CREATE TABLE IF NOT EXISTS guilds (
    guild_id INTEGER PRIMARY KEY,
    owner_id INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    joined_at INTEGER NOT NULL,
    left_at INTEGER,
    name TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_guilds_owner
    ON guilds(owner_id);

CREATE TABLE IF NOT EXISTS guild_members (
    guild_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    joined_at INTEGER NOT NULL,
    left_at INTEGER,
    PRIMARY KEY (guild_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_guild_members_user
    ON guild_members(user_id);
