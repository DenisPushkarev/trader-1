#!/usr/bin/env python3
"""
build-architect-prompt.py — Assembles the full prompt for the architect agent.

Reads context files + task packet, writes a combined prompt to stdout.
The LLM is instructed to produce an implementation plan matching
docs/ai/implementation-plan-template.md.

Usage:
  python3 .github/scripts/build-architect-prompt.py \
    --task-packet /tmp/task-packet.json \
    | python3 .github/scripts/call-llm.py --role architect \
    > docs/ai/plans/ISSUE-<N>.md
"""

import argparse
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
        return f"[FILE NOT FOUND: {path}]\n"
    with open(path) as f:
        return f.read()


def load_project_name() -> str:
    """Read the project name from agent-config.yml."""
    cfg_path = ".github/agent-config.yml"
    if os.path.exists(cfg_path):
        with open(cfg_path) as f:
            cfg = yaml.safe_load(f)
        return cfg.get("project", "this project")
    return "this project"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--task-packet", required=True, help="Path to task-packet JSON")
    args = parser.parse_args()

    with open(args.task_packet) as f:
        task = json.load(f)

    project_name = load_project_name()
    required_contexts = task.get("required_contexts", {})
    parts: list[str] = []

    # 1. Platform global context
    parts.append("# Platform Global Context\n")
    parts.append(read_file_safe(".github/agent-contexts/platform-global.md"))

    # 2. Architect role context
    parts.append("\n\n# Architect Agent Context\n")
    parts.append(read_file_safe(".github/agent-contexts/architect-context.md"))

    # 3. Domain contexts
    for ctx in required_contexts.get("domain", []):
        parts.append(f"\n\n# Domain Context\n")
        parts.append(read_file_safe(ctx))

    # 4. Service contexts
    for ctx in required_contexts.get("service", []):
        parts.append(f"\n\n# Service Context: {os.path.basename(ctx)}\n")
        parts.append(read_file_safe(ctx))

    # 5. Task packet
    parts.append("\n\n# Task Packet\n```json\n")
    parts.append(json.dumps(task, indent=2))
    parts.append("\n```\n")

    # 6. Plan template
    parts.append("\n\n# Implementation Plan Template (your output MUST follow this structure)\n")
    parts.append(read_file_safe("docs/ai/implementation-plan-template.md"))

    # 7. Instructions
    task_id = task.get("task_id", "")
    objective = task.get("objective", "")
    area = task.get("area", "")
    allowed_paths = task.get("allowed_paths", [])
    acceptance_criteria = task.get("acceptance_criteria", [])
    contracts_impact = task.get("contracts_impact", False)
    messaging_subjects = task.get("messaging_subjects_impact", [])

    parts.append(f"""

# Your Task

You are the architect agent for {project_name}.

Produce a complete **Implementation Plan** for:
- Task ID: {task_id}
- Area: {area}
- Objective: {objective}

## Required analysis

1. **Bounded contexts impacted** — which services own which data and events
2. **Messaging subject impact** — which subjects/topics are read/written, ownership changes
   Potentially impacted subjects: {', '.join(f'`{s}`' for s in messaging_subjects) or 'none'}
3. **Contract impact** — is this additive, breaking, or none?
   contracts_impact flag = {str(contracts_impact).lower()}
4. **Idempotency and replay safety** — can historical events be replayed safely?
5. **Developer execution slices** — break work into ordered, independently testable slices
6. **Risks** — what can go wrong, mitigation strategies

## Constraints

- Scope is limited to: {', '.join(f'`{p}`' for p in allowed_paths)}
- Acceptance criteria that MUST be covered:
{chr(10).join(f'  - {c}' for c in acceptance_criteria)}
- Contract changes must be backward-compatible (additive only) unless you explicitly flag breaking and escalate
- Do NOT write production code — only the plan

## Output format

Respond with ONLY the implementation plan in Markdown, following the template structure above.
Start directly with `# Implementation Plan — {task_id}`.
Do not add any preamble or explanation outside the plan document.
""")

    print("".join(parts))


if __name__ == "__main__":
    main()
