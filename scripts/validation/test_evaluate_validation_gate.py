#!/usr/bin/env python3
"""Regression tests for the authoritative Validation Summary policy."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("evaluate_validation_gate.py")
SPEC = importlib.util.spec_from_file_location("evaluate_validation_gate", MODULE_PATH)
assert SPEC and SPEC.loader
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)


def successful_env() -> dict[str, str]:
    return {
        "EVENT_NAME": "pull_request",
        "K8S_APPLICABLE": "true",
        "OPERATOR_APPLICABLE": "true",
        "DETECT_RESULT": "success",
        "YAML_RESULT": "success",
        "ARGOCD_RESULT": "success",
        "KUSTOMIZE_RESULT": "success",
        "SECRET_RESULT": "success",
        "SECURITY_RESULT": "success",
        "BRANCH_RESULT": "success",
        "PROD_RESULT": "success",
        "OPERATOR_RESULT": "success",
    }


class GatePolicyTests(unittest.TestCase):
    def assert_gate(self, env: dict[str, str], expected: bool) -> None:
        _checks, passed = GATE.evaluate(env)
        self.assertEqual(expected, passed)

    def test_all_applicable_success(self) -> None:
        self.assert_gate(successful_env(), True)

    def test_docs_only_allows_intentional_path_skips(self) -> None:
        env = successful_env()
        env.update(
            {
                "K8S_APPLICABLE": "false",
                "OPERATOR_APPLICABLE": "false",
                "YAML_RESULT": "skipped",
                "ARGOCD_RESULT": "skipped",
                "KUSTOMIZE_RESULT": "skipped",
                "PROD_RESULT": "skipped",
                "OPERATOR_RESULT": "skipped",
            }
        )
        self.assert_gate(env, True)

    def test_push_allows_branch_flow_skip(self) -> None:
        env = successful_env()
        env.update({"EVENT_NAME": "push", "BRANCH_RESULT": "skipped"})
        self.assert_gate(env, True)

    def test_applicable_failure_blocks(self) -> None:
        env = successful_env()
        env["KUSTOMIZE_RESULT"] = "failure"
        self.assert_gate(env, False)

    def test_applicable_cancelled_blocks(self) -> None:
        env = successful_env()
        env["YAML_RESULT"] = "cancelled"
        self.assert_gate(env, False)

    def test_applicable_unexpected_skip_blocks(self) -> None:
        env = successful_env()
        env["OPERATOR_RESULT"] = "skipped"
        self.assert_gate(env, False)

    def test_non_applicable_unexpected_execution_blocks(self) -> None:
        env = successful_env()
        env.update({"OPERATOR_APPLICABLE": "false", "OPERATOR_RESULT": "success"})
        self.assert_gate(env, False)


if __name__ == "__main__":
    unittest.main()
