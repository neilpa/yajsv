// yajsv is a command line tool for validating JSON documents against
// a provided JSON Schema - https://json-schema.org/
package main

import (
	"flag"
    "fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

    "github.com/xeipuuv/gojsonschema"
)

var (
	schemaFlag = flag.String("s", "", "primary JSON schema to validate against")
	refFlags stringFlags
	// TODO quiet and progress flags
)

func init() {
	flag.Var(&refFlags, "r", "referenced schema(s), can be globs and/or used multiple times")
}

func main() {
	flag.Parse()
	if *schemaFlag == "" {
		fmt.Fprintf(os.Stderr, "usage: %s -s SCHEMA [-r REF_SCHEMA -r REF_SCHEMA ...] DATA1 DATA2 ...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	sl := gojsonschema.NewSchemaLoader()
	schemaUri := fileUri(*schemaFlag)

	for _, ref := range refFlags {
		paths, err := filepath.Glob(ref)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range paths {
			uri := fileUri(p)
			if uri == schemaUri {
				continue
			}
			loader := gojsonschema.NewReferenceLoader(uri)
			err := sl.AddSchemas(loader)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

    schemaLoader := gojsonschema.NewReferenceLoader(schemaUri)
	schema, err := sl.Compile(schemaLoader)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	// Limit the number of simultaneously open files to avoid ulimit issues
	sem := make(chan int, runtime.GOMAXPROCS(0)+10)

	for _, arg := range flag.Args() {
		log.Println("before glob")
		docPaths, err := filepath.Glob(arg)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("after glob")

		for _, p := range docPaths {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				sem <- 0
				defer func() { <-sem }()

				loader := gojsonschema.NewReferenceLoader(fileUri(path))
				result, err := schema.Validate(loader)
				switch {
				case err != nil:
					fmt.Printf("%s: error: %s\n", err)
				case result.Valid():
					fmt.Printf("%s: ok\n", path)
				default:
					fmt.Printf("%s: invalid\n", path)
					for _, desc := range result.Errors() {
						fmt.Printf("%s: \t%s\n", path, desc)
					}
				}
			}(p)
		}
	}
	wg.Wait()

}

func fileUri(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return "file://" + abs
}

type stringFlags []string

func (sf *stringFlags) String() string {
    return "multi-string"
}

func (sf *stringFlags) Set(value string) error {
	*sf = append(*sf, value)
	return nil
}
