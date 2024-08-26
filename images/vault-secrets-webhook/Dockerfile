ARG BASE_GOLANG_22_ALPINE=registry.deckhouse.io/base_images/golang:1.22.3-alpine@sha256:dbf216b880b802c22e3f4f2ef0a78396b4a9a6983cb9b767c5efc351ebf946b0
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest@sha256:b61c8490b9de65167184c1c39b5125373b111929837aeff7184fe65ebdaee0c8

FROM $BASE_GOLANG_22_ALPINE as artifact

WORKDIR /usr/local/src/vault-secrets-webhook

ARG GOPROXY

COPY go.* ./
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/vault-secrets-webhook . && \
    chown -R 64535:64535 /usr/local/bin/vault-secrets-webhook && \
    chmod 0700 /usr/local/bin/vault-secrets-webhook


FROM $BASE_DISTROLESS

COPY --from=artifact /usr/local/bin/vault-secrets-webhook /usr/local/bin/vault-secrets-webhook
ENTRYPOINT ["vault-secrets-webhook"]
