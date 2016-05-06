#!/bin/sh

set -ex
env

genver() {
    if [ -n "$HMAKE_RELEASE" ]; then
        SUFFIX=""
    elif [ -n "$HMAKE_VER_SUFFIX" ]; then
        SUFFIX="$HMAKE_VER_SUFFIX"
    else
        SUFFIX=$(git log -1 --format=%h)
        test -n "$SUFFIX"
        SUFFIX="-g$SUFFIX"
    fi
    cat <<EOF >hmake-ver.gen.go
// +build genver

package main

// VersionSuffix of hmake
const VersionSuffix = "${SUFFIX}"
EOF
}

versuffix() {
    grep 'const VersionSuffix =' $1 | sed -r 's/^.+"([^"]*)".*$/\1/'
}

build() {
    TAGS="static_build netgo"
    RELEASE=$(grep 'Release = ' main.go | sed -r 's/^.+"([^"]+)".*$/\1/')
    if [ -f "hmake-ver.gen.go" ]; then
        SUFFIX=$(versuffix hmake-ver.gen.go)
        TAGS="$TAGS genver"
    else
        SUFFIX=$(versuffix hmake-ver.go)
    fi
    OUT=bin/hmake-${RELEASE}${SUFFIX}
    if [ -n "$1" -a -n "$2" ]; then
        export GOOS="$1"
        export GOARCH="$2"
        OUT=$OUT-$GOOS-$GOARCH
    fi
    go build -o $OUT \
        -a -tags "$TAGS" -installsuffix netgo \
        -ldflags '-extldflags -static' \
        .
    gzip -c $OUT >$OUT.gz
    cat $OUT.gz | sha1sum >$OUT.gz.sha1sum
}

case "$1" in
    genver) genver ;;
    *) build $@ ;;
esac
