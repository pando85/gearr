GO ?= go
GOFMT ?= $(GO)fmt
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
GOOPTS ?=
GOOS ?= $(shell $(GO) env GOHOSTOS)
GOARCH ?= $(shell $(GO) env GOHOSTARCH)

IMAGE_NAME ?= pando85/transcoder
IMAGE_VERSION ?= latest

.PHONY: build-all
build-all: server worker

.PHONY: server
server: build-server

.PHONY: worker
worker: build-worker

build-%:
	@echo "Building $*"
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) run build.go build $*

.PHONY: images
images: image-server image-worker

image-%:
	@docker buildx build \
		--load \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-$* \
		-f $*/Dockerfile \
		.

