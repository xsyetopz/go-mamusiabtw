# go-mamusiabtw

A Discord bot for helping run and care for your server.

Important: the repo’s stable internal name is `mamusiabtw` (env vars, IDs, and `custom_id` prefixes). Keep it consistent unless you really want to rename everything.

- Engine: Go
- Discord API: `DisgoOrg/disgo`
- Scripting / plugins: `Lua` (embedded via `yuin/gopher-lua`)
- Storage: SQLite (migrations in `migrations/sqlite`)

## Quick Start (Local, No Dashboard)

If you only want the bot running in Discord, start here.

Fastest path:

```bash
go run ./cmd/mamusiabtw init
go run ./cmd/mamusiabtw dev
```

Manual path:

1. Copy the env example:
   - `.env.dev.example` -> `.env.dev`
2. Fill in:
   - `DISCORD_TOKEN` (from the Discord Developer Portal, Bot page)
3. Optional but recommended for fast iteration:
   - set `DISCORD_DEV_GUILD_ID` (register commands to one guild)
4. Run:
   - `go run ./cmd/mamusiabtw`

mamusiabtw creates or opens the SQLite database at `SQLITE_PATH` and applies pending `up` migrations automatically on startup.

If you see migration output: that is expected on first run.

Note: mamusiabtw auto-loads an env file if present:

- dev commands load `.env.dev`
- production-style runs load `.env.prod`

No other dotenv filenames are supported. If you want to disable dotenv entirely,
set `MAMUSIABTW_DISABLE_DOTENV=1`.

For runtime build metadata, use:

- `go run ./cmd/mamusiabtw version`

For explicit migration control, use:

- `go run ./cmd/mamusiabtw migrate status`
- `go run ./cmd/mamusiabtw migrate up`
- `go run ./cmd/mamusiabtw migrate backup`
- `go run ./cmd/mamusiabtw migrate down --to 4`
- `go run ./cmd/mamusiabtw migrate down --steps 1`

`migrate backup` writes a SQLite snapshot into `MAMUSIABTW_MIGRATION_BACKUPS_DIR`.
Old local DB files from older pre-plugin versions are not supported for upgrade and should be recreated.

## Optional Servers

If `MAMUSIABTW_OPS_ADDR` is set, mamusiabtw also starts a small HTTP ops server with:

- `/healthz`
- `/readyz`
- `/metrics`

If `MAMUSIABTW_ADMIN_ADDR` is set, mamusiabtw also starts the website/dashboard API.
The frontend lives in `apps/dashboard/` and uses Discord OAuth against that API.

## Website + Dashboard (Local)

If you want the website/dashboard, you will run two things:

- Terminal A: the bot + admin API (`go run ...`)
- Terminal B (optional): the dashboard frontend dev server for HMR (`bun run dev`)

Important: you always open the dashboard at the admin API origin:

- `http://127.0.0.1:8081/`

Even if Vite is running, the admin API proxies to it so cookies and auth stay
simple. You do not open `:5173` in the browser.

### Step 1: Configure The Bot/API Env

Preferred (zero thinking):

```bash
go run ./cmd/mamusiabtw init
```

That writes `.env.dev` with sane defaults.

If you want to do it manually, copy:

- `.env.dev.example` -> `.env.dev`

Minimum to start the admin API in dev:

- `DISCORD_TOKEN=...`
- `MAMUSIABTW_ADMIN_ADDR=127.0.0.1:8081`

To enable Discord sign-in (OAuth) you also need:

- `MAMUSIABTW_DASHBOARD_CLIENT_ID=...`
- `MAMUSIABTW_DASHBOARD_CLIENT_SECRET=...`

In dev, `MAMUSIABTW_DASHBOARD_SESSION_SECRET` is generated automatically if you
don’t set it (sessions will reset on restart). For stable sessions, set it to a
32+ character random string.

### Step 2: Tell Discord About The Redirect URL (One-Time)

In the Discord Developer Portal, your application must allow the callback URL:

- `http://127.0.0.1:8081/api/auth/callback`

If you prefer `localhost`, you can use:

- `http://localhost:8081/api/auth/callback`

The admin API requests OAuth2 scopes `identify` and `guilds` during login.

### Step 3: Run The Bot/API

Preferred:

```bash
go run ./cmd/mamusiabtw dev
```

You can also run the plain command:

```bash
go run ./cmd/mamusiabtw
```

### Step 4: Run The Dashboard Frontend

If you want HMR while developing the UI:

