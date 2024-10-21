ARG BASE_GOLANG_23_ALPINE=registry.deckhouse.io/base_images/golang:1.23.1-alpine3.20
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest

FROM alpine:3.18 as mount
RUN apk add --no-cache build-base asciidoctor libcap-ng-dev\
    linux-headers bison automake autoconf libtool ncurses-dev zlib-dev linux-pam-dev python3-dev libeconf-dev utmps-dev utmps-static git
RUN cd / && \
    git clone --depth 1 --branch v2.39.2 https://github.com/util-linux/util-linux.git && \
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

FROM $BASE_GOLANG_23_ALPINE as artifact

ARG SOURCE_REPO=https://github.com/kubernetes-sigs/secrets-store-csi-driver.git
ENV SOURCE_REPO=${SOURCE_REPO}
ARG SOURCE_TAG=v1.4.6
ENV SOURCE_TAG=${SOURCE_TAG}

RUN apk add --no-cache git
RUN cd / && git clone --depth 1 --branch ${SOURCE_TAG} ${SOURCE_REPO}
WORKDIR /secrets-store-csi-driver
ADD secrets-store-csi-driver.patch /secrets-store-csi-driver
RUN git apply -v secrets-store-csi-driver.patch
RUN BUILD_TIMESTAMP=$(date +%Y-%m-%d-%H:%M) && \
    export LDFLAGS="-s -w -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.BuildTime=$(BUILD_TIMESTAMP) -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.BuildVersion=v1.4.0 -X sigs.k8s.io/secrets-store-csi-driver/pkg/version.Vcs=fe" && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$LDFLAGS" ./cmd/secrets-store-csi-driver
RUN chown -R 64535:64535 secrets-store-csi-driver && \
    chmod 0700 secrets-store-csi-driver && mv secrets-store-csi-driver secrets-store-csi

FROM $BASE_DISTROLESS
COPY --from=mount /mount /bin/mount
COPY --from=mount /umount /bin/umount
COPY --from=artifact /secrets-store-csi-driver/secrets-store-csi /secrets-store-csi

ENTRYPOINT ["/secrets-store-csi"]
