# Implementation Plan — GH-6

## Objective

Design the complete foundational architecture for the TON Trading Platform — a Go monorepo of event-driven microservices communicating through NATS Core + JetStream with protobuf contracts. This plan covers the initial implementation of all eight mandatory services, the shared packages, protobuf contract definitions, infrastructure scaffolding, and the full event pipeline from raw event collection through to explained signals and simulation.

## Scope

### In scope
1. **Protobuf contract definitions** (`packages/contracts/proto/v1/`) — all message types and enums for the six mandatory NATS subjects
2. **Shared library packages** (`packages/shared/`) — NATS connection/consumer helpers, PostgreSQL connection pool, Redis client wrapper, idempotency/dedup utilities, health check (liveness/readiness), graceful shutdown, structured logging, correlation ID propagation, configuration loading
3. **collector-service** — source adapter framework, Telegram/Twitter/RSS/exchange-listing/on-chain/mini-app adapters (stubs where external APIs are unavailable), dedup-before-publish, cursor/checkpoint management, publish to `events.raw`
4. **normalizer-service** — consume `events.raw`, normalize into unified schema, enrich with sentiment/impact scores, publish to `events.normalized`
5. **market-context-service** — price/volume/volatility feeds for TON/USDT, publish to `market.context.updated`, persist historical snapshots, cache latest context
6. **signal-engine-service** — consume `events.normalized` + `market.context.updated`, aggregate with decay/confirmation/scoring, publish to `signals.generated`
7. **risk-engine-service** — consume `signals.generated`, apply risk rules, publish to `signals.risk_adjusted`
8. **explainability-service** — consume `signals.risk_adjusted`, produce human-readable explanations, publish to `signals.explained`
9. **api-gateway-service** — HTTP API for `/signals/latest`, `/signals/history`, `/events`, `/simulate`; read models from PostgreSQL/Redis; optional NATS request/reply for live simulation
10. **simulation-service** — replay historical normalized events + market context through signal/risk pipelines in isolation; reproducible by scenario ID
11. **Infrastructure scaffolding** — Dockerfiles per service, `docker-compose.yml` with NATS, PostgreSQL, Redis, service containers
12. **Database migrations** — per-service migration files for PostgreSQL schemas
13. **Integration test harness** — NATS-based end-to-end test skeleton covering the full event pipeline

## Out of scope

- Production ML/NLP models for sentiment analysis (stub/rule-based enrichment is in scope)
- Real trading execution engine or wallet integration
- Frontend/UI
- Kubernetes manifests or production deployment orchestration beyond Docker Compose
- Multi-pair support beyond TON/USDT (architecture must not prevent it, but implementation targets TON/USDT only)
- TLS/mTLS certificate management for NATS
- CI/CD pipeline definitions (separate task)

## Bounded contexts impacted

| Bounded Context | Owner Service(s) | Data Owned | Events Owned |
|---|---|---|---|
| **Event Collection** | collector-service | source configs, cursor state, raw event archive | `events.raw` (publisher) |
| **Event Normalization** | normalizer-service | normalized event store | `events.normalized` (publisher) |
| **Market Context** | market-context-service | price/volume/volatility snapshots | `market.context.updated` (publisher) |
| **Signal Generation** | signal-engine-service | signal state, decay windows, confirmation state | `signals.generated` (publisher) |
| **Risk Management** | risk-engine-service | risk rule configuration, adjustment audit log | `signals.risk_adjusted` (publisher) |
| **Explainability** | explainability-service | explanation templates/output | `signals.explained` (publisher) |
| **API / Read Models** | api-gateway-service | denormalized read projections | none (consumer only) |
| **Simulation** | simulation-service | scenario configs, simulation runs/results | none (internal; may use NATS request/reply) |

## Services/packages impacted

All services and packages are impacted as this is the initial platform build:

