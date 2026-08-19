APP        := mhtodo
VERSION    ?= 0.1.0
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BIN        := bin/$(APP)
PREFIX     ?= $(HOME)/.local

# This distro ships webkit2gtk-4.1 only → Wails v2 needs this tag (M0 finding).
# GUI builds also need a *mode* tag: wails build/dev inject production/dev automatically;
# plain `go build` must add it explicitly — hence "$(TAGS) production" below.
# Override on other systems: make build TAGS=   (webkit2gtk-4.0) or TAGS="webkit2_41".
TAGS       ?= webkit2_41

LDFLAGS    := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"
LINTER     := $(shell command -v golangci-lint 2>/dev/null)
GOFORMT    := $(shell command -v gofmt || echo "$$(go env GOROOT)/bin/gofmt")

.PHONY: help all build test lint fmt tidy path clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

all: build

build: ## local release-mode build → bin/mhtodo (M3 switches to wails build once the Vite frontend lands)
	go build -tags "$(TAGS) production" $(LDFLAGS) -o $(BIN) .

test: ## go tests (core, store, cli golden)
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
	@go run -tags "$(TAGS)" . path 2>/dev/null || echo "$${XDG_DATA_HOME:-$$HOME/.local/share}/$(APP)/$(APP).db"

clean: ## remove build artifacts
	rm -rf bin dist frontend/wailsjs
