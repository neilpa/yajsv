ARCH := darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64
VERSION := $(shell git describe --always --dirty)

release:
	gox -output 'build/{{.Dir}}.{{.OS}}.{{.Arch}}' -osarch "${ARCH}" -ldflags "-X main.version=${VERSION}"

clean:
	rm -rf build
