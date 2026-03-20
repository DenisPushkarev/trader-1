# Implementation Plan — GH-2

## Objective

Add a basic signal confidence score to the signal-engine-service that quantifies the level of agreement among evaluated indicators. The confidence score is calculated as the ratio of agreeing indicators to total indicators evaluated, producing a value in the range [0.0, 1.0].

## Scope

- Add `confidence_score` field (float64/double) to the `SignalGenerated` protobuf message
- Implement a pure, deterministic confidence calculation function within signal-engine-service
- Integrate confidence calculation into the signal generation pipeline
- Add comprehensive unit tests for the confidence calculation logic
- Ensure backward compatibility of the protobuf contract

## Out of scope

- Changes to downstream consumers (risk-engine-service, explainability-service, api-gateway-service)
- Modifications to `events.normalized` or `market.context.updated` message schemas
- Persistence of confidence scores to database
- Configuration-driven weighting of indicators (future enhancement)
- UI/API exposure of confidence scores (handled by downstream services)

## Bounded contexts impacted

| Bounded Context | Service | Impact |
|-----------------|---------|--------|
| Signal Generation | signal-engine-service | **Primary** — owns confidence calculation logic and emits enriched signal |
| Risk Adjustment | risk-engine-service | **Read-only consumer** — will receive new field, no code changes required (additive) |
| Explainability | explainability-service | **Read-only consumer** — may utilize new field in future, no immediate changes |
| Simulation | simulation-service | **Read-only consumer** — will observe new field in simulated signals |

Ownership boundaries remain unchanged. The `signals.generated` subject continues to be owned exclusively by signal-engine-service.

## Services/packages impacted

| Path | Change Type | Description |
|------|-------------|-------------|
| `packages/contracts/` | Additive | Add `confidence_score` field to `SignalGenerated` message |
| `services/signal-engine-service/internal/domain/` | New | Confidence calculation pure function |
| `services/signal-engine-service/internal/application/` | Modification | Integrate confidence into signal generation use case |
| `packages/shared/` | None | No changes anticipated |

## NATS subjects impacted

| Subject | Ownership | Impact |
|---------|-----------|--------|
| `events.normalized` | normalizer-service | **No change** — read-only consumption continues |
| `market.context.updated` | market-context-service | **No change** — read-only consumption continues |
| `signals.generated` | signal-engine-service | **Additive payload change** — new field `confidence_score` added to published messages |

No subject semantics change. No new subjects introduced. No ownership transfer.

## Protobuf contract impact

**Classification: ADDITIVE (backward-compatible)**

```protobuf
// packages/contracts/signals/v1/signal_generated.proto
message SignalGenerated {
  // ... existing fields ...
  
  // NEW: Confidence score representing indicator agreement ratio.
  // Range: [0.0, 1.0] where 1.0 = all indicators agree, 0.0 = no agreement.
  // Added in GH-2.
  double confidence_score = <next_field_number>;
}
```

**Compatibility analysis:**
- Existing consumers will ignore the new field (protobuf default behavior)
- No field renumbering or type changes to existing fields
- No semantic change to existing fields
- Wire format remains backward-compatible
- Replay of historical events (without `confidence_score`) will deserialize with default value `0.0`

**Migration notes:**
- Downstream services should treat `confidence_score == 0.0` as either "not computed" (historical) or "zero agreement" — context from signal timestamp can disambiguate if needed
- No coordinated deployment required; services can upgrade independently

## Data/storage impact

**None.**

- No new database tables or columns required
- No Redis key schema changes
- Confidence score is computed on-the-fly and emitted in the event payload
- If future persistence is needed, it will be a separate task

## Idempotency and replay considerations

**Replay safety: PRESERVED**

