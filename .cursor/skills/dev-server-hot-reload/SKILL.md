---
name: dev-server-hot-reload
description: Start Grafana development servers with hot reloading for both backend and frontend. Use when validating UI or UX changes, when the user wants to run the dev server, or when starting development with hot reload.
---

# Development Server with Hot Reloading

## Quick Start

Start both backend and frontend development servers with hot reloading:

1. **Backend**: Run `make run` in one terminal
2. **Frontend**: Run `yarn start:liveReload` in a separate terminal

## Workflow

The backend (`make run`) uses Air for hot reloading and watches for Go file changes. The frontend (`yarn start:liveReload`) runs the webpack dev server with live reload enabled.

Both processes should run simultaneously in separate terminal sessions. The backend typically runs on port 3000, and the frontend dev server proxies to it.

## Notes

- Run these commands as part of full validation for UI, theme, or UX changes (see **Validation requirements** in `AGENTS.md`)
- Also run when the user asks to start the dev server
- Both commands run indefinitely until stopped (Ctrl+C)
- Ensure dependencies are installed (`make deps`) before starting
- Wait for readiness before browser validation: backend should log `HTTP Server Listen`, frontend should log `Compiled successfully`
- Smoke check: `curl -I http://localhost:3000/` should return `HTTP/1.1 302` to `/login`