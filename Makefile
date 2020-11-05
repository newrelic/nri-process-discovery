INTEGRATION  := $(shell basename $(shell pwd))
BINARY_NAME   = $(INTEGRATION)
GO_PKGS      := $(shell go list ./...)

all: build

help: validate
	@go run cmd/$(BINARY_NAME).go --help

run:
	@go run cmd/$(BINARY_NAME).go

run-debug:
	@go run cmd/$(BINARY_NAME).go -debug

build: clean validate compile

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: removing binaries and coverage file..."
	@rm -rfv bin/*

validate:
	@printf "=== $(INTEGRATION) === [ validate ]: running gofmt... "
	@OUTPUT="$(shell gofmt -l $(GO_PKGS))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Incorrect syntax in the following files:" ;\
		echo "$$OUTPUT" ;\
		exit 1 ;\
	fi
	@printf "=== $(INTEGRATION) === [ validate ]: running golint... "
	@OUTPUT="$(shell golint $(GO_PKGS))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Issues found:" ;\
		echo "$$OUTPUT" ;\
		exit 1 ;\
	fi
	@printf "=== $(INTEGRATION) === [ validate ]: running go vet... "
	@OUTPUT="$(shell go vet $(GO_PKGS))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Issues found:" ;\
		echo "$$OUTPUT" ;\
		exit 1;\
	fi

compile-deps:
	@echo "=== $(INTEGRATION) === [ compile-deps ]: installing build dependencies..."
	@go mod vendor

compile-only:
	@echo "=== $(INTEGRATION) === [ compile ]: building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) cmd/$(BINARY_NAME).go

compile: compile-deps compile-only


.PHONY: all build clean validate compile-deps compile-only compile 
