.PHONY: all build lint test test-int docker clean

APP_BOT := ./bin/bot
APP_WEB := ./bin/web
GO      := go
LDFLAGS := -trimpath -ldflags="-s -w"

all: lint test build

## build: compile bot and web binaries
build:
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(APP_BOT) ./cmd/bot
	$(GO) build $(LDFLAGS) -o $(APP_WEB) ./cmd/web

## lint: gofmt + go vet + staticcheck + golangci-lint
lint:
	$(GO) fmt ./...
	$(GO) vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "[warn] staticcheck not installed"
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "[warn] golangci-lint not installed"

## test: unit tests with race detector
test:
	$(GO) test -race -count=1 ./...

## test-int: integration tests (requires TEST_DB_DSN)
test-int:
	$(GO) test -race -count=1 -tags=integration ./...

## docker: build images and validate compose
docker:
	docker compose build
	docker compose config --quiet

## up: start the full stack
up:
	docker compose up -d

## down: stop the full stack
down:
	docker compose down

## logs: tail all services
logs:
	docker compose logs -f

## clean: remove binaries
clean:
	rm -rf bin/

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
