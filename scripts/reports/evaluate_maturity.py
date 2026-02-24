#!/usr/bin/env python3
"""
Evaluate application maturity level by querying the cluster directly

Returns the current tier and shows what's missing for the next level
"""

import argparse
import subprocess
import sys
import json


def run_cmd(cmd):
    """Run a command and return output"""
    try:
        result = subprocess.run(
            cmd, shell=True, capture_output=True, text=True, check=False
        )
        return result.stdout.strip(), result.stderr.strip(), result.returncode
    except Exception as e:
        return "", str(e), 1


def get_deployment(ns, app):
    """Get deployment details"""
    cmd = f"kubectl get deploy -n {ns} {app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        return json.loads(stdout)
    return None


def get_statefulset(ns, app):
    """Get statefulset details"""
    cmd = f"kubectl get sts -n {ns} {app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        return json.loads(stdout)
    return None


def get_pod(ns, app):
    """Get pod details"""
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if data.get("items"):
            return data["items"][0]
    return None


def get_ingress(ns, app):
    """Check if ingress exists"""
    cmd = f"kubectl get ingress -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            if app.lower() in item.get("metadata", {}).get("name", "").lower():
                return item
    return None


def get_secret(ns, app):
    """Check if secrets are managed via Infisical
    
    Returns:
        True - secrets are managed via Infisical
        False - app uses secrets but NOT via Infisical
        None - app doesn't use any secrets (N/A)
    """
    # First check if app actually uses secrets
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    uses_secrets = False
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get("items", []):
            spec = pod.get("spec", {})
            
            # Check for env vars referencing secrets
            for container in spec.get("containers", []):
                for env in container.get("env", []):
                    if "secretKeyRef" in env:
                        uses_secrets = True
                        break
                for env_from in container.get("envFrom", []):
                    if "secretRef" in env_from:
                        uses_secrets = True
                        break
                # Check for secret volumes
                for vol in spec.get("volumes", []):
                    if "secret" in vol:
                        uses_secrets = True
                        break
            
            if uses_secrets:
                break
    
    # If app doesn't use secrets, return None (N/A)
    if not uses_secrets:
        return None
    
    # App uses secrets - check if they're via Infisical
    cmd = f"kubectl get secret -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            secret_type = item.get("type", "")
            # Ignore TLS secrets (cert-manager)
            if secret_type != "kubernetes.io/tls":
                # Check for Infisical annotation or prefix
                annotations = item.get("metadata", {}).get("annotations", {})
                name = item.get("metadata", {}).get("name", "")
                if "infisical" in str(annotations).lower() or "infisical" in name.lower():
                    return True
                # Also check if it's an external-secret
                if "external-secrets" in annotations or "ESO" in annotations.get("controller", ""):
                    return True
                return False
    
    # Has secrets but no secret resources found
    return False
    """Check if secrets are managed via Infisical"""
    cmd = f"kubectl get secret -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            secret_type = item.get("type", "")
            # Ignore TLS secrets (cert-manager) - these are certificates, not app secrets
            if secret_type != "kubernetes.io/tls":
                return True
    return False
    """Check if secrets are managed via Infisical"""
    cmd = f"kubectl get secret -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        return len(data.get("items", [])) > 0
    return False


def check_tls(ingress):
    """Check if TLS is configured"""
    if not ingress:
        return False
    tls = ingress.get("spec", {}).get("tls", [])
    return len(tls) > 0


def check_probe(probes, probe_type):
    """Check if a probe is configured"""
    if not probes:
        return False
    return probe_type in probes


def check_resources(resources, req_type):
    """Check if CPU/Memory requests or limits are set"""
    if not resources:
        return False
    return req_type in resources and resources[req_type]


def get_qos_class(pod):
    """Get QoS class from pod"""
    return pod.get("status", {}).get("qosClass", "BestEffort")


def get_priority_class(pod):
    """Get priority class from pod"""
    return pod.get("spec", {}).get("priorityClassName", None)


def check_goldilocks_annotation(pod):
    """Check if Goldilocks annotation is present"""
    annotations = pod.get("metadata", {}).get("annotations", {})
    return "goldilocks.fairwinds.com/enabled" in annotations


def check_vpa_annotation(pod):
    """Check if VPA annotations are present"""
    annotations = pod.get("metadata", {}).get("annotations", {})
    return (
        "vpa" in annotations.get("recommendations", "").lower()
        or "autoscaling.k8s.io/vpa" in annotations
    )


