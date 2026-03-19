GO ?= go
GOFMT ?= $(GO)fmt
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
GOOPTS ?=
GOOS ?= $(shell $(GO) env GOHOSTOS)
GOARCH ?= $(shell $(GO) env GOHOSTARCH)

IMAGE_NAME ?= ghcr.io/pando85/gearr
IMAGE_VERSION ?= latest

PROJECT_VERSION := 0.1.11

.DEFAULT: help
.PHONY: help
help:	## show this help menu.
	@echo "Usage: make [TARGET ...]"
	@echo ""
	@@egrep -h "#[#]" $(MAKEFILE_LIST) | sed -e 's/\\$$//' | awk 'BEGIN {FS = "[:=].*?#[#] "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: build-all
build-all: server worker
build-all:	## build all binaries

.PHONY: server
server: build-server
server:		## build server binary

.PHONY: worker
worker: build-worker
worker:		## build worker binary

.PHONY: build-%
build-%:
	@echo "Building dist/gearr-$*"
	@if [ "$*" = "server" ]; then \
		cd server/web/ui && \
			npm install && \
			npm run build || exit 1; \
		cd -; \
	fi
	@CGO_ENABLED=0 go build -o dist/gearr-$* $*/main.go

.PHONY: images
images: image-server image-worker
images:		## build container images

.PHONY: images
push-images: push-image-server push-image-worker
push-images:		## build and push container images

CACHE_TYPE ?= registry
CACHE_MODE ?= max

ifeq ($(CACHE_TYPE),gha)
CACHE_FROM_BUILD := --cache-from type=gha --cache-from type=registry,ref=$(IMAGE_NAME):latest-build
CACHE_FROM_BASE := --cache-from type=gha --cache-from type=registry,ref=$(IMAGE_NAME):latest-base
CACHE_FROM_SERVER := --cache-from type=gha --cache-from type=registry,ref=$(IMAGE_NAME):latest-server
CACHE_FROM_WORKER := --cache-from type=gha --cache-from type=registry,ref=$(IMAGE_NAME):latest-worker
CACHE_TO_BUILD := --cache-to type=gha,mode=$(CACHE_MODE)
CACHE_TO_BASE := --cache-to type=gha,mode=$(CACHE_MODE)
CACHE_TO_SERVER := --cache-to type=gha,mode=$(CACHE_MODE)
CACHE_TO_WORKER := --cache-to type=gha,mode=$(CACHE_MODE)
else
CACHE_FROM_BUILD := --cache-from type=registry,ref=$(IMAGE_NAME):latest-build
CACHE_FROM_BASE := --cache-from type=registry,ref=$(IMAGE_NAME):latest-base
CACHE_FROM_SERVER := --cache-from type=registry,ref=$(IMAGE_NAME):latest-server
CACHE_FROM_WORKER := --cache-from type=registry,ref=$(IMAGE_NAME):latest-worker
CACHE_TO_BUILD := 
CACHE_TO_BASE := 
CACHE_TO_SERVER := 
CACHE_TO_WORKER := 
CACHE_TO_PUSH_BUILD := --cache-to type=registry,ref=$(IMAGE_NAME):latest-build,mode=$(CACHE_MODE)
CACHE_TO_PUSH_BASE := --cache-to type=registry,ref=$(IMAGE_NAME):latest-base,mode=$(CACHE_MODE)
CACHE_TO_PUSH_SERVER := --cache-to type=registry,ref=$(IMAGE_NAME):latest-server,mode=$(CACHE_MODE)
CACHE_TO_PUSH_WORKER := --cache-to type=registry,ref=$(IMAGE_NAME):latest-worker,mode=$(CACHE_MODE)
endif

