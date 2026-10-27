"""#4193 source safety guard for a manually gated official UMI Synelia CSI.

This proves manifest invariants only; physical TrueNAS 27 / legacy migration
remain separate gates. Run: python3 -m unittest discover -s
scripts/validation/tests -p 'test_synelia_csi_source.py' -v
"""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
import unittest
from urllib.parse import urlparse

import yaml


ROOT = Path(__file__).resolve().parents[3]
BASE = "apps/01-storage/synelia-csi/umi"
APP = "argocd/overlays/dev/apps/synelia-csi-umi.yaml"
OVERLAY = "apps/01-storage/synelia-csi/overlays/dev"


def read_one(path: str) -> dict:
    with (ROOT / path).open(encoding="utf-8") as handle:
        return yaml.safe_load(handle)


def verify_classes(values: dict) -> None:
    classes = values["storageClasses"]
    names = [sc["name"] for sc in classes]
    expected = {
        "storage-rwo-delete",
        "storage-rwo-retain",
        "storage-rwo-xfs-delete",
        "storage-rwo-xfs-retain",
        "storage-rwx-delete",
        "storage-rwx-retain",
    }
    if len(names) != len(set(names)) or set(names) != expected:
        raise AssertionError(f"Unexpected classes or class collision: {names}")
    for sc in classes:
        name = sc["name"]
        if not name.startswith("storage-rw"):
            raise AssertionError(f"Provider-specific class forbidden: {name}")
        if sc.get("isDefault") is not False:
            raise AssertionError(f"Default class not permitted: {name}")
        if sc["reclaimPolicy"] != name.rsplit("-", 1)[-1].capitalize():
            raise AssertionError(f"Wrong reclaim policy: {name}")
        protocol = sc["parameters"].get("protocol")
        parent = sc["parameters"]["datasetPath"]
        if name.startswith("storage-rwo-"):
            expected_fs = "xfs" if name.startswith("storage-rwo-xfs-") else "ext4"
            if (protocol != "iscsi"
                    or sc["volumeBindingMode"] != "WaitForFirstConsumer"
                    or sc["parameters"].get("fsType") != expected_fs):
                raise AssertionError(
                    f"RWO contract must bind iSCSI/{expected_fs} safely: {name}"
                )
            expected_parent = "k8s/vixens/iscsi"
        elif name.startswith("storage-rwx-"):
            if (protocol != "nfs"
                    or sc["volumeBindingMode"] != "Immediate"
                    or "nfsvers=4.1" not in sc.get("mountOptions", [])):
                raise AssertionError(f"RWX contract must bind NFSv4.1: {name}")
            expected_parent = "k8s/vixens/nfs"
        else:
            raise AssertionError(f"Unexpected access contract: {name}")
        if sc["parameters"].get("pool") != "tank" or parent != expected_parent:
            raise AssertionError(f"Unexpected backend binding: {name}")
        if "umi" in parent.lower() or any(old in parent for old in ("nfs/vols", "iscsi/vols")):
            raise AssertionError(f"Legacy or environment-specific dataset collision: {parent}")



