# imotherbtw

Just Another General-Purpose Discord App.

- Engine: Go
- Discord API: `DisgoOrg/disgo`
- Scripting / plugins: `Lua` (embedded via `yuin/gopher-lua`)
- Storage: SQLite (migrations in `migrations/sqlite`)

## Running

1. Copy `.env.example` to `.env` and fill in at least `DISCORD_TOKEN`.
2. (Recommended) Set `DISCORD_DEV_GUILD_ID` for fast command registration.
3. Start: `go run ./cmd/imotherbtw`

imotherbtw creates/opens the SQLite DB at `SQLITE_PATH` and applies migrations automatically on startup.

## Docker

1. Copy `.env.example` to `.env` and fill in at least `DISCORD_TOKEN`.
2. Start: `docker compose up --build`

`compose.yml` bind-mounts `./data`, `./plugins`, and `./config` into the container for a dev-friendly workflow.

## Built-in Commands

- `/ping`
- `/help`
- `/about`
- `/lookup user|guild`
- `/warn` and `/unwarn` (interactive select-menu)
- `/block` and `/unblock` (owner-only; owner IDs via `OWNER_USER_IDS`)
- Fun: `/flip`, `/roll`, `/8ball`, `/hug`, `/pat`, `/poke`, `/shrug`
- Manager: `/slowmode`, `/nick`, `/purge`, `/roles`, `/emojis`, `/stickers`

## Lua Plugins

Plugins live under `plugins/<plugin>/` with:

- `plugin.json` (manifest)
- `plugin.lua` (script)
- `locales/<locale>/messages.json` (optional plugin i18n)

Plugins are sandboxed: no filesystem or network access. Any plugin capability must be both:
1) requested in `plugin.json`, and
2) granted by the host in `config/permissions.json` (default `IMOTHERBTW_PERMISSIONS_FILE`).

The host injects a global `imotherbtw` table into plugin scripts (see `plugins/jagpda_api.lua:1` for the editor stub).

The repo ships a minimal example plugin in `plugins/example` which exposes `/example`.

### Plugin Localization

If a plugin has `plugins/<id>/locales/<locale>/messages.json`, the host loads it and exposes:

- `imotherbtw.t(message_id, data?, plural_count?)` inside Lua handlers.
- `commands[].description_id` in `plugin.json` to localize slash command descriptions.

### Plugin Entry Points

Plugins can implement:

- `Handle(cmd, ctx)` (required for slash commands)
- `HandleComponent(id, ctx)` (optional; message components)
- `HandleModal(id, ctx)` (optional; modal submits)

`cmd`/`id` is the *local* ID. The host namespaces all Discord `custom_id`s as `imotherbtw:pl:<plugin_id>:<local_id>` and routes them back to the plugin.

### Plugin Responses

Handlers may return either:

- a string (shortcut for “update message” for components, otherwise “create message”), or
- a table describing an action:
  - `{ type="message", content=..., embeds=..., components=..., ephemeral=true|false }`
  - `{ type="update", content=..., embeds=..., components=... }`
  - `{ type="modal", id=..., title=..., components={...text inputs...} }`
  - `{ present={ kind=..., title=..., body=..., fields=... }, ephemeral=true|false }`

For a full schema reference, see the LuaLS type stubs in `plugins/jagpda_api.lua:1`.

Plugin responses are validated against Discord limits (content lengths, embed limits, component limits). Invalid responses are rejected.

### Hot Reload

Use `/plugins reload` (owner-only) to reload plugins from disk and re-register commands.

### Signing (prod)

When `IMOTHERBTW_PROD_MODE=1` and `IMOTHERBTW_WALLOW_UNSIGNED_PLUGINS=0`, plugins must include `signature.json` and be signed by a trusted key.

- Seed keys via `IMOTHERBTW_TRUSTED_KEYS_FILE`
- Additional trusted keys are stored in SQLite (`trusted_signers`)

## Legacy Parity Options

### Command Registration

By default, imotherbtw registers slash commands globally (unless `DISCORD_DEV_GUILD_ID` is set).

Configure:

- `IMOTHERBTW_COMMAND_REGISTRATION_MODE=global|guilds|hybrid`
- `IMOTHERBTW_COMMAND_GUILD_IDS=...` (comma-separated)
- `IMOTHERBTW_COMMAND_REGISTER_ALL_GUILDS=1` (attempt to register in every cached guild)

### Restricted Message Links (build-time)

To include developer/support links in the restricted message, build with:

- `buildinfo.DeveloperURL`
- `buildinfo.SupportServerURL`

To include a mascot image in `/about`, build with `buildinfo.MascotImageURL`.
