# Service Context — collector-service

## Responsibility
Collect external events from Telegram, Twitter/X, exchange listings, RSS, on-chain stubs, and mini app sources; publish raw events.

## Owns
- source adapters
- source polling/webhook normalization into raw envelope
- dedup before publishing when source event IDs are available

## Publishes
- `events.raw`

## Stores
- PostgreSQL for source config / cursor state if needed
- Redis for short-lived dedup / rate limiting / cursor cache
