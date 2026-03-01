#!/usr/bin/env python3
import subprocess
import json
import logging
from datetime import datetime

logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')

TIERS = ["Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond", "Orichalcum"]

def run_cmd(cmd):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip(), result.returncode

def normalize_app_name(name, kind):
    """Normalize resource names to find the base app name."""
    if not name: return None
    
    # Common suffixes to remove
    suffixes = ["-ingress", "-service", "-svc", "-headless", "-metrics", "-config"]
    
    app_name = name
    if kind == "Pod":
        app_name = "-".join(name.split("-")[:-2])
    elif kind == "ReplicaSet":
        app_name = "-".join(name.split("-")[:-1])
    else:
        # For other kinds (Ingress, Service, etc.), strip known suffixes
        for suffix in suffixes:
            if app_name.endswith(suffix):
                app_name = app_name[:len(app_name)-len(suffix)]
                break
    
    return app_name

def sync_maturity():
    stdout, rc = run_cmd("kubectl get policyreport -A -o json")
    if rc != 0: return
    
    reports = json.loads(stdout).get("items", [])
    
    # Key: ns/app_name, Value: {resource_kind: {timestamp: ts, fails: set()}}
    app_components = {}

    for report in reports:
        scope = report.get("scope", {})
        if not scope: continue
        
        ns = scope.get("namespace")
        kind = scope.get("kind")
        name = scope.get("name")
        ts_str = report["metadata"]["creationTimestamp"]
        ts = datetime.fromisoformat(ts_str.replace('Z', '+00:00'))
        
        app_name = normalize_app_name(name, kind)
        if not app_name: continue
        
        app_id = f"{ns}/{app_name}"
        if app_id not in app_components:
            app_components[app_id] = {}
            
        # Store the latest results for THIS specific resource kind
        if kind not in app_components[app_id] or ts > app_components[app_id][kind]["ts"]:
            fails = set()
            for res in report.get("results", []):
                if res.get("result") in ["fail", "error"]:
                    category = res.get("category", "")
                    if "Maturity" in category:
                        try:
                            tier_name = category.split("(")[1].split(")")[0]
                            fails.add(tier_name)
                        except: pass
            
            app_components[app_id][kind] = {"ts": ts, "fails": fails}

    # Final aggregation: merge fails from all components of an app
    for app_id, components in app_components.items():
        ns, name = app_id.split("/")
        
        all_fails = set()
        for kind_data in components.values():
            all_fails.update(kind_data["fails"])
            
        highest_tier = "none"
        for tier in TIERS:
            if tier in all_fails:
                break
            highest_tier = tier.lower()
        
        # Label the primary deployment/statefulset
        for kind in ["deployment", "statefulset", "daemonset"]:
            _, rc = run_cmd(f"kubectl label {kind} -n {ns} {name} vixens.io/maturity={highest_tier} --overwrite 2>/dev/null")
            if rc == 0:
                logging.info(f"Finalized {ns}/{name} ({kind}) -> maturity={highest_tier} (Merged from: {', '.join(components.keys())})")
                break

if __name__ == "__main__":
    sync_maturity()
