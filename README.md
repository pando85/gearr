# Transcoder

Transcoder is a program designed to operate on a server with two distinct types of agents for video transcoding tasks, specifically converting a video library to the x265 format using ffmpeg. The following information provides details on how to use and configure the Transcoder system.

## Container Images

- **Server:** `ghcr.io/pando85/transcoder:latest-server`
- **Worker:** `ghcr.io/pando85/transcoder:latest-worker`
- **PGS Worker:** `ghcr.io/pando85/transcoder:latest-worker-pgs`

## Configuration

Refer to the `config.example.yaml` file for a configuration example.

## Client Execution

### Worker

```bash
DIR=/data/images/encode

mkdir -p $DIR
docker run -it -d --restart unless-stopped --cpuset-cpus 16-32 \
    --name transcoder-worker --hostname $(hostname) \
    -v $DIR:/tmp/ ghcr.io/pando85/transcoder:latest-worker \
    --broker.host transcoder.example.com \
    --worker.priority 9
```

**Note:** Adjust the `--cpuset-cpus` and other parameters according to your system specifications.

### PGS Worker

```bash
DIR=/data/images/pgs

mkdir -p $DIR
docker run -it -d --restart unless-stopped \
    --name transcoder-worker-pgs --hostname $(hostname) \
    -v $DIR:/tmp/ ghcr.io/pando85/transcoder:latest-worker-pgs \
    --broker.host transcoder.example.com \
    --worker.priority 9
```

**Warning:** The PGS agent must be started in advance if PGS is detected. It should run before detection to create the RabbitMQ queue.

## Add Movies from Radarr

```bash
go run ./radarr/main.go --api-key XXXXXX --url https://radarr.example.com --movies 5 --transcoder-url 'https://transcoder.example.com' --transcoder-token XXXXXX
```

Feel free to customize the parameters based on your Radarr and Transcoder setup.
