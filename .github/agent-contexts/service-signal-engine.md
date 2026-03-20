# Service Context — signal-engine-service

## Responsibility
Aggregate normalized events, apply scoring, decay, multi-source confirmation, and emit generated trading signals.

## Consumes
- `events.normalized`
- `market.context.updated`

## Publishes
- `signals.generated`

## Invariants
- same replayed inputs + same market context snapshot => same signal outcome
- decay is time-based and explicit
- scoring model must be config-driven
