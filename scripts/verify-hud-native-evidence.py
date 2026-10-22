#!/usr/bin/env python3
"""Fail-closed, same-SHA aggregation of interactive HUD native evidence.

The native probes verify actual click/geometry and capture PNGs. A human reviewer
must independently mark shadow and transparent click-through as PASS, with a
reviewer identifier and UTC timestamp, before this script can emit GO.
"""
import argparse
import datetime as dt
import hashlib
import json
import pathlib
import re
import sys
import struct
import zlib

PLATFORMS = ("windows", "linux-x11", "darwin")
SHA = re.compile(r"^[0-9a-f]{40}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")


def png_dimensions(path: pathlib.Path) -> tuple[int, int] | None:
    """Reject corrupt/fake PNG headers without decoding sensitive screenshots."""
    try:
        with path.open("rb") as handle:
            header = handle.read(33)
    except OSError:
        return None
    if len(header) != 33 or not header.startswith(b"\x89PNG\r\n\x1a\n"):
        return None
    if header[8:16] != b"\x00\x00\x00\rIHDR":
        return None
    if (zlib.crc32(header[12:29]) & 0xFFFFFFFF) != struct.unpack(">I", header[29:33])[0]:
        return None
    width, height = struct.unpack(">II", header[16:24])
    if width < 80 or height < 40 or width > 16384 or height > 16384:
        return None
    return width, height


def validate(root: pathlib.Path, platform: str, sha: str, min_cycles: int) -> dict:
    results = {"platform": platform, "status": "INCONCLUSIVE", "issues": []}
    path = root / "HUD_NATIVE_RESULTS.json"
    if not path.is_file():
        results["issues"].append("missing HUD_NATIVE_RESULTS.json")
        return results
    try:
        data = json.loads(path.read_text(encoding="utf-8-sig"))
    except (ValueError, UnicodeError, OSError) as exc:
        results["issues"].append("invalid report: " + str(type(exc).__name__))
        return results
    if not isinstance(data, dict):
        results["issues"].append("native report must be an object")
        return results
    actual = data.get("platform")
    if actual != platform:
        results["issues"].append("platform mismatch")
    if data.get("sha") != sha:
        results["issues"].append("source SHA mismatch")
    verdict = data.get("verdict")
    if verdict == "NO_GO":
        results["status"] = "NO_GO"
        results["issues"].append("native click/geometry test failed")
    elif verdict != "PASS_FUNCTIONAL_PENDING_VISUAL":
        results["issues"].append("native click/geometry test not passed")
    completed = data.get("cycles_completed")
    if type(completed) is not int or completed < min_cycles:
        results["issues"].append("insufficient successful click/close cycles")
    if type(data.get("cycles_requested")) is not int or completed != data.get("cycles_requested"):
        results["issues"].append("not all requested cycles were completed")

    captures = {}
    for image in ("HUD_BEFORE.png", "HUD_EXPANDED.png"):
        file = root / image
        if not file.is_file() or file.stat().st_size < 400:
            results["issues"].append("missing/empty screenshot: " + image)
        elif png_dimensions(file) is None:
            results["issues"].append("invalid PNG screenshot: " + image)
        else:
            captures[image] = hashlib.sha256(file.read_bytes()).hexdigest()
    results["capture_sha256"] = captures

    # This cannot be derived from a unit test or browser screenshot. The person
    # inspecting native screenshots/input regions explicitly signs this review.
    for criterion in ("visual_shadow_review", "transparent_click_through"):
        if data.get(criterion) == "FAIL":
            results["status"] = "NO_GO"
            results["issues"].append("native visual/input defect confirmed: " + criterion)
        elif data.get(criterion) != "PASS":
            results["issues"].append("manual native review required: " + criterion)
    review = data.get("review", {})
    if not isinstance(review, dict) or not review.get("reviewer") or not review.get("reviewed_at_utc"):
        results["issues"].append("reviewer and reviewed_at_utc required")
    else:
        try:
            when = dt.datetime.fromisoformat(review["reviewed_at_utc"].replace("Z", "+00:00"))
            if when.tzinfo is None or when > dt.datetime.now(dt.timezone.utc) + dt.timedelta(minutes=5):
                raise ValueError("invalid UTC review timestamp")
        except (ValueError, AttributeError):
            results["issues"].append("invalid reviewed_at_utc")

    # The launched binary must be linked to the tested source, not an unrelated
    # pre-existing build. The reviewer records SHA256 of the executable and a
    # build attestation from the exact SHA. Native scripts must never invent it.
    binary_hash = data.get("binary_sha256")
    if not isinstance(binary_hash, str) or not SHA256.fullmatch(binary_hash):
        results["issues"].append("binary_sha256 missing/invalid")
    build = data.get("build_attestation", {})
    if not isinstance(build, dict) or build.get("source_sha") != sha or build.get("binary_sha256") != binary_hash:
        results["issues"].append("matching exact-SHA build attestation required")
    if not results["issues"]:
        results["status"] = "GO"
    return results


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--sha", required=True, help="full immutable candidate SHA")
    parser.add_argument("--windows", type=pathlib.Path, required=True)
    parser.add_argument("--linux-x11", type=pathlib.Path, required=True)
    parser.add_argument("--darwin", type=pathlib.Path, required=True)
    parser.add_argument("--min-cycles", type=int, default=20)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not SHA.fullmatch(args.sha):
        parser.error("--sha must be a full 40-character lowercase commit SHA")
    if args.min_cycles < 1:
        parser.error("--min-cycles must be positive")
    locations = {"windows": args.windows, "linux-x11": args.linux_x11, "darwin": args.darwin}
    reports = [validate(locations[p], p, args.sha, args.min_cycles) for p in PLATFORMS]
    overall = "GO" if all(x["status"] == "GO" for x in reports) else (
        "NO_GO" if any(x["status"] == "NO_GO" for x in reports) else "INCONCLUSIVE"
    )
    document = {
        "schema_version": 1,
        "source_sha": args.sha,
        "generated_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
        "verdict": overall,
        "platforms": reports,
        "note": "macOS/Windows/Linux X11 graphical evidence only; Wayland remains a separate compatibility gate.",
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(document, indent=2) + "\n", encoding="utf-8")
    for r in reports:
        print(r["platform"], r["status"], "; ".join(r["issues"]) or "native evidence complete")
    print("Overall:", overall)
    return 0 if overall == "GO" else 1


if __name__ == "__main__":
    sys.exit(main())
