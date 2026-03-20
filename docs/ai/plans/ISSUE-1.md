# Implementation Plan — GH-1

## Objective

Add a basic signal confidence score to the signal-engine-service. The confidence score quantifies the level of agreement among indicators contributing to a generated signal, expressed as a ratio of agreeing indicators to total indicators evaluated.

## Scope

- Extend the signal generation logic in `signal-engine-service` to compute a confidence score
- Add `confidence_score` field to the `SignalGenerated` protobuf message (additive change)
- Implement pure, deterministic confidence calculation function
- Add comprehensive unit tests for confidence score computation
- Ensure replay safety by deriving score solely from input state (no external dependencies like wall-clock time)

## Out of scope

- Changes to `risk-engine-service`, `explainability-service`, or downstream consumers
- Modifications to `events.normalized` or `market.context.updated` message schemas
- Confidence score weighting or indicator prioritization (future enhancement)
- UI/API exposure of confidence score (handled by `api-gateway-service` separately)
- Persistence of confidence scores to PostgreSQL (signals are event-sourced)

## Bounded contexts impacted

| Bounded Context | Service | Impact |
|-----------------|---------|--------|
| Signal Generation | signal-engine-service | **Primary** — owns confidence score computation and emission |
| Risk Adjustment | risk-engine-service | **Read-only consumer** — will receive new field, no code change required |
| Explainability | explainability-service | **Read-only consumer** — may reference confidence in future, no change now |

**Ownership clarity:**
- `signal-engine-service` owns the `signals.generated` subject and the `SignalGenerated` message schema
- Confidence score is a signal-engine concern; downstream services may consume but do not define it

## Services/packages impacted

| Path | Change Type | Description |
|------|-------------|-------------|
| `services/signal-engine-service/internal/domain/` | Modify | Add confidence score domain logic |
| `services/signal-engine-service/internal/application/` | Modify | Integrate confidence calculation into signal generation use case |
| `packages/contracts/proto/signals/v1/` | Modify | Add `confidence_score` field to `SignalGenerated` message |
| `packages/contracts/` | Regenerate | Regenerate Go bindings after proto change |
| `services/signal-engine-service/internal/domain/` | Add | Unit tests for confidence calculation |

## NATS subjects impacted

| Subject | Ownership | Impact |
|---------|-----------|--------|
| `events.normalized` | normalizer-service | **Read-only** — no schema or semantic change |
| `market.context.updated` | market-context-service | **Read-only** — no schema or semantic change |
| `signals.generated` | signal-engine-service | **Write** — payload gains new optional field `confidence_score` |

**No subject ownership changes.** The `signals.generated` subject semantics remain unchanged; we are enriching the payload additively.

## Protobuf contract impact

**Classification: Additive (non-breaking)**

```protobuf
// packages/contracts/proto/signals/v1/signal.proto

message SignalGenerated {
  // ... existing fields unchanged ...
  
  // NEW: Confidence score representing indicator agreement ratio.
  // Range: 0.0 (no agreement) to 1.0 (full agreement).
  // Field is optional for backward compatibility; older consumers ignore it.
  double confidence_score = <next_field_number>;
}
```

**Backward compatibility analysis:**
- New field uses next available field number (no reuse of deprecated numbers)
- Field is optional (proto3 default behavior)
- Existing consumers will ignore unrecognized fields (standard protobuf behavior)
- No field renames or semantic changes to existing fields
- Wire format remains compatible in both directions

**Migration notes:** None required. Downstream services will begin receiving the field immediately; they may consume it when ready.

## Data/storage impact

**None.**

- Confidence score is computed at signal generation time and emitted in the event payload
- No new database tables or columns required
- No Redis key changes (deduplication keys remain based on existing event ID strategy)
- Score is fully derived from in-memory indicator state; no persistence needed

## Idempotency and replay considerations

**Replay safety: SAFE ✓**

| Consideration | Analysis |
|---------------|----------|
| Determinism | Confidence score = `agreeing_indicators / total_indicators`. Both values are derived from the replayed `events.normalized` stream and the `market.context.updated` snapshot. No randomness, no wall-clock dependency. |
| Input stability | The same historical events + same market context snapshot produce identical indicator evaluations, therefore identical confidence scores. |
| Ordering sensitivity | Confidence calculation occurs at signal emission time, after indicator aggregation. No ordering assumptions beyond existing signal-engine invariants. |
| Deduplication | Existing Redis-based deduplication (keyed by event ID with TTL) remains valid. Confidence score does not alter event identity. |

**Invariant preserved:** Same replayed inputs + same market context snapshot ⇒ same signal outcome (including confidence score).

