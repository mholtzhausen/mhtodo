---
layout: page
title: About
permalink: /about/
---

**mhtodo** is a personal task manager written in Go. One binary, two frontends over one shared core:

- **CLI** — scriptable, `--json` everywhere; the interface for agentic tool access.
- **GUI** — Wails webview + system tray: board and list views, live sync with the CLI.

Both call the same `core.Service`. Data lives in one SQLite database (`$XDG_DATA_HOME/mhtodo/mhtodo.db`).

- [GitHub repository](https://github.com/mholtzhausen/mhtodo)
- [Install from source or release](https://github.com/mholtzhausen/mhtodo#install)
- Agent skill: run `mhtodo ai` to install/update `~/.claude/skills/mhtodo/`
