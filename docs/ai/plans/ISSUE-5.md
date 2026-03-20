# Implementation Plan — GH-5

## Objective

Implement the foundational TON Event-Driven Trading Signal Platform as a Go monorepo of independent deployable microservices. This includes establishing all core services, protobuf contracts, NATS subject topology, shared infrastructure packages, and the complete event flow from raw event collection through to explained trading signals.

## Scope

- **Protobuf contracts** for all inter-service communication across all six NATS subjects
- **Shared infrastructure packages** (NATS client wrappers, PostgreSQL/Redis clients, logging, health checks, graceful shutdown, dedup utilities)
- **collector-service** — multi-source event collection (Telegram, Twitter/X, RSS, exchange listings, on-chain stubs, mini apps)
- **normalizer-service** — raw-to-normalized transformation with sentiment/impact enrichment
- **market-context-service** — price, volume, volatility context for TON/USDT
- **signal-engine-service** — scoring, decay, multi-source confirmation, signal generation
- **risk-engine-service** — confidence adjustment, risk levels, blocking rules
- **explainability-service** — human-readable rationale generation
- **api-gateway-service** — HTTP API for signals, events, history, simulate endpoints
- **simulation-service** — scenario-based backtesting isolated from live streams
- **Infrastructure** — Docker Compose setup, database migrations, NATS JetStream stream/consumer configuration

## Out of scope

- Production deployment manifests (Kubernetes, Helm)
- CI/CD pipeline configuration
- Execution engine (order placement, position management)
- External API authentication/authorization beyond basic structure
- Real external API integrations (adapters will be stubbed for initial delivery)
- Performance tuning and load testing
- Monitoring dashboards (Grafana, Prometheus scrape configs)

## Bounded contexts impacted

| Bounded Context | Services | Ownership |
|-----------------|----------|-----------|
| Event Collection | collector-service | Raw event ingestion, source adapters, dedup |
| Event Normalization | normalizer-service | Transformation, enrichment, sentiment/impact scoring |
| Market Data | market-context-service | Price/volume/volatility snapshots for TON/USDT |
| Signal Generation | signal-engine-service | Scoring models, decay, multi-source confirmation |
| Risk Management | risk-engine-service | Risk adjustment, confidence filtering, blocking |
| Explainability | explainability-service | Rationale generation from signal factors |
| API Surface | api-gateway-service | HTTP interface, read models, caching |
| Simulation | simulation-service | Backtesting, scenario replay, isolated state |

## Services/packages impacted

### New packages
- `packages/contracts/` — all protobuf definitions and generated Go code
- `packages/shared/nats/` — NATS Core + JetStream client wrapper
- `packages/shared/postgres/` — PostgreSQL connection and migration utilities
- `packages/shared/redis/` — Redis client wrapper with dedup helpers
- `packages/shared/logging/` — structured logging (zerolog)
- `packages/shared/health/` — liveness/readiness probe handlers
- `packages/shared/shutdown/` — graceful shutdown coordinator
- `packages/shared/dedup/` — idempotency/dedup utilities (Redis-backed)

### New services
- `services/collector-service/`
- `services/normalizer-service/`
- `services/market-context-service/`
- `services/signal-engine-service/`
- `services/risk-engine-service/`
- `services/explainability-service/`
- `services/api-gateway-service/`
- `services/simulation-service/`

### Infrastructure
- `infrastructure/docker/` — Docker Compose, Dockerfiles
- `infrastructure/migrations/` — PostgreSQL schema migrations per service

## NATS subjects impacted

| Subject | Publisher | Consumers | Impact |
|---------|-----------|-----------|--------|
| `events.raw` | collector-service | normalizer-service | **New** — JetStream stream required |
| `events.normalized` | normalizer-service | signal-engine-service | **New** — JetStream stream required |
| `market.context.updated` | market-context-service | signal-engine-service, risk-engine-service | **New** — JetStream stream required |
| `signals.generated` | signal-engine-service | risk-engine-service | **New** — JetStream stream required |
| `signals.risk_adjusted` | risk-engine-service | explainability-service, api-gateway-service | **New** — JetStream stream required |
| `signals.explained` | explainability-service | api-gateway-service | **New** — JetStream stream required |

