---
layout: post
title: "2.13.0 — Working directory typeahead"
date: 2026-10-06 17:30:00 +0000
categories: [release]
tags: [gui, templates, cwd, productivity]
---

Setting a ticket’s working directory used to mean typing the full path or hunting through the folder picker. Templates already know common project roots — and so do the tickets you have already filed. **2.13.0** puts both in a typeahead on the cwd field.

## What you get

On **new task** and **task detail**, the working directory field still accepts a free-typed path and the folder button. Focus the field and a warm suggestion list opens immediately (no fetch lag):

- **Template rows** — template name on top, cwd path underneath
- **History rows** — the same two-line layout with an empty name line, for paths used on earlier tickets that aren’t already covered by a template

Type to fuzzy-filter. Matches against **template names** rank above path-only hits, so a short fragment of a template name surfaces the right project quickly. Arrow keys, Enter, and Esc work the same way as the template picker.

The list is built when the app starts and stays current when tasks or templates change (`tasks:changed` / `templates:changed`), so create and edit stay snappy.

Selecting a suggestion **only sets cwd** — it does not apply the rest of the template. Full template apply remains the template picker on create.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 2.13.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#2130-5aca2cd) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.13.0).
