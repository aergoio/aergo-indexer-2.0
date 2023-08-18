# syntax=docker/dockerfile:1

FROM golang:1.19-alpine AS builder
RUN apk update && apk upgrade --no-cache && apk add make
WORKDIR /aergo-indexer

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD . .
RUN make bin/indexer

FROM alpine
RUN apk add libgcc
COPY --from=builder /aergo-indexer/bin/* /usr/local/bin/
ADD arglog.toml $HOME
CMD ["indexer"]
