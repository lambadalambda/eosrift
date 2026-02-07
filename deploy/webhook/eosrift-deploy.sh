#!/bin/sh
set -eu

WORKDIR="${EOSRIFT_DEPLOY_WORKDIR:-/workspace}"
COMPOSE_FILES="${EOSRIFT_DEPLOY_COMPOSE_FILES:-docker-compose.yml}"
SERVICE="${EOSRIFT_DEPLOY_SERVICE:-server}"
HEALTH_URL="${EOSRIFT_DEPLOY_HEALTH_URL:-http://server:8080/healthz}"
HEALTH_ATTEMPTS="${EOSRIFT_DEPLOY_HEALTH_ATTEMPTS:-30}"
HEALTH_SLEEP_SECONDS="${EOSRIFT_DEPLOY_HEALTH_SLEEP_SECONDS:-2}"
STATUS_PATH="${EOSRIFT_DEPLOY_STATUS_PATH:-/data/deploy-status.json}"
RUN_ID="${EOSRIFT_DEPLOY_RUN_ID:-}"
REPOSITORY="${EOSRIFT_DEPLOY_REPOSITORY:-}"
WORKFLOW="${EOSRIFT_DEPLOY_WORKFLOW:-}"
BRANCH="${EOSRIFT_DEPLOY_BRANCH:-}"
SHA="${EOSRIFT_DEPLOY_SHA:-}"
RUN_URL="${EOSRIFT_DEPLOY_RUN_URL:-}"
STARTED_AT="${EOSRIFT_DEPLOY_STARTED_AT:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
FINISHED_AT=""
STAGE="init"

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

json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

write_status() {
  state="$1"
  message="$2"
  now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  if [ -n "$RUN_ID" ]; then
    run_id_json="$RUN_ID"
  else
    run_id_json="null"
  fi

  mkdir -p "$(dirname "$STATUS_PATH")"
  tmp="${STATUS_PATH}.tmp"
  cat > "$tmp" <<EOF
{
  "state": "$(json_escape "$state")",
  "message": "$(json_escape "$message")",
  "updated_at": "$(json_escape "$now")",
  "started_at": "$(json_escape "$STARTED_AT")",
  "finished_at": "$(json_escape "$FINISHED_AT")",
  "run_id": $run_id_json,
  "repository": "$(json_escape "$REPOSITORY")",
  "workflow": "$(json_escape "$WORKFLOW")",
  "branch": "$(json_escape "$BRANCH")",
  "sha": "$(json_escape "$SHA")",
  "run_url": "$(json_escape "$RUN_URL")"
}
EOF
  mv "$tmp" "$STATUS_PATH"
}

on_exit() {
  code=$?
  if [ "$code" -ne 0 ]; then
    FINISHED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    write_status "error" "failed during stage: $STAGE"
  fi
  exit "$code"
}
trap on_exit EXIT

STAGE="pull"
write_status "running" "pulling $SERVICE image"

echo "deploy: pulling $SERVICE"
compose pull "$SERVICE"

STAGE="recreate"
write_status "running" "recreating $SERVICE"

echo "deploy: recreating $SERVICE"
compose up -d --no-deps --force-recreate "$SERVICE"

STAGE="health_check"
write_status "running" "waiting for health check"

attempt=1
while [ "$attempt" -le "$HEALTH_ATTEMPTS" ]; do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    echo "deploy: health check ok"
    FINISHED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    write_status "success" "deploy completed"
    exit 0
  fi
  echo "deploy: health check attempt ${attempt}/${HEALTH_ATTEMPTS} failed"
  sleep "$HEALTH_SLEEP_SECONDS"
  attempt=$((attempt + 1))
done

echo "deploy: health check failed after ${HEALTH_ATTEMPTS} attempts" >&2
exit 1
