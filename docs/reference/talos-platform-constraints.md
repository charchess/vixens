# Talos Linux — Platform Constraints

**Last Updated:** 2026-03-25

## Overview

Talos Linux is an immutable, API-driven OS designed for Kubernetes. Its minimal design means several standard Kubernetes features are **not available**. This document tracks evaluated features and their applicability.

---

## Not Available on Talos

### AppArmor Profiles (evaluated 2026-03-25, issue #2280)

**Status:** Not applicable.

Talos does not ship the AppArmor kernel module or userspace tools (`apparmor_parser`). Kubernetes 1.30+ made AppArmor GA with `appArmorProfile` in `securityContext`, but it requires the **node** to have AppArmor loaded.

**Alternatives for container hardening:**
- **seccomp profiles** — supported by Talos
- **Pod Security Standards (PSA)** — built into Kubernetes
- **Trivy** — already deployed for vulnerability scanning

### NodeLogQuery / KEP-2258 (evaluated 2026-03-25, issue #2281)

**Status:** Not applicable.

NodeLogQuery allows querying node-level logs via the kubelet API. It relies on either:
- **journald** (systemd journal) — Talos does not use systemd
- **File-based log providers** — not available on Talos's read-only filesystem

Still in Beta as of Kubernetes 1.34. Not GA.

**Alternatives:**
- `talosctl logs` / `talosctl dmesg` — for node-level logs
- **Promtail + Loki** — already deployed for centralized log collection

---

## Available on Talos

| Feature | Status | Notes |
|---|---|---|
| seccomp profiles | Supported | Default seccomp provided by containerd |
| iSCSI (via extension) | Supported | `iscsi-tools` extension required |
| Native sidecars (KEP-753) | Supported | K8s 1.28+, `restartPolicy: Always` on init containers |
| fsGroupChangePolicy | Supported | Requires explicit configuration per pod |
| VPA | Supported | Deployed via `apps/00-infra/vpa/` |
