#!/usr/bin/env python3
"""#3728: prove the bounded private POC in five real Kustomize YAML renders.

This validator reads only local source renders and never accesses the cluster.
Existing TXO Infra-backed ExternalSecrets are historic and out of scope:
check *all new OpenBao Fabric objects* and forbid newly introduced Infra
store references via validate-openbao-fabric-source.py.
"""
from __future__ import annotations

import argparse
import pathlib
import re
import sys

import yaml

FABRIC_NAMES = {
    ("StatefulSet", "openbao-fabric"),
    ("ConfigMap", "openbao-fabric-config"),
    ("Service", "openbao-fabric"),
    ("NetworkPolicy", "openbao-fabric-private"),
}
TLS_NAMES = {
    ("Issuer", "openbao-fabric-selfsigned"),
    ("Certificate", "openbao-fabric-ca"),
    ("Issuer", "openbao-fabric-ca-issuer"),
    ("Certificate", "openbao-fabric-server"),
}
SERVICE_DNS = [
    "openbao-fabric",
    "openbao-fabric.txo-fabric-system",
    "openbao-fabric.txo-fabric-system.svc",
    "openbao-fabric.txo-fabric-system.svc.cluster.local",
]
ISSUER_GROUP = "cert-manager.io"
KEY_PROFILE = {"algorithm": "ECDSA", "size": 256, "rotationPolicy": "Always"}

LABELS = {"app.kubernetes.io/name": "openbao-fabric", "app.kubernetes.io/component": "secret-plane"}
FORBIDDEN_KINDS = {"Secret", "ExternalSecret", "SecretStore", "ClusterSecretStore",
                   "Ingress", "Gateway", "HTTPRoute", "Job", "CronJob", "Deployment"}
BAD_STRINGS = ("openbao-token", "nas.truxonline.com", "openbao-fabric-store",
               "ClusterSecretStore", "8642")


def parse_yaml(path: pathlib.Path) -> list[dict]:
    with path.open(encoding="utf-8") as file:
        docs = list(yaml.safe_load_all(file))
    if not docs or any(not isinstance(d, dict) for d in docs):
        raise ValueError("missing or invalid render")
    return docs


