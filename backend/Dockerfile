FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
# Builds only ./cmd, so the air dev tool's modules aren't downloaded. Migrations and email templates are embedded.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
# Mount the IP country database at /app/data/dbip-country-lite.mmdb, or set GEOIP_DB_PATH.
WORKDIR /app
COPY --from=build /server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
