#!/bin/bash
set -euo pipefail
for tool in docker go curl; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Missing required tool: $tool" >&2; exit 2; }
done
PYTHON=""
for candidate in python3 /opt/homebrew/bin/python3 /usr/local/bin/python3; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c 'import json' >/dev/null 2>&1; then
    PYTHON="$candidate"
    break
  fi
done
if [ -z "$PYTHON" ]; then echo "An executable Python 3 installation is required" >&2; exit 2; fi
for port in 4081 4082 55434 16380 19001; do
  if command -v lsof >/dev/null && lsof -nP -iTCP:"$port" -sTCP:LISTEN | grep -q LISTEN; then
    echo "Port $port already in use; refusing to interfere with running services" >&2
    exit 2
  fi
done
if [ "$#" -ne 1 ]; then
  echo "Usage: bash tests/e2e/run_compare.sh /absolute/path/to/dam-creamy-cream-be" >&2
  exit 2
fi
base="$(mktemp -d)"
dir="$(cd "$(dirname "$0")/../.." && pwd)"
nest="$(cd "$1" && pwd)"
if [ ! -f "$nest/dist/src/main.js" ]; then
  echo "NestJS build missing: run npm ci && npx prisma generate && npm run build in $nest" >&2
  exit 2
fi
pg="creamy-pr2-e2e-$$-pg"
redis="creamy-pr2-e2e-$$-redis"
minio="creamy-pr2-e2e-$$-minio"
cleanup() {
  if [ -f "$base/nest-compare.pid" ]; then kill "$(cat "$base/nest-compare.pid")" 2>/dev/null || true; rm -f "$base/nest-compare.pid"; fi
  if [ -f "$base/go-compare.pid" ]; then kill "$(cat "$base/go-compare.pid")" 2>/dev/null || true; rm -f "$base/go-compare.pid"; fi
  docker rm -f "$pg" "$redis" "$minio" >/dev/null 2>&1 || true
  rm -rf "$base"
}
trap cleanup EXIT
for name in "$pg" "$redis" "$minio"; do
 if docker ps -a --format '{{.Names}}' | grep -Fxq "$name"; then echo "Name collision: $name";exit 1;fi
done
docker run -d --name "$pg" -p 127.0.0.1:55434:5432 -e POSTGRES_PASSWORD=fixture-only -e POSTGRES_DB=goapi postgres:17-alpine >/dev/null
docker run -d --name "$redis" -p 127.0.0.1:16380:6379 redis:7-alpine >/dev/null
docker run -d --platform linux/amd64 --name "$minio" -p 127.0.0.1:19001:9000 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin kotsadm/minio@sha256:d3f84d6fc796f6e847070f57630c94744cd14547ff7854cd5872700bb28cbced server /data --address :9000 >/dev/null
for i in $(seq 1 40); do
 if docker exec "$pg" pg_isready -U postgres -d goapi >/dev/null 2>&1 && docker exec "$redis" redis-cli ping | grep -q PONG && curl -fsS http://127.0.0.1:19001/minio/health/live >/dev/null 2>&1; then break;fi
 sleep 1
done
docker exec "$pg" pg_isready -U postgres -d goapi
docker exec "$pg" psql -v ON_ERROR_STOP=1 -U postgres -d goapi -c 'CREATE DATABASE nestapi' >/dev/null
for db in goapi nestapi;do
 for sql in "$dir"/tests/fixtures/db-schema/*.sql;do
  docker exec -i "$pg" psql -v ON_ERROR_STOP=1 -U postgres -d "$db" < "$sql" >/dev/null
 done
 docker exec -i "$pg" psql -v ON_ERROR_STOP=1 -U postgres -d "$db" < "$dir/tests/e2e/fixtures/promotions.sql" >/dev/null
 docker exec -i "$pg" psql -v ON_ERROR_STOP=1 -U postgres -d "$db" < "$dir/tests/e2e/fixtures/remaining_domains.sql" >/dev/null
done
cd "$dir"
go build -o "$base/go-compare-api" ./cmd/api
(
 cd "$nest"
 env PORT=4081 NODE_ENV=development \
 DATABASE_URL='postgresql://postgres:fixture-only@127.0.0.1:55434/nestapi?sslmode=disable' \
 REDIS_URL='redis://127.0.0.1:16380/0' COOKIE_SECRET_KEY='fixture-cookie' \
 CLOUDINARY_CLOUD_NAME='fixture' CLOUDINARY_API_KEY='fixture' CLOUDINARY_API_SECRET='fixture' \
 node dist/src/main.js > "$base/nest-compare.log" 2>&1 &
 echo $! > "$base/nest-compare.pid"
)
(
 cd "$dir"
 env PORT=4082 NODE_ENV=development \
 DATABASE_URL='postgresql://postgres:fixture-only@127.0.0.1:55434/goapi?sslmode=disable' \
 REDIS_URL='redis://127.0.0.1:16380/1' \
 ACCESS_JWT_SECRET='compare-access-secret' REFRESH_JWT_SECRET='compare-refresh-secret' \
 MINIO_ENDPOINT='127.0.0.1:19001' MINIO_ACCESS_KEY='minioadmin' MINIO_SECRET_KEY='minioadmin' MINIO_BUCKET='creamy-compare' MINIO_USE_SSL=false \
 PUBLIC_API_URL='http://127.0.0.1:4082' \
 "$base/go-compare-api" > "$base/go-compare.log" 2>&1 &
 echo $! > "$base/go-compare.pid"
)
for i in $(seq 1 30);do
 nest_status=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:4081/discounts || true)
 go_status=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:4082/discounts || true)
 if [ "$nest_status" = 200 ] && [ "$go_status" = 200 ];then break;fi
 sleep 1
done
if [ "$nest_status" != 200 ] || [ "$go_status" != 200 ];then
 echo "Nest startup status $nest_status; Go startup status $go_status"
 tail -35 "$base/nest-compare.log";tail -35 "$base/go-compare.log";exit 2
fi
"$PYTHON" "$dir/tests/e2e/compare_core.py"
"$PYTHON" "$dir/tests/e2e/compare_remaining.py"
"$PYTHON" "$dir/tests/e2e/compare_promotions.py"
