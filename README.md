# Go Gin Skeleton

> **Go Gin Skeleton template includes all the common packages and setup used for API development
> using [gin](https://gin-gonic.com).**

## Configured With

- Dependency Injection: [fx](https://github.com/uber-go/fx)
- Routing: [gin web framework](https://gin-gonic.com)
- Logging: [zap](https://github.com/uber-go/zap)
- Database: ([mysql](https://gorm.io/driver/mysql) / [sqlmock](https://github.com/DATA-DOG/go-sqlmock))
- ORM: [gorm](https://gorm.io/docs)
- API documentation: [gin-swagger](https://github.com/swaggo/gin-swagger)
- Middlewares
  - CORS (env-driven allowlist via `CORS_ALLOWED_ORIGINS`)
  - Rate Limit (per-user when authenticated, per-IP otherwise)
  - DB Transaction (scoped to write methods only)
  - Request ID (`X-Request-ID`, surfaces in error envelope `trace_id`)
  - Structured error envelope + central handler (`lib/api_errors`)
  - Idempotency-Key (replays POST responses; MySQL or Redis backend, see below)
- CLI tools
    - [atlas](https://atlasgo.io/): for DB migrations
  - [gentool](https://gorm.io/gen/): to generate dao objects from database
  - [swag](https://github.com/swaggo/swag): to generate swagger docs
  - [air](https://github.com/air-verse/air): hot-reload
- Microservice building blocks (see [Microservice Architecture](#microservice-architecture))
  - gRPC surface (protobuf contracts via [buf](https://buf.build), health + reflection)
  - Async events over [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream)
  - Distributed tracing ([OpenTelemetry](https://opentelemetry.io)) + [Prometheus](https://prometheus.io) metrics
  - 12-factor config (env-var-only friendly), inter-service gRPC client factory
  - Production Dockerfile, Helm chart, and GitHub Actions CI

**For Debugging 🐞** Debugger runs at `5002`. Vs code configuration is at `.vscode/launch.json` which will attach
debugger to remote application.

## Running

- Copy `.env.example` to `.env` and update according to requirement.
- ### using Docker
  - run `docker-compose up` (with default configuration will run at `5000` and adminer runs at `5001`)
- ### using Gin Watch
  - ### make sure to add go/bin path to your .bashrc/.zshrc file
    - run `make run`

## Commands 🛳

| Command               | Desc                                                                         |
|-----------------------|------------------------------------------------------------------------------|
| `make install`        | installs goalngci-lint and change the hooks config                           |
| `make run`            | runs the project using gin watcher                                           |
| `make migrate`        | runs [atlas](https://atlasgo.io/) migrate command with env configs from .env |
| `make crud`           | Create crud template                                                         |
| `make swagger`        | Run this command to generate swag docs                                       |
| `make dao`            | Generates go structs from database                                           |
| `make context-upload` | Upload Context to Circle CI Directly                                         |

[//]: # "TODO :: Need a proper name ⬇️"

## External Services

> Run `go get <package name>` to install.

#### Firebase

- Package name: github.com/readytowork-org/go_firebase_service
- Github: https://github.com/readytowork-org/go_firebase_service

#### GCP

- Package name: github.com/readytowork-org/go_gcp_service
- Github: https://github.com/readytowork-org/go_gcp_service

## Microservice Architecture

This skeleton ships as a well-formed microservice: it serves HTTP **and** gRPC,
publishes/consumes async events, emits traces and metrics, and packages for
Kubernetes. Every piece self-gates on configuration, so the service also runs
fine standalone with none of the infrastructure present.

### Configuration (12-factor)

`lib/config/env.go` reads an env file when present (local dev) and always
overlays real environment variables. **The env file is optional** — in a
container, config comes entirely from the environment (ConfigMap/Secret), so no
`.env` needs to be baked into the image. Required values are still validated at
startup. New variables (all optional) are documented in `.env.example`:

| Variable | Purpose |
|----------|---------|
| `SERVICE_NAME` / `SERVICE_VERSION` | Identity in traces/metrics/events |
| `GRPC_PORT` | gRPC listen port (empty disables gRPC) |
| `SERVICE_ENDPOINTS` | Peer registry `name=host:port,...` for gRPC clients |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP/gRPC collector (empty = no export) |
| `OTEL_TRACE_SAMPLE_RATIO` | Trace sampling 0.0–1.0 |
| `METRICS_ENABLED` | Expose `/metrics` |
| `NATS_URL` | Event broker (empty = no-op broker) |
| `NATS_STREAM_NAME` / `EVENT_SUBJECT_PREFIX` | JetStream stream + subject namespace |

### gRPC

Contracts live in `proto/` and are compiled with **buf**; generated Go is
committed under `proto/gen/go`.

```sh
make proto-tools   # one-time: install buf + protoc-gen-go(-grpc)
make proto         # buf lint + buf generate
```

- `lib/grpcserver` runs a gRPC server alongside HTTP under the same fx
  lifecycle, with OpenTelemetry interceptors, panic recovery, the standard
  gRPC **health** service, and **reflection** (for `grpcurl`).
- Feature modules attach handlers via `fx.Invoke` — see
  `api/user/user/grpc.go`, which implements `UserService.GetUser` by reusing
  the *same* domain `Service` the HTTP handlers use.
- `lib/grpcclient` is a factory for dialling peers resolved from
  `SERVICE_ENDPOINTS`, with shared connections, OTel instrumentation, and a
  retry policy.

### Async events

`lib/events` defines a `Broker` with a NATS JetStream backend (durable,
at-least-once) and a no-op fallback selected when `NATS_URL` is empty. Events
use a CloudEvents-style envelope carrying a W3C `traceparent`, so a consumer
continues the producer's distributed trace. `api/admin/user` publishes
`user.created` on user creation; `subscriber.go` shows the consuming side.

### Observability

- **Tracing**: `lib/telemetry` installs W3C propagators always and exports
  spans via OTLP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. `otelgin` and
  `otelgrpc` create server spans automatically.
- **Metrics**: RED (rate/errors/duration) metrics plus Go/process collectors
  are exposed at `GET /metrics` (when `METRICS_ENABLED=true`), labelled by the
  matched route template to stay low-cardinality.
- **Health**: `/livez`, `/readyz`, `/health-check` (HTTP) and the gRPC health
  service.

### Local infrastructure

`docker-compose.yml` includes `nats`, `otel-collector`, and `prometheus`
services. Enable them by pointing the app at them in `.env`:

```sh
NATS_URL=nats://nats:4222
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

Prometheus UI: <http://localhost:9091>.

### Deployment

- **Image**: root `Dockerfile` is a multi-stage, non-root production build
  (distinct from the hot-reload `docker/web.Dockerfile`).
- **Kubernetes**: `deploy/helm/gin-skeleton` is a Helm chart with Deployment
  (HTTP + gRPC ports, liveness `/livez`, readiness `/readyz`), Service,
  ConfigMap, Secret, optional HPA and `ServiceMonitor`.

  ```sh
  helm upgrade --install my-svc deploy/helm/gin-skeleton \
    --set image.repository=your-registry/gin-skeleton \
    --set secrets.JWT_ACCESS_SECRET=... \
    --set secrets.JWT_REFRESH_SECRET=... \
    --set secrets.DB_PASSWORD=...
  ```

- **CI**: `.github/workflows/ci.yml` runs buf lint, a generated-code drift
  check, golangci-lint, build, vet, tests, and a Docker build.

> **Build note:** the project uses a cgo dependency (`chai2010/webp`), so a C
> toolchain is required to compile the full binary (the Dockerfile and CI
> install one). The pre-existing, unwired `api/admin/gcp_billing` sample
> package does not compile and is excluded in CI.

## Swagger docs config

> Please refer to [SWAGGER.md](https://github.com/readytowork-org/go-gin-skeleton/blob/develop/SWAGGER.md)

## Run CLI 🖥

The CLI mode is gated by `os.Args[1] == "cli"` (see `lib/utils/isCli.go`). Any
binary built from this project enters interactive CLI mode when launched with
that argument; otherwise it serves HTTP.

### Local

```sh
go run . cli
```

### Inside Docker

```sh
docker-compose exec web sh
./tmp/main cli       # if built via `make run` (air)
# or
go run . cli
```

You'll get an interactive menu (powered by `promptui`) listing available
commands. Currently:

- `CREATE_SEED_DATA` — runs the seed registry defined in `database/seeds/`.
- `EXIT_APPLICATION` — quit.

### Adding a new CLI command

1. Implement the `cli.Command` interface (`Run()` + `Name()`) in a file under
   `cli/`.
2. Provide it via `fx.Provide(NewYourCommand)` in `cli/cli.go`.
3. Add it to the `commands` slice in `NewApplication` (`cli/app.go`).

### Seeds (non-interactive)

Seeds in `database/seeds/` are also run automatically at HTTP server startup
(see `bootstrap/bootstrap.go`). Add a new seed by implementing `seeds.Seed`
and registering it in the `seeds.Module`.

## Hot reload

`make run` uses [air](https://github.com/air-verse/air); config is in
`.air.toml`. The `tmp/` directory holds the dev binary and is gitignored.

## Idempotency

The Idempotency middleware is mounted globally in `lib/router/router.go`
so every POST route is automatically retry-safe — no per-route wiring
required. The middleware self-gates: it only acts on POST requests that
carry the `Idempotency-Key` header. Non-POST and untagged requests pay
only a cheap method/header check.

When a client sends `Idempotency-Key`, the first response is cached for
`IDEMPOTENCY_TTL` and replayed for any subsequent request with the same
key — duplicates are flagged with the response header
`Idempotent-Replay: true`.

Backend is chosen by `IDEMPOTENCY_STORE`:

| Value     | Backend                                  | Notes                                  |
|-----------|------------------------------------------|----------------------------------------|
| (empty)   | Noop                                     | Disabled — middleware passes through.  |
| `mysql`   | `idempotency_keys` table (existing DB)   | Durable; survives restarts.            |
| `redis`   | `REDIS_ADDR`                             | Fast; native TTL. Compose includes a redis service. |

The MySQL backend uses the `idempotency_keys` table created by
`database/schema.sql` — run `make migrate` after switching to `mysql`.

## Auth

- `POST /api/v1/login` — issues an access/refresh token pair and persists the
  refresh token (hashed) in `refresh_tokens`.
- `POST /api/v1/login/refresh` — rotates the refresh token; reuse of a
  revoked refresh token revokes all sessions for the user.
- `POST /api/v1/logout` — revokes the supplied refresh token. Always returns
  `200` to avoid leaking token validity.

## Health probes

| Path            | Behaviour                                       |
|-----------------|-------------------------------------------------|
| `/livez`        | Process is up — always `200`.                   |
| `/health-check` | Pings the DB with a 2 s timeout.                |
| `/readyz`       | Alias for `/health-check` for k8s-style probes. |

## Implements Google Cloud Proxy by default

This reduces hassle for developer to update IP in Cloud SQL during IP Change.

Implemented through docker image as follows

- Cloud SQL -> Google Proxy Docker Image -> Web App

#### Points to remember for smooth working

- `ServiceAccountKey.json` requires Cloud SQL Read and Write Permission
- `DB_HOST_NAME` value in `.env` is required
- `DB_HOST=cloud-sql-proxy` instead of `IPV4` or `DB_HOST_NAME` for development environment
- `DB_PORT` will be `3306` by default

## For auto generate of CRUD(Create, ReaD, Update & Delete) api following informations are needed and will be asked in terminal:

- resource-name: name of CRUD in upper camelCase. examples:Food,Puppy,ProductCategory etc.

- resource-table-name: name of CRUD in lower snake case. examples:food,puppy,product_category etc.

- plural-resource-table-name: plural name for the table going to be created. example: foods, puppies,
  product_categories.

- plural-resource-name: plural name of CRUD in Upper camelCase. examples:Foods,Puppies,ProductCategories etc.
