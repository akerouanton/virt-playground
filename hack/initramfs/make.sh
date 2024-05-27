#!/bin/bash

[[ "${DEBUG:-}" != "" ]] && set -o xtrace

BUSYBOX_VERSION=${BUSYBOX_VERSION:-1_36_0}
TARGET=${TARGET:-sources}

docker build \
    --progress=plain \
    --target=${TARGET} \
    --build-arg=BUSYBOX_VERSION=${BUSYBOX_VERSION} \
    --tag initramfs:${TARGET} \
    hack/initramfs

docker run --rm -it \
    -v ./hack/initramfs/config:/build/.config \
    initramfs:${TARGET} $@
