# syntax=docker/dockerfile:1

# --- Build the Go server binary ---
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=docker
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION}" \
    -o /out/server ./cmd/server

# --- Pull the Atlas CLI from its official image ---
FROM arigaio/atlas:latest AS atlas

# --- Final runtime image ---
FROM alpine:3.20

# ca-certificates: TLS to the Postgres host. If atlas ever fails to exec here,
# add "libc6-compat" to this line.
RUN apk add --no-cache ca-certificates

COPY --from=atlas /atlas                     /usr/local/bin/atlas
COPY --from=build /out/server                /usr/local/bin/server
COPY internal/platform/migration/migrations  /migrations
COPY scripts/docker-entrypoint.sh            /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Run as a non-root user.
RUN adduser -D -H app
USER app

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
