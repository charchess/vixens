# Implementation Tasks

## 1. Architecture Research
- [ ] 1.1 Verify Traefik supports IngressRouteTCP for non-HTTP services
- [ ] 1.2 Identify exact ports used by Synology mail server (25, 587, 143, 993)
- [ ] 1.3 Confirm TLS passthrough vs Traefik TLS termination approach
- [ ] 1.4 Verify network connectivity from VLAN 208 to 192.168.111.69

## 2. Kubernetes Resources
- [ ] 2.1 Create `apps/mail-gateway/base/namespace.yaml`
- [ ] 2.2 Create `apps/mail-gateway/base/service.yaml` (ExternalName → 192.168.111.69)
- [ ] 2.3 Create `apps/mail-gateway/base/ingressroute-smtp.yaml` (TCP port 25/587)
- [ ] 2.4 Create `apps/mail-gateway/base/ingressroute-imap.yaml` (TCP port 143/993)
- [ ] 2.5 Create `apps/mail-gateway/base/kustomization.yaml`

## 3. Environment Overlays
- [ ] 3.1 Create `apps/mail-gateway/overlays/dev/kustomization.yaml`
- [ ] 3.2 Configure dev-specific DNS (mail.dev.truxonline.com)
- [ ] 3.3 Create ArgoCD Application `argocd/overlays/dev/apps/mail-gateway.yaml`

## 4. DNS Configuration
- [ ] 4.1 Create DNS A record: mail.dev.truxonline.com → 192.168.208.XX (Traefik LB IP)
- [ ] 4.2 Create MX record for mail delivery (if applicable)

## 5. Validation
- [ ] 5.1 Verify ArgoCD syncs mail-gateway successfully
- [ ] 5.2 Test SMTP connection via Traefik: `openssl s_client -connect mail.dev.truxonline.com:587 -starttls smtp`
- [ ] 5.3 Test IMAP connection via Traefik: `openssl s_client -connect mail.dev.truxonline.com:993`
- [ ] 5.4 Verify TLS certificates are from Synology (not Traefik Let's Encrypt)
- [ ] 5.5 Send test email through SMTP relay

## 6. Documentation
- [ ] 6.1 Document mail gateway architecture in CLAUDE.md
- [ ] 6.2 Add runbook for troubleshooting mail connectivity issues
