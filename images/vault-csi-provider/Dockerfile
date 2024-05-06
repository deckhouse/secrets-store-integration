ARG BASE_GOLANG_19_ALPINE=golang:1.20.5-alpine3.18
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
FROM $BASE_GOLANG_19_ALPINE as artifact

ARG SOURCE_REPO=https://github.com/hashicorp/vault-csi-provider.git
ENV SOURCE_REPO=${SOURCE_REPO}
ARG SOURCE_SHA1=77cca3cb7b6937c06e29f5caf679221fa6af1cf0
ENV SOURCE_SHA1=${SOURCE_SHA1}

RUN apk add --no-cache make git
RUN mkdir -p /build && cd /build && git init -b main && git remote add origin $SOURCE_REPO && \
    git fetch --depth 1 origin $SOURCE_SHA1 && git checkout FETCH_HEAD
WORKDIR /build/
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o vault-csi-provider && \
    chown -R 64535:64535 /build/vault-csi-provider && \
    chmod 0700 /build/vault-csi-provider

FROM $BASE_DISTROLESS
COPY --from=artifact /build/vault-csi-provider /bin

ENTRYPOINT ["/bin/vault-csi-provider"]
