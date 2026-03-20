# TON Trading Platform — GitHub Agent Flow

Этот архив содержит готовую конфигурацию GitHub-centric flow для multi-agent разработки под monorepo `ton-trading-platform`.

В составе:
- маршрутизация задач по labels и путям
- workflows для architect / developer / reviewer
- контексты агентов
- policy и handoff contracts
- пример taxonomy labels
- шаблоны артефактов, которые обязаны производить агенты

## Модель ролей

- **architect-agent** — анализ задачи, bounded contexts, event-flow impact, implementation plan, acceptance criteria, риски
- **developer-agent** — реализация по утверждённому плану, локальные тесты, изменение только разрешённого scope
- **reviewer-agent** — code review, architecture compliance, contracts compatibility, NATS/event compatibility, risk findings

## Базовый flow

1. `issue` создаётся с labels `type/*` + `area/*`
2. `agent-router.yml` вычисляет маршрут
3. `architect-agent.yml` создаёт implementation plan
4. после label `agent/build-ready` запускается `developer-agent.yml`
5. после открытия PR запускается `reviewer-agent.yml`

## Структура

```text
.github/
  workflows/
    agent-router.yml
    architect-agent.yml
    developer-agent.yml
    reviewer-agent.yml
  agent-contexts/
    platform-global.md
    architect-context.md
    developer-context.md
    reviewer-context.md
    domain-trading.md
    domain-market-data.md
    service-*.md
  routing/
    labels.md
    routing.yaml
    handoff-contract.json
    policy.md
docs/ai/
  implementation-plan-template.md
  review-findings-template.md
  task-packet-template.json
examples/
  issue-example.md
  pr-review-example.md
```

## Ключевой принцип

Агент не получает весь monorepo целиком. Контекст собирается слоями:
1. global platform context
2. domain context
3. service-specific context
4. task packet
5. affected code paths
6. relevant contracts / NATS topology slice

