# ---------- 1. Build the React frontend ----------
FROM node:24-alpine AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---------- 2. Build the Go backend (pure Go SQLite, no CGO) ----------
FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /leave-report .

# ---------- 3. Runtime ----------
FROM alpine:3.22
RUN adduser -D -u 10001 app && mkdir -p /data && chown app /data
COPY --from=backend /leave-report /usr/local/bin/leave-report
COPY --from=frontend /src/dist /app/web

ENV DB_PATH=/data/leave.db \
    STATIC_DIR=/app/web \
    ADDR=:8080 \
    TZ=Asia/Dhaka

USER app
VOLUME /data
EXPOSE 8080
CMD ["leave-report"]
