FROM golang:1.14.3 as builder
WORKDIR $GOPATH/src/transcoder/
COPY . .
RUN apt-get update && apt-get install -y gcc git libgtk-3-dev libappindicator3-dev
RUN go run  build.go build worker -p linux-amd64 -m console
RUN cp ./dist/transcoderw-console-linux-amd64 /bin/transcoderw && chmod +x /bin/transcoderw

FROM tentacule/pgstosrt
COPY --from=builder /bin/transcoderw /app/transcoderw

ENTRYPOINT ["/app/transcoderw"]