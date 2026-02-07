#!/bin/sh
set -eu

WORKDIR="${EOSRIFT_DEPLOY_WORKDIR:-/workspace}"
COMPOSE_FILES="${EOSRIFT_DEPLOY_COMPOSE_FILES:-docker-compose.yml}"
SERVICE="${EOSRIFT_DEPLOY_SERVICE:-server}"
HEALTH_URL="${EOSRIFT_DEPLOY_HEALTH_URL:-http://server:8080/healthz}"
HEALTH_ATTEMPTS="${EOSRIFT_DEPLOY_HEALTH_ATTEMPTS:-30}"
HEALTH_SLEEP_SECONDS="${EOSRIFT_DEPLOY_HEALTH_SLEEP_SECONDS:-2}"

if [ ! -d "$WORKDIR" ]; then
  echo "deploy workdir does not exist: $WORKDIR" >&2
  exit 1
fi

cd "$WORKDIR"

COMPOSE_FLAGS=""
oldifs=$IFS
IFS=:
for file in $COMPOSE_FILES; do
  COMPOSE_FLAGS="$COMPOSE_FLAGS -f $file"
done
IFS=$oldifs

compose() {
  # shellcheck disable=SC2086
  docker compose $COMPOSE_FLAGS "$@"
}

echo "deploy: pulling $SERVICE"
compose pull "$SERVICE"

echo "deploy: recreating $SERVICE"
compose up -d --no-deps --force-recreate "$SERVICE"

attempt=1
while [ "$attempt" -le "$HEALTH_ATTEMPTS" ]; do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    echo "deploy: health check ok"
    exit 0
  fi
  echo "deploy: health check attempt ${attempt}/${HEALTH_ATTEMPTS} failed"
  sleep "$HEALTH_SLEEP_SECONDS"
  attempt=$((attempt + 1))
done

echo "deploy: health check failed after ${HEALTH_ATTEMPTS} attempts" >&2
exit 1
