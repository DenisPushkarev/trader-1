#!/bin/bash
# install-all-runners.sh — installs all three agent runners sequentially.
# Run this once on the machine that will host all runners.
#
# Usage: ./install-all-runners.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "======================================================"
echo " TON Trading Agent — Self-hosted Runner Installer"
echo "======================================================"
echo ""
echo "This will register 3 runners:"
echo "  1. architect-agent  (label: agent,architect)"
echo "  2. developer-agent  (label: agent,developer)"
echo "  3. reviewer-agent   (label: agent,reviewer)"
echo ""
echo "You will need 3 separate runner registration tokens."
echo "Each token is single-use. Generate them one by one at:"
echo "  https://github.com/DenisPushkarev/trader-1/settings/actions/runners/new"
echo ""
read -r -p "Ready? Press Enter to continue..."

for ROLE in architect developer reviewer; do
  echo ""
  echo "======================================================"
  echo " Installing runner: $ROLE"
  echo "======================================================"
  bash "$SCRIPT_DIR/install-runner.sh" "$ROLE"
done

echo ""
echo "======================================================"
echo " All runners installed!"
echo "======================================================"
echo ""
echo "Next steps:"
echo "  1. Add OPENROUTER_API_KEY to GitHub repository secrets:"
echo "       https://github.com/DenisPushkarev/trader-1/settings/secrets/actions/new"
echo "  2. Add GH_TOKEN (PAT with repo + issues + pull-requests scopes):"
echo "       https://github.com/DenisPushkarev/trader-1/settings/secrets/actions/new"
echo "  3. Authorise gh CLI on this machine: gh auth login"
echo "  4. Push this repository to: git@github.com:DenisPushkarev/trader-1.git"
