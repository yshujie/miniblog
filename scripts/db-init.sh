#!/usr/bin/env bash
set -euo pipefail

: "${DB_ROOT_PASSWORD:=}"

export DB_ROOT_PASSWORD

make db-init
