IMAGE 		?= aws-health-exporter
VERSION 	= $(shell git describe --always --tags --dirty)

all: format build test

test:
	@echo ">> running tests"
	@go test -v ./...

format:
	@echo ">> formatting code"
	@go fmt ./...

build:
	@echo ">> building binaries"
	@go build

docker:
	@echo ">> building docker image"
	@docker build \
		--build-arg SOURCE_COMMIT="$(VERSION)" \
		-t $(IMAGE):$(VERSION) \
		.
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

version:
	echo $(DOCKER_IMAGE_TAG)

.PHONY: all format build test docker
