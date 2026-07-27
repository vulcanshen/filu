# filu — build / run / test / package
#
# 跑 `make`(或 `make help`)列出所有指令。
# unix-first(macOS + Linux),CGO_ENABLED=0 靜態編譯;真正的跨平台 release 走 goreleaser。

BINARY   := filu
PKG      := ./cmd/filu
DIST_DIR := dist

GOOS   := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
# 無 tag 時 fallback 到 short commit;-dirty 標未提交改動(供打包檔名用)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w

.DEFAULT_GOAL := help

##@ 編譯(build)

.PHONY: build
build: ## 編譯本地執行檔 → ./filu(CGO_ENABLED=0 靜態、-trimpath、strip)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)
	@echo "built ./$(BINARY)  ($(VERSION) $(GOOS)/$(GOARCH))"

.PHONY: build-all
build-all: ## 編譯整個 module(go build ./...)
	go build ./...

.PHONY: install
install: ## go install → $$GOBIN/PATH(裝了才能 eval "$$(filu shell)" 開 cd-on-quit)
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: uninstall
uninstall: ## 移除已安裝的 filu(從 $$GOBIN,否則 $$GOPATH/bin)
	@dir="$$(go env GOBIN)"; [ -n "$$dir" ] || dir="$$(go env GOPATH)/bin"; \
	if [ -f "$$dir/$(BINARY)" ]; then rm -f "$$dir/$(BINARY)" && echo "removed $$dir/$(BINARY)"; else echo "not installed ($$dir/$(BINARY))"; fi

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

##@ 執行(run)

.PHONY: run
run: ## 本地跑 filu TUI(go run)
	go run $(PKG)

##@ 測試 / 檢查(test)

.PHONY: test
test: ## 跑所有測試(go test ./...)
	go test ./...

.PHONY: vet
vet: ## go vet ./...
	go vet ./...

.PHONY: fmt
fmt: ## gofmt -w(就地格式化 cmd/ internal/)
	gofmt -w cmd internal

.PHONY: fmt-check
fmt-check: ## gofmt -l(列出未格式化的檔;有輸出即失敗,CI 用)
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "not gofmt'ed:"; echo "$$out"; exit 1; fi

.PHONY: check
check: fmt-check vet test ## fmt-check + vet + test 一次跑(commit 前)

##@ 打包(package)

.PHONY: package
package: build ## 打包本地平台執行檔 → dist/filu_<ver>_<os>_<arch>.tar.gz
	@mkdir -p $(DIST_DIR)
	tar -czf $(DIST_DIR)/$(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz $(BINARY)
	@echo "packaged $(DIST_DIR)/$(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz"

##@ 其他

.PHONY: clean
clean: ## 移除 ./filu 與 dist/
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

.PHONY: help
help: ## 顯示這份說明
	@awk 'BEGIN {FS = ":.*?## "} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
