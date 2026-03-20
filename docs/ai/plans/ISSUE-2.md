# Implementation Plan — GH-2

## Objective

Add a basic confidence score to the signal-engine-service that quantifies signal reliability based on indicator agreement. The confidence score is calculated as the ratio of agreeing indicators to total indicators evaluated, producing a value between 0.0 and 1.0.

## Scope

- Add `confidence_score` field (float64, range 0.0–1.0) to the `SignalGenerated` protobuf message
- Implement a pure, deterministic confidence calculation function within signal-engine-service
- Integrate confidence scoring into the existing signal generation pipeline
- Unit tests covering all specified edge cases

## Out of scope

- Changes to downstream consumers (risk-engine-service, explainability-service, api-gateway-service)
- Persistence of confidence scores to PostgreSQL
- Confidence score thresholds for signal filtering
- Weighted indicator contributions (all indicators weighted equally)
- Changes to `events.normalized` or `market.context.updated` message schemas

## Bounded contexts impacted

| Bounded Context | Impact | Ownership |
|-----------------|--------|-----------|
| Signal Generation (signal-engine-service) | Primary — new field computation and emission | Owns `signals.generated` |
| Risk Adjustment (risk-engine-service) | None — will receive new field but no code changes required | Consumer only |
| Explainability (explainability-service) | None — may later use field but not in scope | Consumer only |

**Ownership clarification**: signal-engine-service is the sole owner of the `SignalGenerated` schema and the `signals.generated` subject. Downstream services consume but do not mutate.

## Services/packages impacted

| Path | Change Type | Description |
|------|-------------|-------------|
| `packages/contracts/` | Additive | Add `confidence_score` field to `SignalGenerated` message |
| `services/signal-engine-service/internal/domain/` | New | Confidence calculation pure function |
| `services/signal-engine-service/internal/application/` | Modification | Integrate confidence into signal generation use case |
| `packages/shared/` | None anticipated | No shared utilities required for this feature |

## NATS subjects impacted

| Subject | Impact | Direction | Notes |
|---------|--------|-----------|-------|
| `events.normalized` | None | Consumed | No schema or consumption logic changes |
| `market.context.updated` | None | Consumed | No schema or consumption logic changes |
| `signals.generated` | Additive | Published | New `confidence_score` field in payload |

**Ownership**: signal-engine-service owns publication to `signals.generated`. No subject ownership changes.

## Protobuf contract impact

**Classification: ADDITIVE (backward-compatible)**

```protobuf
// packages/contracts/signals/v1/signal_generated.proto
message SignalGenerated {
  // ... existing fields unchanged ...
  
  // NEW: Confidence score based on indicator agreement ratio.
  // Range: 0.0 (no agreement) to 1.0 (full agreement).
  // Added in GH-2.
  double confidence_score = <next_field_number>;
}
```

**Compatibility analysis**:
- Existing consumers ignore unknown fields (protobuf3 default behavior)
- No field renumbering or removal
- No semantic change to existing fields
- Default value (0.0) is safe for consumers that don't read the field yet

**Migration notes**: None required. Downstream services can adopt the field at their own pace.

## Data/storage impact

**None.**

- No PostgreSQL schema changes
- No Redis key structure changes
- Confidence score is computed on-the-fly from in-memory indicator state
- No new persistence requirements introduced

## Idempotency and replay considerations

**Replay-safe: YES**

| Consideration | Analysis |
|---------------|----------|
| Determinism | Confidence calculation is a pure function of indicator evaluation results; same inputs → same output |
| Time independence | No wall-clock dependency; uses only event-provided timestamps if needed |
| External state | No external API calls or database reads in confidence calculation |
| Idempotency | Signal emission idempotency unchanged; confidence is just an additional field |

**Verification approach**: Replay integration test should produce byte-identical `SignalGenerated` messages (including confidence_score) for identical input sequences.

## Acceptance criteria

- [ ] `SignalGenerated` protobuf message includes `confidence_score` field (double, field number assigned)
- [ ] Confidence formula: `confidence_score = agreeing_indicators / total_indicators`
- [ ] Function is pure: no side effects, no wall-clock reads, no external state
- [ ] Unit test: all indicators agree → confidence = 1.0
- [ ] Unit test: no indicators agree → confidence = 0.0
- [ ] Unit test: partial agreement (e.g., 2/4) → confidence = 0.5
- [ ] Unit test: single indicator → confidence = 1.0 (agrees with itself) or 0.0 (if signal direction undefined)
- [ ] Unit test: zero indicators evaluated → confidence = 0.0 (defined behavior, no division by zero)
- [ ] All existing signal-engine-service tests pass without modification
- [ ] Protobuf contract compiles and is backward-compatible

