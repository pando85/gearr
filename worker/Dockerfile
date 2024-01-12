FROM golang:1.19.4 as builder

RUN apt-get update && \
    apt-get install -y \
        gcc \
        git \
        libgtk-3-dev \
        libappindicator3-dev \
        && rm -rf /var/lib/apt/lists/*

WORKDIR $GOPATH/src/transcoder/
COPY . .
RUN go run build.go build worker && \
    mv ./dist/transcoderw-linux-amd64 /transcoderw

FROM alpine:3.14 as worker-base
RUN apk add curl && \
    curl -sSLo /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    curl -sSLO https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.31-r0/glibc-2.31-r0.apk && \
    apk add glibc-2.31-r0.apk && \
    rm glibc-2.31-r0.apk

COPY --from=builder /transcoderw /app/transcoderw

ENTRYPOINT ["/app/transcoderw"]

FROM tentacule/pgstosrt as pgs
COPY --from=builder /transcoderw /app/transcoderw

ENTRYPOINT ["/app/transcoderw"]
