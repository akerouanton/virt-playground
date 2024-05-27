#!/bin/bash

set -o errexit
[[ "${DEBUG:-}" != "" ]] && set -o xtrace

KERNEL_VERSION=${KERNEL_VERSION:-6.9.2}

while [ $# -ge 1 ]; do
    case "$1" in
        kernel)
            docker build \
                --progress=plain \
                --target=out-vmlinux \
                --build-arg=KERNEL_VERSION=${KERNEL_VERSION} \
                --output=type=local,dest=build/ \
                hack/linux
        ;;

        kernel-menuconfig)
            ./hack/linux/make.sh make menuconfig
        ;;

        *)
            echo "ERROR: Unsupported command '$1'"
            echo ""

            echo "Usage: $0 kernel|kernel-menuconfig"
            echo ""

            echo "Subcommands:"
            echo "    kernel:             Build the kernel"
            echo "    kernel-menuconfig:  Open Kernel's menuconfig"

            exit 1
        ;;
    esac

    shift
done
