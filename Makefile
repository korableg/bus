PACKAGE=$(shell go list -m)

.PHONY: lint
lint:
	@golangci-lint run --config=./golangci.yml

.PHONY: imports
imports:
	gci write --custom-order -s standard -s default -s "prefix(${PACKAGE})" ${GO_FILES}

.PHONY: test
test:
	docker compose up --wait -d \
	&& go clean -testcache \
	&& go test --tags=integration_test -v -race ./... \
	; docker compose down