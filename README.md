# Transcoderd

## Container images

- server: `segator/transcoderd:master`
- agent: `segator/encoder-agent:master`
- PGs agent: `segator/pgs-agent:master`

## Config

Example in `config.example.yaml`.

## Client execution

```
EXTRA_PARAMS="--broker.host transcoder.example.com --worker.priority 9"

DIR=/images/encode2

mkdir -p $DIR
docker run -it -d --restart unless-stopped --cpuset-cpus 16-32 \
    --name encode-video2 --hostname $(hostname) \
    -v $DIR:/tmp/ segator/encoder-agent:master $EXTRA_PARAMS
```
