# 09 — GUI themes (v0.6)

Named design-token themes (colors, radii, spacing), authored in Settings and
applied at runtime as CSS custom properties. Three built-ins ship seeded;
Slate is the default active theme.

## Decisions (locked)

- **Storage: SQLite**, migration **v13**, table `themes` with a JSON `tokens`
  blob. Active selection in `meta.active_theme_id`. Not `config.yml`.
- **Built-ins editable:** Slate / Paper / Ember can be edited in place.
  **Reset** restores factory tokens. **Duplicate** always available.
  Built-ins **cannot be deleted**.
- **CLI complete.** `mhtodo theme list|search|show|create|update|rm|activate|duplicate|reset`.
- **Tokens:** flat dotted keys validated against a shared registry in
  `internal/core/theme_tokens.go`. Unknown keys rejected; missing keys merged
  from Slate on read.
- **Runtime:** `applyThemeTokens()` sets `--color-*` / `--radius-*` /
  `--spacing-*` on `document.documentElement`. Tailwind `@theme` holds Slate
  fallbacks for first paint.

## Token categories

Surfaces, Borders, Text, Accent, Status, Feedback, Radius, Space — see
`ThemeTokenRegistry` / `THEME_FIELDS`.

## Layers

| Layer | File |
|---|---|
| Migration | `internal/store/migrate.go` (`seedThemesV13`) |
| SQL | `internal/store/themes.go` |
| Domain + rules | `internal/core/theme.go`, `theme_tokens.go` |
| Bound methods | `app.go` (+ `themes:changed`) |
| Shared FE model | `frontend/src/lib/themes.ts` |
| Settings editor | `frontend/src/components/SettingsThemes.svelte` |

## GUI

**Settings → Themes** mirrors Task Templates: second-tier nav, autosave,
flush on unmount, Duplicate / Reset / Delete / Use theme.
