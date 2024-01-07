#!/bin/bash
set -e

DOWNLOAD_URL="https://download4.dvdloc8.com/trailers/divxdigest/simpsons_movie_1080p_hddvd_trailer.zip"
TEMP_DIR=$(mktemp -d)
OUTPUT_DIR=demo-files
OUTPUT_FILE=test.mp4
OUTPUT_PATH="$OUTPUT_DIR/$OUTPUT_FILE"

if [ -e "$OUTPUT_PATH" ]; then
    echo "$OUTPUT_PATH already exists"
    exit 0
fi

echo "Download file"
curl -sL "$DOWNLOAD_URL" -o "$TEMP_DIR/trailer.zip"

echo "Unzip file"
unzip -d "$TEMP_DIR" "$TEMP_DIR/trailer.zip" > /dev/null

if ! [ -d "$OUTPUT_DIR" ]; then
    mkdir "$OUTPUT_DIR"
fi

echo "Move to $OUTPUT_PATH"
mv "$TEMP_DIR/"*.mp4 "$OUTPUT_PATH"
rm -r "$TEMP_DIR"

