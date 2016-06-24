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

gensite() {
    rm -fr site/gh-pages/content site/gh-pages/public
    mkdir -p site/gh-pages/content
    cp -rf docs site/gh-pages/content/
    for md in $(find examples -maxdepth 2 -name '*.md'); do
        mkdir -p site/gh-pages/content/$(dirname $md)
        cp -f $md site/gh-pages/content/$md
    done
    grep -F -v '[![Build Status]' README.md \
        | sed -r 's/^(#\s+)HyperMake/\1Introduction/' \
        > site/gh-pages/content/README.md
    cd site/gh-pages
    hugo
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

    if [ "$GOOS" == "windows" ]; then
        cp -f $OUT bin/hmake.exe
        PKG=$OUT.zip
        rm -f $PKG
        zip -jX9 $PKG bin/hmake.exe
    else
        PKG=$OUT.tar.gz
        tar --posix --owner=0 --group=0 --no-acls --no-xattrs \
            --transform="s/$(basename $OUT)/hmake/" \
            -C $(dirname $OUT) -czf $PKG $(basename $OUT)
    fi
    cat $PKG | sha1sum >$PKG.sha1sum
}

case "$1" in
    genver) genver ;;
    gensite) gensite ;;
    *) build $@ ;;
esac
