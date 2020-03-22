// yajsv is a command line tool for validating JSON and YAML documents against
// a provided JSON Schema - https://json-schema.org/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
	"github.com/xeipuuv/gojsonschema"
)

var (
	version = "undefined"
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
	os.Exit(realMain(os.Args[1:]))
}

func realMain(args []string) int {
	flag.CommandLine.Parse(args)
	if *versionFlag {
		fmt.Println(version)
		return 0
	}
	if *schemaFlag == "" {
		return usageError("missing required -s schema argument")
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
		return usageError("no documents to validate")
	}

	// Compile target schema
	sl := gojsonschema.NewSchemaLoader()
	schemaPath, err := filepath.Abs(*schemaFlag)
	if err != nil {
		log.Fatalf("%s: unable to convert to absolute path: %s\n", *schemaFlag, err)
	}
	for _, ref := range refFlags {
		for _, p := range glob(ref) {
			absPath, absPathErr := filepath.Abs(p)
			if absPathErr != nil {
				log.Fatalf("%s: unable to convert to absolute path: %s\n", absPath, absPathErr)
			}

			if absPath == schemaPath {
				continue
			}

			loader, err := jsonLoader(absPath)
			if err != nil {
				log.Fatalf("%s: unable to load schema ref: %s\n", *schemaFlag, err)
			}

			if err := sl.AddSchemas(loader); err != nil {
				log.Fatalf("%s: invalid schema: %s\n", p, err)
			}
		}
	}

	schemaLoader, err := jsonLoader(schemaPath)
	if err != nil {
		log.Fatalf("%s: unable to load schema: %s\n", *schemaFlag, err)
	}
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
		//fmt.Println(p)
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- 0
			defer func() { <-sem }()


			loader, err := jsonLoader(path)
			if err != nil {
				msg := fmt.Sprintf("%s: error: load doc %s\n", path, err)
				fmt.Println(msg)
				errors = append(errors, msg)
				return
			}
			result, err := schema.Validate(loader)
			switch {
			case err != nil:
				msg := fmt.Sprintf("%s: error: validate: %s", path, err)
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
	exit := 0
	if len(failures) > 0 {
		exit |= 1
	}
	if len(errors) > 0 {
		exit |= 2
	}
	return exit
}

func jsonLoader(path string) (gojsonschema.JSONLoader, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		buf, err = yaml.YAMLToJSON(buf)
	}
	if err != nil {
		return nil, err
	}
	return gojsonschema.NewBytesLoader(buf), nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -s schema.(json|yml) [options] document.(json|yml) ...

  yajsv validates JSON and YAML document(s) against a schema. One of three statuses are
  reported per document:

    pass: Document is valid relative to the schema
    fail: Document is invalid relative to the schema
    error: Document is malformed, e.g. not valid JSON or YAML

  The 'fail' status may be reported multiple times per-document, once for each
  schema validation failure.

  Sets the exit code to 1 on any failures, 2 on any errors, 3 on both, 4 on
  invalid usage. Otherwise, 0 is returned if everything passes validation.

Options:

`, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
}

func usageError(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	printUsage()
	return 4
}

// glob is a wrapper that also resolves `~` since we may be skipping
// the shell expansion when single-quoting globs at the command line
func glob(pattern string) []string {
	pattern, err := homedir.Expand(pattern)
	if err != nil {
		log.Fatal(err)
	}
	universalPaths := make([]string, 0)
	paths, err := filepath.Glob(pattern)
	for _, mypath := range paths {
		universalPaths = append(universalPaths, filepath.ToSlash(mypath))
	}
	if err != nil {
		log.Fatal(err)
	}
	if len(universalPaths) == 0 {
		log.Fatalf("%s: no such file or directory", pattern)
	}
	return universalPaths
}

type stringFlags []string

func (sf *stringFlags) String() string {
	return "multi-string"
}

func (sf *stringFlags) Set(value string) error {
	*sf = append(*sf, value)
	return nil
}
