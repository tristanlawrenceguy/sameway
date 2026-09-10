# Sameway build targets. Windows users without make: see the commands inside each target.

BIN := sameway

.PHONY: build check test lint tokens golden a11y run clean

build:
	go build -o bin/$(BIN) ./cmd/sameway

check: lint test

lint:
	gofmt -l . | tee /dev/stderr | test -z "$$(cat)"
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
