"""Offline regression tests for edge-only pinned-release nomination."""
import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "nominate_edge.py"
spec = importlib.util.spec_from_file_location("nominate_edge", SCRIPT)
nom = importlib.util.module_from_spec(spec)
spec.loader.exec_module(nom)

ENGINE = "nousresearch/hermes-agent@sha256:" + "a"*64
PLUGIN = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + "b"*64
TAG = "v0.21.6"


class NominationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.folder = self.root / nom.RELATIVE
        self.folder.mkdir(parents=True)
        (self.folder / "kustomization.yaml").write_text("kind: Kustomization\nresources:\n  - hermes-edge.yaml\n")
        (self.folder / "hermes-edge.yaml").write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: hermes-edge\n"
            "spec:\n  releaseRef: previous-release\n  storage:\n    size: 2Gi\n")
        self.originals = {}
        for channel in ("hermes-stable", "hermes-canary", "hermes-dev",
                        "hermes-default", "hermes-upgrade-canary"):
            p = self.folder / (channel + ".yaml")
            p.write_text("kind: AgentRuntimeProfile\nmetadata:\n  name: " + channel + "\n")
            self.originals[p] = p.read_bytes()

    def test_pins_edge_only_is_idempotent_and_leaves_legacy_untouched(self):
        new = nom.nominate(self.root, ENGINE, PLUGIN, TAG)
        self.assertEqual(new, "hermes-0-21-6-aaaaaaaaaaaa-bbbbbbbbbbbb")
        target = self.folder / (new + ".yaml")
        self.assertIn("upstream-tag: \"v0.21.6\"", target.read_text())
        self.assertIn("releaseRef: " + new, (self.folder / "hermes-edge.yaml").read_text())
        self.assertIn("size: 2Gi", (self.folder / "hermes-edge.yaml").read_text())
        snapshot = [(self.folder / p).read_bytes() for p in (new + ".yaml", "hermes-edge.yaml", "kustomization.yaml")]
        self.assertEqual(nom.nominate(self.root, ENGINE, PLUGIN, TAG), new)
        self.assertEqual(snapshot, [(self.folder / p).read_bytes() for p in (new + ".yaml", "hermes-edge.yaml", "kustomization.yaml")])
        for p, data in self.originals.items():
            self.assertEqual(p.read_bytes(), data)

    def test_mutable_and_unofficial_tags_and_images_rejected_without_mutation(self):
        previous = (self.folder / "hermes-edge.yaml").read_bytes()
        for engine, plugin, tag in (
            (ENGINE.replace("@sha256:", ":latest"), PLUGIN, TAG),
            (ENGINE, PLUGIN.replace("@sha256:", ":latest"), TAG),
            ("evil/hermes@sha256:"+"a"*64, PLUGIN, TAG),
            (ENGINE, "evil/plugin@sha256:"+"b"*64, TAG),
            (ENGINE, PLUGIN, "latest"),
        ):
            with self.assertRaises(ValueError):
                nom.nominate(self.root, engine, plugin, tag)
        self.assertEqual((self.folder / "hermes-edge.yaml").read_bytes(), previous)

    def test_never_modifies_immutable_release(self):
        name = nom.nominate(self.root, ENGINE, PLUGIN, TAG)
        release = self.folder / (name + ".yaml")
        release.write_text(release.read_text() + "# tampered\n")
        before = (self.folder / "hermes-edge.yaml").read_bytes()
        with self.assertRaisesRegex(ValueError, "immutable"):
            nom.nominate(self.root, ENGINE, PLUGIN, TAG)
        self.assertEqual((self.folder / "hermes-edge.yaml").read_bytes(), before)

    def test_refuses_unexpected_edge_profile(self):
        profile = self.folder / "hermes-edge.yaml"
        profile.write_text(profile.read_text().replace("name: hermes-edge", "name: hermes-stable"))
        with self.assertRaisesRegex(ValueError, "unexpected edge"):
            nom.nominate(self.root, ENGINE, PLUGIN, TAG)


if __name__ == "__main__":
    unittest.main()
