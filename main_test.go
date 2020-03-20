package main

import (
	"log"
)

func ExampleMain_pass() {
	exit := realMain([]string{"-s", "testdata/schema.json", "testdata/data-pass.json"})
	if exit != 0 {
		log.Fatalf("exit: got %d, want 0", exit)
	}
	// Output:
	// testdata/data-pass.json: pass
}

func ExampleMain_fail() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-fail.json"})
	if exit != 1 {
		log.Fatalf("exit: got %d, want 1", exit)
	}
	// Output:
	// testdata/data-fail.json: fail: (root): foo is required
}

func ExampleMain_error() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-error.json"})
	if exit != 2 {
		log.Fatalf("exit: got %d, want 2", exit)
	}
	// Output:
	// testdata/data-error.json: error: invalid character 'o' in literal null (expecting 'u')
}

func ExampleMain_glob() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-*.json"})
	if exit != 3 {
		log.Fatalf("exit: got %d, want 3", exit)
	}
	// Unordered output:
	// testdata/data-error.json: error: invalid character 'o' in literal null (expecting 'u')
	// testdata/data-fail.json: fail: (root): foo is required
}
