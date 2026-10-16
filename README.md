# microservices

Go backend services and a React frontend.

## Layout

```
backend/    Go module (`microservices`)
  cmd/        one main package per service: booking, notification, payment, resource, user
  internal/   service code, plus shared types in internal/domain
frontend/   React + TypeScript + Vite app
```

## Backend

Requires Go 1.27+. [air](https://github.com/air-verse/air) is pinned as a Go tool in `go.mod`, so it needs no separate install.

Run every service with hot reload: in VS Code, run the `dev` task (**Terminal → Run Task → dev**). Each service opens in its own terminal.

Run one service with hot reload:

```sh
cd backend
go tool air --build.cmd 'go build -o ./tmp/booking.exe ./cmd/booking' --build.entrypoint './tmp/booking.exe'
```

Run once without hot reload:

```sh
cd backend
go run ./cmd/booking
```

## Frontend

```sh
cd frontend
npm install
npm run dev      # dev server
npm run build    # type-check and production build
npm run lint
```
