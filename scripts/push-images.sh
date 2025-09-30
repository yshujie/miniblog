#!/usr/bin/env bash
set -euo pipefail

if [[ -n "${PIPELINE_ENV_FILE:-}" && -f "${PIPELINE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${PIPELINE_ENV_FILE}"
  set +a
fi

images=()

if [[ "${RUN_FRONTEND_BUILD:-false}" == "true" ]]; then
  [[ -n "${FRONTEND_BLOG_IMAGE_TAG:-}" ]] && images+=("${FRONTEND_BLOG_IMAGE_TAG}")
  [[ -n "${FRONTEND_ADMIN_IMAGE_TAG:-}" ]] && images+=("${FRONTEND_ADMIN_IMAGE_TAG}")
fi

if [[ "${RUN_BACKEND_BUILD:-false}" == "true" ]]; then
  [[ -n "${BACKEND_IMAGE_TAG:-}" ]] && images+=("${BACKEND_IMAGE_TAG}")
fi

if [[ ${#images[@]} -eq 0 ]]; then
  echo "Images were not built in this run, skipping push."
  exit 0
fi

for image in "${images[@]}"; do
  echo "Pushing image ${image}"
  IMAGE_NAME="${image}" make docker-push-image
done
