# Post-Mortems

Detailed incident analyses and lessons learned from major cluster incidents.

## 📝 List of Post-Mortems

- **[2026-02-07 - DSM Password Cascade Failure](2026-02-07-dsm-password-cascade-failure.md)** - Critical cascade failure affecting 50+ applications after DSM password change without proper credential synchronization.
- **[2026-02-06 - Home Assistant Data Reset & Recovery](2026-02-06-homeassistant-reset-recovery.md)** - Critical incident involving data loss and recovery from backup after security hardening attempt.
- **[2026-01-05 - Cluster Production Reset](2026-01-05-cluster-reset.md)** - Critical incident where the production cluster was accidentally reset via Terraform.

---

## 📖 How to Write a Post-Mortem

Use the [Troubleshooting Template](../../templates/troubleshooting-template.md) (if available) or follow the structure of existing reports.

A good post-mortem should include:
1. **Incident Summary:** What happened, when, and what was the impact.
2. **Root Cause Analysis:** Why it happened (technical and process reasons).
3. **Timeline:** Step-by-step log of the incident and recovery.
4. **Recovery Steps:** What was done to fix it.
5. **Prevention Measures:** How to ensure it doesn't happen again.
6. **Lessons Learned:** Key takeaways for the team.
