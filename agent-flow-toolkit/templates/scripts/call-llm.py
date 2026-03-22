#!/usr/bin/env python3
"""
call-llm.py — Sends a prompt to OpenRouter and prints the response.

Usage:
  python3 call-llm.py --role <role> [--system <file>] [--prompt <file|-]

Arguments:
  --role     Agent role: architect | reviewer | router
  --system   Path to system-prompt file (optional)
  --prompt   Path to user-prompt file, or '-' for stdin (default: -)
  --config   Path to agent-config.yml (default: .github/agent-config.yml)

Environment:
  OPENROUTER_API_KEY   Required. Your OpenRouter API key.
"""

import argparse
import json
import os
import sys
import urllib.error
import urllib.request

try:
    import yaml
except ImportError:
    print("Error: pyyaml not installed. Run: pip3 install pyyaml", file=sys.stderr)
    sys.exit(1)


def load_config(config_path: str) -> dict:
    with open(config_path) as f:
        return yaml.safe_load(f)


def call_openrouter(
    api_key: str,
    model: str,
    messages: list,
    max_tokens: int,
    site_url: str,
    site_name: str,
) -> str:
    url = "https://openrouter.ai/api/v1/chat/completions"
    payload = json.dumps(
        {
            "model": model,
            "messages": messages,
            "max_tokens": max_tokens,
        }
    ).encode("utf-8")

    req = urllib.request.Request(
        url,
        data=payload,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": site_url,
            "X-Title": site_name,
        },
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=300) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return data["choices"][0]["message"]["content"]
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        print(f"OpenRouter API error {e.code}: {body}", file=sys.stderr)
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"Network error: {e.reason}", file=sys.stderr)
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description="Call OpenRouter LLM for an agent role.")
    parser.add_argument(
        "--role", required=True,
        choices=["architect", "developer", "reviewer", "router"],
    )
    parser.add_argument("--config", default=".github/agent-config.yml")
    parser.add_argument("--system", default=None)
    parser.add_argument("--prompt", default="-")
    args = parser.parse_args()

    # -- API key ---------------------------------------------------------------
    api_key = os.environ.get("OPENROUTER_API_KEY", "").strip()
    if not api_key:
        print("Error: OPENROUTER_API_KEY environment variable is not set.", file=sys.stderr)
        sys.exit(1)

    # -- Config ----------------------------------------------------------------
    if not os.path.exists(args.config):
        print(f"Error: config file not found: {args.config}", file=sys.stderr)
        sys.exit(1)

    config = load_config(args.config)
    model = config["models"][args.role]
    max_tokens = config.get("max_tokens", {}).get(args.role, 4096)
    site_url = config.get("openrouter", {}).get("site_url", "")
    site_name = config.get("openrouter", {}).get("site_name", "agent-flow")

    print(f"[call-llm] role={args.role}  model={model}  max_tokens={max_tokens}", file=sys.stderr)

    # -- Messages --------------------------------------------------------------
    messages = []

    if args.system:
        if not os.path.exists(args.system):
            print(f"Error: system prompt file not found: {args.system}", file=sys.stderr)
            sys.exit(1)
        with open(args.system) as f:
            messages.append({"role": "system", "content": f.read()})

    if args.prompt == "-":
        user_content = sys.stdin.read()
    else:
        if not os.path.exists(args.prompt):
            print(f"Error: prompt file not found: {args.prompt}", file=sys.stderr)
            sys.exit(1)
        with open(args.prompt) as f:
            user_content = f.read()

    if not user_content.strip():
        print("Error: prompt is empty.", file=sys.stderr)
        sys.exit(1)

    messages.append({"role": "user", "content": user_content})

    # -- Call ------------------------------------------------------------------
    response = call_openrouter(api_key, model, messages, max_tokens, site_url, site_name)
    print(response)


if __name__ == "__main__":
    main()
