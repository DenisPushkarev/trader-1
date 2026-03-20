# Roadmap — TON Trading Agent Flow

Текущий статус: runner-инфраструктура и OpenRouter-конфигурация готовы.
Агенты зарегистрированы как self-hosted runners, модели параметризованы через `.github/agent-config.yml`.

---

## Фаза 1 — Бутстрап (сейчас)

### 1.1 Подготовить репозиторий
- [ ] Создать репозиторий `git@github.com:DenisPushkarev/trader-1.git` (если не создан)
- [ ] Запушить этот flow-репозиторий:
  ```bash
  git remote add origin git@github.com:DenisPushkarev/trader-1.git
  git push -u origin main
  ```

### 1.2 Настроить секреты GitHub
Перейти: https://github.com/DenisPushkarev/trader-1/settings/secrets/actions

| Secret | Описание |
|--------|----------|
| `OPENROUTER_API_KEY` | Ключ с https://openrouter.ai/keys |
| `GH_TOKEN` | GitHub PAT: scopes `repo`, `issues`, `pull_requests` |

### 1.3 Установить self-hosted runners
```bash
cd setup
chmod +x install-all-runners.sh install-runner.sh
./install-all-runners.sh
```
Нужно 3 регистрационных токена (генерировать по одному на шаге).

### 1.4 Настроить labels в репозитории
Создать taxonomy labels:
```bash
gh label create "type/feature"    --color 0075ca --repo DenisPushkarev/trader-1
gh label create "type/bug"        --color d73a4a --repo DenisPushkarev/trader-1
gh label create "type/refactor"   --color e4e669 --repo DenisPushkarev/trader-1
gh label create "area/signal-engine"   --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/risk-engine"     --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/collector"       --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/normalizer"      --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/market-context"  --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/api-gateway"     --color 0e8a16 --repo DenisPushkarev/trader-1
gh label create "area/contracts"       --color f9d0c4 --repo DenisPushkarev/trader-1
gh label create "area/cross-service"   --color f9d0c4 --repo DenisPushkarev/trader-1
gh label create "agent/planned"        --color 5319e7 --repo DenisPushkarev/trader-1
gh label create "agent/build-ready"    --color 5319e7 --repo DenisPushkarev/trader-1
```

---

## Фаза 2 — Реализация agent-router

**Файл:** `.github/workflows/agent-router.yml`  
**Файлы скриптов:** `.github/scripts/route.py`

Что нужно реализовать:
- Читать labels и paths из event payload
- Матчить `routing.yaml` (уже готов, нужен Python-парсер)
- Генерировать `task-packet.json` по `handoff-contract.json` схеме
- Вызывать `architect-agent.yml` через GitHub API (`gh workflow run`)
- Постить routing summary как комментарий к issue

---

## Фаза 3 — Реализация architect-agent

**Файл:** `.github/workflows/architect-agent.yml`  
**Файлы скриптов:** `.github/scripts/build-architect-prompt.py`

Что нужно реализовать:
- Собирать промпт из: platform-global + architect-context + domain context + service context + task packet
- Включать в промпт шаблон `implementation-plan-template.md` как целевую структуру ответа
- Вызывать `call-llm.py --role architect`
- Сохранять план в `docs/ai/plans/ISSUE-<n>.md`
- Постить план в issue через `gh issue comment`
- Ставить label `agent/planned`
- Опционально ставить `agent/build-ready` если нет human gate

---

## Фаза 4 — Реализация developer-agent

**Файл:** `.github/workflows/developer-agent.yml`  
**Файлы скриптов:** `.github/scripts/build-developer-prompt.py`, `.github/scripts/apply-changes.py`

Что нужно реализовать:
- Загружать только `allowed_paths` из task packet (не весь repo)
- Собирать промпт: platform-global + developer-context + domain + service + task packet + implementation plan + текущий код затронутых файлов
- Вызывать `call-llm.py --role developer`
- Парсить структурированный ответ агента (JSON с patch/file changes)
- Применять изменения к файлам
- Запускать `go test ./...` в scope задачи
- Делать коммит и PR через `gh pr create`

---

## Фаза 5 — Реализация reviewer-agent

**Файл:** `.github/workflows/reviewer-agent.yml`  
**Файлы скриптов:** `.github/scripts/build-reviewer-prompt.py`

Что нужно реализовать:
- Получать diff PR через `gh pr diff`
- Связывать PR с issue → грузить implementation plan
- Собирать промпт: platform-global + reviewer-context + diff + plan + contracts
- Вызывать `call-llm.py --role reviewer`
- Парсить findings по шаблону `review-findings-template.md`
- Постить review через `gh pr review --comment`

---

## Фаза 6 — Собственный monorepo (целевой проект)

Создать `ton-trading-platform` monorepo (отдельный репозиторий) со структурой:
```
services/
  collector-service/
  normalizer-service/
  signal-engine-service/
  risk-engine-service/
  market-context-service/
  explainability-service/
  api-gateway-service/
  simulation-service/
packages/
  contracts/   (protobuf)
  shared/
infrastructure/
  docker/
```

Этот flow-репозиторий будет управлять разработкой monorepo через GitHub Issues.

---

## Приоритетный порядок

```
Фаза 1 (бутстрап)  →  Фаза 2 (router)  →  Фаза 3 (architect)
→  Фаза 6 (monorepo scaffold)  →  Фаза 4 (developer)  →  Фаза 5 (reviewer)
```

Фазу 6 можно начать параллельно с фазой 3 — monorepo scaffold не зависит от готовности агентов.
