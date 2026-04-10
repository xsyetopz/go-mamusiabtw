# Discord API Coverage (Plugins)

This repo exposes a Discord API surface to Lua plugins via `bot.discord.*`.

The goal is practical completeness: everything Discord supports that is usable
with the bot token and makes sense for plugins. Anything that requires user OAuth
tokens is out of scope unless explicitly added as a separate feature set.

## Current Surface (High Level)

Coverage is implemented in the Lua runtime under `internal/runtime/plugins/lua/`.

**Lookup**

- guild, user, member, role, channel

**Management**

- slowmode, nicknames
- role CRUD + add/remove role from members
- message listing + delete + purge
- emoji CRUD
- sticker CRUD

**Messages**

- get message
- channel send, DM send
- reactions: list/add/remove/clear
- pins: pin/unpin
- crosspost

**Channels / Threads / Invites / Webhooks**

- channel create/edit/delete
- permission overwrite set/delete
- thread create/update + membership operations
- invites create/get/delete + list
- webhooks create/get/edit/delete/list + execute

## How To Extend

1. Add/extend a spec type in the Lua host layer (a `*Spec` struct) so payloads are
   explicit and validated.
2. Add a method to the plugin Discord interface (Go), then wire it through the
   Lua runtime (`bot.discord.*`) with clear names and beginner-readable errors.
3. Add tests that cover:
   - request decoding and validation
   - permission checks (when applicable)
   - any Discord API ordering constraints (for example required options first)

If you add a new family (for example scheduled events, automod, or fuller
interaction coverage), extend this doc with a new section and list the exported
Lua functions.
