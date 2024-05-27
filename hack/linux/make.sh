#!/bin/bash

[[ "${DEBUG:-}" != "" ]] && set -o xtrace

KERNEL_VERSION=${KERNEL_VERSION:-6.9.2}
TARGET=${TARGET:-sources}

docker build \
    --progress=plain \
    --target=${TARGET} \
    --build-arg=KERNEL_VERSION=${KERNEL_VERSION} \
    --tag kernel:${TARGET} \
    hack/linux

docker run --rm -it \
    -v ./hack/linux/config-${KERNEL_VERSION}:/build/.config \
    kernel:${TARGET} $@
