#!/usr/bin/env python3
"""Validate inline Helm values blocks in ArgoCD Application manifests.

ArgoCD Applications embed Helm values as a literal YAML string:
  spec.source.helm.values: |
  spec.sources[N].helm.values: |

These blocks are easy to corrupt with indentation errors — outer YAML is
valid, inner content is not. yamllint cannot catch this.

Usage:
  python3 validate-helm-values.py [root_dir]
  python3 validate-helm-values.py file1.yaml file2.yaml

Exit: 0 if all valid, 1 if any invalid block found.
"""

import sys
import glob
import yaml
from pathlib import Path


def iter_helm_values(doc):
    if not isinstance(doc, dict):
        return
    if doc.get("kind") != "Application":
        return
    if "argoproj.io" not in doc.get("apiVersion", ""):
        return
    spec = doc.get("spec") or {}
    values = ((spec.get("source") or {}).get("helm") or {}).get("values")
    if values is not None:
        yield "spec.source.helm.values", values
    for i, src in enumerate(spec.get("sources") or []):
        values = ((src or {}).get("helm") or {}).get("values")
        if values is not None:
            yield f"spec.sources[{i}].helm.values", values


def main():
    targets = sys.argv[1:] if len(sys.argv) > 1 else ["."]
    errors, checked, files_scanned = [], 0, 0

    paths = []
    for t in targets:
        p = Path(t)
        if p.is_file():
            paths.append(str(p))
        elif p.is_dir():
            paths.extend(sorted(glob.glob(f"{t}/**/*.yaml", recursive=True)))

    paths = [p for p in paths if "/app-templates/" not in p]

    for path in paths:
        try:
            content = Path(path).read_text()
        except OSError:
            continue
        if "kind: Application" not in content or "argoproj.io" not in content:
            continue
        files_scanned += 1
        try:
            docs = list(yaml.safe_load_all(content))
        except yaml.YAMLError:
            continue
        for doc in (d for d in docs if d):
            for location, values_str in iter_helm_values(doc):
                if not isinstance(values_str, str):
                    checked += 1
                    continue
                try:
                    parsed = yaml.safe_load(values_str)
                    if parsed is not None and not isinstance(parsed, dict):
                        errors.append(
                            f"  ❌ {path}\n"
                            f"     {location}: got {type(parsed).__name__}, expected dict"
                        )
                    else:
                        checked += 1
                except yaml.YAMLError as e:
                    errors.append(
                        f"  ❌ {path}\n     {location}: {str(e).splitlines()[0]}"
                    )

    print(f"📋 ArgoCD Helm values: {files_scanned} files, {checked} blocks validated")
    if errors:
        print(f"\n❌ {len(errors)} invalid block(s):")
        for e in errors:
            print(e)
        sys.exit(1)
    print("✅ All inline Helm values are valid YAML")


if __name__ == "__main__":
    main()
