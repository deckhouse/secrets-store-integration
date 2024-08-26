ARG BASE_GOLANG_20_ALPINE=registry.deckhouse.io/base_images/golang:1.20.5-alpine3.18@sha256:51a47fb0851397db2f506c15c426735bc23de31177cbdd962880c0879d1906a4
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest@sha256:b61c8490b9de65167184c1c39b5125373b111929837aeff7184fe65ebdaee0c8
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