#!/usr/bin/env python3
"""
evaluate_maturity.py — ADR-023 7-tier goldification evaluator.

Queries a live Kubernetes cluster via kubectl, evaluates a single app's
maturity tier, and reports the current tier plus missing checks for the
next tier.

Usage:
    python3 evaluate_maturity.py <app> [--namespace NS] [--app-path PATH] [--json]
"""

import argparse
import datetime
import json
import subprocess
import sys

# Auto-detect kubeconfig: use project prod kubeconfig if KUBECONFIG not set
import os as _os
_project_root = _os.path.dirname(_os.path.dirname(_os.path.dirname(_os.path.abspath(__file__))))
_prod_kubeconfig = _os.path.join(_project_root, ".secrets", "prod", "kubeconfig-prod")
if not _os.environ.get("KUBECONFIG") and _os.path.isfile(_prod_kubeconfig):
    _os.environ["KUBECONFIG"] = _prod_kubeconfig
TIER_NAMES = [
    "bronze", "silver", "gold", "platinum", "emerald", "diamond", "orichalcum"
]
TIER_EMOJI = {
    "bronze": "🥉", "silver": "🥈", "gold": "🥇",
    "platinum": "💎", "emerald": "🟢", "diamond": "💠", "orichalcum": "🌟",
}
TIER_LEVEL = {name: i + 1 for i, name in enumerate(TIER_NAMES)}


def run_cmd(cmd: str):
    try:
        r = subprocess.run(cmd, shell=True, capture_output=True, text=True, check=False)
        return r.stdout.strip(), r.stderr.strip(), r.returncode
    except Exception as exc:
        return "", str(exc), 1


def _kubectl_json(args: str):
    stdout, _, rc = run_cmd(f"kubectl {args} -o json 2>/dev/null")
    if rc == 0 and stdout:
        try:
            return json.loads(stdout)
        except json.JSONDecodeError:
            pass
    return None


def list_resources(kind: str, ns: str, label_selector: str = "") -> list:
    sel = f"-l {label_selector}" if label_selector else ""
    data = _kubectl_json(f"get {kind} -n {ns} {sel}")
    return data.get("items", []) if data else []


def get_workload(ns: str, app: str):
    for kind in ("deploy", "sts", "ds"):
        data = _kubectl_json(f"get {kind} -n {ns} {app}")
        if data and data.get("kind"):
            return data
    return None


def get_pods(ns: str, app: str) -> list:
    data = _kubectl_json(f"get pods -n {ns} -l app={app}")
    return data.get("items", []) if data else []


def get_ingress(ns: str, app: str):
    for item in list_resources("ingress", ns):
        if app.lower() in item.get("metadata", {}).get("name", "").lower():
            return item
    return None


def find_app(app_name: str):
    """Search all namespaces. Returns (namespace, workload_dict) or (None, None)."""
    for kind in ("deploy", "sts", "ds"):
        data = _kubectl_json(f"get {kind} -A")
        if not data:
            continue
        for item in data.get("items", []):
            if item.get("metadata", {}).get("name", "") == app_name:
                return item.get("metadata", {}).get("namespace"), item
    return None, None


def _ann(obj) -> dict:
    return obj.get("metadata", {}).get("annotations", {}) if obj else {}


def _containers(workload) -> list:
    if not workload:
        return []
    return (workload.get("spec", {})
                    .get("template", {})
                    .get("spec", {})
                    .get("containers", []))


def _pt_annotations(workload) -> dict:
    if not workload:
        return {}
    return (workload.get("spec", {})
                    .get("template", {})
                    .get("metadata", {})
                    .get("annotations", {}))


def _pt_labels(workload) -> dict:
    if not workload:
        return {}
    return (workload.get("spec", {})
                    .get("template", {})
                    .get("metadata", {})
                    .get("labels", {}))


def _pod_spec(workload) -> dict:
    if not workload:
        return {}
    return workload.get("spec", {}).get("template", {}).get("spec", {})


