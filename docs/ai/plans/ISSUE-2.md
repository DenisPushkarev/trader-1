# Implementation Plan — GH-2

## Objective

Add a basic signal confidence score to the signal-engine-service that quantifies the level of agreement among evaluated indicators. The confidence score is computed as the ratio of agreeing indicators to total indicators evaluated, producing a value in the range [0.0, 1.0].

## Scope

- Add `confidence_score` field (float64) to the `SignalGenerated` protobuf message
- Implement a pure, deterministic confidence calculation function in the signal-engine-service domain layer
- Integrate confidence calculation into the signal generation pipeline
- Add comprehensive unit tests for the confidence calculation logic
- Ensure backward compatibility with existing consumers of `signals.generated`

## Out of scope

- Changes to risk-engine-service, explainability-service, or any downstream consumers
- Modifications to indicator evaluation logic itself
- Persistence of confidence scores to database
- UI/API exposure of confidence scores (api-gateway-service changes)
- Confidence score decay or time-based adjustments
- Weighted indicator confidence (all indicators treated equally)

## Bounded contexts impacted

| Bounded Context | Service | Impact |
|-----------------|---------|--------|
| Signal Generation | signal-engine-service | **Primary** — owns signal scoring and emission |
| Contracts | packages/contracts | **Additive** — new field in SignalGenerated |

No ownership changes. signal-engine-service remains the sole producer of `signals.generated`. Downstream consumers (risk-engine-service, explainability-service) will receive the new field but are not required to act on it immediately.

## Services/packages impacted

| Path | Change Type | Description |
|------|-------------|-------------|
| `packages/contracts/proto/signals/v1/signal.proto` | Additive | Add `confidence_score` field to `SignalGenerated` |
| `packages/contracts/` | Regenerate | Regenerate Go code from updated proto |
| `services/signal-engine-service/internal/domain/` | New file | Pure confidence calculation function |
| `services/signal-engine-service/internal/application/` | Modify | Integrate confidence into signal generation use case |
| `services/signal-engine-service/internal/domain/` | New test file | Unit tests for confidence calculation |

## NATS subjects impacted

| Subject | Ownership | Read/Write | Impact |
|---------|-----------|------------|--------|
| `events.normalized` | normalizer-service | Read | No change — input unchanged |
| `market.context.updated` | market-context-service | Read | No change — input unchanged |
| `signals.generated` | signal-engine-service | Write | **Additive** — payload gains `confidence_score` field |

No subject ownership changes. No new subjects introduced. Existing consumers will receive messages with the new optional field; protobuf wire format ensures backward compatibility.

## Protobuf contract impact

**Additive (backward-compatible)**

- New field `confidence_score` added to `SignalGenerated` message with a new field number
- Field type: `double` (maps to Go `float64`)
- Default value: `0.0` (protobuf default for unset double)
- Existing consumers that do not read this field will continue to function
- No field renames, removals, or type changes

```protobuf
// Additive change — new field number (next available)
message SignalGenerated {
  // ... existing fields ...
  double confidence_score = N; // Range [0.0, 1.0]
}
```

**Migration notes:** None required. Purely additive. Old producers (during rollback) will emit messages without the field; consumers will see default `0.0`.

## Data/storage impact

**None**

- No database schema changes
- No new Redis keys
- No persistence of confidence scores (computed on-the-fly)
- Confidence is derived data, not stored state

## Idempotency and replay considerations

**Replay-safe: YES**

| Concern | Assessment |
|---------|------------|
| Determinism | Confidence calculation is a pure function of indicator evaluation results; same inputs yield same output |
| Time dependency | None — no wall-clock, no timestamps in calculation |
| External state | None — no database reads, no Redis lookups in calculation |
| Side effects | None — calculation does not mutate state |
| Ordering | Not sensitive — confidence is computed per-signal, not across signals |

**Replay guarantee:** Given identical `events.normalized` stream and `market.context.updated` snapshots, the signal-engine-service will produce identical `confidence_score` values. Historical reprocessing is safe.

## Acceptance criteria

- [ ] `SignalGenerated` protobuf message contains new field `confidence_score` of type `double`
- [ ] Confidence calculated as: `agreeing_indicators / total_indicators_evaluated`
- [ ] Confidence function is pure: no side effects, no wall-clock reads, no external I/O
- [ ] Confidence function is deterministic: same inputs always produce same output
- [ ] Unit test: all indicators agree → confidence = 1.0
- [ ] Unit test: no indicators agree → confidence = 0.0
- [ ] Unit test: partial agreement (e.g., 2/4) → confidence = 0.5
- [ ] Unit test: single indicator agreeing → confidence = 1.0
- [ ] Unit test: single indicator not agreeing → confidence = 0.0
- [ ] Unit test: edge case with zero indicators → defined behavior (suggest 0.0 with documentation)
- [ ] All existing signal-engine-service tests pass without modification
- [ ] Generated Go code compiles without errors