### JetStream configuration requirements
- All streams: retention by limits (time + count), replay policy for historical reprocessing
- Durable consumers per service with explicit ack, max deliver for retry safety
- Consumer names deterministic: `<service>-<subject-slug>`

## Protobuf contract impact

**Impact level: Additive (new contracts, no existing contracts to break)**

### New message types (packages/contracts/proto/v1/)

**events.proto**
- `RawEvent` — source, source_event_id, payload, timestamp, metadata
- `NormalizedEvent` — event_id, source_ref, event_type, asset, sentiment, impact, content, timestamp, enrichment_metadata

**market.proto**
- `MarketContextSnapshot` — asset, price, volume_24h, volatility, timestamp, indicators

**signals.proto**
- `GeneratedSignal` — signal_id, direction (BULLISH/BEARISH/NEUTRAL), confidence, contributing_events[], market_context_ref, decay_config, timestamp
- `RiskAdjustedSignal` — original_signal_id, adjusted_confidence, risk_level, blocked, block_reason, adjustments[]
- `ExplainedSignal` — signal_id, summary, factors[], recommendation, timestamp

**common.proto**
- `Timestamp` — wrapper for consistent timestamp handling
- `SourceReference` — source, source_event_id
- `Direction` — enum BULLISH/BEARISH/NEUTRAL
- `RiskLevel` — enum LOW/MEDIUM/HIGH/CRITICAL

### Versioning
- Package path: `ton.trading.v1`
- Go package: `github.com/<org>/ton-trading/packages/contracts/gen/go/v1`

## Data/storage impact

### PostgreSQL schemas (per-service isolation)

| Service | Schema | Tables |
|---------|--------|--------|
| collector-service | `collector` | `source_configs`, `collection_cursors`, `raw_events_log` (optional audit) |
| normalizer-service | `normalizer` | `normalized_events` (for replay/audit) |
| market-context-service | `market` | `context_snapshots` |
| signal-engine-service | `signals` | `generated_signals`, `scoring_config` |
| risk-engine-service | `risk` | `risk_adjusted_signals`, `risk_rules` |
| explainability-service | `explain` | `explained_signals` |
| api-gateway-service | `api` | `signals_read_model`, `events_read_model` (denormalized for queries) |
| simulation-service | `simulation` | `scenarios`, `simulation_runs`, `simulation_results` |

### Redis usage

| Service | Key patterns | Purpose | TTL |
|---------|--------------|---------|-----|
| collector-service | `dedup:raw:{source}:{source_event_id}` | Source-level dedup | 24h |
| normalizer-service | `dedup:norm:{event_id}` | Processing dedup | 24h |
| market-context-service | `cache:market:ton_usdt:latest` | Latest context cache | 60s |
| signal-engine-service | `dedup:signal:{hash}` | Signal generation dedup | 1h |
| risk-engine-service | `dedup:risk:{signal_id}` | Risk processing dedup | 1h |
| api-gateway-service | `cache:signals:latest`, `cache:signals:history:{page}` | Response cache | 30s |

## Idempotency and replay considerations

### Event ID strategy
- `RawEvent`: `{source}:{source_event_id}` — deterministic from source
- `NormalizedEvent`: UUID v5 from `(source, source_event_id, version)` — deterministic
- `GeneratedSignal`: UUID v5 from `(sorted_event_ids, market_context_id, config_version)`
- `RiskAdjustedSignal`: UUID v5 from `(signal_id, risk_config_version)`
- `ExplainedSignal`: UUID v5 from `(signal_id, explain_config_version)`

### Replay safety guarantees
1. All handlers check Redis dedup key before processing
2. All handlers are pure functions of input + config snapshot
3. Config versions are embedded in event IDs to detect config drift during replay
4. PostgreSQL writes use upsert (ON CONFLICT) keyed by logical event ID
5. JetStream consumers use explicit ack only after successful processing + dedup write
6. Market context snapshots are immutable; new snapshots don't overwrite old timestamps