def check_render(docs: list[dict], *, selected: bool = False) -> list[str]:
    errors: list[str] = []
    # The base component must contain only these eight sources. A historic
    # Infra ExternalSecret elsewhere in TXO compositions is not a new
    # Fabric consumer; source-only diff guard separately forbids new Infra use.
    expected = FABRIC_NAMES | TLS_NAMES
    fabric = [
        d for d in docs if (
            d.get("metadata", {}).get("labels", {}).get("app.kubernetes.io/name") == "openbao-fabric"
            or (d.get("kind"), d.get("metadata", {}).get("name")) in expected
        )
    ]
    pairs = [(d.get("kind"), d.get("metadata", {}).get("name")) for d in fabric]
    if len(pairs) != len(expected) or set(pairs) != expected:
        return [f"required exact eight Fabric/TLS resources, got {pairs}"]
    if not selected and len(docs) != len(expected):
        return ["unexpected additional client/Secret/resource in private base/overlay"]

    if any(d["metadata"].get("namespace") != "txo-fabric-system" for d in fabric):
        errors.append("wrong Fabric namespace")
    if any(not all(d["metadata"].get("labels", {}).get(k) == v for k, v in LABELS.items()) for d in fabric):
        errors.append("missing Fabric name/component labels")
    by_kind = {d["kind"]: d for d in fabric if (d["kind"], d["metadata"]["name"]) in FABRIC_NAMES}
    tls_resources = {(d["kind"], d["metadata"]["name"]): d for d in fabric
                     if (d["kind"], d["metadata"]["name"]) in TLS_NAMES}
    issuer_ref = lambda name: {"name": name, "kind": "Issuer", "group": ISSUER_GROUP}
    expected_tls = {
        ("Issuer", "openbao-fabric-selfsigned"): {"selfSigned": {}},
        ("Certificate", "openbao-fabric-ca"): {
            "secretName": "openbao-fabric-ca",
            "isCA": True,
            "commonName": "openbao-fabric-ca",
            "duration": "8760h",
            "renewBefore": "720h",
            "privateKey": KEY_PROFILE,
            "usages": ["cert sign", "crl sign"],
            "issuerRef": issuer_ref("openbao-fabric-selfsigned"),
        },
        ("Issuer", "openbao-fabric-ca-issuer"): {
            "ca": {"secretName": "openbao-fabric-ca"},
        },
        ("Certificate", "openbao-fabric-server"): {
            "secretName": "openbao-fabric-tls",
            "duration": "2160h",
            "renewBefore": "360h",
            "privateKey": KEY_PROFILE,
            "usages": ["digital signature", "server auth"],
            "dnsNames": SERVICE_DNS,
            "issuerRef": issuer_ref("openbao-fabric-ca-issuer"),
        },
    }
    for identity, wanted in expected_tls.items():
        obj = tls_resources[identity]
        if obj.get("apiVersion") != ISSUER_GROUP + "/v1" or obj.get("spec") != wanted:
            errors.append(f"invalid internal cert-manager chain/SANs for {identity}")
    if any(d["kind"] in FORBIDDEN_KINDS for d in fabric):
        errors.append("forbidden Fabric resource")
    if any(x in str(fabric) for x in BAD_STRINGS):
        errors.append("Infra/ESO/Hermes data in new Fabric component")
    for item in fabric:
        anns = item["metadata"].get("annotations", {})
        if any(x in str(anns).lower() for x in ("prune", "delete", "force", "sync-options")):
            errors.append("destructive/sync-option annotation in Fabric resource")
    sts = by_kind["StatefulSet"]
    spec = sts["spec"]
    pod = spec["template"]["spec"]
    match = spec["selector"]["matchLabels"]
    if spec.get("replicas") != 1 or spec.get("serviceName") != "openbao-fabric":
        errors.append("exactly one replica and private Service required")
    if spec.get("persistentVolumeClaimRetentionPolicy") != {"whenDeleted": "Retain", "whenScaled": "Retain"}:
        errors.append("PVC template Retain lifecycle required")
    claims = spec.get("volumeClaimTemplates", [])
    if len(claims) != 1:
        errors.append("exactly one claim template required")
    else:
        c = claims[0]
        if (c["metadata"].get("name") != "data" or
            c["spec"].get("storageClassName") != "truenas-iscsi-retain" or
            c["spec"].get("accessModes") != ["ReadWriteOnce"] or
            c["spec"].get("resources", {}).get("requests", {}).get("storage") != "5Gi"):
            errors.append("PVC must be 5Gi RWO truenas-iscsi-retain")
    if not all(spec["template"]["metadata"]["labels"].get(k) == v for k, v in match.items()):
        errors.append("StatefulSet pod selector mismatch")
    if pod.get("automountServiceAccountToken") is not False or "serviceAccountName" in pod:
        errors.append("Kubernetes SA token/binding forbidden")
    if any(k in pod for k in ("initContainers", "nodeName", "nodeSelector", "affinity")):
        errors.append("bootstrap or forced scheduling forbidden")
    pod_security = pod.get("securityContext", {})
    if pod_security.get("runAsNonRoot") is not True:
        errors.append("nonroot required")
    # OCI User=openbao is non-numeric. Kubelet fails closed even with
    # runAsNonRoot=true unless we specify the image-proven numeric UID/GID.
    if (type(pod_security.get("runAsUser")) is not int
            or pod_security["runAsUser"] != 100
            or type(pod_security.get("runAsGroup")) is not int
            or pod_security["runAsGroup"] != 1000
            or type(pod_security.get("fsGroup")) is not int
            or pod_security["fsGroup"] != 1000):
        errors.append("OpenBao 2.6.2 requires image-proven numeric UID 100, GID 1000 and fsGroup 1000")
    containers = pod.get("containers", [])
    if len(containers) != 1:
        errors.append("exactly one runtime container required")
    else:
        c = containers[0]
        if (c.get("image") != "quay.io/openbao/openbao:2.6.2" or
            c.get("command") != ["/usr/bin/bao"] or
            c.get("args") != ["server", "-config=/openbao/config/server.hcl"]):
            errors.append("versioned production OpenBao server required")
        if c.get("resources") != {
            "requests": {"cpu": "100m", "memory": "128Mi"},
            "limits": {"cpu": "500m", "memory": "512Mi"},
        }:
            errors.append("resource budget mismatch")
        if (c.get("readinessProbe", {}).get("httpGet") != {
            "path": "/v1/sys/health", "port": "api", "scheme": "HTTPS"
        } or "livenessProbe" in c or "startupProbe" in c):
            errors.append("unsealed-only readiness and no liveness required")
        if (c.get("securityContext", {}).get("allowPrivilegeEscalation") is not False or
            c.get("securityContext", {}).get("readOnlyRootFilesystem") is not True or
            c.get("securityContext", {}).get("capabilities", {}).get("drop") != ["ALL"]):
            errors.append("server hardening missing")
        if [(p.get("containerPort"), p.get("protocol")) for p in c.get("ports", [])] != [(8200, "TCP")]:
            errors.append("OpenBao container port must be API 8200 only")
        mounts = {m["name"]: m["mountPath"] for m in c.get("volumeMounts", [])}
        if mounts != {
            "data": "/openbao/data",
            "config": "/openbao/config",
            "scratch": "/tmp",
            "tls": "/openbao/tls",
        } or not all(m.get("readOnly") is True for m in c.get("volumeMounts", [])
                     if m.get("name") in {"config", "tls"}):
            errors.append("wrong Raft/config/scratch/TLS volume mounts or not read-only")
    volumes = {vol.get("name"): vol for vol in pod.get("volumes", [])}
    if volumes != {
        "config": {"name": "config", "configMap": {"name": "openbao-fabric-config"}},
        "scratch": {"name": "scratch", "emptyDir": {"sizeLimit": "32Mi"}},
        "tls": {"name": "tls", "secret": {"secretName": "openbao-fabric-tls", "defaultMode": 0o440}},
    }:
        errors.append("only the exact read-only cert-manager TLS Secret reference is allowed")

    service = by_kind["Service"]["spec"]
    if (service.get("type") != "ClusterIP" or service.get("externalIPs") or
        service.get("externalName") or service.get("loadBalancerIP") or
        service.get("selector") != match or
        service.get("ports") != [{"name": "api", "port": 8200, "targetPort": "api", "protocol": "TCP"}]):
        errors.append("private ClusterIP/8200 exact Service required")
    policy = by_kind["NetworkPolicy"]["spec"]
    if (policy.get("podSelector", {}).get("matchLabels") != match or
        set(policy.get("policyTypes", [])) != {"Ingress", "Egress"} or
        policy.get("ingress") != [] or policy.get("egress") != []):
        errors.append("no-client private deny-all network policy required")
    config = by_kind["ConfigMap"].get("data", {})
    hcl = config.get("server.hcl", "")
    if set(config) != {"server.hcl"} or not all(x in hcl for x in (
        'storage "raft"', 'path = "/openbao/data"', 'node_id = "openbao-fabric-0"',
        'listener "tcp"', 'address = "0.0.0.0:8200"',
        'api_addr = "https://openbao-fabric.txo-fabric-system.svc:8200"',
        'tls_cert_file = "/openbao/tls/tls.crt"',
        'tls_key_file = "/openbao/tls/tls.key"',
    )):
        errors.append("non-dev private Raft HTTPS HCL required")
    if re.search(r"(?im)^\s*(?:tls_disable|disable_mlock)\s*=", hcl) or (
        "http://" in hcl or "-----BEGIN" in hcl or
        re.search(r"(?im)^\s*(?:seal|auto_unseal|token|root_token|password|private_key|"
                  r"tls_cert|tls_key|certificate|ca_cert)\s*[={]", hcl)
    ):
        errors.append("HTTP/disabled TLS/obsolete mlock/inline key or secret forbidden in HCL")
    if selected:
        default_denies = [d for d in docs if d.get("kind") == "NetworkPolicy"
                          and d.get("metadata", {}).get("name") == "default-deny"
                          and d.get("metadata", {}).get("namespace") == "txo-fabric-system"]
        if len(default_denies) != 1 or default_denies[0]["spec"].get("podSelector") != {}:
            errors.append("namespace default-deny missing")
    return errors


