#!/bin/bash

set -o errexit
[[ "${DEBUG:-}" != "" ]] && set -o xtrace

KERNEL_VERSION=${KERNEL_VERSION:-6.9.2}
BUSYBOX_VERSION=${BUSYBOX_VERSION:-1_36_0}

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

        initramfs)
            [ ! -d hack/initramfs/headers ] && mkdir hack/initramfs/headers

            # Retrieve kernel headers
            docker build \
                --progress=plain \
                --target=out-headers \
                --build-arg=KERNEL_VERSION=${KERNEL_VERSION} \
                --output=type=local,dest=hack/initramfs/headers/ \
                hack/linux

            # Compile BusyBox and generate an initramfs
            docker build \
                --progress=plain \
                --target=out \
                --build-arg=BUSYBOX_VERSION=${BUSYBOX_VERSION} \
                --output=type=local,dest=build/ \
                hack/initramfs
        ;;

        busybox-menuconfig)
            ./hack/initramfs/make.sh make defconfig menuconfig
            # See http://lists.busybox.net/pipermail/busybox-cvs/2024-January/041752.html
            sed -i -e 's/CONFIG_TC=y/CONFIG_TC=n/' ./hack/initramfs/config
        ;;

        *)
            echo "ERROR: Unsupported command '$1'"
            echo ""

            echo "Usage: $0 kernel|kernel-menuconfig|initramfs|busybox-menuconfig"
            echo ""

            echo "Subcommands:"
            echo "    kernel:             Build the kernel"
            echo "    kernel-menuconfig:  Open Kernel's menuconfig"
            echo "    initramfs:          Build the 'init' binary and create the initramfs file"
            echo "    busybox-menuconfig: Open BusyBox's menuconfig"

            exit 1
        ;;
    esac

    shift
done
