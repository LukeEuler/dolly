all: test

MODULE = "github.com/LukeEuler/dolly"

CURRENT_DIR=$(shell pwd)

GOIMPORTS := $(shell command -v goimports 2> /dev/null)
CILINT := $(shell command -v golangci-lint 2> /dev/null)

style:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	! find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -d -local ${MODULE} | grep '^'

format:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -l -local ${MODULE} | xargs goimports -l -local ${MODULE} -w

cilint:
ifndef CILINT
	$(error "golangci-lint is not available please install golangci-lint")
endif
	golangci-lint run --timeout 5m0s

test: style cilint
	go test -cover ./...

.PHONY: all style format cilint test
