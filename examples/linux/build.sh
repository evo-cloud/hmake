#!/bin/bash

TARGET=${HMAKE_TARGET##target-}
SRCDIR=$HMAKE_PROJECT_DIR/build/src/linux
OUTDIR=$HMAKE_PROJECT_DIR/build/out/$TARGET

fatal() {
    echo "$@" >&2
    exit 1
}

build() {
    mkdir -p $OUTDIR
    make -C $SRCDIR O=$OUTDIR V=1 defconfig
    cp -f config $OUTDIR/.config
    make -C $OUTDIR ARCH=$1 oldconfig V=1
    make -C $OUTDIR ARCH=$1 CROSS_COMPILE=$2 all V=1
}

config() {
    TARGET=config
    OUTDIR=$HMAKE_PROJECT_DIR/build/out/$TARGET
    mkdir -p $OUTDIR
    make -C $SRCDIR O=$OUTDIR V=1 $@
}

clean() {
    rm -fr $OUTDIR
}

set -ex

cmd="$1"
shift
case "$cmd" in
    build) build $@ ;;
    clean) clean ;;
    config) config $@ ;;
    *) fatal "bad command" ;;
esac
