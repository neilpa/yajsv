ARCH := darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64
VERSION := $(shell git describe --always --dirty)
LDFLAGS := -ldflags "-X main.version=${VERSION}"
BUILD_DIR := build

.PHONY: build
build: *.go
	go build ${LDFLAGS}

.PHONY: release
release: *.go
	gox -output '${BUILD_DIR}/{{.Dir}}.{{.OS}}.{{.Arch}}' -osarch "${ARCH}" ${LDFLAGS}

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR} yajsv yajsv.exe coverage.out

.PHONY: fmt
fmt:
	go fmt 	./...

.PHONY: tidy
tidy:
	go mod tidy -v

.PHONY: test
test:
	go test -coverprofile=coverage.out ./...

.PHONY: ci
ci: clean test build