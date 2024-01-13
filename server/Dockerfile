FROM golang:1.21.6 as builder
WORKDIR $GOPATH/src/transcoder/
COPY . .
RUN go run build.go build server && \
    mv ./dist/transcoderd-linux-amd64 /transcoderd

FROM alpine:3.14
COPY --from=builder /transcoderd /bin/transcoderd

ENTRYPOINT ["/bin/transcoderd"]
