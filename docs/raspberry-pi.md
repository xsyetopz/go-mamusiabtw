# Raspberry Pi Guide

This guide is for running `mamusiabtw` on Raspberry Pi hardware.

It is intentionally split by **what you host on the Pi**, because the right
answer is different for a Pi Zero 2 than for a Pi 5.

## Fast Answer

If you want the short version:

- canonical public production for this repo is:
  static dashboard + separate admin API
- Pi Zero 2 W:
  run **Profile A** or **Profile B**
- Pi 3:
  run **Profile B** unless you want the smallest possible setup
- Pi 4 / Pi 5:
  run **Profile C** if you want one box to do everything

If you hate reading long docs:

1. Choose a profile below.
2. Copy `.env.prod.example` to `.env.prod`.
3. Fill in the env block for that profile.
4. Run `go run ./cmd/mamusiabtw doctor`.
5. Build with `./scripts/build-release.sh`.
6. Use `systemd`.

## Which Profile Should I Pick?

### Pick Profile A if

- you mainly care about the bot being online
- you are on Pi Zero 2
- you do not need the web dashboard on the Pi

### Pick Profile B if

- you want the dashboard
- you do not want the Pi to deal with frontend hosting
- you want the best balance of power vs simplicity

### Pick Profile C if

- you want one device to host the bot, admin API, and dashboard
- you are on Pi 4 or Pi 5
- you are okay building or copying `apps/dashboard/dist`

## Support Level

This guide covers:

- Raspberry Pi Zero 2 W
- Raspberry Pi 3
- Raspberry Pi 4
- Raspberry Pi 5

This guide does **not** cover older ARMv6 boards like:

- Raspberry Pi Zero W
- Raspberry Pi 1

Reason:

- this project uses SQLite through a pure-Go driver
- older ARMv6 boards are still not the baseline we want to support here
- older ARMv6 boards are possible in theory, but they are not a sane baseline for this repo

## Choose A Pi Setup

### Profile A: Bot Only On The Pi

Host on Pi:

- Discord bot
- SQLite

Do not host on Pi:

- admin API
- dashboard

Best fit:

- Pi Zero 2 W
- Pi 3
- anyone who wants the smallest, lowest-maintenance setup

Why choose this:

- lowest RAM and CPU usage
- fewest moving parts
- no dashboard OAuth setup
- easiest production path on weak hardware

Why not choose this:

- no web dashboard
- no browser-based server management
- config changes are file/CLI-driven

Recommendation:

- if you mainly want the bot online and stable, start here on Pi Zero 2

Quick setup shape:

- copy `.env.prod.example` -> `.env.prod`
- set `DISCORD_TOKEN`
- build and run the bot
- skip the admin API entirely

### Profile B: Bot + Admin API On The Pi, Dashboard Elsewhere

Host on Pi:

- Discord bot
- SQLite
- admin API

Host elsewhere:

- static dashboard site

Best fit:

- Pi Zero 2 W
- Pi 3
- Pi 4
- users who want dashboard access without asking the Pi to serve the frontend

Why choose this:

- good balance between functionality and Pi load
- full dashboard still works
- frontend can live on GitHub Pages or another static host
- no need to build or serve frontend assets on the Pi

Why not choose this:

- requires separate dashboard hosting
- requires public origin and CORS/OAuth setup

Recommendation:

- this is the best all-around choice if you want the dashboard and you are not on a Pi 4/5 with plenty of headroom

Quick setup shape:

- Pi runs the bot and admin API
- static dashboard lives elsewhere
- browser talks to the Pi API through the public API origin
- this matches the repo's canonical public production topology

### Profile C: Full Stack On The Pi

Host on Pi:

- Discord bot
- SQLite
- admin API
- built dashboard files from `apps/dashboard/dist`

Best fit:

- Pi 4
- Pi 5
- Pi 3 if you accept slower builds and less headroom

Why choose this:

- one box does everything
- easy LAN access at `http://<pi-host>:8081/`
- simplest mental model once deployed

Why not choose this:

