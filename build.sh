#!/bin/sh

set -ex

versuffix() {
    if [ -f ".release" ]; then
        . .release
    fi
    if [ -n "$RELEASE" ]; then
        case "$RELEASE" in
            y|yes|final) return ;;
            *) echo -n "-$RELEASE" ;;
        esac
    else
        local suffix=$(git log -1 --format=%h 2>/dev/null || true)
        if [ -n "$suffix" ]; then
            test -z "$(git status --porcelain 2>/dev/null || true)" || suffix="${suffix}+"
            echo -n "-g${suffix}"
        else
            echo -n -dev
        fi
    fi
}

gensite() {
    rm -fr site/gh-pages/public
    cd site/gh-pages
    mkdir -p static/json
    lunr-hugo -i 'content/**/*.md' -o static/json/search.json -l yaml
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

    if [ "$GOARCH" == "arm" ]; then
        export GOARM=7
    fi

    mkdir -p $(dirname $OUT)
    CGO_ENABLED=0 go build -o $OUT \
        -a -tags "static_build netgo" -installsuffix netgo \
        -ldflags "-s -w -X main.VersionSuffix=$(versuffix) -extldflags -static" \
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
