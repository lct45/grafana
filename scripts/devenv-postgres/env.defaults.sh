#!/usr/bin/env bash
# Default Postgres connection settings for local Grafana development.
# Matches devenv/docker/blocks/postgres/docker-compose.yaml.

export GF_DATABASE_TYPE="${GF_DATABASE_TYPE:-postgres}"
export GF_DATABASE_HOST="${GF_DATABASE_HOST:-127.0.0.1:5432}"
export GF_DATABASE_NAME="${GF_DATABASE_NAME:-grafana}"
export GF_DATABASE_USER="${GF_DATABASE_USER:-grafana}"
export GF_DATABASE_PASSWORD="${GF_DATABASE_PASSWORD:-password}"
export GF_DATABASE_SSL_MODE="${GF_DATABASE_SSL_MODE:-disable}"