- higher CPU, RAM, and storage pressure
- frontend builds are annoying on weak boards
- Pi Zero 2 should not be your first choice for this profile

Important:

- this profile serves the **built** dashboard through the admin API
- it does **not** mean running Vite in normal operation
- Bun is only needed to build the dashboard, not to serve it at runtime

Recommendation:

- use this on Pi 4/5 if you want the simplest single-device setup

Quick setup shape:

- Pi runs the bot and admin API
- Pi also serves built frontend files from `apps/dashboard/dist`
- browser opens the Pi directly
- this is a self-hosted convenience path, not the repo's main public deployment default

## Board Advice

### Pi Zero 2 W

Recommended:

- Profile A
- Profile B

Possible, but not recommended:

- Profile C

Notes:

- runtime is fine if you keep the setup simple
- native builds work, but they are slower
- build the dashboard somewhere else and copy `apps/dashboard/dist` if you insist
  on Profile C

### Pi 3

Recommended:

- Profile A
- Profile B

Acceptable:

- Profile C

Notes:

- a reasonable middle ground
- still worth prebuilding the dashboard on another machine if you want faster
  deploys

### Pi 4

Recommended:

- all profiles

Notes:

- the first Pi where full-stack self-hosting feels normal

### Pi 5

Recommended:

- all profiles

Notes:

- best Pi option for this project
- easiest board for local builds and full local hosting

## Native Build On The Pi

This is the default path because the repo now builds SQLite without CGO.

What you need:

- a Pi with enough patience for a native Go build
- Go 1.26.2 or newer

On Raspberry Pi OS / Debian-style systems:

```bash
sudo apt update
sudo apt install -y git
```

Install Go 1.26.2 or newer.

### Install Go On 64-bit Raspberry Pi OS

If `uname -m` prints `aarch64`, this is the direct path:

```bash
wget https://go.dev/dl/go1.26.2.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.2.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
go version
```

### Install Go On 32-bit Raspberry Pi OS

If `uname -m` prints `armv7l` or `armv6l`, use the 32-bit archive:

```bash
wget https://go.dev/dl/go1.26.2.linux-armv6l.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.2.linux-armv6l.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
go version
```

Why the archive says `armv6l` even on newer boards:

- that is the name Go uses for the 32-bit Linux ARM tarball
- it is still the right download for 32-bit Raspberry Pi OS on newer supported
  boards

If you want one helper instead of copy-pasting commands, clone the repo first
and then run:

```bash
git clone https://github.com/xsyetopz/go-mamusiabtw.git
cd go-mamusiabtw
./scripts/install-go-on-pi.sh
cp .env.prod.example .env.prod
```

That script:

- detects `aarch64` vs 32-bit ARM
- fetches the latest stable Go version from `go.dev`
- installs it into `/usr/local/go`
- adds `/usr/local/go/bin` to `~/.profile` if needed

Then:

Edit `.env.prod`, then sanity-check:

```bash
go run ./cmd/mamusiabtw doctor
```

Build a release binary:

```bash
./scripts/build-release.sh
```

That produces:

- `./dist/mamusiabtw`

If `doctor` fails:

- fix env first
- do not keep building blindly

## Cross-Build Elsewhere

This is useful if:

- your Pi is weak
- you want faster builds
- you want to prebuild for a Pi Zero 2

This is now much easier because SQLite no longer needs CGO.

Practical recommendation:

- use native builds by default
- use cross-build if you want faster builds on a stronger machine

Keep it simple:

- native build is the normal path
- cross-build is just plain Go cross-build now

Typical targets:

- Pi 4 / Pi 5 on 64-bit OS: `linux/arm64`
- Pi Zero 2 / Pi 3 on 32-bit OS: `linux/arm`

## Deployment Layout

Use a fixed working directory. This project defaults to relative paths like:

- `./data/mamusiabtw.sqlite`
- `./plugins`
- `./config`
- `./migrations/sqlite`
- `./apps/dashboard/dist`

Recommended layout:

```text
/opt/mamusiabtw/
  mamusiabtw
  .env.prod
  migrations/sqlite/
  locales/
  config/
  plugins/
  data/
  apps/dashboard/dist/   # only for Profile C
```

