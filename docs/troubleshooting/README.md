# Troubleshooting

Incident logs, post-mortems, and common issues for the Vixens project.

---

## Quick Fixes

- **[Cascade Failure Recovery](cascade-failure-recovery.md)** ✅
  Generic runbook for detecting, diagnosing, and recovering from cascade failures (storage, database, webhook failures).

- **[Common Issues](common-issues.md)** 🚧 Coming soon
  Quick fixes for frequently encountered problems.

---

## Incident Logs

- **[2024-12-25 Final Status](2024-12-25-final-status.md)**
  Final status after cluster rebuild incident.

---

## Post-Mortems

Detailed incident analyses in **[post-mortems/](post-mortems/)**:
- **[2026-01-05 Cluster Production Reset](post-mortems/2026-01-05-cluster-reset.md)** - Critical Terraform incident.
- **2024-12-25 Cluster Rebuild** (see [reports/](../reports/2024-12-25-cluster-redeploy-analysis.md) for detailed analysis).

---

## Troubleshooting Workflow

1. **Check [Common Issues](common-issues.md)** for quick fixes
2. **Search application docs** in [applications/](../applications/)
3. **Check relevant guide** in [guides/](../guides/)
4. **Review ADRs** in [adr/](../adr/) for architectural context
5. **Create incident log** if new issue

---

## Creating Incident Logs

1. Use [templates/troubleshooting-template.md](../templates/troubleshooting-template.md)
2. Follow naming: `YYYY-MM-DD-<short-description>.md`
3. Document symptoms, root cause, and resolution
4. Add to this README
5. Create post-mortem in `post-mortems/` if major incident

---

**Last Updated:** 2025-12-30
