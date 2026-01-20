#!/usr/bin/env python3
import sys
import subprocess
import json
import time

def run_cmd(cmd):
    result = subprocess.run(cmd, capture_output=True, text=True, shell=True)
    return result

def validate_app(app_name, env):
    print(f"🔍 Validating {app_name} in {env}...")
    if app_name == "scripts" or app_name == "infra" or app_name == "storage":
        print(f"✅ Validation skipped for virtual application '{app_name}'")
        return True
    import os
    kubeconfig = os.getenv("KUBECONFIG")
    if not kubeconfig:
        kubeconfig = f".secrets/{env}/kubeconfig-{env}"
    
    # 1. Pod Status
    extra_opts = ""
    if env == "dev":
        extra_opts = "--server=https://192.168.111.160:6443 --insecure-skip-tls-verify"
        
    # Try multiple label patterns
    labels = [f"app={app_name}", f"app={app_name}-{app_name}", f"app.kubernetes.io/name={app_name}"]
    data = {"items": []}
    for label in labels:
        pod_cmd = f"kubectl get pods -A -l {label} --kubeconfig {kubeconfig} {extra_opts} -o json"
        res = run_cmd(pod_cmd)
        if res.returncode == 0:
            current_data = json.loads(res.stdout)
            if current_data['items']:
                data = current_data
                break
    
    if not data['items']:
        print(f"❌ No pods found for app {app_name} (tried labels: {labels})")
        return False
    res = run_cmd(pod_cmd)
    if res.returncode != 0:
        print(f"❌ Failed to get pods: {res.stderr}")
        return False
    
    data = json.loads(res.stdout)
    if not data['items']:
        print(f"❌ No pods found for app {app_name}")
        return False
    
    pod = data['items'][0]
    status = pod['status']['phase']
    if status not in ["Running", "Succeeded"]:
        print(f"❌ Pod is not Running or Succeeded (Status: {status})")
        return False
    
    # 2. PriorityClass
    priority = pod['spec'].get('priorityClassName', "")
    if not priority:
        print("❌ Missing priorityClassName")
        return False
    print(f"✅ Pod is Running with priority {priority}")

    # 3. Network Access (if applicable)
    # Get host from ingress
    ing_cmd = f"kubectl get ingress -A -l app={app_name}-ingress --kubeconfig {kubeconfig} {extra_opts} -o json"
    # Fallback to name search
    res = run_cmd(ing_cmd)
    data = json.loads(res.stdout) if res.returncode == 0 else {"items": []}
    
    if not data['items']:
        ing_cmd = f"kubectl get ingress -A --kubeconfig {kubeconfig} {extra_opts} -o json | jq '.items[] | select(.metadata.name | contains(\"{app_name}\"))' | jq -s ."
        res = run_cmd(ing_cmd)
        data = {"items": json.loads(res.stdout)} if res.returncode == 0 else {"items": []}

    if data['items']:
        host = data['items'][0]['spec']['rules'][0]['host']
        print(f"🌐 Testing URL: https://{host}")
        
        # HTTP -> HTTPS
        http_res = run_cmd(f"curl -I -k http://{host}")
        if "301" in http_res.stdout or "302" in http_res.stdout or "308" in http_res.stdout:
            print("✅ HTTP -> HTTPS Redirection OK")
        else:
            print("⚠️ HTTP -> HTTPS Redirection missing or direct access")
            
        # HTTPS 200
        https_res = run_cmd(f"curl -L -k -s -o /dev/null -w '%{{http_code}}' https://{host}")
        if https_res.stdout.strip() == "200":
            print("✅ HTTPS Access 200 OK")
        else:
            print(f"❌ HTTPS Access failed (HTTP {https_res.stdout})")
            return False
    else:
        print("ℹ️ No ingress found for this app, skipping network tests")

    return True

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: validate.py <app_name> <env>")
        sys.exit(1)
    
    if validate_app(sys.argv[1], sys.argv[2]):
        print(f"🚀 {sys.argv[1]} is VALID")
        sys.exit(0)
    else:
        sys.exit(1)
