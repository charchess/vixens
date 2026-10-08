"""Regression tests for three pinned Hermes GitOps channels."""
import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "guard-hermes-release-channels.py"
spec = importlib.util.spec_from_file_location("hermes_guard", SCRIPT)
guard = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guard)

OLD = "hermes-2026-09-24-baseline"
NEW = "hermes-0-21-6-candidate"
IMG = "nousresearch/hermes-agent@sha256:" + "a" * 64
PLUGIN = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + "b" * 64
LINK = "https://github.com/charchess/vixens/issues/3688#issuecomment-123"


class ChannelGateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        prev = guard.PROFILES
        guard.PROFILES = Path(self.tmp.name)
        self.addCleanup(setattr, guard, "PROFILES", prev)
        for name in (OLD, NEW):
            (guard.PROFILES / (name + ".yaml")).write_text(
                "kind: HermesRuntimeRelease\nmetadata:\n  name: " + name +
                "\nspec:\n  image: " + IMG + "\n  hindsightPluginImage: " + PLUGIN + "\n",
                encoding="utf-8")
        self.ref("hermes-edge", NEW)
        self.ref("hermes-canary", NEW)
        self.ref("hermes-stable", OLD)

    def ref(self, channel, release):
        (guard.PROFILES / (channel + ".yaml")).write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: " + channel +
            "\nspec:\n  releaseRef: " + release + "\n", encoding="utf-8")

    def fake_git(self, channels=(), bootstrap=False, history=True, destructive=False, tenants=False):
        changes = [str(guard.PROFILES / (c + ".yaml")) for c in channels]
        if tenants:
            changes.append("apps/60-services/txo-fabric/tenants/hairem/agents.yaml")
        def runner(*args):
            if args[0] == "diff":
                if "--diff-filter=MD" in args:
                    return str(guard.PROFILES / (OLD + ".yaml")) if destructive else ""
                return "\n".join(changes)
            if args[0] == "ls-tree":
                return "" if bootstrap else args[-1]
            if args[0] == "log":
                return "a" * 40 if history else ""
            raise AssertionError(args)
        return runner

    @staticmethod
    def stable_body():
        return (
            "Hermes-Physical-Evidence: " + LINK + "\n"
            "Hermes-Recovery-Evidence: " + LINK + "\n"
            "Hermes-Owner-Approval: " + LINK + "\n"
            "Hermes-GitOps-Snapshot: dev-v2026.10.3948\n"
        )

    def test_non_channel_pr_is_not_gated(self):
        with patch.object(guard, "git", side_effect=self.fake_git()):
            self.assertEqual(guard.gate("origin/main", "", "Bot"), [])

    def test_single_bootstrap_creates_unbound_profiles_only(self):
        channels = guard.CHANNELS
        with patch.object(guard, "git", side_effect=self.fake_git(channels, bootstrap=True)):
            self.assertEqual(guard.gate("origin/main", "Hermes-Channel-Bootstrap: true", "User"), list(channels))
            self.assertEqual(guard.gate("origin/main", "- Hermes-Channel-Bootstrap: true", "User"), list(channels))
            with self.assertRaisesRegex(ValueError, "human-reviewed"):
                guard.gate("origin/main", "Hermes-Channel-Bootstrap: true", "Bot")
            with self.assertRaisesRegex(ValueError, "Bootstrap"):
                guard.gate("origin/main", "", "User")
        with patch.object(guard, "git", side_effect=self.fake_git(channels, bootstrap=True, tenants=True)):
            with self.assertRaisesRegex(ValueError, "AgentIdentity"):
                guard.gate("origin/main", "Hermes-Channel-Bootstrap: true", "User")
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-stable",), bootstrap=True)):
            with self.assertRaisesRegex(ValueError, "all three"):
                guard.gate("origin/main", "Hermes-Channel-Bootstrap: true", "User")

    def test_edge_bot_can_nominate_but_canary_bot_cannot(self):
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-edge",))):
            self.assertEqual(guard.gate("origin/main", "", "Bot"), ["hermes-edge"])
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-canary",))):
            with self.assertRaisesRegex(ValueError, "bot"):
                guard.gate("origin/main", "", "Bot")

    def test_canary_follows_committed_edge_history_but_not_physical_gate(self):
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-canary",))):
            self.assertEqual(guard.gate("origin/main", "", "User"), ["hermes-canary"])
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-canary",), history=False)):
            with self.assertRaisesRegex(ValueError, "edge history"):
                guard.gate("origin/main", "", "User")

    def test_stable_requires_canary_history_and_actual_evidence_links(self):
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-stable",))):
            with self.assertRaisesRegex(ValueError, "Physical-Evidence"):
                guard.gate("origin/main", "", "User")
            self.assertEqual(guard.gate("origin/main", self.stable_body(), "User"), ["hermes-stable"])
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-stable",), history=False)):
            with self.assertRaisesRegex(ValueError, "canary history"):
                guard.gate("origin/main", self.stable_body(), "User")

    def test_rejects_unpinned_or_non_official_release_and_inline_image(self):
        path = guard.PROFILES / (NEW + ".yaml")
        path.write_text(path.read_text().replace(IMG, "nousresearch/hermes-agent:latest"))
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-edge",))):
            with self.assertRaisesRegex(ValueError, "digest"):
                guard.gate("origin/main", "", "User")
        path.write_text(path.read_text().replace("nousresearch/hermes-agent:latest",
                            "evil/hermes@sha256:" + "a"*64))
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-edge",))):
            with self.assertRaisesRegex(ValueError, "official source"):
                guard.gate("origin/main", "", "User")
        path.write_text(path.read_text().replace("evil/hermes@sha256:"+"a"*64, IMG))
        profile = guard.PROFILES / "hermes-edge.yaml"
        profile.write_text(profile.read_text() + "  image: forbidden:latest\n")
        with patch.object(guard, "git", side_effect=self.fake_git(("hermes-edge",))):
            with self.assertRaisesRegex(ValueError, "releaseRef only"):
                guard.gate("origin/main", "", "User")

    def test_immutable_release_edits_are_always_blocked(self):
        with patch.object(guard, "git", side_effect=self.fake_git(destructive=True)):
            with self.assertRaisesRegex(ValueError, "immutable"):
                guard.gate("origin/main", "", "User")


if __name__ == "__main__":
    unittest.main()
