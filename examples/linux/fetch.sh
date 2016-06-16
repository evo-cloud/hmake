#!/bin/bash

KERNEL_URL=https://cdn.kernel.org/pub/linux/kernel/v4.x/linux-4.6.2.tar.xz
KERNEL=linux-4.6.2

set -ex
mkdir -p build/src
wget -nv -O build/src/$KERNEL.tar.xz $KERNEL_URL
tar -C build/src -Jxf build/src/$KERNEL.tar.xz
ln -sTf $KERNEL build/src/linux
