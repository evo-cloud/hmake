#!/bin/sh

set -ex
env

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

OUT=bin/hmake-1.0.0$SUFFIX
if [ -n "$1" -a -n "$2" ]; then
    export GOOS="$1"
    export GOARCH="$2"
    OUT=$OUT-$GOOS-$GOARCH
fi
go build -o $OUT \
    -a -tags 'genver static_build netgo' -installsuffix netgo \
    -ldflags '-extldflags -static' \
    .
gzip -c $OUT >$OUT.gz
cat $OUT.gz | sha1sum >$OUT.gz.sha1sum
