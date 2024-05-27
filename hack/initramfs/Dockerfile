FROM alpine AS build-tools

RUN apk add --no-cache \
    alpine-sdk \
    ncurses-dev \
    perl

####################

FROM build-tools AS sources

ARG BUSYBOX_VERSION

RUN mkdir /build && \
    cd /build && \
    git clone --depth 1 --branch ${BUSYBOX_VERSION} https://github.com/mirror/busybox

# Copy kernel headers to make sure we've everything needed to compile in the
# 'build' target.
COPY headers/ /usr/include/

WORKDIR /build/busybox

COPY entrypoint /entrypoint
ENTRYPOINT ["/entrypoint"]
CMD ["/bin/sh", "-s"]

####################

FROM sources AS build-busybox

RUN --mount=type=bind,source=config,target=/build/busybox/config \
    cp /build/busybox/config /build/busybox/.config && \
    make -j $(getconf _NPROCESSORS_ONLN) && \
    make install && \
    rm _install/linuxrc

####################

FROM build-tools AS build-rootfs

RUN mkdir -p /rootfs/dev /rootfs/etc /rootfs/proc /rootfs/run /rootfs/sys

COPY --from=build-busybox /build/busybox/_install /rootfs
COPY --from=build-busybox /lib/ld-musl-aarch64.so.1 /rootfs/lib/
COPY --from=build-busybox /lib/libc.musl-aarch64.so.1 /rootfs/lib/
COPY init /rootfs/init

RUN mknod -m 644 /rootfs/dev/random c 1 8 && \
    mknod -m 644 /rootfs/dev/urandom c 1 9

RUN mkdir /boot && \
    cd /rootfs && \
    find . -print0 | cpio --null -ovt --format=newc | gzip -9 > /boot/initramfs

####################

FROM scratch AS out

COPY --from=build-rootfs /boot/ /
