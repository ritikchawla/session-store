# Session Store Service

A backend microservice for generating, validating, and managing user session tokens, storing them in Aerospike with TTL, and tracking metadata and access logs.

## Features

- REST API (Gin)
- Session tokens (UUID)
- Aerospike storage with TTL (default: 30 min)
- Access/audit logs per session
- Dockerized setup (includes Aerospike)

## Endpoints

- `POST /session`  
  Create a new session token.  
  **Body:**  
  ```json
  {
    "user_id": "string",
    "ip": "string",
    "user_agent": "string",
    "device": "string (optional)"
  }
  ```
  **Returns:**  
  `{ "token": "..." }`

- `GET /session/:token`  
  Validate a session token.

- `DELETE /session/:token`  
  Invalidate (logout) a session token.

- `GET /session/:token/logs`  
  Fetch audit logs for a session.

- `GET /health`  
  Health check.

## Running Locally

1. **Build & Start (Docker Compose):**
   ```sh
   docker compose up --build -d
   ```

2. **Check health:**
   ```sh
   curl http://localhost:8080/health
   ```

3. **Test API (example):**
   ```sh
   curl -X POST http://localhost:8080/session \
     -H "Content-Type: application/json" \
     -d '{"user_id":"u001","ip":"127.0.0.1","user_agent":"test-agent"}'
   ```

## Configuration

- `AEROSPIKE_HOST` (default: `aerospike`)
- `AEROSPIKE_PORT` (default: `3000`)
- `SESSION_TTL` (default: `1800` seconds)

## Testing

- Use `curl` or Postman to test endpoints.
- Health endpoint: `GET /health`
- Run unit tests (if present):  
  ```sh
  docker compose exec session-store go test ./...
  ```

## Notes

- Aerospike data is persisted in a Docker volume.
- Audit logs are stored in a separate Aerospike set.
- TTL is configurable via environment variable.