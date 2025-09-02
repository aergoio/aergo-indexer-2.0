# syntax=docker/dockerfile:1

FROM golang:alpine3.21 AS builder
RUN apk update && apk upgrade --no-cache && apk add make
WORKDIR /aergo-indexer
COPY go.mod ./
COPY go.sum ./
RUN go mod download
ADD . .
RUN make clean
RUN make bin/indexer
# build aergoluac on the same build image
RUN apk add git cmake build-base m4
RUN git clone --branch develop --recursive https://github.com/aergoio/aergo.git \
    && cd aergo \
    && make aergoluac \
    && cp bin/aergoluac ../bin/
# run the unit tests to make sure everything is working
RUN cp bin/aergoluac /usr/local/bin/ && make unit-test

FROM alpine:3.21
RUN apk update && apk upgrade --no-cache \
    && apk add libgcc libcrypto3 libssl3
COPY --from=builder /aergo-indexer/bin/* /usr/local/bin/
COPY --from=builder /usr/local/bin/aergoluac /usr/local/bin/
ENV HOME="/root"
ADD arglog.toml $HOME
WORKDIR $HOME
CMD ["indexer"]
