# microservices

Go backend and a React frontend.

## Layout

```
backend/      Go module (`booking-system`), one server
  cmd/          main.go, the server entry point
  internal/     feature code: admin, auth, booking (with payments), notification, outbox, resource, user, plus router
  pkg/          shared helpers: auth (JWT, passwords), db, rdb (Redis), geoip, middleware, response, stripe
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
| `TMDB_TOKEN` | no | [TMDB](https://www.themoviedb.org/settings/api) API read access token. Within a minute of seeding, films with a TMDB ID get TMDB's synopsis, runtime, poster, backdrop and YouTube trailer; without it they keep the seed's text and a drawn poster. TMDB's API is free for non-commercial use, and its terms require the credit in the page footer |
| `TRUST_PROXY` | no | `true` to take the client IP (for rate limits and login alerts) from `X-Real-IP`. Set it only when every request comes through a proxy that overwrites that header, like the frontend image's nginx; otherwise clients could forge it |
| `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY` | for booking | Stripe API keys (`sk_test_...`, `pk_test_...`). Without them checkout says payments aren't available. See [Payments](#payments) |
| `STRIPE_WEBHOOK_SECRET` | for booking | Signing secret (`whsec_...`) of the webhook that sends `payment_intent.succeeded` to `/payments/webhook` |
| `RESEND_API_KEY`, `EMAIL_FROM` | no | [Resend](https://resend.com) key and sender for login alerts and password reset emails; without them no emails are sent |

`docker-compose.yml` runs Postgres 18, Redis, Kafka and Kafka Connect (Debezium) locally; its header lists the matching settings. If you already run one of them, start only the others, e.g. `docker compose up -d redis kafka connect connect-init`.

```sh
docker compose up -d
```

On every `up`, the one-off `connect-init` service registers the outbox connector (`debezium/outbox.json`), which sends outbox rows to Kafka. It keeps retrying until the backend's migrations have made the `outbox` table, then exits; `docker compose logs connect-init` shows where it's up to.

Postgres needs `wal_level=logical` for Debezium, which the compose file sets.

For films to book, load the dev seed once the migrations have run: 5 films, 3 screens and a week of showings from today. It wipes all movies, rooms, showings and bookings first, so re-run it to move the showings forward (Git Bash):

```sh
docker compose exec -T postgres psql -U booking -d booking < backend/seed.sql
```

PowerShell has no `<` redirection, so pipe the file in instead:

```powershell
Get-Content backend/seed.sql -Raw | docker compose exec -T postgres psql -U booking -d booking
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

Tests that need Postgres (e.g. listing bookings) skip unless `DATABASE_URL` is set; CI runs them against Postgres 18. They work in a transaction that's rolled back, but point them at a dev database anyway:

```sh
cd backend
DATABASE_URL=postgres://booking:booking@localhost:5432/booking go test ./...
```

### Docker

`backend/Dockerfile` and `frontend/Dockerfile` build the server and an nginx image that serves the app and forwards API paths to the backend. To run the whole stack, put `JWT_PRIVATE_KEY` (and any optional keys) in `backend/.env`, then:

```sh
docker compose --profile app up -d --build
```

and open http://localhost:8081. Compose points the backend at its own Postgres, Redis and Kafka, so the connection settings in `.env` are overridden. It waits for Postgres to be healthy before starting the backend, and restarts the backend if it exits. Load the seed as above.

### Payments

