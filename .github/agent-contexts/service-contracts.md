# Service Context — contracts package

## Responsibility
Own protobuf contract definitions and generated code boundaries for inter-service communication.

## Rules
- version packages under `v1`
- backward-compatible evolution only unless migration is explicitly approved
- protobuf is the single inter-service contract source of truth
