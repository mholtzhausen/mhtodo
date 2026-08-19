APP        := mhtodo
VERSION    ?= 0.1.0
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BIN        := bin/$(APP)
DIST       := dist
FRONTEND   := frontend
PREFIX     ?= $(HOME)/.local

# This distro ships webkit2gtk-4.1 only → Wails v2 needs this tag (M0 finding).
# GUI builds also need a *mode* tag: wails build/dev inject production/dev automatically;
# plain `go build` must add it explicitly — hence "$(TAGS) production" below.
# Override on other systems: make build TAGS=   (webkit2gtk-4.0) or TAGS="webkit2_41".
TAGS       ?= webkit2_41

# Version stamping (06-makefile.md): raw ldflags content, passed as -ldflags "$(STAMP)".
STAMP      := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
GOFLAGS    := -tags $(TAGS)

# arm64 cross-build needs an aarch64 C toolchain (CGO: gtk/webkit). If absent,
# `release` degrades gracefully to amd64-only with a warning (M6 guard).
AARCH64_CC ?= aarch64-linux-gnu-gcc
LINTER     := $(shell command -v golangci-lint 2>/dev/null)
GOFORMT    := $(shell command -v gofmt || echo "$$(go env GOROOT)/bin/gofmt")

.PHONY: help all dev build test lint fmt tidy fe-install fe-bindings fe-build fe-dev release install uninstall path clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

all: build

## --- development -----------------------------------------------------------

dev: ## wails dev: hot-reload frontend + go (runs GUI)
	wails dev $(GOFLAGS)

fe-install: ## install frontend npm deps
	cd $(FRONTEND) && npm install

fe-bindings: ## regenerate wailsjs bindings from Go bound methods
	wails generate module -tags $(TAGS)

fe-build: fe-bindings fe-install ## build the Vite frontend → frontend/dist
	cd $(FRONTEND) && npm run build

fe-dev: fe-install ## standalone vite dev server (no webview; for UI work in a browser)
	cd $(FRONTEND) && npm run dev

## --- build -----------------------------------------------------------------

# wails build injects the production mode tag itself and runs the frontend
# install/build. wails build writes to build/bin/<name>; copy it out to bin/
# for install/release.
build: ## release-mode local build → bin/mhtodo (wails build runs the frontend too)
	wails build $(GOFLAGS) -ldflags "$(STAMP)" -o $(APP)
	install -Dm755 build/bin/$(APP) $(BIN)

test: ## go tests (core, store, cli golden, instance lock)
	go test ./...

lint: ## golangci-lint if installed, else go vet
ifeq ($(LINTER),)
	go vet ./...
else
	$(LINTER) run
endif

fmt: ## gofmt -s
	$(GOFORMT) -w -s .

tidy: ## go mod tidy
	go mod tidy

path: ## print where the DB lives
	@go run $(GOFLAGS) . path 2>/dev/null || echo "$${XDG_DATA_HOME:-$$HOME/.local/share}/$(APP)/$(APP).db"

## --- release ---------------------------------------------------------------

# Cross-builds linux tarballs into dist/. wails build writes to build/bin/<name>;
# we stage the binary + packaging files and tar them under <app>_<version>/.
# arm64 requires $(AARCH64_CC); without it we warn and ship amd64 only.
release: ## cross-build linux tarballs → dist/ (arm64 needs $(AARCH64_CC))
	@mkdir -p $(DIST)/stage
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 wails build $(GOFLAGS) -ldflags "$(STAMP)" -o mhtodo-amd64
	cp build/bin/mhtodo-amd64 $(DIST)/stage/$(APP) && chmod 755 $(DIST)/stage/$(APP)
	cp packaging/$(APP).desktop assets/icon.png README.md $(DIST)/stage/
	tar czf $(DIST)/$(APP)_$(VERSION)_linux_amd64.tar.gz --transform 's|^|$(APP)_$(VERSION)/|' \
	    -C $(DIST)/stage $(APP) $(APP).desktop icon.png README.md
	@if command -v $(AARCH64_CC) >/dev/null 2>&1; then \
	  echo "→ arm64 (CC=$(AARCH64_CC))"; \
	  GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=$(AARCH64_CC) wails build $(GOFLAGS) -ldflags "$(STAMP)" -o mhtodo-arm64; \
	  cp build/bin/mhtodo-arm64 $(DIST)/stage/$(APP); \
	  tar czf $(DIST)/$(APP)_$(VERSION)_linux_arm64.tar.gz --transform 's|^|$(APP)_$(VERSION)/|' \
	    -C $(DIST)/stage $(APP) $(APP).desktop icon.png README.md; \
	else \
	  echo "warning: $(AARCH64_CC) not found — skipping arm64 (install gcc-aarch64-linux-gnu to enable)"; \
	fi
	@rm -rf $(DIST)/stage
	@echo "→ $(DIST)/"

## --- install ---------------------------------------------------------------

# Installs into $(PREFIX) (default ~/.local): binary on PATH, .desktop in the
# user applications dir, icon in hicolor. update-desktop-database refreshes the
# launcher menu; a running desktop picks it up without a restart.
install: build ## install binary + .desktop + icon into $(PREFIX) (~/.local)
	install -Dm755 $(BIN) $(PREFIX)/bin/$(APP)
	install -Dm644 packaging/$(APP).desktop $(PREFIX)/share/applications/$(APP).desktop
	install -Dm644 assets/icon.png $(PREFIX)/share/icons/hicolor/512x512/apps/$(APP).png
	-update-desktop-database $(PREFIX)/share/applications 2>/dev/null || true

uninstall: ## remove installed files
	rm -f $(PREFIX)/bin/$(APP) \
	      $(PREFIX)/share/applications/$(APP).desktop \
	      $(PREFIX)/share/icons/hicolor/512x512/apps/$(APP).png
	-update-desktop-database $(PREFIX)/share/applications 2>/dev/null || true

clean: ## remove build artifacts
	rm -rf bin dist frontend/wailsjs
