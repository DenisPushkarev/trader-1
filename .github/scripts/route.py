#!/usr/bin/env python3
"""
route.py — Agent Router script for TON Trading Agent Flow.

Reads GitHub issue context (labels, title, body) and routing.yaml,
produces a task-packet.json, posts a routing summary comment to the issue,
and triggers architect-agent.yml via `gh workflow run`.

Usage (called from agent-router.yml):
  python3 .github/scripts/route.py \
    --issue-number <N> \
    --repo <owner/repo>

Environment:
  GH_TOKEN   — GitHub token (set automatically in Actions)
  GITHUB_REPOSITORY — set automatically in Actions
"""

import argparse
import json
import os
import re
import sys

try:
    import yaml
except ImportError:
    print("Error: pyyaml not installed. Run: pip3 install --break-system-packages pyyaml", file=sys.stderr)
    sys.exit(1)

# ── Area → allowed paths mapping ─────────────────────────────────────────────

AREA_PATHS = {
    "area/collector":       ["services/collector-service/**", "packages/shared/**"],
    "area/normalizer":      ["services/normalizer-service/**", "packages/shared/**"],
    "area/signal-engine":   ["services/signal-engine-service/**", "packages/shared/**"],
    "area/risk-engine":     ["services/risk-engine-service/**", "packages/shared/**"],
    "area/market-context":  ["services/market-context-service/**", "packages/shared/**"],
    "area/explainability":  ["services/explainability-service/**", "packages/shared/**"],
    "area/api-gateway":     ["services/api-gateway-service/**", "packages/shared/**"],
    "area/simulation":      ["services/simulation-service/**", "packages/shared/**"],
    "area/contracts":       ["packages/contracts/**", "packages/shared/**"],
    "area/cross-service":   ["services/**", "packages/**"],
    "area/platform":        ["services/**", "packages/**", "infrastructure/**"],
}

# ── Area → NATS subjects typically impacted ───────────────────────────────────

AREA_NATS = {
    "area/collector":       ["events.raw"],
    "area/normalizer":      ["events.raw", "events.normalized"],
    "area/signal-engine":   ["events.normalized", "market.context.updated", "signals.generated"],
    "area/risk-engine":     ["signals.generated", "signals.risk_adjusted"],
    "area/market-context":  ["market.context.updated"],
    "area/explainability":  ["signals.risk_adjusted", "signals.explained"],
    "area/api-gateway":     ["signals.explained"],
    "area/simulation":      ["signals.generated", "signals.risk_adjusted"],
    "area/contracts":       [],
    "area/cross-service":   [],
    "area/platform":        [],
}

# ── Reviewer focus per area ───────────────────────────────────────────────────

AREA_REVIEW_FOCUS = {
    "area/contracts":       ["correctness", "contracts", "architecture", "reliability"],
    "area/cross-service":   ["correctness", "contracts", "architecture", "reliability"],
    "area/platform":        ["correctness", "architecture", "reliability"],
    "area/signal-engine":   ["correctness", "reliability", "testing"],
    "area/risk-engine":     ["correctness", "reliability", "testing"],
    "area/collector":       ["reliability", "testing"],
    "area/normalizer":      ["correctness", "reliability", "testing"],
    "area/market-context":  ["correctness", "reliability"],
    "area/explainability":  ["correctness", "testing"],
    "area/api-gateway":     ["correctness", "security", "testing"],
    "area/simulation":      ["correctness", "testing"],
}


