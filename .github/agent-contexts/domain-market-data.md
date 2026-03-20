# Domain Context — Event Collection and Market Context

## Includes
- collector-service
- normalizer-service
- market-context-service

## Core concepts
- raw external event
- normalized market event
- enrichment
- sentiment
- impact
- market context snapshot
- volatility / volume / price context

## Invariants
- raw events are append-only and replayable
- normalization must preserve source identity and original event references
- enrichment should be explicit and auditable
- market context updates should be timestamped and scoped to TON/USDT unless extended explicitly
