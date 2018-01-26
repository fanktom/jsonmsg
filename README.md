# jsonmsg
A Go package that parses API specs to generate server/client source code in any supported language
## Features

* Parses spec documents based on https://github.com/jsonmsg/spec
* Generates source code for any supported language (currently only Go/server)
* Test suite with shared schema fixtures
* Library and standalone compiler binary `jsonmsgc`

## GoDoc

Godoc is available from https://godoc.org/github.com/tfkhsr/jsonmsg.

## Install

To install as library run:

```
go get -u github.com/tfkhsr/jsonmsg
```

To install the standalone compiler binary `jsonmsgc` run:

```
go get -u github.com/tfkhsr/jsonmsg/cmd/jsonmsgc
```
