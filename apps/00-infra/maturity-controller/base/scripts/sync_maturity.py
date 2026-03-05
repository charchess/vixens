#!/usr/bin/env python3
"""
Maturity Controller — in-cluster sync script.

Reads Kyverno PolicyReports, computes the maturity tier for each
Deployment/StatefulSet/DaemonSet, and applies two labels:
  - vixens.io/maturity        : current tier (e.g. "gold")
  - vixens.io/maturity-missing: next tier name (or "none" if at max)
"""

import subprocess
import json
import logging
from datetime import datetime

logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")

TIERS = ["Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond", "Orichalcum"]
WORKLOAD_KINDS = ["deployment", "statefulset", "daemonset"]


def run_cmd(cmd):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip(), result.returncode


def normalize_app_name(name, kind):
    """Strip generated suffixes to recover the base app name."""
    if not name:
        return None
    suffixes = ["-ingress", "-service", "-svc", "-headless", "-metrics", "-config"]
    app_name = name
    if kind == "Pod":
        parts = name.split("-")
        app_name = "-".join(parts[:-2]) if len(parts) > 2 else name
    elif kind == "ReplicaSet":
        parts = name.split("-")
        app_name = "-".join(parts[:-1]) if len(parts) > 1 else name
    else:
        for suffix in suffixes:
            if app_name.endswith(suffix):
                app_name = app_name[: -len(suffix)]
                break
    return app_name or name


def sync_maturity():
    stdout, rc = run_cmd("kubectl get policyreport -A -o json")
    if rc != 0:
        logging.error("Failed to fetch PolicyReports")
        return

    reports = json.loads(stdout).get("items", [])

    # Key: "ns/app_name"
    # Value: {kind: {ts: datetime, fails: set[tier_name]}}
    app_components: dict = {}

    for report in reports:
        scope = report.get("scope", {})
        if not scope:
            continue

        ns = scope.get("namespace")
        kind = scope.get("kind")
        name = scope.get("name")
        ts_str = report["metadata"].get("creationTimestamp", "1970-01-01T00:00:00Z")
        ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))

        app_name = normalize_app_name(name, kind)
        if not app_name:
            continue

        app_id = f"{ns}/{app_name}"
        if app_id not in app_components:
            app_components[app_id] = {}

        # Keep only the most recent report per resource kind
        if (
            kind not in app_components[app_id]
            or ts > app_components[app_id][kind]["ts"]
        ):
            fails: set = set()
            for res in report.get("results", []):
                if res.get("result") in ("fail", "error"):
                    category = res.get("category", "")
                    if "Maturity" in category:
                        try:
                            tier_name = category.split("(")[1].split(")")[0]
                            fails.add(tier_name)
                        except (IndexError, ValueError):
                            pass
            app_components[app_id][kind] = {"ts": ts, "fails": fails}

    # Aggregate: merge fails from all resource kinds, compute highest passing tier
    for app_id, components in app_components.items():
        ns, name = app_id.split("/", 1)

        all_fails: set = set()
        for kind_data in components.values():
            all_fails.update(kind_data["fails"])

        current_tier = "none"
        missing_tier = TIERS[0].lower()  # default: missing Bronze

        for i, tier in enumerate(TIERS):
            if tier in all_fails:
                missing_tier = tier.lower()
                break
            current_tier = tier.lower()
            if i + 1 < len(TIERS):
                missing_tier = TIERS[i + 1].lower()
            else:
                missing_tier = "none"  # at Orichalcum

        # Apply labels to the first matching workload kind
        labeled = False
        for kind in WORKLOAD_KINDS:
            label_cmd = (
                f"kubectl label {kind} -n {ns} {name} "
                f"vixens.io/maturity={current_tier} "
                f"vixens.io/maturity-missing={missing_tier} "
                f"--overwrite 2>/dev/null"
            )
            _, rc = run_cmd(label_cmd)
            if rc == 0:
                logging.info(
                    f"{ns}/{name} ({kind}) -> tier={current_tier}, missing={missing_tier}"
                    f"  [sources: {', '.join(components.keys())}]"
                )
                labeled = True
                break

        if not labeled:
            logging.warning(f"Could not label {app_id} — no matching workload found")


if __name__ == "__main__":
    sync_maturity()
