ARG BASE_GOLANG_ALPINE=registry.deckhouse.io/base_images/golang:1.22.8-alpine
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest

FROM $BASE_GOLANG_ALPINE as artifact

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