| Aspect | Assessment |
|--------|------------|
| Determinism | Confidence calculation is pure: `f(indicators) → score`. Same input indicators yield identical score. |
| Time independence | No wall-clock dependency. No `time.Now()` in calculation path. |
| Side effects | None. Calculation does not mutate state or trigger external calls. |
| Historical replay | Safe. Replaying `events.normalized` with same `market.context.updated` snapshot produces identical `signals.generated` including `confidence_score`. |
| Deduplication | Existing dedup strategy (Redis key by event ID) remains valid. |

**Invariant preserved:** Same replayed inputs + same market context snapshot ⇒ same signal outcome (including confidence).

## Acceptance criteria

- [ ] `SignalGenerated` protobuf message contains new field `confidence_score` of type `double`
- [ ] Field is assigned the next available field number with no renumbering of existing fields
- [ ] Confidence calculation implemented as: `confidence_score = agreeing_indicators / total_indicators`
- [ ] Calculation function is pure — no side effects, no wall-clock reads, no I/O
- [ ] Calculation function is deterministic — same inputs always produce same output
- [ ] Unit test: all indicators agree → `confidence_score = 1.0`
- [ ] Unit test: no indicators agree → `confidence_score = 0.0`
- [ ] Unit test: partial agreement (e.g., 2/4) → `confidence_score = 0.5`
- [ ] Unit test: single indicator evaluated → `confidence_score = 1.0` (if agrees) or `0.0` (if not)
- [ ] Edge case: zero indicators evaluated → handled gracefully (define behavior: return 0.0 or error)
- [ ] All existing signal-engine-service tests pass without modification (unless they explicitly check message structure)
- [ ] Protobuf regeneration successful with no breaking changes detected

## Developer slices

### Slice 1: Protobuf contract update (packages/contracts)

**Goal:** Add `confidence_score` field to `SignalGenerated` message.

**Tasks:**
1. Open `packages/contracts/signals/v1/signal_generated.proto`
2. Identify next available field number
3. Add field: `double confidence_score = N;`
4. Add field comment documenting semantics, range, and GH-2 reference
5. Run protobuf code generation (`make proto` or equivalent)
6. Verify generated Go code compiles
7. Commit with message: `proto: add confidence_score to SignalGenerated (GH-2)`

**Verification:** `go build ./...` succeeds, no existing tests break.

**Estimated complexity:** Low

---

### Slice 2: Domain logic — confidence calculation function (signal-engine-service)

**Goal:** Implement pure confidence calculation function in domain layer.

**Tasks:**
1. Create file: `services/signal-engine-service/internal/domain/confidence.go`
2. Define types if needed:
   ```go
   type IndicatorResult struct {
       Name    string
       Agrees  bool // true if indicator supports signal direction
   }
   ```
3. Implement function:
   ```go
   func CalculateConfidence(results []IndicatorResult) float64
   ```
4. Handle edge cases:
   - Empty slice → return `0.0` (document this decision)
   - All agree → `1.0`
   - None agree → `0.0`
5. Ensure no dependencies on time, I/O, or external state
6. Commit with message: `domain: add CalculateConfidence function (GH-2)`

**Verification:** Function compiles, ready for unit tests.

**Estimated complexity:** Low

---

### Slice 3: Unit tests for confidence calculation (signal-engine-service)

**Goal:** Comprehensive test coverage for `CalculateConfidence`.

**Tasks:**
1. Create file: `services/signal-engine-service/internal/domain/confidence_test.go`
2. Implement table-driven tests:
   ```go
   func TestCalculateConfidence(t *testing.T) {
       tests := []struct {
           name     string
           results  []IndicatorResult
           expected float64
       }{
           {"all agree", [...], 1.0},
           {"none agree", [...], 0.0},
           {"partial 2/4", [...], 0.5},
           {"single agree", [...], 1.0},
           {"single disagree", [...], 0.0},
           {"empty", [], 0.0},
       }
       // ...
   }
   ```
3. Use tolerance-based float comparison (e.g., `math.Abs(got-want) < 1e-9`)
4. Commit with message: `test: add CalculateConfidence unit tests (GH-2)`

