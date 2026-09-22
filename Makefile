# Sameway build targets. Windows users without make: see the commands inside each target.

BIN := sameway

# The version this build says it is, which is what auto-update compares
# against the latest release. Only a checkout sitting exactly on a tag
# gets one: anything else is a build from source, and a build from source
# never replaces itself with a release. See internal/update.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
VERSION := dev
endif
LDFLAGS := -X github.com/tristanlawrenceguy/sameway/internal/update.Version=$(VERSION)

.PHONY: build check test lint tokens golden a11y run clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/sameway

check: lint test

lint:
	gofmt -l . > /dev/null 2>&1 || (gofmt -l . && exit 1)
	go vet ./...
	go run ./tools/check

test:
	go test ./...

tokens:
	go run ./tools/tokens

golden:
	UPDATE_GOLDEN=1 go test ./internal/render/...

a11y:
	cd tools/a11y-runner && npm install && npm test

# Live-page tests need a running server with llm.provider: none, e.g. `make run`.
pages:
	cd tools/a11y-runner && npm install && SAMEWAY_URL=http://127.0.0.1:8080 npm run test:pages

run: build
	./bin/$(BIN) --workspace examples/workspaces/starter serve

clean:
	rm -rf bin
