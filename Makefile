include .env
export

build_dir = ./cmd/gophermart
dev_log_level=debug

##dev: build app in dev mode
.PHONY: dev
dev:
	GOARCH=arm64 GOOS=darwin go build -o ${build_dir}/gophermart-darwin ${build_dir}
	GOARCH=arm64 DEBUG=true go run ${build_dir} -d ${DATABASE_URI}

.PHONY: help
help:
	@echo "Commands list:"
	@sed -n "s/^##//p" $(MAKEFILE_LIST) | column -t -s ":" | sed -e "s/^/ /"
