# yajsv

Yet Another [JSON-Schema](https://json-schema.org) Validator. Command line tool for validating JSON documents against provided schemas. Assumes referenced schemas and documents are on the local file-system.

The real credit goes to [xeipuuv/gojsonschema](https://github.com/xeipuuv/gojsonschema) which this wraps to create the CLI.

## Installation

Simply use `go get` to install

```
go get github.com/neilpa/yajsv
```

## Usage

```
yajsv -s schema.json [-r ref-schema.json -r ...] document.json [...]

  -r value
    	referenced schema(s), can be globs and/or used multiple times
  -s string
    	primary JSON schema to validate against
```
