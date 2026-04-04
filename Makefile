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

# Requires docker compose Postgres (matches config/auther.yaml and config/skeeper.yaml ports).
.PHONY: test-integration
test-integration:
	docker compose up -d auther-db skeeper-db
	AUTHER_TEST_DSN="postgres://auther_user:auther_password@127.0.0.1:5431/auther_db?sslmode=disable" \
	SKEEPER_TEST_DSN="postgres://skeeper_user:skeeper_password@127.0.0.1:5432/skeeper_db?sslmode=disable" \
		$(GO) test -tags=integration ./internal/auther/repository/postgres/... ./internal/skeeper/repository/postgres/...
	docker compose down

.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || \
		( echo "golangci-lint not found; running go vet only. Install: https://golangci-lint.run/usage/install/" ; $(GO) vet ./... )

.PHONY: check
check: fmt vet test

.PHONY: build
build: bin/auther bin/skeeper bin/skeepercli

.PHONY: bin/auther
bin/auther:
	@mkdir -p bin
	$(GO) build -o $@ ./cmd/auther

.PHONY: bin/skeeper
bin/skeeper:
	@mkdir -p bin
	$(GO) build -o $@ ./cmd/skeeper

.PHONY: bin/skeepercli
bin/skeepercli:
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

.PHONY: gen-keys
gen-keys:
	openssl genrsa -out config/keys/jwt_private.pem 2048
	openssl rsa -in config/keys/jwt_private.pem -pubout -out config/keys/jwt_public.pem

.PHONY: run_auther
run_auther:
	go run cmd/auther/main.go -config config/auther.yaml
	
.PHONY: run_skeeper
run_skeeper:
	go run cmd/skeeper/main.go -config config/skeeper.yaml

.PHONY: goose_down
goose_down:
	goose -dir ./migrations/auther postgres "postgres://auther_user:auther_password@127.0.0.1:5431/auther_db?sslmode=disable" status