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

.PHONY: run-all
run-all: images
run-all:
	@docker-compose up -d postgres rabbitmq
	@ATTEMPT=1; \
	while [ $$ATTEMPT -le $(MAX_ATTEMPTS) ]; do \
		echo "Attempt $$ATTEMPT of $(MAX_ATTEMPTS)"; \
		if docker-compose exec postgres psql -U postgres -d transcoder -c "SELECT 1"; then \
			echo "Command succeeded."; \
			break; \
		fi; \
		ATTEMPT=$$((ATTEMPT + 1)); \
	done
	@docker-compose up -d

