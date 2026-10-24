"""Regression contract for #4185: UMI ESO bootstrap is dev-only and secret-free.

This is a *source* guard, not evidence of a valid token, safe transport, or
runtime ESO readiness. A missing UMI bootstrap Secret must fail closed.
"""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
BASE = "apps/00-infra/openbao/base/cluster-secret-store.yaml"
DEV = "apps/00-infra/openbao/overlays/dev/kustomization.yaml"
PROD = "apps/00-infra/openbao/overlays/prod/kustomization.yaml"


def load(path: str) -> dict:
    with (ROOT / path).open(encoding="utf-8") as stream:
        documents = list(yaml.safe_load_all(stream))
    if len(documents) != 1 or not isinstance(documents[0], dict):
        raise AssertionError(f"{path} must have exactly one document")
    return documents[0]


def apply_umi_patch(overlay: dict) -> dict:
    """Accept only a single existing tokenSecretRef.name replacement."""
    patched = deepcopy(load(BASE))
    self_ref = patched["spec"]["provider"]["vault"]["auth"]["tokenSecretRef"]
    if patched["kind"] != "ClusterSecretStore":
        raise AssertionError("unexpected base kind")
    patches = overlay.get("patches", [])
    if len(patches) != 1:
        raise AssertionError("expected exactly one dev-only bootstrap patch")
    p = patches[0]
    if p.get("target") != {"kind": "ClusterSecretStore", "name": "openbao"}:
        raise AssertionError("unexpected UMI patch target")
    operations = yaml.safe_load(p["patch"])
    if operations != [{
        "op": "replace",
        "path": "/spec/provider/vault/auth/tokenSecretRef/name",
        "value": "openbao-token-umi",
    }]:
        raise AssertionError("unexpected UMI store mutation")
    self_ref["name"] = "openbao-token-umi"
    return patched


class DedicatedUMIESOIdentityTests(unittest.TestCase):
    def test_dev_references_dedicated_existing_namespace_secret_only(self):
        base = load(BASE)
        dev_overlay = load(DEV)
        self.assertEqual(dev_overlay["resources"], ["../../base"])
        applied = apply_umi_patch(dev_overlay)
        original_vault = base["spec"]["provider"]["vault"]
        changed_vault = applied["spec"]["provider"]["vault"]
        self.assertEqual(base["metadata"]["name"], "openbao")
        self.assertEqual(original_vault["auth"]["tokenSecretRef"], {
            "namespace": "external-secrets",
            "name": "openbao-token",
            "key": "token",
        })
        self.assertEqual(changed_vault["auth"]["tokenSecretRef"], {
            "namespace": "external-secrets",
            "name": "openbao-token-umi",
            "key": "token",
        })
        # No path/server/engine/auth-method changes are allowed in this slice.
        self.assertEqual(changed_vault["server"], original_vault["server"])
        self.assertEqual(changed_vault["path"], original_vault["path"])
        self.assertEqual(changed_vault["version"], original_vault["version"])
        self.assertEqual(changed_vault["server"], "http://nas.truxonline.com:8200")

    def test_prod_remains_on_base_bootstrap_secret_and_prod_stable(self):
        prod_overlay = load(PROD)
        self.assertEqual(prod_overlay["resources"], ["../../base"])
        self.assertNotIn("patches", prod_overlay)
        prod_app = load("argocd/overlays/prod/apps/openbao-secrets.yaml")
        self.assertEqual(prod_app["spec"]["source"]["targetRevision"], "prod-stable")
        self.assertEqual(
            prod_app["spec"]["source"]["path"],
            "apps/00-infra/openbao/overlays/prod",
        )
        dev_app = load("argocd/overlays/dev/apps/openbao-secrets.yaml")
        self.assertEqual(dev_app["spec"]["source"]["targetRevision"], "main")
        self.assertEqual(
            dev_app["spec"]["source"]["path"],
            "apps/00-infra/openbao/overlays/dev",
        )

    def test_reject_wrong_secret_or_changed_endpoint(self):
        overlay = load(DEV)
        for replacement in ("openbao-token", "prod-token", "umi-eso"):
            with self.subTest(name=replacement):
                bad = deepcopy(overlay)
                bad["patches"][0]["patch"] = bad["patches"][0]["patch"].replace(
                    "openbao-token-umi", replacement
                )
                with self.assertRaises(AssertionError):
                    apply_umi_patch(bad)
        bad = deepcopy(overlay)
        bad["patches"][0]["patch"] += (
            "\n- op: replace\n  path: /spec/provider/vault/server"
            "\n  value: https://example.invalid:8200"
        )
        with self.assertRaises(AssertionError):
            apply_umi_patch(bad)


if __name__ == "__main__":
    unittest.main()
