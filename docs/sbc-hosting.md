# SBC Hosting Guide

This guide is for running `mamusiabtw` on small single-board computers:

- Raspberry Pi
- Orange Pi
- ODROID

It is intentionally split by:

- what you host on the device
- what you build on
- how much RAM and CPU headroom the board actually has

## Fast Answer

If you want the short version:

- weakest boards should run **Profile A** or **Profile B**
- stronger boards can run **Profile C**
- weak boards should usually **run** the bot, not **build** the release binary
- after the SQLite CGO removal, cross-build is normal Go cross-build again
- the cleanest production path for tiny boards is:
  build elsewhere, copy binary, run locally

## Table Of Contents

1. Fast answer
2. Profiles
3. Board matrix
4. Host build matrix
5. Exact cross-build commands
6. Raspberry Pi guidance
7. Orange Pi guidance
8. ODROID guidance
9. Deployment layout
10. First install on a fresh device
11. Environment examples
12. Dashboard build guidance
13. Update flow
14. Troubleshooting

## Profiles

### Profile A: Bot Only On Device

Host on device:

- Discord bot
- SQLite

Do not host on device:

- admin API
- dashboard

Best for:

- weakest boards
- lowest RAM use
- lowest setup complexity

### Profile B: Bot + Admin API On Device, Dashboard Elsewhere

Host on device:

- Discord bot
- SQLite
- admin API

Host elsewhere:

- static dashboard site

Best for:

- small boards that can run the API but should not waste effort on frontend hosting
- users who want dashboard access without local frontend build pain

### Profile C: Full Stack On Device

Host on device:

- Discord bot
- SQLite
- admin API
- built `apps/dashboard/dist`

Best for:

- stronger boards
- single-box homelab setups
- LAN/self-hosted dashboards

Important:

- this means built frontend files only
- this does not mean normal Vite dev-server usage

## Board Matrix

Use this as the main pick table.

| Board / Class         | Target arch                                          | RAM class                 | Native build                        | Best profiles | Dashboard build advice                     |
| --------------------- | ---------------------------------------------------- | ------------------------- | ----------------------------------- | ------------- | ------------------------------------------ |
| Raspberry Pi Zero 2 W | `linux/arm64` on 64-bit OS, `linux/arm` on 32-bit OS | 512MB                     | painful                             | A, B          | prebuild elsewhere                         |
| Orange Pi Zero 2W     | `linux/arm64`                                        | 1GB / 1.5GB / 2GB / 4GB   | weak-to-acceptable depending on RAM | A, B          | prebuild elsewhere                         |
| Raspberry Pi 3        | `linux/arm` or `linux/arm64`                         | 1GB                       | acceptable but slow                 | A, B          | prebuild elsewhere                         |
| ODROID M1S            | `linux/arm64`                                        | 4GB / 8GB                 | good                                | A, B, C       | local build is fine; prebuild still faster |
| Raspberry Pi 4        | `linux/arm64` preferred                              | 2GB / 4GB / 8GB           | good                                | A, B, C       | local build is fine                        |
| Raspberry Pi 5        | `linux/arm64`                                        | 4GB / 8GB                 | very good                           | A, B, C       | local build is fine                        |
| Orange Pi 5 family    | `linux/arm64`                                        | 4GB / 8GB / 16GB / higher | very good                           | A, B, C       | local build is fine                        |

Rule for unmapped boards:

- if it feels like Zero-class hardware, treat it like Zero 2 / Zero 2W
- if it feels like Pi 3 / RK3566 class hardware, treat it like Pi 3 / ODROID M1S
- if it has 4GB+ and a modern 64-bit SoC, treat it like Pi 4/5 or Orange Pi 5 class

Official representative board pages:

- Raspberry Pi Zero 2 W:
  <https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/>
- Orange Pi Zero 2W:
  <https://www.orangepi.org/orangepiwiki/index.php/Orange_Pi_Zero_2W>
- Orange Pi 5:
  <https://www.orangepi.org/html/hardWare/computerAndMicrocontrollers/details/Orange-Pi-5.html>
- ODROID M1S:
  <https://www.hardkernel.com/blog-2/odroid-m1s/>

## Host Build Matrix

This is about the machine doing the build, not the machine running the bot.

| Build host                   | Good for                             | Recommendation                                   |
| ---------------------------- | ------------------------------------ | ------------------------------------------------ |
| `linux/amd64` / macOS x86_64 | `linux/arm64`, `linux/arm`           | best general build host                          |
| `linux/arm64` / macOS arm64  | `linux/arm64`, `linux/arm`           | also excellent                                   |
| `linux/arm`                  | `linux/arm`, sometimes `linux/arm64` | possible, but not the preferred cross-build host |

Important:

- after the CGO removal, the toolchain complexity is much lower
- the main choice is now target architecture, not C cross-compilers
- host architecture mostly affects speed and memory headroom

## Exact Cross-Build Commands

These are the main release commands you want.

### Build For 64-bit ARM SBCs

Examples:

- Raspberry Pi 4 / 5 on 64-bit OS
- Orange Pi Zero 2W
- Orange Pi 5
- ODROID M1S

