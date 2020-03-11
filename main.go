// yajsv is a command line tool for validating JSON documents against
// a provided JSON Schema - https://json-schema.org/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/xeipuuv/gojsonschema"
)

const (
	version = "v1.1.0"
)

var (
	schemaFlag  = flag.String("s", "", "primary JSON schema to validate against, required")
	quietFlag   = flag.Bool("q", false, "quiet, only print validation failures and errors")
	versionFlag = flag.Bool("v", false, "print version and exit")

	listFlags stringFlags
	refFlags  stringFlags
)

func init() {
	flag.Var(&listFlags, "l", "validate JSON documents from newline separated paths and/or globs in a text file (relative to the basename of the file itself)")
	flag.Var(&refFlags, "r", "referenced schema(s), can be globs and/or used multiple times")
	flag.Usage = printUsage
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	if *schemaFlag == "" {
		usageError("missing required -s schema argument")
	}

	// Resolve document paths to validate
	docs := make([]string, 0)
	for _, arg := range flag.Args() {
		docs = append(docs, glob(arg)...)
	}
	for _, list := range listFlags {
		dir := filepath.Dir(list)
		f, err := os.Open(list)
		if err != nil {
			log.Fatalf("%s: %s\n", list, err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			// Calclate the glob relative to the directory of the file list
			pattern := strings.TrimSpace(scanner.Text())
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(dir, pattern)
			}
			docs = append(docs, glob(pattern)...)
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("%s: invalid file list: %s\n", list, err)
		}
	}
	if len(docs) == 0 {
		usageError("no JSON documents to validate")
	}

	// Compile target schema
	sl := gojsonschema.NewSchemaLoader()
	schemaUri := fileUri(*schemaFlag)
	for _, ref := range refFlags {
		for _, p := range glob(ref) {
			uri := fileUri(p)
			if uri == schemaUri {
				continue
			}
			loader := gojsonschema.NewReferenceLoader(uri)
			err := sl.AddSchemas(loader)
			if err != nil {
				log.Fatalf("%s: invalid schema: %s\n", p, err)
			}
		}
	}
	schemaLoader := gojsonschema.NewReferenceLoader(schemaUri)
	schema, err := sl.Compile(schemaLoader)
	if err != nil {
		log.Fatalf("%s: invalid schema: %s\n", *schemaFlag, err)
	}

	// Validate the schema against each doc in parallel, limiting simultaneous
	// open files to avoid ulimit issues.
	var wg sync.WaitGroup
	sem := make(chan int, runtime.GOMAXPROCS(0)+10)
	failures := make([]string, 0)
	errors := make([]string, 0)
	for _, p := range docs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- 0
			defer func() { <-sem }()

			loader := gojsonschema.NewReferenceLoader(fileUri(path))
			result, err := schema.Validate(loader)
			switch {
			case err != nil:
				msg := fmt.Sprintf("%s: error: %s", path, err)
				fmt.Println(msg)
				errors = append(errors, msg)

			case !result.Valid():
				lines := make([]string, len(result.Errors()))
				for i, desc := range result.Errors() {
					lines[i] = fmt.Sprintf("%s: fail: %s", path, desc)
				}
				msg := strings.Join(lines, "\n")
				fmt.Println(msg)
				failures = append(failures, msg)

			case !*quietFlag:
				fmt.Printf("%s: pass\n", path)
			}
		}(p)
	}
	wg.Wait()

	// Summarize results (e.g. errors)
	if !*quietFlag {
		if len(failures) > 0 {
			fmt.Printf("%d of %d failed validation\n", len(failures), len(docs))
			fmt.Println(strings.Join(failures, "\n"))
		}
		if len(errors) > 0 {
			fmt.Printf("%d of %d malformed documents\n", len(errors), len(docs))
			fmt.Println(strings.Join(errors, "\n"))
		}
	}
	if len(failures) > 0 || len(errors) > 0 {
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -s schema.json [options] document.json ...

  yajsv validates JSON document(s) against a schema. One of three statuses are
  reported per document:

    pass: Document is valid relative to the schema
    fail: Document is invalid relative to the schema
    error: Document is malformed, e.g. not valid JSON

  The 'fail' status may be reported multiple times per-document, once for each
  schema validation failure.

Options:

`, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
}

func usageError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	printUsage()
	os.Exit(2)
}

func fileUri(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("%s: %s", path, err)
	}

	uri := "file://"

	if runtime.GOOS == "windows" {
		// This is not formally correct for all corner cases in windows
		// file handling but should work for all standard cases. See:
		// https://docs.microsoft.com/en-us/archive/blogs/ie/file-uris-in-windows
		uri = uri + "/" + strings.ReplaceAll(
			strings.ReplaceAll(abs, "\\", "/"),
			" ", "%20",
		)
	} else {
		uri = uri + abs
	}

	return uri
}

// glob is a wrapper that also resolves `~` since we may be skipping
// the shell expansion when single-quoting globs at the command line
func glob(pattern string) []string {
	pattern, err := homedir.Expand(pattern)
	if err != nil {
		log.Fatal(err)
	}
	paths, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatal(err)
	}
	if len(paths) == 0 {
		log.Fatalf("%s: no such file or directory", pattern)
	}
	return paths
}

type stringFlags []string

func (sf *stringFlags) String() string {
	return "multi-string"
}

func (sf *stringFlags) Set(value string) error {
	*sf = append(*sf, value)
	return nil
}