Checkout takes card payments through [Stripe](https://stripe.com), in test mode during development (free, no real money). Put the test keys from the Stripe dashboard in `backend/.env`. The checkout page gets the publishable key from the backend, so the frontend needs no setting. To receive webhooks locally, run the [Stripe CLI](https://stripe.com/docs/stripe-cli), which prints the `STRIPE_WEBHOOK_SECRET` to use:

```sh
stripe listen --events payment_intent.succeeded --forward-to localhost:8080/payments/webhook   # or localhost:8081 for the Docker stack
```

Pay with the test card `4242 4242 4242 4242`, any future expiry and any CVC. In production, add the webhook endpoint in the dashboard instead.

### Staff

Staff manage films and showings at `/admin` (an **Admin** link appears in the header). Make an account staff with SQL; the role reaches its token at the next sign in or refresh, within 15 minutes:

```sh
docker compose exec postgres psql -U booking -d booking -c "UPDATE users SET user_role = 'admin' WHERE email = 'you@example.com'"
```

Screens and their seats are still set up with SQL, as in `backend/seed.sql`. A showing can't start while its screen is showing another film, and once a showing has sold seats it can't be changed or deleted.

### IP country database

Login alerts fire when a user logs in from a new browser, or from an IP that's on a new network and in a new country for them. The country data comes from the free [DB-IP Lite](https://db-ip.com/db/download/ip-to-country-lite) database, which is gitignored. Download it (Git Bash), and again each month for updates:

```sh
cd backend
mkdir -p data
curl -fsSL "https://download.db-ip.com/free/dbip-country-lite-$(date +%Y-%m).mmdb.gz" | gunzip > data/dbip-country-lite.mmdb.tmp
mv data/dbip-country-lite.mmdb.tmp data/dbip-country-lite.mmdb
```

Stop the server before replacing the file, and start it again afterwards. Set `GEOIP_DB_PATH` to keep the file somewhere else. Without the file, the server logs a warning and only compares IP ranges.

### Design decisions

**Seat holds live in Redis, and booking keeps working when Redis is down.** Reaching checkout holds the chosen seats in Redis for 10 minutes, one key per seat, while the user pays (`PUT /events/{id}/hold`; going back releases them with `DELETE`). Seats someone else holds show as taken on the seat map. A `bookings` row is only written once payment succeeds, so every row is a paid booking. If Redis is unreachable, booking skips the hold instead of failing: two users can then both reach checkout and pay for the same seat. Postgres still allows only one of them to keep it (`booking_seats` is unique on `(event_id, seat_id)`), so the second payment's insert fails and that payment is refunded. We accept the occasional refund during a Redis outage over having booking go down with Redis.

**A payment is booked by whichever arrives first: the checkout page or Stripe's webhook.** `POST /events/{id}/payment` starts a Stripe PaymentIntent for exactly the chosen seats, which go in its metadata. Once Stripe takes the payment, the page calls `POST /events/{id}/bookings`, and Stripe also sends `payment_intent.succeeded`, so a buyer who closes the tab straight after paying still gets their booking. Both look the payment up at Stripe and make the booking from its metadata, and `bookings.payment_intent_id` is unique, so the second one finds the booking already made. If the booking can't be made (a seat sold meanwhile, the showing started, the price changed), the payment is refunded, and a refunded payment is never booked later.

**Bookings can be cancelled up to an hour before the showing, for a full refund.** `DELETE /bookings/{id}` locks the booking, refunds it at Stripe, then deletes it, which frees its seats; if the refund fails, the booking stays.

**Login and reset emails have their own rate limit.** On top of the app-wide limit, `POST /auth/login` and `POST /auth/password/forgot` allow 5 tries per IP, refilled one a minute. **Reset tokens are made when the email is sent**, in the notification consumer, so the raw token never sits in the outbox or Kafka. A link works once, for an hour, and using it signs out every session.

## Frontend

In dev, Vite forwards `/auth`, `/user`, `/movies`, `/events`, `/bookings`, `/payments` and `/admin` to the backend on `localhost:8080` (see `vite.config.ts`), so the page and API share an origin and the auth cookies need no CORS. Add each new API path prefix there. `src/lib/api.ts` wraps `fetch`: when the 15-minute access token has expired it calls `/auth/refresh` once and retries.

```sh
cd frontend
npm install
npm run dev      # dev server
npm run build    # type-check and production build
npm run lint
npm test         # vitest
```

On Windows, run the tests from a path with an upper-case drive letter (`C:\...`, as PowerShell gives). Vitest loads itself twice from a lower-case `c:\...` path, as some editor terminals give, and every test file then fails in `describe`.

Signed-in users reach `/account` from their name in the header: their bookings (`GET /bookings`), which they can cancel up to an hour before (`DELETE /bookings/{id}`), their details (`PATCH /user`) and a password change (`POST /auth/password`). The confirmation page at `/bookings/{ref}` fetches the booking (`GET /bookings/{ref}`) when it's opened from a link rather than straight after booking.
