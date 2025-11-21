# Networking Specification - DNS Redirections

## ADDED Requirements

### Requirement: DNS records SHALL follow consistent hostname pattern

All services SHALL use the pattern `{service}.{env}.truxonline.com` for non-production environments, and `{service}.truxonline.com` for production.

#### Scenario: Service hostname in dev environment
- **GIVEN** A service named "homeassistant"
- **WHEN** Deployed in dev environment
- **THEN** hostname SHALL be `homeassistant.dev.truxonline.com`
- **AND** DNS record SHALL resolve to Traefik LoadBalancer IP 192.168.208.70

#### Scenario: Service hostname in prod environment
- **GIVEN** A service named "homeassistant"
- **WHEN** Deployed in prod environment
- **THEN** hostname SHALL be `homeassistant.truxonline.com`
- **AND** DNS record SHALL resolve to Traefik LoadBalancer IP 192.168.200.70

#### Scenario: Multiple services in same environment
- **GIVEN** Services "mail", "homeassistant", "traefik" in dev
- **WHEN** DNS records are created
- **THEN** hostnames SHALL be:
  - `mail.dev.truxonline.com`
  - `homeassistant.dev.truxonline.com`
  - `traefik.dev.truxonline.com`
- **AND** all SHALL resolve to same Traefik LoadBalancer IP

### Requirement: DNS records SHALL point to Traefik LoadBalancer

All service DNS records SHALL resolve to the Traefik LoadBalancer IP for that environment.

#### Scenario: DNS resolution for dev services
- **WHEN** Querying DNS for any service in dev (e.g., `mail.dev.truxonline.com`)
- **THEN** DNS SHALL return A record with value 192.168.208.70
- **AND** this SHALL be the Traefik LoadBalancer IP in VLAN 208

#### Scenario: DNS resolution for test services
- **WHEN** Querying DNS for any service in test (e.g., `homeassistant.test.truxonline.com`)
- **THEN** DNS SHALL return A record with value 192.168.209.70
- **AND** this SHALL be the Traefik LoadBalancer IP in VLAN 209

#### Scenario: DNS resolution for prod services
- **WHEN** Querying DNS for any service in prod (e.g., `mail.truxonline.com`)
- **THEN** DNS SHALL return A record with value 192.168.200.70
- **AND** this SHALL be the Traefik LoadBalancer IP in VLAN 200

### Requirement: DNS TTL SHALL be short for homelab flexibility

DNS records SHALL use short TTL (Time To Live) to enable rapid changes during development and troubleshooting.

#### Scenario: DNS record TTL for all environments
- **WHEN** Creating or updating DNS records
- **THEN** TTL SHALL be set to 300 seconds (5 minutes)
- **AND** this SHALL allow rapid DNS updates during troubleshooting
- **AND** propagation SHALL complete within 5 minutes

### Requirement: Wildcard DNS SHALL be supported for environment-wide resolution

Wildcard DNS records SHALL be used to automatically resolve all subdomains for an environment without explicit A records.

#### Scenario: Wildcard DNS for dev environment
- **WHEN** Wildcard record `*.dev.truxonline.com` exists
- **THEN** any subdomain SHALL resolve to 192.168.208.70
- **AND** this includes `mail.dev`, `homeassistant.dev`, `newservice.dev`, etc.
- **AND** explicit A records SHALL take precedence over wildcard

#### Scenario: New service benefits from wildcard DNS
- **GIVEN** Wildcard record `*.dev.truxonline.com` → 192.168.208.70
- **WHEN** Deploying new service "authentik" without creating DNS record
- **THEN** `authentik.dev.truxonline.com` SHALL automatically resolve to 192.168.208.70
- **AND** service SHALL be immediately accessible after Ingress creation

### Requirement: DNS verification SHALL be performed before service deployment

DNS resolution SHALL be verified before deploying services that depend on external access.

#### Scenario: Verify DNS before Home Assistant deployment
- **GIVEN** Home Assistant will be deployed with hostname `homeassistant.dev.truxonline.com`
- **WHEN** Preparing for deployment
- **THEN** DNS query SHALL confirm hostname resolves
- **AND** resolved IP SHALL match Traefik LoadBalancer IP
- **AND** deployment SHALL proceed only after DNS verification

#### Scenario: Verify DNS propagation completed
- **GIVEN** DNS record was just created or updated
- **WHEN** Verifying propagation
- **THEN** queries from multiple locations SHALL return consistent results
- **AND** queries SHALL wait at least one TTL period (300s) before testing
- **AND** both public DNS (8.8.8.8) and local resolvers SHALL return correct value

### Requirement: DNS configuration SHALL be documented

All DNS records and management procedures SHALL be documented in runbooks.

#### Scenario: DNS runbook documents procedures
- **WHEN** Need to add new service DNS record
- **THEN** runbook SHALL provide step-by-step procedure
- **AND** runbook SHALL document Gandi LiveDNS access
- **AND** runbook SHALL include troubleshooting for common DNS issues

#### Scenario: DNS inventory tracks all records
- **WHEN** Auditing DNS configuration
- **THEN** DNS inventory document SHALL list all records
- **AND** inventory SHALL include: hostname, record type, value, TTL, last verified date
- **AND** inventory SHALL distinguish vixens records from external records (MX, SPF, etc.)
