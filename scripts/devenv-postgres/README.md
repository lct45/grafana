# Postgres local development

Use Postgres instead of the default SQLite dev database.

## Quick start

```bash
make devenv-postgres-dev   # start Postgres (devenv/docker/blocks/postgres)
make run-postgres          # run Grafana with GF_DATABASE_* env vars
```

On first run, `conf/custom.ini` is created from `conf/custom.ini.postgres.example` if it does not exist.

## Manual configuration

Environment variables (defaults match the devenv Postgres block):

| Variable | Default |
| -------- | ------- |
| `GF_DATABASE_TYPE` | `postgres` |
| `GF_DATABASE_HOST` | `127.0.0.1:5432` |
| `GF_DATABASE_NAME` | `grafana` |
| `GF_DATABASE_USER` | `grafana` |
| `GF_DATABASE_PASSWORD` | `password` |
| `GF_DATABASE_SSL_MODE` | `disable` |

Or copy `conf/custom.ini.postgres.example` to `conf/custom.ini`.

## Verification

```bash
make verify-postgres-dev
```

This starts Postgres, builds Grafana, runs a smoke test against `/api/health` and `/login`, then exits.
