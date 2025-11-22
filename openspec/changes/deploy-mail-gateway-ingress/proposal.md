# Deploy Mail Gateway Ingress

## Why

The Synology NAS at 192.168.111.69 runs a mail server (Postfix/Dovecot via Docker) that needs to be accessible from external clients. Currently, mail services are only accessible via direct IP from VLAN 111 (internal network).

To enable proper mail delivery and access from internet, we need Traefik IngressRoute configurations that expose SMTP (25/587), IMAP (143/993), and optionally webmail interface through the services VLAN (208) with proper TLS termination.

## What Changes

- **ADDED**: Kubernetes Service (ExternalName) pointing to 192.168.111.69
- **ADDED**: Traefik IngressRouteTCP for SMTP (port 25/587) with TLS passthrough
- **ADDED**: Traefik IngressRouteTCP for IMAP (port 143/993) with TLS passthrough
- **ADDED**: Optional HTTP IngressRoute for webmail (if applicable)
- **CONFIGURED**: DNS records for mail.truxonline.com pointing to Traefik LoadBalancer IP

## Impact

- **Affected specs**: kubernetes (new mail-gateway capability)
- **Affected code**:
  - Create `apps/mail-gateway/base/` with Service + IngressRouteTCP
  - Create `apps/mail-gateway/overlays/dev/` with environment-specific configs
  - Add ArgoCD Application in `argocd/overlays/dev/apps/mail-gateway.yaml`
- **Dependencies**: Traefik must support TCP routing (IngressRouteTCP CRD)
- **Security**: TLS passthrough to avoid certificate management in Traefik
- **Network**: Mail server must be accessible from services VLAN (208) to internal VLAN (111)

## Validation

- SMTP connection test: `openssl s_client -connect mail.dev.truxonline.com:587 -starttls smtp`
- IMAP connection test: `openssl s_client -connect mail.dev.truxonline.com:993`
- Verify TLS certificates are served by Synology mail server (not Traefik)
- Test actual mail delivery (send test email through SMTP)
