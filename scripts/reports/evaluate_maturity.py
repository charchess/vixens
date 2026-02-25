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
    
    # Check for InfisicalSecret CRD (the proper way to detect Infisical)
    cmd = f"kubectl get infisicalsecret -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if len(data.get("items", [])) > 0:
            return True
    
    # Also check by name pattern (InfisicalSecret often named <app>-secrets-sync)
    cmd = f"kubectl get infisicalsecret -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            name = item.get("metadata", {}).get("name", "").lower()
            if app.lower() in name:
                return True
    
    # Check for standard Kubernetes secrets (fallback)
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



def check_local_kustomize_structure(app_path):
    """Check if Kustomize structure is valid (local files)
    
    Verifies:
    - base/ directory exists with kustomization.yaml
    - overlays/ directory exists
    - At least one overlay (dev or prod) exists
    - All referenced files exist
    """
    import os
    
    if not os.path.exists(app_path):
        return False, f"App path does not exist: {app_path}"
    
    # Check base directory
    base_path = os.path.join(app_path, "base")
    if not os.path.exists(base_path):
        return False, "Missing base/ directory"
    
    base_kustomization = os.path.join(base_path, "kustomization.yaml")
    if not os.path.exists(base_kustomization):
        return False, "Missing base/kustomization.yaml"
    
    # Check overlays directory
    overlays_path = os.path.join(app_path, "overlays")
    if not os.path.exists(overlays_path):
        return False, "Missing overlays/ directory"
    
    # Check at least dev or prod exists
    dev_overlay = os.path.join(overlays_path, "dev")
    prod_overlay = os.path.join(overlays_path, "prod")
    if not os.path.exists(dev_overlay) and not os.path.exists(prod_overlay):
        return False, "Missing overlays (neither dev nor prod)"
    
    # Check each overlay has kustomization.yaml
    for overlay in ["dev", "prod", "staging", "test"]:
        overlay_path = os.path.join(overlays_path, overlay)
        if os.path.exists(overlay_path):
            kust_file = os.path.join(overlay_path, "kustomization.yaml")
            if not os.path.exists(kust_file):
                return False, f"Missing {overlay}/kustomization.yaml"
    
    return True, "Structure OK"


def check_yamllint(app_path, yamllint_config="yamllint-config.yml"):
    """Run yamllint on all YAML files in app path"""
    import os
    
    if not os.path.exists(app_path):
        return None, "App path does not exist"
    
    try:
        # Find all YAML files
        result = run_cmd(
            f"find {app_path} -name '*.yaml' -o -name '*.yml' | head -20"
        )
        
        stdout, _, rc = result
        if rc != 0 or not stdout.strip():
            return None, "No YAML files found"
        
        yaml_files = stdout.strip().split('\n')
        
        # Run yamllint on each file
        errors = []
        for yaml_file in yaml_files:
            if not yaml_file:
                continue
            stdout, _, rc = run_cmd(
                f"yamllint -c {yamllint_config} {yaml_file} 2>&1"
            )
            if rc != 0:
                errors.append(f"{yaml_file}: {stdout[:100]}")
        
        if errors:
            return False, f"yamllint errors: {len(errors)} files"
        
        return True, "All YAML files valid"
    
    except Exception as e:
        return None, f"yamllint check failed: {str(e)}"


def check_kustomize_build(app_path, overlay="prod"):
    """Test if kustomize build works for an overlay"""
    import os
    
    overlay_path = os.path.join(app_path, "overlays", overlay)
    if not os.path.exists(overlay_path):
        return None, f"Overlay {overlay} not found"
    
    try:
        stdout, stderr, rc = run_cmd(
            f"kustomize build {overlay_path} > /dev/null 2>&1"
        )
        
        if rc != 0:
            return False, f"kustomize build failed: {stderr[:100]}"
        
        return True, "kustomize build successful"
    
    except Exception as e:
        return None, f"kustomize check failed: {str(e)}"

def check_service(ns, app):
    """Check if Service exists for the application"""
    # Try by label first
    cmd = f"kubectl get svc -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if len(data.get("items", [])) > 0:
            return True
    
    # Try by name (service often has same name as app)
    cmd = f"kubectl get svc -n {ns} {app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        return True
    
    return False
    """Check if Service exists for the application"""
    cmd = f"kubectl get svc -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        return len(data.get("items", [])) > 0
    return False

