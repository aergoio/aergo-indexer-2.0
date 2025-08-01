# syntax=docker/dockerfile:1

FROM golang:alpine3.21 AS builder
RUN apk update && apk upgrade --no-cache && apk add make
WORKDIR /aergo-indexer
COPY go.mod ./
COPY go.sum ./
RUN go mod download
ADD . .
RUN make bin/indexer

FROM alpine:3.21
RUN apk update && apk upgrade --no-cache \
    && apk add libgcc libcrypto3 libssl3
COPY --from=builder /aergo-indexer/bin/* /usr/local/bin/
COPY /usr/local/bin/aergoluac /usr/local/bin/
ENV HOME="/root"
ADD arglog.toml $HOME
CMD ["indexer"]
