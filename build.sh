#!/bin/sh
mkdir -p bin
set -ex
apk update && apk add git
go get github.com/FiloSottile/gvt && gvt restore
export GOOS=$1 GOARCH=$2
go build -o bin/hmake-$GOOS-$GOARCH \
    -a -tags 'static_build netgo' -installsuffix netgo \
    -ldflags '-extldflags -static' \
    .