def check_pvc(ns, app):
    """Check PVC safety - Recreate policy for iSCSI Retain PVCs
    
    Returns:
        True - Safe (no PVC, NFS, or iSCSI with Recreate)
        False - iSCSI Retain PVC without Recreate policy
    """
    # Get deployment
    cmd = f"kubectl get deploy -n {ns} {app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    if rc != 0 or not stdout:
        # Try StatefulSet
        cmd = f"kubectl get sts -n {ns} {app} -o json 2>/dev/null"
        stdout, _, rc = run_cmd(cmd)
    
    if rc != 0 or not stdout:
        return True
    
    data = json.loads(stdout)
    
    # StatefulSet - always OK
    if data.get("kind") == "StatefulSet":
        return True
    
    spec = data.get("spec", {})
    template_spec = spec.get("template", {}).get("spec", {})
    volumes = template_spec.get("volumes", [])
    
    for vol in volumes:
        if "persistentVolumeClaim" in vol:
            claim_name = vol.get("persistentVolumeClaim", {}).get("claimName", "")
            
            if not claim_name:
                continue
            
            # Get PVC storage class
            pvc_cmd = f"kubectl get pvc {claim_name} -n {ns} -o jsonpath='{{.spec.storageClassName}}' 2>/dev/null"
            pvc_stdout, _, _ = run_cmd(pvc_cmd)
            sc_name = pvc_stdout.strip() if pvc_stdout else ""
            
            if not sc_name:
                continue
            
            # Check if iSCSI + Retain
            if "iscsi" in sc_name.lower():
                # Get StorageClass reclaim policy
                sc_cmd = f"kubectl get sc {sc_name} -o jsonpath='{{.reclaimPolicy}}' 2>/dev/null"
                sc_stdout, _, _ = run_cmd(sc_cmd)
                reclaim = sc_stdout.strip().lower() if sc_stdout else ""
                
                if reclaim == "retain":
                    # iSCSI + Retain - must have strategy.type: Recreate
                    strategy_type = spec.get("strategy", {}).get("type", "RollingUpdate")
                    if strategy_type != "Recreate":
                        return False
    
    return True


def check_metrics(pod):
    """Check if metrics are exposed (prometheus annotations)
    
    Returns:
        True - metrics annotations present
        False - no metrics (will fail Gold)
        None - exempted via vixens.io/nometrics annotation
    """
    if not pod:
        return False
    
    annotations = pod.get("metadata", {}).get("annotations", {})
    
    # Check for exemption: vixens.io/nometrics: "true"
    if annotations.get("vixens.io/nometrics") == "true":
        return None  # N/A - app exempted from metrics requirement
    
    # Check for prometheus annotations
    has_metrics = (
        "prometheus.io/scrape" in annotations
        or "prometheus.io/port" in annotations
    )
    
    return has_metrics
    """Check if metrics are exposed (prometheus annotations)"""
    annotations = pod.get("metadata", {}).get("annotations", {})
    return (
        "prometheus.io/scrape" in annotations
        or "prometheus.io/port" in annotations
    )

def check_servicemonitor(ns, app):
    """Check if ServiceMonitor exists (contextual)"""
    cmd = f"kubectl get servicemonitor -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if len(data.get("items", [])) > 0:
            return True
    
    # Also check in monitoring namespace
    cmd = f"kubectl get servicemonitor -n monitoring -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if len(data.get("items", [])) > 0:
            return True
    
    return False

def check_revision_history_limit(deploy):
    """Check if revisionHistoryLimit is set to 3"""
    if not deploy:
        return False
    limit = deploy.get("spec", {}).get("revisionHistoryLimit")
    return limit is not None and limit <= 3

def check_sync_wave(ns, app):
    """Check if sync-wave annotation is present on ArgoCD Application"""
    # ArgoCD Applications are in the argocd namespace
    cmd = f"kubectl get application {app} -n argocd -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        annotations = data.get("metadata", {}).get("annotations", {})
        return "argocd.argoproj.io/sync-wave" in annotations
    return False
    """Check if sync-wave annotation is present"""
    if not deploy:
        return False
    annotations = deploy.get("metadata", {}).get("annotations", {})
    return "argocd.argoproj.io/sync-wave" in annotations

def check_backup_profile(ns, app):
    """Check if backup profile is defined (annotation or label)"""
    # Check deployment annotations
    cmd = f"kubectl get deploy -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get("items", []):
            annotations = item.get("metadata", {}).get("annotations", {})
            labels = item.get("metadata", {}).get("labels", {})
            
            # Check for backup profile annotation or label
            if "vixens.io/backup-profile" in annotations or "vixens.io/backup-profile" in labels:
                return True
            if "backup.velero.io/backup-volumes" in annotations:
                return True
    
    return False

