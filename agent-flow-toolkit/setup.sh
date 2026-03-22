#!/usr/bin/env bash
# ============================================================================
# Agent Flow Toolkit — Interactive Setup
#
# Generates a complete multi-agent CI/CD configuration for any GitHub project.
# Includes: GitHub Actions workflows, routing, agent contexts, runner setup,
# label creation, and OpenRouter/Anthropic integration.
#
# Usage:
#   ./setup.sh
#
# Prerequisites: bash 4+, python3, pip3
# ============================================================================

set -euo pipefail

TOOLKIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source library modules
source "$TOOLKIT_DIR/lib/ui.sh"
source "$TOOLKIT_DIR/lib/config.sh"
source "$TOOLKIT_DIR/lib/generate.sh"

# ── Pre-flight checks ───────────────────────────────────────────────────────

preflight() {
  local missing=()

  if ! command -v python3 &>/dev/null; then
    missing+=("python3")
  fi

  if ! command -v bash &>/dev/null || [[ "${BASH_VERSINFO[0]}" -lt 4 ]]; then
    missing+=("bash 4+")
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    err "Missing required tools: ${missing[*]}"
    err "Please install them and try again."
    exit 1
  fi

  # Optional tools (warn but don't fail)
  if ! command -v gh &>/dev/null; then
    warn "'gh' (GitHub CLI) not found. Label creation and runner registration require it."
    warn "Install: https://cli.github.com"
    echo ""
  fi
}

# ── Post-setup actions ──────────────────────────────────────────────────────

post_setup() {
  if [[ "$CREATE_LABELS" == "true" ]]; then
    step "Creating GitHub Labels"
    if command -v gh &>/dev/null; then
      bash "$OUTPUT_DIR/setup/create-labels.sh"
    else
      warn "gh CLI not found — skipping label creation."
      info "Run later: bash ${OUTPUT_DIR}/setup/create-labels.sh"
    fi
  fi

  if [[ "$INSTALL_RUNNERS" == "true" ]]; then
    step "Installing Self-Hosted Runners"
    if command -v gh &>/dev/null; then
      bash "$OUTPUT_DIR/setup/install-all-runners.sh"
    else
      warn "gh CLI not found — skipping runner installation."
      info "Run later: bash ${OUTPUT_DIR}/setup/install-all-runners.sh"
    fi
  fi
}

# ── Summary ─────────────────────────────────────────────────────────────────

print_summary() {
  step "Setup Complete"

  echo -e "  ${BOLD}Output directory:${NC} ${OUTPUT_DIR}"
  echo ""
  echo -e "  ${BOLD}Generated files:${NC}"

  # Count generated files
  local count
  count=$(find "$OUTPUT_DIR" -type f | wc -l)
  echo -e "    ${count} files generated"
  echo ""

  echo -e "  ${BOLD}Next steps:${NC}"
  echo ""
  echo "  1. Review and customize the context files:"
  echo "     ${OUTPUT_DIR}/.github/agent-contexts/"
  echo ""
  echo "  2. Add GitHub secrets to your repository:"
  echo "     - GH_TOKEN (PAT with repo, issues, pull-requests scopes)"
  echo "     - OPENROUTER_API_KEY (for architect, reviewer, router)"
  echo "     - ANTHROPIC_API_KEY (for Claude Code developer agent)"
  echo "     https://github.com/${GITHUB_REPO}/settings/secrets/actions"
  echo ""

  if [[ "$INSTALL_RUNNERS" == "false" ]]; then
    echo "  3. Install self-hosted runners:"
    echo "     cd ${OUTPUT_DIR} && bash setup/install-all-runners.sh"
    echo ""
  fi

  if [[ "$CREATE_LABELS" == "false" ]]; then
    echo "  4. Create GitHub labels:"
    echo "     cd ${OUTPUT_DIR} && bash setup/create-labels.sh"
    echo ""
  fi

  echo "  5. Push the generated files to your repository:"
  echo "     cd ${OUTPUT_DIR}"
  echo "     git init && git add -A"
  echo "     git commit -m 'chore: agent flow setup'"
  echo "     git remote add origin git@github.com:${GITHUB_REPO}.git"
  echo "     git push -u origin main"
  echo ""
  echo "  6. Create an issue with labels (type/* + area/*) and comment /plan"
  echo ""

  success "Done! Happy building with AI agents."
}

# ── Main ─────────────────────────────────────────────────────────────────────

main() {
  banner
  preflight
  collect_config
  generate_all
  post_setup
  print_summary
}

main "$@"
