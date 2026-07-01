#!/usr/bin/env python3
"""Manual OpenAI-compatible translation concurrency probe.

This script intentionally uses only the Python standard library so it can run on
Windows or the Raspberry Pi without preparing a project environment.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import os
import time
import urllib.error
import urllib.request


PROMPT = """Translate the following subtitle segments to Chinese. Keep the %% separators.

The quick brown fox jumps over the lazy dog.
%%
We need stable translations under bursty browser-extension concurrency.
%%
Queueing is acceptable, but upstream EOF storms are not.
%%
Return only translated text with the original separators.
"""


def parse_levels(raw: str) -> list[int]:
    levels: list[int] = []
    for item in raw.split(","):
        item = item.strip()
        if not item:
            continue
        value = int(item)
        if value <= 0:
            raise ValueError("concurrency levels must be positive")
        levels.append(value)
    if not levels:
        raise ValueError("at least one concurrency level is required")
    return levels


def post_chat_completion(endpoint: str, api_key: str, model: str, timeout: float) -> dict[str, object]:
    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": "You are a precise subtitle translation engine."},
            {"role": "user", "content": PROMPT},
        ],
        "temperature": 0,
    }
    body = json.dumps(payload).encode("utf-8")
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "User-Agent": "ppap-translation-concurrency-probe/1",
        "Origin": "chrome-extension://amkbmndfnliijdhojkpoglbnaaahippg",
    }
    request = urllib.request.Request(endpoint, data=body, headers=headers, method="POST")
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            response_body = response.read(2048)
            status = response.status
            ok = 200 <= status < 300
            error_head = ""
    except urllib.error.HTTPError as exc:
        response_body = exc.read(2048)
        status = exc.code
        ok = False
        error_head = response_body.decode("utf-8", errors="replace")[:500]
    except Exception as exc:  # noqa: BLE001 - this is a diagnostics script.
        response_body = b""
        status = 0
        ok = False
        error_head = repr(exc)[:500]
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    return {
        "status": status,
        "ok": ok,
        "elapsed_ms": elapsed_ms,
        "bytes": len(response_body),
        "error_head": error_head,
    }


def run_level(endpoint: str, api_key: str, model: str, level: int, timeout: float) -> dict[str, object]:
    with concurrent.futures.ThreadPoolExecutor(max_workers=level) as executor:
        futures = [
            executor.submit(post_chat_completion, endpoint, api_key, model, timeout)
            for _ in range(level)
        ]
        results = [future.result() for future in concurrent.futures.as_completed(futures)]
    statuses: dict[str, int] = {}
    for result in results:
        status_key = str(result["status"])
        statuses[status_key] = statuses.get(status_key, 0) + 1
    elapsed_values = sorted(int(result["elapsed_ms"]) for result in results)
    errors = [str(result["error_head"]) for result in results if result["error_head"]]
    return {
        "concurrency": level,
        "ok": sum(1 for result in results if result["ok"]),
        "total": len(results),
        "statuses": statuses,
        "elapsed_ms_min": elapsed_values[0],
        "elapsed_ms_p50": elapsed_values[len(elapsed_values) // 2],
        "elapsed_ms_max": elapsed_values[-1],
        "error_heads": errors[:3],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default=os.environ.get("CPA_BASE_URL", "http://127.0.0.1:8317/v1"))
    parser.add_argument("--api-key", default=os.environ.get("CPA_API_KEY", ""))
    parser.add_argument("--model", default=os.environ.get("CPA_MODEL", "gpt-5.3-codex-spark"))
    parser.add_argument("--levels", default="10,12,13,14,16,24")
    parser.add_argument("--timeout", type=float, default=90.0)
    args = parser.parse_args()

    if not args.api_key:
        raise SystemExit("missing API key: pass --api-key or set CPA_API_KEY")

    endpoint = args.base_url.rstrip("/") + "/chat/completions"
    for level in parse_levels(args.levels):
        summary = run_level(endpoint, args.api_key, args.model, level, args.timeout)
        print(json.dumps(summary, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
