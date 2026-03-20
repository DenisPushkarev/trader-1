# Implementation Plan — GH-3

## Objective

Add a configurable `MaxSignalsPerMinute` rate limit to the signal-engine-service that controls the maximum number of signals emitted per minute. When the limit is reached, excess signals are dropped. When set to 0, rate limiting is disabled.

## Scope

- Add `MaxSignalsPerMinute int` field to signal-engine configuration struct
- Implement in-memory token-bucket or sliding-window counter for rate limiting
- Integrate rate limiter into signal emission path
- Add unit tests for rate limiting behavior
- Ensure existing tests remain passing

## Out of scope

- Protobuf contract changes (not required per task constraints)
- Queueing/buffering of excess signals (dropping is acceptable per AC)
- Distributed rate limiting across instances (in-memory is explicitly allowed)
- Metrics/observability for dropped signals (not in AC, can be follow-up)
- Persistence of rate limit state across restarts

## Bounded contexts impacted

| Context | Impact |
|---------|--------|
| signal-engine-service | Primary — owns signal generation and emission logic |
| packages/shared | Potential — if rate limiter is generic enough for reuse |

No cross-service bounded context impact. The rate limiter is internal to signal-engine-service and does not affect upstream (normalizer) or downstream (risk-engine) services' responsibilities.

## Services/packages impacted

| Path | Change Type |
|------|-------------|
| `services/signal-engine-service/internal/config/` | Add `MaxSignalsPerMinute` field |
| `services/signal-engine-service/internal/domain/` or `internal/application/` | Add rate limiter interface and implementation |
| `services/signal-engine-service/internal/infrastructure/` | Integrate rate limiter into signal publisher |
| `packages/shared/ratelimit/` | Optional — if implementing reusable rate limiter |

## NATS subjects impacted

| Subject | Impact |
|---------|--------|
| `events.normalized` | None — consumption unchanged |
| `market.context.updated` | None — consumption unchanged |
| `signals.generated` | Behavioral — emission rate may be throttled; no schema or ownership change |

**Ownership unchanged**: signal-engine-service remains the sole publisher to `signals.generated`.

## Protobuf contract impact

**None**

- No changes to protobuf message definitions
- No changes to event payloads
- The rate limit is an internal operational control, not a contract concern

## Data/storage impact

**None**

- Rate limiter state is in-memory only
- No PostgreSQL schema changes
- No Redis usage required (in-memory counter is sufficient per constraints)
- State is ephemeral and lost on restart (acceptable for this use case)

## Idempotency and replay considerations

### Replay Safety Analysis

| Concern | Assessment |
|---------|------------|
| Determinism | **Affected** — replay with rate limiting enabled may produce different outputs than original run |
| Idempotency | **Preserved** — individual signal processing remains idempotent |
| Historical reprocessing | **Conditional** — safe only if `MaxSignalsPerMinute=0` during replay |

### Mitigation Strategy

1. **Document clearly**: Rate limiting should be disabled (`MaxSignalsPerMinute=0`) during historical replay/backfill operations
2. **Config-driven**: Since the limit is config-driven, operators can set it to 0 for replay scenarios
3. **No silent mutation**: Dropped signals should be logged (debug level) so operators can detect rate limiting during replay if misconfigured

### Recommendation

Add a comment/documentation noting that for deterministic replay, rate limiting must be disabled. This preserves the platform invariant that "same replayed inputs + same market context snapshot => same signal outcome" when properly configured.

## Acceptance criteria

- [ ] New field `MaxSignalsPerMinute int` exists in signal-engine config struct
- [ ] Config field is properly loaded from environment/config file
- [ ] When `MaxSignalsPerMinute = 0`, no rate limiting is applied (all signals emitted)
- [ ] When `MaxSignalsPerMinute > 0`, signals beyond the limit within a minute are dropped
- [ ] Dropped signals are logged at debug level with event ID for traceability
- [ ] Unit test verifies rate limiting drops excess signals
- [ ] Unit test verifies `MaxSignalsPerMinute = 0` disables rate limiting
- [ ] Unit test verifies signals at or below limit are emitted normally
- [ ] All existing signal-engine unit tests pass without modification

## Developer slices