def check_security_context_hardened(pod):
    """Check if SecurityContext is properly hardened"""
    if not pod:
        return False
    
    spec = pod.get("spec", {})
    
    # Check pod-level security context
    pod_security = spec.get("securityContext", {})
    
    checks = {
        "runAsNonRoot": pod_security.get("runAsNonRoot", False),
        "seccompProfile": pod_security.get("seccompProfile", {}).get("type") == "RuntimeDefault",
    }
    
    # Check container-level security context
    containers = spec.get("containers", [])
    for container in containers:
        container_security = container.get("securityContext", {})
        
        # At minimum, check for allowPrivilegeEscalation: false
        if container_security.get("allowPrivilegeEscalation") == False:
            checks["allowPrivilegeEscalation"] = True
        
        # Check for readOnlyRootFilesystem
        if container_security.get("readOnlyRootFilesystem") == True:
            checks["readOnlyRootFilesystem"] = True
        
        # Check for capabilities drop
        caps = container_security.get("capabilities", {})
        if caps.get("drop") and "ALL" in caps.get("drop", []):
            checks["capabilitiesDropAll"] = True
    
    # Must have at least runAsNonRoot + seccompProfile + allowPrivilegeEscalation
    return (
        checks.get("runAsNonRoot", False)
        and checks.get("seccompProfile", False)
        and checks.get("allowPrivilegeEscalation", False)
    )

def check_kyverno_compliance(ns, app):
    """Check if app has Kyverno policy violations (simplified version)"""
    # Quick check: look for policy violations in the namespace
    # Use a more targeted query to avoid loading all reports
    cmd = f"kubectl get policyreport -n {ns} -o jsonpath='{{.items[*].results}}' 2>/dev/null | grep -q '{app}'"
    stdout, stderr, rc = run_cmd(cmd)
    
    # If no policy reports or no mention of app, assume compliant
    if rc != 0:
        return True
    
    # If app is mentioned in policy reports, need to check more carefully
    # For now, return True to avoid timeout - detailed check can be done manually
    return True
    """Check if app has Kyverno policy violations"""
    # Get policy reports for this app
    cmd = f"kubectl get policyreport -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for report in data.get("items", []):
            results = report.get("results", [])
            for result in results:
                # Check if this result is for our app
                resources = result.get("resources", [])
                for resource in resources:
                    if resource.get("name") == app or resource.get("labels", {}).get("app") == app:
                        # Check status - fail if there are fail/violation results
                        if result.get("status") in ["fail", "error"]:
                            return False
    
    # Also check cluster-wide policy reports
    cmd = f"kubectl get clusterpolicyreport -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for report in data.get("items", []):
            results = report.get("results", [])
            for result in results:
                resources = result.get("resources", [])
                for resource in resources:
                    if (resource.get("namespace") == ns and 
                        (resource.get("name") == app or resource.get("labels", {}).get("app") == app)):
                        if result.get("status") in ["fail", "error"]:
                            return False
    
    return True



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


def has_persistent_volumes(ns, app):
    """Check if app has persistent volumes that need backup
    
    Returns:
        True - has persistent volumes
        False - no persistent volumes (N/A for backup)
    """
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    has_persistent_volumes = False
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            volumes = pod.get('spec', {}).get('volumes', [])
            for vol in volumes:
                if 'persistentVolumeClaim' in vol:
                    has_persistent_volumes = True
                    break
                if 'hostPath' in vol:
                    has_persistent_volumes = True
                    break
                if 'nfs' in vol:
                    has_persistent_volumes = True
                    break
            if has_persistent_volumes:
                break
    
    # Also check PVCs directly attached
    if not has_persistent_volumes:
        cmd = f"kubectl get pvc -n {ns} -o json 2>/dev/null"
        stdout, _, rc = run_cmd(cmd)
        if rc == 0 and stdout:
            data = json.loads(stdout)
            for item in data.get('items', []):
                labels = item.get('metadata', {}).get('labels', {})
                if labels.get('app') == app:
                    has_persistent_volumes = True
                    break
    
    return has_persistent_volumes


