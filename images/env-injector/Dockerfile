ARG BASE_GOLANG_22_ALPINE=registry.deckhouse.io/base_images/golang:1.22.3-alpine
ARG BASE_DISTROLESS=registry.deckhouse.io/base_images/distroless:latest

FROM $BASE_GOLANG_22_ALPINE as artifact

WORKDIR /src/env-injector

ARG GOPROXY

COPY go.* ./
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/env-injector . && \
    chown -R 64535:64535 /bin/env-injector && \
    chmod 0555 /bin/env-injector


FROM $BASE_DISTROLESS
COPY --from=artifact /bin/env-injector /bin/env-injector
