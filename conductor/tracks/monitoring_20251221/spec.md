# Specification: Monitoring & Observability Enhancements

## Goal
To achieve a comprehensive, full-stack observability platform by deploying Alertmanager, expanding resource analysis (Goldilocks) to the entire cluster, integrating diverse metrics into Grafana, and deploying key visualization tools (Headlamp, Hubble UI).

## Scope
-   **Alertmanager:** Deploy and configure integration with Prometheus.
    -   **Receiver:** Webhook (Credentials retrieved from Infisical).
-   **Goldilocks:** Generalize Vertical Pod Autoscaler (VPA) recommendation analysis to **ALL** namespaces (including system namespaces where safe).
-   **Observability Tools:**
    -   **Headlamp:** Deploy for cluster visualization.
    -   **Hubble UI:** Deploy for network flow observability (Cilium).
    -   **Authentication:** Basic Ingress (No SSO/Auth initially) for both tools.
-   **Integration:**
    -   Verify/Add Dashboards: Kubernetes Compute, PostgreSQL, Traefik, Home Assistant.
    -   Verify Log Aggregation: Ensure Loki/Promtail covers all new namespaces.

## Requirements
-   **GitOps:** All changes via `argocd/overlays`.
-   **Base/Overlay:** Create `base` configurations for Headlamp and Alertmanager if missing.
-   **Secrets:** Use Infisical for Alertmanager webhook URL.
-   **Validation:** Verify access to Headlamp, Hubble UI, and Goldilocks Dashboard. Test Alert firing.