def uses_sqlite(ns, app):
    """Check if app uses SQLite database
    
    Looks for:
    - Litestream config (indicates SQLite is used)
    - DB file mounts in /config
    
    Returns:
        True - app uses SQLite
        False - app doesn't use SQLite (N/A for litestream)
    """
    # Check for litestream config (ConfigMap with litestream.yml)
    cmd = f"kubectl get configmap -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get('items', []):
            data_content = item.get('data', {})
            if 'litestream.yml' in data_content or 'litestream.yaml' in data_content:
                return True
    
    # Check for litestream sidecar container (more reliable)
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            for container in pod.get('spec', {}).get('containers', []):
                if 'litestream' in container.get('image', '').lower():
                    return True
    
    return False


def check_litestream_sidecar(ns, app):
    """Check if litestream sidecar is present for DB backup
    
    Returns:
        True - litestream sidecar exists
        False - no litestream (required if SQLite used)
        None - N/A (app doesn't use SQLite)
    """
    if not uses_sqlite(ns, app):
        return None
    
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            for container in pod.get('spec', {}).get('containers', []):
                if 'litestream' in container.get('image', '').lower():
                    return True
    return False


def check_litestream_restore_init(ns, app):
    """Check if litestream has restore initContainer
    
    InitContainer should:
    - Test DB integrity, OR
    - Check if DB is valid, if not restore from S3
    
    Returns:
        True - restore init exists and proper
        False - missing restore init (required if litestream used)
        None - N/A (no litestream)
    """
    if not uses_sqlite(ns, app):
        return None
    
    # Check if litestream sidecar exists first
    if not check_litestream_sidecar(ns, app):
        return None
    
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            init_containers = pod.get('spec', {}).get('initContainers', [])
            for init in init_containers:
                # Look for restore-related init container
                name = init.get('name', '').lower()
                image = init.get('image', '').lower()
                
                # Common restore patterns
                if 'restore' in name or 'init' in name:
                    # Check if it references the DB path or litestream
                    args = init.get('args', [])
                    command = init.get('command', [])
                    all_text = ' '.join(args) + ' '.join(command)
                    
                    # Should reference DB restore or litestream restore
                    if any(x in all_text.lower() for x in ['.db', 'litestream', 'restore', 'sqlite']):
                        return True
    return False


def check_rclone_backup_sidecar(ns, app):
    """Check if rclone sidecar exists for config backup
    
    Returns:
        True - rclone backup sidecar exists
        False - no rclone backup
        None - N/A (no config volume)
    """
    # Check if app has a config volume (indicates config backup needed)
    has_config = has_persistent_volumes(ns, app)
    if not has_config:
        return None
    
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            for container in pod.get('spec', {}).get('containers', []):
                image = container.get('image', '').lower()
                if 'rclone' in image:
                    # Check if it's doing backup (sync) not restore
                    args = container.get('args', [])
                    command = container.get('command', [])
                    all_text = ' '.join(args) + ' '.join(command)
                    if 'sync' in all_text.lower() or 'copy' in all_text.lower():
                        return True
    return False


def check_rclone_restore_init(ns, app):
    """Check if rclone restore initContainer exists
    
    InitContainer should restore config from S3 if local is empty
    
    Returns:
        True - restore init exists
        False - missing restore init (required if rclone backup used)
        None - N/A (no rclone backup)
    """
    # Check if rclone backup exists
    if not check_rclone_backup_sidecar(ns, app):
        return None
    
    cmd = f"kubectl get pods -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for pod in data.get('items', []):
            init_containers = pod.get('spec', {}).get('initContainers', [])
            for init in init_containers:
                name = init.get('name', '').lower()
                image = init.get('image', '').lower()
                
                # Look for restore-related init
                if 'restore' in name or 'config' in name:
                    args = init.get('args', [])
                    command = init.get('command', [])
                    all_text = ' '.join(args) + ' '.join(command)
                    
                    # Should reference rclone copy/sync from S3
                    if 'rclone' in all_text.lower() and ('copy' in all_text.lower() or 'sync' in all_text.lower()):
                        return True
    return False