Why this matters:

- the app expects those relative paths unless you override them
- your service must run with `WorkingDirectory=/opt/mamusiabtw`

Minimal rule:

- keep the binary, `.env.prod`, migrations, config, locales, and plugins in one
  predictable directory tree

## Environment Files

Only these env files are supported:

- `.env.dev`
- `.env.prod`

Do not use:

- `.env`
- `.env.production`
- `.env.local`

Rule:

- this repo is strict on env filenames
- do not invent your own convention here

### Profile A: Bot Only

Start from `.env.prod.example` and keep it minimal:

```dotenv
DISCORD_TOKEN=your-token-here

MAMUSIABTW_PROD_MODE=1
MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0

# Optional override
# SQLITE_PATH=./data/mamusiabtw.sqlite

# Optional fallback if owner auto-detection fails
# OWNER_USER_ID=123456789012345678
```

### Profile B: Bot + Admin API On Pi, Dashboard Elsewhere

Example:

```dotenv
DISCORD_TOKEN=your-token-here

MAMUSIABTW_PROD_MODE=1
MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0

MAMUSIABTW_ADMIN_ADDR=0.0.0.0:8081
MAMUSIABTW_DASHBOARD_CLIENT_ID=your-discord-client-id
MAMUSIABTW_DASHBOARD_CLIENT_SECRET=your-discord-client-secret
MAMUSIABTW_DASHBOARD_SESSION_SECRET=use-at-least-32-characters-here

MAMUSIABTW_PUBLIC_DASHBOARD_ORIGIN=https://example.com
MAMUSIABTW_PUBLIC_API_ORIGIN=https://api.example.com
MAMUSIABTW_DASHBOARD_ALLOWED_ORIGINS=https://example.com
```

Use this when:

- the Pi runs the bot and admin API
- the browser dashboard is a static site on another origin

Reality check:

- this is usually the sweet spot for a Pi

### Profile C: Full Stack On The Pi

Example:

```dotenv
DISCORD_TOKEN=your-token-here

MAMUSIABTW_PROD_MODE=1
MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0

MAMUSIABTW_ADMIN_ADDR=0.0.0.0:8081
MAMUSIABTW_DASHBOARD_CLIENT_ID=your-discord-client-id
MAMUSIABTW_DASHBOARD_CLIENT_SECRET=your-discord-client-secret
MAMUSIABTW_DASHBOARD_SESSION_SECRET=use-at-least-32-characters-here

MAMUSIABTW_PUBLIC_DASHBOARD_ORIGIN=https://pi-host-or-public-domain
MAMUSIABTW_PUBLIC_API_ORIGIN=https://pi-host-or-public-domain
MAMUSIABTW_DASHBOARD_ALLOWED_ORIGINS=https://pi-host-or-public-domain
```

Use this when:

- the Pi also serves `apps/dashboard/dist`
- the admin API and dashboard live on the same public origin

Important:

- if `MAMUSIABTW_ADMIN_ADDR` is enabled in prod mode, OAuth/session/public origin
  config must be complete

Reality check:

- this is the easiest runtime setup on Pi 4/5
- it is not the easiest build setup on Pi Zero 2

## Building The Dashboard

This only matters for Profile C.

If you want the cleanest experience:

- build frontend elsewhere for weak boards
- copy `apps/dashboard/dist`
- only run the Go binary on the Pi

Build once:

```bash
cd apps/dashboard
bun install
bun run build
```

At runtime, the Go admin API serves `apps/dashboard/dist` automatically if it
exists.

That means:

- Bun is a build tool here
- Bun is not required to serve the production dashboard

Practical advice:

- Pi Zero 2: build on another machine and copy `apps/dashboard/dist`
- Pi 3: either path is fine, but prebuilding elsewhere is still nicer
- Pi 4/5: local build is reasonable

Do not use the Vite dev server as your normal Raspberry Pi deployment flow.

## systemd Service

Create a dedicated user if you want a cleaner setup:

```bash
sudo useradd --system --home /opt/mamusiabtw --shell /usr/sbin/nologin mamusiabtw
```