- `packages/contracts` — **new**: all protobuf definitions
- `packages/shared` — **new**: NATS, Postgres, Redis, dedup, health, shutdown, logging, config utilities
- `services/collector-service` — **new**
- `services/normalizer-service` — **new**
- `services/market-context-service` — **new**
- `services/signal-engine-service` — **new**
- `services/risk-engine-service` — **new**
- `services/explainability-service` — **new**
- `services/api-gateway-service` — **new**
- `services/simulation-service` — **new**
- `infrastructure/docker` — **new**: Dockerfiles, docker-compose.yml

## NATS subjects impacted

| Subject | Type | Owner (Publisher) | Consumers | JetStream Stream | Notes |
|---|---|---|---|---|---|
| `events.raw` | JetStream | collector-service | normalizer-service | `EVENTS` | Durable consumer `normalizer-raw` |
| `events.normalized` | JetStream | normalizer-service | signal-engine-service, api-gateway-service, simulation-service | `EVENTS` | Durable consumers per service |
| `market.context.updated` | JetStream | market-context-service | signal-engine-service, risk-engine-service (optional), api-gateway-service | `MARKET` | Durable consumers per service |
| `signals.generated` | JetStream | signal-engine-service | risk-engine-service | `SIGNALS` | Durable consumer `risk-engine-generated` |
| `signals.risk_adjusted` | JetStream | risk-engine-service | explainability-service, api-gateway-service | `SIGNALS` | Durable consumer `explainability-adjusted`, `api-adjusted` |
| `signals.explained` | JetStream | explainability-service | api-gateway-service | `SIGNALS` | Durable consumer `api-explained` |

**Stream definitions:**
- `EVENTS` — subjects: `events.>` — retention: limits, max age configurable (e.g., 30d for replay)
- `MARKET` — subjects: `market.>` — retention: limits, max age configurable
- `SIGNALS` — subjects: `signals.>` — retention: limits, max age configurable

All streams use file-based storage for durability. Replicas configurable (1 for dev, 3 for prod).

## Protobuf contract impact

**Additive** — This is the initial definition of all contracts. No existing fields to break. All messages are new.

### Proposed protobuf structure

```
packages/contracts/proto/v1/
├── common.proto          # Timestamp, EventMetadata, CorrelationID, Pair
├── raw_event.proto       # RawEvent envelope
├── normalized_event.proto # NormalizedEvent with enrichment
├── market_context.proto  # MarketContextSnapshot
├── signal.proto          # GeneratedSignal, RiskAdjustedSignal, ExplainedSignal
└── simulation.proto      # SimulationRequest, SimulationResult
```

### Key message designs

**common.proto**
- `EventMetadata`: `string event_id`, `google.protobuf.Timestamp timestamp`, `string correlation_id`, `string source_service`
- `TradingPair`: enum with `TON_USDT = 0` (extensible)

**raw_event.proto** (`RawEvent`)
- `EventMetadata metadata`
- `string source_type` (telegram, twitter, rss, exchange_listing, onchain, miniapp)
- `string source_id` (dedup key from source)
- `string source_url` (optional)
- `google.protobuf.Timestamp source_timestamp`
- `string title`
- `string body`
- `bytes raw_payload` (original JSON/data for audit/replay)
- `map<string, string> source_metadata`

**normalized_event.proto** (`NormalizedEvent`)
- `EventMetadata metadata`
- `string raw_event_id` (reference to originating RawEvent)
- `TradingPair pair`
- `string source_type`
- `string category` (e.g., listing, partnership, hack, regulatory, whale_move)
- `string title`
- `string summary`
- `Sentiment sentiment` — enum: `SENTIMENT_UNKNOWN`, `BULLISH`, `BEARISH`, `NEUTRAL`
- `double sentiment_score` (-1.0 to 1.0)
- `Impact impact` — enum: `IMPACT_UNKNOWN`, `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`
- `double impact_score` (0.0 to 1.0)
- `google.protobuf.Timestamp source_timestamp`
- `map<string, string> enrichment_metadata`

**market_context.proto** (`MarketContextSnapshot`)
- `EventMetadata metadata`
- `TradingPair pair`
- `double price`
- `double price_change_1h`, `price_change_24h`
- `double volume_24h`
- `double volatility_1h`, `volatility_24h`
- `google.protobuf.Timestamp snapshot_timestamp`
- `map<string, double> indicators` (extensible)

