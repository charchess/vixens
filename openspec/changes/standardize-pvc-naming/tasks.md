# Tasks - Standardize PVC Naming Convention

## Phase 1: Documentation
- [ ] Update CLAUDE.md with PVC naming section (format, examples, rules)
- [ ] Create `.github/PULL_REQUEST_TEMPLATE.md` with PVC naming checklist
- [ ] Document current PVC inventory: `kubectl get pvc -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name`

## Phase 2: Validation
- [ ] Audit existing PVCs for compliance (homeassistant-config ✅, media-pvc ❌)
- [ ] Tag non-compliant PVCs for future rename (optional, low priority)

## Phase 3: Enforcement
- [ ] Add PVC naming check to PR review process
- [ ] All new applications must follow convention
- [ ] Update deploy-* proposals to use convention

---
**Estimated Time:** 30 minutes (documentation only)
**No code changes required** - convention applies to future PVCs
