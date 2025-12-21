# Tech Stack: Vixens GitOps Platform

## Core Infrastructure
- **Operating System:** Talos Linux (Immutable, API-managed OS).
- **Container Orchestration:** Kubernetes.
- **Infrastructure as Code:** Terraform (Version-controlled provisioning).
- **GitOps Engine:** ArgoCD (Continuous Delivery for K8s resources).

## Networking & Security
- **CNI:** Cilium (High-performance networking with L2 announcements and observability).
- **Secrets Management:** Infisical (Centralized secret storage and injection).
- **Certificates:** Cert-manager (Automated TLS via Let's Encrypt and Gandi DNS).
- **Ingress Controller:** Traefik (Edge router and load balancer).

## Storage
- **CSI Driver:** Synology CSI (Support for persistent data via iSCSI and NFS).

## Monitoring & Observability
- **Stack:** Prometheus, Grafana, Loki (Full-stack visibility).
- **Network Observability:** Cilium Hubble.

## Development & Automation
- **Task Management:** Archon MCP (AI-driven task tracking and documentation).
- **Validation:** Playwright (Frontend testing), Curl (API/Endpoint verification).
- **Configuration:** Kustomize (Overlay-based K8s management).
