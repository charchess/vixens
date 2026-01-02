# ADR-013: Renovate Discord Approval Workflow

**Date:** 2026-01-02
**Status:** ✅ Implemented
**Deciders:** DevOps, Automation Team
**Tags:** `automation`, `renovate`, `discord`, `approval-workflow`, `gitops`

---

## Context

Renovate Bot was deployed as self-hosted CronJob but lacked:
- ❌ Visibility on new dependency updates
- ❌ Approval workflow before merging
- ❌ Team notifications
- ❌ Configuration transparency (no renovate.json in repo)

Current state:
- Self-hosted Renovate (CronJob in `tools` namespace)
- Configuration in ConfigMap (not visible in Git)
- Manual PR review required
- No notifications
- 10+ orphaned Renovate PRs from previous runs

---

## Decision

**Implement automated Renovate workflow with Discord approval mechanism:**

1. **renovate.json in root** - Configuration visible and versioned in Git
2. **Discord notifications** - Real-time updates on new PRs
3. **Label-based approval** - Add "approved" label to trigger auto-merge
4. **Auto-merge after approval** - Squash & merge for minor/patch updates
5. **Manual review for major** - Major updates require explicit review

### Workflow

```
Renovate Bot detects update
    ↓
Creates PR → dev branch
    ↓
GitHub Actions → Discord webhook notification
    ↓
User reviews PR via Discord link
    ↓
Adds "approved" label on GitHub
    ↓
GitHub Actions triggers auto-merge
    ↓
PR merged → ArgoCD syncs to dev cluster
    ↓
Discord notification: "Deployed to dev"
```

---

## Configuration

### renovate.json (root)
```json
{
  "baseBranches": ["dev"],
  "automerge": true (only for minor/patch with "approved" label),
  "labels": ["dependencies", "renovate"],
  "prConcurrentLimit": 3,
  "prHourlyLimit": 2,
  "schedule": ["every 6 hours"]
}
```

### GitHub Actions Workflow
`.github/workflows/renovate-discord-notify.yaml`:
- Triggered on PR events (opened, labeled, closed)
- Sends Discord embeds with PR details
- Auto-merges when "approved" label added
- Notifies deployment status

### Required Secrets
- `DISCORD_WEBHOOK_RENOVATE` - Discord webhook URL for notifications

---

## Consequences

### Positive ✅

1. **Visibility** - Team knows immediately when updates are available
2. **Approval Control** - No auto-merge without explicit approval
3. **Faster Feedback** - Discord notifications reduce response time
4. **Audit Trail** - All approvals visible in GitHub labels
5. **Configuration Transparency** - renovate.json visible in Git
6. **Reduced PR Noise** - Only 3 concurrent PRs max
7. **Automatic Deployment** - Approved updates deploy to dev automatically

### Negative ⚠️

1. **Discord Dependency** - Requires Discord webhook (could fail silently)
2. **GitHub Secrets Required** - Must configure `DISCORD_WEBHOOK_RENOVATE`
3. **Label Management** - Users must remember to add "approved" label
4. **No Discord Buttons** - Cannot approve directly from Discord (GitHub limitation)

### Mitigations

- Fallback: Check PRs directly on GitHub if Discord fails
- Clear PR templates with approval instructions
- GitHub mobile app for easy label addition

---

## Alternatives Considered

### Alternative A: Dependency Dashboard Only
**Rejected:** Too manual, requires constant checking

### Alternative B: Full Auto-Merge (No Approval)
**Rejected:** Too risky for production-critical dependencies

### Alternative C: Slack Instead of Discord
**Rejected:** Team uses Discord, not Slack

### Alternative D: GitHub Renovate App (Cloud)
**Rejected:** Self-hosted for control and privacy

---

## Implementation

### Phase 1: ✅ Configuration
- [x] Create `renovate.json` in root
- [x] Create `.github/workflows/renovate-discord-notify.yaml`
- [x] Document in ADR-013

### Phase 2: 🔄 Secrets Setup (User Action Required)
- [ ] Create Discord webhook in #renovate channel
- [ ] Add `DISCORD_WEBHOOK_RENOVATE` secret in GitHub repo settings

### Phase 3: ✅ Cleanup
- [x] Close orphaned Renovate PRs (10+)
- [x] Let Renovate recreate PRs with new config

### Phase 4: 📋 Testing
- [ ] Trigger Renovate manually
- [ ] Verify PR created → Discord notification
- [ ] Test approval workflow (add "approved" label)
- [ ] Verify auto-merge + deployment notification

---

## Success Metrics

**After 2 weeks:**
- ✅ 100% of dependency updates visible in Discord
- ✅ Average approval time < 1 hour
- ✅ Zero missed critical security updates
- ✅ All approved PRs auto-merged within 5 minutes

---

## Rollback Strategy

If workflow fails:
1. Remove `.github/workflows/renovate-discord-notify.yaml`
2. Restore manual PR review process
3. Keep `renovate.json` (still useful for configuration)

---

## References

- [Renovate Documentation](https://docs.renovatebot.com/)
- [GitHub Actions - Pull Request Events](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request)
- [Discord Webhooks - Embeds](https://discord.com/developers/docs/resources/webhook#execute-webhook)
- ADR-007: Renovate Dev-First Workflow (Superseded by ADR-013)
- ADR-008: Trunk-Based GitOps Workflow

---

**Decision Owner:** DevOps Team
**Target Implementation Date:** 2026-01-02
**Review Date:** 2026-01-16 (2 weeks post-implementation)
