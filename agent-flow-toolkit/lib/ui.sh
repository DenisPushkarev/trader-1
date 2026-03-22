#!/usr/bin/env bash
# ui.sh — Terminal UI helpers: colors, prompts, banners.
# Sourced by setup.sh — do not execute directly.

# -- Colors ----------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# -- Output helpers --------------------------------------------------------

banner() {
  echo ""
  echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${BOLD}${BLUE}║        Agent Flow Toolkit — Interactive Setup           ║${NC}"
  echo -e "${BOLD}${BLUE}║   Multi-agent CI/CD for GitHub + OpenRouter + Claude    ║${NC}"
  echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

info()    { echo -e "${BLUE}ℹ${NC}  $*"; }
success() { echo -e "${GREEN}✔${NC}  $*"; }
warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
err()     { echo -e "${RED}✖${NC}  $*" >&2; }
step()    { echo -e "\n${BOLD}${CYAN}── $* ──${NC}\n"; }
dim()     { echo -e "${DIM}$*${NC}"; }

# -- Prompt helpers --------------------------------------------------------

# prompt_text <variable_name> <prompt_text> [default_value]
# Reads user input; stores in the named variable.
prompt_text() {
  local __var="$1" __prompt="$2" __default="${3:-}"
  local __input

  if [[ -n "$__default" ]]; then
    read -rp "$(echo -e "${CYAN}${__prompt}${NC} ${DIM}[${__default}]${NC}: ")" __input
    __input="${__input:-$__default}"
  else
    while true; do
      read -rp "$(echo -e "${CYAN}${__prompt}${NC}: ")" __input
      [[ -n "$__input" ]] && break
      warn "This field is required."
    done
  fi

  # Use printf %q to safely assign
  eval "$__var=\$__input"
}

# prompt_yesno <prompt_text> [default: y|n]
# Returns 0 for yes, 1 for no.
prompt_yesno() {
  local __prompt="$1" __default="${2:-y}"
  local __hint __input

  if [[ "$__default" == "y" ]]; then
    __hint="Y/n"
  else
    __hint="y/N"
  fi

  read -rp "$(echo -e "${CYAN}${__prompt}${NC} ${DIM}[${__hint}]${NC}: ")" __input
  __input="${__input:-$__default}"

  case "${__input,,}" in
    y|yes) return 0 ;;
    *)     return 1 ;;
  esac
}

# prompt_number <variable_name> <prompt_text> <default> <min> <max>
prompt_number() {
  local __var="$1" __prompt="$2" __default="$3" __min="$4" __max="$5"
  local __input

  while true; do
    read -rp "$(echo -e "${CYAN}${__prompt}${NC} ${DIM}[${__default}]${NC}: ")" __input
    __input="${__input:-$__default}"
    if [[ "$__input" =~ ^[0-9]+$ ]] && (( __input >= __min && __input <= __max )); then
      eval "$__var=\$__input"
      return
    fi
    warn "Please enter a number between ${__min} and ${__max}."
  done
}

# separator
separator() {
  echo -e "${DIM}──────────────────────────────────────────────${NC}"
}
