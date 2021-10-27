.PHONY: all test build-binary clean

GO ?= go
GO_CMD := CGO_ENABLED=0 $(GO)
GIT_VERSION := $(shell git describe --tags --dirty)
VERSION := $(GIT_VERSION:v%=%)
GIT_COMMIT := $(shell git rev-parse HEAD)

all: test build-binary

test:
	$(GO_CMD) test -cover ./...

build-binary:
	$(GO_CMD) build -tags netgo -ldflags "-w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT)" -o flowercare-exporter .

clean:
	rm -f flowercare-exporter
