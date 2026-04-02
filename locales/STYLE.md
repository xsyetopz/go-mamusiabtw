# Locale Style Guide (imotherbtw)

This folder contains user-facing translations for Mommy.

## Tone Marks (… / ~ / locale equivalents)

Mommy can be expressive, but she should never be noisy or spammy.

### Where it's OK

- Runtime replies, prompts, and errors (e.g. `ok.*`, `err.*`, `mod.*`, `mgr.*`, `wellness.*`, `info.*`).

### Where to keep it clean

- Slash command metadata (anything under `cmd.*`), especially:
  - command names/descriptions
  - option/subcommand names/descriptions

### Rules of thumb

- Prefer *at most one* tone mark per short sentence.
- Avoid stacking (`...~~`, `……～～`).
- Don't add new template placeholders when editing translations.
- Keep the persona wholesome (nurturing/protective). No romantic/sexual framing.

### Locale preferences

- `en-US`, `en-GB`: `...` + occasional `~`.
- `ja`: prefer `…` and `〜` (avoid ASCII `~`).
- `zh-CN`, `zh-TW`: prefer `…`/`……` and `～` (avoid ASCII `~`).
- `ko`: `...` + occasional `~`.
- Most other locales: prefer ellipses over tildes; keep `~` rare.

## Kaomoji

Kaomoji are allowed, sparingly, in a few high-impact runtime messages.

- Keep the set tiny and re-use the same ones.
- Prefer using them in `ja`/`zh-*`/`ko` locales; English can use them too if it reads naturally.

