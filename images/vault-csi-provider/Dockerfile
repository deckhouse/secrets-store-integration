ARG BASE_GOLANG_ALPINE=registry.deckhouse.io/base_images/golang:1.23.6-alpine3.20
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
FROM $BASE_GOLANG_ALPINE as artifact

ARG SOURCE_REPO=https://github.com/hashicorp/vault-csi-provider.git
ENV SOURCE_REPO=${SOURCE_REPO}
ARG SOURCE_SHA1=3f1ac9bc273e060fec03f0ddacc72b3663949355
ENV SOURCE_SHA1=${SOURCE_SHA1}

RUN apk add --no-cache make git
RUN mkdir -p /build && cd /build && git init -b main && git remote add origin $SOURCE_REPO && \
    git fetch --depth 1 origin $SOURCE_SHA1 && git checkout FETCH_HEAD
ADD patches /build/patches
RUN cd /build && git apply -v patches/*.patch  && rm -rf tools
WORKDIR /build/
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o vault-csi-provider && \
    chown -R 64535:64535 /build/vault-csi-provider && \
    chmod 0700 /build/vault-csi-provider

FROM $BASE_DISTROLESS
COPY --from=artifact /build/vault-csi-provider /bin

ENTRYPOINT ["/bin/vault-csi-provider"]
