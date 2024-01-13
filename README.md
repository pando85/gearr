# Transcoderd

## Container images

- server: `ghcr.io/pando85/transcoder:latest-server`
- worker: `ghcr.io/pando85/transcoder:latest-worker`
- PGS worker: `ghcr.io/pando85/transcoder:latest-worker-pgs`

## Config

Example in `config.example.yaml`.

## Client execution

```
DIR=/tmp/images/encode

mkdir -p $DIR
docker run -it -d --restart unless-stopped --cpuset-cpus 16-32 \
    --name transcoder-worker --hostname $(hostname) \
    -v $DIR:/tmp/ pando85/ghcr.io/pando85/transcoder:latest-worker \
    --broker.host transcoder.example.com \
    --worker.priority 9
```

**Warning:** PGS agent is also needed if PGS are detected. And it needs to run before they are detected to create the RabbitMQ queue.

```
DIR=/tmp/images/pgs

mkdir -p $DIR
docker run -it -d --restart unless-stopped --cpuset-cpus 1-2 \
    --name transcoder-worker-pgs --hostname $(hostname) \
    -v $DIR:/tmp/ ghcr.io/pando85/transcoder:latest-worker-pgs \
    --broker.host transcoder.example.com \
    --worker.priority 9
```


## Add movies from Radarr

```bash
go run ./radarr/main.go --api-key XXXXXX --url https://radarr.example.com --movies 5 --transcoder-url 'https://transcorder.example.com' --transcoder-token XXXXXX
```
