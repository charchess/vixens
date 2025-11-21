# Kubernetes Specification Delta

## ADDED Requirements

### Requirement: Mail Gateway TCP Routing

The system SHALL provide TCP-level routing for mail protocols (SMTP, IMAP) from Traefik ingress controller to Synology NAS mail server at 192.168.111.69.

#### Scenario: SMTP submission routed via Traefik
- **GIVEN** Synology mail server listens on 192.168.111.69:587
- **WHEN** external client connects to mail.dev.truxonline.com:587
- **THEN** Traefik SHALL route TCP connection to 192.168.111.69:587
- **AND** TLS handshake SHALL be performed directly with Synology (passthrough)

#### Scenario: IMAP SSL routed via Traefik
- **GIVEN** Synology mail server listens on 192.168.111.69:993
- **WHEN** mail client connects to mail.dev.truxonline.com:993
- **THEN** Traefik SHALL route TCP connection to 192.168.111.69:993
- **AND** TLS certificate SHALL be served by Synology mail server

#### Scenario: Service discovery via ExternalName
- **GIVEN** mail-gateway namespace exists
- **WHEN** Kubernetes Service is created with type ExternalName
- **THEN** Service SHALL resolve to 192.168.111.69 without creating Endpoints
- **AND** IngressRouteTCP SHALL reference this Service for routing

### Requirement: Multi-Port Mail Protocol Support

The system SHALL support multiple mail protocols simultaneously (SMTP port 25/587, IMAP port 143/993) through distinct IngressRouteTCP resources.

#### Scenario: SMTP and IMAP coexist without conflict
- **GIVEN** IngressRouteTCP for SMTP (ports 25, 587) exists
- **AND** IngressRouteTCP for IMAP (ports 143, 993) exists
- **WHEN** both resources are applied to cluster
- **THEN** Traefik SHALL accept connections on all 4 ports
- **AND** each port SHALL route to corresponding backend service port

#### Scenario: TLS passthrough for all protocols
- **GIVEN** IngressRouteTCP resources use `passthrough: true`
- **WHEN** client initiates TLS connection on any mail port
- **THEN** Traefik SHALL forward encrypted traffic without decryption
- **AND** Synology mail server SHALL handle TLS termination
- **AND** certificate validation SHALL occur between client and Synology

### Requirement: Environment-Specific DNS Configuration

The system SHALL support environment-specific DNS prefixes for mail gateway access (mail.dev, mail.test, mail.staging, mail subdomain).

#### Scenario: Dev environment uses mail.dev subdomain
- **GIVEN** dev environment Traefik LoadBalancer has IP 192.168.208.XX
- **WHEN** IngressRouteTCP specifies Host match `mail.dev.truxonline.com`
- **THEN** DNS A record SHALL point mail.dev.truxonline.com → 192.168.208.XX
- **AND** mail clients configured with mail.dev.truxonline.com SHALL connect successfully

#### Scenario: Production uses root mail subdomain
- **GIVEN** prod environment Traefik LoadBalancer has public IP
- **WHEN** IngressRouteTCP specifies Host match `mail.truxonline.com`
- **THEN** DNS A record SHALL point mail.truxonline.com → public IP
- **AND** MX record SHALL exist for @truxonline.com → mail.truxonline.com

### Requirement: Network Connectivity Between VLANs

The system SHALL enable connectivity from services VLAN (208) to internal VLAN (111) for mail gateway functionality.

#### Scenario: Traefik pod routes to internal NAS
- **GIVEN** Traefik pod runs on services VLAN 208
- **AND** Synology NAS listens on internal VLAN 111 (192.168.111.69)
- **WHEN** IngressRouteTCP forwards connection to ExternalName service
- **THEN** TCP connection SHALL traverse from VLAN 208 to VLAN 111
- **AND** routing tables SHALL permit this cross-VLAN traffic
- **AND** response traffic SHALL return successfully to client
