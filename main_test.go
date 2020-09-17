package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	tests := []struct {
		in string
		out []string
		exit int
	} {
		{
			"-s testdata/schema.yml testdata/data-pass.yml",
			[]string{"testdata/data-pass.yml: pass"},
			0,
		}, {
			"-s testdata/schema.json testdata/data-pass.yml",
			[]string{"testdata/data-pass.yml: pass"},
			0,
		}, {
			"-s testdata/schema.json testdata/data-pass.json",
			[]string{"testdata/data-pass.json: pass"},
			0,
		}, {
			"-s testdata/schema.yml testdata/data-pass.json",
			[]string{"testdata/data-pass.json: pass"},
			0,
		}, {
			"-q -s testdata/schema.yml testdata/data-fail.yml",
			[]string{"testdata/data-fail.yml: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/schema.json testdata/data-fail.yml",
			[]string{"testdata/data-fail.yml: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/schema.json testdata/data-fail.json",
			[]string{"testdata/data-fail.json: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/schema.yml testdata/data-fail.json",
			[]string{"testdata/data-fail.json: fail: (root): foo is required"},
			1,
		}, {
			"-q -s testdata/schema.json testdata/data-error.json",
			[]string{"testdata/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')"},
			2,
		}, {
			"-q -s testdata/schema.yml testdata/data-error.yml",
			[]string{"testdata/data-error.yml: error: load doc: yaml: found unexpected end of stream"},
			2,
		}, {
			"-q -s testdata/schema.json testdata/data-*.json",
			[]string{
				"testdata/data-fail.json: fail: (root): foo is required",
				"testdata/data-error.json: error: validate: invalid character 'o' in literal null (expecting 'u')",
			}, 3,
		}, {
			"-q -s testdata/schema.yml testdata/data-*.yml",
			[]string{
				"testdata/data-error.yml: error: load doc: yaml: found unexpected end of stream",
				"testdata/data-fail.yml: fail: (root): foo is required",
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

