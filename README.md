# SlateSSH

SlateSSH is a lightweight web SSH and SFTP workspace rebuilt in Go.

## Structure
- `backend/` Go server, API, WebSocket SSH/SFTP, status polling
- `frontend/` static frontend served by the Go backend
- `Dockerfile` single-image deployment
- `docker-compose.yml` local container deployment

## Run locally
```bash
cd backend
go mod tidy
go run ./cmd/slatessh
```

Then open `http://localhost:3210`.

## Docker
```bash
docker compose up --build
```
