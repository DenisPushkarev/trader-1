# Service Context — simulation-service

## Responsibility
Run scenario-based simulations over bullish, bearish, fake hype, and conflicting signal sets.

## Inputs
- historical normalized events
- market context snapshots
- strategy/risk configuration

## Invariants
- isolated from live production state
- reproducible by scenario ID/config snapshot
