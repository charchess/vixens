# Procedures

Step-by-step operational procedures for the Vixens project.

---

## Available Procedures

### Deployment

- **[Deployment Standard](deployment-standard.md)**
  Standard procedure for deploying applications to the cluster using the current GitOps and OpenBao/External Secrets conventions.

### Operations (Coming Soon)

- **Backup & Restore** 🚧
  Backup and restore procedures using Velero.

- **Disaster Recovery** 🚧
  Disaster recovery procedures for cluster failures.

- **Cluster Upgrade** 🚧
  Upgrading Kubernetes and Talos versions.

- **Secret Rotation** 🚧
  Rotating values in OpenBao and validating External Secrets reconciliation into Kubernetes Secrets.

- **Certificate Renewal** 🚧
  cert-manager certificate renewal and recovery procedures.

---

## Procedure vs Guide

**Procedure:** Operational step-by-step for **operators** (backup, DR, upgrades, recovery).

**Guide:** How-to for **contributors** (adding apps, GitOps, task management, secret patterns).

`WORKFLOW.md` remains authoritative for repository workflow and production promotion.

---

## Creating New Procedures

1. Start from current `main` and inspect open PRs first.
2. Use [templates/procedure-template.md](../templates/procedure-template.md) when it matches the task.
3. Follow naming: `<action>-<object>.md` (for example `backup-restore.md`).
4. Include prerequisites, rollback/recovery considerations, and explicit validation steps.
5. Do not embed credentials or secret values; reference OpenBao paths conceptually when needed.
6. Add the procedure to this README and link it from the relevant active documentation.

Historical procedures and audits may mention retired tooling; active procedures must use current platform terminology and mechanisms.

---

**Last Updated:** 2026-09-25