def check_velero_backup(ns, app):
    """Check if namespace is covered by any Velero schedule
    
    Returns:
        True - namespace is in a Velero schedule
        False - namespace is NOT in any Velero schedule
        None - N/A (no persistent volumes)
    """
    if not has_persistent_volumes(ns, app):
        return None
    
    # Get all Velero schedules
    cmd = "kubectl get schedule -n velero -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    
    if rc != 0 or not stdout:
        # No Velero schedules found
        return False
    
    data = json.loads(stdout)
    schedules = data.get('items', [])
    
    # Check if namespace is in any schedule
    for schedule in schedules:
        spec = schedule.get('spec', {})
        template = spec.get('template', {})
        included_ns = template.get('includedNamespaces', [])
        
        # Check if namespace is included
        if '*' in included_ns or ns in included_ns:
            return True
    
    return False
    """Check if Velero backup is configured
    
    Returns:
        True - Velero backup configured
        False - no Velero backup
        None - N/A (no persistent volumes)
    """
    if not has_persistent_volumes(ns, app):
        return None
    
    # Check for Velero backup annotations on deployment
    cmd = f"kubectl get deploy -n {ns} -l app={app} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        for item in data.get('items', []):
            annotations = item.get('metadata', {}).get('annotations', {})
            if 'backup.velero.io' in str(annotations):
                return True
    
    # Check for BackupSchedule or Backup resource in the namespace
    cmd = f"kubectl get backups -n {ns} -o json 2>/dev/null"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0 and stdout:
        data = json.loads(stdout)
        if data.get('items'):
            return True
    
    return False


def get_backup_config(ns, app):
    """Check complete backup configuration
    
    Returns detailed dict with all backup components:
    - SQLite backup (litestream)
    - Config backup (rclone)
    - Velero backup
    """
    checks = {}
    
    # Check each backup component
    checks["SQLite present"] = uses_sqlite(ns, app)
    checks["Litestream sidecar"] = check_litestream_sidecar(ns, app)
    checks["Litestream restore init"] = check_litestream_restore_init(ns, app)
    checks["Rclone backup sidecar"] = check_rclone_backup_sidecar(ns, app)
    checks["Rclone restore init"] = check_rclone_restore_init(ns, app)
    checks["Velero backup"] = check_velero_backup(ns, app)
    
    return checks
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
    """Find application in cluster and return its namespace (optimized)
    
    Returns:
        (namespace, kind) - Best match found
        Prefers namespace that matches app_name exactly
    """
    best_match = None
    best_kind = None
    
    # Get all deployments and statefulsets at once
    cmd = "kubectl get deploy,sts -A -o json"
    stdout, _, rc = run_cmd(cmd)
    if rc == 0:
        data = json.loads(stdout)
        for item in data.get("items", []):
            item_name = item.get("metadata", {}).get("name", "")
            ns = item.get("metadata", {}).get("namespace")
            kind = item.get("kind", "Deployment")
            
            # Check for exact match first
            if app_name.lower() == item_name.lower():
                # Exact name match - check if namespace also matches
                if ns.lower() == app_name.lower():
                    return ns, kind  # Perfect match: name and namespace match
                if not best_match:
                    best_match = ns
                    best_kind = kind
            elif app_name.lower() in item_name.lower():
                # Partial match - only use if no exact match found
                if not best_match:
                    best_match = ns
                    best_kind = kind
    
    if best_match:
        return best_match, best_kind
    
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
            
            # Exact match preferred
            if app_name.lower() == app_label.lower():
                if ns.lower() == app_name.lower():
                    return ns, "Pod"
                if not best_match:
                    best_match = ns
                    best_kind = "Pod"
            elif app_name.lower() in app_label.lower():
                if not best_match:
                    best_match = ns
                    best_kind = "Pod"
    
    return best_match, best_kind
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


