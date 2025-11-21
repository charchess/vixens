# Verify ClusterIssuer Versions Across Environments

## Why

ClusterIssuers are critical infrastructure components for TLS certificate management. Inconsistencies across environments can lead to:

**Problems:**
- Certificate issuance failures in non-dev environments
- Debugging confusion (works in dev, fails in prod)
- Security issues (using staging Let's Encrypt in production)
- API rate limiting (excessive retries against wrong ACME server)

**Current Concerns:**
- Unknown if `letsencrypt-staging` and `letsencrypt-prod` exist in all environments
- Unclear which ClusterIssuer is used by default in each environment
- No validation that DNS-01 challenge configuration is consistent
- Risk of mismatched Gandi API credentials across environments

**Expected State:**
Each environment should have **two ClusterIssuers**:
1. **letsencrypt-staging** - For testing (high rate limits, untrusted certs)
2. **letsencrypt-prod** - For production use (trusted certs, rate limited)

**Usage Pattern:**
- Dev/Test: Primarily use `letsencrypt-staging` for rapid iteration
- Staging: Use `letsencrypt-prod` to validate full certificate chain
- Prod: Only use `letsencrypt-prod`

## What Changes

### Phase 1: ClusterIssuer Discovery

**Inventory existing ClusterIssuers:**
- Query each environment: `kubectl get clusterissuer`
- Document which ClusterIssuers exist in each environment
- Check for unexpected or deprecated ClusterIssuers

**Verify ClusterIssuer configuration:**
- Extract YAML: `kubectl get clusterissuer {name} -o yaml`
- Compare key fields across environments:
  - ACME server URL (staging vs prod)
  - DNS-01 solver configuration (Gandi webhook)
  - Email address for certificate notifications
  - Gandi API token secret reference

**Check cert-manager webhook:**
- Verify `cert-manager-webhook-gandi` deployed in all environments
- Check webhook version consistency
- Validate webhook service and endpoints

### Phase 2: Configuration Standardization

**Define ClusterIssuer standard:**

**letsencrypt-staging:**
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: admin@truxonline.com
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
      - dns01:
          webhook:
            groupName: acme.gandi.net
            solverName: gandi
            config:
              apiKeySecretRef:
                name: gandi-credentials
                key: api-token
```

**letsencrypt-prod:**
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@truxonline.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - dns01:
          webhook:
            groupName: acme.gandi.net
            solverName: gandi
            config:
              apiKeySecretRef:
                name: gandi-credentials
                key: api-token
```

**Validate key differences:**
- Only `spec.acme.server` should differ
- Email, solver config, and secret refs must be identical
- Namespace for `gandi-credentials` secret must be `cert-manager`

### Phase 3: Certificate Verification

**Check existing certificates:**
- List all certificates: `kubectl get certificate -A`
- Verify issuerRef for each certificate:
  - Dev/Test: Expect mix of staging and prod
  - Staging: Expect mostly prod
  - Prod: Expect ONLY prod
- Check certificate status (Ready, expiration date)

**Identify certificates using wrong issuer:**
- Find certificates using `letsencrypt-staging` in prod
- Document plan to recreate with `letsencrypt-prod`
- Ensure no service disruption during rotation

**Test certificate issuance:**
- Create test certificate in each environment
- Verify DNS-01 challenge completes successfully
- Check Let's Encrypt logs for errors
- Validate issued certificate trust chain

### Phase 4: Remediation (If Needed)

**If ClusterIssuers missing or misconfigured:**
- Update `apps/infrastructure/cert-manager-webhook-gandi/base/clusterissuer.yaml`
- Ensure both `letsencrypt-staging` and `letsencrypt-prod` defined
- Apply via ArgoCD sync

**If Gandi credentials missing:**
- Verify Infisical secrets deployed: `kubectl get infisicalsecret -n cert-manager`
- Check secret exists: `kubectl get secret gandi-credentials -n cert-manager`
- Validate secret contains `api-token` key
- Test credentials: Issue test certificate

**If certificates using wrong issuer:**
- Update Ingress annotations or Certificate resources
- Delete old certificate: `kubectl delete certificate {name} -n {namespace}`
- Wait for cert-manager to reissue with correct ClusterIssuer
- Verify new certificate is trusted

### Phase 5: Documentation

**Create ClusterIssuer runbook:**
- Document standard ClusterIssuer configuration
- Explain staging vs prod usage
- Provide troubleshooting procedures
- Document Let's Encrypt rate limits

**Update CLAUDE.md:**
- Add ClusterIssuer configuration details
- Document which issuer to use in which environment

**Create certificate management guide:**
- How to request new certificate
- How to rotate certificates
- How to troubleshoot issuance failures
- How to check rate limit status

## Impact

**Security:**
- ✅ Ensures production uses trusted Let's Encrypt certificates
- ✅ Prevents staging certificates in production
- ✅ Consistent ACME server configuration

**Reliability:**
- ✅ Reduces certificate issuance failures
- ✅ Prevents rate limit exhaustion
- ✅ Consistent DNS-01 challenge configuration

**Operations:**
- ✅ Predictable certificate behavior across environments
- ✅ Clear troubleshooting procedures
- ✅ Documentation for certificate management

**Risk:**
- ⚠️ Certificate rotation may cause brief service interruption
- ⚠️ Rate limit exhaustion if recreating many certificates
- ⚠️ DNS propagation delays may slow DNS-01 challenges
- Mitigation: Rotate certificates during maintenance window, use staging for testing

## Non-Goals

- Not implementing certificate monitoring/alerting (future enhancement)
- Not automating certificate rotation beyond cert-manager
- Not configuring HTTP-01 challenge (only DNS-01)
- Not implementing certificate backup/restore (rely on Let's Encrypt re-issuance)
- Not supporting multiple DNS providers (only Gandi)
