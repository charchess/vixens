#!/usr/bin/env python3
"""Validate the production CSI Helm render as Kubernetes objects, not text.

This is an offline GitOps contract; it does not contact Kubernetes.
"""
from pathlib import Path
import sys

import yaml

EXPECTED_CLASSES = {
    "storage-rwx-retain", "storage-rwx-delete",
    "storage-rwo-retain", "storage-rwo-delete",
    "storage-rwo-xfs-retain", "storage-rwo-xfs-delete",
}
EXPECTED_DRIVER = "synelia.prod.csi.truenas.io"


def assert_production_render(documents: list[dict]) -> None:
    drivers = [obj for obj in documents if obj.get("kind") == "CSIDriver"]
    if len(drivers) != 1 or drivers[0].get("metadata", {}).get("name") != EXPECTED_DRIVER:
        raise AssertionError("production CSI render must have exactly one canonical CSIDriver")

    classes = [obj for obj in documents if obj.get("kind") == "StorageClass"]
    names = [obj.get("metadata", {}).get("name") for obj in classes]
    if len(classes) != len(EXPECTED_CLASSES) or set(names) != EXPECTED_CLASSES:
        raise AssertionError("production render has missing, duplicate or unexpected StorageClasses")

    for storage_class in classes:
        name = storage_class["metadata"]["name"]
        expected_reclaim = name.rsplit("-", 1)[-1].capitalize()
        spec = storage_class.get("parameters") or {}
        if storage_class.get("provisioner") != EXPECTED_DRIVER:
            raise AssertionError(f"{name} uses a foreign CSI provisioner")
        if (storage_class.get("metadata", {}).get("annotations") or {}).get(
            "storageclass.kubernetes.io/is-default-class"
        ) != "false":
            raise AssertionError(f"{name} may become the default StorageClass")
        if storage_class.get("reclaimPolicy") != expected_reclaim:
            raise AssertionError(f"{name} has an unexpected reclaim policy")
        if name.startswith("storage-rwo-"):
            fs = "xfs" if name.startswith("storage-rwo-xfs-") else "ext4"
            expected = ("iscsi", "k8s/vixens/iscsi", "WaitForFirstConsumer")
            if spec.get("fsType") != fs:
                raise AssertionError(f"{name} has an unexpected filesystem")
        else:
            expected = ("nfs", "k8s/vixens/nfs", "Immediate")
            if "nfsvers=4.1" not in storage_class.get("mountOptions", []):
                raise AssertionError(f"{name} must use the native NFSv4.1 mount contract")
        protocol, parent, binding = expected
        if (spec.get("protocol") != protocol
                or spec.get("datasetPath") != parent
                or storage_class.get("volumeBindingMode") != binding):
            raise AssertionError(f"{name} has incorrect access or storage binding")

    if any(obj.get("kind") == "Secret" for obj in documents):
        raise AssertionError("CSI Helm render must not include credential Secrets")


def main(path: Path) -> int:
    try:
        documents = [x for x in yaml.safe_load_all(path.read_text(encoding="utf-8")) if x]
        assert_production_render(documents)
    except (OSError, ValueError, yaml.YAMLError, AssertionError) as exc:
        print(f"FAIL: production CSI render contract: {exc}", file=sys.stderr)
        return 1
    print("PASS: production CSI driver, six exclusive native classes and no rendered Secret")
    return 0


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: check_synelia_prod_render.py <helm-render.yaml>", file=sys.stderr)
        sys.exit(2)
    sys.exit(main(Path(sys.argv[1])))
