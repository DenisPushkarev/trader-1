#!/bin/bash
# install-runner.sh — registers and optionally installs as a systemd service
# one GitHub Actions self-hosted runner for a given agent role.
#
# Usage: ./install-runner.sh <role>
#   role: architect | developer | reviewer
#
# Prerequisites: curl, tar, python3, pip3, gh CLI
#
# Runner token: GitHub → trader-1 → Settings → Actions → Runners → New self-hosted runner
# (token is single-use, valid for 1 hour)

set -euo pipefail

ROLE="${1:-}"
REPO_URL="https://github.com/DenisPushkarev/trader-1"
RUNNERS_BASE_DIR="$HOME/actions-runners"
RUNNER_DIR="$RUNNERS_BASE_DIR/${ROLE}-runner"

# ── Validate ──────────────────────────────────────────────────────────────────

if [[ -z "$ROLE" ]]; then
  echo "Usage: $0 <role>"
  echo "  Roles: architect  developer  reviewer"
  exit 1
fi

if [[ ! "$ROLE" =~ ^(architect|developer|reviewer)$ ]]; then
  echo "Error: role must be one of: architect, developer, reviewer"
  exit 1
fi

# ── Check dependencies ────────────────────────────────────────────────────────

echo "=== Checking dependencies ==="
for cmd in curl tar python3 pip3; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: '$cmd' is required but not installed."
    echo "  Install via: sudo apt-get install -y ${cmd/pip3/python3-pip}"
    exit 1
  fi
done

# gh CLI (https://cli.github.com) — used by agent scripts to comment on issues / open PRs
if ! command -v gh &>/dev/null; then
  echo ""
  echo "WARNING: 'gh' (GitHub CLI) is not installed."
  echo "  Agents need it to post comments and open PRs."
  echo "  Install: https://cli.github.com/manual/installation"
  echo "  Continuing runner registration without it..."
fi

# ── Python dependencies ───────────────────────────────────────────────────────

echo ""
echo "=== Installing Python dependencies ==="
pip3 install --quiet --break-system-packages pyyaml requests

# ── Download runner ───────────────────────────────────────────────────────────

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

# ── Register ──────────────────────────────────────────────────────────────────

echo ""
echo "=== Runner Registration ==="
echo "Open this URL to generate a registration token:"
echo "  https://github.com/DenisPushkarev/trader-1/settings/actions/runners/new?runnerOs=linux"
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

# ── Systemd service (optional) ────────────────────────────────────────────────

echo ""
read -r -p "Install as systemd service (requires sudo)? [y/N] " INSTALL_SVC
if [[ "$INSTALL_SVC" =~ ^[Yy]$ ]]; then
  sudo ./svc.sh install
  sudo ./svc.sh start
  echo "Service status:"
  sudo ./svc.sh status
else
  echo ""
  echo "To start manually (foreground):"
  echo "  cd $RUNNER_DIR && ./run.sh"
fi

echo ""
echo "Done. Runner '${ROLE}-agent' is ready."
echo "Labels: self-hosted, linux, agent, ${ROLE}"