**signal.proto**
- `GeneratedSignal`: `EventMetadata metadata`, `TradingPair pair`, `SignalDirection direction` (LONG/SHORT/NEUTRAL), `double confidence_score` (0-1), `double bullish_score`, `double bearish_score`, `repeated string contributing_event_ids`, `string market_context_snapshot_id`, `google.protobuf.Timestamp generated_at`, `map<string, double> factor_weights`
- `RiskAdjustedSignal`: `EventMetadata metadata`, `string original_signal_id`, `GeneratedSignal original_signal`, `double adjusted_confidence`, `RiskLevel risk_level` (LOW/MEDIUM/HIGH/EXTREME), `bool blocked`, `string block_reason`, `repeated RiskFactor risk_factors`, `google.protobuf.Timestamp adjusted_at`
- `RiskFactor`: `string name`, `double value`, `string description`
- `ExplainedSignal`: `EventMetadata metadata`, `string risk_adjusted_signal_id`, `RiskAdjustedSignal signal`, `string explanation_text`, `repeated ExplanationFactor factors`, `google.protobuf.Timestamp explained_at`
- `ExplanationFactor`: `string factor_name`, `string human_description`, `double contribution`

**simulation.proto**
- `SimulationRequest`: `string scenario_id`, `repeated NormalizedEvent events`, `MarketContextSnapshot market_context`, configuration overrides
- `SimulationResult`: `string scenario_id`, `repeated GeneratedSignal signals`, `repeated RiskAdjustedSignal adjusted_signals`, performance metrics

### Backward compatibility rules
- All fields use explicit field numbers starting from 1
- Enums include `_UNKNOWN = 0` sentinel
- `map`, `repeated`, and `optional` fields used for extensibility
- No `required` fields (proto3 default)
- Future evolution: add fields with new field numbers, never reuse or remove

## Data/storage impact

### PostgreSQL (per-service schemas)

| Service | Schema/Tables | Purpose |
|---|---|---|
| collector-service | `collector.source_configs`, `collector.cursor_state`, `collector.raw_events` | Source configuration, polling cursor checkpoints, raw event archive |
| normalizer-service | `normalizer.normalized_events` | Normalized event store for audit/replay |
| market-context-service | `market.context_snapshots` | Historical market context snapshots |
| signal-engine-service | `signal.generated_signals`, `signal.scoring_config` | Generated signal archive, scoring model config |
| risk-engine-service | `risk.adjusted_signals`, `risk.rules_config` | Risk-adjusted signal archive, rule configuration |
| explainability-service | `explain.explained_signals` | Explained signal archive |
| api-gateway-service | `api.signals_read_model`, `api.events_read_model` | Denormalized read projections |
| simulation-service | `simulation.scenarios`, `simulation.runs`, `simulation.results` | Scenario definitions and results |

Each service owns its schema. No cross-service direct database access.

### Redis

| Service | Key Pattern | Purpose | TTL |
|---|---|---|---|
| collector-service | `dedup:raw:{source_type}:{source_id}` | Source-level dedup | 24h |
| collector-service | `cursor:{source_type}:{source_name}` | Polling cursor cache | none (persistent) |
| normalizer-service | `dedup:norm:{raw_event_id}` | Normalization dedup | 24h |
| signal-engine-service | `dedup:signal:{event_id_set_hash}` | Signal generation dedup | 1h |
| signal-engine-service | `mctx:latest` | Latest market context cache | 5m |
| risk-engine-service | `dedup:risk:{signal_id}` | Risk adjustment dedup | 1h |
| explainability-service | `dedup:explain:{signal_id}` | Explanation dedup | 1h |
| market-context-service | `mctx:ton_usdt:latest` | Latest market context | 60s |
| api-gateway-service | `cache:signals:latest` | API response cache | 10s |

## Idempotency and replay considerations

### Idempotency strategy per service

1. **collector-service**: Dedup by `{source_type}:{source_id}` in Redis before publishing. For sources without stable IDs, use content hash. Each RawEvent gets a UUID `event_id`. Re-publishing the same `source_id` is a no-op.