def verify_umi_network_policies(policies: list[dict]) -> None:
    """Fail closed if an unsupported kind, wildcard or required egress disappears."""
    expected = {
        "synelia-csi-controller-egress": {
            "app": "synelia-csi-controller",
            "nas_ports": {443},
        },
        "synelia-csi-node-egress": {
            "app": "synelia-csi-node",
            "nas_ports": {443, 2049, 3260},
        },
    }
    if len(policies) != 2 or {p["metadata"]["name"] for p in policies} != set(expected):
        raise AssertionError("Expected exactly two dedicated Synelia policies")
    for policy in policies:
        if policy["apiVersion"] != "networking.k8s.io/v1" or policy["kind"] != "NetworkPolicy":
            raise AssertionError("UMI Flannel cannot apply CiliumNetworkPolicy")
        if policy["metadata"].get("namespace") != "synelia-csi":
            raise AssertionError("NetworkPolicy escaped Synelia namespace")
        spec = policy["spec"]
        if spec.get("policyTypes") != ["Egress"] or "ingress" in spec:
            raise AssertionError("Do not accidentally change inbound CSI traffic")
        name = policy["metadata"]["name"]
        if spec.get("podSelector") != {"matchLabels": {"app": expected[name]["app"]}}:
            raise AssertionError("NetworkPolicy Pod scope must be CSI-specific")

        actual = {}
        for rule in spec["egress"]:
            if len(rule.get("to", [])) != 1:
                raise AssertionError("Each egress must target one narrow destination")
            target = rule["to"][0]
            ports = {(p["protocol"], p["port"]) for p in rule["ports"]}
            if "ipBlock" in target:
                if set(target) != {"ipBlock"}:
                    raise AssertionError("Unexpected IP target fields")
                cidr = target["ipBlock"]["cidr"]
                if cidr in actual:
                    raise AssertionError("Duplicate destination")
                actual[cidr] = ports
            else:
                if target != {
                    "namespaceSelector": {"matchLabels": {
                        "kubernetes.io/metadata.name": "kube-system"
                    }},
                    "podSelector": {"matchLabels": {"k8s-app": "kube-dns"}},
                }:
                    raise AssertionError("DNS destination must be the observed CoreDNS")
                actual["CoreDNS"] = ports
        want = {
            "CoreDNS": {("UDP", 53), ("TCP", 53)},
            "10.43.0.1/32": {("TCP", 443)},
            "172.30.208.64/32": {("TCP", 6443)},
            "192.168.111.69/32": {
                ("TCP", port) for port in expected[name]["nas_ports"]
            },
        }
        if actual != want:
            raise AssertionError(f"Egress policy drift for {name}: {actual}")


def verify_namespace_bootstrap(namespace: dict) -> None:
    """Keep Synelia namespace scoped, ordered before CSI and safe from prune."""
    if namespace.get("apiVersion") != "v1" or namespace.get("kind") != "Namespace":
        raise AssertionError("Expected one core-v1 Namespace")
    metadata = namespace["metadata"]
    if metadata.get("name") != "synelia-csi" or "namespace" in metadata:
        raise AssertionError("Namespace identity must be Synelia only")
    if metadata.get("labels", {}).get("app.kubernetes.io/part-of") != "synelia-csi":
        raise AssertionError("Namespace must remain scoped to Synelia")
    if metadata.get("annotations") != {
        "argocd.argoproj.io/sync-wave": "-1",
        "argocd.argoproj.io/sync-options": "Prune=false",
    }:
        raise AssertionError("Namespace must precede CSI and reject prune")


