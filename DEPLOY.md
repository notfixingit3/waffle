# Deployment Guide

## Quick Start (Local Development)

```bash
docker compose up --build
```

- Application: http://localhost:8383
- Admin password: `admin123` (change with `ADMIN_PASSWORD` env var)

---

## Production Deployment

### Environment Variables

Create a `.env` file in the project root:

```bash
# Required
JWT_SECRET=your-super-secret-jwt-key-min-32-chars
ADMIN_PASSWORD=your-secure-admin-password

# Optional (defaults shown)
DATABASE_URL=postgres://user:pass@host:5432/syrup?sslmode=require
```

### Option 1: Docker Compose (Single Server)

```bash
# Production build
docker compose -f docker-compose.yml up -d --build

# View logs
docker compose logs -f

# Update
docker compose pull
docker compose up -d
```

### Option 2: Railway

1. Connect your GitHub repo to Railway
2. Add PostgreSQL plugin
3. Set environment variables in Railway dashboard
4. Deploy

Railway will auto-detect the `Dockerfile` in each service directory.

### Option 3: Render

1. Create a new Web Service
   - Build command: `docker compose up --build`
   - Start command: (handled by Docker)

2. Create a PostgreSQL database
4. Set `DATABASE_URL` and other env vars

### Option 4: Fly.io

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Application (uses root Dockerfile)
fly launch --dockerfile Dockerfile --name syrup-app

# Database
fly postgres create --name syrup-db
```

---

## Instagram In-App Browser Notes

The app is optimized for Instagram's in-app browser (WebKit on iOS, Chrome Custom Tabs on Android):

- Viewport locked to prevent zoom
- Touch targets minimum 44x44px
- No pull-to-refresh conflicts
- Fast tap response (no 300ms delay)

Test by sharing your waffle link in an Instagram DM and opening it.

---

## SSL / HTTPS

Required for production. The app uses WebSockets which require HTTPS in most browsers.

**Recommended:** Put Cloudflare or nginx in front for SSL termination.

Example nginx config:

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://localhost:8383;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /ws/ {
        proxy_pass http://localhost:8383;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## Database Migrations

Migrations run automatically on backend startup. To run manually:

```bash
docker compose exec backend ./main migrate
```

---

## Backup

```bash
# Backup
docker compose exec postgres pg_dump -U syrup syrup > backup.sql

# Restore
docker compose exec -T postgres psql -U syrup syrup < backup.sql
```

---

## Monitoring

Health check endpoint: `GET /health`

Returns:
```json
{"status": "ok", "db": "connected"}
```

Set up UptimeRobot or similar to ping `/health` every 5 minutes.