**Verification:** `go test ./services/signal-engine-service/internal/domain/...` passes.

**Estimated complexity:** Low

---

### Slice 4: Integration into signal generation pipeline (signal-engine-service)

**Goal:** Wire confidence calculation into the application layer where signals are generated.

**Tasks:**
1. Locate signal generation use case (likely `internal/application/` or `internal/service/`)
2. Identify where `SignalGenerated` message is constructed
3. Collect indicator evaluation results into `[]IndicatorResult` structure
4. Call `domain.CalculateConfidence(results)`
5. Assign result to `SignalGenerated.ConfidenceScore` field
6. Ensure no introduction of time-dependent or non-deterministic behavior
7. Commit with message: `app: integrate confidence_score into signal generation (GH-2)`

**Verification:** Service compiles, existing tests pass.

**Estimated complexity:** Medium (requires understanding existing signal generation flow)

---

### Slice 5: Integration test validation (signal-engine-service)

**Goal:** Verify end-to-end signal generation includes confidence score.

**Tasks:**
1. Review existing integration/e2e tests for signal-engine-service
2. Add or extend test case that:
   - Publishes known `events.normalized` messages
   - Sets up known `market.context.updated` state
   - Consumes resulting `signals.generated`
   - Asserts `confidence_score` is present and correct
3. If no integration tests exist, document this as a gap (do not block PR)
4. Commit with message: `test: validate confidence_score in signal output (GH-2)`

**Verification:** Integration tests pass, or gap documented.

**Estimated complexity:** Low–Medium

---

### Slice 6: Documentation and final verification

**Goal:** Update documentation and perform final validation.

**Tasks:**
1. Update service README if it documents output schema
2. Add inline code comments explaining confidence calculation semantics
3. Run full test suite: `go test ./services/signal-engine-service/...`
4. Run linter: `golangci-lint run ./services/signal-engine-service/...`
5. Verify protobuf backward compatibility (no field renumbering)
6. Commit with message: `docs: document confidence_score feature (GH-2)`

**Verification:** All checks pass, PR ready for review.

**Estimated complexity:** Low

## Reviewer focus

| Reviewer | Focus Areas |
|----------|-------------|
| reviewer-trading-logic | **Correctness** — Verify confidence formula matches acceptance criteria. Validate edge case handling (empty indicators, single indicator). Confirm determinism and purity of calculation. |
| reviewer-architecture | **Contracts** — Verify protobuf change is additive only. Confirm field number assignment is correct. Validate no breaking changes to existing fields. **Architecture** — Confirm domain/application layer separation. Verify no inappropriate coupling introduced. |
| Both | **Reliability** — Ensure replay safety preserved. Verify no wall-clock or side-effect dependencies. Confirm existing tests unaffected. **Testing** — Validate test coverage meets acceptance criteria. Check edge cases are covered. |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Protobuf field number collision | Low | High | Verify next available field number before implementation. Use `protoc` validation. |
| Downstream services fail on new field | Very Low | Medium | Protobuf guarantees backward compatibility for additive changes. No mitigation needed, but monitor after deployment. |
| Indicator agreement semantics unclear | Medium | Medium | Clarify with domain expert: what constitutes "agreement"? Document decision in code comments. |
| Division by zero (zero indicators) | Low | Medium | Explicitly handle in `CalculateConfidence` — return `0.0` and document. |
| Float precision issues in tests | Low | Low | Use tolerance-based comparison in unit tests. |
| Existing tests assume exact message structure | Low | Low | Review existing tests; update assertions if they explicitly check for absence of `confidence_score`. |
| Signal generation flow harder to modify than expected | Medium | Low | Slice 4 may require refactoring. If significant, split into sub-slices and document deviations. |

**Rollback strategy:** If issues discovered post-deployment, confidence calculation can be disabled by setting `confidence_score = 0.0` unconditionally. Downstream consumers already handle this value. Full rollback requires re-deploying previous signal-engine-service version — no data migration needed.
