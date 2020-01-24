release:
	gox -output 'build/{{.Dir}}.{{.OS}}.{{.Arch}}' -osarch 'windows/amd64 windows/386 darwin/amd64 linux/386 linux/amd64'

clean:
	rm -rf build
