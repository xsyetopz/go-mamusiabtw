# go-mamusiabtw

A Discord bot for helping run and care for your server.

Important: the repo’s stable internal name is `mamusiabtw` (env vars, IDs, and `custom_id` prefixes). Keep it consistent unless you really want to rename everything.

- Engine: Go
- Discord API: `DisgoOrg/disgo`
- Scripting / plugins: `Lua` (embedded via `yuin/gopher-lua`)
- Storage: SQLite (migrations in `migrations/sqlite`)

## Running

1. Copy `.env.example` to `.env` and fill in `DISCORD_TOKEN`.
2. (Recommended) Set `DISCORD_DEV_GUILD_ID` for quicker command registration.
3. Start: `go run ./cmd/mamusiabtw`

mamusiabtw creates or opens the SQLite database at `SQLITE_PATH` and applies migrations automatically on startup.

The direct-binary flow and the Docker flow use the same env vars and the same `config/`, `plugins/`, `locales/`, and `migrations/` folders.

## Docker

1. Copy `.env.example` to `.env` and fill in at least `DISCORD_TOKEN`.
2. Start: `docker compose up --build`

`compose.yml` bind-mounts `./data`, `./plugins`, and `./config` into the container for a dev-friendly workflow.

## Built-in Commands

- `/ping`
- `/help`
- `/about`
- `/lookup user|guild|role|channel`
- `/warn` and `/unwarn` (interactive select-menu)
- `/block` and `/unblock` (owner-only; owner IDs via `OWNER_USER_IDS`)
- Wellness: `/timezone`, `/checkin`, `/remind`
- Fun: `/flip`, `/roll`, `/8ball`, `/hug`, `/pat`, `/poke`, `/shrug`
- Manager: `/slowmode`, `/nick`, `/purge`, `/roles`, `/emojis`, `/stickers`

## Lua Plugins

Plugins live under `plugins/<plugin>/` with:

- `plugin.json` (manifest)
- `plugin.lua` (entrypoint; returns a plugin descriptor table)
- `lib/*.lua` (optional local modules loaded via `bot.require("lib/foo.lua")`)
- `locales/<locale>/messages.json` (optional plugin i18n)

Plugins are sandboxed: no filesystem or network access. Any plugin capability must be both:

1) requested in `plugin.json`, and
2) granted by the host in `config/permissions.json` (default `MAMUSIABTW_PERMISSIONS_FILE`).

The host injects a namespaced global `bot` into plugin scripts (see `sdk/lua/bot_api.lua:1` for the editor stub). A flat `mamusiabtw` alias remains for older plugins, but new plugins should use `bot`.

The repo ships a minimal example plugin in `examples/plugins/example` which exposes `/example`.

### JSON Schemas

For editor validation/autocomplete, these JSON files support a `$schema` URL (Raw GitHub):

- `plugins/<plugin>/plugin.json` → `schemas/plugin.schema.v1.json`
- `config/permissions.json` → `schemas/permissions.schema.v1.json`
- `config/trusted_keys.json` → `schemas/trusted_keys.schema.v1.json`
- `plugins/<plugin>/signature.json` → `schemas/signature.schema.v1.json`

`locales/<locale>/messages.json` is a JSON array, so it can’t embed `$schema`, but the repo ships `schemas/messages.schema.v1.json`.

### Plugin Authoring Model

Plugins are authored as `route + context + effect`:

- `plugin.lua` returns `bot.plugin({ ... })`
- commands, components, modals, events, and jobs are declared in that descriptor
- route handlers receive a typed `ctx` table instead of raw hidden keys
- handlers return effects using `bot.ui.*` or `bot.effects.*`

Minimal shape:

```lua
return bot.plugin({
  commands = {
    bot.command("hello", {
      description = "Say hi",
      run = function(ctx)
        return bot.ui.reply({ content = "hi" })
      end
    })
  }
})
```

### Plugin Localization

If a plugin has `plugins/<id>/locales/<locale>/messages.json`, the host loads it and exposes:

- `bot.i18n.t(message_id, data?, plural_count?)` inside Lua handlers.
- `description_id` in descriptor-defined commands/options/subcommands/groups to localize slash command descriptions.

Locale folders must use official Discord locale codes (the same ones shipped under `./locales/`, like `en-US`, `fr`, `ja`, `zh-CN`)... anything else is ignored.~

### Plugin Entry Points

Plugins can declare route tables in the descriptor:

- `commands = { bot.command(...), ... }`
- `components = { ["local_id"] = function(ctx) ... end }`
- `modals = { ["local_id"] = function(ctx) ... end }`
- `events = { ["guild_member_join"] = function(ctx) ... end }`
- `jobs = { bot.job(...), ... }`

Supported event names:

- `guild_member_join`
- `guild_member_leave`
- `guild_ban`
- `guild_unban`

`cmd`/`id` is the *local* ID. The host namespaces all Discord `custom_id`s as `mamusiabtw:pl:<plugin_id>:<local_id>` and routes them back to the plugin.

### Plugin Responses

Handlers may return either:

- a string (shortcut for “update message” for components, otherwise “create message”), or
- a table describing an action:
  - `{ type="message", content=..., embeds=..., components=..., ephemeral=true|false }`
  - `{ type="update", content=..., embeds=..., components=... }`
  - `{ type="modal", id=..., title=..., components={...text inputs...} }`
  - `{ present={ kind=..., title=..., body=..., fields=... }, ephemeral=true|false }`

For a full schema and SDK reference, see the LuaLS type stubs in `sdk/lua/bot_api.lua:1`.

Plugin responses are validated against Discord limits (content lengths, embed limits, component limits). Invalid responses are rejected.

### Hot Reload

Use `/plugins reload` (owner-only) to reload plugins from disk and re-register commands.

### Signing (prod)

When `MAMUSIABTW_PROD_MODE=1` and `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0`, plugins must include `signature.json` and be signed by a trusted key.

- Seed keys via `MAMUSIABTW_TRUSTED_KEYS_FILE`
- Additional trusted keys are stored in SQLite (`trusted_signers`)

### Plugin Trust Modes

- Production signed mode: `MAMUSIABTW_PROD_MODE=1` and `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0`
- Mixed dev mode: `MAMUSIABTW_PROD_MODE=0` and `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=1`
- Recommended release default: keep unsigned plugins off anywhere you treat as production

## Legacy Parity Options

### Cooldowns

- Global: `MAMUSIABTW_SLASH_COOLDOWN_MS`
- Overrides: `MAMUSIABTW_SLASH_COOLDOWN_OVERRIDES_MS` (comma-separated `name=ms`, supports subcommands like `lookup:user=2500` and grouped subcommands like `manager:roles:add=2500`)

### Command Registration

By default, mamusiabtw registers slash commands globally (unless `DISCORD_DEV_GUILD_ID` is set).

Configure:

- `MAMUSIABTW_COMMAND_REGISTRATION_MODE=global|guilds|hybrid`
- `MAMUSIABTW_COMMAND_GUILD_IDS=...` (comma-separated)
- `MAMUSIABTW_COMMAND_REGISTER_ALL_GUILDS=1` (attempt to register in every cached guild)

### Restricted Message Links (build-time)

To include developer/support links in the restricted message, build with:

- `buildinfo.DeveloperURL`
- `buildinfo.SupportServerURL`

To include a mascot image in `/about`, build with `buildinfo.MascotImageURL`.

## License

[MIT](LICENSE)