```bash
GOOS=linux GOARCH=arm64 ./scripts/build-release.sh dist/mamusiabtw-linux-arm64
```

### Build For 32-bit ARMv7 SBCs

Examples:

- Raspberry Pi 3 on 32-bit OS
- Raspberry Pi Zero 2 W on 32-bit Raspberry Pi OS

```bash
GOOS=linux GOARCH=arm GOARM=7 ./scripts/build-release.sh dist/mamusiabtw-linux-armv7
```

### Copy To The Device

For an already-prepared device, copy to a user-writable path first:

```bash
rsync -a dist/mamusiabtw-linux-arm64 krystian@device:~/mamusiabtw
ssh krystian@device 'sudo install -Dm755 ~/mamusiabtw /opt/mamusiabtw/mamusiabtw'
```

For 32-bit targets:

```bash
rsync -a dist/mamusiabtw-linux-armv7 krystian@device:~/mamusiabtw
ssh krystian@device 'sudo install -Dm755 ~/mamusiabtw /opt/mamusiabtw/mamusiabtw'
```

### Restart The Service

Example:

```bash
ssh krystian@device 'sudo systemctl restart mamusiabtw && sudo systemctl status mamusiabtw --no-pager'
```

### Practical Rule

- weak SBCs should usually receive a copied binary
- stronger SBCs can build locally if you prefer
- the device does not need to be the build machine

## Raspberry Pi Guidance

### Raspberry Pi Zero 2 W

Recommended:

- Profile A
- Profile B

Not recommended:

- local release builds as the normal workflow
- Profile C unless you really know why

Why:

- it is a great runtime target
- it is not a comfortable release-build box
- low RAM makes `modernc` compile spikes painful

Best workflow:

- cross-build on x86_64 or arm64
- copy binary to the device
- only prebuild dashboard elsewhere if you insist on Profile C

If you need Go installed on Raspberry Pi OS itself:

```bash
./scripts/install-go-on-pi.sh
```

That helper is Raspberry Pi OS-oriented and only covers the Raspberry Pi archive naming cases.

### Raspberry Pi 3

Recommended:

- Profile A
- Profile B

Acceptable:

- Profile C with a prebuilt dashboard

Why:

- decent runtime box
- still not a machine you should force into repeated heavy builds

### Raspberry Pi 4

Recommended:

- all profiles

Why:

- first Raspberry Pi tier where full-stack hosting feels normal

### Raspberry Pi 5

Recommended:

- all profiles

Why:

- easiest Raspberry Pi for local builds and one-box hosting

## Orange Pi Guidance

### Orange Pi Zero 2W

Recommended:

- Profile A
- Profile B

Possible:

- Profile C only on higher-RAM variants, and even then it is not the first choice

Why:

- still a small board class
- stronger than Zero-class only in some RAM variants, not in “build anything comfortably” terms

### Orange Pi 5 Family

Recommended:

- all profiles

Why:

- this is firmly in the “comfortable arm64 board” class
- good fit for local builds, full dashboard hosting, and one-box deployments

## ODROID Guidance

### ODROID M1S

Recommended:

- Profile A
- Profile B
- Profile C if you want a single-box deployment

Why:

- RK3566 + 4GB/8GB RAM puts it above the weak-board class
- more realistic for local builds than Zero/Pi 3 class hardware

Practical note:

- if you want the smoothest experience, still prebuild the dashboard elsewhere

## Deployment Layout

Use one predictable working directory, for example:

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

Why:

- the repo uses relative defaults like `./data`, `./plugins`, `./config`, and `./apps/dashboard/dist`
- the service should run with `WorkingDirectory=/opt/mamusiabtw`

## First Install On A Fresh Device

This is the missing part that must happen before any update or restart flow.

### 1. Create The Install Tree

```bash
sudo useradd --system --home /opt/mamusiabtw --shell /usr/sbin/nologin mamusiabtw || true
sudo install -d -o mamusiabtw -g mamusiabtw /opt/mamusiabtw
sudo install -d -o mamusiabtw -g mamusiabtw /opt/mamusiabtw/data
```

### 2. Copy The Repo Assets You Need

From your build machine:

```bash
rsync -a dist/mamusiabtw-linux-arm64 user@device:~/mamusiabtw
rsync -a migrations locales plugins config user@device:~/
scp .env.prod user@device:~/.env.prod
```

Then on the device:

```bash
sudo install -Dm755 ~/mamusiabtw /opt/mamusiabtw/mamusiabtw
sudo rsync -a ~/migrations/ /opt/mamusiabtw/migrations/
sudo rsync -a ~/locales/ /opt/mamusiabtw/locales/
sudo rsync -a ~/plugins/ /opt/mamusiabtw/plugins/
sudo rsync -a ~/config/ /opt/mamusiabtw/config/
sudo install -Dm600 ~/.env.prod /opt/mamusiabtw/.env.prod
sudo chown -R mamusiabtw:mamusiabtw /opt/mamusiabtw
```

For 32-bit targets, replace the binary name with `dist/mamusiabtw-linux-armv7`.

### 3. Install The Service

