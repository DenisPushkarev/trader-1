#!/usr/bin/env python3
"""
build-developer-prompt.py — Assembles the full prompt for the developer agent.

Reads context files, task packet, implementation plan, and current code
of allowed paths; writes a combined prompt to stdout (pipe into Claude CLI).

Usage:
  python3 .github/scripts/build-developer-prompt.py \
    --task-packet /tmp/task-packet.json \
    --plan docs/ai/plans/ISSUE-<N>.md \
    --repo-root . \
    | claude --allowedTools "Edit,Write,Bash,Read,MultiEdit" --print
"""

import argparse
import glob
import json
import os
import sys

try:
    import yaml
except ImportError:
    print("Error: pyyaml not installed.", file=sys.stderr)
    sys.exit(1)


def read_file_safe(path: str) -> str:
    if not os.path.exists(path):
        return f"[FILE NOT FOUND: {path}]"
    with open(path) as f:
        return f.read()


def load_project_name() -> str:
    cfg_path = ".github/agent-config.yml"
    if os.path.exists(cfg_path):
        with open(cfg_path) as f:
            cfg = yaml.safe_load(f)
        return cfg.get("project", "this project")
    return "this project"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--task-packet", required=True)
    parser.add_argument("--plan", required=True)
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args()

    with open(args.task_packet) as f:
        task = json.load(f)

    project_name = load_project_name()
    allowed_paths = task.get("allowed_paths", [])
    contracts_impact = task.get("contracts_impact", False)
    required_contexts = task.get("required_contexts", {})
    sections: list[str] = []

    # 1. Platform global context
    sections.append("# Platform Global Context\n")
    sections.append(read_file_safe(".github/agent-contexts/platform-global.md"))

    # 2. Developer role context
    sections.append("\n\n# Developer Agent Context\n")
    sections.append(read_file_safe(".github/agent-contexts/developer-context.md"))

    # 3. Domain contexts
    for ctx_file in required_contexts.get("domain", []):
        sections.append(f"\n\n# Domain Context: {ctx_file}\n")
        sections.append(read_file_safe(ctx_file))

    # 4. Service contexts
    for ctx_file in required_contexts.get("service", []):
        sections.append(f"\n\n# Service Context: {ctx_file}\n")
        sections.append(read_file_safe(ctx_file))

    # 5. Task packet
    sections.append("\n\n# Task Packet\n```json\n")
    sections.append(json.dumps(task, indent=2))
    sections.append("\n```\n")

    # 6. Implementation plan
    sections.append("\n\n# Implementation Plan\n")
    sections.append(read_file_safe(args.plan))

    # 7. Current code of allowed paths
    sections.append("\n\n# Current Code (allowed paths)\n")
    sections.append("Read and modify ONLY the files listed below.\n")
    for pattern in allowed_paths:
        matched = glob.glob(os.path.join(args.repo_root, pattern), recursive=True)
        if not matched:
            sections.append(f"\n[No files matched: {pattern}]\n")
            continue
        for filepath in sorted(matched):
            if os.path.isfile(filepath):
                rel = os.path.relpath(filepath, args.repo_root)
                sections.append(f"\n## {rel}\n```\n")
                sections.append(read_file_safe(filepath))
                sections.append("\n```\n")

    # 8. Instructions
    sections.append(f"""\n\n# Instructions

You are the developer agent for {project_name}.

STRICT RULES:
1. Modify ONLY files within the allowed_paths listed in the Task Packet above.
2. Do NOT change any contract/interface files unless `contracts_impact` is explicitly `true`.
   contracts_impact = {str(contracts_impact).lower()}
3. Keep all event handlers idempotent and replay-safe.
4. Use constructor-based dependency injection (no global state).
5. After making code changes, run the project's test suite for changed directories.
6. Do NOT auto-merge. Do NOT push to main.
7. After all changes and tests pass:
   a. Stage all changes: git add -A
   b. Commit: git commit -m "feat: <concise summary from the plan>"
   c. Push branch: git push origin HEAD
   Then STOP — the PR will be opened by the workflow step that follows.

TASK OBJECTIVE:
{task.get('objective', '(see task packet above)')}

ACCEPTANCE CRITERIA:
{chr(10).join(f'- {c}' for c in task.get('acceptance_criteria', []))}

Begin implementation now.
""")

    print("".join(sections))


if __name__ == "__main__":
    main()