1. In another terminal: `cd apps/dashboard`
2. Install + run:

```bash
bun install
bun run dev
```

Then open:

- `http://127.0.0.1:8081/`

### Where Do I Get CLIENT_ID / CLIENT_SECRET / SESSION_SECRET?

`MAMUSIABTW_DASHBOARD_CLIENT_ID` and `MAMUSIABTW_DASHBOARD_CLIENT_SECRET`:

1. Discord Developer Portal -> your application
2. OAuth2
3. Copy:
   - Client ID -> `MAMUSIABTW_DASHBOARD_CLIENT_ID`
   - Client Secret (you may need to reset/reveal it) -> `MAMUSIABTW_DASHBOARD_CLIENT_SECRET`

`MAMUSIABTW_DASHBOARD_SESSION_SECRET`:

- generate a random secret and paste it
- examples:

```bash
openssl rand -hex 32
# or
openssl rand -base64 48
```

## Website + Dashboard (Production)

The recommended production shape is still single-origin:

- the admin API serves the dashboard files
- the browser talks to `/api/...` on the same origin

Checklist:

1. Copy `.env.prod.example` -> `.env.prod`
2. Set at least:
   - `DISCORD_TOKEN=...`
   - `MAMUSIABTW_ADMIN_ADDR=0.0.0.0:8081` (or behind a reverse proxy)
   - `MAMUSIABTW_DASHBOARD_CLIENT_ID=...`
   - `MAMUSIABTW_DASHBOARD_CLIENT_SECRET=...`
   - `MAMUSIABTW_DASHBOARD_SESSION_SECRET=...` (32+ chars)
3. Build the dashboard once:
   - `cd apps/dashboard && bun install && bun run build`

If `apps/dashboard/dist/index.html` exists, the admin API serves it automatically.

## Env Convention

Repo standard:

- root dev bot/API: `.env.dev`
- root prod bot/API: `.env.prod`

## Dashboard Routes

- `#/` home
- `#/servers` server picker
- `#/servers/<guild_id>` server dashboard
- `#/owner` owner-only control area

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

If the API is not reachable or the dashboard URLs are invalid, the app opens into setup diagnostics (instead of a blank login failure).

## Common Setup Problems (Fast Checks)

- Dashboard shows `127.0.0.1:8081/api/auth/me ... ERR_CONNECTION_REFUSED`:
  - the bot/admin API is not running, or `MAMUSIABTW_ADMIN_ADDR` is wrong
  - run `go run ./cmd/mamusiabtw doctor` to see what config the bot thinks it has
- Quick self-checks:
  - `curl -I http://127.0.0.1:8081/` (dashboard HTML, via admin API)
  - `curl -I http://127.0.0.1:8081/api/setup` (admin API)
- Dashboard loads on `:5173` but login/session never “sticks”:
  - you opened the Vite dev server directly
  - open `http://127.0.0.1:8081/` instead (the admin API proxies Vite so cookies/auth work)
- Bot exits with `failed to open gateway connection: websocket: close 4014: Disallowed intent(s).`:
  - Discord is rejecting privileged gateway intents your bot requests
  - fix: Discord Developer Portal -> your application -> Bot -> Privileged Gateway Intents
  - enable: `Server Members Intent` (mamusiabtw requests guild member events by default)
  - dev note: `mamusiabtw dev` keeps the admin API running even if the bot cannot connect,
    so the dashboard setup page stays reachable while you fix intents/tokens
- Login redirects but Discord errors:
  - your OAuth Redirect URI does not match exactly
  - make sure it’s `http://127.0.0.1:8081/api/auth/callback` for local
- Owner page denies access:
  - the bot must be able to resolve the Discord application owner
  - fallback: set `OWNER_USER_ID=...`

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

The direct-binary flow and the Docker flow use the same env vars and the same `config/`, `plugins/`, `locales/`, and `migrations/` folders.

The bot also supports runtime module toggles from `config/modules.json`, with official first-party plugins and user plugins sharing the same `plugins/` root.

## Docker

1. Copy `.env.dev.example` to `.env` and fill in at least `DISCORD_TOKEN`.
2. Start: `docker compose up --build`

`compose.yml` bind-mounts `./data`, `./plugins`, and `./config` into the container for a dev-friendly workflow.

## Built-in Commands

- `/ping`
- `/help`
- `/block` and `/unblock` (owner-only; owner resolved from the Discord application, with optional `OWNER_USER_ID` fallback)
- `/plugins`
- `/modules`

