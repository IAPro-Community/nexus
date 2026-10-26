#!/usr/bin/env python3
"""Run one read-only-risk-controlled provider smoke in an isolated Nexus profile.

The harness uses Nexus's supported host-session import endpoint, which copies
only validated auth files into a disposable profile. It never prints tokens,
account identity, or imported profile names. The workspace is a temporary Git
fixture and the imported credentials are removed after all Nexus processes stop.
"""

from __future__ import annotations

import http.cookiejar
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


REPO = Path(__file__).resolve().parents[4]
NEXUS = shutil.which("nexus") or "/home/desenvolvedor/.local/bin/nexus"
GOAL = "Fix typo: replace only wrold with world in README.md. Do not modify any other files."
EXPECTED_README = "NEXUS RUNTIME SMOKE: world\n"


def run_cli(args: list[str], *, cwd: Path, env: dict[str, str], timeout: int = 30) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [NEXUS, *args],
        cwd=cwd,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=timeout,
    )


def runtime_rows(payload: Any) -> list[dict[str, Any]]:
    if isinstance(payload, list):
        return [row for row in payload if isinstance(row, dict)]
    if isinstance(payload, dict) and isinstance(payload.get("runtimes"), list):
        return [row for row in payload["runtimes"] if isinstance(row, dict)]
    raise ValueError("runtime list was not valid JSON")


def query_runtimes(cwd: Path, env: dict[str, str]) -> list[dict[str, Any]]:
    result = run_cli(["ps", "--json"], cwd=cwd, env=env, timeout=15)
    if result.returncode != 0:
        raise RuntimeError("could not inspect isolated runtime list")
    return runtime_rows(json.loads(result.stdout))


def safe_text(value: str, *, root: Path, secrets: tuple[str, ...] = ()) -> str:
    result = value.replace(str(root), "<ISOLATED_ROOT>")
    for secret in secrets:
        if secret:
            result = result.replace(secret, "<REDACTED>")
    result = re.sub(
        r"(?i)(authorization|access[_-]?token|refresh[_-]?token|api[_-]?key|bearer)\s*[:=]\s*\S+",
        r"\1=[REDACTED]",
        result,
    )
    result = re.sub(r"\bsk-[A-Za-z0-9_-]{8,}\b", "[REDACTED_TOKEN]", result)
    result = re.sub(r"\b[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}\b", "[REDACTED_EMAIL]", result)
    return result


