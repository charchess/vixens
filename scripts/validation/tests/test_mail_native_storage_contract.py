"""Mail GitOps steady-state regression, independent of one-shot SSH imports (#4407)."""

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
BASE = ROOT / "apps/60-services/mail/base"


def documents(filename: str) -> list[dict]:
    return [d for d in yaml.safe_load_all((BASE / filename).read_text(encoding="utf-8")) if d]


class MailSteadyStateStorageTests(unittest.TestCase):
    def test_runtime_render_does_not_include_csi_import_resources(self) -> None:
        kustomization = yaml.safe_load((BASE / "kustomization.yaml").read_text(encoding="utf-8"))
        resources = kustomization["resources"]
        self.assertNotIn("external-secret-migration.yaml", resources)
        self.assertNotIn("migration-egress-policy.yaml", resources)
        self.assertNotIn("migration-job-v4.yaml", resources)
        self.assertTrue(all("migration" not in name for name in resources))

    def test_mail_and_gmail_runtime_credentials_remain_declared(self) -> None:
        kustomization = yaml.safe_load((BASE / "kustomization.yaml").read_text(encoding="utf-8"))
        resources = kustomization["resources"]
        for required in (
            "external-secrets.yaml",
            "external-secrets-runtime.yaml",
            "gmail-import-cronjob.yaml",
            "gmail-import-egress-policy.yaml",
            "mailserver-statefulset.yaml",
            "mailserver-ldap-egress-policy.yaml",
        ):
            self.assertIn(required, resources)
        secrets = {
            doc["spec"]["target"]["name"]
            for name in ("external-secrets.yaml", "external-secrets-runtime.yaml")
            for doc in documents(name)
            if doc["kind"] == "ExternalSecret"
        }
        self.assertIn("mail-gmail-import", secrets)
        self.assertIn("mail-imapsync-runtime", secrets)
        self.assertNotIn("mail-migration-ssh", secrets)
        jobs = documents("gmail-import-cronjob.yaml")
        self.assertEqual(len(jobs), 1)
        self.assertEqual(jobs[0]["metadata"]["name"], "gmail-import")
        text = (BASE / "gmail-import-cronjob.yaml").read_text(encoding="utf-8")
        self.assertIn("secretName: mail-gmail-import", text)
        self.assertIn("secretName: mail-imapsync-runtime", text)


if __name__ == "__main__":
    unittest.main()