2. **normalizer-service**: Dedup by `raw_event_id` in Redis. Processing the same RawEvent twice produces no duplicate NormalizedEvent. The JetStream consumer uses AckExplicit — only ack after successful publish of normalized event.

3. **market-context-service**: Each snapshot has a unique `event_id`. Consumers treat each snapshot as the latest state — processing the same snapshot twice is naturally idempotent (same state overwrite).

4. **signal-engine-service**: Deterministic signal generation — same input events + same market context = same signal. Dedup by hashing the contributing event ID set + market context snapshot ID. For replay: scoring config must be versioned/snapshotted to guarantee determinism.

5. **risk-engine-service**: Dedup by `original_signal_id`. Same signal processed twice produces identical risk adjustment (rules are deterministic and config-versioned).

6. **explainability-service**: Dedup by `risk_adjusted_signal_id`. Same input produces same explanation.

7. **api-gateway-service**: Read model projections are upsert-based (INSERT ON CONFLICT UPDATE). Replaying events safely converges to correct state.

8. **simulation-service**: Isolated by design. Each simulation run has a unique ID. No side effects on production streams.

### Replay safety

- All JetStream streams retain messages for configurable duration (e.g., 30 days)
- Durable consumers can be reset by deleting and recreating with `DeliverAll` or `DeliverByStartSequence`
- All handlers are idempotent — replay produces correct state convergence
- No destructive side effects (DELETE operations) triggered by event handlers
- Scoring/risk configuration must be versioned so replay with historical config produces historical results
- `source_timestamp` preserved throughout the pipeline for temporal reasoning during replay

## Acceptance criteria

- [ ] All six NATS subjects (`events.raw`, `events.normalized`, `market.context.updated`, `signals.generated`, `signals.risk_adjusted`, `signals.explained`) have protobuf message definitions in `packages/contracts/proto/v1/`
- [ ] Generated Go code compiles and is importable by all services
- [ ] `packages/shared` provides: NATS client factory, JetStream stream/consumer provisioning, PostgreSQL pool factory, Redis client factory, dedup helper, health check HTTP server, graceful shutdown orchestrator, structured logger factory, config loader
- [ ] collector-service publishes valid `RawEvent` messages to `events.raw` with at least one working source adapter (stub or real)
- [ ] normalizer-service consumes `events.raw`, produces `NormalizedEvent` on `events.normalized` with sentiment/impact enrichment
- [ ] market-context-service publishes `MarketContextSnapshot` to `market.context.updated` on a configurable interval
- [ ] signal-engine-service consumes `events.normalized` + `market.context.updated`, produces `GeneratedSignal` on `signals.generated`
- [ ] risk-engine-service consumes `signals.generated`, produces `RiskAdjustedSignal` on `signals.risk_adjusted`
- [ ] explainability-service consumes `signals.risk_adjusted`, produces `ExplainedSignal` on `signals.explained`
- [ ] api-gateway-service exposes `/signals/latest`, `/signals/history`, `/events`, `/simulate` endpoints returning valid JSON
- [ ] simulation-service can execute a scenario from historical data and return results
- [ ] Every service has liveness (`/healthz`) and readiness (`/readyz`) endpoints
- [ ] Every service performs graceful shutdown on SIGTERM/SIGINT
- [ ] Every service uses structured logging with correlation/event ID propagation
- [ ] All event handlers are idempotent (verified by integration tests that process duplicate messages)
- [ ] Docker Compose brings up the full platform (NATS, PostgreSQL, Redis, all 8 services)
- [ ] End-to-end integration test: inject a raw event → observe explained signal output
- [ ] Database migrations exist and run cleanly for all services
- [ ] Protobuf definitions follow backward-compatibility rules (no required fields, unknown enums default to 0, explicit field numbering)

## Developer slices

