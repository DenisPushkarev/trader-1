#!/usr/bin/env bash
# config.sh — Interactive configuration collection.
# Sourced by setup.sh — do not execute directly.

# ── Global config variables ──────────────────────────────────────────────────

PROJECT_NAME=""
GITHUB_REPO=""
PROJECT_DESC=""
OUTPUT_DIR=""

# Areas (parallel arrays)
declare -a AREA_NAMES=()
declare -a AREA_PATHS=()
declare -a AREA_REVIEW_FOCUS=()
declare -a AREA_MESSAGING=()

# Developer profiles (parallel arrays)
declare -a DEV_NAMES=()
declare -a DEV_AREAS=()         # comma-separated area names
declare -a DEV_DESCRIPTIONS=()

# Domains (parallel arrays)
declare -a DOMAIN_NAMES=()
declare -a DOMAIN_AREAS=()      # comma-separated area names

# Models
ARCHITECT_MODEL="anthropic/claude-opus-4.6"
REVIEWER_MODEL="anthropic/claude-sonnet-4-5"
ROUTER_MODEL="openai/gpt-4o-mini"
ARCHITECT_TOKENS=8192
DEVELOPER_TOKENS=16384
REVIEWER_TOKENS=4096
ROUTER_TOKENS=2048

# Flags
INSTALL_RUNNERS=false
CREATE_LABELS=false

# ── Collection functions ─────────────────────────────────────────────────────

collect_project_basics() {
  step "Project Basics"

  prompt_text PROJECT_NAME  "Project name (slug, e.g. my-platform)"
  prompt_text GITHUB_REPO   "GitHub repository (owner/repo)"
  prompt_text PROJECT_DESC  "Brief project description" "A multi-service platform"

  local default_out
  default_out="$(pwd)/${PROJECT_NAME}"
  prompt_text OUTPUT_DIR "Output directory" "$default_out"

  echo ""
  success "Project: ${PROJECT_NAME}"
  success "Repo:    ${GITHUB_REPO}"
  success "Output:  ${OUTPUT_DIR}"
}


collect_areas() {
  step "Service Areas"
  info "Define the areas/services your project has."
  info "Each area gets its own label (area/<name>), path patterns, review focus."
  dim "Examples: api, auth, payments, data-pipeline, frontend, contracts"
  echo ""

  local i=0
  while true; do
    separator
    local name=""
    read -rp "$(echo -e "${CYAN}Area name${NC} ${DIM}(empty to finish)${NC}: ")" name
    [[ -z "$name" ]] && [[ $i -gt 0 ]] && break
    if [[ -z "$name" ]]; then
      warn "At least one area is required."
      continue
    fi

    # Sanitize: lowercase, replace spaces with hyphens
    name="${name,,}"
    name="${name// /-}"

    local default_path="services/${name}/**"
    local paths=""
    read -rp "$(echo -e "${CYAN}  Path patterns${NC} ${DIM}[${default_path}]${NC}: ")" paths
    paths="${paths:-$default_path}"

    local focus=""
    read -rp "$(echo -e "${CYAN}  Review focus${NC} ${DIM}[correctness,testing]${NC}: ")" focus
    focus="${focus:-correctness,testing}"

    local messaging=""
    read -rp "$(echo -e "${CYAN}  Messaging topics${NC} ${DIM}(optional, comma-separated)${NC}: ")" messaging

    AREA_NAMES+=("$name")
    AREA_PATHS+=("$paths")
    AREA_REVIEW_FOCUS+=("$focus")
    AREA_MESSAGING+=("$messaging")
    ((i++))

    success "Area: ${name} → ${paths}"
  done

  echo ""
  success "Total areas: ${#AREA_NAMES[@]}"
}


collect_domains() {
  step "Domain Grouping"
  info "Group areas into domains for shared context."
  info "Domains share architectural context (e.g. 'trading', 'market-data', 'infra')."
  dim "Areas: ${AREA_NAMES[*]}"
  echo ""

  if prompt_yesno "Group areas into domains?" "y"; then
    local di=0
    while true; do
      separator
      local dname=""
      read -rp "$(echo -e "${CYAN}Domain name${NC} ${DIM}(empty to finish)${NC}: ")" dname
      [[ -z "$dname" ]] && [[ $di -gt 0 ]] && break
      [[ -z "$dname" ]] && warn "At least one domain is required." && continue

      dname="${dname,,}"
      dname="${dname// /-}"

      echo -e "  Available areas: ${BOLD}${AREA_NAMES[*]}${NC}"
      local areas=""
      read -rp "$(echo -e "${CYAN}  Areas in this domain${NC} ${DIM}(comma-separated)${NC}: ")" areas

      DOMAIN_NAMES+=("$dname")
      DOMAIN_AREAS+=("$areas")
      ((di++))

      success "Domain: ${dname} → [${areas}]"
    done
  else
    # Single default domain with all areas
    DOMAIN_NAMES+=("general")
    local all_areas
    all_areas=$(IFS=,; echo "${AREA_NAMES[*]}")
    DOMAIN_AREAS+=("$all_areas")
    success "Single domain: general → all areas"
  fi
}


