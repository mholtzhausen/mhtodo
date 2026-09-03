# 08 — Task templates (v0.5)

Named sets of task presets, authored in Settings and applied when creating a task.

## Decisions (locked)

- **Storage: SQLite**, migration **v9**, table `task_templates`. Not `config.yml`, even
  though templates are settings-shaped — they are user data with IDs, uniqueness and
  ordering, and they belong next to tasks.
- **No CLI commands in this slice.** All rules live in `core.Service`, so
  `mhtodo template list|show|add|edit|rm` and `mhtodo add --template NAME` are a thin
  later add. This is a deliberate, temporary deviation from the AGENTS.md parity rule;
  `Service.CreateFromTemplate` already exists so the CLI has nothing to reimplement.
- **Fields:** `title_prefix`, `description`, `status`, `cwd`, `slack_thread`,
  `human_only`, `include_in_report`.

## Core semantics: unset vs empty

Every preset column is **NULLable on purpose**. `NULL` means "not part of this template",
so task creation falls back to the normal default (the GUI-settings defaults for cwd /
human-only / include-in-report, or core's own). A present-but-empty string or an explicit
`false` is a real value that *overrides* that default.

This is why `task_templates` does not follow the `tasks` convention of `NOT NULL` with
empty-string defaults (migrations v6–v8): for a task, "" and unset are the same thing; for
a template they are not.

```
template.cwd = NULL   → new task gets settings.default_cwd
template.cwd = ""     → new task gets no working directory
template.cwd = "/x"   → new task gets /x
```

`Template.Apply(CreateInput) CreateInput` is the single place this overlay is expressed in
Go; `applyTemplate()` in `frontend/src/lib/templates.ts` mirrors it for the live form.

## Update is full replace, not patch

`Service.UpdateTemplate(ref, TemplateInput)` replaces every preset: a nil field clears it.
The settings editor always submits the whole template, so the "trash-can clears a field"
behaviour is a plain nil rather than a double pointer (`**string`).

## Layers

| Layer | File |
|---|---|
| Migration | `internal/store/migrate.go` (`schemaV9`) |
| SQL | `internal/store/templates.go` |
| Domain + rules | `internal/core/template.go` |
| Repo interface | `internal/core/service.go` (`TaskRepository`) |
| Bound methods | `app.go` (`ListTemplates`, `GetTemplate`, `CreateTemplate`, `UpdateTemplate`, `DeleteTemplate`) |
| Shared FE model | `frontend/src/lib/templates.ts` (`TEMPLATE_FIELDS` is the single source of truth) |

Name uniqueness is enforced by a `COLLATE NOCASE UNIQUE` column and surfaced as
`core.DuplicateTemplateNameError` — the store maps SQLite's `SQLITE_CONSTRAINT_UNIQUE`
rather than matching on message text.

Template mutations emit a **`templates:changed`** Wails event, kept separate from
`tasks:changed` so they do not trigger a task reload or a tray recount.

## GUI

**Settings → Task Templates** is the only section with a second nav tier: each template is
a sub-item, always visible, plus `+ New template`. The editor autosaves on a 400 ms debounce
through its own path (not the whole-settings `persist()`), and its status is merged into the
shared header Saving/Unsaved/Saved indicator. Writes are serialized through a promise chain,
and a pending edit is flushed on unmount so switching templates or closing the dialog cannot
drop it.

**Tri-state booleans.** `human_only` and `include_in_report` must distinguish unset / false /
true, which a native checkbox cannot do: `indeterminate` reads as "partially true" rather
than "not set", renders in WebKitGTK's default style instead of the accent-tinted style used
elsewhere, and clicking it jumps straight to true. `TriStateCheck.svelte` is a
`<button role="checkbox">` with `aria-checked="mixed"` for unset, cycling
`unset → on → off → unset`, and always naming its state in adjacent text so the three states
never depend on reading the glyph alone. Because the cycle reaches unset in at most two
clicks, boolean rows have no trash-can; text/status rows do.

**Save as template** (`SaveAsTemplateDialog.svelte`) needs no tri-state: its per-field
checkbox *is* the null selector, so a boolean gets an include checkbox plus an ordinary value
checkbox. Reachable from the new-task dialog header and the task-detail header.

**Applying** (`TemplatePicker.svelte`) is a popover under the template icon in the new-task
dialog header: filter input autofocused, `/` refocuses it from the list, arrows navigate,
Enter applies. Rows show chips for the fields the template presets; boolean chips carry the
value (`human-only: off`), since a template forcing a flag off would otherwise look identical
to one forcing it on. On apply, the title is focused with the caret **after** the prefix, and
switching templates swaps the previous prefix instead of stacking a second one.

**Entry points:** header split button (right half opens with the picker up), tray
`New Task from Template`, and the picker icon inside the dialog. `App.openNewTask()` is the
single funnel for all of them.

## Gotchas

- The global `/` shortcut in `App.svelte` now returns early while the create dialog is open,
  so it cannot steal focus to the board search from behind the modal.
- `wails generate module` refuses to run while a GUI instance holds the single-instance lock;
  the template bindings in `frontend/wailsjs/` were written by hand (see PROGRESS notes).
- `go fmt ./...` reformats files that were never gofmt-clean. Format only the files you touch.
