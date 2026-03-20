# Platform Global Context — TON Trading Platform

## System goal
Build an event-driven trading signal platform for TON/USDT as a Go monorepo of independent deployable microservices communicating through NATS Core + JetStream using protobuf contracts.

## Architectural constraints
- Communication contract is protobuf only.
- Inter-service communication is event-driven through NATS subjects.
- Persistence: PostgreSQL for durable relational state, Redis for cache and ephemeral dedup state.
- Services must be horizontally scalable and idempotent.
- Delivery expectation for JetStream consumers is at-least-once.
- Replay support must be preserved; do not introduce logic that makes historical reprocessing unsafe.
- No global mutable state.
- Use `context.Context` across application and infrastructure boundaries.
- Use constructor-based dependency injection only.
- Clean Architecture preferred: domain / application / infrastructure.

## Mandatory NATS subjects
- `events.raw`
- `events.normalized`
- `signals.generated`
- `signals.risk_adjusted`
- `signals.explained`
- `market.context.updated`

## Mandatory services
- collector-service
- normalizer-service
- signal-engine-service
- risk-engine-service
- market-context-service
- explainability-service
- api-gateway-service
- simulation-service

## Contract rules
- protobuf package versioning must use `v1`.
- Backward compatibility required for event payload evolution.
- Do not rename or repurpose existing fields silently.
- Avoid breaking subject semantics without explicit migration notes.

## Reliability rules
- All event handlers must be idempotent.
- Deduplication may use Redis keys with TTL keyed by logical event id.
- Retries must be safe.
- JetStream durable consumers must be configured deterministically.
- Preserve ordering assumptions only where explicitly documented; do not rely on incidental delivery order.

## Observability rules
- structured logging via zap or zerolog
- liveness and readiness endpoints required
- graceful shutdown required
- log correlation IDs / event IDs when available

## Repository layout
- `services/<service-name>`
- `packages/contracts`
- `packages/shared`
- `infrastructure/docker`
- `docs/`

## Agent operating model
The agent receives only layered context relevant to the task:
1. platform global context
2. domain context
3. service context
4. task packet
5. impacted code and contracts

The agent must not assume ownership outside the supplied scope.
