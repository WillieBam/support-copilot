# ---- frontend build stage ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- build stage ----
FROM docker.io/library/golang:1.26.1-alpine AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY db/ ./db/
COPY backend/ ./backend/
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./backend/main.go

# ---- runtime stage ----
FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=backend-builder /server ./server
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN chown -R app:app /app
USER app

EXPOSE 8080
ENTRYPOINT ["./server", "server"]