### Slice 1: Protobuf contracts and code generation
**Package**: `packages/contracts`
**Deliverables**:
- `proto/v1/common.proto`, `raw_event.proto`, `normalized_event.proto`, `market_context.proto`, `signal.proto`, `simulation.proto`
- `buf.gen.yaml` or `Makefile` target for Go code generation
- Generated Go code in `packages/contracts/gen/go/v1/`
- Go module `go.mod` for contracts package
- Unit test that imports and instantiates each message type

**Dependencies**: None
**Estimated effort**: Small

---

### Slice 2: Shared infrastructure packages
**Package**: `packages/shared`
**Deliverables**:
- `nats/` — NATS connection factory, JetStream stream/consumer helpers, publish-with-retry helper
- `postgres/` — connection pool factory with context, migration runner interface
- `redis/` — client factory, dedup helper (`SetNX` with TTL, check-and-set pattern)
- `health/` — HTTP server with `/healthz` and `/readyz` endpoints, readiness probe registration
- `shutdown/` — graceful shutdown orchestrator (context cancellation + wait group)
- `logging/` — zerolog or zap factory with correlation ID extraction from context
- `config/` — env-based config loader with validation
- Unit tests for dedup helper, shutdown orchestrator, health endpoints

**Dependencies**: Slice 1 (for protobuf types in NATS helpers)
**Estimated effort**: Medium

---

### Slice 3: Infrastructure scaffolding
**Package**: `infrastructure/docker`
**Deliverables**:
- Multi-stage Dockerfile template for Go services
- `docker-compose.yml` with NATS (with JetStream enabled), PostgreSQL, Redis, and all 8 service containers
- NATS JetStream stream provisioning script or init container
- PostgreSQL initialization script (create per-service databases/schemas)
- `.env.example` with all configuration variables

**Dependencies**: Slice 2
**Estimated effort**: Small-Medium

---

### Slice 4: collector-service
**Package**: `services/collector-service`
**Deliverables**:
- Clean Architecture structure: `domain/`, `application/`, `infrastructure/`, `cmd/`
- Domain: `RawEvent` entity, `Source` interface, `EventPublisher` interface
- Application: `CollectUseCase` — orchestrates source polling, dedup, publish
- Infrastructure: NATS publisher, Redis dedup, PostgreSQL cursor store
- Source adapters: at minimum one real adapter (RSS or public Telegram channel via HTTP) and stubs for others
- Adapter registry with config-driven source enabling
- `main.go` with DI wiring, health endpoints, graceful shutdown
- Database migration for `collector` schema
- Unit tests for dedup logic, adapter parsing
- Integration test with embedded NATS

**Dependencies**: Slices 1, 2
**Estimated effort**: Medium-Large

---

### Slice 5: normalizer-service
**Package**: `services/normalizer-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `NormalizedEvent` entity, normalization rules, sentiment/impact scoring interface
- Application: `NormalizeUseCase` — consume raw, classify, enrich, publish
- Infrastructure: NATS consumer + publisher, Redis dedup, PostgreSQL store
- Sentiment enrichment: rule-based keyword/pattern matching (stub for ML)
- Impact scoring: rule-based by source type and category
- Database migration for `normalizer` schema
- Unit tests for normalization logic, sentiment scoring, impact scoring
- Integration test: publish RawEvent → verify NormalizedEvent

**Dependencies**: Slices 1, 2
**Estimated effort**: Medium

---

### Slice 6: market-context-service
**Package**: `services/market-context-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `MarketContextSnapshot` entity, price/volume/volatility model
- Application: `UpdateContextUseCase` — fetch market data, compute indicators, publish
- Infrastructure: external API adapter (e.g., CoinGecko, Binance public API for TON/USDT), NATS publisher, PostgreSQL snapshot store, Redis cache
- Configurable polling interval (default 60s)
- Volatility calculation (e.g., rolling std dev of price changes)
- Database migration for `market` schema
- Unit tests for volatility calculation, snapshot creation
- Integration test: verify `market.context.updated` messages

**Dependencies**: Slices 1, 2
**Estimated effort**: Medium

---

