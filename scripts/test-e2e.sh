#!/bin/bash
set -e

TOKEN=super-secret-token
JOBS_URL="http://localhost:8080/api/v1/job/"
AUTH_HEADER="Authorization: Bearer ${TOKEN}"
DEBUG_OUTPUT=""

log_debug() {
    DEBUG_OUTPUT="${DEBUG_OUTPUT}$1\n"
}

print_debug_on_failure() {
    if [ "$1" != "0" ]; then
        echo ""
        echo "========== DEBUG OUTPUT =========="
        echo -e "$DEBUG_OUTPUT"
        echo "=================================="
        echo ""
        echo "========== JOB DETAILS =========="
        if [ -n "$job_id" ]; then
            echo "Job ID: $job_id"
            job_details=$(curl -s "${JOBS_URL}${job_id}" --header "${AUTH_HEADER}" 2>/dev/null || echo "Could not fetch job details")
            echo "$job_details" | jq '.' 2>/dev/null || echo "$job_details"
        fi
        echo "=================================="
        echo ""
        echo "========== DATABASE QUEUE STATUS =========="
        echo "=== encode_queue ==="
        docker compose exec postgres psql -U postgres -d gearr -c "SELECT * FROM encode_queue;" 2>/dev/null || echo "Could not query encode_queue"
        echo ""
        echo "=== pgs_queue ==="
        docker compose exec postgres psql -U postgres -d gearr -c "SELECT * FROM pgs_queue;" 2>/dev/null || echo "Could not query pgs_queue"
        echo ""
        echo "=== task_event_queue ==="
        docker compose exec postgres psql -U postgres -d gearr -c "SELECT * FROM task_event_queue ORDER BY event_time DESC LIMIT 20;" 2>/dev/null || echo "Could not query task_event_queue"
        echo ""
        echo "=== job_actions ==="
        docker compose exec postgres psql -U postgres -d gearr -c "SELECT * FROM job_actions ORDER BY created_at DESC LIMIT 20;" 2>/dev/null || echo "Could not query job_actions"
        echo "============================================"
        echo ""
        echo "========== SERVER LOGS (last 50 lines) =========="
        docker compose logs --tail=50 server 2>/dev/null || echo "Could not fetch server logs"
        echo "================================================"
        echo ""
        echo "========== WORKER LOGS (last 50 lines) =========="
        docker compose logs --tail=50 worker 2>/dev/null || echo "Could not fetch worker logs"
        echo "================================================="
        echo ""
        echo "========== WORKER-PGS LOGS (last 30 lines) =========="
        docker compose logs --tail=30 worker-pgs 2>/dev/null || echo "Could not fetch worker-pgs logs"
        echo "===================================================="
        echo ""
        echo "========== POSTGRES LOGS (last 30 lines) =========="
        docker compose logs --tail=30 postgres 2>/dev/null || echo "Could not fetch postgres logs"
        echo "===================================================="
    fi
}

trap 'print_debug_on_failure $?' EXIT

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

log_debug "Creating job with source_path: $url"

response=$(curl -s --location --request POST "${JOBS_URL}" \
  --header "${AUTH_HEADER}" \
  --header 'Content-Type: application/json' \
  --data "$data")

log_debug "Job creation response: $response"
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
  
  job_response=$(curl -s "${JOBS_URL}${job_id}" --header "${AUTH_HEADER}")
  status=$(echo "$job_response" | jq -r '.status')
  log_debug "Attempt $attempt: status=$status"
  log_debug "Response: $job_response"
  
  if [ "${status}" = "completed" ]; then
      echo "OK"
      exit 0
  elif [ "${status}" = "failed" ]; then
      echo "Job failed!"
      log_debug "Job entered failed state"
      echo "$job_response" | jq '.' 2>/dev/null || echo "$job_response"
      exit 1
  else
      echo "Status: $status. Waiting..."
      sleep 1
  fi
done

echo "Job status did not become completed after $MAX_ATTEMPTS attempts. Exiting with KO."
exit 1