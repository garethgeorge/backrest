#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="${SCRIPT_DIR}"

SERVICE="postgres"
DB_USER="keycloak"
DB_NAME="keycloak"
OUT_FILE="${COMPOSE_DIR}/keycloak_dump.sql"

cd "${COMPOSE_DIR}"

if [ -z "$(docker compose ps -q "${SERVICE}" 2>/dev/null)" ]; then
  echo "error: compose service '${SERVICE}' is not running. Start it with 'docker compose up -d ${SERVICE} keycloak' first." >&2
  exit 1
fi

echo "Dumping '${DB_NAME}' database from service '${SERVICE}'..."

TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

docker compose exec -T "${SERVICE}" \
  pg_dump --username "${DB_USER}" --no-password "${DB_NAME}" > "${TMP_FILE}"

if [ ! -s "${TMP_FILE}" ]; then
  echo "error: pg_dump produced an empty file; aborting without overwriting ${OUT_FILE}." >&2
  exit 1
fi

mv "${TMP_FILE}" "${OUT_FILE}"
trap - EXIT

echo "Wrote $(wc -l < "${OUT_FILE}") lines to ${OUT_FILE}"
