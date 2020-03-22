// +build windows !windows

package main

import (
	"log"
)

func ExampleMain_pass_ymlschema_ymldoc() {
	exit := realMain([]string{"-s", "testdata/schema.yml", "testdata/data-pass.yml"})
	if exit != 0 {
		log.Fatalf("exit: got %d, want 0", exit)
	}
	// Output:
	// testdata/data-pass.yml: pass
}

func ExampleMain_pass_jsonschema_ymldoc() {
	exit := realMain([]string{"-s", "testdata/schema.json", "testdata/data-pass.yml"})
	if exit != 0 {
		log.Fatalf("exit: got %d, want 0", exit)
	}
	// Output:
	// testdata/data-pass.yml: pass
}

func ExampleMain_pass_jsonschema_jsondoc() {
	exit := realMain([]string{"-s", "testdata/schema.json", "testdata/data-pass.json"})
	if exit != 0 {
		log.Fatalf("exit: got %d, want 0", exit)
	}
	// Output:
	// testdata/data-pass.json: pass
}

func ExampleMain_pass_ymlschema_jsondoc() {
	exit := realMain([]string{"-s", "testdata/schema.yml", "testdata/data-pass.json"})
	if exit != 0 {
		log.Fatalf("exit: got %d, want 0", exit)
	}
	// Output:
	// testdata/data-pass.json: pass
}

func ExampleMain_fail_ymlschema_ymldoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.yml", "testdata/data-fail.yml"})
	if exit != 1 {
		log.Fatalf("exit: got %d, want 1", exit)
	}
	// Output:
	// testdata/data-fail.yml: fail: (root): foo is required
}

func ExampleMain_fail_jsonschema_ymldoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-fail.yml"})
	if exit != 1 {
		log.Fatalf("exit: got %d, want 1", exit)
	}
	// Output:
	// testdata/data-fail.yml: fail: (root): foo is required
}

func ExampleMain_fail_jsonschema_jsondoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-fail.json"})
	if exit != 1 {
		log.Fatalf("exit: got %d, want 1", exit)
	}
	// Output:
	// testdata/data-fail.json: fail: (root): foo is required
}

func ExampleMain_fail_ymlschema_jsondoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.yml", "testdata/data-fail.json"})
	if exit != 1 {
		log.Fatalf("exit: got %d, want 1", exit)
	}
	// Output:
	// testdata/data-fail.json: fail: (root): foo is required
}

func ExampleMain_error_jsonschema_jsondoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-error.json"})
	if exit != 2 {
		log.Fatalf("exit: got %d, want 2", exit)
	}
	// Output:
	// testdata/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')
}

func ExampleMain_error_ymlschema_ymldoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.yml", "testdata/data-error.yml"})
	if exit != 2 {
		log.Fatalf("exit: got %d, want 2", exit)
	}
	// Output:
	// testdata/data-error.yml: error: load doc yaml: found unexpected end of stream
}

func ExampleMain_glob_jsonschema_jsondoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.json", "testdata/data-*.json"})
	if exit != 3 {
		log.Fatalf("exit: got %d, want 3", exit)
	}
	// Unordered output:
	// testdata/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')
	// testdata/data-fail.json: fail: (root): foo is required
}

func ExampleMain_glob_ymlschema_ymldoc() {
	exit := realMain([]string{"-q", "-s", "testdata/schema.yml", "testdata/data-*.yml"})
	if exit != 3 {
		log.Fatalf("exit: got %d, want 3", exit)
	}
	// Unordered output:
	// testdata/data-fail.yml: fail: (root): foo is required
	//
	// testdata/data-error.yml: error: load doc yaml: found unexpected end of stream
}
