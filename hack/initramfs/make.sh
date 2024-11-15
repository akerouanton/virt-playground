#!/bin/bash

[[ "${DEBUG:-}" != "" ]] && set -o xtrace

BUSYBOX_VERSION=${BUSYBOX_VERSION:-1_36_0}
TARGET=${TARGET:-sources}
VARIANT=${VARIANT:-alpine}

docker build \
    --progress=plain \
    --target=${TARGET} \
    --build-arg=BUSYBOX_VERSION=${BUSYBOX_VERSION} \
    --tag initramfs:${TARGET} \
    --file hack/initramfs/Dockerfile.${VARIANT} \
    hack/initramfs

docker run --rm -it \
    -v ./hack/initramfs/config:/build/.config \
    initramfs:${TARGET} $@
