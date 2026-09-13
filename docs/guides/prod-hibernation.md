# Production workload hibernation

## Purpose

The production cluster is temporarily capacity-constrained. Selected non-core applications remain declared in GitOps and in ArgoCD, but their workload controllers are hibernated to release scheduling capacity.

## Mechanism

Each selected production overlay imports `apps/_shared/components/prod-hibernate`.

- `Deployment` and `StatefulSet` workloads are set to `replicas: 0`.
- `CronJob` workloads are set to `suspend: true`.
- `DaemonSet` workloads use the unmatched selector `vixens.io/hibernate: "true"`; Kubernetes keeps the controller but schedules no Pods while no node has that label.
- The ArgoCD `Application`, PVCs, Secrets, Services, ingress configuration, and Git history remain present.

KEDA controls are intentionally not modified in this temporary capacity-recovery policy.

## Hibernated applications

- Authentik, BirdNET-Go, Booklore, Bookshelf, Changedetection, CrowdSec, Docspell
- Firefly III and Firefly III Importer
- Grafana, Grafana Ingress, Fluent Bit, Fluent Bit Syslog, Loki, Prometheus, Prometheus Ingress, Promtail, SNMP Exporter, VictoriaMetrics
- Homepage, Hydrus Client, Jellyfin, Jellyseerr, Linkwarden, Mealie, n8n
- NetBird, NetBox, NetVisor, Nexterm, Nightscout, NocoDB, Penpot
- Prowlarr, pyLoad, qBittorrent, Sakapuss, Stirling PDF, Trivy, Vaultwarden, Vikunja, Whisparr
- IT-Tools and Mylar

`infisical-operator` is deprecated in favor of OpenBao/External Secrets, but is not stopped by this hibernation policy: active retained workloads still declare Infisical resources and need a separately validated migration before the operator can be retired.

## Retirement distinction

`g4f` is removed from desired state. `openclaw` was already removed from GitOps in the earlier OpenClaw retirement change; any remaining live object is an ArgoCD reconciliation remnant and must be pruned, not reintroduced.

## Reactivation

To reactivate one application, remove the `../../../_shared/components/prod-hibernate` entry only from that application's `overlays/prod/kustomization.yaml`, then validate, merge to `main`, and promote `prod-stable`.

Do not hibernate ArgoCD, Cilium, cert-manager, External Secrets/OpenBao, ingress/DNS, storage drivers, databases, or Synology CSI as part of this policy.
