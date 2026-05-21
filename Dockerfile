# Stage 1: Build Tailwind CSS
FROM node:20-alpine AS tailwind
WORKDIR /app
COPY backend/cmd/api/static/css/input.css ./cmd/api/static/css/
COPY backend/cmd/api/tailwind.config.js ./cmd/api/
COPY backend/templates ./templates
RUN npm install tailwindcss @tailwindcss/cli && \
    npx @tailwindcss/cli -i ./cmd/api/static/css/input.css -o ./cmd/api/static/css/output.css --content="./templates/**/*.html" --content="./cmd/api/static/js/**/*.js" --minify

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=tailwind /app/cmd/api/static/css/output.css ./cmd/api/static/css/
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/waffle ./cmd/api

# Stage 3: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/waffle .
COPY --from=builder /app/cmd/api/static ./static
COPY --from=builder /app/templates ./templates
EXPOSE 8383
CMD ["./waffle"]
