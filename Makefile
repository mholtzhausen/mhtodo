APP        := mhtodo
VERSION    ?= 2.0.0
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

.PHONY: help all dev build test lint fmt tidy fe-install fe-bindings fe-build fe-dev release _bump release-tag publish install install-files uninstall service-install service-remove path clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

all: build

## --- development -----------------------------------------------------------

dev: ## wails dev: hot-reload (default start hidden; override in Settings → General)
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
	@if command -v zip >/dev/null 2>&1; then \
	  echo "→ zip (amd64)"; \
	  rm -rf $(DIST)/ztmp; mkdir -p $(DIST)/ztmp/$(APP)_$(VERSION); \
	  cp $(DIST)/stage/$(APP) $(DIST)/stage/$(APP).desktop $(DIST)/stage/icon.png $(DIST)/stage/README.md $(DIST)/ztmp/$(APP)_$(VERSION)/; \
	  ( cd $(DIST)/ztmp && zip -r -q ../$(APP)_$(VERSION)_linux_amd64.zip $(APP)_$(VERSION) ); \
	  rm -rf $(DIST)/ztmp; \
	else \
	  echo "warning: zip not found — skipping .zip asset"; \
	fi
	@if command -v $(AARCH64_CC) >/dev/null 2>&1; then \
	  echo "→ arm64 (CC=$(AARCH64_CC))"; \
	  GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=$(AARCH64_CC) wails build $(GOFLAGS) -ldflags "$(STAMP)" -o mhtodo-arm64; \
	  cp build/bin/mhtodo-arm64 $(DIST)/stage/$(APP); \
	  tar czf $(DIST)/$(APP)_$(VERSION)_linux_arm64.tar.gz --transform 's|^|$(APP)_$(VERSION)/|' \
	    -C $(DIST)/stage $(APP) $(APP).desktop icon.png README.md; \
	  if command -v zip >/dev/null 2>&1; then \
	    rm -rf $(DIST)/ztmp; mkdir -p $(DIST)/ztmp/$(APP)_$(VERSION); \
	    cp $(DIST)/stage/$(APP) $(DIST)/stage/$(APP).desktop $(DIST)/stage/icon.png $(DIST)/stage/README.md $(DIST)/ztmp/$(APP)_$(VERSION)/; \
	    ( cd $(DIST)/ztmp && zip -r -q ../$(APP)_$(VERSION)_linux_arm64.zip $(APP)_$(VERSION) ); \
	    rm -rf $(DIST)/ztmp; \
	  fi; \
	else \
	  echo "warning: $(AARCH64_CC) not found — skipping arm64 (install gcc-aarch64-linux-gnu to enable)"; \
	fi
	@rm -rf $(DIST)/stage
	@echo "→ $(DIST)/"

# --- release process -------------------------------------------------------
# Full flow (interactive):    make release-tag            # asks major/minor/patch
# Non-interactive:            make release-tag BUMP=minor
#   _bump     prompt for major/minor/patch (unless BUMP is set) → update VERSION in this
#             file, commit the Makefile, tag v<new>
#   publish   re-run with the new $(VERSION): cross-build tarballs (release), gh release
#             create v<new> with them as assets, push main + tag to origin

BUMP ?=    # major | minor | patch — omit to be asked interactively by _bump

_bump: ## @internal — bump VERSION in this file by BUMP (prompt if unset); commit Makefile; tag
	@if [ -n "$(BUMP)" ]; then pick="$(BUMP)"; else \
	  printf "\nbump version  [%s]ajor / [m]inor / [p]atch (default p): " M < /dev/tty; \
	  read -r pick < /dev/tty || { echo "no tty — re-run with BUMP=major|minor|patch"; exit 1; }; fi; \
	pick="$${pick:-p}"; old="$(VERSION)"; \
	major=$$(printf %s "$$old" | cut -d. -f1); minor=$$(printf %s "$$old" | cut -d. -f2); patch=$$(printf %s "$$old" | cut -d. -f3); \
	case "$$pick" in \
	  M|maj*) major=$$((major+1)); minor=0; patch=0 ;; \
	  m|mi*)  minor=$$((minor+1)); patch=0 ;; \
	  p|pa*)  patch=$$((patch+1)) ;; \
	  *) echo "bad bump: '$$(pick)' (want major | minor | patch)"; exit 2 ;; \
	esac; new="$$major.$$minor.$$patch"; echo "$$old → $$new"; \
	sed -i "s|^VERSION.*|VERSION    ?= $$new|" $(firstword $(MAKEFILE_LIST)); \
	git add -- $(firstword $(MAKEFILE_LIST)) && git commit -m "v$$new"; \
	git tag "v$$new"

