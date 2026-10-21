"""Regression tests for fail-closed native HUD cross-platform certification."""
import datetime as dt
import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parent
AGGREGATOR = ROOT / "verify-hud-native-evidence.py"
SHA = "a" * 40
BIN_HASH = "b" * 64


class HUDNativeEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = pathlib.Path(self.tmp.name)
        self.platforms = {p: self.root / p for p in ("windows", "linux-x11", "darwin")}
        for path in self.platforms.values():
            path.mkdir(parents=True)

    def create_report(self, platform, *, review=False, sha=SHA, cycles=20):
        folder = self.platforms[platform]
        data = {
            "sha": sha,
            "platform": platform,
            "verdict": "PASS_FUNCTIONAL_PENDING_VISUAL",
            "cycles_requested": 20,
            "cycles_completed": cycles,
            "binary_sha256": BIN_HASH,
            "visual_shadow_review": "PASS" if review else "NOT_TESTED",
            "transparent_click_through": "PASS" if review else "NOT_TESTED",
        }
        if review:
            data["review"] = {
                "reviewer": "native-platform-reviewer",
                "reviewed_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
            }
            data["build_attestation"] = {"source_sha": sha, "binary_sha256": BIN_HASH}
        (folder / "HUD_NATIVE_RESULTS.json").write_text(json.dumps(data), encoding="utf-8")
        for name in ("HUD_BEFORE.png", "HUD_EXPANDED.png"):
            (folder / name).write_bytes(b"\x89PNG\r\n\x1a\n" + b"a" * 700)

    def verify(self):
        args = [
            sys.executable, str(AGGREGATOR),
            "--sha", SHA,
            "--windows", str(self.platforms["windows"]),
            "--linux-x11", str(self.platforms["linux-x11"]),
            "--darwin", str(self.platforms["darwin"]),
            "--output", str(self.root / "summary.json"),
        ]
        proc = subprocess.run(args, capture_output=True, text=True, check=False)
        return proc, json.loads((self.root / "summary.json").read_text(encoding="utf-8"))

    def test_missing_platforms_remain_inconclusive(self):
        proc, report = self.verify()
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(report["verdict"], "INCONCLUSIVE")

    def test_click_success_cannot_automatically_claim_visual_parity(self):
        for platform in self.platforms:
            self.create_report(platform)
        proc, report = self.verify()
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(report["verdict"], "INCONCLUSIVE")
        self.assertTrue(any("manual native review" in issue
                            for item in report["platforms"] for issue in item["issues"]))

    def test_mismatched_sha_blocks_certification(self):
        for platform in self.platforms:
            self.create_report(platform, review=True)
        self.create_report("darwin", review=True, sha="c" * 40)
        proc, report = self.verify()
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(report["verdict"], "INCONCLUSIVE")
        self.assertIn("source SHA mismatch", report["platforms"][2]["issues"])

    def test_all_approved_same_sha_evidence_is_go(self):
        for platform in self.platforms:
            self.create_report(platform, review=True)
        proc, report = self.verify()
        self.assertEqual(proc.returncode, 0)
        self.assertEqual(report["verdict"], "GO")

    def test_failed_native_mouse_test_is_no_go(self):
        for platform in self.platforms:
            self.create_report(platform, review=True)
        file = self.platforms["windows"] / "HUD_NATIVE_RESULTS.json"
        data = json.loads(file.read_text())
        data["verdict"] = "NO_GO"
        file.write_text(json.dumps(data))
        proc, report = self.verify()
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(report["verdict"], "NO_GO")


if __name__ == "__main__":
    unittest.main()