Example unit:

```ini
[Unit]
Description=mamusiabtw
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mamusiabtw
Group=mamusiabtw
WorkingDirectory=/opt/mamusiabtw
ExecStart=/opt/mamusiabtw/mamusiabtw
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Why this is enough:

- the binary auto-loads `.env.prod` from the working directory
- the default relative paths line up with the deployment layout above

Low-friction rule:

- do not overcomplicate the service file unless you have a real reason

Enable it:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mamusiabtw
sudo systemctl status mamusiabtw
```

## Networking

### LAN Or Local-Only Use

If you bind:

```dotenv
MAMUSIABTW_ADMIN_ADDR=0.0.0.0:8081
```

Then you can open:

```text
http://<pi-hostname-or-ip>:8081/
```

This is a good fit for:

- private homelab use
- local admin access
- testing Profile C without a public domain first

### Internet-Facing Use

Recommended:

- put a reverse proxy with TLS in front of the Pi
- use a real public domain

Do not expose a raw Pi dashboard/admin API to the public internet without TLS and basic reverse-proxy hygiene.

## Updates

Typical update flow:

```bash
sudo systemctl stop mamusiabtw
cd /opt/mamusiabtw
# pull or copy new binary/assets
sudo systemctl start mamusiabtw
```

If the dashboard changed and you use Profile C:

- update `apps/dashboard/dist` too

Migrations are applied automatically at startup.

Safe habit:

- stop service
- back up SQLite
- update binary/assets
- start service

## Backups

SQLite lives at:

- `SQLITE_PATH`
- default: `./data/mamusiabtw.sqlite`

Before risky upgrades:

- stop the service
- copy the SQLite file somewhere safe

The app also has migration backup support through:

- `MAMUSIABTW_MIGRATION_BACKUPS_DIR`

## Plugin Signing In Production

When:

- `MAMUSIABTW_PROD_MODE=1`

Then:

- plugins must be signed
- `MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0` is the correct production setting

See:

- `docs/reference.md`

For signing commands and trusted key setup.

## Troubleshooting

### Build fails with SQLite or compiler errors

You are probably missing a normal Go build prerequisite, or your Go install is not set up correctly.

Install:

```bash
sudo apt install -y git
```

### Binary will not start: `exec format error`

You built for the wrong architecture.

Check whether your target Pi OS is:

- `linux/arm`
- `linux/arm64`

### Dashboard does not load on the Pi

If you use Profile C:

- make sure `apps/dashboard/dist/index.html` exists in the deployment directory

If it does not exist, the admin API falls back to a Vite proxy flow that is only
for development.

### Admin API is enabled in prod but startup fails

If `MAMUSIABTW_ADMIN_ADDR` is set in prod mode, you must also provide:

- `MAMUSIABTW_DASHBOARD_CLIENT_ID`
- `MAMUSIABTW_DASHBOARD_CLIENT_SECRET`
- `MAMUSIABTW_DASHBOARD_SESSION_SECRET`
- `MAMUSIABTW_PUBLIC_DASHBOARD_ORIGIN`
- `MAMUSIABTW_PUBLIC_API_ORIGIN`
- `MAMUSIABTW_DASHBOARD_ALLOWED_ORIGINS`

### Bot exits with `4014 Disallowed intent(s)`

Discord Developer Portal -> Bot -> Privileged Gateway Intents:

- enable `Server Members Intent`

### Pi Zero 2 feels slow

That is normal during:

- native compilation
- frontend builds
- first-time setup

If you want the least pain on Zero 2:

- use Profile A or Profile B
- prebuild the dashboard elsewhere

## If Raspberry Pi Is Not The Right Fit

Alternatives:

- keep only the bot on the Pi, host the dashboard statically elsewhere
- move the full stack to a small x86 box or VPS
- use Docker on a stronger Pi if you want a more container-shaped deployment

Docker note:

- Docker is now aligned with `.env.prod`
- it is still not the simplest first path for Pi Zero 2
- it is a better fit for Pi 4 / Pi 5 users who already like Docker