def _replicas(workload) -> int:
    return workload.get("spec", {}).get("replicas", 1) if workload else 1


def check_bronze(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    containers = _containers(workload)

    c["Image tag is not :latest"] = bool(containers) and all(
        not cnt.get("image", "").endswith(":latest") for cnt in containers
    )

    svcs = list_resources("svc", ns, f"app={app}")
    if not svcs:
        exact = _kubectl_json(f"get svc -n {ns} {app}")
        svcs = [exact] if exact and exact.get("kind") == "Service" else []
    c["Service exists"] = len(svcs) > 0

    if _ann(workload).get("vixens.io/noingressneeded") == "true":
        c["Ingress configured"] = None
    else:
        c["Ingress configured"] = get_ingress(ns, app) is not None

    c["Resource requests (CPU+memory) on all containers"] = bool(containers) and all(
        cnt.get("resources", {}).get("requests", {}).get("cpu")
        and cnt.get("resources", {}).get("requests", {}).get("memory")
        for cnt in containers
    )
    return c


def check_silver(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    containers = _containers(workload)
    wl_ann = _ann(workload)

    c["Resource limits (CPU+memory) on all containers"] = bool(containers) and all(
        cnt.get("resources", {}).get("limits", {}).get("cpu")
        and cnt.get("resources", {}).get("limits", {}).get("memory")
        for cnt in containers
    )

    c["Liveness probe on all containers"] = bool(containers) and all(
        cnt.get("livenessProbe") is not None for cnt in containers
    )

    c["Readiness probe on all containers"] = bool(containers) and all(
        cnt.get("readinessProbe") is not None for cnt in containers
    )

    if wl_ann.get("vixens.io/slow-start") == "true":
        c["Startup probe (vixens.io/slow-start set)"] = bool(containers) and all(
            cnt.get("startupProbe") is not None for cnt in containers
        )
    else:
        c["Startup probe (vixens.io/slow-start set)"] = None

    uses_secrets = _workload_uses_secrets(workload)
    if not uses_secrets:
        c["Infisical secret (app uses K8s secrets)"] = None
    else:
        c["Infisical secret (app uses K8s secrets)"] = _has_infisical_secret(ns, app)

    pvcs = _get_pvcs_for_workload(ns, workload)
    if not pvcs:
        c["Storage strategy (StorageClass suffix -retain/-delete)"] = None
    else:
        suffix = "-retain" if "prod" in ns else "-delete"
        c["Storage strategy (StorageClass suffix -retain/-delete)"] = all(
            pvc.get("spec", {}).get("storageClassName", "").endswith(suffix)
            for pvc in pvcs
        )
    return c


def check_gold(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    wl_ann = _ann(workload)
    pt_ann = _pt_annotations(workload)

    c["Goldilocks annotation (goldilocks.fairwinds.com/enabled: true)"] = (
        wl_ann.get("goldilocks.fairwinds.com/enabled") == "true"
    )

    c["VPA updateMode annotation (vpa.kubernetes.io/updateMode)"] = (
        "vpa.kubernetes.io/updateMode" in wl_ann
    )

    rhl = workload.get("spec", {}).get("revisionHistoryLimit") if workload else None
    c["revisionHistoryLimit <= 3"] = rhl is not None and rhl <= 3

    c["ArgoCD sync-wave annotation (argocd.argoproj.io/sync-wave)"] = (
        "argocd.argoproj.io/sync-wave" in wl_ann
    )

    if pt_ann.get("vixens.io/nometrics") == "true":
        c["Metrics exposed (prometheus annotations on pod template)"] = None
        c["ServiceMonitor exists"] = None
    else:
        c["Metrics exposed (prometheus annotations on pod template)"] = (
            "prometheus.io/scrape" in pt_ann or "prometheus.io/port" in pt_ann
        )
        sm = list_resources("servicemonitor", ns, f"app={app}")
        sm_mon = list_resources("servicemonitor", "monitoring", f"app={app}")
        c["ServiceMonitor exists"] = len(sm) > 0 or len(sm_mon) > 0
    return c


def check_platinum(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    wl_ann = _ann(workload)
    pt_labels = _pt_labels(workload)

    pc = pods[0].get("spec", {}).get("priorityClassName", "") if pods else ""
    c["Priority class (vixens.io/*)"] = pc.startswith("vixens")

    qos = pods[0].get("status", {}).get("qosClass", "BestEffort") if pods else "BestEffort"
    c["QoS is not BestEffort (Burstable or Guaranteed)"] = qos != "BestEffort"

    c["Sizing label (vixens.io/sizing.<container>) on pod template"] = any(
        k.startswith("vixens.io/sizing.") for k in pt_labels
    )

    pdbs = list_resources("pdb", ns)
    c["PodDisruptionBudget exists"] = any(
        p.get("spec", {}).get("selector", {}).get("matchLabels", {}).get("app") == app
        for p in pdbs
    )

    if _replicas(workload) <= 1:
        c["Topology spread or podAntiAffinity (multi-replica)"] = None
    else:
        pspec = _pod_spec(workload)
        c["Topology spread or podAntiAffinity (multi-replica)"] = (
            bool(pspec.get("topologySpreadConstraints"))
            or bool(pspec.get("affinity", {}).get("podAntiAffinity"))
        )

    if wl_ann.get("vixens.io/has-long-connections") == "true":
        containers = _containers(workload)
        c["preStop lifecycle hook (vixens.io/has-long-connections set)"] = bool(containers) and all(
            cnt.get("lifecycle", {}).get("preStop") is not None for cnt in containers
        )
    else:
        c["preStop lifecycle hook (vixens.io/has-long-connections set)"] = None

    if wl_ann.get("vixens.io/needs-autoscaling") == "true":
        hpas = list_resources("hpa", ns, f"app={app}")
        keda = list_resources("scaledobject.keda.sh", ns, f"app={app}")
        c["HPA or KEDA ScaledObject (vixens.io/needs-autoscaling set)"] = len(hpas) > 0 or len(keda) > 0
    else:
        c["HPA or KEDA ScaledObject (vixens.io/needs-autoscaling set)"] = None
    return c


def check_emerald(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    has_pvc = bool(_get_pvcs_for_workload(ns, workload))
    has_sqlite = _app_uses_sqlite(ns, app, pods)

    if has_sqlite:
        ls = _has_litestream_sidecar(pods)
        c["Litestream sidecar (SQLite backup)"] = ls
        c["Litestream restore initContainer"] = _has_litestream_restore_init(pods) if ls else None
    else:
        c["Litestream sidecar (SQLite backup)"] = None
        c["Litestream restore initContainer"] = None

    if has_pvc and not has_sqlite:
        rclone = _has_rclone_sidecar(pods)
        c["Rclone backup sidecar (config/data)"] = rclone
        c["Rclone restore initContainer"] = _has_rclone_restore_init(pods) if rclone else None
    else:
        c["Rclone backup sidecar (config/data)"] = None
        c["Rclone restore initContainer"] = None

    c["Velero schedule covers namespace"] = _velero_covers_namespace(ns) if has_pvc else None
    return c


def check_diamond(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    wl_ann = _ann(workload)
    containers = _containers(workload)

    ns_data = _kubectl_json(f"get ns {ns}")
    ns_labels = ns_data.get("metadata", {}).get("labels", {}) if ns_data else {}
    c["Pod Security Admission label on namespace"] = (
        "pod-security.kubernetes.io/enforce" in ns_labels
    )

    netpols = list_resources("netpol", ns)
    c["NetworkPolicy exists (podSelector: app=APP)"] = any(
        p.get("spec", {}).get("podSelector", {}).get("matchLabels", {}).get("app") == app
        for p in netpols
    )

    if containers:
        psc = _pod_spec(workload).get("securityContext", {})
        c["Security context hardened (runAsNonRoot + drop:ALL)"] = (
            psc.get("runAsNonRoot", False)
            and all(
                cnt.get("securityContext", {}).get("allowPrivilegeEscalation") is False
                and "ALL" in cnt.get("securityContext", {}).get("capabilities", {}).get("drop", [])
                for cnt in containers
            )
        )
    else:
        c["Security context hardened (runAsNonRoot + drop:ALL)"] = False

    if wl_ann.get("vixens.io/nossoneeded") == "true":
        c["Authentik SSO (nginx.ingress.kubernetes.io/auth-url on ingress)"] = None
    else:
        ingress = get_ingress(ns, app)
        c["Authentik SSO (nginx.ingress.kubernetes.io/auth-url on ingress)"] = (
            "nginx.ingress.kubernetes.io/auth-url" in _ann(ingress)
        )

    if wl_ann.get("vixens.io/cve-accepted") == "true":
        c["Trivy VulnerabilityReport (no critical CVEs)"] = None
    else:
        c["Trivy VulnerabilityReport (no critical CVEs)"] = _trivy_clean(ns, app, {"critical"})

    if wl_ann.get("vixens.io/nohomepage") == "true":
        c["Homepage widget (gethomepage.dev/enabled on service)"] = None
    else:
        svcs = list_resources("svc", ns, f"app={app}")
        c["Homepage widget (gethomepage.dev/enabled on service)"] = any(
            _ann(s).get("gethomepage.dev/enabled") == "true" for s in svcs
        )

    if wl_ann.get("vixens.io/digest-pinned") == "true":
        c["Image digest pinned (@sha256:)"] = bool(containers) and all(
            "@sha256:" in cnt.get("image", "") for cnt in containers
        )
    else:
        c["Image digest pinned (@sha256:)"] = None

    c["Velero backup schedule for namespace"] = _velero_covers_namespace(ns)
    return c


def check_orichalcum(ns: str, app: str, workload, pods: list) -> dict:
    c = {}
    wl_ann = _ann(workload)
    containers = _containers(workload)
    pt_labels = _pt_labels(workload)

    c["Runbook certified (vixens.io/runbook-certified: <date>)"] = (
        "vixens.io/runbook-certified" in wl_ann
    )
    c["DR tested (vixens.io/dr-tested: <date>)"] = "vixens.io/dr-tested" in wl_ann
    c["SLO defined (vixens.io/slo-defined: <date>)"] = "vixens.io/slo-defined" in wl_ann

    if pods:
        zero_restarts = all(
            cs.get("restartCount", 0) == 0
            for pod in pods
            for cs in pod.get("status", {}).get("containerStatuses", [])
        )
        now = datetime.datetime.now(datetime.timezone.utc)
        oldest_days = None
        for pod in pods:
            start_str = pod.get("status", {}).get("startTime")
            if start_str:
                try:
                    start = datetime.datetime.fromisoformat(start_str.replace("Z", "+00:00"))
                    age = (now - start).days
                    if oldest_days is None or age > oldest_days:
                        oldest_days = age
                except ValueError:
                    pass
        no_oom = all(
            cs.get("lastState", {}).get("terminated", {}).get("reason") != "OOMKilled"
            for pod in pods
            for cs in pod.get("status", {}).get("containerStatuses", [])
        )
        c["7-day stability (no restarts, no OOMKill, pod age >= 7d)"] = (
            zero_restarts and (oldest_days is not None and oldest_days >= 7) and no_oom
        )
    else:
        c["7-day stability (no restarts, no OOMKill, pod age >= 7d)"] = False

    all_labeled = bool(containers) and all(
        f"vixens.io/sizing.{cnt.get('name', '')}" in pt_labels for cnt in containers
    )
    vpa_rec = any(v.get("status", {}).get("recommendation") for v in list_resources("vpa", ns))
    c["Sizing validated (sizing labels or VPA recommendation)"] = all_labeled or vpa_rec

    c["No HIGH/CRITICAL CVEs in VulnerabilityReport"] = _trivy_clean(ns, app, {"critical", "high"})
    return c


def _workload_uses_secrets(workload) -> bool:
    for cnt in _containers(workload):
        for env in cnt.get("env", []):
            if "secretKeyRef" in env.get("valueFrom", {}):
                return True
        for ef in cnt.get("envFrom", []):
            if "secretRef" in ef:
                return True
    for vol in _pod_spec(workload).get("volumes", []):
        if "secret" in vol:
            return True
    return False


def _has_infisical_secret(ns: str, app: str) -> bool:
    if list_resources("infisicalsecret", ns, f"app={app}"):
        return True
    return any(
        app.lower() in i.get("metadata", {}).get("name", "").lower()
        for i in list_resources("infisicalsecret", ns)
    )


def _get_pvcs_for_workload(ns: str, workload) -> list:
    pvcs = []
    for vol in _pod_spec(workload).get("volumes", []):
        claim = vol.get("persistentVolumeClaim", {}).get("claimName")
        if claim:
            pvc = _kubectl_json(f"get pvc -n {ns} {claim}")
            if pvc and pvc.get("kind") == "PersistentVolumeClaim":
                pvcs.append(pvc)
    return pvcs


def _app_uses_sqlite(ns: str, app: str, pods: list) -> bool:
    for cm in list_resources("configmap", ns, f"app={app}"):
        if "litestream.yml" in cm.get("data", {}) or "litestream.yaml" in cm.get("data", {}):
            return True
    return _has_litestream_sidecar(pods)


def _has_litestream_sidecar(pods: list) -> bool:
    return any(
        "litestream" in cnt.get("image", "").lower()
        for pod in pods
        for cnt in pod.get("spec", {}).get("containers", [])
    )


def _has_litestream_restore_init(pods: list) -> bool:
    for pod in pods:
        for init in pod.get("spec", {}).get("initContainers", []):
            name = init.get("name", "").lower()
            text = " ".join(init.get("args", []) + init.get("command", []))
            if ("restore" in name or "init" in name) and any(
                kw in text.lower() for kw in [".db", "litestream", "restore", "sqlite"]
            ):
                return True
    return False


def _has_rclone_sidecar(pods: list) -> bool:
    for pod in pods:
        for cnt in pod.get("spec", {}).get("containers", []):
            if "rclone" in cnt.get("image", "").lower():
                text = " ".join(cnt.get("args", []) + cnt.get("command", []))
                if "sync" in text.lower() or "copy" in text.lower():
                    return True
    return False


def _has_rclone_restore_init(pods: list) -> bool:
    for pod in pods:
        for init in pod.get("spec", {}).get("initContainers", []):
            name = init.get("name", "").lower()
            text = " ".join(init.get("args", []) + init.get("command", []))
            if ("restore" in name or "config" in name) and "rclone" in text.lower():
                return True
    return False


def _velero_covers_namespace(ns: str) -> bool:
    data = _kubectl_json("get schedule -n velero")
    if not data:
        return False
    for sched in data.get("items", []):
        included = sched.get("spec", {}).get("template", {}).get("includedNamespaces", [])
        if "*" in included or ns in included:
            return True
    return False


def _trivy_clean(ns: str, app: str, levels: set) -> bool:
    """True if VulnerabilityReport exists for app AND all given severity counts are zero."""
    for r in list_resources("vulnerabilityreport", ns):
        if app.lower() in r.get("metadata", {}).get("name", "").lower():
            summary = r.get("report", {}).get("summary", {})
            return all(summary.get(f"{lvl}Count", 0) == 0 for lvl in levels)
    return False


TIER_CHECKS = [
    ("bronze",     check_bronze),
    ("silver",     check_silver),
    ("gold",       check_gold),
    ("platinum",   check_platinum),
    ("emerald",    check_emerald),
    ("diamond",    check_diamond),
    ("orichalcum", check_orichalcum),
]


def _tier_passes(checks: dict) -> bool:
    return all(v for v in checks.values() if v is not None)


def evaluate(ns: str, app: str, workload, pods: list):
    """
    Walk tiers Bronze → Orichalcum. Stop at first failure.

    Returns:
        current_tier: str | None  — last fully-passing tier (None = below Bronze)
        missing: list[str]        — False checks of the first failed tier
    """
    current_tier = None
    missing = []
    for tier_name, check_fn in TIER_CHECKS:
        checks = check_fn(ns, app, workload, pods)
        if _tier_passes(checks):
            current_tier = tier_name
        else:
            missing = [k for k, v in checks.items() if v is False]
            break
    return current_tier, missing


def _next_tier(current: str):
    if current is None:
        return "bronze"
    idx = TIER_NAMES.index(current)
    return TIER_NAMES[idx + 1] if idx + 1 < len(TIER_NAMES) else None


def display(ns: str, app: str, current_tier, missing: list, next_checks: dict):
    emoji = TIER_EMOJI.get(current_tier, "❌") if current_tier else "❌"
    level = TIER_LEVEL.get(current_tier, 0) if current_tier else 0
    label = current_tier.capitalize() if current_tier else "None (below Bronze)"
    next_t = _next_tier(current_tier)

    print(f"App:        {app}")
    print(f"Namespace:  {ns}")
    if current_tier:
        print(f"Tier:       {emoji} {label} (Level {level})")
    else:
        print("Tier:       ❌ Below Bronze")
    print()

    if not next_t:
        print("🌟 Application has reached maximum tier Orichalcum. Nothing more to do!")
        return

    next_emoji = TIER_EMOJI[next_t]
    next_label = next_t.capitalize()
    next_level = TIER_LEVEL[next_t]
    status_hdr = "❌" if missing else "✅"
    print(f"{status_hdr} Checks for {next_emoji} {next_label} (Level {next_level}):")
    for name, result in next_checks.items():
        sym = "✅" if result is True else ("❌" if result is False else "N/A")
        print(f"  {sym} {name}")


def display_json(ns: str, app: str, current_tier, missing: list):
    print(json.dumps({
        "app": app,
        "namespace": ns,
        "current_tier": current_tier or "none",
        "current_level": TIER_LEVEL.get(current_tier, 0) if current_tier else 0,
        "missing_for_next": missing,
        "next_tier": _next_tier(current_tier),
    }, indent=2))


def main():
    parser = argparse.ArgumentParser(
        prog="evaluate_maturity.py",
        description="Evaluate application maturity tier per ADR-023",
    )
    parser.add_argument("app", help="Application name (Deployment/StatefulSet/DaemonSet name)")
    parser.add_argument("--namespace", "-n", metavar="NS",
                        help="Namespace (auto-detect if not provided)")
    parser.add_argument("--app-path", metavar="PATH",
                        help="Local path to app dir (optional, informational)")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    args = parser.parse_args()

    app_name = args.app
    ns = args.namespace

    if ns:
        workload = get_workload(ns, app_name)
        if not workload:
            print(f"Error: '{app_name}' not found in namespace '{ns}'.", file=sys.stderr)
            sys.exit(1)
    else:
        print(f"Searching for '{app_name}' in cluster...", file=sys.stderr)
        ns, workload = find_app(app_name)
        if not ns:
            print(f"Error: Application '{app_name}' not found in cluster.", file=sys.stderr)
            sys.exit(1)

    pods = get_pods(ns, app_name)
    current_tier, missing = evaluate(ns, app_name, workload, pods)

    next_t = _next_tier(current_tier)
    next_checks = {}
    if next_t:
        for tier_name, check_fn in TIER_CHECKS:
            if tier_name == next_t:
                next_checks = check_fn(ns, app_name, workload, pods)
                break

    if args.json:
        display_json(ns, app_name, current_tier, missing)
    else:
        display(ns, app_name, current_tier, missing, next_checks)


if __name__ == "__main__":
    main()
