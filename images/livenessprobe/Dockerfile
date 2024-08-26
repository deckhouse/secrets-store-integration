ARG BASE_GOLANG_20_ALPINE=registry.deckhouse.io/base_images/golang:1.21.4-alpine3.18@sha256:cf84f3d6882c49ea04b6478ac514a2582c8922d7e5848b43d2918fff8329f6e6
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
FROM $BASE_GOLANG_20_ALPINE as artifact

ARG SOURCE_REPO=https://github.com/kubernetes-csi/livenessprobe.git
ENV SOURCE_REPO=${SOURCE_REPO}
ARG SOURCE_SHA1=14ea471689e7c7a669a96b3b0aa99434e74a8732
ENV SOURCE_SHA1=${SOURCE_SHA1}

RUN apk add --no-cache make git bash
RUN mkdir -p /build && cd /build && git init -b main && git remote add origin $SOURCE_REPO && \
    git fetch --depth 1 origin $SOURCE_SHA1 && git checkout FETCH_HEAD
WORKDIR /build/
RUN make && \
    chown -R 64535:64535 /build/bin/livenessprobe && \
    chmod 0700 /build/bin/livenessprobe

FROM $BASE_DISTROLESS
COPY --from=artifact /build/bin/livenessprobe /

ENTRYPOINT ["/livenessprobe"]