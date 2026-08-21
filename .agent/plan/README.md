# mhtodo — Implementation Plan

A todo app in Go with two frontends over one shared core:

- **CLI** (`mhtodo add|list|show|edit|status|done|rm|archive|unarchive`) — for agentic tool access, JSON-friendly.
- **GUI** (Wails v2 webview + system tray) — the human view: board/list views, task detail editing,
  desktop notifications, live sync so CLI changes appear without restart.

One binary, one SQLite database in the XDG data dir. Linux-first. Full details in `AGENTS.md`.

## Goal

Build and maintain mhtodo as a single Go binary with full CLI ↔ GUI parity over one shared core
(`internal/core.Service`), plus system tray, desktop notifications, live sync, packaging, and docs.

## Initial planning discussion

v0.1 (M0–M6) shipped 2026-08-19 and v0.2 (archive/unarchive) shipped 2026-08-20. The detailed
implementation plan for those releases has been **archived** to `.agent/archive/plans/` — this keeps
the active plan folder focused on what comes next rather than a completed build.

This folder is now reset and ready for the next phase of work. Fill in `README.md` with the concrete
goal for the upcoming release, add numbered task detail files (`01-*.md`, …) under it, and track
checkbox progress in `PROGRESS.md`. See `AGENTS.md` for how this folder is organized.

<!-- TODO: state the goal of the next plan here (feature scope, target version/date), then list tasks
     as numbered detail files below. -->

## Files in this plan

| File | Contents |
|---|---|
| `README.md` | Goal of this plan + initial-planning discussion (start here) |
| `PROGRESS.md` | Overall checkbox progress |
| `01-*.md`, … | Per-task detail files (one numbered file per task) |
