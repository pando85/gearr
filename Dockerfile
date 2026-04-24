# This Dockerfile uses BuildKit cache mounts for efficient ffmpeg builds.
# The ffmpeg compilation is cached between builds unless:
# 1. FFMPEG_BUILD_SCRIPT_VERSION changes
# 2. Build dependencies in apt-get change
# 3. Cache is explicitly invalidated

ARG BASE_IMAGE=ubuntu:26.04

FROM ${BASE_IMAGE} AS ffmpeg-builder

ARG FFMPEG_BUILD_SCRIPT_VERSION=1.58.1
ARG FFMPEG_BUILD_OPTIONS=--enable-gpl-and-non-free

ENV DEBIAN_FRONTEND=noninteractive
ENV CFLAGS="-std=gnu17"

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update \
    && apt-get -y --no-install-recommends install \
        build-essential \
        curl \
        ca-certificates \
        libva-dev \
        zlib1g-dev \
        python3 \
        python-is-python3 \
        ninja-build \
        meson \
        git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /usr/share/doc/* \
    && update-ca-certificates

WORKDIR /build

RUN --mount=type=cache,target=/build/packages,sharing=locked \
    --mount=type=cache,target=/build/workspace,sharing=locked \
    --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    echo "ffmpeg-build-script:${FFMPEG_BUILD_SCRIPT_VERSION}" > /build/.ffmpeg-version && \
    curl -sLO \
    https://raw.githubusercontent.com/markus-perl/ffmpeg-build-script/v${FFMPEG_BUILD_SCRIPT_VERSION}/build-ffmpeg && \
    chmod 755 ./build-ffmpeg && \
    sed -i 's/^CFLAGS="-I/CFLAGS="-std=gnu17 -I/' build-ffmpeg && \
    SKIPINSTALL=yes ./build-ffmpeg \
        --build \
        ${FFMPEG_BUILD_OPTIONS} && \
    mkdir -p /output && \
    cp -a workspace/bin/. /output/

FROM ${BASE_IMAGE} AS base

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update \
    && apt-get install -y \
        ca-certificates \
        mkvtoolnix \
        libva-drm2 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ffmpeg-builder /output/ff* /usr/bin/

RUN ldd /usr/bin/ffmpeg && \
    ldd /usr/bin/ffprobe

FROM base AS server
COPY ./dist/gearr-server /app/gearr-server

ENTRYPOINT ["/app/gearr-server"]

FROM base AS worker
COPY ./dist/gearr-worker /app/gearr-worker

ENTRYPOINT ["/app/gearr-worker"]

FROM tentacule/pgstosrt AS worker-pgs
COPY ./dist/gearr-worker /app/gearr-worker

ENTRYPOINT ["/app/gearr-worker"]