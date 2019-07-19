# Build Geth in a stock Go builder container
FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /gttc
RUN cd /gttc && make gttc

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /gttc/build/bin/gttc /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gttc"]