release-tag: ## prompt major/minor/patch (or BUMP=...) → bump+tag → build tarballs → publish GitHub Release
	@$(MAKE) _bump
	@$(MAKE) -f $(firstword $(MAKEFILE_LIST)) publish

publish: release ## cross-build, then gh release create v$(VERSION) + push main and the tag to origin
	@command -v gh >/dev/null 2>&1 || { echo "error: gh not found — install GitHub CLI"; exit 1; }
	@gh auth status >/dev/null 2>&1 || { echo "error: gh is not authenticated — run 'gh auth login'"; exit 1; }
	@git push origin HEAD:main 2>/dev/null || true
	@git push origin "v$(VERSION)" 2>/dev/null || echo "note: tag v$(VERSION) not pushed (already on remote?); continuing"
	@assets=$$(ls $(DIST)/$(APP)_$(VERSION)_linux_*.tar.gz $(DIST)/$(APP)_$(VERSION)_linux_*.zip 2>/dev/null); \
	gh release create v$(VERSION) --title "v$(VERSION)" \
	  --notes "$(APP) v$(VERSION) (linux amd64$([ -f $(DIST)/$(APP)_$(VERSION)_linux_arm64.tar.gz ] && echo ', arm64'))" \
	$$assets
	@remote="$$(git remote get-url origin 2>/dev/null || true)"; repo_url=""; \
	case "$$remote" in \
	  git@github.com:*/*) repo_url="https://github.com/$${remote#git@github.com:}" ;; \
	  https://github.com/*) repo_url="$$remote" ;; \
	esac; repo_url="$${repo_url%.git}"; \
	[ -n "$$repo_url" ] && echo "→ published $$repo_url/releases/tag/v$(VERSION)" || echo "→ published v$(VERSION)"

## --- install ---------------------------------------------------------------

# Installs into $(PREFIX) (default ~/.local): binary on PATH, .desktop in the
# user applications dir, icon in hicolor. update-desktop-database refreshes the
# launcher menu; a running desktop picks it up without a restart.
install-files: ## @ internal — copy binary + .desktop + icon into $(PREFIX)
	install -Dm755 $(BIN) $(PREFIX)/bin/$(APP)
	install -Dm644 packaging/$(APP).desktop $(PREFIX)/share/applications/$(APP).desktop
	install -Dm644 assets/icon.png $(PREFIX)/share/icons/hicolor/512x512/apps/$(APP).png
	-update-desktop-database $(PREFIX)/share/applications 2>/dev/null || true

install: build install-files ## install binary + .desktop + icon into $(PREFIX) (~/.local)

uninstall: ## remove installed files
	rm -f $(PREFIX)/bin/$(APP) \
	      $(PREFIX)/share/applications/$(APP).desktop \
	      $(PREFIX)/share/icons/hicolor/512x512/apps/$(APP).png
	-update-desktop-database $(PREFIX)/share/applications 2>/dev/null || true

## --- service ---------------------------------------------------------------

# User-level systemd unit that keeps the GUI (tray app) running across logins.
# Re-running service-install replaces a previous install: stop → overwrite
# binary/files → rewrite unit → daemon-reload → enable --now (fresh start).
SERVICE_UNIT  := $(APP).service
USER_UNIT_DIR ?= $(HOME)/.config/systemd/user

service-install: build ## build + install, then run as user systemd service (replaces if present)
	@systemctl --user stop $(SERVICE_UNIT) 2>/dev/null || true
	$(MAKE) install-files
	@mkdir -p $(USER_UNIT_DIR)
	@{ \
	  echo "[Unit]"; \
	  echo "Description=mhtodo — todo manager (GUI + system tray)"; \
	  echo "After=graphical-session.target"; \
	  echo ""; \
	  echo "[Service]"; \
	  echo "Type=simple"; \
	  echo "ExecStart=$(PREFIX)/bin/$(APP) gui"; \
	  echo "Restart=on-failure"; \
	  echo "RestartSec=2"; \
	  echo ""; \
	  echo "[Install]"; \
	  echo "WantedBy=default.target"; \
	} > $(USER_UNIT_DIR)/$(SERVICE_UNIT)
	@for v in DISPLAY WAYLAND_DISPLAY XDG_RUNTIME_DIR; do \
	  [ -n "$$v" ] && systemctl --user import-environment $$v || true; \
	done
	systemctl --user daemon-reload
	systemctl --user enable --now $(SERVICE_UNIT)

service-remove: ## stop + remove the user systemd service (files stay; see uninstall)
	-systemctl --user disable --now $(SERVICE_UNIT) 2>/dev/null || true
	rm -f $(USER_UNIT_DIR)/$(SERVICE_UNIT)
	systemctl --user daemon-reload

clean: ## remove build artifacts
	rm -rf bin dist frontend/wailsjs
