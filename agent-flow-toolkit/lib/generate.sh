#!/usr/bin/env bash
# generate.sh — File generation functions.
# Sourced by setup.sh — do not execute directly.
#
# All functions write into $OUTPUT_DIR using config variables from config.sh.

# ── Helpers ──────────────────────────────────────────────────────────────────

# Copy a template file from the toolkit's templates/ directory.
copy_template() {
  local src="$1" dst="$2"
  if [[ ! -f "$src" ]]; then
    err "Template not found: $src"
    return 1
  fi
  cp "$src" "$dst"
}

# ── Directory scaffold ───────────────────────────────────────────────────────

create_directories() {
  local out="$OUTPUT_DIR"
  mkdir -p "$out/.github/workflows"
  mkdir -p "$out/.github/routing"
  mkdir -p "$out/.github/scripts"
  mkdir -p "$out/.github/agent-contexts"
  mkdir -p "$out/.github/tmp"
  mkdir -p "$out/docs/ai/plans"
  mkdir -p "$out/setup"
}

# ── .github/agent-config.yml ─────────────────────────────────────────────────

generate_agent_config() {
  local f="$OUTPUT_DIR/.github/agent-config.yml"
  cat > "$f" <<EOF
# Agent Model Configuration — ${PROJECT_NAME}
# Repository: https://github.com/${GITHUB_REPO}
#
# Models must be available on OpenRouter: https://openrouter.ai/models
#
# Recommended cost/quality tradeoffs:
#   High quality : anthropic/claude-opus-4-5
#   Balanced     : anthropic/claude-sonnet-4-5  |  openai/gpt-4o
#   Fast/cheap   : openai/gpt-4o-mini  |  google/gemini-flash-1.5

project: ${PROJECT_NAME}

models:
  # Architect: designs plans, analyzes scope — high reasoning required
  architect: ${ARCHITECT_MODEL}

  # Developer: uses Claude Code CLI (Anthropic API), NOT OpenRouter.
  # Claude model is controlled by ANTHROPIC_API_KEY + claude CLI defaults.
  # developer: (managed by Claude Code CLI)

  # Reviewer: reviews diffs, checks contracts and architecture
  reviewer: ${REVIEWER_MODEL}

  # Router: label validation, task packet assembly — cheap model is fine
  router: ${ROUTER_MODEL}

openrouter:
  base_url: https://openrouter.ai/api/v1
  site_url: https://github.com/${GITHUB_REPO}
  site_name: ${PROJECT_NAME}

# Maximum tokens per agent call
max_tokens:
  architect: ${ARCHITECT_TOKENS}
  developer: ${DEVELOPER_TOKENS}
  reviewer: ${REVIEWER_TOKENS}
  router: ${ROUTER_TOKENS}
EOF
  success "Generated .github/agent-config.yml"
}

# ── .github/routing/routing.yaml ─────────────────────────────────────────────

