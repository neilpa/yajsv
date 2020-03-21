.DEFAULT_GOAL := all

.PHONY: all
all: test build

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
	go build -v -mod vendor ./...

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
ci: test build
