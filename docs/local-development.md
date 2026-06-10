# Local Development

Use the local compose file for development. It is already configured for:

- frontend: `http://localhost:3000`
- backend API: `http://localhost:8080`
- backend CORS origin: `http://localhost:3000`

## Full Docker start

```powershell
copy .env.local.example .env
docker compose up --build
```

Open:

- `http://localhost:3000`
- `http://localhost:8080/health/ready`

Do not use `docker-compose.prod.yml` for normal local development. Production compose expects a real domain and Caddy HTTPS.

## If Docker cannot fetch an anonymous token

An error like this means Docker could not reach Docker Hub to download a base image:

```text
failed to fetch anonymous token
```

This is not caused by NetQuest API URLs. The local URLs are still `localhost`; Docker just cannot download images at that moment.

Try one of these options:

```powershell
docker logout
docker login
docker pull golang:1.25-alpine
docker pull node:22-alpine
docker compose up --build
```

If Docker Hub is still unavailable, run only infrastructure in Docker and start the app on the host:

```powershell
docker compose up -d postgres redis nats
```

Backend:

```powershell
Get-Content .env.local.example | Where-Object { $_ -and $_ -notmatch "^\s*#" } | ForEach-Object { $name, $value = $_ -split "=", 2; Set-Item -Path "env:$name" -Value $value }
cd backend
go run ./cmd/migrate up
go run ./cmd/api
```

Frontend in a second terminal:

```powershell
cd frontend
$env:NEXT_PUBLIC_API_BASE_URL="http://localhost:8080"
npm install
npm run dev
```

Then open `http://localhost:3000`.
