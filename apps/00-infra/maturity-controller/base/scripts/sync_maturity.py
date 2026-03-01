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
    # 1. Map all Deployments/StatefulSets by name and app label
    resource_map = {}
    for kind in ["deployment", "statefulset"]:
        items = get_resources(kind)
        for item in items:
            ns = item["metadata"]["namespace"]
            name = item["metadata"]["name"]
            app_label = item["metadata"].get("labels", {}).get("app")
            
            # Map by full name AND app label
            resource_map[f"{ns}/{name}"] = {"kind": kind, "name": name, "ns": ns}
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
            
            # Find target resource
            resource_info = res.get("resources", [{}])[0]
            res_name = resource_info.get("name", "")
            app_label = resource_info.get("labels", {}).get("app")
            
            # Try to find a match in our resource map
            target_key = None
            if f"{ns}/{app_label}" in resource_map:
                target_key = f"{ns}/{app_label}"
            elif f"{ns}/{res_name}" in resource_map:
                target_key = f"{ns}/{res_name}"
            else:
                # If it's a pod, try to find the deployment by prefix
                for key in resource_map.keys():
                    if key.startswith(f"{ns}/") and res_name.startswith(key.split("/")[1]):
                        target_key = key
                        break
            
            if not target_key: continue
            
            if target_key not in app_results:
                app_results[target_key] = {t: True for t in TIERS}

            if res.get("result") != "pass":
                app_results[target_key][tier_name] = False

    # 3. Apply labels to resources
    for target_key, results in app_results.items():
        target = resource_map[target_key]
        
        current_maturity = "none"
        for tier in TIERS:
            if results[tier]:
                current_maturity = tier.lower()
            else:
                break
        
        logging.info(f"App {target_key} -> maturity: {current_maturity}")
        
        cmd = f"kubectl label {target['kind']} -n {target['ns']} {target['name']} vixens.io/maturity={current_maturity} --overwrite"
        _, rc = run_cmd(cmd)
        if rc != 0:
            logging.error(f"Failed to label {target_key}")

if __name__ == "__main__":
    sync_maturity()
