ROOT_DIR  := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BINARY    := kubectl-meta
MODULE    := github.com/nickytd/kubectl-meta-plugin
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

TOOLS_DIR := $(ROOT_DIR)/tools
TOOLS_MOD := $(TOOLS_DIR)/go.mod
GO_TOOL   := go tool -modfile=$(TOOLS_MOD)

GCI_OPT ?= -s standard -s default -s "prefix($(shell go list -m))" --skip-generated

.PHONY: fmt gci lint license license-check check tidy build clean govulncheck

.DEFAULT_GOAL := build

fmt:
	gofmt -s -w $(ROOT_DIR)
	$(GO_TOOL) goimports -w $(ROOT_DIR)

gci:
	$(GO_TOOL) gci write $(GCI_OPT) $(ROOT_DIR)/cmd $(ROOT_DIR)/pkg

lint:
	$(GO_TOOL) golangci-lint run $(ROOT_DIR)/...

license:
	$(GO_TOOL) addlicense -s=only -c "nickytd" -l apache $(ROOT_DIR)/cmd $(ROOT_DIR)/pkg

license-check:
	$(GO_TOOL) addlicense -s=only -c "nickytd" -l apache -check $(ROOT_DIR)/cmd $(ROOT_DIR)/pkg

govulncheck:
	$(GO_TOOL) govulncheck $(ROOT_DIR)/...

check: fmt gci lint license
	go fix $(ROOT_DIR)/...

tidy:
	go mod tidy
	go -C $(TOOLS_DIR) mod tidy

build: check
	CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(ROOT_DIR)/bin/$(BINARY) \
		$(ROOT_DIR)/cmd/main.go

clean:
	rm -rf $(ROOT_DIR)/bin/ $(ROOT_DIR)/dist/