### Ordering considerations
- `events.normalized` does not guarantee order; signal-engine handles windowing internally
- `market.context.updated` uses timestamp-based supersession; latest wins in cache
- Signal ordering within same timestamp window is non-deterministic but deterministic for same input set

## Acceptance criteria

- [ ] All protobuf contracts defined in `packages/contracts/proto/v1/` with generated Go code
- [ ] Shared packages implemented: NATS, PostgreSQL, Redis, logging, health, shutdown, dedup
- [ ] collector-service collects from at least 3 stub sources and publishes to `events.raw`
- [ ] normalizer-service transforms raw events and publishes to `events.normalized`
- [ ] market-context-service publishes context snapshots to `market.context.updated`
- [ ] signal-engine-service generates signals from normalized events + market context
- [ ] risk-engine-service adjusts signals and publishes to `signals.risk_adjusted`
- [ ] explainability-service produces human-readable output on `signals.explained`
- [ ] api-gateway-service exposes `/signals/latest`, `/signals/history`, `/events`, `/simulate`
- [ ] simulation-service runs isolated scenario replays
- [ ] All services have health endpoints (liveness + readiness)
- [ ] All services implement graceful shutdown
- [ ] All event handlers are idempotent (verified by replay tests)
- [ ] Docker Compose brings up full platform with NATS, PostgreSQL, Redis
- [ ] Integration test demonstrates full flow: raw event → explained signal via API

## Developer slices

### Slice 1: Foundation — Contracts and Shared Packages
**Owner:** contracts-developer  
**Estimated effort:** 3-4 days

1.1. Initialize repository structure (go.mod, workspace if needed)
1.2. Define all protobuf contracts (`events.proto`, `market.proto`, `signals.proto`, `common.proto`)
1.3. Set up buf.yaml and buf.gen.yaml for code generation
1.4. Generate Go code, validate compilation
1.5. Implement `packages/shared/nats/` — connection, publish, subscribe, JetStream consumer wrapper
1.6. Implement `packages/shared/postgres/` — connection pool, migration runner
1.7. Implement `packages/shared/redis/` — connection, dedup helpers
1.8. Implement `packages/shared/logging/` — zerolog wrapper with correlation ID support
1.9. Implement `packages/shared/health/` — HTTP handlers for liveness/readiness
1.10. Implement `packages/shared/shutdown/` — signal handling, graceful coordinator
1.11. Implement `packages/shared/dedup/` — Redis-backed idempotency check/set
1.12. Unit tests for all shared packages

**Exit criteria:** Shared packages compilable, unit tested; contracts generate valid Go code

---

### Slice 2: Infrastructure Setup
**Owner:** infrastructure-developer  
**Estimated effort:** 1-2 days

2.1. Create `infrastructure/docker/docker-compose.yml` (NATS, PostgreSQL, Redis)
2.2. Create JetStream stream initialization script/sidecar
2.3. Create per-service Dockerfiles (multi-stage build template)
2.4. Create `infrastructure/migrations/` structure with per-service subdirectories
2.5. Write initial migrations for all service schemas
2.6. Verify docker-compose up brings healthy infrastructure

**Exit criteria:** `docker-compose up` starts all infrastructure; migrations apply cleanly

---

### Slice 3: collector-service
**Owner:** collector-developer  
**Estimated effort:** 2-3 days

3.1. Service skeleton with Clean Architecture layers
3.2. Domain: RawEvent entity, Source interface, CollectorService
3.3. Application: CollectUseCase, adapters orchestration
3.4. Infrastructure: NATS publisher, PostgreSQL repository, Redis dedup
3.5. Stub adapters: TelegramAdapter, TwitterAdapter, RSSAdapter (return mock data)
3.6. Health endpoints, graceful shutdown integration
3.7. Config via environment variables
3.8. Unit tests for domain/application
3.9. Integration test: stub source → `events.raw` published

**Exit criteria:** Service starts, publishes stub events to `events.raw`, idempotent

---

### Slice 4: normalizer-service
**Owner:** normalizer-developer  
**Estimated effort:** 2-3 days

