FROM redis:8-alpine AS redis

# Expose the standard Redis port
EXPOSE 6379

# Start Redis with a default password (matoi)
CMD ["redis-server", "--requirepass", "matoi"]
