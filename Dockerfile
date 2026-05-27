# Stage 1: Build Tailwind CSS
FROM node:20-alpine AS tailwind
WORKDIR /app
COPY backend/cmd/api/static/css/input.css ./cmd/api/static/css/
COPY backend/templates ./templates
RUN npm install tailwindcss @tailwindcss/cli daisyui && \
    npx @tailwindcss/cli -i ./cmd/api/static/css/input.css -o ./cmd/api/static/css/output.css --content="./templates/**/*.html" --content="./cmd/api/static/js/**/*.js" --minify

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
ARG VERSION=dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=tailwind /app/cmd/api/static/css/output.css ./cmd/api/static/css/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=${VERSION}" -o bin/waffle ./cmd/api

# Stage 3: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
RUN adduser -D appuser
WORKDIR /app
COPY --from=builder /app/bin/waffle .
COPY --from=builder /app/cmd/api/static ./static
COPY --from=builder /app/templates ./templates
USER appuser
EXPOSE 8383
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget --no-verbose --tries=1 --spider http://localhost:8383/health || exit 1
STOPSIGNAL SIGTERM
CMD ["./waffle"]
