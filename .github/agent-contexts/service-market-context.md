# Service Context — market-context-service

## Responsibility
Produce price, volume, volatility, and other market context relevant to TON/USDT.

## Publishes
- `market.context.updated`

## Stores
- PostgreSQL for historical snapshots
- Redis for latest context cache
