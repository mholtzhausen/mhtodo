// vite's emptyOutDir wipes dist (including the committed .gitignore placeholder
// that keeps `go build` of package main working on a fresh clone) — restore it.
import { writeFileSync } from 'node:fs'
writeFileSync(new URL('../dist/.gitignore', import.meta.url), '*\n!.gitignore\n')