Optional first-party plugins now live in `plugins/` too:

- `info`: `/about`, `/lookup user|guild|role|channel`
- `fun`: `/flip`, `/roll`, `/8ball`, `/hug`, `/pat`, `/poke`, `/shrug`
- `wellness`: `/timezone`, `/checkin`, `/remind`
- `moderation`: `/warn`, `/unwarn`
- `manager`: `/slowmode`, `/nick`, `/purge`, `/roles`, `/emojis`, `/stickers`

## Modules

mamusiabtw now treats built-ins and plugins as modules:

- required core/admin built-ins stay available even if optional modules are disabled
- official first-party plugins live in `plugins/` beside user-made plugins
- official vs user plugin classification is host-owned, not self-declared by a manifest field
- owner-only `/modules` lets you list, inspect, enable, disable, reset, and reload modules

Default module seeds live in `config/modules.json`, and runtime overrides are stored in SQLite.

## Lua Plugins

Plugins live under `plugins/<plugin>/` with:

- `plugin.json` (manifest)
- `plugin.lua` (entrypoint; returns a plugin descriptor table and can assemble the rest of the plugin workspace)
- `commands/*.lua`, `lib/*.lua`, or any other local layout you want, loaded via `bot.require("...")`
- `locales/<locale>/messages.json` (optional plugin i18n)

Plugins are sandboxed: no filesystem access, and no network access except through explicitly granted host capabilities. Any plugin capability must be both:

1) requested in `plugin.json`, and
2) granted by the host in `config/permissions.json` (default `MAMUSIABTW_PERMISSIONS_FILE`).

The Lua host currently exposes `bot.random.*` and a generic `bot.http.*` client for plugin-owned integrations, with HTTP access gated by the permissions policy.

The host injects a namespaced global `bot` into plugin scripts (see `sdk/lua/bot_api.lua:1` for the editor stub). A flat `mamusiabtw` alias remains for older plugins, but new plugins should use `bot`.

The repo ships a minimal example plugin in `examples/plugins/example` which exposes `/example`.

### JSON Schemas

For editor validation/autocomplete, these JSON files support a `$schema` URL (Raw GitHub):

- `plugins/<plugin>/plugin.json` → `schemas/plugin.schema.v1.json`
- `config/permissions.json` → `schemas/permissions.schema.v1.json`
- `config/modules.json` → `schemas/modules.schema.v1.json`
- `config/trusted_keys.json` → `schemas/trusted_keys.schema.v1.json`
- `plugins/<plugin>/signature.json` → `schemas/signature.schema.v1.json`

`locales/<locale>/messages.json` is a JSON array, so it can’t embed `$schema`, but the repo ships `schemas/messages.schema.v1.json`.

### Plugin Authoring Model

Plugins are authored as `route + context + effect`:

- `plugin.lua` returns `bot.plugin({ ... })`
- `plugin.lua` is just the entrypoint; split commands, subcommands, views, and helpers into as many local Lua files as you want
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

Locale folders must use official Discord locale codes (the same ones shipped under `./locales/`, like `en-US`, `fr`, `ja`, `zh-CN`)... anything else is ignored.

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

Use `/modules reload` to rebuild the full module catalog and command registration, including built-ins plus every plugin found in `plugins/`.

### Signing (prod)

When `MAMUSIABTW_PROD_MODE=1`, plugins must include `signature.json` and be signed by a trusted key. mamusiabtw rejects `MAMUSIABTW_PROD_MODE=1` together with `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=1`.

- Sign a plugin directory with:
  `go run ./cmd/mamusiabtw sign-plugin --dir ./plugins/<id> --key-id <key_id> --private-key-file <path>`
- Seed keys via `MAMUSIABTW_TRUSTED_KEYS_FILE`
- Additional trusted keys are stored in SQLite (`trusted_signers`)

### Plugin Trust Modes

- Production signed mode: `MAMUSIABTW_PROD_MODE=1` and `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0`
- Mixed dev mode: `MAMUSIABTW_PROD_MODE=0` and `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=1`
- Recommended release default: keep unsigned plugins off anywhere you treat as production

## Compatibility Options

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

The shipped Docker build supports these build args:

- `BUILD_VERSION`
- `BUILD_REPOSITORY`
- `BUILD_DESCRIPTION`
- `BUILD_DEVELOPER_URL`
- `BUILD_SUPPORT_SERVER_URL`
- `BUILD_MASCOT_IMAGE_URL`

## License

[MIT](LICENSE)
