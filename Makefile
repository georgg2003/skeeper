GO ?= go
MODULE := github.com/georgg2003/skeeper
GO_DIRS := $(shell $(GO) list -f '{{.Dir}}' ./... | sort -u)

export PATH := $(PATH):$(shell go env GOPATH)/bin

.PHONY: generate
generate:
	$(GO) generate ./...

.PHONY: proto
proto:
	protoc --proto_path=api \
		--go_out=. --go_opt=module=$(MODULE) --go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		auther.proto skeeper.proto

.PHONY: imports
imports:
	@$(GO) tool goimports -local $(MODULE) -w $(GO_DIRS)

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-race
test-race:
	$(GO) test -race ./...

.PHONY: lint
lint:
	@$(GO) tool golangci-lint run ./...

.PHONY: check
check: imports lint test

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

goose_up:
	goose -dir ./migrations/auther postgres "postgres://auther_user:auther_password@127.0.0.1:5431/auther_db?sslmode=disable" up
	goose -dir ./migrations/skeeper postgres "postgres://skeeper_user:skeeper_password@127.0.0.1:5432/skeeper_db?sslmode=disable" up