## Acceptance criteria

- [ ] `signals.generated` NATS message includes `confidence_score` field (type `double`, range 0.0–1.0)
- [ ] Score is derived from indicator agreement ratio: `agreeing_indicators / total_indicators`
- [ ] Unit tests cover edge cases:
  - All indicators agree → score = 1.0
  - Half indicators agree → score = 0.5
  - No indicators agree → score = 0.0
  - Single indicator (boundary) → score = 1.0 or 0.0 depending on agreement
  - Zero indicators (defensive) → score = 0.0 (or explicit handling)
- [ ] Protobuf change is additive only; no existing field modifications
- [ ] Replay of historical `events.normalized` stream produces identical confidence scores for identical input conditions
- [ ] Structured logging includes `confidence_score` in signal generation log entries

## Developer slices

### Slice 1: Protobuf contract update
**Scope:** `packages/contracts/proto/signals/v1/signal.proto`, `packages/contracts/`

1. Add `confidence_score` field to `SignalGenerated` message with appropriate field number
2. Add field documentation comment explaining semantics and range
3. Regenerate Go bindings (`make proto` or equivalent)
4. Verify generated code compiles and existing tests pass

**Exit criteria:** Proto change committed, bindings regenerated, CI green.

**Estimated complexity:** Low

---

### Slice 2: Domain logic for confidence calculation
**Scope:** `services/signal-engine-service/internal/domain/`

1. Create `confidence.go` (or extend existing scoring module) with pure function:
   ```go
   func CalculateConfidenceScore(agreeingCount, totalCount int) float64
   ```
2. Handle edge cases:
   - `totalCount == 0` → return `0.0` (defensive, logged as warning)
   - Normal case → `float64(agreeingCount) / float64(totalCount)`
3. Add comprehensive unit tests in `confidence_test.go`:
   - Table-driven tests for all acceptance criteria scenarios
   - Property: result always in [0.0, 1.0]
   - Property: `agreeingCount > totalCount` returns error or clamps (define behavior)

**Exit criteria:** Domain function implemented with 100% test coverage on specified scenarios.

**Estimated complexity:** Low

---

### Slice 3: Application layer integration
**Scope:** `services/signal-engine-service/internal/application/`

1. Modify signal generation use case to:
   - Track agreeing vs. total indicators during aggregation
   - Call `CalculateConfidenceScore` after indicator evaluation
   - Populate `ConfidenceScore` field in `SignalGenerated` protobuf message
2. Add structured log field for confidence score at signal emission
3. Update existing unit tests for signal generation to assert `confidence_score` is set correctly

**Exit criteria:** Integration complete, existing tests updated, new field populated in emitted signals.

**Estimated complexity:** Medium

---

### Slice 4: Integration verification
**Scope:** `services/signal-engine-service/`

1. Manual or scripted integration test:
   - Publish known `events.normalized` messages to local NATS
   - Capture emitted `signals.generated` messages
   - Verify `confidence_score` field present and correct
2. Replay test:
   - Replay same events twice
   - Assert identical `confidence_score` values

**Exit criteria:** Integration test passes, replay determinism verified.

**Estimated complexity:** Low

## Reviewer focus

| Reviewer | Focus Areas |
|----------|-------------|
| **reviewer-trading-logic** | Correctness of confidence calculation; edge case handling; indicator agreement semantics; determinism for replay |
| **reviewer-architecture** | Proto field placement and numbering; backward compatibility; domain/application layer separation; no scope creep into downstream services |

**Specific review checkpoints:**
- [ ] Proto field number does not conflict with reserved/deprecated numbers
- [ ] No modification to existing proto fields
- [ ] Confidence calculation is pure (no side effects, no I/O, no time dependency)
- [ ] Unit tests cover all acceptance criteria scenarios
- [ ] Logging follows structured logging conventions with correlation IDs
- [ ] No changes outside allowed paths

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Downstream services fail to ignore new field | Very Low | Low | Protobuf guarantees unknown field handling. No action required, but document in release notes. |
| Indicator count tracking introduces complexity | Low | Medium | Keep tracking logic minimal; use existing aggregation loop. Review for coupling. |
| Zero-indicator edge case causes division by zero | Low | High | Explicit check with defensive return of 0.0 and warning log. |
| Confidence score semantics misunderstood by consumers | Medium | Medium | Document field semantics in proto comment. Add to API documentation in follow-up task. |
| Proto field number collision on merge | Low | Medium | Coordinate with any parallel proto changes. CI should catch duplicate field numbers. |

**No blocking risks identified.** All mitigations are standard engineering practices already in place.