### Slice 1: Configuration Extension
**Scope**: Add config field only  
**Testable**: Yes — unit test config parsing  
**Files**:
- `services/signal-engine-service/internal/config/config.go`
- `services/signal-engine-service/internal/config/config_test.go`

**Tasks**:
1. Add `MaxSignalsPerMinute int` field to config struct
2. Add environment variable binding (e.g., `SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE`)
3. Default value = 0 (disabled)
4. Add config validation (must be >= 0)
5. Unit test: verify field loads correctly from env/config
6. Unit test: verify default value is 0
7. Unit test: verify negative values are rejected

### Slice 2: Rate Limiter Implementation
**Scope**: Implement rate limiter component  
**Testable**: Yes — unit test limiter in isolation  
**Files**:
- `services/signal-engine-service/internal/domain/ratelimiter.go` (or `internal/application/`)
- `services/signal-engine-service/internal/domain/ratelimiter_test.go`

**Tasks**:
1. Define `RateLimiter` interface:
   ```go
   type RateLimiter interface {
       Allow() bool
   }
   ```
2. Implement `SignalRateLimiter` using token bucket or sliding window counter
3. Constructor: `NewSignalRateLimiter(maxPerMinute int) RateLimiter`
4. If `maxPerMinute <= 0`, return a no-op limiter that always allows
5. Unit test: limiter allows up to N signals per minute
6. Unit test: limiter denies signals beyond N per minute
7. Unit test: limiter resets after minute boundary
8. Unit test: zero/negative maxPerMinute creates no-op limiter

### Slice 3: Integration into Signal Emission Path
**Scope**: Wire rate limiter into signal publisher  
**Testable**: Yes — unit test with mocked dependencies  
**Files**:
- `services/signal-engine-service/internal/infrastructure/publisher.go` (or equivalent)
- `services/signal-engine-service/internal/application/service.go` (or equivalent)
- Corresponding test files

**Tasks**:
1. Inject `RateLimiter` into signal publisher/emitter via constructor
2. Before publishing to `signals.generated`, call `rateLimiter.Allow()`
3. If not allowed, log dropped signal at debug level with event ID
4. If allowed, proceed with normal publication
5. Update service composition to wire config → limiter → publisher
6. Integration unit test: mock NATS, verify signals are dropped when limit exceeded
7. Integration unit test: verify existing signal emission works with limit=0
8. Verify all existing tests pass (may need to inject no-op limiter)

### Slice 4: Documentation
**Scope**: Document rate limiting behavior  
**Testable**: N/A  
**Files**:
- `services/signal-engine-service/README.md`
- `docs/` (if service-level docs exist)

**Tasks**:
1. Document `MaxSignalsPerMinute` config option
2. Document behavior when limit is reached (signals dropped)
3. Document replay consideration: set to 0 for deterministic replay
4. Add example configuration

## Reviewer focus

### Correctness
- Rate limiter correctly counts signals per minute window
- Window boundary handling is correct (no off-by-one errors)
- Zero value correctly disables rate limiting
- Dropped signals are logged with sufficient context

### Reliability
- Rate limiter is thread-safe if signal emission can be concurrent
- No goroutine leaks from timer-based implementations
- Graceful behavior under high load (no panics, no unbounded memory)

### Testing
- Unit tests cover all acceptance criteria
- Edge cases: exactly at limit, one over limit, zero limit, negative limit
- Time-based tests use injectable clock or short windows to avoid flaky tests
- Existing tests are not broken

### Architecture
- Rate limiter follows Clean Architecture (domain/application layer, not infrastructure)
- Dependency injection pattern is maintained
- Interface is used to allow future extension (e.g., distributed rate limiting)
- No global mutable state introduced

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Flaky time-based tests | Medium | Low | Use injectable clock interface in rate limiter; avoid wall-clock time in tests |
| Thread-safety issues | Low | High | Use `sync.Mutex` or atomic operations; document concurrency model |
| Replay determinism violation | Medium | Medium | Document that rate limiting must be disabled for replay; consider logging warning if enabled during replay mode (if detectable) |
| Silent signal loss in production | Medium | Medium | Ensure dropped signals are logged; consider future metrics emission |
| Token bucket implementation complexity | Low | Low | Prefer simple sliding window counter for v1; token bucket can be future enhancement |
| Breaking existing tests | Low | Medium | Inject no-op rate limiter in existing test setups; verify all tests pass before merge |
