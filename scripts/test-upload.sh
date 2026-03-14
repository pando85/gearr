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

# Clean up existing jobs before running
echo "Cleaning up existing jobs..."
existing_jobs=$(curl -s "${JOBS_URL}" --header "${AUTH_HEADER}" | jq -r '.[].id' 2>/dev/null || true)
for job_id in $existing_jobs; do
  curl -s -X DELETE "${JOBS_URL}${job_id}" --header "${AUTH_HEADER}" > /dev/null 2>&1 || true
done
echo "Cleanup complete"

data=$(payload "${url}")

echo "Upload job"

response=$(curl -s --location --request POST "${JOBS_URL}" \
  --header "${AUTH_HEADER}" \
  --header 'Content-Type: application/json' \
  --data "$data")

echo "$response"
echo ''

# Extract job ID from response
job_id=$(echo "$response" | jq -r '.id')

if [ -z "$job_id" ] || [ "$job_id" = "null" ]; then
  echo "Failed to create job"
  exit 1
fi

MAX_ATTEMPTS=50
for attempt in $(seq 1 $MAX_ATTEMPTS); do
  echo "Attempt $attempt to get job status"

  status=$(curl -s "${JOBS_URL}${job_id}" --header "${AUTH_HEADER}" | jq -r '.status')

  if [ "${status}" = "completed" ]; then
      echo "OK"
      exit 0
  else
      echo "Status is not completed yet. Waiting..."
      sleep 1
  fi
done

echo "Job status did not become completed after $MAX_ATTEMPTS attempts. Exiting with KO."
exit 1