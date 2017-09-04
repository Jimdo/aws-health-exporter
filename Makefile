pkgs   = $(shell go list ./... | grep -v /vendor/)

DOCKER_IMAGE_NAME       ?= aws-health-exporter
DOCKER_IMAGE_TAG        ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

all: format build test

test:
	@echo ">> running tests"
	@go test $(pkgs)

format:
	@echo ">> formatting code"
	@go fmt $(pkgs)

build: 
	@echo ">> building binaries"
	@go build

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

.PHONY: all format build test docker 
