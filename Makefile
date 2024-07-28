BIN := "./bin/image_previewer"

build:
	go build -v -o $(BIN) ./cmd

run:
	docker compose -f deploy/docker-compose.yaml up -d

down:
	docker compose -f deploy/docker-compose.yaml down

test:
	go test -race -count 10 ./internal/...

integration-tests:
	docker compose -f $(shell go env GOPATH)/deploy/docker-compose-test.yaml up --build --force-recreate --abort-on-container-exit --exit-code-from tests && \
	docker compose -f $(shell go env GOPATH)/deploy/docker-compose-test.yaml down

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.59.1

lint: install-lint-deps
	golangci-lint run ./...

.PHONY: build run test lint