## Developer slices

### Slice 1: Protobuf contract update
**Scope:** `packages/contracts/`
**Estimated effort:** Small
**Dependencies:** None

1. Add `confidence_score` field to `SignalGenerated` in `proto/signals/v1/signal.proto`
2. Assign next available field number
3. Add field documentation comment
4. Regenerate Go code (`make proto-gen` or equivalent)
5. Verify generated code compiles
6. Commit with message: `feat(contracts): add confidence_score to SignalGenerated`

**Testable independently:** Yes — compile check

---

### Slice 2: Domain function for confidence calculation
**Scope:** `services/signal-engine-service/internal/domain/`
**Estimated effort:** Small
**Dependencies:** None (can parallel with Slice 1)

1. Create `confidence.go` with pure function:
   ```go
   func CalculateConfidence(agreeing, total int) float64
   ```
2. Handle edge cases:
   - `total == 0` → return `0.0` (documented behavior)
   - `agreeing > total` → clamp or error (defensive)
   - Negative inputs → error or clamp to 0
3. Ensure no external dependencies (no context, no I/O)
4. Create `confidence_test.go` with table-driven tests:
   - `(4, 4) → 1.0`
   - `(0, 4) → 0.0`
   - `(2, 4) → 0.5`
   - `(1, 1) → 1.0`
   - `(0, 1) → 0.0`
   - `(0, 0) → 0.0`
5. Commit with message: `feat(signal-engine): add confidence calculation domain function`

**Testable independently:** Yes — unit tests

---

### Slice 3: Integrate confidence into signal generation
**Scope:** `services/signal-engine-service/internal/application/`
**Estimated effort:** Medium
**Dependencies:** Slice 1, Slice 2

1. Identify where signal scoring aggregates indicator results
2. Extract or expose counts: `agreeing_indicators`, `total_indicators`
3. Call `domain.CalculateConfidence(agreeing, total)`
4. Populate `ConfidenceScore` field on `SignalGenerated` before publishing
5. Ensure integration does not break existing signal flow
6. Run existing tests — all must pass
7. Add integration-level test if signal generation has such coverage
8. Commit with message: `feat(signal-engine): integrate confidence score into signal generation`

**Testable independently:** Yes — existing tests + new integration assertion

---

### Slice 4: Final validation and documentation
**Scope:** `services/signal-engine-service/`, `docs/`
**Estimated effort:** Small
**Dependencies:** Slice 3

1. Run full test suite for signal-engine-service
2. Verify no regressions
3. Add brief documentation in service README or inline comments explaining confidence semantics
4. Update `docs/` if architectural decision records are maintained
5. Final commit with message: `docs(signal-engine): document confidence score calculation`

**Testable independently:** Yes — CI/CD pipeline

## Reviewer focus

| Reviewer | Focus Areas |
|----------|-------------|
| **reviewer-trading-logic** | Correctness of confidence formula; edge case handling (zero indicators, invalid inputs); determinism guarantee; test coverage completeness |
| **reviewer-architecture** | Protobuf backward compatibility; no breaking changes to contract; proper layer separation (domain vs application); no coupling leakage |

### Specific review checklist
- [ ] Protobuf field number does not conflict with existing or reserved numbers
- [ ] No `time.Now()` or similar in confidence calculation path
- [ ] No database/Redis calls in confidence calculation path
- [ ] Function signature is pure (inputs → output, no side effects)
- [ ] Edge case `total == 0` is handled explicitly and documented
- [ ] Test coverage includes boundary conditions
- [ ] Existing tests unmodified and passing
- [ ] No changes outside allowed paths

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Field number collision in protobuf | Low | High | Review existing proto; use next sequential number; CI lint for proto |
| Downstream services fail on new field | Very Low | Medium | Protobuf guarantees forward compatibility; field is optional with default 0.0 |
| Division by zero in confidence calculation | Medium | High | Explicit guard: `if total == 0 return 0.0`; unit test coverage |
| Indicator count extraction breaks encapsulation | Medium | Medium | Review signal scoring code; may require minor refactor to expose counts cleanly |
| Performance regression from added calculation | Very Low | Low | Calculation is O(1) arithmetic; negligible overhead |
| Rollback scenario leaves inconsistent data | Low | Low | No persistence; rolled-back version simply omits field; consumers see default 0.0 |

### Rollback plan
1. Revert signal-engine-service deployment to previous version
2. Previous version does not populate `confidence_score`
3. Consumers receive `0.0` (protobuf default) — no crashes
4. Protobuf field remains in contract (do not remove to preserve wire compatibility)
5. No data migration required
