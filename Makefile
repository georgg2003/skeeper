GO ?= go

.PHONY: generate
generate:
	$(GO) generate ./...

.PHONY: fmt
fmt:
	@$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-race
test-race:
	$(GO) test -race ./...

.PHONY: test-integration
test-integration:
	$(GO) test -tags=integration ./internal/auther/repository/postgres/... ./internal/skeeper/repository/postgres/...

.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || \
		( echo "golangci-lint not found; running go vet only. Install: https://golangci-lint.run/usage/install/" ; $(GO) vet ./... )

.PHONY: check
check: fmt vet test

.PHONY: build
build: bin/auther bin/skeeper bin/gophkeeper

bin/auther:
	@mkdir -p bin
	$(GO) build -o $@ ./cmd/auther

bin/skeeper:
	@mkdir -p bin
	$(GO) build -o $@ ./cmd/skeeper

bin/gophkeeper:
	@mkdir -p bin
	$(GO) build -ldflags "-X main.version=dev -X main.buildTime=unknown" -o $@ ./cmd/client

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: vendor
vendor:
	$(GO) mod vendor

.PHONY: tidyvendor
tidyvendor: tidy vendor

.PHONY: clean
clean:
	rm -rf bin/
