#!/usr/bin/env python3
import subprocess
import json
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')

TIERS = ["Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond", "Orichalcum"]

def run_cmd(cmd):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip(), result.returncode

def get_resources(kind):
    stdout, rc = run_cmd(f"kubectl get {kind} -A -o json")
    if rc != 0: return []
    return json.loads(stdout).get("items", [])

def sync_maturity():
    # 1. Map all Deployments/StatefulSets by their 'app' label
    resource_map = {}
    for kind in ["deployment", "statefulset"]:
        items = get_resources(kind)
        for item in items:
            ns = item["metadata"]["namespace"]
            name = item["metadata"]["name"]
            app_label = item["metadata"].get("labels", {}).get("app")
            if app_label:
                resource_map[f"{ns}/{app_label}"] = {"kind": kind, "name": name, "ns": ns}

    # 2. Collect Kyverno PolicyReports
    stdout, rc = run_cmd("kubectl get policyreport -A -o json")
    if rc != 0:
        logging.error("Failed to fetch PolicyReports")
        return
    
    reports = json.loads(stdout).get("items", [])
    app_results = {}

    for report in reports:
        ns = report["metadata"]["namespace"]
        for res in report.get("results", []):
            category = res.get("category", "")
            if "Maturity" not in category: continue
            
            try:
                tier_name = category.split("(")[1].split(")")[0]
            except IndexError:
                continue
            
            # Find the app label from the resource entry in results
            resource_info = res.get("resources", [{}])[0]
            # Use 'app' label as the primary key for grouping results
            app_label = resource_info.get("labels", {}).get("app")
            
            if not app_label: continue
            
            app_id = f"{ns}/{app_label}"
            if app_id not in app_results:
                app_results[app_id] = {t: True for t in TIERS}

            # If a single rule fails for a tier, the whole tier is False for this app
            if res.get("result") != "pass":
                app_results[app_id][tier_name] = False

    # 3. Apply labels to resources
    for app_id, results in app_results.items():
        if app_id not in resource_map:
            continue
            
        target = resource_map[app_id]
        
        # Calculate highest reached tier sequentially
        current_maturity = "none"
        for tier in TIERS:
            if results[tier]:
                current_maturity = tier.lower()
            else:
                break
        
        logging.info(f"App {app_id} -> maturity: {current_maturity}")
        
        # Label the resource
        cmd = f"kubectl label {target['kind']} -n {target['ns']} {target['name']} vixens.io/maturity={current_maturity} --overwrite"
        _, rc = run_cmd(cmd)
        if rc != 0:
            logging.error(f"Failed to label {app_id}")

if __name__ == "__main__":
    sync_maturity()
