# syntax=docker/dockerfile:1

FROM golang:1.17-alpine AS builder
RUN apk update && apk add git glide build-base
WORKDIR /aergo-indexer

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD . .
RUN make bin/indexer

FROM alpine
RUN apk add libgcc
COPY --from=builder /app/bin/* /usr/local/bin/
CMD ["indexer"]
