# yajsv

[![CI](https://github.com/neilpa/yajsv/workflows/CI/badge.svg)](https://github.com/neilpa/yajsv/actions/)

Yet Another [JSON-Schema](https://json-schema.org) Validator. Command line tool for validating JSON and/or YAML documents against provided schemas.

The real credit goes to [xeipuuv/gojsonschema](https://github.com/xeipuuv/gojsonschema) which does the heavy lifting behind this CLI.

## Installation

Simply use `go get` to install

```
go get neilpa.me/yajsv
```

There are also pre-built static binaries for Windows, Mac and Linux on the [releases tab](https://github.com/neilpa/yajsv/releases/latest).

## Usage

Yajsv validates JSON (and/or YAML) documents against a JSON-Schema, providing a status per document:

  * pass: Document is valid relative to the schema
  * fail: Document is invalid relative to the schema
  * error: Document is malformed, e.g. not valid JSON or YAML

The 'fail' status may be reported multiple times per-document, once for each schema validation failure.

Basic usage example

```
$ yajsv -s schema.json document.json
document.json: pass
```

Or with both schema and doc in YAML.

```
$ yajsv -s schema.yml document.yml
document.yml: pass
```

With multiple schema files and docs

```
$ yajsv -s schema.json -r foo.json -r bar.yaml doc1.json doc2.yaml
doc1.json: pass
doc2.json: pass
```

Or with file globs (note the quotes to side-step shell expansion)

```
$ yajsv -s main.schema.json -r '*.schema.json' 'docs/*.json'
docs/a.json: pass
docs/b.json: fail: Validation failure message
...
```

Note that each referenced schema is assumed to be a path on the local filesystem. These are not
URI references to either local or external files.

See `yajsv -h` for more details

## Licence

[MIT](/LICENSE)
