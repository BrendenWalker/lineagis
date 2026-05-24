#!/bin/sh
set -eu

mc alias set local "http://${MINIO_HOST:-minio}:9000" "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}"
mc mb "local/${MINIO_REGISTRY_BUCKET:-registry}" --ignore-existing
