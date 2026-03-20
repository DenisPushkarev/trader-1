# Domain Context — Trading Logic

## Includes
- signal-engine-service
- risk-engine-service
- explainability-service
- simulation-service
- api-gateway-service

## Core concepts
- normalized event
- bullish/bearish score
- signal decay
- multi-source confirmation
- confidence score
- risk-adjusted signal
- explainable rationale
- simulation scenario

## Invariants
- signal generation must be deterministic for the same replayed input set and market context snapshot
- risk adjustment must not mutate the original signal semantics silently; the adjusted form must be explicit
- explainability output must reference actual factors used by the signal/risk engines
- simulation must be isolated from live production streams
