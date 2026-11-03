"""Fail closed if offline source tests can be skipped as a green CI summary.

Only workflow/source wiring is inspected. No Kubernetes or network connection.
"""

from copy import deepcopy
from pathlib import Path
import subprocess
import sys
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / ".github/workflows/validate.yaml"
REQUIRED = {
    "test_synelia_csi_source.py",
    "test_fabric_native_storage_declarations.py",
    "test_fabric_native_ai_gateway_declarations.py",
    "test_fabric_rwx_acceptance.py",
    "test_mail_native_storage_contract.py",
    "test_hermes_umi_stateful_contract.py",
    "test_ci_offline_source_routing.py",
}


def assert_required_offline_gate(workflow: dict) -> None:
    jobs = workflow.get("jobs") or {}
    offline = jobs.get("offline-source-regressions")
    summary = jobs.get("validation-summary")
    if not isinstance(offline, dict) or not isinstance(summary, dict):
        raise AssertionError("offline regression job and summary are mandatory")
    if "if" in offline or "needs" in offline:
        raise AssertionError("offline source tests may not be conditionally skipped")
    steps = offline.get("steps") or []
    commands = "\n".join(str(step.get("run", "")) for step in steps)
    if "set -euo pipefail" not in commands:
        raise AssertionError("offline test execution must fail on any suite")
    if "unittest discover" not in commands:
        raise AssertionError("offline suites must actually run")
    for suite in REQUIRED:
        if commands.count(suite) != 1:
            raise AssertionError(f"source regression not run exactly once: {suite}")
    # The workflow must not keep a second execution in an unrelated
    # Kustomize job which can be skipped on Python-only changes.
    kustomize = jobs.get("kustomize-validate") or {}
    kube_commands = "\n".join(str(step.get("run", "")) for step in kustomize.get("steps", []))
    if any(suite in kube_commands for suite in REQUIRED):
        raise AssertionError("source tests incorrectly coupled to Kustomize")
    if "offline-source-regressions" not in summary.get("needs", []):
        raise AssertionError("validation summary does not wait for source regressions")
    summary_steps = summary.get("steps") or []
    summary_env = "\n".join(str(v) for step in summary_steps for v in (step.get("env") or {}).values())
    summary_script = "\n".join(str(step.get("run", "")) for step in summary_steps)
    if "needs.offline-source-regressions.result" not in summary_env:
        raise AssertionError("summary does not consume exact source job result")
    if 'require_success "Offline source contracts" "$OFFLINE_SOURCE_RESULT"' not in summary_script:
        raise AssertionError("source regression failure/skip is not a hard gate")
    if summary.get("if") != "always()":
        raise AssertionError("summary must run even when a dependent job fails")


class OfflineSourceRoutingTests(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text(encoding="utf-8"))

    def test_current_workflow_enforces_all_offline_guards(self):
        assert_required_offline_gate(self.workflow)

    def test_inactive_or_missing_source_job_fails_closed(self):
        for mutation in (
            lambda w: w["jobs"]["offline-source-regressions"].update({"if": "false"}),
            lambda w: w["jobs"]["offline-source-regressions"].update({"needs": ["kustomize-validate"]}),
            lambda w: w["jobs"].pop("offline-source-regressions"),
            lambda w: w["jobs"]["validation-summary"]["needs"].remove("offline-source-regressions"),
            lambda w: w["jobs"]["validation-summary"]["steps"][-1]["env"].pop("OFFLINE_SOURCE_RESULT"),
            lambda w: w["jobs"]["validation-summary"].update({"if": "success()"}),
        ):
            mutated = deepcopy(self.workflow)
            mutation(mutated)
            with self.subTest(mutation=str(mutation)), self.assertRaises(AssertionError):
                assert_required_offline_gate(mutated)

    def test_removing_any_suite_from_execution_fails_closed(self):
        for suite in sorted(REQUIRED):
            mutated = deepcopy(self.workflow)
            steps = mutated["jobs"]["offline-source-regressions"]["steps"]
            step = next(s for s in steps if "unittest discover" in s.get("run", ""))
            step["run"] = step["run"].replace(suite, "omitted_test.py")
            with self.subTest(suite=suite), self.assertRaises(AssertionError):
                assert_required_offline_gate(mutated)

    def test_failing_unittest_exits_nonzero(self):
        # Demonstrate the CI command's fail-closed unittest exit semantics
        # without introducing a deliberately failing repository test.
        result = subprocess.run(
            [sys.executable, "-c",
             "import unittest; "
             "suite=unittest.TestSuite([unittest.FunctionTestCase("
             "lambda: (_ for _ in ()).throw(AssertionError('injected failure')))]); "
             "raise SystemExit(not unittest.TextTestRunner().run(suite).wasSuccessful())"],
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("injected failure", result.stderr)


if __name__ == "__main__":
    unittest.main()