def gh(args: list[str]) -> dict | list | str:
    """Run a gh CLI command and return parsed JSON output."""
    import subprocess
    result = subprocess.run(
        ["gh"] + args,
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f"gh error: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return result.stdout.strip()


def gh_run(args: list[str]) -> None:
    """Run a gh CLI command, print output, exit on error."""
    import subprocess
    result = subprocess.run(["gh"] + args, capture_output=True, text=True)
    if result.stdout:
        print(result.stdout.strip())
    if result.returncode != 0:
        print(f"gh error: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)


def parse_acceptance_criteria(body: str) -> list[str]:
    """Extract checkbox items from issue body."""
    criteria = []
    for line in body.splitlines():
        m = re.match(r"\s*-\s*\[[ x]\]\s*(.+)", line, re.IGNORECASE)
        if m:
            criteria.append(m.group(1).strip())
    return criteria


def parse_constraints(body: str) -> list[str]:
    """Extract bullet points from ## Notes or ## Constraints section."""
    constraints = []
    in_section = False
    for line in body.splitlines():
        if re.match(r"^##\s+(Notes|Constraints)", line, re.IGNORECASE):
            in_section = True
            continue
        if in_section and re.match(r"^##", line):
            break
        if in_section:
            m = re.match(r"\s*[-*]\s*(.+)", line)
            if m:
                constraints.append(m.group(1).strip())
    return constraints


def load_routing_config() -> dict:
    path = ".github/routing/routing.yaml"
    if not os.path.exists(path):
        print(f"Error: routing config not found: {path}", file=sys.stderr)
        sys.exit(1)
    with open(path) as f:
        return yaml.safe_load(f)


def match_route(labels: list[str], routing_config: dict) -> dict:
    """Match issue labels against routing rules, return matched route."""
    label_set = set(labels)
    for rule in routing_config.get("routing_rules", []):
        when = rule.get("when", {})
        any_labels = set(when.get("any_labels", []))
        all_labels = set(when.get("all_labels", []))
        if any_labels and not (any_labels & label_set):
            continue
        if all_labels and not all_labels.issubset(label_set):
            continue
        return rule["route"]
    return routing_config.get("default_route", {})


def build_required_contexts(area_labels: list[str], routing_config: dict) -> dict:
    """Build required_contexts from path_to_context mapping."""
    path_ctx = routing_config.get("path_to_context", {})
    domain_files = set()
    service_files = set()

    for area in area_labels:
        paths = AREA_PATHS.get(area, [])
        for pattern in paths:
            # Match pattern prefix against path_to_context keys
            for ctx_path, ctx in path_ctx.items():
                prefix = ctx_path.rstrip("/**").rstrip("/*")
                if pattern.startswith(prefix) or prefix in pattern:
                    if ctx.get("domain"):
                        domain_files.add(f".github/agent-contexts/{ctx['domain']}")
                    if ctx.get("service"):
                        service_files.add(f".github/agent-contexts/{ctx['service']}")

    return {
        "global": [".github/agent-contexts/platform-global.md"],
        "domain": sorted(domain_files),
        "service": sorted(service_files),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--issue-number", required=True, type=int)
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", ""))
    parser.add_argument("--trigger-comment", default="", help="Comment body if triggered by comment")
    args = parser.parse_args()

    if not args.repo:
        print("Error: --repo or GITHUB_REPOSITORY must be set", file=sys.stderr)
        sys.exit(1)

    print(f"[route] issue=#{args.issue_number} repo={args.repo}")

    # ── Fetch issue ───────────────────────────────────────────────────────────
    issue = gh(["api", f"repos/{args.repo}/issues/{args.issue_number}"])
    labels = [lbl["name"] for lbl in issue.get("labels", [])]
    title = issue["title"]
    body = issue.get("body", "") or ""

    print(f"[route] labels: {labels}")
    print(f"[route] title: {title}")

    # ── Validate required labels ──────────────────────────────────────────────
    routing_config = load_routing_config()
    required = routing_config.get("labels", {}).get("required", [])
    for req in required:
        options = req.get("one_of", [])
        if options and not any(lbl in labels for lbl in options):
            comment = (
                f"⚠️ **Agent Router**: missing required label.\n\n"
                f"Issue must have one of: `{'`, `'.join(options)}`\n\n"
                f"Please add the appropriate label and the router will re-run."
            )
            gh_run(["issue", "comment", str(args.issue_number),
                    "--repo", args.repo, "--body", comment])
            print(f"[route] Missing required label group: {options}", file=sys.stderr)
            sys.exit(0)  # Not an error — just missing labels

    # ── Extract type and area ─────────────────────────────────────────────────
    type_labels = [l for l in labels if l.startswith("type/")]
    area_labels = [l for l in labels if l.startswith("area/")]
    task_type = type_labels[0] if type_labels else "type/chore"
    area = area_labels[0] if area_labels else "area/platform"
    contracts_impact = "area/contracts" in area_labels

    # ── Match route ───────────────────────────────────────────────────────────
    route = match_route(labels, routing_config)
    requires_manual_approval = route.get("requires_manual_plan_approval", True)

    # ── Build allowed paths ───────────────────────────────────────────────────
    allowed_paths: list[str] = []
    for al in area_labels:
        for p in AREA_PATHS.get(al, []):
            if p not in allowed_paths:
                allowed_paths.append(p)

    # ── Build NATS subjects ───────────────────────────────────────────────────
    nats_subjects: list[str] = []
    for al in area_labels:
        for s in AREA_NATS.get(al, []):
            if s not in nats_subjects:
                nats_subjects.append(s)

    # ── Parse issue body ──────────────────────────────────────────────────────
    acceptance_criteria = parse_acceptance_criteria(body)
    if not acceptance_criteria:
        acceptance_criteria = [f"Implement: {title}"]
    constraints = parse_constraints(body)

    # ── Build review focus ────────────────────────────────────────────────────
    review_focus: list[str] = []
    for al in area_labels:
        for f in AREA_REVIEW_FOCUS.get(al, []):
            if f not in review_focus:
                review_focus.append(f)
    if not review_focus:
        review_focus = ["correctness", "architecture"]

    # ── Required contexts ─────────────────────────────────────────────────────
    required_contexts = build_required_contexts(area_labels, routing_config)

    # ── Assemble task packet ──────────────────────────────────────────────────
    task_packet = {
        "task_id": f"GH-{args.issue_number}",
        "task_type": task_type,
        "area": area,
        "objective": title,
        "constraints": constraints,
        "acceptance_criteria": acceptance_criteria,
        "allowed_paths": allowed_paths,
        "required_contexts": required_contexts,
        "contracts_impact": contracts_impact,
        "nats_subjects_impact": nats_subjects,
        "implementation_plan_path": f"docs/ai/plans/ISSUE-{args.issue_number}.md",
        "review_focus": review_focus,
        "route": route,
    }

    # ── Write task packet and commit to repo ─────────────────────────────────
    # The architect-agent runs on a self-hosted runner with a fresh checkout,
    # so the packet must be committed to the repo before triggering the workflow.
    import subprocess
    os.makedirs(".github/tmp", exist_ok=True)
    packet_path = f".github/tmp/task-packet-{args.issue_number}.json"
    with open(packet_path, "w") as f:
        json.dump(task_packet, f, indent=2)
    print(f"[route] task packet written: {packet_path}")

    # Commit and push so architect-agent can access it via checkout
    result = subprocess.run(
        ["git", "add", packet_path],
        capture_output=True, text=True
    )
    result = subprocess.run(
        ["git", "diff", "--cached", "--quiet"],
        capture_output=True, text=True
    )
    if result.returncode != 0:  # there are staged changes
        subprocess.run(
            ["git", "commit", "-m",
             f"chore: task packet for issue #{args.issue_number} [skip ci]"],
            check=True
        )
        subprocess.run(["git", "push", "origin", "main"], check=True)
        print(f"[route] task packet committed and pushed")
    else:
        print(f"[route] task packet unchanged, no commit needed")

    # ── Post routing summary comment ──────────────────────────────────────────
    approval_note = (
        "⚠️ **Manual plan approval required** before developer can start."
        if requires_manual_approval
        else "✅ Auto-proceed to build after architect plan is published."
    )
    contracts_note = (
        "🔴 **Contracts impact detected** — protobuf review mandatory."
        if contracts_impact else ""
    )

    comment = f"""🤖 **Agent Router** — routing summary for #{args.issue_number}

**Task:** `{task_packet['task_id']}` · `{task_type}` · `{area}`
**Objective:** {title}

**Allowed paths:**
{chr(10).join(f'- `{p}`' for p in allowed_paths)}

**NATS subjects impacted:** {', '.join(f'`{s}`' for s in nats_subjects) or 'none'}
**Contracts impact:** {'yes' if contracts_impact else 'no'}

**Required contexts:** {', '.join(required_contexts.get('service', []) or ['platform-global'])}

**Reviewers focus:** {', '.join(review_focus)}

{approval_note}
{contracts_note}

---
Next: architect-agent will produce an implementation plan and post it here.
"""

    gh_run(["issue", "comment", str(args.issue_number),
            "--repo", args.repo, "--body", comment])

    # ── Trigger architect-agent ───────────────────────────────────────────────
    print(f"[route] triggering architect-agent.yml for issue #{args.issue_number}")
    gh_run([
        "workflow", "run", "architect-agent.yml",
        "--repo", args.repo,
        "--ref", "main",
        "-f", f"issue_number={args.issue_number}",
        "-f", f"task_packet_path={packet_path}",
    ])
    print("[route] architect-agent.yml triggered ✓")


if __name__ == "__main__":
    main()