.PHONY: image-%
.PHONY: push-image-%
image-% push-image-%: build-%
	@IS_PUSH="$(findstring push,$@)"; \
	if [ -n "$${IS_PUSH}" ]; then PUSH_OR_LOAD="--push"; else PUSH_OR_LOAD="--load"; fi; \
	if [ "$(CACHE_TYPE)" = "gha" ]; then \
		CACHE_TO_BUILD_VAL="$(CACHE_TO_BUILD)"; \
		CACHE_TO_BASE_VAL="$(CACHE_TO_BASE)"; \
		CACHE_TO_SERVER_VAL="$(CACHE_TO_SERVER)"; \
		CACHE_TO_WORKER_VAL="$(CACHE_TO_WORKER)"; \
	else \
		CACHE_TO_BUILD_VAL="$${IS_PUSH:+$(CACHE_TO_PUSH_BUILD)}"; \
		CACHE_TO_BASE_VAL="$${IS_PUSH:+$(CACHE_TO_PUSH_BASE)}"; \
		CACHE_TO_SERVER_VAL="$${IS_PUSH:+$(CACHE_TO_PUSH_SERVER)}"; \
		CACHE_TO_WORKER_VAL="$${IS_PUSH:+$(CACHE_TO_PUSH_WORKER)}"; \
	fi; \
	if [ "$*" = "server" ]; then \
		echo "Building build stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $${CACHE_TO_BUILD_VAL} \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-build \
		--target build \
		-f Dockerfile \
		. ; \
		echo "Building base stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $(CACHE_FROM_BASE) $${CACHE_TO_BASE_VAL} \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-base \
		--target base \
		-f Dockerfile \
		. ; \
		echo "Building server stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $(CACHE_FROM_BASE) $(CACHE_FROM_SERVER) $${CACHE_TO_SERVER_VAL} \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-$* \
		-f Dockerfile \
		--target $* \
		. ; \
	else \
		echo "Building build stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $${CACHE_TO_BUILD_VAL} \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-build \
		--target build \
		-f Dockerfile \
		. ; \
		echo "Building worker-pgs stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $(CACHE_FROM_BASE) \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-worker-pgs \
		--target worker-pgs \
		-f Dockerfile \
		. ; \
		echo "Building worker stage with cache..."; \
		docker buildx build \
		$(CACHE_FROM_BUILD) $(CACHE_FROM_BASE) $(CACHE_FROM_WORKER) $${CACHE_TO_WORKER_VAL} \
		$${PUSH_OR_LOAD} \
		-t $(IMAGE_NAME):$(IMAGE_VERSION)-$* \
		-f Dockerfile \
		--target $* \
		. ; \
	fi;

.PHONY: pull-cache
pull-cache:		## pull cache images from registry
	@docker pull $(IMAGE_NAME):latest-build 2>/dev/null || true
	@docker pull $(IMAGE_NAME):latest-base 2>/dev/null || true
	@docker pull $(IMAGE_NAME):latest-server 2>/dev/null || true
	@docker pull $(IMAGE_NAME):latest-worker 2>/dev/null || true

.PHONY: run-all
run-all: pull-cache images
run-all: export NOT_RUN_FRONT=true
run-all:	## run all services in local using docker-compose
run-all:
	@scripts/run-all.sh

.PHONY: down
down:		## stop all containers from docker-compose
down:
	@docker compose down --volumes

.PHONY: logs
logs:	## show logs
logs:
	@docker compose logs -f

.PHONY: demo-files
demo-files:		## download demo file
demo-files:
	@scripts/get-demo-files.sh

.PHONY: test
test:	## run unit tests with race detection and coverage
	go test -race -cover -short ./helper/... ./worker/... ./cmd/... ./model/... ./internal/... ./server/queue/... ./server/repository/...

.PHONY: test-integration
test-integration:	## run integration tests (requires PostgreSQL)
	go test -race -cover ./server/repository/... -run Integration

.PHONY: test-e2e
test-e2e:	## run e2e test (requires docker-compose)
test-e2e: demo-files run-all
	@scripts/test-e2e.sh

.PHONY: test-all
test-all: test test-e2e	## run all tests

.PHONY: update-changelog
update-changelog:	## automatically update changelog based on commits
	git cliff -t v$(PROJECT_VERSION) -u -p CHANGELOG.md

.PHONY: tag
tag:	## create a tag using version from Cargo.toml
	git tag -s v$(PROJECT_VERSION)  -m "v$(PROJECT_VERSION)" && \
	git push origin v$(PROJECT_VERSION)
