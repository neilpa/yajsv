CONFIG_PATH ?= main
VERSION := $(shell git describe --always --dirty)
BINARY_NAME ?= yajsv
BUILD_DIR := bin
GIT_REV_PARSE := $(shell git rev-parse HEAD)
COMMIT_ID := $(if ${GIT_REV_PARSE},${GIT_REV_PARSE},unknown)
DATECMD := date$(if $(findstring Windows,$(OS)),.exe,)
BUILD_TIMESTAMP := $(shell ${DATECMD} +%Y-%m-%dT%H:%m:%S%z)

.DEFAULT_GOAL := all

.PHONY: all
all: clean test build

.PHONY: build
build:
	@make --no-print-directory build-platform GOOS=windows GOARCH=amd64 CGO_ENABLED=0
	@make --no-print-directory build-platform GOOS=linux GOARCH=amd64 CGO_ENABLED=0
	@make --no-print-directory build-platform GOOS=darwin GOARCH=amd64 CGO_ENABLED=0

.PHONY: win
 win:
	@make --no-print-directory build-platform GOOS=windows GOARCH=amd64

.PHONY: build-platform
build-platform:
	@echo Building ${GOOS}-${GOARCH}
	$(eval BINARY := ${BINARY_NAME}$(if $(findstring windows,$(GOOS)),.exe,))
	go build -v -mod vendor -o ${BUILD_DIR}/${GOOS}-${GOARCH}/$(BINARY) \
		-ldflags=all="-X ${CONFIG_PATH}.Version=${VERSION} -X ${CONFIG_PATH}.CommitId=${COMMIT_ID} -X ${CONFIG_PATH}.BuildTimestamp=${BUILD_TIMESTAMP}" .

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

.PHONY: fmt
fmt:
	go fmt 	./...

.PHONY: tidy
tidy:
	go mod tidy -v

.PHONY: vendor
vendor: tidy
	go mod vendor -v

.PHONY: test
test:
	go test -v -mod=vendor -coverprofile=coverage.out ./...

.PHONY: ci
ci: clean test build