def check_psa_labels(ns):
    """Check PSA labels on namespace"""
    cmd = f"kubectl get ns {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        labels = data.get("metadata", {}).get("labels", {})
        return "pod-security.kubernetes.io/enforce" in labels
    return False


def check_network_policy(ns, app):
    """Check if NetworkPolicy exists"""
    cmd = f"kubectl get netpol -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            selector = item.get("spec", {}).get("podSelector", {})
            if selector.get("matchLabels", {}).get("app") == app:
                return True
    return False


def check_pdb(ns, app):
    """Check if PodDisruptionBudget exists"""
    cmd = f"kubectl get pdb -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            selector = item.get("spec", {}).get("selector", {})
            if selector.get("matchLabels", {}).get("app") == app:
                return True
    return False


def get_backup_config(ns, app):
    """Check if backup is configured (Velero or Litestream)
    
    Returns:
        True - backup is configured
        False - no backup configured (has volumes)
        None - N/A (no persistent volumes to backup)
    """
    # First check if app uses any PVCs (Persistent Volume Claims)
    # or if there are volumes that need backup
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    has_persistent_volumes = False
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            # Check volumes - only count PVCs and hostPath volumes as persistent
            volumes = pod.get('spec', {}).get('volumes', [])
            for vol in volumes:
                # Check for PVC volume
                if 'persistentVolumeClaim' in vol:
                    has_persistent_volumes = True
                    break
                # Check for hostPath volume
                if 'hostPath' in vol:
                    has_persistent_volumes = True
                    break
                # Check for NFS volume
                if 'nfs' in vol:
                    has_persistent_volumes = True
                    break
            if has_persistent_volumes:
                break
    
    # Also check if there are any PVCs directly attached to the app
    if not has_persistent_volumes:
        cmd = f"kubectl get pvc -n {ns} -o json 2>/dev/null"
        stdout, _, rc = run_cmd(cmd)
        if rc == 0 and stdout:
            data = json.loads(stdout)
            for item in data.get('items', []):
                # Check if PVC is used by this app
                labels = item.get('metadata', {}).get('labels', {})
                if labels.get('app') == app:
                    has_persistent_volumes = True
                    break
    
    # If no persistent volumes, backup is N/A
    if not has_persistent_volumes:
        return None
    
    # Has persistent volumes - check for backup configuration
    # Check for Velero backup annotations
    cmd = f"kubectl get deploy -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get('items', []):
            annotations = item.get('metadata', {}).get('annotations', {})
            if 'backup.velero.io' in annotations:
                return True

    # Check for Litestream sidecar
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            for container in pod.get('spec', {}).get('containers', []):
                if 'litestream' in container.get('image', '').lower():
                    return True
    return False
    """Check if backup is configured (Velero or Litestream)
    
    Returns:
        True - backup is configured
        False - no backup configured (has volumes)
        None - N/A (no persistent volumes to backup)
    """
    # First check if app uses any persistent volumes
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    has_volumes = False
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            # Check for volume mounts
            for container in pod.get('spec', {}).get('containers', []):
                if container.get('volumeMounts'):
                    has_volumes = True
                    break
            # Also check pod volumes
            if pod.get('spec', {}).get('volumes'):
                has_volumes = True
            if has_volumes:
                break
    
    # If no volumes, backup is N/A
    if not has_volumes:
        return None
    
    # Has volumes - check for backup configuration
    # Check for Velero backup annotations
    cmd = f"kubectl get deploy -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get('items', []):
            annotations = item.get('metadata', {}).get('annotations', {})
            if 'backup.velero.io' in annotations:
                return True

    # Check for Litestream sidecar
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            for container in pod.get('spec', {}).get('containers', []):
                if 'litestream' in container.get('image', '').lower():
                    return True
    return False
    """Check if backup is configured (Velero or Litestream)"""
    # Check for Velero backup annotations
    cmd = f"kubectl get deploy -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            annotations = item.get("metadata", {}).get("annotations", {})
            if "backup.velero.io" in annotations:
                return True

    # Check for Litestream sidecar
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get("items", []):
            for container in pod.get("spec", {}).get("containers", []):
                if "litestream" in container.get("image", "").lower():
                    return True
    return False


