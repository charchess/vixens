# Tasks - Verify ClusterIssuer Versions

## Phase 1: ClusterIssuer Discovery (Dev)

- [ ] Check ClusterIssuers in dev environment:
  - [ ] `kubectl --kubeconfig terraform/environments/dev/kubeconfig-dev get clusterissuer`
  - [ ] Document which ClusterIssuers exist
  - [ ] Check for unexpected or deprecated issuers

- [ ] Extract ClusterIssuer configurations:
  - [ ] `kubectl get clusterissuer letsencrypt-staging -o yaml > /tmp/dev-staging.yaml`
  - [ ] `kubectl get clusterissuer letsencrypt-prod -o yaml > /tmp/dev-prod.yaml`

- [ ] Verify key configuration fields:
  - [ ] ACME server URL (staging: `acme-staging-v02.api.letsencrypt.org`, prod: `acme-v02.api.letsencrypt.org`)
  - [ ] Email address (should be consistent)
  - [ ] DNS-01 solver configuration (Gandi webhook)
  - [ ] Secret reference name: `gandi-credentials`
  - [ ] Webhook groupName: `acme.gandi.net`

- [ ] Check ClusterIssuer status:
  - [ ] `kubectl get clusterissuer -o wide`
  - [ ] Verify both show "Ready" status
  - [ ] Check for error conditions

- [ ] Verify cert-manager-webhook-gandi:
  - [ ] `kubectl get pods -n cert-manager | grep webhook-gandi`
  - [ ] Check webhook is running and healthy
  - [ ] Verify webhook service exists
  - [ ] Check webhook version matches expected

## Phase 2: ClusterIssuer Discovery (Other Environments)

- [ ] Repeat Phase 1 for test environment (if deployed):
  - [ ] List ClusterIssuers
  - [ ] Extract configurations
  - [ ] Compare with dev standard

- [ ] Repeat Phase 1 for staging environment (if deployed)
- [ ] Repeat Phase 1 for prod environment (if deployed)

- [ ] Create comparison matrix:
  - [ ] Document differences between environments
  - [ ] Identify configuration drift
  - [ ] Highlight missing ClusterIssuers

## Phase 3: Certificate Inventory

- [ ] List all certificates in dev:
  - [ ] `kubectl get certificate -A`
  - [ ] Document certificate names, namespaces, and issuerRef
  - [ ] Check certificate status (Ready, expiration dates)

- [ ] Identify certificates by issuer:
  - [ ] Count using letsencrypt-staging
  - [ ] Count using letsencrypt-prod
  - [ ] Identify any using unexpected issuers

- [ ] Check certificate details:
  - [ ] `kubectl describe certificate -n {namespace} {name}`
  - [ ] Verify DNS-01 challenge status
  - [ ] Check for recent renewal events
  - [ ] Identify any stuck or failing certificates

- [ ] Repeat certificate inventory for test, staging, prod (if deployed)

## Phase 4: Gandi Credentials Verification

- [ ] Check Infisical secrets in dev:
  - [ ] `kubectl get infisicalsecret -n cert-manager`
  - [ ] Verify `gandi-credentials-sync` exists and is synced
  - [ ] Check InfisicalSecret status and conditions

- [ ] Verify Kubernetes secret exists:
  - [ ] `kubectl get secret gandi-credentials -n cert-manager`
  - [ ] Verify secret type is Opaque
  - [ ] Check secret has `api-token` key: `kubectl get secret gandi-credentials -n cert-manager -o jsonpath='{.data.api-token}' | base64 -d | wc -c` (should return >0)

- [ ] Repeat Gandi credentials check for all environments

## Phase 5: Test Certificate Issuance

- [ ] Create test certificate in dev (staging issuer):
  ```bash
  kubectl apply -f - <<EOF
  apiVersion: cert-manager.io/v1
  kind: Certificate
  metadata:
    name: test-cert-staging
    namespace: cert-manager
  spec:
    secretName: test-cert-staging-tls
    issuerRef:
      name: letsencrypt-staging
      kind: ClusterIssuer
    dnsNames:
      - test-staging.dev.truxonline.com
  EOF
  ```

- [ ] Monitor certificate issuance:
  - [ ] `kubectl describe certificate test-cert-staging -n cert-manager`
  - [ ] Watch for DNS-01 challenge progress
  - [ ] Check for "Certificate issued successfully" event
  - [ ] Verify secret created: `kubectl get secret test-cert-staging-tls -n cert-manager`

- [ ] Verify certificate trust chain:
  - [ ] Extract certificate: `kubectl get secret test-cert-staging-tls -n cert-manager -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/test-staging.crt`
  - [ ] Check issuer: `openssl x509 -in /tmp/test-staging.crt -noout -issuer`
  - [ ] Verify it's a staging certificate (issuer contains "Fake LE")

- [ ] Create test certificate with prod issuer:
  - [ ] Repeat above with `letsencrypt-prod` issuer
  - [ ] Verify it's a trusted certificate (issuer contains "Let's Encrypt")

