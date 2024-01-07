FROM golang:1.14.3-alpine3.11 as builder
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/transcoder/
COPY . .
RUN go run build.go build server
RUN cp ./dist/transcoderd-linux-amd64 /bin/transcoderd && chmod +x /bin/transcoderd

FROM scratch
COPY --from=builder /bin/transcoderd /bin/transcoderd

ENTRYPOINT ["/bin/transcoderd"]
