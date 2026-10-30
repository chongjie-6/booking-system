# microservices

Go backend and a React frontend.

## Layout

```
backend/      Go module (`booking-system`), one server
  cmd/          main.go, the server entry point
  internal/     feature code: auth, booking, notification, payment, resource, user, plus router
  pkg/          shared helpers: auth (JWT, passwords), db, rdb (Redis), geoip, middleware, response
  migrations/   SQL migrations, applied on startup
frontend/     React + TypeScript + Vite app
```

## Backend

Requires Go 1.27+ and PostgreSQL 18+ (the migrations use `uuidv7()`). Redis is optional; without it the server logs a warning and skips rate limiting. [air](https://github.com/air-verse/air) is pinned as a Go tool in `go.mod`, so it needs no separate install.

Settings come from environment variables, or from `backend/.env` (gitignored), which is loaded on startup:

| Variable | Required | Value |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string, e.g. `postgres://user:pass@localhost:5432/booking` |
| `JWT_PRIVATE_KEY` | yes | P-256 private key in PEM, for signing access tokens (ES256) |
| `KAFKA_ADDRESS` | yes | Kafka brokers, comma-separated, e.g. `localhost:9092`. Login alerts go through the `notifications` topic |
| `ALLOWED_ORIGINS` | yes | The one frontend origin allowed by CORS, e.g. `http://localhost:5173`; it isn't split on commas |
| `PORT` | no | Port to listen on; default `8080` |
| `GIN_MODE` | no | `debug` (default), `release` or `test` |
| `REDIS_ADDRESS`, `REDIS_PASSWORD` | no | Redis for rate limiting, e.g. `localhost:6379` |
| `GEOIP_DB_PATH` | no | See [IP country database](#ip-country-database) |

Generate a JWT key (Git Bash) with `openssl ecparam -name prime256v1 -genkey -noout`, and paste it into `.env` in double quotes so the line breaks are kept:

```sh
JWT_PRIVATE_KEY="-----BEGIN EC PRIVATE KEY-----
...
-----END EC PRIVATE KEY-----"
```

Run with hot reload (settings in `.air.toml`):

```sh
cd backend
go tool air
```

Run once without hot reload:

```sh
cd backend
go run ./cmd
```

### IP country database

Login alerts fire when a user logs in from a new browser, or from an IP that's on a new network and in a new country for them. The country data comes from the free [DB-IP Lite](https://db-ip.com/db/download/ip-to-country-lite) database, which is gitignored. Download it (Git Bash), and again each month for updates:

```sh
cd backend
mkdir -p data
curl -fsSL "https://download.db-ip.com/free/dbip-country-lite-$(date +%Y-%m).mmdb.gz" | gunzip > data/dbip-country-lite.mmdb.tmp
mv data/dbip-country-lite.mmdb.tmp data/dbip-country-lite.mmdb
```

Stop the server before replacing the file, and start it again afterwards. Set `GEOIP_DB_PATH` to keep the file somewhere else. Without the file, the server logs a warning and only compares IP ranges.

DB-IP Lite is licensed under CC BY 4.0, so the site must show the credit "IP Geolocation by DB-IP" linking to https://db-ip.com.

## Frontend

```sh
cd frontend
npm install
npm run dev      # dev server
npm run build    # type-check and production build
npm run lint
```
