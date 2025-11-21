# Verify DNS Redirections for Mail and Home Assistant

## Why

Multiple services will use domain names under `{env}.truxonline.com` pattern. To prevent configuration drift and DNS issues, we need to:

1. **Verify current redirections** - Ensure mail.{env}.truxonline.com and homeassistant.{env}.truxonline.com resolve correctly
2. **Document expected behavior** - Clarify whether these should be CNAME, A records, or Traefik IngressRoutes
3. **Standardize pattern** - Establish consistent DNS configuration across all environments

**Current Concerns:**
- Unclear if `mail.{env}.truxonline.com` redirections exist for all environments
- Unknown if `homeassistant.{env}.truxonline.com` needs DNS setup before deployment
- No documented source of truth for DNS configuration
- Risk of DNS misconfigurations blocking service access

**Expected DNS Patterns:**

| Service | Dev | Test | Staging | Prod |
|---------|-----|------|---------|------|
| **Mail** | mail.dev.truxonline.com | mail.test.truxonline.com | mail.staging.truxonline.com | mail.truxonline.com |
| **Home Assistant** | homeassistant.dev.truxonline.com | homeassistant.test.truxonline.com | homeassistant.staging.truxonline.com | homeassistant.truxonline.com |
| **Traefik** | traefik.dev.truxonline.com | traefik.test.truxonline.com | traefik.staging.truxonline.com | traefik.truxonline.com |
| **ArgoCD** | argocd.dev.truxonline.com | argocd.test.truxonline.com | argocd.staging.truxonline.com | argocd.truxonline.com |

**DNS Resolution Options:**
1. **External DNS (Gandi LiveDNS)**: A/AAAA records pointing to Traefik LoadBalancer IPs
2. **Wildcard DNS**: `*.{env}.truxonline.com` → Traefik LoadBalancer IP
3. **DNS Automation**: External-DNS operator managing DNS records automatically

## What Changes

### Phase 1: DNS Discovery and Verification

**Verify current DNS configuration:**
- Query each expected hostname with `dig` or `nslookup`
- Identify resolution method (A record, CNAME, wildcard)
- Document LoadBalancer IPs for each environment

**Check Gandi LiveDNS configuration:**
- Login to Gandi control panel
- List all DNS records for truxonline.com domain
- Identify existing patterns (subdomains, wildcards)
- Document current TTL values

### Phase 2: DNS Configuration Standards

**Establish DNS standard for vixens infrastructure:**

**Option A: Explicit A Records (Current Approach)**
- Each service gets explicit A record in Gandi
- Pros: Explicit, easy to audit
- Cons: Manual management, no automation

**Option B: Wildcard DNS**
- `*.dev.truxonline.com` → 192.168.208.70 (Traefik LB)
- `*.test.truxonline.com` → 192.168.209.70
- Pros: Single record per environment, automatic for new services
- Cons: Less granular, all subdomains resolve

**Option C: External-DNS Automation**
- Deploy external-dns operator
- Automatically sync Ingress resources to Gandi DNS
- Pros: Full automation, GitOps-compliant
- Cons: Additional complexity, API credentials needed

**Recommendation:** Start with **Option B (Wildcard DNS)** for simplicity, migrate to **Option C (External-DNS)** when infrastructure matures.

### Phase 3: DNS Records Creation/Verification

**For each environment, ensure DNS records exist:**

| Environment | Record Type | Name | Value | TTL |
|-------------|-------------|------|-------|-----|
| Dev | A | *.dev.truxonline.com | 192.168.208.70 | 300 |
| Test | A | *.test.truxonline.com | 192.168.209.70 | 300 |
| Staging | A | *.staging.truxonline.com | 192.168.210.70 | 300 |
| Prod | A | *.truxonline.com | 192.168.200.70 | 300 |

**Verify services resolve correctly:**
```bash
# Dev environment
dig mail.dev.truxonline.com
dig homeassistant.dev.truxonline.com
dig traefik.dev.truxonline.com
dig argocd.dev.truxonline.com

# Test environment
dig mail.test.truxonline.com
dig homeassistant.test.truxonline.com
# ... etc
```

### Phase 4: Ingress Verification

**Ensure Traefik IngressRoutes exist and match DNS:**
- Verify each service has IngressRoute with correct hostname
- Check TLS certificates are issued by cert-manager
- Validate HTTP → HTTPS redirect works
- Test actual service access via browser/curl

### Phase 5: Documentation

**Create DNS runbook:**
- Document current DNS configuration
- Explain wildcard vs explicit records
- Provide procedures for adding new services
- Document Gandi API credentials location (Infisical)

**Update CLAUDE.md and README.md:**
- Add DNS configuration section
- Document hostname patterns for all environments
- Link to DNS runbook

## Impact

**Reliability:**
- ✅ Prevents DNS resolution failures before service deployment
- ✅ Standardized DNS patterns across all environments
- ✅ Clear documentation for DNS troubleshooting

**Operations:**
- ✅ No surprises when deploying new services
- ✅ Automated DNS via wildcard (or external-dns in future)
- ✅ Reduced manual DNS management overhead

**Developer Experience:**
- ✅ Predictable hostname patterns
- ✅ Fast DNS propagation (low TTL)
- ✅ Easy to add new services (wildcard covers all)

**Risk:**
- ⚠️ Wildcard DNS exposes all subdomains (no filtering)
- ⚠️ External-DNS requires Gandi API credentials in Kubernetes
- ⚠️ DNS changes may take time to propagate (TTL)
- Mitigation: Use short TTL (300s), test in dev first, document rollback

## Non-Goals

- Not implementing custom DNS server (use Gandi LiveDNS)
- Not deploying external-dns operator (future enhancement)
- Not creating reverse DNS (PTR records)
- Not configuring DNSSEC (out of scope for homelab)
- Not setting up internal DNS server (CoreDNS sufficient for internal)
