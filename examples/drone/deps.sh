#!/bin/sh

set -ex
go get -u golang.org/x/tools/cmd/cover
go get -u github.com/eknkc/amber/...
go get -u github.com/eknkc/amber
go get -u github.com/jteeuwen/go-bindata/...
go get -u github.com/elazarl/go-bindata-assetfs/...
go get -u github.com/dchest/jsmin
go get -u github.com/franela/goblin
