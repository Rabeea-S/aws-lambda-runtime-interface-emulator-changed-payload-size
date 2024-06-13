# RELEASE_BUILD_LINKER_FLAGS disables DWARF and symbol table generation to reduce binary size
RELEASE_BUILD_LINKER_FLAGS=-s -w

BINARY_NAME=aws-lambda-rie

# Define all architectures and their corresponding destinations
ARCHITECTURES := amd64 arm64 386
DESTINATIONS := $(addprefix bin/$(BINARY_NAME)-,$(ARCHITECTURES))

# Default target
.PHONY: all
all: $(DESTINATIONS)

# Build targets for each architecture
bin/$(BINARY_NAME)-amd64: ARCH := amd64
bin/$(BINARY_NAME)-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -ldflags "${RELEASE_BUILD_LINKER_FLAGS}" -o $@ ./cmd/aws-lambda-rie

bin/$(BINARY_NAME)-arm64: ARCH := arm64
bin/$(BINARY_NAME)-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -ldflags "${RELEASE_BUILD_LINKER_FLAGS}" -o $@ ./cmd/aws-lambda-rie

bin/$(BINARY_NAME)-386: ARCH := 386
bin/$(BINARY_NAME)-386:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -ldflags "${RELEASE_BUILD_LINKER_FLAGS}" -o $@ ./cmd/aws-lambda-rie

# Dockerized build target
.PHONY: compile-with-docker
compile-with-docker:
	docker run --env GOPROXY=direct -v $(shell pwd):/LambdaRuntimeLocal -w /LambdaRuntimeLocal golang:1.19 make ARCH=amd64 compile-lambda-linux-amd64
	docker run --env GOPROXY=direct -v $(shell pwd):/LambdaRuntimeLocal -w /LambdaRuntimeLocal golang:1.19 make ARCH=arm64 compile-lambda-linux-arm64
	docker run --env GOPROXY=direct -v $(shell pwd):/LambdaRuntimeLocal -w /LambdaRuntimeLocal golang:1.19 make ARCH=386 compile-lambda-linux-386

# Test targets
.PHONY: tests
tests:
	go test ./...

.PHONY: integ-tests
integ-tests:
	python3 -m venv .venv
	.venv/bin/pip install --upgrade pip
	.venv/bin/pip install requests parameterized
	.venv/bin/python3 test/integration/local_lambda/test_end_to_end.py

# Integration tests and compile
.PHONY: integ-tests-and-compile
integ-tests-and-compile: tests all integ-tests

# Integration tests with Docker
.PHONY: integ-tests-with-docker
integ-tests-with-docker: tests compile-with-docker integ-tests

# Clean target
.PHONY: clean
clean:
	rm -f $(DESTINATIONS)
