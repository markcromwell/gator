#!/usr/bin/env bash
set -euo pipefail

# Run integration tests locally using Docker Postgres.
# Usage: ./scripts/run_integration.sh [--keep] [--port PORT]

KEEP=0
PORT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep) KEEP=1; shift ;;
    --port) PORT="$2"; shift 2 ;;
    -h|--help) echo "Usage: $0 [--keep] [--port PORT]"; exit 0 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }

CONTAINER_NAME="gator-test-$$"

find_free_port() {
  if [[ -n "$PORT" ]]; then
    echo "$PORT"
    return
  fi
  for p in {5432..5442}; do
    if command -v ss >/dev/null 2>&1; then
      if ss -ltn | awk '{print $4}' | grep -qE ":${p}$"; then
        continue
      else
        echo "$p"
        return
      fi
    elif command -v nc >/dev/null 2>&1; then
      if nc -z 127.0.0.1 $p >/dev/null 2>&1; then
        continue
      else
        echo "$p"
        return
      fi
    else
      # fallback assume 5432 free
      echo 5432
      return
    fi
  done
  echo "5432"
}

PORT=$(find_free_port)
echo "Using Postgres port: $PORT"

echo "Starting postgres container ($CONTAINER_NAME)..."
docker run -d --rm --name "$CONTAINER_NAME" -e POSTGRES_PASSWORD=pass -e POSTGRES_USER=gator -e POSTGRES_DB=gator -p ${PORT}:5432 postgres:15 >/dev/null

cleanup() {
  if [[ "$KEEP" -eq 1 ]]; then
    echo "Leaving container running: $CONTAINER_NAME"
    return
  fi
  echo "Stopping container $CONTAINER_NAME..."
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "Waiting for Postgres to be ready..."
for i in {1..60}; do
  if docker exec "$CONTAINER_NAME" pg_isready -U gator >/dev/null 2>&1; then
    echo "Postgres ready"
    break
  fi
  sleep 1
  if [[ $i -eq 60 ]]; then
    echo "Postgres did not become ready" >&2
    exit 1
  fi
done

export GATOR_TEST_DB="postgres://gator:pass@127.0.0.1:${PORT}/gator?sslmode=disable"
echo "GATOR_TEST_DB=$GATOR_TEST_DB"

if command -v sqlc >/dev/null 2>&1; then
  echo "Running sqlc generate..."
  sqlc generate
fi

echo "Running tests (this may take a while)..."
# run full test suite; integration test picks up GATOR_TEST_DB
go test ./...

echo "All tests finished."

if [[ "$KEEP" -eq 1 ]]; then
  echo "Container left running: $CONTAINER_NAME"
else
  echo "Cleaning up container..."
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
fi

exit 0
