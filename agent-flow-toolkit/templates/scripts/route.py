#!/usr/bin/env python3
"""
route.py — Agent Router: validates labels, builds task packet, triggers architect.

All area paths, review focus, and messaging topics are read from routing.yaml.
No project-specific values are hardcoded.

Usage (called from agent-router.yml):
  python3 .github/scripts/route.py \
    --issue-number <N> \
    --repo <owner/repo>

Environment:
  GITHUB_TOKEN   — GitHub token (set automatically in Actions)
"""

import argparse
import json
import os
import re
import subprocess
import sys

try:
    import yaml
except ImportError:
    print("Error: pyyaml not installed. Run: pip3 install pyyaml", file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------------------
# GitHub CLI helpers
# ---------------------------------------------------------------------------

def gh(args: list[str]):
    """Run a gh CLI command and return parsed JSON (or raw text)."""
    result = subprocess.run(["gh"] + args, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"gh error: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return result.stdout.strip()


def gh_run(args: list[str]) -> None:
    """Run a gh CLI command; exit on error."""
    result = subprocess.run(["gh"] + args, capture_output=True, text=True)
    if result.stdout:
        print(result.stdout.strip())
    if result.returncode != 0:
        print(f"gh error: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)


# ---------------------------------------------------------------------------
# Configuration loaders
# ---------------------------------------------------------------------------

def load_routing_config() -> dict:
    path = ".github/routing/routing.yaml"
    if not os.path.exists(path):
        print(f"Error: routing config not found: {path}", file=sys.stderr)
        sys.exit(1)
    with open(path) as f:
        return yaml.safe_load(f)


def load_area_maps(routing_config: dict):
    """Build area_paths, area_review_focus, area_messaging from routing.yaml."""
    areas = routing_config.get("areas", {})
    area_paths: dict[str, list[str]] = {}
    area_review_focus: dict[str, list[str]] = {}
    area_messaging: dict[str, list[str]] = {}

    for name, cfg in areas.items():
        label = f"area/{name}"
        area_paths[label] = cfg.get("paths", [f"services/{name}/**"])
        area_review_focus[label] = cfg.get("review_focus", ["correctness", "testing"])
        area_messaging[label] = cfg.get("messaging_topics", [])

    return area_paths, area_review_focus, area_messaging


# ---------------------------------------------------------------------------
# Issue body parsers
# ---------------------------------------------------------------------------

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


# ---------------------------------------------------------------------------
# Routing logic
# ---------------------------------------------------------------------------

def match_route(labels: list[str], routing_config: dict) -> dict:
    """Match issue labels against routing rules; return matched route."""
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
    area_paths, _, _ = load_area_maps(routing_config)

    domain_files: set[str] = set()
    service_files: set[str] = set()

    for area in area_labels:
        paths = area_paths.get(area, [])
        for pattern in paths:
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


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--issue-number", required=True, type=int)
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", ""))
    parser.add_argument("--trigger-comment", default="")
    parser.add_argument("--trigger-label", default="")
    args = parser.parse_args()

    if not args.repo:
        print("Error: --repo or GITHUB_REPOSITORY must be set", file=sys.stderr)
        sys.exit(1)

    print(f"[route] issue=#{args.issue_number} repo={args.repo}")

    # -- Fetch issue -----------------------------------------------------------
    issue = gh(["api", f"repos/{args.repo}/issues/{args.issue_number}"])
    labels = [lbl["name"] for lbl in issue.get("labels", [])]
    title = issue["title"]
    body = issue.get("body", "") or ""

    print(f"[route] labels: {labels}")
    print(f"[route] title: {title}")

    # -- Load routing config ---------------------------------------------------
    routing_config = load_routing_config()
    area_paths, area_review_focus, area_messaging = load_area_maps(routing_config)

    # -- Validate required labels ----------------------------------------------
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
            sys.exit(0)

    # -- Extract type and area -------------------------------------------------
    type_labels = [l for l in labels if l.startswith("type/")]
    area_labels = [l for l in labels if l.startswith("area/")]
    task_type = type_labels[0] if type_labels else "type/chore"
    area = area_labels[0] if area_labels else "area/platform"
    contracts_impact = "area/contracts" in area_labels

    # -- Match route -----------------------------------------------------------
    route = match_route(labels, routing_config)
    requires_manual_approval = route.get("requires_manual_plan_approval", True)

    # -- Build allowed paths ---------------------------------------------------
    allowed_paths: list[str] = []
    for al in area_labels:
        for p in area_paths.get(al, []):
            if p not in allowed_paths:
                allowed_paths.append(p)

    # -- Build messaging subjects ----------------------------------------------
    messaging_subjects: list[str] = []
    for al in area_labels:
        for s in area_messaging.get(al, []):
            if s not in messaging_subjects:
                messaging_subjects.append(s)

    # -- Parse issue body ------------------------------------------------------
    acceptance_criteria = parse_acceptance_criteria(body)
    if not acceptance_criteria:
        acceptance_criteria = [f"Implement: {title}"]
    constraints = parse_constraints(body)

    # -- Build review focus ----------------------------------------------------
    review_focus: list[str] = []
    for al in area_labels:
        for f in area_review_focus.get(al, []):
            if f not in review_focus:
                review_focus.append(f)
    if not review_focus:
        review_focus = ["correctness", "architecture"]

    # -- Required contexts -----------------------------------------------------
    required_contexts = build_required_contexts(area_labels, routing_config)

    # -- Assemble task packet --------------------------------------------------
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
        "messaging_subjects_impact": messaging_subjects,
        "implementation_plan_path": f"docs/ai/plans/ISSUE-{args.issue_number}.md",
        "review_focus": review_focus,
        "route": route,
        "requires_manual_plan_approval": requires_manual_approval,
    }

    task_packet_json = json.dumps(task_packet)
    print(f"[route] task packet built for issue #{args.issue_number}")

    # -- agent/build-ready → trigger developer-agent ---------------------------
    if args.trigger_label == "agent/build-ready":
        plan_path = f"docs/ai/plans/ISSUE-{args.issue_number}.md"
        print("[route] agent/build-ready label — triggering developer-agent.yml")
        gh_run([
            "workflow", "run", "developer-agent.yml",
            "--repo", args.repo,
            "--ref", "main",
            "-f", f"issue_number={args.issue_number}",
            "-f", f"task_packet_json={task_packet_json}",
            "-f", f"plan_path={plan_path}",
        ])
        print("[route] developer-agent.yml triggered")
        return

    # -- Post routing summary comment ------------------------------------------
    approval_note = (
        "⚠️ **Manual plan approval required** before developer can start."
        if requires_manual_approval
        else "✅ Auto-proceed to build after architect plan is published."
    )
    contracts_note = (
        "🔴 **Contracts impact detected** — contract review mandatory."
        if contracts_impact else ""
    )

    comment = f"""🤖 **Agent Router** — routing summary for #{args.issue_number}

**Task:** `{task_packet['task_id']}` · `{task_type}` · `{area}`
**Objective:** {title}

**Allowed paths:**
{chr(10).join(f'- `{p}`' for p in allowed_paths)}

**Messaging subjects impacted:** {', '.join(f'`{s}`' for s in messaging_subjects) or 'none'}
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

    # -- Trigger architect-agent -----------------------------------------------
    print(f"[route] triggering architect-agent.yml for issue #{args.issue_number}")
    gh_run([
        "workflow", "run", "architect-agent.yml",
        "--repo", args.repo,
        "--ref", "main",
        "-f", f"issue_number={args.issue_number}",
        "-f", f"task_packet_json={task_packet_json}",
    ])
    print("[route] architect-agent.yml triggered")


if __name__ == "__main__":
    main()
