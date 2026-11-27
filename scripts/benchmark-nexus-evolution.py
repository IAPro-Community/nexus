#!/usr/bin/env python3
"""Reproducible Nexus performance benchmark (CLI, HTTP API latencies, process RSS/CPU).

Usage:
  python3 scripts/benchmark-nexus-evolution.py [--url http://127.0.0.1:13000] [--runs 5] [--json]
"""

from __future__ import annotations

import argparse
import json
import math
import os
import statistics
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def read_session_token() -> str:
    env_token = os.environ.get("NEXUS_API_TOKEN", "").strip()
    if env_token:
        return env_token
    real_home = Path(os.environ.get("NEXUS_REAL_HOME", "/home/desenvolvedor"))
    for home in (Path.home(), real_home):
        sess_file = home / ".local" / "share" / "ai-manager" / "web-auth" / "sessions.json"
        try:
            payload = json.loads(sess_file.read_text(encoding="utf-8"))
            sessions = payload.get("sessions") or []
            if isinstance(sessions, list):
                for item in reversed(sessions):
                    if isinstance(item, dict) and item.get("id"):
                        return str(item["id"]).strip()
        except (OSError, ValueError):
            continue
    return ""


def percentile(values: list[float], pct: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    idx = (len(ordered) - 1) * pct
    lo = math.floor(idx)
    hi = math.ceil(idx)
    if lo == hi:
        return ordered[lo]
    return ordered[lo] * (hi - idx) + ordered[hi] * (idx - lo)


def bench_cli_cold_start(binary: str, runs: int) -> dict[str, Any]:
    latencies: list[float] = []
    version = ""
    for _ in range(runs):
        t0 = time.perf_counter()
        proc = subprocess.run([binary, "--version"], capture_output=True, text=True, check=False)
        elapsed_ms = (time.perf_counter() - t0) * 1000.0
        if proc.returncode != 0:
            return {"binary": binary, "error": proc.stderr.strip() or f"exit {proc.returncode}"}
        if not version:
            version = proc.stdout.strip()
        latencies.append(elapsed_ms)
    return {
        "binary": binary,
        "version": version,
        "runs": runs,
        "min_ms": round(min(latencies), 2),
        "p50_ms": round(percentile(latencies, 0.50), 2),
        "p95_ms": round(percentile(latencies, 0.95), 2),
        "avg_ms": round(statistics.mean(latencies), 2),
        "max_ms": round(max(latencies), 2),
    }


def fetch_json(url: str, token: str) -> tuple[int, bytes, float]:
    headers: dict[str, str] = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, headers=headers, method="GET")
    t0 = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read()
            elapsed_ms = (time.perf_counter() - t0) * 1000.0
            return resp.status, body, elapsed_ms
    except urllib.error.HTTPError as err:
        elapsed_ms = (time.perf_counter() - t0) * 1000.0
        return err.code, err.read(), elapsed_ms


def bench_endpoint(base_url: str, path: str, token: str, runs: int) -> dict[str, Any]:
    url = base_url.rstrip("/") + path
    latencies: list[float] = []
    statuses: set[int] = set()
    sizes: list[int] = []
    for _ in range(runs):
        status, body, elapsed_ms = fetch_json(url, token)
        statuses.add(status)
        sizes.append(len(body))
        latencies.append(elapsed_ms)
    return {
        "path": path,
        "statuses": sorted(statuses),
        "bytes": sizes[-1] if sizes else 0,
        "runs": runs,
        "min_ms": round(min(latencies), 2),
        "p50_ms": round(percentile(latencies, 0.50), 2),
        "p95_ms": round(percentile(latencies, 0.95), 2),
        "avg_ms": round(statistics.mean(latencies), 2),
        "max_ms": round(max(latencies), 2),
    }


