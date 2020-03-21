ARCH := darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64
VERSION := $(shell git describe --always --dirty)
LDFLAGS := -ldflags "-X main.version=${VERSION}"

yajsv: *.go
	go build ${LDFLAGS}

release: *.go
	gox -output 'build/{{.Dir}}.{{.OS}}.{{.Arch}}' -osarch "${ARCH}" ${LDFLAGS}

clean:
	rm -rf build yajsv
