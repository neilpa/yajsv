# yajsv

Yet Another [JSON-Schema](https://json-schema.org) Validator. Command line tool for validating JSON documents against provided schemas.

The real credit goes to [xeipuuv/gojsonschema](https://github.com/xeipuuv/gojsonschema) which does the heavy lifting behind this CLI.

## Installation

Simply use `go get` to install

```
go get github.com/neilpa/yajsv
```

## Usage

yajsv validates JSON documents against a schema, providing a status per document:

  * pass: Document is valid relative to the schema
  * fail: Document is invalid relative to the schema
  * error: Document is malformed, e.g. not valid JSON

The 'fail' status may be reported multiple times per-document, once for each schema validation failure.

Basic usage

```
$ yajsv -s schema.json document.json
document.json: pass
```

With multiple schema files and docs

```
$ yajsv -s schema.json -r foo.json -r bar.json doc1.json doc2.json
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

Note that each of the referenced schema is assumed to be a path on the local filesystem. These are not
URI references to either local or external schemas and documents.

See `yajsv -h` for more details
