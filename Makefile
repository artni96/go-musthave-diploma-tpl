include .env
export

build_dir = ./cmd/gophermart
dev_log_level=debug

.PHONY: help
help:
	@echo "Commands list:"
	@sed -n "s/^##//p" $(MAKEFILE_LIST) | column -t -s ":" | sed -e "s/^/ /"

##dev: build app in dev mode
.PHONY: dev
dev:
	GOARCH=arm64 GOOS=darwin go build -o ${build_dir}/gophermart-darwin ${build_dir}
	GOARCH=arm64 DEBUG=true go run ${build_dir} -d ${DATABASE_URI}

##accr: launch accrual system
.PHONY: accr
accr:
	GOARCH=arm64 ./cmd/accrual/accrual_darwin_arm64 -a ${ACCRUAL_SYSTEM_ADDRESS}

##test: run unit tests
.PHONY: test
test:
	GOARCH=arm64 go test ./...

##test-cover: run unit tests with coverage
.PHONY: test-cover
test-cover:
	GOARCH=arm64 go test ./... -cover

##up-test-db: initialize db for unit tests
.PHONY: up-test-db
up-test-db:
	docker compose -f docker-compose.test.yml up -d --build

##down-test-db: downsize db for unit tests
.PHONY: down-test-db
down-test-db:
	docker compose -f docker-compose.test.yml down