# Makefile (complete draft)

Single `TAGS` variable carries the distro-specific webkit tag so other machines can override:
`make build TAGS=` on a webkit2gtk-4.0 system.

```make
APP        := mhtodo
VERSION    ?= 0.1.0
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BIN        := bin/$(APP)
DIST       := dist
FRONTEND   := frontend
PREFIX     ?= $(HOME)/.local

# This distro ships webkit2gtk-4.1 only → Wails v2 needs this tag.
# Note: wails build/dev auto-inject the mode tags (production/dev). Plain `go build`/`go run` for
# GUI code must add them explicitly, e.g. -tags "webkit2_41 production" (M0 finding, 2026-08-19).
# Override: make build TAGS=      (for webkit2gtk-4.0 systems)
TAGS       ?= webkit2_41

GOFLAGS    := -tags $(TAGS)
LDFLAGS    := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: help all dev build test lint fmt tidy fe-install fe-dev         release install uninstall path clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-12s\033[0m %s
", $$1, $$2}'

all: build

## --- development -----------------------------------------------------------

dev: ## wails dev: hot-reload frontend + go (runs GUI)
	wails dev $(GOFLAGS)

fe-install: ## install frontend npm deps
	cd $(FRONTEND) && npm install

fe-dev: ## standalone vite dev server (no webview; for UI work in a browser)
	cd $(FRONTEND) && npm run dev

## --- build -----------------------------------------------------------------

build: ## release-mode local build → bin/mhtodo (builds frontend too)
	wails build $(GOFLAGS) -o $(BIN)

test: ## go tests (core, store, cli golden)
	go test ./...

lint: ## golangci-lint
	golangci-lint run

fmt: ## gofmt + goimports
	gofmt -w -s . && goimports -w -local github.com/yourorg/mhtodo .

tidy: ## go mod tidy
	go mod tidy

## --- release ---------------------------------------------------------------

release: ## cross-build linux amd64+arm64 tarballs → dist/
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 wails build $(GOFLAGS) -o $(BIN)-tmp && 	  tar czf $(DIST)/$(APP)_$(VERSION)_linux_amd64.tar.gz --transform 's|^|$(APP)_$(VERSION)|' 	    $(BIN)-tmp packaging/mhtodo.desktop assets/icon.png README.md && rm $(BIN)-tmp
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc wails build $(GOFLAGS) -o $(BIN)-tmp && 	  tar czf $(DIST)/$(APP)_$(VERSION)_linux_arm64.tar.gz --transform 's|^|$(APP)_$(VERSION)|' 	    $(BIN)-tmp packaging/mhtodo.desktop assets/icon.png README.md && rm $(BIN)-tmp
	@echo "→ $(DIST)/"

## --- install ---------------------------------------------------------------

install: build ## install binary + .desktop + icon into ~/.local
	install -Dm755 $(BIN) $(PREFIX)/bin/$(APP)
	install -Dm644 packaging/mhtodo.desktop $(PREFIX)/share/applications/$(APP).desktop
	install -Dm644 assets/icon.png $(PREFIX)/share/icons/hicolor/256x256/apps/$(APP).png
	-update-desktop-database $(PREFIX)/share/applications 2>/dev/null || true

uninstall: ## remove installed files
	rm -f $(PREFIX)/bin/$(APP) 	      $(PREFIX)/share/applications/$(APP).desktop 	      $(PREFIX)/share/icons/hicolor/256x256/apps/$(APP).png

## --- misc ------------------------------------------------------------------

path: ## print where the DB lives
	@go run $(GOFLAGS) . path 2>/dev/null || echo "$${XDG_DATA_HOME:-$$HOME/.local/share}/$(APP)/$(APP).db"

clean: ## remove build artifacts
	rm -rf bin dist frontend/wailsjs
```

Notes / caveats to resolve during M6:

- **arm64 cross-build** needs an aarch64 gcc toolchain (`gcc-aarch64-linux-gnu`); if absent, `release`
  should degrade gracefully (build amd64 only + warn). Add that guard in the final Makefile.
- `wails build` runs the frontend npm install/build itself; `fe-install` exists for IDE/dev speed.
- Version stamping: `main.go` declares `var version = "dev"` / `commit = "none"`, surfaced by
  `mhtodo --version` and in the GUI About footer.