class SyneliaCSISourceTests(unittest.TestCase):
    def setUp(self):
        self.values = read_one(f"{BASE}/values.yaml")
        self.app = read_one(APP)

    def test_manual_umi_only_pinned_release_and_identity(self):
        self.assertEqual(self.app["metadata"]["name"], "synelia-csi-umi")
        self.assertEqual(self.app["spec"]["destination"]["namespace"], "synelia-csi")
        self.assertNotIn("automated", self.app["spec"]["syncPolicy"])
        sources = self.app["spec"]["sources"]
        self.assertEqual(sources[0]["chart"], "truenas-csi")
        self.assertEqual(sources[0]["targetRevision"], "1.4.0")
        self.assertEqual(sources[0]["helm"]["releaseName"], "synelia-csi")
        self.assertEqual(
            sources[0]["helm"]["valueFiles"], [f"$values/{BASE}/values.yaml"]
        )
        self.assertEqual(sources[1]["ref"], "values")
        self.assertEqual(sources[2]["path"], OVERLAY)
        self.assertEqual(
            read_one(f"{OVERLAY}/kustomization.yaml")["resources"],
            ["../../umi"],
        )
        self.assertEqual(self.values["driverName"], "synelia.csi.truenas.io")
        self.assertEqual(self.values["image"]["driver"]["tag"], "v1.4.0")
        self.assertEqual(self.values["kubeletDir"], "/var/lib/kubelet")
        self.assertEqual(self.values["volumeSnapshotClass"]["enabled"], False)
        self.assertNotIn(
            "apps/synelia-csi-umi.yaml",
            read_one("argocd/overlays/prod/kustomization.yaml")["resources"],
        )
        self.assertIn(
            "apps/synelia-csi-umi.yaml",
            read_one("argocd/overlays/dev/kustomization.yaml")["resources"],
        )

    def test_app_of_apps_discovers_child_via_dev_overlay(self):
        """Root -> dev App-of-Apps -> Synelia child -> dev Kustomize layer.

        Tests Git desired-state wiring only, never live ArgoCD registration.
        """
        root = (ROOT / "argocd/base/root-app.yaml.tpl").read_text(
            encoding="utf-8"
        )
        self.assertIn("name: vixens-app-of-apps", root)
        self.assertIn("path: ${overlay_path}", root)
        dev = read_one("argocd/overlays/dev/kustomization.yaml")
        prod = read_one("argocd/overlays/prod/kustomization.yaml")
        child = "apps/synelia-csi-umi.yaml"
        self.assertEqual(dev["resources"].count(child), 1)
        self.assertNotIn(child, prod["resources"])
        self.assertEqual(self.app["spec"]["sources"][2]["path"], OVERLAY)

    def test_classes_isolated_not_default_and_reclaim_compatible(self):
        verify_classes(self.values)
        # The retired driver must never reappear as a provisioner or a class name.
        rendered = yaml.safe_dump(self.values).lower()
        self.assertNotIn("democratic-csi", rendered)
        self.assertNotIn("truenas-iscsi", rendered)
        self.assertNotIn("truenas-nfs", rendered)

    def test_safety_regressions_fail_closed(self):
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["name"] = "synelia-iscsi-umi-delete"
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["parameters"]["datasetPath"] = (
            "k8s/vixens/synelia-umi-test/nfs"
        )
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["isDefault"] = True
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["parameters"]["datasetPath"] = (
            "k8s/vixens/nfs/vols"
        )
        with self.assertRaises(AssertionError):
            verify_classes(mutated)
        mutated = deepcopy(self.values)
        mutated["storageClasses"][0]["reclaimPolicy"] = "Delete"
        with self.assertRaises(AssertionError):
            verify_classes(mutated)

    def test_rwo_retained_contract_fail_closed(self):
        classes = self.values["storageClasses"]
        names = [s["name"] for s in classes]
        index = names.index("storage-rwo-retain")
        sc = classes[index]
        self.assertEqual(sc["reclaimPolicy"], "Retain")
        self.assertIs(sc["isDefault"], False)
        self.assertTrue(sc["allowVolumeExpansion"])
        self.assertEqual(sc["parameters"]["datasetPath"], "k8s/vixens/iscsi")
        self.assertEqual(sc["parameters"]["protocol"], "iscsi")
        self.assertEqual(sc["parameters"]["fsType"], "ext4")
        self.assertEqual(sc["volumeBindingMode"], "WaitForFirstConsumer")
        for change, new_value in (
            ("datasetPath", "k8s/vixens/iscsi/vols"),
            ("reclaimPolicy", "Delete"),
            ("isDefault", True),
            ("protocol", "nfs"),
        ):
            altered = deepcopy(self.values)
            candidate = altered["storageClasses"][index]
            if change in candidate["parameters"]:
                candidate["parameters"][change] = new_value
            else:
                candidate[change] = new_value
            with self.subTest(change=change, value=new_value), self.assertRaises(AssertionError):
                verify_classes(altered)

    def test_tls_and_openbao_infra_secret_contract(self):
        cfg = self.values["truenas"]
        endpoint = urlparse(cfg["url"])
        self.assertEqual(endpoint.scheme, "wss")
        self.assertEqual(endpoint.hostname, "nas.truxonline.com")
        self.assertEqual(endpoint.port, 8443)
        self.assertEqual(endpoint.path, "/api/current")
        self.assertIs(cfg["insecureSkipTLS"], False)
        self.assertEqual(cfg["existingSecret"], "synelia-csi-credentials")
        self.assertEqual(cfg["existingSecretKey"], "api-key")
        self.assertNotIn("apiKey", cfg)
        self.assertEqual(cfg["iscsiPortal"], "192.168.111.69:3260")
        eso = read_one(f"{BASE}/externalsecret.yaml")
        self.assertEqual(eso["metadata"]["namespace"], "synelia-csi")
        self.assertEqual(
            eso["spec"]["secretStoreRef"],
            {"name": "openbao", "kind": "ClusterSecretStore"},
        )
        self.assertEqual(eso["spec"]["target"]["deletionPolicy"], "Retain")
        self.assertEqual(
            eso["spec"]["data"][0],
            {
                "secretKey": "api-key",
                "remoteRef": {
                    "key": "vixens/dev/apps/01-storage/synelia-csi/api",
                    "property": "api-key",
                },
            },
        )

    def test_umi_namespace_bootstrap_before_namespaced_resources(self):
        """Argo+SSA should have a first-wave explicit Namespace manifest."""
        namespace = read_one(f"{BASE}/namespace.yaml")
        verify_namespace_bootstrap(namespace)
        resources = read_one(f"{BASE}/kustomization.yaml")["resources"]
        self.assertEqual(resources[0], "namespace.yaml")
        self.assertEqual(resources.count("namespace.yaml"), 1)
        self.assertEqual(self.app["spec"]["destination"]["namespace"], "synelia-csi")
        self.assertNotIn("automated", self.app["spec"]["syncPolicy"])

    def test_umi_namespace_bootstrap_negative_regressions(self):
        namespace = read_one(f"{BASE}/namespace.yaml")
        for mutation in ("wrong-name", "wrong-kind", "wrong-wave", "allow-prune"):
            altered = deepcopy(namespace)
            if mutation == "wrong-name":
                altered["metadata"]["name"] = "hindsight"
            elif mutation == "wrong-kind":
                altered["kind"] = "ConfigMap"
            elif mutation == "wrong-wave":
                altered["metadata"]["annotations"]["argocd.argoproj.io/sync-wave"] = "0"
            elif mutation == "allow-prune":
                altered["metadata"]["annotations"]["argocd.argoproj.io/sync-options"] = "Prune=true"
            with self.subTest(mutation=mutation), self.assertRaises(AssertionError):
                verify_namespace_bootstrap(altered)

    def test_umi_k3s_network_policy_supported_and_scoped(self):
        kustomization = read_one(f"{BASE}/kustomization.yaml")
        self.assertEqual(
            kustomization["resources"],
            ["namespace.yaml", "externalsecret.yaml", "networkpolicy.yaml"]
        )
        policies = [
            policy for policy in yaml.safe_load_all(
                (ROOT / BASE / "networkpolicy.yaml").read_text(encoding="utf-8")
            ) if policy
        ]
        verify_umi_network_policies(policies)

    def test_umi_k3s_network_policy_negative_regressions(self):
        policies = list(yaml.safe_load_all(
            (ROOT / BASE / "networkpolicy.yaml").read_text(encoding="utf-8")
        ))
        for change in ("cilium-kind", "broad-nas", "remove-api", "wrong-pods"):
            mutated = deepcopy(policies)
            if change == "cilium-kind":
                mutated[0]["kind"] = "CiliumNetworkPolicy"
            elif change == "broad-nas":
                mutated[0]["spec"]["egress"][-1]["to"][0]["ipBlock"]["cidr"] = "0.0.0.0/0"
            elif change == "remove-api":
                mutated[0]["spec"]["egress"].pop(1)
            elif change == "wrong-pods":
                mutated[0]["spec"]["podSelector"] = {}
            with self.subTest(change=change), self.assertRaises(AssertionError):
                verify_umi_network_policies(mutated)


if __name__ == "__main__":
    unittest.main()
