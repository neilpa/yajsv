// yajsv is a command line tool for validating JSON documents against
// a provided JSON Schema - https://json-schema.org/
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	//"neilpa.me/go-x/fileuri"
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
		return usageError("no JSON documents to validate")
	}


	// Compile target schema
	sl := gojsonschema.NewSchemaLoader()
	schemaUri := *schemaFlag
	for _, ref := range refFlags {
		for _, p := range glob(ref) {
			uri := fileUri(p)
			if uri == schemaUri {
				continue
			}
			var loader gojsonschema.JSONLoader = nil

			if strings.HasSuffix(uri, ".yaml") || strings.HasSuffix(uri, ".yml") {
				valuesJSON, err := convertToJson(uri)
				if err != nil {
					log.Fatal("unable to parse YAML schema\n", err)
				}
				loader = gojsonschema.NewBytesLoader(valuesJSON)
			} else {
				loader = gojsonschema.NewReferenceLoader(uri)
			}
			err := sl.AddSchemas(loader)
			if err != nil {
				log.Fatalf("%s: invalid schema: %s\n", p, err)
			}
		}
	}

	var schemaLoader gojsonschema.JSONLoader = nil

	if strings.HasSuffix(schemaUri, ".yaml") || strings.HasSuffix(schemaUri, ".yml") {
		valuesJSON, err := convertToJson(schemaUri)
		if err != nil {
			log.Fatal("unable to parse YAML schema\n", err)
		}
		schemaLoader = gojsonschema.NewBytesLoader(valuesJSON)
	} else {
		schemaLoader = gojsonschema.NewReferenceLoader(schemaUri)
	}

	//schemaLoader := gojsonschema.NewReferenceLoader(schemaUri)
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

			var loader gojsonschema.JSONLoader = nil

			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				valuesJSON, err := convertToJson(path)
				if err != nil {
					log.Fatal("unable to parse YAML\n", err)
				}
				loader = gojsonschema.NewBytesLoader(valuesJSON)
			} else {
				loader = gojsonschema.NewReferenceLoader(fileUri(path))
			}

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
	exit := 0
	if len(failures) > 0 {
		exit |= 1
	}
	if len(errors) > 0 {
		exit |= 2
	}
	return exit
}

func convertToJson(path string) ([]byte, error) {
	values, err := ReadValuesFile(path)
	if err != nil {
		//return errors.Wrap(err, "unable to parse YAML")
		return []byte{}, err
	}
	valuesData, err := yaml.Marshal(values)
	if err != nil {
		return []byte{}, err
	}
	valuesJSON, err := yaml.YAMLToJSON(valuesData)
	if err != nil {
		return []byte{}, err
	}
	if bytes.Equal(valuesJSON, []byte("null")) {
		valuesJSON = []byte("{}")
	}
	return valuesJSON, err
}

type Values map[string]interface{}

// ReadValues will parse YAML byte data into a Values.
func ReadValues(data []byte) (vals Values, err error) {
	err = yaml.Unmarshal(data, &vals)
	if len(vals) == 0 {
		vals = Values{}
	}
	return vals, err
}

// ReadValuesFile will parse a YAML file into a map of values.
func ReadValuesFile(filename string) (Values, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return ReadValues(data)
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