def find_app_in_cluster(app_name):
    """Find application in cluster and return its namespace (optimized)"""

    # Get all deployments and statefulsets at once
    cmd = "kubectl get deploy,sts -A -o json"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0:
        data = json.loads(stdout)
        for item in data.get("items", []):
            item_name = item.get("metadata", {}).get("name", "")
            ns = item.get("metadata", {}).get("namespace")

            if app_name.lower() in item_name.lower():
                return ns, item.get("kind", "Deployment")

    # Also check pods (for apps without deploy/sts)
    cmd = "kubectl get pods -A -o json"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0:
        data = json.loads(stdout)
        for item in data.get("items", []):
            labels = item.get("metadata", {}).get("labels", {})
            ns = item.get("metadata", {}).get("namespace")

            # Check common app labels
            app_label = labels.get("app", "") or labels.get("name", "")
            if app_name.lower() in app_label.lower():
                return ns, "Pod"

    return None, None


def evaluate_bronze(ns, app, deploy_kind):
    """Level 1: Bronze - App exists"""
    checks = {}

    if deploy_kind == "Deployment":
        resource = get_deployment(ns, app)
    elif deploy_kind == "StatefulSet":
        resource = get_statefulset(ns, app)
    else:
        resource = None

    pod = get_pod(ns, app)

    # Check CPU/Memory requests
    if pod:
        containers = pod.get("spec", {}).get("containers", [])
        for container in containers:
            resources = container.get("resources", {})
            checks["CPU Request"] = check_resources(
                resources, "requests"
            ) and "cpu" in resources.get("requests", {})
            checks["Memory Request"] = check_resources(
                resources, "requests"
            ) and "memory" in resources.get("requests", {})
            break  # Check first container
    else:
        checks["CPU Request"] = False
        checks["Memory Request"] = False

    return all(checks.values()), checks


def evaluate_silver(ns, app, deploy_kind):
    """Level 2: Silver - Production Ready"""
    checks = {}

    pod = get_pod(ns, app)
    ingress = get_ingress(ns, app)

    if pod:
        containers = pod.get("spec", {}).get("containers", [])
        for container in containers:
            resources = container.get("resources", {})
            checks["CPU Limit"] = check_resources(
                resources, "limits"
            ) and "cpu" in resources.get("limits", {})
            checks["Memory Limit"] = check_resources(
                resources, "limits"
            ) and "memory" in resources.get("limits", {})

            probes = container.get("readinessProbe") or container.get(
                "readinessGates", []
            )
            checks["Readiness Probe"] = bool(probes)
            break
    else:
        checks["CPU Limit"] = False
        checks["Memory Limit"] = False
        checks["Readiness Probe"] = False

    checks["TLS/HTTPS"] = check_tls(ingress)
    # None = app doesn't use secrets (N/A), True = uses Infisical, False = uses secrets but not Infisical
    secret_check = get_secret(ns, app)
    checks["Secrets via Infisical"] = secret_check in [True, None]

    return all(checks.values()), checks
    # None = app doesn't use secrets (N/A), True = uses Infisical, False = uses secrets but not Infisical
    secret_check = get_secret(ns, app)
    checks["Secrets via Infisical"] = secret_check in [True, None]
    checks["Secrets via Infisical"] = get_secret(ns, app)

    return all(checks.values()), checks


def evaluate_gold(ns, app, deploy_kind):
    """Level 3: Gold - Standard Quality"""
    checks = {}

    pod = get_pod(ns, app)

    if pod:
        containers = pod.get("spec", {}).get("containers", [])
        for container in containers:
            probes = container.get("livenessProbe") or container.get(
                "livenessGates", []
            )
            checks["Liveness Probe"] = bool(probes)
            break
    else:
        checks["Liveness Probe"] = False

    checks["Goldilocks Enabled"] = check_goldilocks_annotation(pod) if pod else False
    checks["VPA Annotations"] = check_vpa_annotation(pod) if pod else False

    return all(checks.values()), checks


def evaluate_platinum(ns, app, deploy_kind):
    """Level 4: Platinum - Reliability"""
    checks = {}

    pod = get_pod(ns, app)

    checks["PriorityClass"] = get_priority_class(pod) is not None if pod else False
    checks["QoS Class"] = (
        get_qos_class(pod) in ["Guaranteed", "Burstable"] if pod else False
    )
    checks["PodDisruptionBudget"] = check_pdb(ns, app)

    return all(checks.values()), checks


