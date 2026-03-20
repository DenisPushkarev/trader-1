# Service Context — normalizer-service

## Responsibility
Transform raw external events into unified normalized protobuf events with enrichment such as sentiment and impact.

## Consumes
- `events.raw`

## Publishes
- `events.normalized`

## Invariants
- preserve original source references
- enrichment must be explicit and reproducible