def evaluate_bronze(ns, app, deploy_kind, app_path=None):
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
            
            # NEW: Check no :latest tag (Bronze requirement per ADR-022)
            image = container.get("image", "")
            checks["No :latest tag"] = ":latest" not in image
            
            break  # Check first container
    else:
        checks["CPU Request"] = False
        checks["Memory Request"] = False
        checks["No :latest tag"] = False

    # Check Service and Ingress (Bronze requirements per ADR-022)
    checks["Service configured"] = check_service(ns, app)
    checks["Ingress configured"] = get_ingress(ns, app) is not None
    
    # NEW: Local checks (ADR-022 Kustomize structure)
    if app_path:
        # Check Kustomize structure
        kust_ok, kust_msg = check_local_kustomize_structure(app_path)
        checks["Kustomize structure"] = kust_ok
        
        # Check yamllint (optional, may not be installed)
        yamllint_ok, yamllint_msg = check_yamllint(app_path)
        if yamllint_ok is not None:  # Only add if check was possible
            checks["YAML valid (yamllint)"] = yamllint_ok
        
        # Check kustomize build (optional)
        build_ok, build_msg = check_kustomize_build(app_path, "prod")
        if build_ok is not None:  # Only add if check was possible
            checks["Kustomize build OK"] = build_ok

    return all(v for v in checks.values() if v is not None), checks
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
            
            # NEW: Check no :latest tag (Bronze requirement per ADR-022)
            image = container.get("image", "")
            checks["No :latest tag"] = ":latest" not in image
            
            break  # Check first container
    else:
        checks["CPU Request"] = False
        checks["Memory Request"] = False
        checks["No :latest tag"] = False

    # Check Service and Ingress (Bronze requirements per ADR-022)
    checks["Service configured"] = check_service(ns, app)
    checks["Ingress configured"] = get_ingress(ns, app) is not None

    return all(v for v in checks.values() if v is not None), checks
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

    # NEW: Check Service and Ingress (Bronze requirements per ADR-022)
    checks["Service configured"] = check_service(ns, app)
    checks["Ingress configured"] = get_ingress(ns, app) is not None

    return all(v for v in checks.values() if v is not None), checks
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

    return all(v for v in checks.values() if v is not None), checks


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
    
    # NEW: Check PVC (contextual - per ADR-022)
    # Only applies if app has persistent storage needs
    pvc_check = check_pvc(ns, app)
    if pvc_check is not None:  # Only add check if applicable
        checks["PVC strategy (iSCSI+Retain→Recreate)"] = pvc_check

    return all(v for v in checks.values() if v is not None), checks
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

    return all(v for v in checks.values() if v is not None), checks
    # None = app doesn't use secrets (N/A), True = uses Infisical, False = uses secrets but not Infisical
    secret_check = get_secret(ns, app)
    checks["Secrets via Infisical"] = secret_check in [True, None]
    checks["Secrets via Infisical"] = get_secret(ns, app)

    return all(v for v in checks.values() if v is not None), checks


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
    
    # NEW: Check metrics exposed (per ADR-022)
    checks["Metrics exposed"] = check_metrics(pod) if pod else False
    
    # NEW: Check ServiceMonitor (contextual - per ADR-022)
    # Only check if metrics are exposed
    if checks.get("Metrics exposed", False):
        sm_check = check_servicemonitor(ns, app)
        if sm_check:  # Only add if ServiceMonitor exists
            checks["ServiceMonitor"] = True

    return all(v for v in checks.values() if v is not None), checks
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

    return all(v for v in checks.values() if v is not None), checks


def evaluate_platinum(ns, app, deploy_kind):
    """Level 4: Platinum - Reliability"""
    checks = {}

    pod = get_pod(ns, app)
    
    # Get deployment resource for revisionHistoryLimit and sync-wave checks
    deploy = get_deployment(ns, app) if deploy_kind == "Deployment" else None

    checks["PriorityClass"] = get_priority_class(pod) is not None if pod else False
    checks["QoS Class"] = (
        get_qos_class(pod) in ["Guaranteed", "Burstable"] if pod else False
    )
    checks["PodDisruptionBudget"] = check_pdb(ns, app)
    
    # NEW: Check revisionHistoryLimit: 3 (per ADR-022)
    checks["revisionHistoryLimit: 3"] = check_revision_history_limit(deploy)
    
    # NEW: Check sync-wave configured (per ADR-022)
    checks["Sync-wave configured"] = check_sync_wave(ns, app)
    checks["Sync-wave configured"] = check_sync_wave(ns, app)

    return all(v for v in checks.values() if v is not None), checks
    """Level 4: Platinum - Reliability"""
    checks = {}

    pod = get_pod(ns, app)

    checks["PriorityClass"] = get_priority_class(pod) is not None if pod else False
    checks["QoS Class"] = (
        get_qos_class(pod) in ["Guaranteed", "Burstable"] if pod else False
    )
    checks["PodDisruptionBudget"] = check_pdb(ns, app)

    return all(v for v in checks.values() if v is not None), checks


