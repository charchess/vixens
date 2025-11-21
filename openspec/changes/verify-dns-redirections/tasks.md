# Tasks - Verify DNS Redirections

## Phase 1: DNS Discovery

- [ ] Identify Traefik LoadBalancer IPs for each environment:
  - [ ] Dev: `kubectl --kubeconfig terraform/environments/dev/kubeconfig-dev get svc -n traefik traefik -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
  - [ ] Test: Get Traefik LB IP (if deployed)
  - [ ] Staging: Get Traefik LB IP (if deployed)
  - [ ] Prod: Get Traefik LB IP (when deployed)

- [ ] Document expected vs actual LoadBalancer IPs:
  - [ ] Dev expected: 192.168.208.70
  - [ ] Test expected: 192.168.209.70
  - [ ] Staging expected: 192.168.210.70
  - [ ] Prod expected: 192.168.200.70

- [ ] Query existing DNS records:
  - [ ] `dig mail.dev.truxonline.com`
  - [ ] `dig homeassistant.dev.truxonline.com`
  - [ ] `dig traefik.dev.truxonline.com`
  - [ ] `dig argocd.dev.truxonline.com`
  - [ ] `dig whoami.dev.truxonline.com` (existing test service)
  - [ ] Repeat for test, staging, prod environments

- [ ] Check for wildcard DNS records:
  - [ ] `dig *.dev.truxonline.com`
  - [ ] `dig *.test.truxonline.com`
  - [ ] `dig *.staging.truxonline.com`
  - [ ] `dig *.truxonline.com` (prod - be careful with existing records)

- [ ] Document current DNS configuration:
  - [ ] Create inventory table with: hostname, record type, value, TTL
  - [ ] Identify gaps (missing records)
  - [ ] Identify inconsistencies (wrong IPs, CNAMEs pointing to wrong targets)

## Phase 2: Gandi DNS Audit

- [ ] Access Gandi control panel:
  - [ ] Login to account.gandi.net
  - [ ] Navigate to truxonline.com domain
  - [ ] Export current DNS zone file

- [ ] Audit existing records:
  - [ ] List all A records
  - [ ] List all CNAME records
  - [ ] List all wildcard records
  - [ ] Identify records NOT managed by vixens infrastructure
  - [ ] Document external dependencies (MX, TXT, SPF, DKIM records)

- [ ] Verify API access:
  - [ ] Confirm Gandi API key exists in Infisical (if used)
  - [ ] Test API access: `curl -H "Authorization: Apikey $GANDI_API_KEY" https://api.gandi.net/v5/livedns/domains/truxonline.com`

## Phase 3: DNS Standard Decision

- [ ] Evaluate DNS management options:
  - [ ] Document pros/cons of explicit A records
  - [ ] Document pros/cons of wildcard DNS
  - [ ] Document pros/cons of external-dns operator
  - [ ] Recommend approach based on current needs

- [ ] Create DNS configuration standard:
  - [ ] Define hostname pattern: `{service}.{env}.truxonline.com`
  - [ ] Define record type (A vs CNAME)
  - [ ] Define TTL standard (recommend 300s for homelab)
  - [ ] Document special cases (prod without env prefix)

## Phase 4: DNS Records Creation (If Needed)

**If using Wildcard DNS (Recommended):**

- [ ] Create wildcard records in Gandi:
  - [ ] Create `*.dev.truxonline.com` → 192.168.208.70 (A record, TTL 300)
  - [ ] Create `*.test.truxonline.com` → 192.168.209.70 (A record, TTL 300)
  - [ ] Create `*.staging.truxonline.com` → 192.168.210.70 (A record, TTL 300)
  - [ ] Review `*.truxonline.com` carefully (conflicts with existing prod services?)

**If using Explicit Records:**

- [ ] Create explicit A records for dev:
  - [ ] `mail.dev.truxonline.com` → 192.168.208.70
  - [ ] `homeassistant.dev.truxonline.com` → 192.168.208.70
  - [ ] Repeat for test, staging, prod

## Phase 5: DNS Verification

- [ ] Wait for DNS propagation (TTL duration):
  - [ ] Wait 5 minutes for 300s TTL
  - [ ] Flush local DNS cache if testing immediately

- [ ] Verify DNS resolution from multiple locations:
  - [ ] From management host (grenat): `dig mail.dev.truxonline.com @8.8.8.8`
  - [ ] From external network (use online DNS checker)
  - [ ] From within Kubernetes cluster: `kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup mail.dev.truxonline.com`

- [ ] Verify all expected hostnames resolve:
  - [ ] mail.{env}.truxonline.com for each environment
  - [ ] homeassistant.{env}.truxonline.com for each environment
  - [ ] traefik.{env}.truxonline.com for each environment
  - [ ] argocd.{env}.truxonline.com for each environment
  - [ ] whoami.{env}.truxonline.com for each environment

- [ ] Test DNS response times:
  - [ ] Measure query time: `dig mail.dev.truxonline.com | grep "Query time"`
  - [ ] Verify acceptable latency (<100ms)

## Phase 6: Ingress and TLS Verification

- [ ] Verify Traefik receives requests:
  - [ ] `curl -v http://mail.dev.truxonline.com` (should redirect to HTTPS or 404)
  - [ ] Check Traefik logs for incoming requests

- [ ] Verify TLS certificates:
  - [ ] Check cert-manager certificates: `kubectl get certificate -A`
  - [ ] Verify certificates issued for all expected hostnames
  - [ ] Test HTTPS access: `curl -v https://whoami.dev.truxonline.com`
  - [ ] Verify certificate chain with: `openssl s_client -connect whoami.dev.truxonline.com:443 -servername whoami.dev.truxonline.com </dev/null`

- [ ] Verify HTTP → HTTPS redirect:
  - [ ] `curl -I http://whoami.dev.truxonline.com` (should return 301/302 with Location: https://...)

## Phase 7: Documentation

- [ ] Create DNS runbook:
  - [ ] Create `docs/runbooks/dns-management.md`
  - [ ] Document current DNS configuration (wildcard or explicit)
  - [ ] Document Gandi API access procedures
  - [ ] Add procedures for adding new services
  - [ ] Add troubleshooting section (DNS not resolving, slow propagation, etc.)

- [ ] Update CLAUDE.md:
  - [ ] Add DNS configuration section
  - [ ] Document hostname patterns for all services
  - [ ] Link to DNS runbook

- [ ] Update README.md:
  - [ ] Add DNS section with example hostnames
  - [ ] Document how to access services

- [ ] Create DNS inventory document:
  - [ ] Create `docs/dns-inventory.md`
  - [ ] Table with all DNS records
  - [ ] Last verified date
  - [ ] Owner/responsible party

## Phase 8: Future Enhancement Planning

- [ ] Research external-dns operator:
  - [ ] Review external-dns documentation
  - [ ] Identify Gandi provider support
  - [ ] Document deployment requirements
  - [ ] Create placeholder ADR for future external-dns deployment

- [ ] Document DNS automation roadmap:
  - [ ] Phase 1: Manual wildcard (current)
  - [ ] Phase 2: Semi-automated with scripts
  - [ ] Phase 3: external-dns operator (future)
