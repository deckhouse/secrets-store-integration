ARG BASE_GOLANG_ALPINE=registry.deckhouse.io/base_images/golang:1.22.8-alpine
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
FROM $BASE_GOLANG_ALPINE as artifact

ARG SOURCE_REPO=https://github.com/kubernetes-csi/node-driver-registrar.git
ENV SOURCE_REPO=${SOURCE_REPO}
ARG SOURCE_SHA1=3bff1dbf16e154e8fe6783242af1ad8d0c507fa4
ENV SOURCE_SHA1=${SOURCE_SHA1}

RUN apk add --no-cache make git bash
RUN mkdir -p /build && cd /build && git init -b main && git remote add origin $SOURCE_REPO && \
    git fetch --depth 1 origin $SOURCE_SHA1 && git checkout FETCH_HEAD
WORKDIR /build/
RUN make && \
    chown -R 64535:64535 /build/bin/csi-node-driver-registrar && \
    chmod 0700 /build/bin/csi-node-driver-registrar

FROM $BASE_DISTROLESS
COPY --from=artifact /build/bin/csi-node-driver-registrar /

ENTRYPOINT ["/csi-node-driver-registrar"]
