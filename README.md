# Red Packet Rain Backend

End-to-end Golang backend that powers a red packet rain campaign workflow. Merchants create campaigns, users open packets atomically through Redis + Lua, claims are streamed to Kafka, and a consumer persists logs to PostgreSQL.

## Stack
- **API**: Go 1.22 + Gin
- **Storage**: PostgreSQL for configs/logs, Redis for inventory & dedup, Kafka for async events
- **Deployment**: Docker Compose with Redis, Postgres, Zookeeper, Kafka, API, and consumer services
- **Lua**: `scripts/lua/claim.lua` implements the atomic claim logic

## Quick start
```bash
docker compose up --build
```
Compose provisions infrastructure, builds the Go binaries, and starts both services. The API automatically ensures the schema exists on boot (same SQL lives in `migrations/001_init.sql`).

### Database schema
`migrations/001_init.sql` contains the required tables:
- `campaign`
- `campaign_inventory`
- `claim_log`

To run the migration manually (optional because the services run it automatically):
```bash
cat migrations/001_init.sql | docker compose exec -T postgres psql -U postgres -d redpacket
```

## API
Base URL defaults to `http://localhost:8080`.

### Create campaign
```bash
curl -X POST http://localhost:8080/campaign \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Year Blast",
    "start_time": "2025-01-01T00:00:00Z",
    "end_time": "2025-01-07T00:00:00Z",
    "inventory": {"20": 10, "5": 100}
  }'
```
Response:
```json
{"id":1}
```

### Open red packet
```bash
curl -X POST http://localhost:8080/campaign/1/open \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123"}'
```
Possible responses:
- `200 OK` `{ "status": "OK", "amount": 20 }`
- `409 Conflict` `{ "status": "ALREADY_OPENED" }`
- `410 Gone` `{ "status": "SOLD_OUT" }`
- `404 Not Found` if campaign missing
- `400 Bad Request` if the campaign is outside its start/end window

## Kafka consumer
`cmd/consumer` listens to `claim_events`, inserts rows into `claim_log`, and increments `opened_count` in `campaign_inventory`. Logs from the consumer container show processed offsets.

## Configuration
Environment variables (Compose already wires defaults):
- `PORT` – API port (default `8080`)
- `REDIS_ADDR` – Redis host:port
- `POSTGRES_DSN` – Postgres DSN including database
- `KAFKA_BROKERS` – comma-separated broker list (e.g. `kafka:9092`)
- `KAFKA_TOPIC` – Kafka topic for events (`claim_events`)
- `KAFKA_GROUP` – consumer group id (consumer service)
- `METRICS_ADDR` – (consumer) HTTP address that exposes Prometheus metrics, default `:9091`

## Lua script
`scripts/lua/claim.lua` performs:
1. Dedup via `SISMEMBER` on `campaign:{id}:opened`
2. Randomly picks a reward amount with remaining inventory
3. `DECR` inventory, `SADD` the user, and returns `{status, amount}`

## Development
- Run locally: `go run ./cmd/api` and `go run ./cmd/consumer` (ensure Postgres/Redis/Kafka running)
- Tests not included; add integration tests as needed to cover business rules.
- **Performance Enhancements (roadmap)**:
  1. Scale the `api` service horizontally (multiple replicas behind a load balancer) to prevent a single instance from saturating CPU under high QPS.
  2. Consider sharding or clustering Redis (or adopting a multi-threaded variant) so the Lua script is no longer bound by one Redis core.
  3. Tune Kafka publishing by enabling batching or using the async producer to reduce per-request blocking on `WaitForAll` acknowledgements.
  4. Batch consumer writes into Postgres (multiple rows per transaction/COPY) and scale consumer replicas to keep Kafka lag near zero.
  5. Add observability (Prometheus/Grafana, Redis/Kafka metrics) to validate improvements during k6 stress tests.

## Observability
- **Prometheus metrics**:  
  - API exposes `/metrics` on the same port as the HTTP server (default `8080`).  
  - Consumer exposes metrics on `METRICS_ADDR` (defaults to `:9091`). When running via Compose, scrape `http://localhost:9091/metrics`.
- **Prometheus server**: `docker compose up` now launches Prometheus (`http://localhost:9090`) using `prometheus.yml`. It scrapes the API and consumer endpoints automatically. Point Grafana or alerting at this Prometheus instance for richer dashboards.
- **What’s tracked**:  
  - HTTP request latency per route/method/status.  
  - Database, Redis, and Kafka operation duration histograms.  
  - Consumer processing durations per claim event step.
- **Logging**: critical failures during claim persistence and Kafka message handling now log context (campaign/user IDs) so you can correlate spikes with metrics.
- **Verification**: run `curl http://localhost:8080/metrics` or `curl http://localhost:9091/metrics` (consumer) to confirm metrics are emitted, then point Prometheus/Grafana to those endpoints.
