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

LDFLAGS    := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"
GOFLAGS    := -tags $(TAGS)
LINTER     := $(shell command -v golangci-lint 2>/dev/null)
GOFORMT    := $(shell command -v gofmt || echo "$$(go env GOROOT)/bin/gofmt")

.PHONY: help all dev build test lint fmt tidy fe-install fe-bindings fe-build fe-dev path clean

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
# install/build; LDFLAGS is passed raw because it already carries -ldflags.
# wails build writes to build/bin/<name>; copy it out to bin/ for install/release.
build: ## release-mode local build → bin/mhtodo (wails build runs the frontend too)
	wails build $(GOFLAGS) -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o $(APP)
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

clean: ## remove build artifacts
	rm -rf bin dist frontend/wailsjs