4.1. Service skeleton with Clean Architecture layers
4.2. Domain: NormalizedEvent entity, EnrichmentService interface
4.3. Application: NormalizeUseCase
4.4. Infrastructure: NATS consumer (`events.raw`), NATS publisher (`events.normalized`), Redis dedup
4.5. Enrichment stubs: SentimentAnalyzer, ImpactScorer (deterministic mock logic)
4.6. Health endpoints, graceful shutdown
4.7. Unit tests
4.8. Integration test: consume raw → publish normalized

**Exit criteria:** Transforms raw events, preserves source refs, enrichment applied, idempotent

---

### Slice 5: market-context-service
**Owner:** market-developer  
**Estimated effort:** 2 days

5.1. Service skeleton
5.2. Domain: MarketContextSnapshot entity, PriceProvider interface
5.3. Application: UpdateContextUseCase (scheduled or triggered)
5.4. Infrastructure: NATS publisher (`market.context.updated`), PostgreSQL for history, Redis cache
5.5. Stub price provider (returns controlled test data)
5.6. Scheduler for periodic updates (configurable interval)
5.7. Health endpoints, graceful shutdown
5.8. Unit tests
5.9. Integration test: service publishes context on schedule

**Exit criteria:** Publishes market context snapshots, stores history, caches latest

---

### Slice 6: signal-engine-service
**Owner:** signal-developer  
**Estimated effort:** 3-4 days

6.1. Service skeleton
6.2. Domain: GeneratedSignal entity, ScoringModel interface, DecayCalculator
6.3. Application: GenerateSignalUseCase, event windowing/aggregation logic
6.4. Infrastructure: NATS consumers (`events.normalized`, `market.context.updated`), NATS publisher (`signals.generated`), Redis dedup + market context cache read
6.5. Scoring model: config-driven weights, multi-source confirmation rules
6.6. Decay: time-based confidence reduction
6.7. PostgreSQL for generated signals, scoring config
6.8. Health endpoints, graceful shutdown
6.9. Unit tests for scoring determinism
6.10. Integration test: normalized events + context → signal generated

**Exit criteria:** Deterministic signal generation, same inputs = same output, config-driven

---

### Slice 7: risk-engine-service
**Owner:** risk-developer  
**Estimated effort:** 2-3 days

7.1. Service skeleton
7.2. Domain: RiskAdjustedSignal entity, RiskRule interface
7.3. Application: AdjustRiskUseCase
7.4. Infrastructure: NATS consumer (`signals.generated`), NATS publisher (`signals.risk_adjusted`), Redis dedup, market context cache read
7.5. Risk rules: confidence threshold, volatility-based adjustment, blocking rules
7.6. PostgreSQL for adjusted signals, rule config
7.7. Health endpoints, graceful shutdown
7.8. Unit tests
7.9. Integration test: signal in → risk-adjusted signal out

**Exit criteria:** Adjustments auditable, blocking deterministic, original signal semantics preserved

---

### Slice 8: explainability-service
**Owner:** explain-developer  
**Estimated effort:** 2 days

8.1. Service skeleton
8.2. Domain: ExplainedSignal entity, ExplanationGenerator interface
8.3. Application: ExplainUseCase
8.4. Infrastructure: NATS consumer (`signals.risk_adjusted`), NATS publisher (`signals.explained`), Redis dedup
8.5. Explanation templates: factor-based natural language generation
8.6. PostgreSQL for explained signals
8.7. Health endpoints, graceful shutdown
8.8. Unit tests
8.9. Integration test: risk-adjusted → explained

**Exit criteria:** Explanations reference actual factors from input signal

---

### Slice 9: api-gateway-service
**Owner:** api-developer  
**Estimated effort:** 2-3 days

9.1. Service skeleton with HTTP router (chi or gin)
9.2. Domain: read models for signals, events
9.3. Application: query use cases
9.4. Infrastructure: PostgreSQL read models, Redis response cache, NATS consumers for read model updates
9.5. Endpoints: `GET /signals/latest`, `GET /signals/history`, `GET /events`, `POST /simulate`
9.6. `/simulate` — request/reply or async job submission to simulation-service
9.7. Health endpoints, graceful shutdown
9.8. Unit tests for handlers
9.9. Integration test: query endpoints return data

**Exit criteria:** All endpoints functional, caching works, graceful degradation on cache miss

