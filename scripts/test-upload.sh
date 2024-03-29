#!/bin/bash
set -e

TOKEN=super-secret-token
JOBS_URL="http://localhost:8080/api/v1/job/"
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

payload() {
  cat <<EOF
{
  "source_path": "$1"
}
EOF
}

if [ -n "$1" ]; then
  url="$1"
else
  url='test.mp4'
fi

data=$(payload "${url}")

echo "Upload job"

curl -s --location --request POST "${JOBS_URL}" \
  --header "${AUTH_HEADER}" \
  --header 'Content-Type: text/plain' \
  --data "$data"

echo -e '\n'

MAX_ATTEMPTS=50
for attempt in $(seq 1 $MAX_ATTEMPTS); do
  echo "Attempt $attempt to get job status"

  id=$(curl -s "${JOBS_URL}" --header "${AUTH_HEADER}" | jq -r '.[0].id')
  status=$(curl -s "${JOBS_URL}${id}" --header "${AUTH_HEADER}" | jq -r '.status')

  if [ "${status}" = "completed" ]; then
      echo "OK"
      exit 0
  else
      echo "Status is not completed yet. Waiting..."
      sleep 1
  fi
done

echo "Job status did not become completed after 10 attempts. Exiting with KO."
exit 1
