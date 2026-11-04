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

Requires Go 1.27+ and PostgreSQL 18+ (the migrations use `uuidv7()`). Redis is optional; without it the server logs a warning and skips rate limiting and seat holds (see [Design decisions](#design-decisions)). [air](https://github.com/air-verse/air) is pinned as a Go tool in `go.mod`, so it needs no separate install.

Settings come from environment variables, or from `backend/.env` (gitignored), which is loaded on startup:

| Variable | Required | Value |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string, e.g. `postgres://user:pass@localhost:5432/booking` |
| `JWT_PRIVATE_KEY` | yes | P-256 private key in PEM, for signing access tokens (ES256) |
| `KAFKA_ADDRESS` | yes | Kafka brokers, comma-separated, e.g. `localhost:9092`. Login alerts go through the `notifications` topic |
| `ALLOWED_ORIGINS` | yes | The one frontend origin allowed by CORS, e.g. `http://localhost:5173`; it isn't split on commas. Password reset emails link here |
| `PORT` | no | Port to listen on; default `8080` |
| `REDIS_ADDRESS`, `REDIS_PASSWORD` | no | Redis for rate limiting, e.g. `localhost:6379` |
| `GEOIP_DB_PATH` | no | See [IP country database](#ip-country-database) |
| `RESEND_API_KEY`, `EMAIL_FROM` | no | [Resend](https://resend.com) key and sender for login alerts and password reset emails; without them no emails are sent |

`docker-compose.yml` runs Postgres 18, Redis and Kafka locally; its header lists the matching settings. If you already run one of them, start only the others, e.g. `docker compose up -d redis kafka`:

```sh
docker compose up -d
```

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

### Design decisions

**Seat holds live in Redis, and booking keeps working when Redis is down.** Choosing seats holds them in Redis with a TTL, one key per seat, while the user pays. A `bookings` row is only written once payment succeeds, so every row is a paid booking. If Redis is unreachable, booking skips the hold instead of failing: two users can then both reach checkout and pay for the same seat. Postgres still allows only one of them to keep it (`booking_seats` is unique on `(event_id, seat_id)`), so the second payment's insert fails and that payment is refunded. We accept the occasional refund during a Redis outage over having booking go down with Redis.

**Login and reset emails have their own rate limit.** On top of the app-wide limit, `POST /auth/login` and `POST /auth/password/forgot` allow 5 tries per IP, refilled one a minute. **Reset tokens are made when the email is sent**, in the notification consumer, so the raw token never sits in the outbox or Kafka. A link works once, for an hour, and using it signs out every session.

## Frontend

In dev, Vite forwards `/auth` and `/user` to the backend on `localhost:8080` (see `vite.config.ts`), so the page and API share an origin and the auth cookies need no CORS. Add each new API path prefix there. `src/api.ts` wraps `fetch`: when the 15-minute access token has expired it calls `/auth/refresh` once and retries.

```sh
cd frontend
npm install
npm run dev      # dev server
npm run build    # type-check and production build
npm run lint
```
