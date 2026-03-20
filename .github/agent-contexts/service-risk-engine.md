# Service Context — risk-engine-service

## Responsibility
Filter generated signals, adjust confidence, assign risk levels, block unsafe or low-quality opportunities.

## Consumes
- `signals.generated`
- optional latest market context cache/snapshot

## Publishes
- `signals.risk_adjusted`

## Invariants
- risk adjustment is explicit and auditable
- blocking rules must be deterministic
