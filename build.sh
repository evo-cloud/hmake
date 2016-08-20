#!/bin/sh

set -ex

versuffix() {
    if [ -n "$HMAKE_RELEASE" ]; then
        echo -n ""
    elif [ -n "$HMAKE_RC" ]; then
        echo -n "$HMAKE_RC"
    else
        local suffix=$(git log -1 --format=%h || true)
        if [ -n "$suffix" ]; then
            echo -n -g$suffix
        else
            echo -n dev
        fi
    fi
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

checkfmt() {
    local files="$(gofmt -l . | grep -v vendor)"
    if [ -n "$files" ]; then
        echo "$files" >&2
        return 1
    fi
}

lint() {
    gometalinter \
        --disable=gotype \
        --vendor \
        --skip=examples \
        --skip=test \
        --deadline=60s \
        --severity=golint:error \
        --errors \
        ./...
}

build() {
    VERSION_SUFFIX=$(versuffix)
    RELEASE=$(grep 'Release = ' main.go | sed -r 's/^.+"([^"]+)".*$/\1/')
    VERSION=${RELEASE}${VERSION_SUFFIX}
    OUT=bin/hmake
    if [ -n "$1" -a -n "$2" ]; then
        export GOOS="$1"
        export GOARCH="$2"
        OUT=bin/$GOOS/$GOARCH/hmake
        PKG_SUFFIX=-$GOOS-$GOARCH
        if [ "$GOOS" == "windows" ]; then
            OUT=$OUT.exe
        fi
    fi
    mkdir -p $(dirname $OUT)
    CGO_ENABLED=0 go build -o $OUT \
        -a -tags "static_build netgo" -installsuffix netgo \
        -ldflags "-X main.VersionSuffix=${VERSION_SUFFIX} -extldflags -static" \
        .

    PKG=bin/hmake
    if [ "$GOOS" == "windows" ]; then
        PKG=${PKG}${PKG_SUFFIX}.zip
        rm -f $PKG
        zip -jX9 $PKG $OUT
    else
        PKG=${PKG}${PKG_SUFFIX}.tar.gz
        rm -f $PKG
        tar --posix --owner=0 --group=0 --no-acls --no-xattrs \
            --transform="s/$(basename $OUT)/hmake/" \
            -C $(dirname $OUT) -czf $PKG $(basename $OUT)
    fi
    cat $PKG | sha256sum >$PKG.sha256sum
}

case "$1" in
    gensite) gensite ;;
    lint) lint ;;
    checkfmt) checkfmt ;;
    *) build $@ ;;
esac
