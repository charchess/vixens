#!/usr/bin/env python3
"""Evaluate the path-aware CI results that back the required Validation Summary check."""

from __future__ import annotations

import os
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping


@dataclass(frozen=True)
class Check:
    name: str
    applicable: bool
    result: str

    @property
    def expected(self) -> str:
        return "success" if self.applicable else "skipped"

    @property
    def passed(self) -> bool:
        return self.result == self.expected


def _is_true(value: str | None) -> bool:
    return (value or "").strip().lower() == "true"


def evaluate(env: Mapping[str, str]) -> tuple[list[Check], bool]:
    k8s = _is_true(env.get("K8S_APPLICABLE"))
    operator = _is_true(env.get("OPERATOR_APPLICABLE"))
    branch = env.get("EVENT_NAME") == "pull_request"

    checks = [
        Check("Detect changes", True, env.get("DETECT_RESULT", "")),
        Check("YAML syntax & style", k8s, env.get("YAML_RESULT", "")),
        Check("ArgoCD structure", k8s, env.get("ARGOCD_RESULT", "")),
        Check("Kustomize + Kubeconform", k8s, env.get("KUSTOMIZE_RESULT", "")),
        Check("Secret scan", True, env.get("SECRET_RESULT", "")),
        Check("Security best practices", True, env.get("SECURITY_RESULT", "")),
        Check("Branch flow", branch, env.get("BRANCH_RESULT", "")),
        Check("Production config", k8s, env.get("PROD_RESULT", "")),
        Check("TXO Fabric operator", operator, env.get("OPERATOR_RESULT", "")),
    ]
    return checks, all(check.passed for check in checks)


def render(checks: list[Check], passed: bool) -> str:
    lines = [
        "## Validation Report",
        "",
        "| Validation | Applicable | Result | Gate |",
        "|---|---:|---|---|",
    ]
    for check in checks:
        lines.append(
            f"| {check.name} | {'yes' if check.applicable else 'no'} | "
            f"{check.result or '<missing>'} | {'✅' if check.passed else '❌'} |"
        )
    lines.extend(
        [
            "",
            (
                "✅ All applicable validations completed successfully."
                if passed
                else "❌ One or more applicable validations did not complete successfully."
            ),
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> int:
    checks, passed = evaluate(os.environ)
    report = render(checks, passed)
    print(report, end="")

    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_path:
        Path(summary_path).write_text(report, encoding="utf-8")

    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