collect_developers() {
  step "Developer Profiles"
  info "Define developer agent profiles."
  info "Each profile focuses on specific areas and has its own context."
  echo ""

  local count
  prompt_number count "Number of developer profiles" 1 1 20

  local i
  for ((i=0; i<count; i++)); do
    separator
    info "Developer profile $((i+1)) of ${count}"

    local name=""
    prompt_text name "  Profile name (e.g. backend, frontend, contracts)"

    name="${name,,}"
    name="${name// /-}"

    echo -e "  Available areas: ${BOLD}${AREA_NAMES[*]}${NC}"
    local areas=""
    read -rp "$(echo -e "${CYAN}  Assigned areas${NC} ${DIM}(comma-separated, or 'all')${NC}: ")" areas
    if [[ "$areas" == "all" ]] || [[ -z "$areas" ]]; then
      areas=$(IFS=,; echo "${AREA_NAMES[*]}")
    fi

    local desc=""
    read -rp "$(echo -e "${CYAN}  Brief context description${NC} ${DIM}[General development]${NC}: ")" desc
    desc="${desc:-General development for assigned areas}"

    DEV_NAMES+=("$name")
    DEV_AREAS+=("$areas")
    DEV_DESCRIPTIONS+=("$desc")

    success "Developer: ${name} → [${areas}]"
  done

  echo ""
  success "Total developer profiles: ${#DEV_NAMES[@]}"
}


collect_models() {
  step "AI Model Configuration"
  info "Choose models for each agent role."
  info "Models must be available on OpenRouter (https://openrouter.ai/models)."
  info "Developer agent uses Claude Code CLI (Anthropic API directly)."
  echo ""

  prompt_text ARCHITECT_MODEL "Architect model" "$ARCHITECT_MODEL"
  prompt_text REVIEWER_MODEL  "Reviewer model"  "$REVIEWER_MODEL"
  prompt_text ROUTER_MODEL    "Router model"    "$ROUTER_MODEL"

  echo ""
  if prompt_yesno "Customize max token limits?" "n"; then
    prompt_number ARCHITECT_TOKENS "  Architect max tokens" "$ARCHITECT_TOKENS" 1024 32768
    prompt_number DEVELOPER_TOKENS "  Developer max tokens" "$DEVELOPER_TOKENS" 1024 65536
    prompt_number REVIEWER_TOKENS  "  Reviewer max tokens"  "$REVIEWER_TOKENS" 1024 16384
    prompt_number ROUTER_TOKENS    "  Router max tokens"    "$ROUTER_TOKENS" 512 8192
  fi

  success "Architect: ${ARCHITECT_MODEL} (${ARCHITECT_TOKENS} tokens)"
  success "Reviewer:  ${REVIEWER_MODEL} (${REVIEWER_TOKENS} tokens)"
  success "Router:    ${ROUTER_MODEL} (${ROUTER_TOKENS} tokens)"
}


collect_post_setup() {
  step "Post-Setup Options"

  if prompt_yesno "Install self-hosted runners on this machine?" "n"; then
    INSTALL_RUNNERS=true
  fi

  if prompt_yesno "Create GitHub labels via gh CLI after generation?" "n"; then
    CREATE_LABELS=true
  fi
}


confirm_config() {
  step "Configuration Summary"

  echo -e "  ${BOLD}Project:${NC}     ${PROJECT_NAME}"
  echo -e "  ${BOLD}Repository:${NC}  ${GITHUB_REPO}"
  echo -e "  ${BOLD}Description:${NC} ${PROJECT_DESC}"
  echo -e "  ${BOLD}Output:${NC}      ${OUTPUT_DIR}"
  echo ""
  echo -e "  ${BOLD}Areas:${NC}       ${AREA_NAMES[*]}"
  echo -e "  ${BOLD}Domains:${NC}     ${DOMAIN_NAMES[*]}"
  echo -e "  ${BOLD}Developers:${NC}  ${DEV_NAMES[*]}"
  echo ""
  echo -e "  ${BOLD}Architect:${NC}   ${ARCHITECT_MODEL}"
  echo -e "  ${BOLD}Reviewer:${NC}    ${REVIEWER_MODEL}"
  echo -e "  ${BOLD}Router:${NC}      ${ROUTER_MODEL}"
  echo ""

  if ! prompt_yesno "Proceed with generation?" "y"; then
    err "Aborted by user."
    exit 1
  fi
}


# ── Main collection entry point ──────────────────────────────────────────────

collect_config() {
  collect_project_basics
  collect_areas
  collect_domains
  collect_developers
  collect_models
  collect_post_setup
  confirm_config
}