def check_prod_gates(docs: list[dict]) -> list[str]:
    errors = []
    for resource, variable in (
        ("txo-fabric-webui-bff", "TXO_WEBUI_BFF_BROWSER_ENABLED"),
        ("txo-fabric-operator", "TXO_FABRIC_PRIVATE_BRIDGE_ENABLED"),
    ):
        matches = [d for d in docs if d.get("kind") == "Deployment"
                   and d.get("metadata", {}).get("name") == resource]
        if len(matches) != 1:
            errors.append(f"missing gate owner {resource}")
            continue
        envs = [env for c in matches[0]["spec"]["template"]["spec"]["containers"]
                for env in c.get("env", []) if env.get("name") == variable]
        if len(envs) != 1 or envs[0].get("value") != "false":
            errors.append(f"{variable} gate changed")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("base", "dev", "prod", "txo-dev", "txo-prod"):
        parser.add_argument("--" + name, type=pathlib.Path, required=True)
    args = parser.parse_args()
    errors = []
    for name in ("base", "dev", "prod", "txo-dev", "txo-prod"):
        try:
            docs = parse_yaml(getattr(args, name.replace("-", "_")))
            errors.extend(f"{name}: {e}" for e in check_render(docs, selected=name.startswith("txo-")))
            if name == "txo-prod":
                errors.extend(f"{name}: {e}" for e in check_prod_gates(docs))
        except (OSError, ValueError, yaml.YAMLError, TypeError, KeyError) as exc:
            errors.append(f"{name}: invalid render ({type(exc).__name__})")
    for error in errors:
        print("FAIL: " + error, file=sys.stderr)
    if not errors:
        print("PASS: five Kustomize renders: four private Fabric + four namespace-local cert-manager objects")
        print("PASS: exact Service SANs, HTTPS listener and probe, TLS Secret reference-only, Raft Retain")
        print("PASS: deny-all, no consumers/credential inline; BFF/bridge gates false")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
