# Reference

This file intentionally contains the longer “power user” documentation that
would otherwise make the main `README.md` hard to scan.

## Dashboard Coverage (Today)

- sign in with Discord
- server picker for guilds the user can manage
- per-server install/setup status
- server settings (plugin config per guild)
- server manager actions: slowmode, nick, roles, purge, emojis, stickers
- moderation actions: warn/unwarn
- owner overview / runtime state
- owner module enable / disable / reset / reload
- owner plugin list / reload / signing state
- owner plugin scaffolding
- owner setup diagnostics for API + OAuth wiring
- owner migration status / backup

## Release Builds

Use `./scripts/build-release.sh` to build a binary with injected `buildinfo` metadata.

Supported env overrides:

- `VERSION`
- `REPOSITORY`
- `DESCRIPTION`
- `DEVELOPER_URL`
- `SUPPORT_SERVER_URL`
- `MASCOT_IMAGE_URL`

The Docker build accepts the same values as `BUILD_*` args.

## Docker

1. Copy `.env.dev.example` to `.env.dev` and fill in at least `DISCORD_TOKEN`.
2. Start: `docker compose up --build`

`compose.yml` bind-mounts `./data`, `./plugins`, and `./config` into the container.

## Built-in Commands

- `/ping`
- `/help`
- `/block` and `/unblock` (owner-only)
- `/plugins`
- `/modules`

Optional first-party plugins live in `plugins/` too:

- `info`: `/about`, `/lookup user|guild|role|channel`
- `fun`: `/flip`, `/roll`, `/8ball`, `/hug`, `/pat`, `/poke`, `/shrug`
- `wellness`: `/timezone`, `/checkin`, `/remind`
- `moderation`: `/warn`, `/unwarn`
- `manager`: `/slowmode`, `/nick`, `/purge`, `/roles`, `/emojis`, `/stickers`

## Modules

mamusiabtw treats built-ins and plugins as modules.

Default module seeds: `config/modules.json`
Runtime overrides: stored in SQLite.

## Lua Plugins

Plugins live under `plugins/<plugin>/` with:

- `plugin.json` (manifest)
- `plugin.lua` (entrypoint; returns a plugin descriptor table)
- `commands/*.lua`, `lib/*.lua`, or any layout you want, loaded via `bot.require("...")`
- `locales/<locale>/messages.json` (optional plugin i18n)

Plugins are sandboxed:

- no filesystem access
- no network access except through explicitly granted host capabilities

Any plugin capability must be both:

1. requested in `plugin.json`, and
2. granted by the host in `config/permissions.json` (default `MAMUSIABTW_PERMISSIONS_FILE`).

The host injects a namespaced global `bot` into plugin scripts (see `sdk/lua/bot_api.lua:1`).

### JSON Schemas

- `plugins/<plugin>/plugin.json` → `schemas/plugin.schema.v1.json`
- `config/permissions.json` → `schemas/permissions.schema.v1.json`
- `config/modules.json` → `schemas/modules.schema.v1.json`
- `config/trusted_keys.json` → `schemas/trusted_keys.schema.v1.json`
- `plugins/<plugin>/signature.json` → `schemas/signature.schema.v1.json`

### Hot Reload

- `/plugins reload` reloads plugins from disk and re-registers commands (owner-only).
- `/modules reload` rebuilds the module catalog and command registration.

### Signing (prod)

When `MAMUSIABTW_PROD_MODE=1`, plugins must be signed.

- Sign a plugin directory with:
  `go run ./cmd/mamusiabtw sign-plugin --dir ./plugins/<id> --key-id <key_id> --private-key-file <path>`
- Seed keys via `MAMUSIABTW_TRUSTED_KEYS_FILE`
- Additional trusted keys live in SQLite (`trusted_signers`)

## Compatibility Options

### Cooldowns

- Global: `MAMUSIABTW_SLASH_COOLDOWN_MS`
- Overrides: `MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS` (comma-separated `name=ms`)

### Command Registration

By default, mamusiabtw registers slash commands globally (unless `DISCORD_DEV_GUILD_ID` is set).

- `MAMUSIABTW_COMMAND_REGISTRATION_MODE=global|guilds|hybrid`
- `MAMUSIABTW_COMMAND_GUILD_IDS=...`
- `MAMUSIABTW_COMMAND_REGISTER_ALL_GUILDS=1`

