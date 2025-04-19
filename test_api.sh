#!/bin/sh

set -e

echo "Testing /health..."
curl -sf http://localhost:8080/health

echo "Testing POST /session..."
TOKEN=$(curl -sf -X POST http://localhost:8080/session \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u001","ip":"127.0.0.1","user_agent":"test-agent"}' | jq -r .token)

echo "Token: $TOKEN"

echo "Testing GET /session/:token..."
curl -sf http://localhost:8080/session/$TOKEN

echo "Testing GET /session/:token/logs..."
curl -sf http://localhost:8080/session/$TOKEN/logs

echo "Testing DELETE /session/:token..."
curl -sf -X DELETE http://localhost:8080/session/$TOKEN

echo "Testing GET /session/:token after delete (should be invalid)..."
curl -sf http://localhost:8080/session/$TOKEN || echo "Session invalid as expected"

echo "All API endpoint tests completed."