- [ ] Cleanup test certificates:
  - [ ] `kubectl delete certificate test-cert-staging -n cert-manager`
  - [ ] `kubectl delete certificate test-cert-prod -n cert-manager`

## Phase 6: Configuration Standardization

- [ ] Review GitOps ClusterIssuer configuration:
  - [ ] Check `apps/infrastructure/cert-manager-webhook-gandi/base/clusterissuer.yaml`
  - [ ] Verify both staging and prod issuers defined
  - [ ] Validate ACME server URLs are correct
  - [ ] Check email address is consistent

- [ ] Validate Kustomize overlays:
  - [ ] Check `apps/infrastructure/cert-manager-webhook-gandi/overlays/{env}/`
  - [ ] Verify no environment-specific ClusterIssuer patches
  - [ ] Ensure ClusterIssuers are consistent across environments

- [ ] If configuration drift detected:
  - [ ] Update base ClusterIssuer configuration
  - [ ] Remove environment-specific patches if inappropriate
  - [ ] Commit changes to Git
  - [ ] Push to trigger ArgoCD sync

## Phase 7: Remediation (If Needed)

**If ClusterIssuers missing in an environment:**

- [ ] Verify ArgoCD Application for cert-manager-config:
  - [ ] `kubectl get application cert-manager-config -n argocd`
  - [ ] Check sync status
  - [ ] Review application events

- [ ] Force ArgoCD sync:
  - [ ] Via UI: ArgoCD dashboard → cert-manager-config → Sync
  - [ ] Via CLI: `kubectl patch application cert-manager-config -n argocd --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"normal"}}}'`

- [ ] Validate ClusterIssuers created:
  - [ ] `kubectl get clusterissuer`
  - [ ] Verify both staging and prod exist

**If Gandi credentials missing:**

- [ ] Check Infisical secret sync status:
  - [ ] `kubectl get infisicalsecret gandi-credentials-sync -n cert-manager -o yaml`
  - [ ] Review conditions and events
  - [ ] Check Infisical operator logs

- [ ] Verify Infisical configuration:
  - [ ] Check projectSlug is `vixens`
  - [ ] Check envSlug matches environment (dev, test, staging, prod)
  - [ ] Check secretsPath is `/cert-manager`
  - [ ] Verify clientId/clientSecret in `infisical-universal-auth` secret

- [ ] Test Infisical API access manually:
  - [ ] Use clientId/clientSecret to authenticate
  - [ ] Verify secret exists in Infisical UI

**If certificates using wrong issuer:**

- [ ] Identify affected certificates:
  - [ ] Production certificates using letsencrypt-staging

- [ ] Update certificate issuerRef:
  - [ ] Edit Certificate resource
  - [ ] Change `spec.issuerRef.name` to `letsencrypt-prod`
  - [ ] Or update Ingress annotation: `cert-manager.io/cluster-issuer: letsencrypt-prod`

- [ ] Force certificate renewal:
  - [ ] Delete certificate: `kubectl delete certificate {name} -n {namespace}`
  - [ ] cert-manager will recreate with new issuer
  - [ ] Monitor issuance: `kubectl describe certificate {name} -n {namespace}`

## Phase 8: Documentation

- [ ] Create ClusterIssuer runbook:
  - [ ] Create `docs/runbooks/clusterissuer-management.md`
  - [ ] Document standard configuration for both issuers
  - [ ] Explain when to use staging vs prod
  - [ ] Add troubleshooting section (common errors, rate limits)
  - [ ] Document Let's Encrypt rate limits

- [ ] Update CLAUDE.md:
  - [ ] Add ClusterIssuer configuration section
  - [ ] Document which issuer is used in each environment
  - [ ] Link to ClusterIssuer runbook

- [ ] Create certificate management guide:
  - [ ] Create `docs/procedures/certificate-management.md`
  - [ ] How to request new certificate (Ingress annotation vs Certificate resource)
  - [ ] How to check certificate status
  - [ ] How to troubleshoot issuance failures
  - [ ] How to rotate certificates manually (if needed)

- [ ] Document ClusterIssuer inventory:
  - [ ] Create comparison table in docs
  - [ ] List ClusterIssuers in each environment
  - [ ] Document last verified date
  - [ ] Include configuration differences (if any)

## Phase 9: Validation

- [ ] Run comprehensive validation:
  - [ ] Verify all ClusterIssuers show "Ready" status in all environments
  - [ ] Verify all existing certificates are "Ready"
  - [ ] Test issuance of new certificate in each environment
  - [ ] Validate DNS-01 challenge completes successfully
  - [ ] Check no rate limit errors in cert-manager logs

- [ ] Create validation checklist:
  - [ ] ClusterIssuers exist: ✓ letsencrypt-staging, ✓ letsencrypt-prod
  - [ ] Gandi credentials synced via Infisical
  - [ ] Webhook operational
  - [ ] Test certificates issued successfully
  - [ ] Documentation complete

- [ ] Sign off on verification:
  - [ ] Document verification date
  - [ ] Note any known issues or deviations
  - [ ] Schedule next verification (quarterly?)
