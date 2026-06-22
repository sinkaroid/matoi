# FlareSolverr Setup

```bash
docker pull ghcr.io/sinkaroid/matoi-flaresolverr:latest
```

```bash
docker run -d \
  --name matoi-flaresolverr \
  -p 8191:8191 \
  -p 8192:8192 \
  -e LOG_LEVEL=info \
  -e LOG_HTML=false \
  --memory="512m" \
  --restart unless-stopped \
  ghcr.io/sinkaroid/matoi-flaresolverr:latest
```