def main() -> int:
    cache = Path.home() / ".cache"
    root = Path(tempfile.mkdtemp(prefix="nexus-linux-runtime-isolated-2026-10-26-", dir=cache))
    root.chmod(0o700)
    workspace = root / "workspace"
    workspace.mkdir(mode=0o700)
    for name in ("data", "config", "state", "xdg-data", "xdg-config", "xdg-state", "tmp"):
        (root / name).mkdir(mode=0o700)
    (workspace / "README.md").write_text("NEXUS RUNTIME SMOKE: wrold\n", encoding="utf-8")
    (workspace / "package.json").write_text(
        json.dumps(
            {
                "name": "nexus-runtime-smoke",
                "private": True,
                "scripts": {"test": "node verify.cjs"},
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    (workspace / "verify.cjs").write_text(
        "const fs = require('node:fs');\n"
        "const expected = 'NEXUS RUNTIME SMOKE: world\\n';\n"
        "if (fs.readFileSync('README.md', 'utf8') !== expected) {\n"
        "  console.error('README does not match the requested exact result');\n"
        "  process.exit(1);\n"
        "}\n"
        "console.log('isolated README acceptance check: PASS');\n",
        encoding="utf-8",
    )

    for args in (
        ["init", "-q"],
        ["config", "user.name", "Nexus Homologation"],
        ["config", "user.email", "nexus-homologation@example.invalid"],
        ["add", "README.md", "package.json", "verify.cjs"],
        ["commit", "-qm", "fixture baseline"],
    ):
        subprocess.run(["git", *args], cwd=workspace, check=True, stdout=subprocess.DEVNULL)
    baseline_commit = subprocess.run(
        ["git", "rev-parse", "HEAD"], cwd=workspace, check=True, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    ).stdout.strip()

    env = os.environ.copy()
    isolated_keys = (
        "OPENAI_API_KEY",
        "ANTHROPIC_API_KEY",
        "GOOGLE_API_KEY",
        "GEMINI_API_KEY",
        "CODEX_API_KEY",
        "CLAUDE_CODE_OAUTH_TOKEN",
        "OPENROUTER_API_KEY",
        "GH_TOKEN",
        "GITHUB_TOKEN",
        "AUTH_GITLAB_NEXUS",
        "NEXUS_DATA_DIR",
        "NEXUS_CONFIG_DIR",
        "NEXUS_STATE_DIR",
        "AI_MANAGER_DATA_DIR",
        "AI_CLI_DATA_DIR",
        "AI_MANAGER_CONFIG_DIR",
        "AI_CLI_CONFIG_DIR",
        "NEXUS_WEB_PORT",
        "NEXUS_HUD_SPAWN",
        "NEXUS_ENABLE_HUD",
        "NEXUS_DISABLE_HUD",
        "XDG_DATA_HOME",
        "XDG_CONFIG_HOME",
        "XDG_STATE_HOME",
        "TMPDIR",
        "PWD",
    )
    for key in isolated_keys:
        env.pop(key, None)
    env.update(
        {
            "NEXUS_DATA_DIR": str(root / "data"),
            "NEXUS_CONFIG_DIR": str(root / "config"),
            "NEXUS_STATE_DIR": str(root / "state"),
            "AI_MANAGER_DATA_DIR": str(root / "data"),
            "AI_CLI_DATA_DIR": str(root / "data"),
            "AI_MANAGER_CONFIG_DIR": str(root / "config"),
            "AI_CLI_CONFIG_DIR": str(root / "config"),
            "XDG_DATA_HOME": str(root / "xdg-data"),
            "XDG_CONFIG_HOME": str(root / "xdg-config"),
            "XDG_STATE_HOME": str(root / "xdg-state"),
            "TMPDIR": str(root / "tmp"),
            "PWD": str(workspace),
            "NEXUS_HUD_SPAWN": "0",
        }
    )

    logs = [
        "Nexus isolated real-provider runtime smoke — 2026-10-26",
        f"Git base: {subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip()} + dirty shared worktree",
        f"Runtime CLI: {Path(NEXUS).name}; sha256={hashlib.sha256(Path(NEXUS).read_bytes()).hexdigest()}",
        "Isolation: temporary Nexus data/config/state/XDG/workspace on root filesystem; supported host-session import only; API-key environment variables cleared.",
        "Harness correction: an initial import request received HTTP 403 because it omitted the required CSRF header; no provider request was made in that attempt.",
        "A prior authenticated run reached strict plan admission but had no detected project verification command; that fixture was unchanged and this rerun adds an exact README acceptance test.",
        "The first run with that fixture completed the provider mission, but the harness incorrectly required a separate integrated workspace. This rerun checks the verified single-worktree delivery and confirms the canonical fixture stays unchanged.",
        "Fixture contains an exact README acceptance test so Nexus detects a real npm verification command for strict plan admission.",
    ]
    server: subprocess.Popen[str] | None = None
    run_started = False
    runtime_count_after: int | None = None
    outcome = "BLOCKED"
    detail = "not completed"
    imported_statuses: list[tuple[str, str]] = []
    selected_profile = ""
    bootstrap_token = ""

    try:
        server = subprocess.Popen(
            [NEXUS, "web", "--no-open", "--listen", "127.0.0.1", "--port", "0"],
            cwd=workspace,
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            start_new_session=True,
        )
        bootstrap_url = ""
        deadline = time.time() + 30
        while time.time() < deadline and server.poll() is None:
            url_result = run_cli(["web", "url"], cwd=workspace, env=env, timeout=5)
            if url_result.returncode == 0 and "#nexus_bootstrap=" in url_result.stdout:
                bootstrap_url = url_result.stdout.strip()
                break
            time.sleep(0.25)
        if not bootstrap_url:
            raise RuntimeError("isolated Core did not publish its loopback bootstrap state")
        parsed_bootstrap = urllib.parse.urlsplit(bootstrap_url)
        base_url = urllib.parse.urlunsplit((parsed_bootstrap.scheme, parsed_bootstrap.netloc, "", "", ""))
        bootstrap_token = urllib.parse.parse_qs(urllib.parse.urlsplit(bootstrap_url).fragment).get(
            "nexus_bootstrap", [""]
        )[0]
        if not bootstrap_token:
            raise RuntimeError("isolated bootstrap URL did not contain a token")

        health_ready = False
        for _ in range(40):
            try:
                with urllib.request.urlopen(base_url + "/api/v1/health", timeout=2) as response:
                    health_ready = response.status == 200
                if health_ready:
                    break
            except Exception:
                time.sleep(0.25)
        if not health_ready:
            raise RuntimeError("isolated Core did not answer its loopback health check")

        jar = http.cookiejar.CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
        bootstrap_request = urllib.request.Request(
            base_url + "/api/v1/auth/bootstrap",
            data=json.dumps({"token": bootstrap_token}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with opener.open(bootstrap_request, timeout=10) as response:
            if response.status != 200:
                raise RuntimeError(f"isolated bootstrap HTTP {response.status}")
            bootstrap_session = json.loads(response.read().decode("utf-8"))
        csrf_token = bootstrap_session.get("csrf_token", "")
        if not csrf_token:
            raise RuntimeError("isolated bootstrap did not return a CSRF token")

        import_request = urllib.request.Request(
            base_url + "/api/v1/profiles/import-host-sessions",
            data=b"{}",
            headers={"Content-Type": "application/json", "X-CSRF-Token": csrf_token},
            method="POST",
        )
        with opener.open(import_request, timeout=120) as response:
            imported = json.loads(response.read().decode("utf-8"))
        imported_statuses = [
            (item.get("provider", ""), item.get("status", ""))
            for item in imported.get("results", [])
        ]
        logs.append(
            "Host session import statuses: "
            + ", ".join(f"{provider}={status}" for provider, status in imported_statuses)
        )
        codex_result = next(
            (item for item in imported.get("results", []) if item.get("provider") == "codex"),
            None,
        )
        if not codex_result or codex_result.get("status") not in ("IMPORTED", "ALREADY_AUTHENTICATED"):
            detail = "no authenticated Codex host session imported or recognized"
            raise RuntimeError(detail)
        selected_profile = codex_result.get("profile", "")
        if not selected_profile:
            raise RuntimeError("authenticated Codex profile did not have a usable profile name")

        added = run_cli(
            ["projects", "add", str(workspace), "--name", "nexus-linux-isolated-smoke", "--isolation", "none"],
            cwd=workspace,
            env=env,
            timeout=30,
        )
        if added.returncode != 0:
            raise RuntimeError("isolated smoke workspace registration failed")
        selected = run_cli(["use", "codex", selected_profile], cwd=workspace, env=env, timeout=15)
        if selected.returncode != 0:
            raise RuntimeError("could not select imported Codex profile in isolated data")
        bound = run_cli(["bind", f"codex:{selected_profile}"], cwd=workspace, env=env, timeout=15)
        if bound.returncode != 0:
            raise RuntimeError("could not bind authenticated Codex profile to isolated workspace")
        logs.append("Authenticated Codex host session imported, selected, and bound to isolated fixture.")

        refresh_request = urllib.request.Request(
            base_url + "/api/v1/resources/refresh",
            data=json.dumps({"provider": "codex", "profile": selected_profile}).encode(),
            headers={"Content-Type": "application/json", "X-CSRF-Token": csrf_token},
            method="POST",
        )
        with opener.open(refresh_request, timeout=45) as response:
            resource_data = json.loads(response.read().decode("utf-8"))
        codex_account = next(
            (
                account
                for account in resource_data.get("accounts", [])
                if account.get("provider") == "codex" and account.get("profile") == selected_profile
            ),
            {},
        )
        quota_status = str(codex_account.get("quota_view", {}).get("status", "UNKNOWN"))
        quota_eligible = bool(codex_account.get("available", False))
        logs.append(
            f"Codex quota refresh: status={quota_status}; eligible={quota_eligible}; "
            f"authenticated={bool(codex_account.get('authenticated', False))}"
        )
        if quota_status.upper() == "UNKNOWN" or not codex_account.get("authenticated", False) or not quota_eligible:
            detail = "Codex session imported, but its quota/account state is not eligible for an autonomous run"
            raise RuntimeError(detail)

        baseline = query_runtimes(workspace, env)
        if baseline:
            raise RuntimeError(f"isolated runtime baseline was not empty ({len(baseline)})")
        logs.append("Active runtimes before request: 0")

        run_started = True
        run = subprocess.Popen(
            [NEXUS, "run", GOAL],
            cwd=workspace,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            start_new_session=True,
        )
        try:
            stdout, stderr = run.communicate(timeout=180)
            run_code = run.returncode
        except subprocess.TimeoutExpired:
            os.killpg(run.pid, signal.SIGTERM)
            try:
                stdout, stderr = run.communicate(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(run.pid, signal.SIGKILL)
                stdout, stderr = run.communicate()
            run_code = 124
            logs.append("Provider request exceeded 180 seconds; Nexus CLI process group was terminated.")

        try:
            run_summary = json.loads(stdout)
        except json.JSONDecodeError:
            run_summary = {}
        run_id = run_summary.get("run_id", "") if isinstance(run_summary, dict) else ""
        plan_id = run_summary.get("plan_id", "") if isinstance(run_summary, dict) else ""
        provider_task_verified = False
        if plan_id:
            plan_detail = run_cli(["plan", "show", str(plan_id)], cwd=workspace, env=env, timeout=15)
            if plan_detail.returncode == 0:
                try:
                    plan_data = json.loads(plan_detail.stdout)
                    package_rows = [
                        package
                        for phase in plan_data.get("phases", [])
                        for package in phase.get("packages", [])
                    ]
                    logs.append(
                        "Persisted plan: "
                        f"packages={len(package_rows)}; "
                        f"verification_requirements_present={all(bool(package.get('verification_requirements')) for package in package_rows)}"
                    )
                except (json.JSONDecodeError, AttributeError, TypeError):
                    logs.append("Persisted plan detail could not be summarized safely.")
        if run_id:
            try:
                detail_request = urllib.request.Request(base_url + "/api/v1/runs/" + urllib.parse.quote(str(run_id)))
                with opener.open(detail_request, timeout=10) as response:
                    run_detail = json.loads(response.read().decode("utf-8"))
                mission_state = str(run_detail.get("state", "UNKNOWN"))
                logs.append(
                    "Mission report: "
                    f"state={mission_state}; "
                    f"paused_reason={safe_text(str(run_detail.get('paused_reason', '')), root=root, secrets=(bootstrap_token, selected_profile))[:200] or 'none'}"
                )
                package_runs = run_detail.get("package_runs", [])
                package_checks: list[bool] = []
                for package in package_runs if isinstance(package_runs, list) else []:
                    error_message = safe_text(
                        str(package.get("error_message", "")),
                        root=root,
                        secrets=(bootstrap_token, selected_profile),
                    ).replace("\n", " ")[:240]
                    logs.append(
                        "Mission package: "
                        f"state={package.get('state', 'UNKNOWN')}; "
                        f"dispatch={package.get('dispatch_state', 'UNKNOWN')}; "
                        f"failure_kind={package.get('failure_kind', 'none')}; "
                        f"no_progress_count={package.get('no_progress_count', 0)}; "
                        f"error={error_message or 'none'}"
                    )
                    package_workspace_raw = str(package.get("workspace", "")).strip()
                    package_workspace: Path | None = None
                    workspace_safe = False
                    package_readme_ok = False
                    package_changed: list[str] = []
                    package_untracked: list[str] = []
                    if package_workspace_raw:
                        candidate = Path(package_workspace_raw)
                        try:
                            resolved = candidate.resolve(strict=True)
                            workspace_safe = resolved.is_relative_to(root.resolve(strict=True))
                            if workspace_safe and resolved.is_dir():
                                package_workspace = resolved
                        except (OSError, RuntimeError):
                            workspace_safe = False
                    if package_workspace:
                        try:
                            package_readme_ok = (package_workspace / "README.md").read_text(encoding="utf-8") == EXPECTED_README
                            package_changed = subprocess.run(
                                ["git", "diff", baseline_commit, "--name-only"],
                                cwd=package_workspace, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                            package_untracked = subprocess.run(
                                ["git", "ls-files", "--others", "--exclude-standard"],
                                cwd=package_workspace, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                        except (OSError, subprocess.CalledProcessError):
                            package_readme_ok = False
                    receipt = package.get("work_receipt") if isinstance(package.get("work_receipt"), dict) else {}
                    receipt_changed = receipt.get("changed_files", []) if isinstance(receipt, dict) else []
                    verification = receipt.get("verification", []) if isinstance(receipt, dict) else []
                    if not isinstance(verification, list) or not verification:
                        verification = package.get("verifications", [])
                    checks_passed = bool(verification) and all(
                        isinstance(item, dict) and item.get("passed") is True and int(item.get("exit_code", -1)) == 0
                        for item in verification
                    )
                    receipt_status = str(receipt.get("status", "UNKNOWN")) if isinstance(receipt, dict) else "UNKNOWN"
                    log_workspace = "<outside-isolated-root>" if package_workspace_raw and not workspace_safe else (
                        str(package_workspace.relative_to(root)) if package_workspace else "unavailable"
                    )
                    logs.append(
                        "Package evidence: "
                        f"workspace={log_workspace}; receipt={receipt_status}; "
                        f"README_exact={package_readme_ok}; changed={','.join(receipt_changed) if isinstance(receipt_changed, list) else 'invalid'}; "
                        f"workspace_changed={','.join(package_changed + package_untracked) or 'none'}; "
                        f"verification_passed={checks_passed}; "
                        f"verification_commands={','.join(str(item.get('command', '')) for item in verification if isinstance(item, dict)) or 'none'}"
                    )
                    pkg_state = str(package.get("state", "UNKNOWN"))
                    dispatch_state = str(package.get("dispatch_state", "UNKNOWN"))
                    package_checks.append(
                        workspace_safe
                        and package_workspace is not None
                        and package_readme_ok
                        and package_changed == ["README.md"]
                        and not package_untracked
                        and isinstance(receipt_changed, list)
                        and receipt_changed == ["README.md"]
                        and receipt_status.upper() == "VERIFIED"
                        and checks_passed
                        and pkg_state.upper() == "VERIFIED"
                        and dispatch_state.upper() == "COMPLETED"
                    )

                global_verifications = run_detail.get("global_verifications", [])
                global_passed = bool(global_verifications) and all(
                    isinstance(item, dict) and item.get("passed") is True and int(item.get("exit_code", -1)) == 0
                    for item in global_verifications
                )
                logs.append(
                    "Global Definition of Done: "
                    f"passed={global_passed}; "
                    f"commands={','.join(str(item.get('command', '')) for item in global_verifications if isinstance(item, dict)) or 'none'}"
                )

                canonical_raw = str(run_detail.get("workspace", "") or "").strip()
                canonical_unchanged = False
                if canonical_raw:
                    try:
                        canonical = Path(canonical_raw).resolve(strict=True)
                        if canonical.is_relative_to(root.resolve(strict=True)) and canonical.is_dir():
                            canonical_readme = (canonical / "README.md").read_text(encoding="utf-8")
                            canonical_changes = subprocess.run(
                                ["git", "diff", baseline_commit, "--name-only"],
                                cwd=canonical, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                            canonical_untracked = subprocess.run(
                                ["git", "ls-files", "--others", "--exclude-standard"],
                                cwd=canonical, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                            canonical_unchanged = (
                                canonical_readme == "NEXUS RUNTIME SMOKE: wrold\n"
                                and not canonical_changes
                                and not canonical_untracked
                            )
                            logs.append(
                                "Canonical fixture isolation: "
                                f"unchanged={canonical_unchanged}; changed={','.join(canonical_changes + canonical_untracked) or 'none'}"
                            )
                        else:
                            logs.append("Canonical workspace is outside the isolated root; it was not inspected.")
                    except (OSError, RuntimeError, subprocess.CalledProcessError):
                        logs.append("Canonical workspace could not be safely inspected inside the isolated root.")

                # A single verified agent worktree is the delivery tree for this
                # Nexus flow. A separate integrated workspace is optional.
                integrated_raw = str(run_detail.get("integrated_workspace", "") or "").strip()
                integrated_ok = not integrated_raw
                if integrated_raw:
                    try:
                        integrated = Path(integrated_raw).resolve(strict=True)
                        if integrated.is_relative_to(root.resolve(strict=True)) and integrated.is_dir():
                            integrated_readme_ok = (integrated / "README.md").read_text(encoding="utf-8") == EXPECTED_README
                            integrated_changes = subprocess.run(
                                ["git", "diff", baseline_commit, "--name-only"],
                                cwd=integrated, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                            integrated_untracked = subprocess.run(
                                ["git", "ls-files", "--others", "--exclude-standard"],
                                cwd=integrated, check=True, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            ).stdout.strip().splitlines()
                            integrated_ok = integrated_readme_ok and integrated_changes == ["README.md"] and not integrated_untracked
                            logs.append(
                                "Integrated workspace: "
                                f"path={integrated.relative_to(root)}; README_exact={integrated_readme_ok}; "
                                f"changed={','.join(integrated_changes + integrated_untracked) or 'none'}"
                            )
                        else:
                            logs.append("Integrated workspace is outside the isolated root; it was not inspected.")
                    except (OSError, RuntimeError, subprocess.CalledProcessError):
                        logs.append("Integrated workspace could not be safely inspected inside the isolated root.")
                else:
                    logs.append("Mission report did not provide an integrated workspace.")
                provider_task_verified = (
                    run_code == 0
                    and mission_state.upper() == "COMPLETED_VERIFIED"
                    and len(package_checks) == 1
                    and all(package_checks)
                    and global_passed
                    and canonical_unchanged
                    and integrated_ok
                )
            except Exception:
                logs.append("Mission detail endpoint could not be summarized safely.")
        sanitized_tail = safe_text(
            "\n".join((stdout + "\n" + stderr).splitlines()[-12:]),
            root=root,
            secrets=(bootstrap_token, selected_profile),
        )
        if sanitized_tail.strip():
            logs.append("Sanitized CLI tail: " + " | ".join(line[:300] for line in sanitized_tail.splitlines()))

        runtimes = query_runtimes(workspace, env)
        if runtimes:
            logs.append(f"Active runtimes before isolated cleanup: {len(runtimes)}")
            for runtime in runtimes:
                runtime_id = runtime.get("id") or runtime.get("runtime_id")
                if runtime_id:
                    run_cli(["stop", str(runtime_id)], cwd=workspace, env=env, timeout=20)
            time.sleep(1)
            runtimes = query_runtimes(workspace, env)
        runtime_count_after = len(runtimes)
        logs.append(f"Active runtimes after request/cleanup: {runtime_count_after}")

        logs.append(f"nexus run exit={run_code}; mission worktree delivery and canonical isolation verified={provider_task_verified}")
        if run_code == 0 and provider_task_verified and runtime_count_after == 0:
            outcome = "PASS"
            detail = "authenticated Codex run completed and verified the one-file task in its worktree, preserved the canonical fixture, and left no runtime"
        else:
            outcome = "BLOCKED"
            detail = "provider-backed run did not satisfy the fixture assertion or left runtime state"
    except Exception as exc:  # noqa: BLE001 - evidence harness records a safe summary and continues cleanup.
        if outcome != "BLOCKED" or detail == "not completed":
            detail = f"{type(exc).__name__}: {exc}"
        logs.append("Run status: " + safe_text(detail[:240], root=root, secrets=(bootstrap_token, selected_profile)))
    finally:
        if server is not None and server.poll() is None:
            os.killpg(server.pid, signal.SIGTERM)
            try:
                server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(server.pid, signal.SIGKILL)
                server.wait(timeout=10)
        if server is not None:
            logs.append(f"Isolated Core stopped; exit={server.returncode}")
        logs.append(f"Final result: {outcome} — {detail}")

        log_path = REPO / "DEV/validation/linux-100-2026-10-25/logs/runtime-real-nexus-run-isolated-worktree-2026-10-26.log"
        log_path.write_text("\n".join(logs) + "\n", encoding="utf-8")
        safe_to_remove = not run_started or runtime_count_after == 0
        if safe_to_remove and (server is None or server.poll() is not None):
            shutil.rmtree(root)
            with log_path.open("a", encoding="utf-8") as stream:
                stream.write("Temporary imported credentials and workspace removed after process shutdown.\n")
        else:
            with log_path.open("a", encoding="utf-8") as stream:
                stream.write("Isolated root retained because process/runtime cleanup was not fully confirmed.\n")

        print(
            json.dumps(
                {
                    "result": outcome,
                    "detail": detail,
                    "codex_status": next(
                        (status for provider, status in imported_statuses if provider == "codex"), None
                    ),
                    "runtime_count_after": runtime_count_after,
                    "log": str(log_path),
                    "temporary_root_removed": not root.exists(),
                },
                ensure_ascii=False,
            )
        )
    return 0 if outcome == "PASS" else 2


if __name__ == "__main__":
    raise SystemExit(main())
