# Common tasks. Run `make help` to list them.
BIN := bin/kongkow
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/sgnd/kongkow/internal/cli.Version=$(VERSION)

STATICCHECK := v0.8.2
GOVULNCHECK := v1.9.0

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
export PLATFORMS

.PHONY: help build install test lint security bins dist clean

help: ## Show this help
	@grep -E '^[a-z0-9-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-15s %s\n", $$1, $$2}'

build: ## Build ./bin/kongkow
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kongkow

install: ## Install kongkow into $$GOPATH/bin
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./cmd/kongkow

test: ## Unit tests
	CGO_ENABLED=0 go test ./...

lint: ## gofmt + go vet + staticcheck
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK) ./...

security: ## Known vulnerabilities (govulncheck)
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK) ./...

GOBUILD = CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"
OUT ?= bins

bins: ## Build release binaries into OUT/kongkow_OS_ARCH/
	@for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; name=kongkow_$${os}_$${arch}; \
	  exe=kongkow; [ $$os = windows ] && exe=kongkow.exe; \
	  echo "  $$name"; mkdir -p $(OUT)/$$name && \
	  GOOS=$$os GOARCH=$$arch $(GOBUILD) -o $(OUT)/$$name/$$exe ./cmd/kongkow || exit 1; \
	done

dist: bins ## Release archives and checksums.txt into dist/
	@mkdir -p dist
	@for d in $(OUT)/kongkow_*; do \
	  name=$$(basename $$d); \
	  case $$name in \
	    *windows*) (cd $$d && zip -q $(CURDIR)/dist/$$name.zip kongkow.exe) ;; \
	    *) tar -C $$d -czf dist/$$name.tar.gz kongkow ;; \
	  esac || exit 1; \
	done
	@cd dist && (command -v sha256sum >/dev/null && sha256sum *.tar.gz *.zip || shasum -a 256 *.tar.gz *.zip) >checksums.txt
	@ls dist

clean: ## Remove build output
	rm -rf bin bins dist
