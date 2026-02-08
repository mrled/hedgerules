# Hedgerules

MAKEFLAGS += --warn-undefined-variables
.SHELLFLAGS := -euc
.SUFFIXES:

GO_SRC := $(shell find hedgerules -name '*.go' -o -name '*.js') hedgerules/go.mod hedgerules/go.sum
HUGO_SRC := $(shell find www/config www/content themes -type f 2>/dev/null)

# Show a nice table of Make targets.
.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Go ---

.PHONY: gofmt
gofmt: ## Run gofmt on all Go source files.
	cd hedgerules && gofmt -w .

.PHONY: govet
govet: ## Run go vet on all Go source files.
	cd hedgerules && go vet ./...

bin/hedgerules: $(GO_SRC) | gofmt govet ## Build the hedgerules binary.
	mkdir -p $(dir $@)
	cd hedgerules && go build -o $(abspath $@) ./cmd/hedgerules

# --- Hugo ---

www/public/production/.sentinel: $(HUGO_SRC) ## Build the Hugo site for production.
	cd www && hugo --environment production
	@touch $@

# --- Clean ---

clean: ## Clean up all build artifacts.
	rm -rf hedgerules/bin www/public/production
