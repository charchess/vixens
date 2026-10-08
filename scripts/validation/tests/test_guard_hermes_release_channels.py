"""Offline regression tests for the Hermes release-channel PR guard."""

import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "guard-hermes-release-channels.py"
spec = importlib.util.spec_from_file_location("hermes_guard", SCRIPT)
guard = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guard)

REF = "hermes-2026-09-24-test"
IMAGE = "nousresearch/hermes-agent@sha256:" + "a" * 64
PLUGIN = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + "b" * 64
PHYSICAL = "https://github.com/charchess/vixens/issues/3688#issuecomment-123"
RECOVERY = "https://github.com/charchess/vixens/issues/3688#issuecomment-456"
APPROVAL = "https://github.com/charchess/vixens/issues/3947#issuecomment-789"


class GateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.old_profiles = guard.PROFILES
        guard.PROFILES = Path(self.tmp.name)
        self.addCleanup(setattr, guard, "PROFILES", self.old_profiles)
        (guard.PROFILES / f"{REF}.yaml").write_text(
            "kind: HermesRuntimeRelease\nmetadata:\n  name: " + REF + "\n"
            "spec:\n  image: " + IMAGE + "\n"
            "  hindsightPluginImage: " + PLUGIN + "\n", encoding="utf-8")
        for channel in ("hermes-test", "hermes-stable"):
            (guard.PROFILES / f"{channel}.yaml").write_text(
                "kind: AgentRuntimeProfile\nmetadata:\n  name: " + channel + "\n"
                "spec:\n  releaseRef: " + REF + "\n", encoding="utf-8")

    def body(self, stable=False):
        text = ("Hermes-Physical-Evidence: " + PHYSICAL + "\n"
                "Hermes-Recovery-Evidence: " + RECOVERY + "\n")
        if stable:
            text += ("Hermes-Owner-Approval: " + APPROVAL + "\n"
                     "Hermes-GitOps-Snapshot: dev-v2026.10.3948\n")
        return text

    def fake_git(self, changed, history=True):
        def call(*args):
            if args[0] == "diff":
                return str(guard.PROFILES / f"{changed}.yaml") if changed else ""
            if args[0] == "log":
                return "a" * 40 if history else ""
            raise AssertionError(args)
        return call

    def test_ordinary_candidate_has_no_test_or_stable_gate(self):
        with patch.object(guard, "git", side_effect=self.fake_git(None)):
            self.assertEqual(guard.gate("origin/main", "", "Bot"), [])

    def test_bot_cannot_advance_test(self):
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-test")):
            with self.assertRaisesRegex(ValueError, "bot-authored"):
                guard.gate("origin/main", self.body(), "Bot")

    def test_test_requires_evidence_and_prior_dev_history(self):
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-test")):
            with self.assertRaisesRegex(ValueError, "Physical"):
                guard.gate("origin/main", "", "User")
            self.assertEqual(guard.gate("origin/main", self.body(), "User"), ["hermes-test"])
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-test", history=False)):
            with self.assertRaisesRegex(ValueError, "never selected"):
                guard.gate("origin/main", self.body(), "User")

    def test_stable_requires_manual_approval_exact_snapshot_and_test_history(self):
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-stable")):
            with self.assertRaisesRegex(ValueError, "Owner-Approval"):
                guard.gate("origin/main", self.body(), "User")
            self.assertEqual(guard.gate("origin/main", self.body(True), "User"), ["hermes-stable"])
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-stable", history=False)):
            with self.assertRaisesRegex(ValueError, "hermes-test"):
                guard.gate("origin/main", self.body(True), "User")

    def test_bad_release_digest_and_inline_image_fail_closed(self):
        release = guard.PROFILES / f"{REF}.yaml"
        release.write_text(release.read_text().replace(PLUGIN, "plugin:latest"), encoding="utf-8")
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-test")):
            with self.assertRaisesRegex(ValueError, "digest"):
                guard.gate("origin/main", self.body(), "User")
        release.write_text(release.read_text().replace("plugin:latest", PLUGIN), encoding="utf-8")
        profile = guard.PROFILES / "hermes-test.yaml"
        profile.write_text(profile.read_text() + "  image: legacy:mutable\n", encoding="utf-8")
        with patch.object(guard, "git", side_effect=self.fake_git("hermes-test")):
            with self.assertRaisesRegex(ValueError, "releaseRef only"):
                guard.gate("origin/main", self.body(), "User")


if __name__ == "__main__":
    unittest.main()
