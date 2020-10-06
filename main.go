// yajsv is a command line tool for validating JSON and YAML documents against
// a provided JSON Schema - https://json-schema.org/
package main

//go:generate go run gen_testdata.go

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
	"github.com/xeipuuv/gojsonschema"
)

var (
	version     = "v1.4.0-dev"
	schemaFlag  = flag.String("s", "", "primary JSON schema to validate against, required")
	quietFlag   = flag.Bool("q", false, "quiet, only print validation failures and errors")
	versionFlag = flag.Bool("v", false, "print version and exit")
	bomFlag     = flag.Bool("b", false, "allow BOM in JSON files, error if seen and unset")

	listFlags stringFlags
	refFlags  stringFlags
)

// https://en.wikipedia.org/wiki/Byte_order_mark#Byte_order_marks_by_encoding
const (
	bomUTF8    = "\xEF\xBB\xBF"
	bomUTF16BE = "\xFE\xFF"
	bomUTF16LE = "\xFF\xFE"
)

var (
	encUTF16BE = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	encUTF16LE = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
)

func init() {
	flag.Var(&listFlags, "l", "validate JSON documents from newline separated paths and/or globs in a text file (relative to the basename of the file itself)")
	flag.Var(&refFlags, "r", "referenced schema(s), can be globs and/or used multiple times")
	flag.Usage = printUsage
}

func main() {
	log.SetFlags(0)
	os.Exit(realMain(os.Args[1:], os.Stdout))
}

func realMain(args []string, w io.Writer) int {
	flag.CommandLine.Parse(args)
	if *versionFlag {
		fmt.Fprintln(w, version)
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
			return schemaError("%s: %s", list, err)
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
			return schemaError("%s: invalid file list: %s", list, err)
		}
	}
	if len(docs) == 0 {
		return usageError("no documents to validate")
	}

	// Compile target schema
	sl := gojsonschema.NewSchemaLoader()
	schemaPath, err := filepath.Abs(*schemaFlag)
	if err != nil {
		return schemaError("%s: unable to convert to absolute path: %s", *schemaFlag, err)
	}
	for _, ref := range refFlags {
		for _, p := range glob(ref) {
			absPath, err := filepath.Abs(p)
			if err != nil {
				return schemaError("%s: unable to convert to absolute path: %s", absPath, err)
			}

			if absPath == schemaPath {
				continue
			}

			loader, err := jsonLoader(absPath)
			if err != nil {
				return schemaError("%s: unable to load schema ref: %s", *schemaFlag, err)
			}

			if err := sl.AddSchemas(loader); err != nil {
				return schemaError("%s: invalid schema: %s", p, err)
			}
		}
	}

	schemaLoader, err := jsonLoader(schemaPath)
	if err != nil {
		return schemaError("%s: unable to load schema: %s", *schemaFlag, err)
	}
	schema, err := sl.Compile(schemaLoader)
	if err != nil {
		return schemaError("%s: invalid schema: %s", *schemaFlag, err)
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

			loader, err := jsonLoader(path)
			if err != nil {
				msg := fmt.Sprintf("%s: error: load doc: %s", path, err)
				fmt.Fprintln(w, msg)
				errors = append(errors, msg)
				return
			}
			result, err := schema.Validate(loader)
			switch {
			case err != nil:
				msg := fmt.Sprintf("%s: error: validate: %s", path, err)
				fmt.Fprintln(w, msg)
				errors = append(errors, msg)

			case !result.Valid():
				lines := make([]string, len(result.Errors()))
				for i, desc := range result.Errors() {
					lines[i] = fmt.Sprintf("%s: fail: %s", path, desc)
				}
				msg := strings.Join(lines, "\n")
				fmt.Fprintln(w, msg)
				failures = append(failures, msg)

			case !*quietFlag:
				fmt.Fprintf(w, "%s: pass\n", path)
			}
		}(p)
	}
	wg.Wait()

	// Summarize results (e.g. errors)
	if !*quietFlag {
		if len(failures) > 0 {
			fmt.Fprintf(w, "%d of %d failed validation\n", len(failures), len(docs))
			fmt.Fprintln(w, strings.Join(failures, "\n"))
		}
		if len(errors) > 0 {
			fmt.Fprintf(w, "%d of %d malformed documents\n", len(errors), len(docs))
			fmt.Fprintln(w, strings.Join(errors, "\n"))
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
		// TODO YAML requires the precense of a BOM to detect UTF-16
		// text. Is there a decent hueristic to detect UTF-16 text
		// missing a BOM so we can provide a better error message?
		buf, err = yaml.YAMLToJSON(buf)
	default:
		buf, err = jsonDecodeCharset(buf)
	}
	if err != nil {
		return nil, err
	}
	// TODO What if we have an empty document?
	return gojsonschema.NewBytesLoader(buf), nil
}

// jsonDecodeCharset attempts to detect UTF-16 (LE or BE) JSON text and
// decode as appropriate. It also skips a BOM at the start of the buffer
// if `-b` was specified. Presence of a BOM is an error otherwise.
func jsonDecodeCharset(buf []byte) ([]byte, error) {
	if len(buf) < 2 { // UTF-8
		return buf, nil
	}

	bom := ""
	var enc encoding.Encoding
	switch {
	case bytes.HasPrefix(buf, []byte(bomUTF8)):
		bom = bomUTF8
	case bytes.HasPrefix(buf, []byte(bomUTF16BE)):
		bom = bomUTF16BE
		enc = encUTF16BE
	case bytes.HasPrefix(buf, []byte(bomUTF16LE)):
		bom = bomUTF16LE
		enc = encUTF16LE
	case buf[0] == 0:
		enc = encUTF16BE
	case buf[1] == 0:
		enc = encUTF16LE
	}

	if bom != "" {
		if !*bomFlag {
			return nil, fmt.Errorf("unexpected BOM, see `-b` flag")
		}
		buf = buf[len(bom):]
	}
	if enc != nil {
		return enc.NewDecoder().Bytes(buf)
	}
	return buf, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -s schema.(json|yml) [options] document.(json|yml) ...

  yajsv validates JSON and YAML document(s) against a schema. One of three status
  results are reported per document:

    pass: Document is valid relative to the schema
    fail: Document is invalid relative to the schema
    error: Document is malformed, e.g. not valid JSON or YAML

  The 'fail' status may be reported multiple times per-document, once for each
  schema validation failure.

  Sets the exit code to 1 on any failures, 2 on any errors, 3 on both, 4 on
  invalid usage, 5 on schema definition or file-list errors. Otherwise, 0 is
  returned if everything passes validation.

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

func schemaError(format string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	return 5
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
