"""Offline candidate generator tests; no GitHub token or Kubernetes required."""

import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "nominate_dev.py"
spec = importlib.util.spec_from_file_location("nominate_dev", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

ENGINE = "nousresearch/hermes-agent@sha256:" + "a" * 64
PLUGIN = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + "b" * 64


class NominateDevTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.directory = self.root / module.RELATIVE
        self.directory.mkdir(parents=True)
        (self.directory / "kustomization.yaml").write_text(
            "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n"
            "  - hermes-default.yaml\n  - hermes-dev.yaml\n", encoding="utf-8")

    def test_legacy_inline_promotion_is_idempotent_and_only_canary_changes(self):
        default = self.directory / "hermes-default.yaml"
        default.write_text("spec:\n  image: ghcr.io/charchess/legacy:old\n", encoding="utf-8")
        canary = self.directory / "hermes-dev.yaml"
        canary.write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: hermes-dev\n"
            "spec:\n  image: nousresearch/hermes-agent:v2026.9.24\n"
            "  bootstrap:\n    hindsightPluginImage: ghcr.io/charchess/bundle:old\n"
            "  storage:\n    storageClassName: truenas-iscsi-retain\n", encoding="utf-8")
        name = module.nominate(self.root, ENGINE, PLUGIN)
        self.assertIn("releaseRef: " + name, canary.read_text())
        self.assertNotIn("  image:", canary.read_text())
        self.assertNotIn("  bootstrap:", canary.read_text())
        self.assertIn("truenas-iscsi-retain", canary.read_text())
        self.assertEqual(default.read_text(), "spec:\n  image: ghcr.io/charchess/legacy:old\n")
        rel = self.directory / f"{name}.yaml"
        self.assertIn(ENGINE, rel.read_text())
        self.assertIn(PLUGIN, rel.read_text())
        snapshot = [p.read_bytes() for p in (rel, canary, self.directory / "kustomization.yaml")]
        self.assertEqual(name, module.nominate(self.root, ENGINE, PLUGIN))
        self.assertEqual(snapshot, [p.read_bytes() for p in (rel, canary, self.directory / "kustomization.yaml")])
        self.assertEqual((self.directory / "kustomization.yaml").read_text().count(rel.name), 1)

    def test_reject_mutable_or_untrusted_images(self):
        canary = self.directory / "hermes-dev.yaml"
        canary.write_text("spec:\n  image: legacy:tag\n", encoding="utf-8")
        for engine, plugin in [(ENGINE.replace("@sha256:", ":v"), PLUGIN),
                               (ENGINE, PLUGIN.replace("@sha256:", ":main-")),
                               ("ghcr.io/another/hermes@sha256:" + "a" * 64, PLUGIN)]:
            with self.assertRaises(ValueError):
                module.nominate(self.root, engine, plugin)
        self.assertEqual(canary.read_text(), "spec:\n  image: legacy:tag\n")

    def test_refuses_to_erase_unrecognized_bootstrap_policy(self):
        canary = self.directory / "hermes-dev.yaml"
        canary.write_text(
            "spec:\n  image: legacy\n  bootstrap:\n"
            "    hindsightPluginImage: old\n    futureSetting: keep-me\n"
            "  storage:\n    storageClassName: persistent\n", encoding="utf-8")
        with self.assertRaises(ValueError):
            module.nominate(self.root, ENGINE, PLUGIN)



    def test_other_runtime_channels_are_untouched(self):
        dev = self.directory / "hermes-dev.yaml"
        dev.write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: hermes-dev\n"
            "spec:\n  releaseRef: known-candidate\n", encoding="utf-8")
        originals = {}
        for name in ("hermes-default", "hermes-upgrade-canary", "hermes-test", "hermes-stable"):
            path = self.directory / f"{name}.yaml"
            path.write_text("kind: AgentRuntimeProfile\nmetadata:\n  name: " + name + "\n"
                            "spec:\n  releaseRef: known-good\n", encoding="utf-8")
            originals[path] = path.read_bytes()
        module.nominate(self.root, ENGINE, PLUGIN)
        for path, expected in originals.items():
            self.assertEqual(path.read_bytes(), expected, str(path))

    def test_immutable_release_cannot_be_edited_by_regeneration(self):
        dev = self.directory / "hermes-dev.yaml"
        dev.write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: hermes-dev\n"
            "spec:\n  releaseRef: previous\n", encoding="utf-8")
        name = module.nominate(self.root, ENGINE, PLUGIN)
        release = self.directory / f"{name}.yaml"
        release.write_text(release.read_text() + "# modified by hand\n", encoding="utf-8")
        dev_before = dev.read_bytes()
        with self.assertRaisesRegex(ValueError, "immutable release"):
            module.nominate(self.root, ENGINE, PLUGIN)
        self.assertEqual(dev_before, dev.read_bytes())

    def test_refuse_to_modify_an_unexpected_profile(self):
        path = self.directory / "hermes-dev.yaml"
        path.write_text(
            "kind: AgentRuntimeProfile\nmetadata:\n  name: hermes-stable\n"
            "spec:\n  releaseRef: protected\n", encoding="utf-8")
        before = path.read_bytes()
        with self.assertRaisesRegex(ValueError, "unexpected dev profile"):
            module.nominate(self.root, ENGINE, PLUGIN)
        self.assertEqual(before, path.read_bytes())

if __name__ == "__main__":
    unittest.main()