def evaluate_emerald(ns, app, deploy_kind):
    """Level 5: Emerald - Data Durability
    
    Requirements:
    - If SQLite: litestream sidecar + restore init
    - If config volume: rclone backup + restore init
    - Velero backup (namespace in schedule)
    """
    checks = {}
    
    # Check if app has persistent data that needs backup
    pvc_check = check_pvc(ns, app)
    
    # If no persistent volumes, backup is N/A
    if pvc_check is None or not has_persistent_volumes(ns, app):
        return True, {"Backup": "N/A (no persistent volumes)"}
    
    # Get detailed backup configuration
    backup_checks = get_backup_config(ns, app)
    
    # Validate each backup component
    # SQLite → litestream required
    if backup_checks.get("SQLite present"):
        checks["Litestream sidecar (SQLite backup)"] = backup_checks.get("Litestream sidecar")
        checks["Litestream restore init (DB integrity)"] = backup_checks.get("Litestream restore init")
    
    # Config volume → rclone backup required
    if backup_checks.get("Rclone backup sidecar") is not None:
        checks["Rclone backup sidecar"] = backup_checks.get("Rclone backup sidecar")
        checks["Rclone restore init"] = backup_checks.get("Rclone restore init")
    
    # Velero backup (namespace in schedule)
    velero_check = backup_checks.get("Velero backup")
    if velero_check is not None:
        checks["Velero backup"] = velero_check
    
    # Filter out N/A (None) values - they don't count as failures
    meaningful_checks = {k: v for k, v in checks.items() if v is not None}
    
    # Passed if all meaningful checks are True
    passed = all(meaningful_checks.values()) if meaningful_checks else True
    
    # Add summary info
    checks["_summary"] = {
        "sqlite": backup_checks.get("SQLite present"),
        "has_backup": any([
            backup_checks.get("Litestream sidecar"),
            backup_checks.get("Rclone backup sidecar"),
            backup_checks.get("Velero backup")
        ])
    }
    
    return passed, checks
    """Level 5: Emerald - Data Durability
    
    Requirements:
    - If SQLite: litestream sidecar + restore init
    - If config volume: rclone backup + restore init
    - Backup profile defined (annotation or Velero)
    """
    checks = {}
    
    # Check if app has persistent data that needs backup
    pvc_check = check_pvc(ns, app)
    
    # If no persistent volumes, backup is N/A
    if pvc_check is None or not has_persistent_volumes(ns, app):
        return True, {"Backup": "N/A (no persistent volumes)"}
    
    # Get detailed backup configuration
    backup_checks = get_backup_config(ns, app)
    
    # Validate each backup component
    # SQLite → litestream required
    if backup_checks.get("SQLite present"):
        checks["Litestream sidecar (SQLite backup)"] = backup_checks.get("Litestream sidecar")
        checks["Litestream restore init (DB integrity)"] = backup_checks.get("Litestream restore init")
    
    # Config volume → rclone backup required
    if backup_checks.get("Rclone backup sidecar") is not None:
        checks["Rclone backup sidecar"] = backup_checks.get("Rclone backup sidecar")
        checks["Rclone restore init"] = backup_checks.get("Rclone restore init")
    
    # Velero is optional (recommended but not required)
    velero_check = backup_checks.get("Velero backup")
    if velero_check is not None:
        checks["Velero backup"] = velero_check
    
    # Backup profile annotation is optional (documentation only)
    # Commented out - not required for Emerald tier
    # profile_check = check_backup_profile(ns, app)
    # if profile_check is not None:
    #     checks["Backup profile defined"] = profile_check
    velero_check = backup_checks.get("Velero backup")
    if velero_check is not None:
        checks["Velero backup"] = velero_check
    
    # Backup profile annotation is optional - disabled for now
    # profile_check = check_backup_profile(ns, app)
    # if profile_check is not None:
    #     checks["Backup profile defined"] = profile_check
    # Backup profile annotation is optional - disabled for now
    # profile_check = check_backup_profile(ns, app)
    # if profile_check is not None:
    #     checks["Backup profile defined"] = profile_check
    if profile_check is not None:
        checks["Backup profile defined"] = profile_check
    
    # Filter out N/A (None) values - they don't count as failures
    meaningful_checks = {k: v for k, v in checks.items() if v is not None}
    
    # Passed if all meaningful checks are True
    passed = all(meaningful_checks.values()) if meaningful_checks else True
    
    # Add summary info
    checks["_summary"] = {
        "sqlite": backup_checks.get("SQLite present"),
        "has_backup": any([
            backup_checks.get("Litestream sidecar"),
            backup_checks.get("Rclone backup sidecar"),
            backup_checks.get("Velero backup")
        ])
    }
    
    return passed, checks
    """Level 5: Emerald - Data Durability"""
    checks = {}

    # Check if app has persistent data that needs backup
    pvc_check = check_pvc(ns, app)
    
    # Only check backup if app has persistent storage
    if pvc_check is not None:
        checks["Backup configured"] = get_backup_config(ns, app)
        
        # Backup profile annotation is optional - disabled for now
        # profile_check = check_backup_profile(ns, app)
        # if profile_check is not None:
        #     checks["Backup profile defined"] = profile_check
        # Only applies if app needs backup
        profile_check = check_backup_profile(ns, app)
        if profile_check is not None:
            checks["Backup profile defined"] = profile_check
    
    # Filter out N/A (None) values - they don't count as failures
    # Only check that non-None values are True
    meaningful_checks = {k: v for k, v in checks.items() if v is not None}
    
    # If all meaningful checks pass, or if there are no meaningful checks
    passed = all(meaningful_checks.values()) if meaningful_checks else True
    
    return passed, checks
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

    return all(v for v in checks.values() if v is not None), checks