generate_routing_yaml() {
  local f="$OUTPUT_DIR/.github/routing/routing.yaml"
  local area_labels=""
  local i

  # Build area label list for required labels
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    [[ -n "$area_labels" ]] && area_labels="${area_labels}, "
    area_labels="${area_labels}area/${AREA_NAMES[$i]}"
  done

  # Start writing
  cat > "$f" <<EOF
version: 1
project: ${PROJECT_NAME}

# -- Area definitions (consumed by route.py) --------------------------------
areas:
EOF

  # Write each area config
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    local name="${AREA_NAMES[$i]}"
    local paths="${AREA_PATHS[$i]}"
    local focus="${AREA_REVIEW_FOCUS[$i]}"
    local messaging="${AREA_MESSAGING[$i]}"

    # Convert comma-separated paths to YAML array
    local paths_yaml=""
    IFS=',' read -ra path_arr <<< "$paths"
    for p in "${path_arr[@]}"; do
      p="$(echo "$p" | xargs)"  # trim whitespace
      paths_yaml="${paths_yaml}\"${p}\", "
    done
    paths_yaml="[${paths_yaml%, }]"

    # Convert focus to YAML array
    local focus_yaml=""
    IFS=',' read -ra focus_arr <<< "$focus"
    for ff in "${focus_arr[@]}"; do
      ff="$(echo "$ff" | xargs)"
      focus_yaml="${focus_yaml}\"${ff}\", "
    done
    focus_yaml="[${focus_yaml%, }]"

    # Convert messaging to YAML array
    local msg_yaml="[]"
    if [[ -n "$messaging" ]]; then
      msg_yaml=""
      IFS=',' read -ra msg_arr <<< "$messaging"
      for m in "${msg_arr[@]}"; do
        m="$(echo "$m" | xargs)"
        msg_yaml="${msg_yaml}\"${m}\", "
      done
      msg_yaml="[${msg_yaml%, }]"
    fi

    cat >> "$f" <<EOF
  ${name}:
    paths: ${paths_yaml}
    review_focus: ${focus_yaml}
    messaging_topics: ${msg_yaml}
EOF
  done

  # Required labels
  cat >> "$f" <<EOF

# -- Label validation -------------------------------------------------------
labels:
  required:
    - one_of: [type/feature, type/bug, type/refactor, type/docs, type/test, type/chore]
    - one_of: [${area_labels}]

# -- Routing rules -----------------------------------------------------------
routing_rules:
EOF

  # Generate routing rules based on developer profiles
  for ((i=0; i<${#DEV_NAMES[@]}; i++)); do
    local dev_name="${DEV_NAMES[$i]}"
    local dev_areas="${DEV_AREAS[$i]}"

    IFS=',' read -ra areas_for_dev <<< "$dev_areas"
    for area_name in "${areas_for_dev[@]}"; do
      area_name="$(echo "$area_name" | xargs)"
      cat >> "$f" <<EOF
  - name: ${area_name}-local
    when:
      all_labels: [area/${area_name}]
    route:
      architect: architect-agent
      developers: [developer-${dev_name}]
      reviewers: [reviewer-architecture]
      requires_manual_plan_approval: false
      requires_integration_review: false

EOF
    done
  done

  # Default route
  local default_dev="developer-${DEV_NAMES[0]}"
  cat >> "$f" <<EOF
default_route:
  architect: architect-agent
  developers: [${default_dev}]
  reviewers: [reviewer-architecture]
  requires_manual_plan_approval: true
  requires_integration_review: true

# -- Path to context mapping -----------------------------------------------
path_to_context:
EOF

  # Generate path_to_context for each area
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    local name="${AREA_NAMES[$i]}"
    local paths="${AREA_PATHS[$i]}"

    # Find which domain this area belongs to
    local domain="general"
    local di
    for ((di=0; di<${#DOMAIN_NAMES[@]}; di++)); do
      if echo "${DOMAIN_AREAS[$di]}" | grep -qw "$name"; then
        domain="${DOMAIN_NAMES[$di]}"
        break
      fi
    done

    IFS=',' read -ra path_arr <<< "$paths"
    for p in "${path_arr[@]}"; do
      p="$(echo "$p" | xargs)"
      cat >> "$f" <<EOF
  ${p}:
    domain: domain-${domain}.md
    service: service-${name}.md
EOF
    done
  done

  success "Generated .github/routing/routing.yaml"
}

# ── .github/routing/labels.md ────────────────────────────────────────────────

generate_labels_md() {
  local f="$OUTPUT_DIR/.github/routing/labels.md"
  cat > "$f" <<'EOF'
# Recommended Labels

## Type
- `type/feature`
- `type/bug`
- `type/refactor`
- `type/docs`
- `type/test`
- `type/chore`

## Area
EOF

  local i
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    echo "- \`area/${AREA_NAMES[$i]}\`" >> "$f"
  done

  cat >> "$f" <<'EOF'

## Agent state
- `agent/planned`
- `agent/build-ready`
- `agent/in-review`
- `agent/needs-human-decision`
- `agent/blocked`

## Risk
- `risk/low`
- `risk/medium`
- `risk/high`
- `risk/contracts`
EOF
  success "Generated .github/routing/labels.md"
}

# ── .github/routing/policy.md ────────────────────────────────────────────────

generate_policy() {
  copy_template "$TOOLKIT_DIR/templates/routing/policy.md" \
                "$OUTPUT_DIR/.github/routing/policy.md"
  success "Generated .github/routing/policy.md"
}

# ── .github/routing/handoff-contract.json ─────────────────────────────────────

generate_handoff_contract() {
  copy_template "$TOOLKIT_DIR/templates/routing/handoff-contract.json" \
                "$OUTPUT_DIR/.github/routing/handoff-contract.json"
  success "Generated .github/routing/handoff-contract.json"
}

# ── Workflow files ────────────────────────────────────────────────────────────

generate_workflows() {
  local tpl="$TOOLKIT_DIR/templates/workflows"
  local dst="$OUTPUT_DIR/.github/workflows"

  copy_template "$tpl/agent-router.yml"     "$dst/agent-router.yml"
  copy_template "$tpl/architect-agent.yml"   "$dst/architect-agent.yml"
  copy_template "$tpl/developer-agent.yml"   "$dst/developer-agent.yml"
  copy_template "$tpl/reviewer-agent.yml"    "$dst/reviewer-agent.yml"

  success "Generated .github/workflows/ (4 workflow files)"
}

# ── Python scripts ────────────────────────────────────────────────────────────

generate_scripts() {
  local tpl="$TOOLKIT_DIR/templates/scripts"
  local dst="$OUTPUT_DIR/.github/scripts"

  copy_template "$tpl/route.py"                  "$dst/route.py"
  copy_template "$tpl/call-llm.py"               "$dst/call-llm.py"
  copy_template "$tpl/build-architect-prompt.py"  "$dst/build-architect-prompt.py"
  copy_template "$tpl/build-developer-prompt.py"  "$dst/build-developer-prompt.py"

  success "Generated .github/scripts/ (4 Python files)"
}

# ── Agent context files ──────────────────────────────────────────────────────

generate_contexts() {
  local ctx_dir="$OUTPUT_DIR/.github/agent-contexts"

  # Platform global context
  cat > "$ctx_dir/platform-global.md" <<EOF
# Platform Global Context — ${PROJECT_NAME}

## System goal
${PROJECT_DESC}

## Architectural constraints
<!-- TODO: Describe your architecture constraints here. Examples: -->
- Communication between services uses defined contracts/interfaces.
- Persistence choices (PostgreSQL, Redis, etc.) and their roles.
- Services must be horizontally scalable and idempotent.
- All inter-service communication is event-driven / request-response (choose one).

## Core services
<!-- TODO: List your services. Example: -->
EOF

  local i
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    echo "- **${AREA_NAMES[$i]}**: <!-- brief description -->" >> "$ctx_dir/platform-global.md"
  done

  cat >> "$ctx_dir/platform-global.md" <<'EOF'

## Contract rules
<!-- TODO: Describe your API contract strategy. Examples: -->
- Use protobuf / OpenAPI / GraphQL schemas for all contracts.
- All changes must be backward-compatible unless explicitly approved.

## Reliability requirements
<!-- TODO: Describe reliability requirements. Examples: -->
- All handlers must be idempotent.
- Deduplication strategy for at-least-once delivery.
- Graceful shutdown with in-flight message draining.
EOF

  # Architect context
  cat > "$ctx_dir/architect-context.md" <<'EOF'
# Architect Agent Context

## Role
You are responsible for architecture, decomposition, bounded context integrity,
event-flow impact analysis, and implementation planning.

## You must produce
- Implementation scope
- Affected services and packages
- Affected messaging subjects (if applicable)
- Contract impact assessment
- Data/storage impact
- Acceptance criteria (testable)
- Developer execution slices (ordered, independently testable)
- Reviewer focus areas
- Risk register

## You must NOT
- Write production code
- Approve unreviewed breaking changes
- Skip cross-service impact analysis
EOF

  # Developer context (generic)
  cat > "$ctx_dir/developer-context.md" <<'EOF'
# Developer Agent Context

## Role
You implement only the approved scope from the architect plan.

## You must do
- Modify only allowed paths listed in the task packet
- Preserve architectural boundaries
- Keep contracts as the only inter-service interface
- Add or update tests for all touched logic
- Maintain idempotency and replay safety
- Use dependency injection (no global mutable state)
- Report assumptions, blockers, or risks to the issue

## You must NOT
- Change contract/interface files unless contracts_impact is true
- Auto-merge pull requests
- Push directly to main
- Exceed the scope defined in the implementation plan
EOF

  # Reviewer context
  cat > "$ctx_dir/reviewer-context.md" <<'EOF'
# Reviewer Agent Context

## Role
You review correctness, architecture compliance, contracts compatibility,
reliability, and tests.

## Review categories
- correctness
- architecture
- contracts
- reliability
- security
- observability
- testing

## Severity levels
- **critical**: Breaks core invariants, contracts, or replay safety
- **high**: Would cause production defect
- **medium**: Maintainability or performance concern
- **low**: Style or minor improvement

## You must NOT
- Modify code directly
- Approve changes that break contracts without human escalation
- Skip checking test coverage for changed logic
EOF

  # Generate domain context files
  for ((i=0; i<${#DOMAIN_NAMES[@]}; i++)); do
    local dname="${DOMAIN_NAMES[$i]}"
    local dareas="${DOMAIN_AREAS[$i]}"
    cat > "$ctx_dir/domain-${dname}.md" <<EOF
# Domain Context — ${dname}

## Services in this domain
$(echo "$dareas" | tr ',' '\n' | while read -r a; do a="$(echo "$a" | xargs)"; echo "- ${a}"; done)

## Domain concepts
<!-- TODO: Define the key domain concepts, entities, and invariants. -->

## Domain invariants
<!-- TODO: List invariants that must hold across all services in this domain. -->
EOF
  done

  # Generate service context files for each area
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    local name="${AREA_NAMES[$i]}"
    local paths="${AREA_PATHS[$i]}"
    cat > "$ctx_dir/service-${name}.md" <<EOF
# Service Context — ${name}

## Ownership
- Path: ${paths}
- Responsibilities: <!-- TODO: describe what this service does -->

## Inputs
<!-- TODO: What events/requests does this service consume? -->

## Outputs
<!-- TODO: What events/responses does this service produce? -->

## Data stores
<!-- TODO: What databases/caches does this service use? -->

## Key invariants
<!-- TODO: What must always be true for this service? -->
EOF
  done

  # Generate per-developer context files
  local di
  for ((di=0; di<${#DEV_NAMES[@]}; di++)); do
    local dname="${DEV_NAMES[$di]}"
    local dareas="${DEV_AREAS[$di]}"
    local ddesc="${DEV_DESCRIPTIONS[$di]}"
    cat > "$ctx_dir/developer-context-${dname}.md" <<EOF
# Developer Context — ${dname}

## Focus
${ddesc}

## Assigned areas
$(echo "$dareas" | tr ',' '\n' | while read -r a; do a="$(echo "$a" | xargs)"; echo "- ${a}"; done)

## Additional guidelines
<!-- TODO: Add any developer-specific guidelines, coding standards, or conventions. -->
EOF
  done

  success "Generated .github/agent-contexts/ ($(ls "$ctx_dir" | wc -l) files)"
}

# ── docs/ai/ templates ───────────────────────────────────────────────────────

generate_docs() {
  local tpl="$TOOLKIT_DIR/templates/docs"
  local dst="$OUTPUT_DIR/docs/ai"

  copy_template "$tpl/implementation-plan-template.md" "$dst/implementation-plan-template.md"
  copy_template "$tpl/review-findings-template.md"     "$dst/review-findings-template.md"

  # Generate a project-specific task packet template
  local first_area="${AREA_NAMES[0]:-general}"
  local first_path="${AREA_PATHS[0]:-services/**}"
  IFS=',' read -ra first_path_arr <<< "$first_path"
  local first_path_item
  first_path_item="$(echo "${first_path_arr[0]}" | xargs)"

  cat > "$dst/task-packet-template.json" <<EOF
{
  "task_id": "GH-1",
  "task_type": "type/feature",
  "area": "area/${first_area}",
  "objective": "Example: implement feature X for ${first_area}",
  "constraints": [
    "No breaking contract changes",
    "Must be idempotent"
  ],
  "acceptance_criteria": [
    "Feature X works as specified",
    "Unit tests cover new logic",
    "Integration test passes"
  ],
  "allowed_paths": [
    "${first_path_item}"
  ],
  "required_contexts": {
    "global": [
      ".github/agent-contexts/platform-global.md"
    ],
    "domain": [
      ".github/agent-contexts/domain-${DOMAIN_NAMES[0]:-general}.md"
    ],
    "service": [
      ".github/agent-contexts/service-${first_area}.md"
    ]
  },
  "contracts_impact": false,
  "messaging_subjects_impact": [],
  "implementation_plan_path": "docs/ai/plans/ISSUE-1.md",
  "review_focus": [
    "correctness",
    "testing"
  ]
}
EOF

  # .gitkeep for plans directory
  touch "$OUTPUT_DIR/docs/ai/plans/.gitkeep"

  success "Generated docs/ai/ (3 template files)"
}

# ── setup/ scripts ───────────────────────────────────────────────────────────

generate_setup_scripts() {
  local dst="$OUTPUT_DIR/setup"

  # -- install-runner.sh -----------------------------------------------------
  cat > "$dst/install-runner.sh" <<'RUNEOF'
#!/bin/bash
# install-runner.sh — Register a GitHub Actions self-hosted runner for an agent role.
#
# Usage: ./install-runner.sh <role> <repo_url>
#   role: architect | developer | reviewer
#
# Prerequisites: curl, tar, python3, pip3, gh CLI

set -euo pipefail

ROLE="${1:-}"
REPO_URL="${2:-}"
RUNNERS_BASE_DIR="$HOME/actions-runners"
RUNNER_DIR="$RUNNERS_BASE_DIR/${ROLE}-runner"

if [[ -z "$ROLE" ]] || [[ -z "$REPO_URL" ]]; then
  echo "Usage: $0 <role> <repo_url>"
  echo "  Roles: architect  developer  reviewer"
  echo "  Example: $0 architect https://github.com/owner/repo"
  exit 1
fi

if [[ ! "$ROLE" =~ ^(architect|developer|reviewer)$ ]]; then
  echo "Error: role must be one of: architect, developer, reviewer"
  exit 1
fi

# -- Check dependencies ------------------------------------------------------
echo "=== Checking dependencies ==="
for cmd in curl tar python3 pip3; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: '$cmd' is required but not installed."
    exit 1
  fi
done

if ! command -v gh &>/dev/null; then
  echo ""
  echo "WARNING: 'gh' (GitHub CLI) is not installed."
  echo "  Agents need it to post comments and open PRs."
  echo "  Install: https://cli.github.com/manual/installation"
fi

# -- Python dependencies -----------------------------------------------------
echo ""
echo "=== Installing Python dependencies ==="
pip3 install --quiet --break-system-packages pyyaml requests 2>/dev/null || \
  pip3 install --quiet pyyaml requests

# -- Download runner ----------------------------------------------------------
echo ""
echo "=== Downloading GitHub Actions runner ==="
RUNNER_VERSION=$(curl -fsSL https://api.github.com/repos/actions/runner/releases/latest \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['tag_name'].lstrip('v'))")
echo "Latest version: $RUNNER_VERSION"

ARCHIVE_NAME="actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
ARCHIVE_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${ARCHIVE_NAME}"

mkdir -p "$RUNNER_DIR"
cd "$RUNNER_DIR"

if [[ ! -f "$ARCHIVE_NAME" ]]; then
  echo "Downloading $ARCHIVE_URL ..."
  curl -fsSLo "$ARCHIVE_NAME" "$ARCHIVE_URL"
fi

echo "Extracting..."
tar xzf "$ARCHIVE_NAME"

# -- Register -----------------------------------------------------------------
echo ""
echo "=== Runner Registration ==="
# Extract owner/repo from URL for settings link
OWNER_REPO=$(echo "$REPO_URL" | sed 's|https://github.com/||')
echo "Generate a registration token at:"
echo "  https://github.com/${OWNER_REPO}/settings/actions/runners/new?runnerOs=linux"
echo ""
read -r -p "Paste runner token: " RUNNER_TOKEN

./config.sh \
  --url  "$REPO_URL" \
  --token "$RUNNER_TOKEN" \
  --name  "${ROLE}-agent" \
  --labels "self-hosted,linux,agent,${ROLE}" \
  --unattended \
  --replace

echo ""
echo "=== Runner '${ROLE}-agent' registered successfully! ==="
echo ""
echo "To run as a service:"
echo "  cd $RUNNER_DIR && sudo ./svc.sh install && sudo ./svc.sh start"
RUNEOF
  chmod +x "$dst/install-runner.sh"

  # -- install-all-runners.sh ------------------------------------------------
  local repo_url="https://github.com/${GITHUB_REPO}"
  local n_devs=${#DEV_NAMES[@]}
  local total_runners=$((2 + n_devs))  # architect + reviewer + N developers

  cat > "$dst/install-all-runners.sh" <<EOF
#!/bin/bash
# install-all-runners.sh — Install all agent runners sequentially.
#
# Usage: ./install-all-runners.sh

set -euo pipefail

SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
REPO_URL="${repo_url}"

echo "======================================================"
echo " ${PROJECT_NAME} — Self-hosted Runner Installer"
echo "======================================================"
echo ""
echo "This will register ${total_runners} runners:"
echo "  1. architect-agent  (label: agent,architect)"
EOF

  local runner_num=2
  for ((i=0; i<n_devs; i++)); do
    echo "echo \"  ${runner_num}. developer-agent  (label: agent,developer)\"" >> "$dst/install-all-runners.sh"
    ((runner_num++))
  done
  echo "echo \"  ${runner_num}. reviewer-agent   (label: agent,reviewer)\"" >> "$dst/install-all-runners.sh"

  cat >> "$dst/install-all-runners.sh" <<'EOF'
echo ""
echo "You will need separate runner registration tokens for each."
echo "Each token is single-use."
echo ""
read -r -p "Ready? Press Enter to continue..."
EOF

  # Architect runner
  cat >> "$dst/install-all-runners.sh" <<EOF

echo ""
echo "======================================================"
echo " Installing runner: architect"
echo "======================================================"
bash "\$SCRIPT_DIR/install-runner.sh" architect "\$REPO_URL"
EOF

  # Developer runners
  for ((i=0; i<n_devs; i++)); do
    cat >> "$dst/install-all-runners.sh" <<EOF

echo ""
echo "======================================================"
echo " Installing runner: developer (profile: ${DEV_NAMES[$i]})"
echo "======================================================"
bash "\$SCRIPT_DIR/install-runner.sh" developer "\$REPO_URL"
EOF
  done

  # Reviewer runner
  cat >> "$dst/install-all-runners.sh" <<EOF

echo ""
echo "======================================================"
echo " Installing runner: reviewer"
echo "======================================================"
bash "\$SCRIPT_DIR/install-runner.sh" reviewer "\$REPO_URL"

echo ""
echo "======================================================"
echo " All runners installed!"
echo "======================================================"
echo ""
echo "Next steps:"
echo "  1. Add OPENROUTER_API_KEY to GitHub repository secrets:"
echo "       https://github.com/${GITHUB_REPO}/settings/secrets/actions/new"
echo "  2. Add ANTHROPIC_API_KEY (for Claude Code developer agent):"
echo "       https://github.com/${GITHUB_REPO}/settings/secrets/actions/new"
echo "  3. Add GH_TOKEN (PAT with repo + issues + pull-requests scopes):"
echo "       https://github.com/${GITHUB_REPO}/settings/secrets/actions/new"
echo "  4. Authorize gh CLI on this machine: gh auth login"
EOF
  chmod +x "$dst/install-all-runners.sh"

  success "Generated setup/ (2 runner install scripts)"
}

# ── setup/create-labels.sh ───────────────────────────────────────────────────

generate_labels_script() {
  local f="$OUTPUT_DIR/setup/create-labels.sh"

  cat > "$f" <<EOF
#!/bin/bash
# create-labels.sh — Create recommended GitHub labels for ${PROJECT_NAME}.
#
# Prerequisites: gh CLI authenticated (gh auth login)
#
# Usage: ./create-labels.sh

set -euo pipefail

REPO="${GITHUB_REPO}"

echo "Creating labels for \$REPO..."

create_label() {
  local name="\$1" color="\$2" desc="\$3"
  gh label create "\$name" --repo "\$REPO" --color "\$color" --description "\$desc" --force
}

# Type labels
create_label "type/feature"   "0e8a16" "New feature"
create_label "type/bug"       "d73a4a" "Bug fix"
create_label "type/refactor"  "ededed" "Code refactoring"
create_label "type/docs"      "0075ca" "Documentation"
create_label "type/test"      "fbca04" "Tests"
create_label "type/chore"     "c5def5" "Maintenance"

# Area labels
EOF

  local i
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    echo "create_label \"area/${AREA_NAMES[$i]}\" \"1d76db\" \"Area: ${AREA_NAMES[$i]}\"" >> "$f"
  done

  cat >> "$f" <<'EOF'

# Agent state labels
create_label "agent/planned"              "5319e7" "Agent: plan generated"
create_label "agent/build-ready"          "0e8a16" "Agent: ready for developer"
create_label "agent/in-review"            "fbca04" "Agent: PR under review"
create_label "agent/needs-human-decision" "d93f0b" "Agent: requires human input"
create_label "agent/blocked"              "b60205" "Agent: blocked"

# Risk labels
create_label "risk/low"       "c2e0c6" "Low risk"
create_label "risk/medium"    "fbca04" "Medium risk"
create_label "risk/high"      "d93f0b" "High risk"
create_label "risk/contracts" "b60205" "Contracts impacted"

echo ""
echo "All labels created!"
EOF
  chmod +x "$f"

  success "Generated setup/create-labels.sh"
}

# ── .gitignore ───────────────────────────────────────────────────────────────

generate_gitignore() {
  cat > "$OUTPUT_DIR/.gitignore" <<'EOF'
# Runner binaries
actions-runner-*
_work/
_diag/

# Agent temp files
.github/tmp/

# OS
.DS_Store
Thumbs.db

# IDE
.idea/
.vscode/
*.swp
*.swo

# Environment (secrets must NOT be committed)
.env
.env.local
*.key
EOF
  success "Generated .gitignore"
}

# ── README.md ────────────────────────────────────────────────────────────────

generate_readme() {
  local f="$OUTPUT_DIR/README.md"
  local dev_list=""
  local i
  for ((i=0; i<${#DEV_NAMES[@]}; i++)); do
    dev_list="${dev_list}  - \`developer-${DEV_NAMES[$i]}\`: ${DEV_DESCRIPTIONS[$i]}\n"
  done

  local area_list=""
  for ((i=0; i<${#AREA_NAMES[@]}; i++)); do
    area_list="${area_list}| \`area/${AREA_NAMES[$i]}\` | \`${AREA_PATHS[$i]}\` |\n"
  done

  cat > "$f" <<EOF
# ${PROJECT_NAME} — AI Agent Flow

${PROJECT_DESC}

## Overview

This repository uses a multi-agent CI/CD flow powered by GitHub Actions,
OpenRouter (for architect & reviewer), and Claude Code (for developer).

### Agent Roles

| Agent | Model | Purpose |
|-------|-------|---------|
| **Architect** | \`${ARCHITECT_MODEL}\` | Analyzes scope, designs implementation plans |
| **Developer** | Claude Code CLI | Implements code changes from the plan |
| **Reviewer** | \`${REVIEWER_MODEL}\` | Reviews PRs for correctness, architecture, contracts |
| **Router** | \`${ROUTER_MODEL}\` | Validates labels, builds task packets |

### Developer Profiles

$(echo -e "$dev_list")

### Service Areas

| Label | Paths |
|-------|-------|
$(echo -e "$area_list")

## How It Works

1. **Create an issue** with required labels (\`type/*\` + \`area/*\`)
2. **Trigger the agent** by commenting \`/plan\` on the issue
3. **Router** validates labels, builds a task packet, triggers the architect
4. **Architect** generates an implementation plan, posts it to the issue
5. **Manual approval** (or auto-proceed): add \`agent/build-ready\` label
6. **Developer** agent creates a feature branch, implements the plan, opens a PR
7. **Reviewer** agent reviews the PR and posts structured findings

## Setup

### 1. GitHub Secrets

Add these secrets to your repository settings:

| Secret | Description |
|--------|-------------|
| \`GH_TOKEN\` | GitHub PAT with \`repo\`, \`issues\`, \`pull-requests\` scopes |
| \`OPENROUTER_API_KEY\` | OpenRouter API key (for architect, reviewer, router) |
| \`ANTHROPIC_API_KEY\` | Anthropic API key (for Claude Code developer agent) |

### 2. Self-Hosted Runners

Install self-hosted runners on a Linux machine:

\`\`\`bash
# Install all runners at once
./setup/install-all-runners.sh

# Or install individually
./setup/install-runner.sh architect https://github.com/${GITHUB_REPO}
./setup/install-runner.sh developer https://github.com/${GITHUB_REPO}
./setup/install-runner.sh reviewer  https://github.com/${GITHUB_REPO}
\`\`\`

### 3. GitHub Labels

Create the recommended labels:

\`\`\`bash
./setup/create-labels.sh
\`\`\`

### 4. Customize Context Files

Edit the files in \`.github/agent-contexts/\` to describe your project:

- \`platform-global.md\` — Overall architecture and constraints
- \`architect-context.md\` — Architect role instructions
- \`developer-context.md\` — Developer role instructions
- \`reviewer-context.md\` — Reviewer role instructions
- \`domain-*.md\` — Domain-specific context
- \`service-*.md\` — Per-service context

## Configuration

- **Models**: \`.github/agent-config.yml\`
- **Routing rules**: \`.github/routing/routing.yaml\`
- **Agent policy**: \`.github/routing/policy.md\`
- **Task packet schema**: \`.github/routing/handoff-contract.json\`
- **Labels**: \`.github/routing/labels.md\`

## File Structure

\`\`\`
.github/
├── agent-config.yml          # Model configuration
├── workflows/
│   ├── agent-router.yml      # Issue routing
│   ├── architect-agent.yml   # Plan generation
│   ├── developer-agent.yml   # Code implementation
│   └── reviewer-agent.yml    # PR review
├── routing/
│   ├── routing.yaml          # Routing rules and area definitions
│   ├── labels.md             # Label taxonomy
│   ├── policy.md             # Agent authority policy
│   └── handoff-contract.json # Task packet JSON schema
├── scripts/
│   ├── route.py              # Router logic
│   ├── call-llm.py           # OpenRouter client
│   ├── build-architect-prompt.py
│   └── build-developer-prompt.py
└── agent-contexts/
    ├── platform-global.md
    ├── architect-context.md
    ├── developer-context.md
    ├── reviewer-context.md
    ├── domain-*.md
    └── service-*.md
docs/ai/
├── implementation-plan-template.md
├── review-findings-template.md
├── task-packet-template.json
└── plans/                    # Generated plans stored here
setup/
├── install-runner.sh
├── install-all-runners.sh
└── create-labels.sh
\`\`\`
EOF
  success "Generated README.md"
}

# ── Main generation entry point ──────────────────────────────────────────────

generate_all() {
  step "Generating Project Files"

  create_directories

  generate_agent_config
  generate_workflows
  generate_routing_yaml
  generate_labels_md
  generate_policy
  generate_handoff_contract
  generate_scripts
  generate_contexts
  generate_docs
  generate_setup_scripts
  generate_labels_script
  generate_gitignore
  generate_readme

  echo ""
  success "All files generated in: ${OUTPUT_DIR}"
}