### Slice 7: signal-engine-service
**Package**: `services/signal-engine-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `GeneratedSignal`, scoring model (config-driven weights), decay function, multi-source confirmation logic
- Application: `GenerateSignalUseCase` — aggregate events within time window, apply scoring, emit signal
- Infrastructure: NATS consumers (normalized events + market context), NATS publisher, Redis for latest market context cache and dedup, PostgreSQL for signal archive and scoring config
- Scoring config: YAML/JSON loaded at startup, versioned in DB
- Time-window aggregation: sliding window of normalized events (configurable, e.g., 15m)
- Decay: exponential decay based on event age
- Multi-source confirmation: require events from N distinct source types to boost confidence
- Database migration for `signal` schema
- Unit tests for scoring, decay, confirmation logic (determinism verification)
- Integration test: publish normalized events + market context → verify generated signal

**Dependencies**: Slices 1, 2, 5 (contract), 6 (contract)
**Estimated effort**: Large

---

### Slice 8: risk-engine-service
**Package**: `services/risk-engine-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `RiskAdjustedSignal`, `RiskRule` interface, `RiskFactor`, blocking logic
- Application: `AdjustRiskUseCase` — apply risk rules to generated signal, adjust confidence, assign risk level, optionally block
- Infrastructure: NATS consumer + publisher, Redis dedup + market context cache, PostgreSQL for audit log and rules config
- Rules: configurable rule chain (e.g., low-volume filter, high-volatility dampening, confidence floor, max position sizing hint)
- Database migration for `risk` schema
- Unit tests for each risk rule, blocking logic
- Integration test: publish GeneratedSignal → verify RiskAdjustedSignal

**Dependencies**: Slices 1, 2, 7 (contract)
**Estimated effort**: Medium

---

