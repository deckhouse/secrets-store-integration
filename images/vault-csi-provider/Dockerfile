ARG BASE_GOLANG_ALPINE=registry.deckhouse.io/base_images/golang:1.23.6-alpine3.20
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest
FROM $BASE_GOLANG_ALPINE as artifact

WORKDIR /build
ADD vault-csi-provider /build
ADD patches /build/patches
RUN git apply -v patches/*.patch
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o vault-csi-provider && \
    chown -R 64535:64535 /build/vault-csi-provider && \
    chmod 0700 /build/vault-csi-provider

FROM $BASE_DISTROLESS
COPY --from=artifact /build/vault-csi-provider /bin

ENTRYPOINT ["/bin/vault-csi-provider"]
