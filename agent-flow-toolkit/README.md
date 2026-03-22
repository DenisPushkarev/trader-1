# Agent Flow Toolkit

Interactive setup tool that generates a complete **multi-agent CI/CD configuration** for any GitHub project. Produces working GitHub Actions workflows, routing rules, agent context files, runner install scripts, and label management — all parameterized through an interactive wizard.

## What It Generates

For any project, the toolkit creates:

- **GitHub Actions Workflows** — Router, Architect, Developer, Reviewer agents
- **Routing Configuration** — Label validation, area-to-developer mapping, routing rules
- **Agent Context Files** — Platform, domain, service, and role-specific context skeletons
- **Python Scripts** — Router logic, OpenRouter LLM client, prompt builders
- **Runner Setup** — Self-hosted runner install scripts for each agent role
- **Label Management** — Script to create all recommended GitHub labels
- **Documentation** — README, plan/review templates, task packet schema

## Architecture

```
Issue created → /plan comment → Router validates labels → Architect generates plan
  → Manual approval → Developer implements via Claude Code → PR opened
  → Reviewer agent posts structured findings
```

### Agent Roles

| Agent | Provider | Purpose |
|-------|----------|---------|
| Architect | OpenRouter (configurable model) | Scope analysis, implementation planning |
| Developer | Claude Code CLI (Anthropic API) | Code implementation from plan |
| Reviewer | OpenRouter (configurable model) | PR review with structured findings |
| Router | OpenRouter (cheap model) | Label validation, task packet assembly |

## Quick Start

```bash
# 1. Clone or copy this toolkit
cd agent-flow-toolkit

# 2. Run the interactive setup
chmod +x setup.sh
./setup.sh

# 3. Follow the prompts to configure your project
```

The wizard will ask for:

1. **Project basics** — name, GitHub repo, description, output directory
2. **Service areas** — your project's modules/services with path patterns
3. **Domain grouping** — optional grouping of areas into domains
4. **Developer profiles** — N developer profiles with area assignments
5. **AI models** — model choices for architect, reviewer, router
6. **Post-setup** — optional runner installation and label creation

## Prerequisites

**Required:**
- Bash 4+
- Python 3.6+

**For full functionality:**
- [GitHub CLI](https://cli.github.com) (`gh`) — for label creation and runner registration
- [Node.js](https://nodejs.org) — for Claude Code CLI (`npm install -g @anthropic-ai/claude-code`)

**GitHub Secrets (configured after generation):**
- `GH_TOKEN` — GitHub PAT with `repo`, `issues`, `pull-requests` scopes
- `OPENROUTER_API_KEY` — [OpenRouter](https://openrouter.ai) API key
- `ANTHROPIC_API_KEY` — [Anthropic](https://console.anthropic.com) API key

## File Structure

```
agent-flow-toolkit/
├── README.md                 # This file
├── setup.sh                  # Interactive entry point
├── lib/
│   ├── ui.sh                 # Terminal colors and prompt helpers
│   ├── config.sh             # Interactive configuration collection
│   └── generate.sh           # All file generation functions
└── templates/
    ├── workflows/            # GitHub Actions workflow templates
    │   ├── agent-router.yml
    │   ├── architect-agent.yml
    │   ├── developer-agent.yml
    │   └── reviewer-agent.yml
    ├── scripts/              # Python script templates
    │   ├── route.py
    │   ├── call-llm.py
    │   ├── build-architect-prompt.py
    │   └── build-developer-prompt.py
    ├── routing/              # Routing config templates
    │   ├── policy.md
    │   └── handoff-contract.json
    └── docs/                 # Documentation templates
        ├── implementation-plan-template.md
        └── review-findings-template.md
```

## Generated Output Structure

```
<project>/
├── .github/
│   ├── agent-config.yml
│   ├── workflows/          (4 workflow files)
│   ├── routing/            (routing.yaml, labels.md, policy.md, handoff-contract.json)
│   ├── scripts/            (4 Python files)
│   └── agent-contexts/     (platform, role, domain, service contexts)
├── docs/ai/                (templates + plans directory)
├── setup/                  (runner install + label creation scripts)
├── .gitignore
└── README.md
```

## Customization After Generation

1. **Context files** (`.github/agent-contexts/`) — Fill in the TODO sections with your project's architecture, domain concepts, service descriptions, and invariants. This is the most impactful customization.

2. **Routing rules** (`.github/routing/routing.yaml`) — Adjust which developer profiles handle which areas, set manual vs auto approval per rule.

3. **Models** (`.github/agent-config.yml`) — Switch models anytime. Any model on [OpenRouter](https://openrouter.ai/models) works.

4. **Workflows** (`.github/workflows/`) — Adjust timeouts, runner labels, or add custom steps.

## How the Agents Work

### Router (`route.py`)
- Reads area definitions and routing rules from `routing.yaml`
- Validates issue labels against required label groups
- Builds a task packet (JSON) with: objective, allowed paths, contexts, review focus
- Posts routing summary to issue, triggers architect workflow

### Architect (`build-architect-prompt.py` → `call-llm.py`)
- Assembles layered context: global → role → domain → service → task
- Calls LLM to produce structured implementation plan
- Commits plan to repo, posts to issue
- Optionally auto-triggers developer agent

### Developer (`build-developer-prompt.py` → Claude Code CLI)
- Assembles context + plan + current code of allowed paths
- Passes to Claude Code which edits files directly
- Creates feature branch, runs tests, opens PR

### Reviewer (`call-llm.py`)
- Collects PR diff and related implementation plan
- Calls LLM for structured review (severity, category, recommendation)
- Posts findings as PR comment

## Security Notes

- **No secrets are stored** in generated files
- API keys go exclusively into GitHub repository secrets
- The `GH_TOKEN` secret is used (not `GITHUB_TOKEN`) to allow triggering workflows
- Developer agent is scoped to `allowed_paths` from the task packet
- Policy file defines mandatory human gates for contract/breaking changes