Create `/etc/systemd/system/mamusiabtw.service`:

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
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mamusiabtw
sudo systemctl status mamusiabtw --no-pager
```

### 4. Check The Installed Binary

Run `doctor` against the installed binary itself:

```bash
/opt/mamusiabtw/mamusiabtw doctor
```

Important:

- `doctor` should be run from the installed path for production checks
- the binary now looks for `.env.prod` in the current directory and next to the executable
- if `/opt/mamusiabtw/.env.prod` exists, `doctor` from `/opt/mamusiabtw/mamusiabtw` should see it

## Environment Examples

Only these env filenames are supported:

- `.env.dev`
- `.env.prod`

### Profile A

```dotenv
DISCORD_TOKEN=your-token-here

MAMUSIABTW_PROD_MODE=1
MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0
```

### Profile B

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

### Profile C

```dotenv
DISCORD_TOKEN=your-token-here

MAMUSIABTW_PROD_MODE=1
MAMUSIABTW_ALLOW_UNSIGNED_PLUGINS=0

MAMUSIABTW_ADMIN_ADDR=0.0.0.0:8081
MAMUSIABTW_DASHBOARD_CLIENT_ID=your-discord-client-id
MAMUSIABTW_DASHBOARD_CLIENT_SECRET=your-discord-client-secret
MAMUSIABTW_DASHBOARD_SESSION_SECRET=use-at-least-32-characters-here

MAMUSIABTW_PUBLIC_DASHBOARD_ORIGIN=https://device-or-domain.example
MAMUSIABTW_PUBLIC_API_ORIGIN=https://device-or-domain.example
MAMUSIABTW_DASHBOARD_ALLOWED_ORIGINS=https://device-or-domain.example
```

## Dashboard Build Guidance

This only matters for Profile C.

Best rule:

- build frontend elsewhere for weak boards
- copy `apps/dashboard/dist`
- run only the Go binary on the target

Build once:

```bash
cd apps/dashboard
bun install
bun run build
```

At runtime:

- the admin API serves `apps/dashboard/dist` automatically if it exists
- Bun is not needed to serve the production dashboard

## Update Flow

### If The Device Is Also The Build Machine

```bash
cd /opt/mamusiabtw
cp .env.prod .env.prod.backup
git pull --ff-only
diff -u .env.prod.example .env.prod
./scripts/build-release.sh
sudo systemctl restart mamusiabtw
```

### If You Cross-Build Elsewhere

```bash
git pull --ff-only
GOOS=linux GOARCH=arm64 ./scripts/build-release.sh dist/mamusiabtw-linux-arm64
rsync -a dist/mamusiabtw-linux-arm64 user@device:~/mamusiabtw
ssh user@device 'sudo install -Dm755 ~/mamusiabtw /opt/mamusiabtw/mamusiabtw'
ssh user@device 'sudo systemctl restart mamusiabtw'
```

For 32-bit targets:

```bash
git pull --ff-only
GOOS=linux GOARCH=arm GOARM=7 ./scripts/build-release.sh dist/mamusiabtw-linux-armv7
rsync -a dist/mamusiabtw-linux-armv7 user@device:~/mamusiabtw
ssh user@device 'sudo install -Dm755 ~/mamusiabtw /opt/mamusiabtw/mamusiabtw'
ssh user@device 'sudo systemctl restart mamusiabtw'
```

## Troubleshooting

### `compile: signal: killed`

That usually means the board ran out of memory during compile.

Most common fix:

- stop native release building on that board
- cross-build on a stronger machine

For boards like Pi Zero 2 W, that should be your normal expectation.

### `exec format error`

You built for the wrong target.

Check whether the device OS is:

- `linux/arm64`
- `linux/arm`

### Dashboard Does Not Load

If you use Profile C:

- make sure `apps/dashboard/dist/index.html` exists on the target

### Admin API In Prod Fails At Startup

If `MAMUSIABTW_ADMIN_ADDR` is set in prod mode, make sure the required dashboard OAuth/session/public-origin vars are complete.

### `doctor` Says `discord_token: false`

Check these in order:

- does `/opt/mamusiabtw/.env.prod` exist
- are you running `/opt/mamusiabtw/mamusiabtw doctor`
- does `.env.prod` actually contain `DISCORD_TOKEN=...`
- if needed, run `doctor` from `/opt/mamusiabtw` as well

The production doctor path should prefer `.env.prod`, not `.env.dev`.

### `systemctl restart mamusiabtw` Says Unit Not Found

That means first-install service setup was never completed.

You need to:

- create `/etc/systemd/system/mamusiabtw.service`
- run `sudo systemctl daemon-reload`
- run `sudo systemctl enable --now mamusiabtw`

### `rsync` To `/opt/mamusiabtw/...` Fails

That usually means one of these is true:

- `/opt/mamusiabtw` does not exist yet
- your SSH user cannot write there directly

Use the documented safe flow:

- rsync to `~/mamusiabtw`
- `sudo install -Dm755` into `/opt/mamusiabtw/mamusiabtw`

### Weak Board Feels Miserable To Build On

That is normal.

Use this rule:

- weak board: run it there, build elsewhere
- strong board: either is fine