## Developer slices

### Slice 1: Protobuf contract update
**Scope**: `packages/contracts/`  
**Effort**: Small  
**Dependencies**: None

1. Add `confidence_score` field to `SignalGenerated` in `packages/contracts/signals/v1/signal_generated.proto`
2. Assign next available field number
3. Add field documentation comment explaining semantics and range
4. Run `make proto` or equivalent to regenerate Go bindings
5. Verify compilation succeeds

**Exit criteria**: Generated Go code compiles; no existing tests broken.

---

### Slice 2: Domain layer — confidence calculation function
**Scope**: `services/signal-engine-service/internal/domain/`  
**Effort**: Small  
**Dependencies**: None (can parallel with Slice 1)

1. Create `confidence.go` (or add to existing scoring module)
2. Implement pure function:
   ```go
   func CalculateConfidence(agreeing, total int) float64
   ```
3. Handle edge cases:
   - `total == 0` → return `0.0`
   - `agreeing > total` → clamp or error (define behavior)
4. No dependencies on external packages beyond stdlib

**Exit criteria**: Function implemented with clear documentation.

---

### Slice 3: Unit tests for confidence calculation
**Scope**: `services/signal-engine-service/internal/domain/`  
**Effort**: Small  
**Dependencies**: Slice 2

1. Create `confidence_test.go`
2. Table-driven tests covering:
   - `(4, 4)` → `1.0` (all agree)
   - `(0, 4)` → `0.0` (none agree)
   - `(2, 4)` → `0.5` (partial)
   - `(1, 1)` → `1.0` (single indicator)
   - `(0, 0)` → `0.0` (no indicators)
3. Test determinism: same inputs always produce same output
4. Run `go test -race` to verify no concurrency issues

**Exit criteria**: All unit tests pass; coverage on confidence function ≥ 95%.

---

### Slice 4: Application layer integration
**Scope**: `services/signal-engine-service/internal/application/`  
**Effort**: Medium  
**Dependencies**: Slice 1, Slice 2

1. Identify signal generation use case / handler
2. Determine where indicator evaluation results are available
3. Call `CalculateConfidence(agreeing, total)` after indicator evaluation
4. Populate `confidence_score` field in `SignalGenerated` message before publishing
5. Ensure no mutation of existing signal fields

**Exit criteria**: Confidence score populated in all emitted signals.

---

### Slice 5: Integration verification
**Scope**: `services/signal-engine-service/`  
**Effort**: Small  
**Dependencies**: Slice 4

1. Run all existing signal-engine-service tests
2. Verify no regressions
3. Add or update integration test to verify `confidence_score` is present and correct in emitted messages
4. Verify replay produces deterministic confidence values

**Exit criteria**: All tests green; CI passes.

## Reviewer focus

| Reviewer | Focus Areas |
|----------|-------------|
| reviewer-trading-logic | Correctness of confidence formula; edge case handling; determinism guarantee |
| reviewer-architecture | Protobuf evolution safety; no unintended coupling; clean architecture adherence |

**Specific review checkpoints**:

- **Correctness**: Verify `agreeing/total` logic matches documented indicator agreement semantics
- **Contracts**: Confirm field number doesn't conflict; documentation present; backward-compatible
- **Architecture**: Confidence function in domain layer (not application); no infrastructure dependencies
- **Reliability**: No panics on edge cases; no floating-point precision issues affecting equality checks

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Ambiguous "agreeing" definition | Medium | Medium | Document clearly: agreeing = indicator direction matches signal direction |
| Division by zero | Low | High | Explicit guard: `if total == 0 { return 0.0 }` |
| Floating-point comparison issues in tests | Low | Low | Use tolerance-based comparison (`math.Abs(got-want) < epsilon`) |
| Field number collision in protobuf | Low | High | Check existing `.proto` file for next available number before PR |
| Downstream consumer breaks on new field | Very Low | Low | Protobuf3 ignores unknown fields; no action needed |
| Confidence semantics unclear for single indicator | Medium | Low | Define: single indicator always agrees with itself → 1.0 if signal emitted |

**No blocking risks identified. Proceed with implementation.**
