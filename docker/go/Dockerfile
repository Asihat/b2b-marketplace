# ---- Build stage -----------------------------------------------------------
FROM golang:1.24-alpine AS build

WORKDIR /src

# Dependencies first for better layer caching.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime stage ---------------------------------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server

# Uploaded files (admin icon, ...) live here; mount a volume to persist them.
ENV STORAGE_PATH=/app/storage
RUN mkdir -p /app/storage/public && chown -R app:app /app
USER app

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8000/up || exit 1

# Migrates, seeds demo data (SEED_ON_BOOT=false to skip) and serves the API.
CMD ["server", "serve"]
