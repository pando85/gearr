#!/bin/bash
set -e

DOWNLOAD_URL="https://d2qguwbxlx1sbt.cloudfront.net/TextInMotion-VideoSample-720p.mp4"
OUTPUT_DIR=demo-files
OUTPUT_FILE=test.mp4
OUTPUT_PATH="$OUTPUT_DIR/$OUTPUT_FILE"

if [ -e "$OUTPUT_PATH" ]; then
    echo "$OUTPUT_PATH already exists"
    exit 0
fi

if ! [ -d "$OUTPUT_DIR" ]; then
    mkdir "$OUTPUT_DIR"
fi

echo "Download file"
curl -sL "$DOWNLOAD_URL" -o "$OUTPUT_PATH"

