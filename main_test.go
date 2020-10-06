package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func init() {
	// TODO: Cleanup this global monkey-patching
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		panic(err)
	}
	os.Stderr = devnull
}

func TestMain(t *testing.T) {
	tests := []struct {
		in   string
		out  []string
		exit int
	}{
		{
			"-s testdata/utf-16be_bom/schema.json testdata/utf-16le_bom/data-fail.yml",
			[]string{},
			5,
		}, {
			"-s testdata/utf-8/schema.yml testdata/utf-8/data-pass.yml",
			[]string{"testdata/utf-8/data-pass.yml: pass"},
			0,
		}, {
			"-s testdata/utf-8/schema.json testdata/utf-8/data-pass.yml",
			[]string{"testdata/utf-8/data-pass.yml: pass"},
			0,
		}, {
			"-s testdata/utf-8/schema.json testdata/utf-8/data-pass.json",
			[]string{"testdata/utf-8/data-pass.json: pass"},
			0,
		}, {
			"-s testdata/utf-8/schema.yml testdata/utf-8/data-pass.json",
			[]string{"testdata/utf-8/data-pass.json: pass"},
			0,
		}, {
			"-q -s testdata/utf-8/schema.yml testdata/utf-8/data-fail.yml",
			[]string{"testdata/utf-8/data-fail.yml: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/utf-8/schema.json testdata/utf-8/data-fail.yml",
			[]string{"testdata/utf-8/data-fail.yml: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/utf-8/schema.json testdata/utf-8/data-fail.json",
			[]string{"testdata/utf-8/data-fail.json: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/utf-8/schema.yml testdata/utf-8/data-fail.json",
			[]string{"testdata/utf-8/data-fail.json: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/utf-8/schema.json testdata/utf-8/data-error.json",
			[]string{"testdata/utf-8/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')"},
			2,
		}, {
			"-q -s testdata/utf-8/schema.yml testdata/utf-8/data-error.yml",
			[]string{"testdata/utf-8/data-error.yml: error: load doc: yaml: found unexpected end of stream"},
			2,
		}, {
			"-q -s testdata/utf-8/schema.json testdata/utf-8/data-*.json",
			[]string{
				"testdata/utf-8/data-fail.json: fail: (root): foo is required",
				"testdata/utf-8/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')",
			}, 3,
		}, {
			"-q -s testdata/utf-8/schema.yml testdata/utf-8/data-*.yml",
			[]string{
				"testdata/utf-8/data-error.yml: error: load doc: yaml: found unexpected end of stream",
				"testdata/utf-8/data-fail.yml: fail: (root): foo is required",
			}, 3,
		},
	}

	for _, tt := range tests {
		in := strings.Replace(tt.in, "/", string(filepath.Separator), -1)
		sort.Strings(tt.out)
		out := strings.Join(tt.out, "\n")
		out = strings.Replace(out, "/", string(filepath.Separator), -1)

		t.Run(in, func(t *testing.T) {
			var w strings.Builder
			exit := realMain(strings.Split(in, " "), &w)
			if exit != tt.exit {
				t.Fatalf("exit: got %d, want %d", exit, tt.exit)
			}
			lines := strings.Split(w.String(), "\n")
			sort.Strings(lines)
			got := strings.Join(lines[1:], "\n")
			if got != out {
				t.Errorf("got\n%s\nwant\n%s", got, out)
			}
		})
	}
}

func TestMatrix(t *testing.T) {
	// schema.{format} {encoding}{_bom}/data-{expect}.{format}
	type testcase struct {
		schemaEnc, schemaFmt      string
		dataEnc, dataFmt, dataRes string
		allowBOM                  bool
	}

	encodings := []string{"utf-8", "utf-16be", "utf-16le", "utf-8_bom", "utf-16be_bom", "utf-16le_bom"}
	formats := []string{"json", "yml"}
	results := []string{"pass", "fail", "error"}
	tests := []testcase{}

	// poor mans cartesian product
	for _, senc := range encodings {
		for _, sfmt := range formats {
			for _, denc := range encodings {
				for _, dfmt := range formats {
					for _, dres := range results {
						tests = append(tests, testcase{senc, sfmt, denc, dfmt, dres, false})
						tests = append(tests, testcase{senc, sfmt, denc, dfmt, dres, true})
					}
				}
			}
		}
	}

	for _, tt := range tests {
		schemaBOM := strings.HasSuffix(tt.schemaEnc, "_bom")
		schema16 := strings.HasPrefix(tt.schemaEnc, "utf-16")
		dataBOM := strings.HasSuffix(tt.dataEnc, "_bom")
		data16 := strings.HasPrefix(tt.dataEnc, "utf-16")

		schema := fmt.Sprintf("testdata/%s/schema.%s", tt.schemaEnc, tt.schemaFmt)
		data := fmt.Sprintf("testdata/%s/data-%s.%s", tt.dataEnc, tt.dataRes, tt.dataFmt)
		cmd := fmt.Sprintf("-s %s %s", schema, data)
		if tt.allowBOM {
			cmd = "-b " + cmd
		}

		t.Run(cmd, func(t *testing.T) {
			want := 0
			switch {
			// Schema Errors (exit = 5)
			// - YAML w/out BOM for UTF-16
			// - JSON w/ BOM but missing allowBOM flag
			case tt.schemaFmt == "yml" && !schemaBOM && schema16:
				want = 5
			case tt.schemaFmt == "json" && schemaBOM && !tt.allowBOM:
				want = 5
			// Data Errors (exit = 2)
			// - YAML w/out BOM for UTF-16
			// - JSON w/ BOM but missing allowBOM flag
			// - standard malformed files (e.g. data-error)
			case tt.dataFmt == "yml" && !dataBOM && data16:
				want = 2
			case tt.dataFmt == "json" && dataBOM && !tt.allowBOM:
				want = 2
			case tt.dataRes == "error":
				want = 2
			// Data Failures
			case tt.dataRes == "fail":
				want = 1
			}

			// TODO: Cleanup this global monkey-patching
			*bomFlag = tt.allowBOM

			var w strings.Builder
			got := realMain(strings.Split(cmd, " "), &w)
			if got != want {
				t.Errorf("got(%d) != want(%d) bomflag %t", got, want, *bomFlag)
			}
		})
	}
}
