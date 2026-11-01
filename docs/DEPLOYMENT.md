# Deployment Guide

## Build

```powershell
go mod tidy
go test ./...
go vet ./...
go build -trimpath -ldflags="-s -w" -o bin/secure-api-gateway.exe ./cmd/api
```

## Run

```powershell
.\bin\secure-api-gateway.exe
```

## Health checks

```powershell
Invoke-RestMethod http://127.0.0.1:3000/health/live
Invoke-RestMethod http://127.0.0.1:3000/health/ready
```

## Notes

- Use a separate `.env` per environment.
- Do not commit production secrets.
- Run database migrations before deploying.
- Keep PostgreSQL and Redis private.
- Configure trusted proxy settings before exposing through Nginx/Caddy.