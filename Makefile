SHELL := bash
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(patsubst %/,%,$(dir $(mkfile_path)))
PATH := $(mkfile_dir)/bin:$(PATH)
.SHELLFLAGS := -eu -o pipefail -c # -c: Needed in .SHELLFLAGS. Default is -c.
.DEFAULT_GOAL := build

# dotenv := $(PWD)/.env
# -include $(dotenv)

# app := app
# stage := dev

export

dst := /tmp/api

depends_cmds := go golangci-lint #goreleaser statik
check:
	@for cmd in ${depends_cmds}; do command -v $$cmd >&/dev/null || (echo "No $$cmd command" && exit 1); done
	@echo "[OK] check ok!"

all: clean tidy fmt lint build test
clean:
	@echo "==> Cleaning" >&2
	@rm -f $(dst)
	@go clean -cache -testcache
deps:
	@go list -m all
update:
	@go get -u ./...
tidy:
	@echo "==> Running go mod tidy -v"
	@go mod tidy -v
tidy-go:
	@v=$(shell go version|awk '{print $$3}' |sed -e 's,go\(.*\)\..*,\1,g') && go mod tidy -go=$${v}
fmt:
	@echo "==> Running go fmt ./..." >&2
	@go fmt ./...
lint:
	@echo "==> Running golangci-lint run" >&2
	@golangci-lint run
# build:
# 	@echo "==> Go Building" >&2
# 	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o $(dst) cmd/main.go

build: build-linux
build-linux:
	@make GOOS=linux GOARCH=amd64 _build
build-mac:
	@make GOOS=darwin GOARCH=arm64 _build
build-windows:
	@make GOOS=windows GOARCH=amd64 _build
build-android:
	@make GOOS=android GOARCH=arm64 _build
_build: clean tidy fmt lint
	@env CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -v -o $(dst) ./main.go

run: check clean
	@cat<./test/urls.txt | grep -v '^#' | LOG_LEVEL=debug X_COOKIE_JSON=./cookies.json go run .

pkg := ./...
cover_mode := atomic
cover_out := cover.out
.PHONY: test
test: testsum-cover-check
# test-normal:
# 	@echo "==> Testing $(pkg)" >&2
# 	@go test -v $(pkg)
# test-cover:
# 	@echo "==> Running go test with coverage check" >&2
# 	@go test $(pkg) -coverprofile=$(cover_out) -covermode=$(cover_mode) -coverpkg=$(pkg)
# test-cover-count:
# 	@echo "==> Running go test with coverage check (count mode)" >&2
# 	@make test-cover cover_mode=count
# 	@go tool cover -func=$(cover_out)
# test-cover-html: test-cover
# 	@go tool cover -html=$(cover_out) -o cover.html
# test-cover-open: test-cover
# 	@go tool cover -html=$(cover_out)
# test-cover-check: test-cover-html
# 	@echo "==> Checking coverage threshold" >&2
# 	@go-test-coverage --config=./.testcoverage.yml
testsum:
	@echo "==> Running go testsum" >&2
	@gotestsum --format testname -- -v $(pkg) -coverprofile=$(cover_out) -covermode=$(cover_mode) -coverpkg=$(pkg)
testsum-cover-check: testsum
	@echo "==> Running test-coverage" >&2
	@go-test-coverage --config=./.testcoverage.yaml


# gen: mockery
# mockery:
# 	@echo "==> Running mockery" >&2
# 	@mockery

tag:
	@v=$$(git tag --list |sort -V |tail -1) && nv="$${v%.*}.$$(($${v##*.}+1))" && echo "==> New tag: $${nv}" && git tag $${nv}
tagp: tag
	@git push --tags


gr_init:
	@goreleaser init
gr_check:
	@goreleaser check
gr_snap:
	@goreleaser release --snapshot --clean $(OPT)
gr_snap_skip_publish:
	@OPT=--skip-publish make gr_snap
gr_build:
	@goreleaser build --snapshot --clean
