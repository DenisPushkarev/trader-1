# Service Context — api-gateway-service

## Responsibility
Provide HTTP API over signals, events, history, and simulations.

## Endpoints
- `/signals/latest`
- `/signals/history`
- `/events`
- `/simulate`

## Dependencies
- PostgreSQL read models
- Redis cache
- optional request/reply integration for live simulate requests