### Slice 9: explainability-service
**Package**: `services/explainability-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `ExplainedSignal`, `ExplanationFactor`, template engine
- Application: `ExplainUseCase` — decompose risk-adjusted signal into human-readable explanation
- Infrastructure: NATS consumer + publisher, Redis dedup, PostgreSQL store
- Explanation generation: template-based with factor interpolation (e.g., "Confidence reduced from 0.85 to 0.62 due to high volatility (1h vol: 4.2%)")
- Database migration for `explain` schema
- Unit tests for explanation generation with various signal profiles
- Integration test: publish RiskAdjustedSignal → verify ExplainedSignal

**Dependencies**: Slices 1, 2, 8 (contract)
**Estimated effort**: Small-Medium

---

### Slice 10: api-gateway-service
**Package**: `services/api-gateway-service`
**Deliverables**:
- Clean Architecture structure
- Domain: read model entities for signals, events
- Application: query use cases, projection handlers for incoming events
- Infrastructure: HTTP server (net/http or chi), NATS consumers for read model projection, PostgreSQL read store, Redis response cache
- Endpoints: `GET /signals/latest`, `GET /signals/history?from=&to=&limit=`, `GET /events?limit=&source_type=`, `POST /simulate` (proxies to simulation-service via NATS request/reply)
- Read model projection: consume `signals.explained`, `events.normalized`, `market.context.updated` → upsert into denormalized tables
- Database migration for `api` schema
- Unit tests for handlers, projection logic
- Integration test: full pipeline → verify API responses

**Dependencies**: Slices 1, 2, 9 (contract), 11 (for `/simulate`)
**Estimated effort**: Medium

---

### Slice 11: simulation-service
**Package**: `services/simulation-service`
**Deliverables**:
- Clean Architecture structure
- Domain: `Scenario`, `SimulationRun`, `SimulationResult`
- Application: `RunSimulationUseCase` — load historical events + market context, replay through signal + risk logic in-process (embedded, not via production NATS), produce results
- Infrastructure: NATS request/reply responder (for API gateway), PostgreSQL for scenarios and results
- Isolation: uses its own instances of signal/risk domain logic, does NOT publish to production subjects
- Pre-built scenarios: bullish_surge, bearish_crash, fake_hype, conflicting_signals
- Database migration for `simulation` schema
- Unit tests for scenario execution
- Integration test: submit simulation request → verify results

**Dependencies**: Slices 1, 2, 7 (domain logic importable), 8 (domain logic importable)
**Estimated effort**: Medium-Large

---

### Slice 12: End-to-end integration testing and documentation
**Package**: `infrastructure/`, `docs/`
**Deliverables**:
- End-to-end test: start all services via Docker Compose, inject raw event, assert explained signal appears in API
- Pipeline latency benchmarks (optional)
- `docs/architecture.md` — system overview, event flow diagram, bounded context map
- `docs/event-catalog.md` — all NATS subjects, message types, ownership
- `docs/replay-guide.md` — how to replay historical events safely
- `docs/development.md` — local setup, running services, running tests
- `README.md` update

**Dependencies**: All previous slices
**Estimated effort**: Medium

## Reviewer focus

### Correctness
- Verify protobuf field numbering consistency and enum zero-value conventions
- Confirm each service's consumer group/durable name is unique and deterministic
- Validate that signal-engine determinism claim holds: same events + same market context + same config version = same signal
- Confirm dedup keys are collision-resistant (no false positives suppressing legitimate distinct events)

### Contracts
- All proto files use `package tonplatform.v1`
- No `required` fields; all enums have `_UNKNOWN = 0`
- `EventMetadata` is consistently used as the first field in all top-level messages
- `raw_event_id` linkage from NormalizedEvent back to RawEvent is preserved
- `original_signal_id` linkage from RiskAdjustedSignal back to GeneratedSignal is preserved
- `bytes raw_payload` in RawEvent preserves original source data for audit

### Architecture
- Clean Architecture layers are respected: domain has no infrastructure imports
- No service directly accesses another service's database
- NATS subject ownership is not violated (only the designated publisher writes to each subject)
- Simulation-service does not accidentally publish to production NATS subjects
- `context.Context` is threaded through all layers consistently
- Constructor-based DI only; no `init()` side effects or global state

### Reliability
- All NATS consumers use `AckExplicit` with appropriate `AckWait` and `MaxDeliver` settings
- Redis dedup TTLs are long enough to cover expected redelivery windows but short enough to not exhaust memory
- Graceful shutdown drains NATS consumers before exiting
- PostgreSQL transactions are used where atomicity is needed (e.g., write event + ack)
- Health checks distinguish liveness (process alive) from readiness (dependencies connected)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **External API rate limiting** (market data, Telegram, etc.) | High | Medium | Implement exponential backoff, circuit breaker in adapters; cache aggressively; use multiple API sources |
| **JetStream message ordering assumptions** | Medium | High | Document explicitly where ordering matters (within a single source in collector); use partition keys where needed; signal-engine must tolerate out-of-order events within its aggregation window |
| **Sentiment enrichment accuracy** (rule-based) | High | Low | Accept rule-based as v1 baseline; design enrichment interface to swap in ML model later without contract changes |
| **Protobuf schema drift across services** | Medium | High | Single source of truth in `packages/contracts`; CI validation that all services compile against latest generated code; semantic versioning of contracts package |
| **Redis single point of failure for dedup** | Medium | Medium | Dedup is defense-in-depth; handlers are idempotent regardless; Redis loss causes duplicate processing (not data loss); use Redis persistence (RDB/AOF) in production |
| **Signal-engine aggregation window edge cases** | Medium | Medium | Events arriving at window boundaries may be included or excluded non-deterministically; use inclusive-start/exclusive-end semantics; document and test edge cases |
| **Simulation-service accidentally using live NATS subjects** | Low | Critical | Simulation uses in-process domain logic invocation, NOT NATS publish; code review must verify no production subject publishing; integration test asserts no messages on production subjects during simulation |
| **Database migration ordering across services** | Low | Medium | Each service owns its schema independently; no cross-service migration dependencies; migrations are idempotent (IF NOT EXISTS patterns) |
| **Replay of historical events with current (newer) scoring config** | Medium | Medium | Scoring config is versioned; replay mode should optionally accept config version override; document that replaying with current config produces different results than original |
| **Docker Compose resource exhaustion** (8 services + 3 infra) | Medium | Low | Optimize Docker images (multi-stage, minimal base); configure resource limits; document minimum hardware requirements |
