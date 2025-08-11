# Logship Syslog Filter

A lightweight, containerized syslog filter and forwarder with a modern UI.
Place this between your log sources and your destination SIEM/syslog server to whitelist traffic using scalable rules.

## Features
- Receive syslog over UDP/TCP (default listen port: **5514** on the host, mapped to container 514).
- Whitelist rules by source IP/CIDR, hostname, app-name, facility, severity, and message regex.
- Default action is **Block** (configurable).
- Forward to a user-configured destination syslog server when rules match.
- Forwarding threshold (daily/weekly/30d or custom) to halt forwarding when exceeded.
- Dashboard with:
  - Recent 25 forwarded, dropped, unmatched, and received logs.
  - 24-hour graphs for received/processed.
- Settings pages for Rules, Default Action, Destination, Storage, Thresholds.
- Click-through from recent log to matching rule (and create-from-sample for dropped/unmatched).

## Quick Start (Docker Compose)
1. Copy `.env.example` to `.env` and adjust if needed.
2. Ensure host port **5514** (UDP/TCP) is open and forwarded to the container (unprivileged host port avoids root requirements).
3. Build and run:
   ```bash
   docker compose up -d --build
   ```
4. Open the UI at http://localhost:8080

## Ports
- UI: http://localhost:8080
- Backend API: http://localhost:8000 (internal network)
- Syslog listener: host ports 5514/udp and 5514/tcp -> container 514

> If you need to use privileged port 514 on the host, update `docker-compose.yml` port mappings accordingly.

## Persistence
- SQLite DB and log cache stored in the `data` volume.

## GitHub Actions: Build & Deploy
This repo includes `.github/workflows/deploy.yml` which:
- Builds and pushes `backend` and `frontend` images to GHCR (ghcr.io).
- Optionally deploys to a remote Linux host via SSH and `docker compose` if you set secrets:
  - `CR_PAT` (a GitHub Personal Access Token with `packages:write`)
  - `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY` (base64-encoded private key or raw OpenSSH key)
  - `DEPLOY_COMPOSE_PATH` (remote absolute path where the compose file lives)
  - `DEPLOY_ENV` (optional, remote `.env` path)

> The workflow is safe to run even without SSH deploy secrets; it will build/push only.

## Development
- Backend (FastAPI): `uvicorn app.main:app --reload --host 0.0.0.0 --port 8000`
- Frontend (Vite React): `npm run dev` in `frontend`

## License
MIT