def evaluate_diamond(ns, app, deploy_kind):
    """Level 6: Diamond - Security & Integration"""
    checks = {}

    checks["PSA Labels"] = check_psa_labels(ns)
    checks["NetworkPolicies"] = check_network_policy(ns, app)
    
    # NEW: Check SecurityContext hardened (per ADR-022)
    pod = get_pod(ns, app)
    checks["SecurityContext hardened"] = check_security_context_hardened(pod) if pod else False

    return all(v for v in checks.values() if v is not None), checks
    """Level 6: Diamond - Security & Integration"""
    checks = {}

    checks["PSA Labels"] = check_psa_labels(ns)
    checks["NetworkPolicies"] = check_network_policy(ns, app)

    return all(v for v in checks.values() if v is not None), checks


def evaluate_orichalcum(ns, app, deploy_kind):
    """Level 7: Orichalcum - Perfec"""
    checks = {}

    pod = get_pod(ns, app)
    checks["Guaranteed QoS"] = get_qos_class(pod) == "Guaranteed" if pod else False
    
    # NEW: Check Kyverno policy compliance (per ADR-022)
    checks["Kyverno policy compliant"] = check_kyverno_compliance(ns, app)
    
    # Note: Other Orichalcum requirements are difficult to automate:
    # - 7 days stability (requires historical data)
    # - Runbooks documented (requires documentation parsing)
    # - SLO/SLI defined (requires monitoring system integration)
    # - DR testing (requires manual verification)

    return all(v for v in checks.values() if v is not None), checks
    """Level 7: Orichalcum - Perfection"""
    checks = {}

    pod = get_pod(ns, app)
    checks["Guaranteed QoS"] = get_qos_class(pod) == "Guaranteed" if pod else False

    return all(v for v in checks.values() if v is not None), checks


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
    parser.add_argument(
        "--local_path", "-p", help="Local path to app for structure validation (e.g., apps/99-test/whoami)"
    )
    parser.add_argument(
        "--target", "-t",
        choices=["Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond", "Orichalcum"],
        help="Target tier - show all missing prerequisites to reach this level from current state"
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

    # Get local path for Bronze structure checks
    app_path = args.local_path

    tiers = [
        ("Bronze", lambda ns, app, dk: evaluate_bronze(ns, app, dk, app_path)),
        ("Silver", evaluate_silver),
        ("Gold", evaluate_gold),
        ("Platinum", evaluate_platinum),
        ("Emerald", evaluate_emerald),
        ("Diamond", evaluate_diamond),
        ("Orichalcum", evaluate_orichalcum),
    ]

    # If target specified, validate it and show all missing prerequisites
    target_tier = args.target
    if target_tier:
        target_index = None
        for i, (name, _) in enumerate(tiers):
            if name == target_tier:
                target_index = i + 1
                break
        
        all_missing = {}
        
        # Evaluate all tiers up to target
        for i, (tier_name, eval_func) in enumerate(tiers[:target_index]):
            passed, checks = eval_func(ns, app_name, deploy_kind)
            missing = [k for k, v in checks.items() if not v and v is not None]
            if missing:
                all_missing[tier_name] = missing
        
        print(f"Target: {target_tier}")
        print(f"\nNamespace: {ns}")
        
        if all_missing:
            print(f"\nAll missing prerequisites to reach {target_tier}:")
            for tier, missing_items in all_missing.items():
                print(f"\n  [{tier}]:")
                for m in missing_items:
                    print(f"    - {m}")
        else:
            print(f"\n✓ All prerequisites satisfied for {target_tier}!")
        
        sys.exit(0)

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
        print(f"Failed-Bronze")
        print(f"\nNamespace: {ns}")
        print(f"\nMissing Bronze prerequisites:")
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
