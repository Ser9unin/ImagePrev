BIN := "./bin/image_previewer"

build:
	go build -v -o $(BIN) ./cmd

run:
	docker compose -f deploy/docker-compose.yaml up -d

down:
	docker compose -f deploy/docker-compose.yaml down

test:
	go test -race -count 100 ./internal/...

integration-tests:
	set -e ;\
	docker-compose -f deploy/docker-compose-test.yaml -p integration_test up --build -d ;\
	test_status_code=0 ;\
	docker-compose -f deploy/docker-compose-test.yaml run integration_test || test_status_code=$$? ;\
	docker-compose -f deploy/docker-compose-test.yaml down ;\
	exit $$test_status_code ;

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.59.1

lint: install-lint-deps
	golangci-lint run ./...

.PHONY: build run test lint