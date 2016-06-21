#!/bin/sh

set -ex

fatal() {
    echo "$@" >&2
    exit 1
}

fetch() {
    rm -fr src/github.com/drone/drone
    mkdir -p src/github.com/drone
    git clone https://github.com/drone/drone src/github.com/drone/drone
    go generate github.com/drone/drone/static
    go generate github.com/drone/drone/template
    go generate github.com/drone/drone/store/datastore/ddl
}

build() {
    local os=$1 arch=$2 ldflags
    local out=release/$os/$arch
    mkdir -p $out
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 \
        go build -o $out/drone github.com/drone/drone/drone
}

cmd="$1"
shift
case "$cmd" in
    fetch) fetch $@ ;;
    build) build $@ ;;
    *) fatal "unknown command" ;;
esac
