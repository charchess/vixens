# Procedures

Step-by-step operational procedures for the Vixens project.

---

## Available Procedures

### Deployment

- **[Deployment Standard](deployment-standard.md)**
  Standard procedure for deploying applications to the cluster.

### Operations

- **[DSM Password Change](dsm-password-change.md)** ✅
  Procedure for changing Synology DSM password and updating CSI credentials. Includes impact analysis and recovery steps.

- **Backup & Restore** 🚧
  Backup and restore procedures using Velero.

- **Disaster Recovery** 🚧
  Disaster recovery procedures for cluster failures.

- **Cluster Upgrade** 🚧
  Upgrading Kubernetes and Talos versions.

- **Secret Rotation** 🚧
  Rotating secrets in Infisical and Kubernetes (general case).

- **Certificate Renewal** 🚧
  Renewing TLS certificates (manual process if needed).

---

## Procedure vs Guide

**Procedure:** Operational step-by-step for **operators** (backup, DR, upgrades)
**Guide:** How-to for **developers** (adding apps, GitOps, task management)

---

## Creating New Procedures

1. Use [templates/procedure-template.md](../templates/procedure-template.md)
2. Follow naming: `<action>-<object>.md` (e.g., `backup-restore.md`)
3. Include clear prerequisites and validation steps
4. Add to this README
5. Link from main [docs/README.md](../README.md)

---

**Last Updated:** 2025-12-30
