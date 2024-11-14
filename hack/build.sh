#!/bin/bash

set -o errexit
set -o xtrace

KERNEL_VERSION=${KERNEL_VERSION:-6.9.2}
BUSYBOX_VERSION=${BUSYBOX_VERSION:-1_36_0}

while [ $# -ge 1 ]; do
    case "$1" in
        virt)
            go build -o build/virt ./cmd/virt
            codesign --deep --force --options=runtime \
                --entitlements=hack/entitlements.plist \
                --sign - \
                build/virt
        ;;

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
                --target=out-initramfs \
                --build-arg=BUSYBOX_VERSION=${BUSYBOX_VERSION} \
                --output=type=local,dest=build/ \
                --file=hack/initramfs/Dockerfile.${VARIANT:-alpine} \
                hack/initramfs
        ;;

        busybox-menuconfig)
            VARIANT=busybox ./hack/initramfs/make.sh make defconfig menuconfig
            # See http://lists.busybox.net/pipermail/busybox-cvs/2024-January/041752.html
            sed -i -e 's/CONFIG_TC=y/CONFIG_TC=n/' ./hack/initramfs/config
        ;;

        rootfs)
            # Generate an initramfs
            docker build \
                --progress=plain \
                --target=out-rootfs \
                --output=type=local,dest=build/rootfs \
                --file=hack/initramfs/Dockerfile.${VARIANT:-alpine} \
                hack/initramfs
        ;;

        *)
            echo "ERROR: Unsupported command '$1'"
            echo ""

            echo "Usage: $0 virt|kernel|kernel-menuconfig|initramfs|busybox-menuconfig"
            echo ""

            echo "Subcommands:"
            echo "    virt:               Build the 'virt' binary"
            echo "    kernel:             Build the kernel"
            echo "    kernel-menuconfig:  Open Kernel's menuconfig"
            echo "    initramfs:          Build the initramfs file"
            echo "    rootfs:             Build an uncompressed / unarchived rootfs"
            echo "    busybox-menuconfig: Open BusyBox's menuconfig"

            exit 1
        ;;
    esac

    shift
done
