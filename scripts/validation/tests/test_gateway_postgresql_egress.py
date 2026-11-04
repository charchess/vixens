"""Regression for #4441: gateway and schema-upgrade Job must reach their declared DB only."""

from pathlib import Path
import unittest

import yaml


GATEWAY = Path(__file__).resolve().parents[3] / "apps/60-services/txo-fabric/gateway"


class GatewayPostgresqlEgressTests(unittest.TestCase):
    def test_policies_target_declared_database_cluster_without_widening(self):
        # Both Database and DatabaseRole point to the same declared CNPG cluster.
        docs = list(yaml.safe_load_all((GATEWAY / "database.yaml").read_text()))
        clusters = {doc["spec"]["cluster"]["name"] for doc in docs
                    if doc and doc.get("kind") in ("Database", "DatabaseRole")}
        self.assertEqual(clusters, {"postgresql-shared-recovery"})
        names = {doc["metadata"]["name"] for doc in docs if doc and doc.get("kind") in ("Database", "DatabaseRole")}
        self.assertEqual(names, {"txo-ai-gateway-role-recovery", "txo-ai-gateway-database-recovery"})
        for filename in ("networkpolicy.yaml", "schema-upgrade-networkpolicy.yaml"):
            with self.subTest(policy=filename):
                policy = yaml.safe_load((GATEWAY / filename).read_text())
                rules = policy["spec"]["egress"]
                database_rules = [
                    rule for rule in rules
                    if any(target.get("namespaceSelector", {}).get("matchLabels", {}).get(
                        "kubernetes.io/metadata.name") == "databases"
                        for target in rule.get("to", []))
                ]
                self.assertEqual(len(database_rules), 1)
                rule = database_rules[0]
                self.assertEqual(rule["ports"], [{"protocol": "TCP", "port": 5432}])
                self.assertEqual(rule["to"], [{
                    "namespaceSelector": {"matchLabels": {
                        "kubernetes.io/metadata.name": "databases"}},
                    "podSelector": {"matchLabels": {
                        "cnpg.io/cluster": "postgresql-shared-recovery"}},
                }])

    def test_app_and_schema_upgrade_job_use_same_canonical_service(self):
        for filename in ("deployment.yaml", "schema-upgrade-job.yaml"):
            with self.subTest(workload=filename):
                text = (GATEWAY / filename).read_text()
                self.assertIn("postgresql-shared-rw.databases.svc:5432/txo_ai_gateway", text)
                self.assertNotIn("postgresql-shared-recovery", text)


if __name__ == "__main__":
    unittest.main()