---

### Slice 10: simulation-service
**Owner:** simulation-developer  
**Estimated effort:** 2-3 days

10.1. Service skeleton
10.2. Domain: Scenario, SimulationRun, SimulationResult entities
10.3. Application: RunSimulationUseCase
10.4. Infrastructure: isolated NATS subject namespace or in-memory event replay, PostgreSQL for scenarios/results
10.5. Scenario loading from config/database
10.6. Replay historical normalized events through signal/risk logic (reuse domain packages or call services)
10.7. Health endpoints, graceful shutdown
10.8. Unit tests
10.9. Integration test: run scenario, get results

**Exit criteria:** Simulations isolated, reproducible by scenario ID

---

### Slice 11: End-to-End Integration
**Owner:** integration-lead  
**Estimated effort:** 2 days

11.1. Write E2E test: inject raw event → verify explained signal appears
11.2. Write replay test: replay same events twice, verify idempotency (no duplicates)
11.3. Write API test: full flow via HTTP endpoints
11.4. Document test scenarios in `docs/testing.md`
11.5. Fix any cross-service integration issues discovered

**Exit criteria:** Full pipeline demonstrable, replay safe, API returns expected data

---

### Slice 12: Documentation and Cleanup
**Owner:** tech-writer / architect  
**Estimated effort:** 1-2 days

12.1. `docs/architecture.md` — system overview, bounded contexts, event flow diagram
12.2. `docs/contracts.md` — protobuf contract documentation
12.3. `docs/deployment.md` — Docker Compose usage, environment variables
12.4. `docs/replay.md` — replay procedures, idempotency guarantees
12.5. `README.md` — quickstart guide
12.6. Code cleanup, linting, final review prep

**Exit criteria:** Documentation complete, repository ready for review

## Reviewer focus

### Correctness
- Signal generation determinism: verify same input set + config = same output
- Event ID generation: verify UUID v5 usage is deterministic
- Dedup logic: verify Redis key structure prevents duplicates

### Contracts
- Protobuf backward compatibility: no field removals/renumbers
- Package versioning: `v1` consistently used
- Generated code compiles and imports correctly

### Architecture
- Clean Architecture adherence: no infrastructure in domain layer
- Service boundaries: no cross-service database access
- Subject ownership: publishers/consumers match documented ownership

### Reliability
- Idempotency: all handlers safe for at-least-once delivery
- Graceful shutdown: in-flight messages completed before exit
- JetStream configuration: durable consumers, explicit ack
- Dedup TTLs: appropriate for replay windows

### Security (basic review)
- No secrets in code
- Environment variable configuration
- Input validation on API endpoints

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Protobuf schema churn during development | Medium | Medium | Freeze contracts after Slice 1 review; changes require architect approval |
| Signal determinism broken by floating-point or map iteration | Medium | High | Use deterministic sorting, fixed-precision arithmetic in scoring |
| JetStream consumer misconfiguration causes message loss | Low | High | Document consumer config requirements; integration tests verify delivery |
| Redis dedup TTL too short for replay windows | Medium | Medium | Make TTL configurable; document replay time bounds |
| Cross-service integration issues discovered late | Medium | Medium | Slice 11 scheduled early enough for buffer; daily integration builds |
| Stub adapters mask real integration complexity | Medium | Low | Out of scope for this task; tracked as follow-up work |
| Market context staleness affects signal quality | Low | Medium | Health check includes context freshness; alerts if stale |
| Simulation isolation breach affects live data | Low | High | Simulation uses separate database schema; no write access to live tables |

### Rollback considerations
- All services are independently deployable; rollback individual service
- Protobuf contracts are additive; old consumers ignore new fields
- Database migrations include down migrations
- JetStream streams retain history; no data loss on service rollback
- Redis dedup keys are ephemeral; no rollback needed

### Replay considerations
- All event IDs are deterministic; replay produces same IDs
- Dedup prevents duplicate processing during replay
- Config version embedded in signal IDs; config changes during replay are detectable
- Market context snapshots are immutable; replay uses historical snapshots
- Simulation service is designed for replay; production replay uses same mechanisms