def sample_processes() -> dict[str, Any]:
    proc = subprocess.run(
        ["ps", "-eo", "pid,ppid,%cpu,rss,comm,args"],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        return {"error": proc.stderr.strip()}

    rows: list[dict[str, Any]] = []
    for line in proc.stdout.splitlines()[1:]:
        parts = line.strip().split(None, 5)
        if len(parts) < 6:
            continue
        pid, ppid, cpu_s, rss_s, comm, args = parts
        rows.append(
            {
                "pid": int(pid),
                "ppid": int(ppid),
                "cpu_pct": float(cpu_s),
                "rss_mb": round(int(rss_s) / 1024.0, 2),
                "comm": comm,
                "args": args,
            }
        )

    by_pid = {r["pid"]: r for r in rows}

    def root_kind(row: dict[str, Any]) -> str:
        args = row["args"]
        comm = row["comm"]
        if "nexus web" in args or (comm == "nexus" and "web" in args):
            return "nexus_core"
        if "nexus-desktop" in args or comm == "nexus-desktop":
            return "nexus_desktop"
        if "nexus-hud" in args or comm == "nexus-hud":
            return "nexus_hud"
        if "__control-host" in args or "__browser-mcp-stdio" in args:
            return "nexus_session_helpers"
        return ""

    groups: dict[str, list[dict[str, Any]]] = {
        "nexus_core": [],
        "nexus_desktop": [],
        "nexus_hud": [],
        "nexus_session_helpers": [],
    }

    for row in rows:
        kind = root_kind(row)
        if kind:
            groups[kind].append(row)
            continue
        if row["comm"].startswith("WebKit") or "WebKitWebProcess" in row["args"] or "WebKitNetworkProcess" in row["args"]:
            parent = by_pid.get(row["ppid"])
            if parent:
                pkind = root_kind(parent)
                if pkind in ("nexus_desktop", "nexus_hud"):
                    groups[pkind].append(row)

    summary: dict[str, Any] = {}
    nexus_idle_rss_mb = 0.0
    nexus_idle_cpu_pct = 0.0
    nexus_idle_procs = 0

    for name, items in groups.items():
        rss = round(sum(i["rss_mb"] for i in items), 2)
        cpu = round(sum(i["cpu_pct"] for i in items), 2)
        summary[name] = {
            "process_count": len(items),
            "rss_mb": rss,
            "cpu_pct": cpu,
            "processes": [
                {"pid": i["pid"], "comm": i["comm"], "rss_mb": i["rss_mb"], "cpu_pct": i["cpu_pct"]}
                for i in items
            ],
        }
        if name in ("nexus_core", "nexus_desktop", "nexus_hud"):
            nexus_idle_rss_mb += rss
            nexus_idle_cpu_pct += cpu
            nexus_idle_procs += len(items)

    summary["nexus_idle_total"] = {
        "process_count": nexus_idle_procs,
        "rss_mb": round(nexus_idle_rss_mb, 2),
        "cpu_pct": round(nexus_idle_cpu_pct, 2),
    }
    return summary


def find_first_project_id(base_url: str, token: str) -> str:
    status, body, _ = fetch_json(base_url.rstrip("/") + "/api/v1/projects", token)
    if status != 200:
        return ""
    try:
        data = json.loads(body.decode("utf-8"))
        if isinstance(data, list) and data:
            first = data[0]
            if isinstance(first, dict):
                return str(first.get("id", "")).strip()
    except (ValueError, UnicodeDecodeError):
        return ""
    return ""


def main() -> int:
    parser = argparse.ArgumentParser(description="Reproducible Nexus performance benchmark")
    parser.add_argument("--url", default="http://127.0.0.1:13000", help="Nexus web base URL")
    parser.add_argument("--binary", default="/home/desenvolvedor/.local/bin/nexus", help="Nexus CLI binary path")
    parser.add_argument("--runs", type=int, default=5, help="HTTP latency samples per endpoint")
    parser.add_argument("--cli-runs", type=int, default=10, help="CLI cold start samples")
    parser.add_argument("--json", action="store_true", help="Emit JSON output")
    args = parser.parse_args()

    token = read_session_token()
    cli_res = bench_cli_cold_start(args.binary, args.cli_runs)

    endpoints = [
        "/api/v1/health",
        "/api/v1/projects",
        "/api/v1/events",
        "/api/v1/profiles",
        "/api/v1/hud",
        "/api/v1/nex/hub-status",
        "/api/v1/runtimes",
        "/api/v1/resources",
        "/api/v1/providers",
    ]
    project_id = find_first_project_id(args.url, token)
    if project_id:
        endpoints.append(f"/api/v1/projects/{project_id}/git/branches?summary=1")
        endpoints.append(f"/api/v1/projects/{project_id}/git/branches")

    api_results = [bench_endpoint(args.url, path, token, args.runs) for path in endpoints]
    proc_summary = sample_processes()

    report = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "base_url": args.url,
        "auth_token_present": bool(token),
        "project_id": project_id,
        "cli": cli_res,
        "api": api_results,
        "processes": proc_summary,
    }

    if args.json:
        print(json.dumps(report, indent=2))
        return 0

    print(f"# Nexus Benchmark ({report['timestamp']})")
    print(f"- CLI `{cli_res.get('binary')}`: min={cli_res.get('min_ms')}ms avg={cli_res.get('avg_ms')}ms p95={cli_res.get('p95_ms')}ms")
    print(f"- Nexus Idle Total: {proc_summary['nexus_idle_total']['rss_mb']} MB RSS across {proc_summary['nexus_idle_total']['process_count']} procs ({proc_summary['nexus_idle_total']['cpu_pct']}% CPU)")
    for row in api_results:
        print(f"  - {row['path']}: status={row['statuses']} avg={row['avg_ms']}ms p95={row['p95_ms']}ms min={row['min_ms']}ms ({row['bytes']} B)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
