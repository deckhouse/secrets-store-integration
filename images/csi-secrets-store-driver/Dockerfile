ARG BASE_GOLANG_ALPINE=golang:1.24.2-alpine3.20
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
ARG BASE_ALPINE=registry.deckhouse.io/base_images/alpine:3.20.3

FROM $BASE_ALPINE as mount
RUN apk add --no-cache build-base asciidoctor libcap-ng-dev\
    linux-headers bison automake autoconf libtool ncurses-dev zlib-dev linux-pam-dev libeconf-dev utmps-dev utmps-static flex git
RUN cd / && \
    git clone --depth 1 --branch v2.40.4 https://github.com/util-linux/util-linux.git && \
    cd /util-linux && \
    ./autogen.sh && \
    export LDFLAGS="-Wl,-z,now -Wl,-z,relro -static -s" && \
    export LDFLAGS="-fPIC -pie -fstack-protector-all -O2 -D_FORTIFY_SOURCE=2 -static -s" && \
    ./configure --disable-all-programs  --enable-libmount --enable-libblkid --enable-static-programs=mount,umount \
                --enable-mount --enable-umount --enable-static --disable-pylibmount --without-python && \
    make mount.static umount.static && \
    mv ./mount.static /mount && \
    mv ./umount.static /umount && \
    chown -R 64535:64535 /mount /umount && \
    chmod 0700 /mount /umount && \
    cd / && rm -rf util-linux

FROM $BASE_GOLANG_ALPINE as artifact
RUN apk add --no-cache git
WORKDIR /build
ADD secrets-store-csi-driver /build
ADD patches /build/patches
RUN git apply -v patches/*.patch
RUN BUILD_TIMESTAMP=$(date +%Y-%m-%d-%H:%M) && \
    export LDFLAGS="-s -w -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.BuildTime=$(BUILD_TIMESTAMP) -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.BuildVersion=v1.4.0 -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.Vcs=fe" && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$LDFLAGS" ./cmd/secrets-store-csi-driver
RUN chown -R 64535:64535 secrets-store-csi-driver && \
    chmod 0700 secrets-store-csi-driver && mv secrets-store-csi-driver secrets-store-csi

FROM $BASE_DISTROLESS
COPY --from=mount /mount /bin/mount
COPY --from=mount /umount /bin/umount
COPY --from=artifact /build/secrets-store-csi /secrets-store-csi

ENTRYPOINT ["/secrets-store-csi"]
