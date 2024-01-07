#!/bin/bash

payload() {
  cat <<EOF
{
  "SourcePath": "$1"
}
EOF
}

if [ -n "$1" ]; then
  url="$1"
else
  url='test.mp4'
fi

data=$(payload "${url}")
echo "$data"
curl --location --request POST 'http://localhost:8080/api/v1/job/?token=super-secret-token' \
  --header 'Content-Type: text/plain' \
  --data "$data" -vvv
