FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder-basis
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/global-pinger

COPY go.mod go.mod
COPY go.sum go.sum

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go mod download

FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder
ARG TARGETOS
ARG TARGETARCH

COPY --from=builder-basis /go/pkg /go/pkg

WORKDIR /app/global-pinger

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/global-pinger-executor ./cmd/executor
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/global-pinger-proxy ./cmd/proxy
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/global-pinger-hub ./cmd/hub

FROM debian:bookworm

COPY --from=builder /app/global-pinger/bin/global-pinger-executor /usr/local/bin/global-pinger-executor
COPY --from=builder /app/global-pinger/bin/global-pinger-proxy /usr/local/bin/global-pinger-proxy
COPY --from=builder /app/global-pinger/bin/global-pinger-hub /usr/local/bin/global-pinger-hub