def evaluate_emerald(ns, app, deploy_kind):
    """Level 5: Emerald - Data Durability"""
    checks = {}

    checks["Backup configured"] = get_backup_config(ns, app)

    # Filter out N/A (None) values - they don't count as failures
    # Only check that non-None values are True
    meaningful_checks = {k: v for k, v in checks.items() if v is not None}
    
    # If all meaningful checks pass, or if there are no meaningful checks
    passed = all(meaningful_checks.values()) if meaningful_checks else True
    
    return passed, checks
    """Level 5: Emerald - Data Durability"""
    checks = {}

    checks["Backup configured"] = get_backup_config(ns, app)

    return all(checks.values()), checks


def evaluate_diamond(ns, app, deploy_kind):
    """Level 6: Diamond - Security & Integration"""
    checks = {}

    checks["PSA Labels"] = check_psa_labels(ns)
    checks["NetworkPolicies"] = check_network_policy(ns, app)

    return all(checks.values()), checks


def evaluate_orichalcum(ns, app, deploy_kind):
    """Level 7: Orichalcum - Perfection"""
    checks = {}

    pod = get_pod(ns, app)
    checks["Guaranteed QoS"] = get_qos_class(pod) == "Guaranteed" if pod else False

    return all(checks.values()), checks


def main():
    parser = argparse.ArgumentParser(
        description="Evaluate application maturity level from cluster"
    )
    parser.add_argument("application", nargs="?", help="Application name to evaluate")
    parser.add_argument(
        "--namespace", "-n", help="Namespace (auto-detect if not provided)"
    )
    parser.add_argument(
        "--list", "-l", action="store_true", help="List all applications in cluster"
    )
    args = parser.parse_args()

    if args.list:
        cmd = "kubectl get deploy,sts -A -o json"
        stdout, _, rc = run_cmd(cmd)
        if rc == 0:
            data = json.loads(stdout)
            print("Applications in cluster:")
            for item in data.get("items", []):
                name = item.get("metadata", {}).get("name", "")
                ns = item.get("metadata", {}).get("namespace", "")
                kind = item.get("kind", "")
                print(f"  {ns}/{name} ({kind})")
        sys.exit(0)

    if not args.application:
        parser.error("application name is required (or use --list)")

    app_name = args.application

    if args.list:
        cmd = "kubectl get deploy,sts -A -o json"
        stdout, _, rc = run_cmd(cmd)
        if rc == 0:
            data = json.loads(stdout)
            print("Applications in cluster:")
            for item in data.get("items", []):
                name = item.get("metadata", {}).get("name", "")
                ns = item.get("metadata", {}).get("namespace", "")
                kind = item.get("kind", "")
                print(f"  {ns}/{name} ({kind})")
        sys.exit(0)

    # Find app in cluster
    if args.namespace:
        ns = args.namespace
        deploy_kind = None
    else:
        print(f"Searching for '{app_name}' in cluster...")
        ns, deploy_kind = find_app_in_cluster(app_name)

    if not ns:
        print(f"Application '{app_name}' not found in cluster")
        sys.exit(1)

    # Get resource kind if not found
    if not deploy_kind:
        if get_deployment(ns, app_name):
            deploy_kind = "Deployment"
        elif get_statefulset(ns, app_name):
            deploy_kind = "StatefulSet"
        else:
            deploy_kind = "Deployment"  # Default

    tiers = [
        ("Bronze", evaluate_bronze),
        ("Silver", evaluate_silver),
        ("Gold", evaluate_gold),
        ("Platinum", evaluate_platinum),
        ("Emerald", evaluate_emerald),
        ("Diamond", evaluate_diamond),
        ("Orichalcum", evaluate_orichalcum),
    ]

    current_tier = None
    failed_tier = None
    missing_for_next = None

    for tier_name, eval_func in tiers:
        passed, checks = eval_func(ns, app_name, deploy_kind)
        if passed:
            current_tier = tier_name
        else:
            failed_tier = tier_name
            missing_for_next = [k for k, v in checks.items() if not v and v is not None]
            break

    if current_tier is None:
        print(f"Failed-{failed_tier}")
        print(f"\nNamespace: {ns}")
        print(f"\nMissing {failed_tier} prerequisites:")
        for m in missing_for_next:
            print(f"  - {m}")
        sys.exit(1)
        print("failed")
        print(f"\nNamespace: {ns}")
        print("\nMissing Bronze prerequisites:")
        for m in missing_for_next:
            print(f"  - {m}")
        sys.exit(1)

    print(f"{current_tier}")
    print(f"\nNamespace: {ns}")

    if current_tier != "Orichalcum" and missing_for_next:
        print(f"\nMissing {failed_tier} prerequisites:")
        for m in missing_for_next:
            print(f"  - {m}")


if __name__ == "__main__":
    main()
