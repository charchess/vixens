#!/usr/bin/env python3
"""Validate PodDisruptionBudget selector consistency against workload pod labels.

Reads kustomize-built YAML (stdin or file) and checks that every PDB's
spec.selector.matchLabels is a subset of at least one Deployment/StatefulSet
pod template label in the same built output.

This must run on kustomize-built output (not raw sources) so that:
- Namespace is resolved
- Kustomize patches are applied (e.g. shared PDB component adds minAvailable)
- All resources are in a single document stream

Usage:
  kustomize build apps/foo/overlays/prod | python3 validate-pdb-selectors.py
  python3 validate-pdb-selectors.py /tmp/built.yaml

Exit code: 0 if all valid, 1 if any mismatch found.
"""

import sys
import yaml

WORKLOAD_KINDS = {"Deployment", "StatefulSet", "DaemonSet"}
ERRORS = []
CHECKED = 0
SKIPPED = 0


def load_docs(source):
    content = source.read()
    return [d for d in yaml.safe_load_all(content) if d is not None]


def collect(docs):
    workloads = []
    pdbs = []

    for doc in docs:
        kind = doc.get("kind", "")
        meta = doc.get("metadata") or {}
        name = meta.get("name", "<unnamed>")
        namespace = meta.get("namespace", "")

        if kind in WORKLOAD_KINDS:
            pod_labels = (doc.get("spec") or {}).get("template", {}).get(
                "metadata", {}
            ).get("labels") or {}
            workloads.append(
                {
                    "kind": kind,
                    "name": name,
                    "namespace": namespace,
                    "pod_labels": pod_labels,
                }
            )

        elif kind == "PodDisruptionBudget":
            match_labels = (doc.get("spec") or {}).get("selector", {}).get(
                "matchLabels"
            ) or {}
            pdbs.append(
                {"name": name, "namespace": namespace, "match_labels": match_labels}
            )

    return workloads, pdbs


def validate(docs):
    global CHECKED, SKIPPED
    workloads, pdbs = collect(docs)

    for pdb in pdbs:
        match_labels = pdb["match_labels"]
        ns = pdb["namespace"]

        if not match_labels:
            SKIPPED += 1
            print(
                f"  ⚠️  PDB '{pdb['name']}' (ns={ns}): empty selector — matches all pods"
            )
            continue

        candidates = [w for w in workloads if w["namespace"] == ns]

        if not candidates:
            SKIPPED += 1
            print(
                f"  ⚠️  PDB '{pdb['name']}' (ns={ns}): no workloads in same namespace — may be Helm-managed"
            )
            continue

        match_set = set(match_labels.items())
        matched = any(
            match_set.issubset(set(w["pod_labels"].items())) for w in candidates
        )

        if matched:
            CHECKED += 1
        else:
            mismatches = []
            for w in candidates:
                missing = match_set - set(w["pod_labels"].items())
                mismatches.append(
                    f"    {w['kind']}/{w['name']}: missing {dict(missing)}"
                )
            ERRORS.append(
                f"  ❌ PDB '{pdb['name']}' (ns={ns})\n"
                f"     selector: {dict(match_labels)}\n"
                f"     No matching workload:\n" + "\n".join(mismatches)
            )


def main():
    if len(sys.argv) > 1:
        with open(sys.argv[1]) as f:
            docs = load_docs(f)
    else:
        docs = load_docs(sys.stdin)

    validate(docs)

    total = CHECKED + SKIPPED + len(ERRORS)
    print(
        f"📋 PDB selectors: {total} checked, {CHECKED} OK, {SKIPPED} skipped, {len(ERRORS)} errors"
    )

    if ERRORS:
        print(f"\n❌ {len(ERRORS)} PDB selector mismatch(es):")
        for e in ERRORS:
            print(e)
        sys.exit(1)

    print("✅ All PDB selectors validated")


if __name__ == "__main__":
    main()
