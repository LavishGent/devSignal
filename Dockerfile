# DevSignal production image: build the SPA, compile the Go API, and ship
# one small container where the API serves both JSON and static assets.

# --- Stage 1: frontend build ---
FROM node:24-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: backend build ---
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /devsignal ./cmd/api

# --- Stage 3: runtime ---
FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 devsignal
COPY --from=backend /devsignal /usr/local/bin/devsignal
COPY --from=frontend /app/dist /srv/static

USER devsignal
ENV PORT=8080 \
    APP_ENV=production \
    STATIC_DIR=/srv/static
EXPOSE 8080

CMD ["devsignal"]
