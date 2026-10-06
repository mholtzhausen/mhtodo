# 06 — Review status + Pull Request lane

- Order: pending → wip → waiting → **review** → **pr** → done
- Board columns; FilterBar chip; StatusPicker (6 statuses)
- Keys 1–6 statuses; **7** archived (list)
- CSS `--color-st-review`, `--color-st-pr`; no desktop notify on →review / →pr by default
- `pr_url` field: empty→non-empty advances to `pr`; PR column collapsed by default (`mhtodo.collapsedColumns.v2`)
