.PHONY: all test build-binary clean

GO ?= go
GO_CMD := CGO_ENABLED=0 $(GO)
GIT_VERSION := $(shell git describe --tags --dirty)
VERSION := $(GIT_VERSION:v%=%)
GIT_COMMIT := $(shell git rev-parse HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_REPO ?= ghcr.io/xperimental/flowercare-exporter
DOCKER_TAG ?= dev

all: test build-binary

test:
	$(GO_CMD) test -cover ./...

build-binary:
	$(GO_CMD) build -tags netgo -ldflags "-w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(DATE)" -o flowercare-exporter .

.PHONY: image
image:
	docker buildx build -t "$(DOCKER_REPO):$(DOCKER_TAG)" --load .

.PHONY: all-images
all-images:
	docker buildx build -t "$(DOCKER_REPO):$(DOCKER_TAG)" --platform linux/amd64,linux/arm64 --push .

clean:
	rm -f flowercare-exporter
