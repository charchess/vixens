# Kubernetes Specification - HTTPS Redirect

## ADDED Requirements

### Requirement: All Ingress Resources SHALL Redirect HTTP to HTTPS

Every Ingress resource exposing an application SHALL automatically redirect HTTP requests to HTTPS using Traefik middleware.

#### Scenario: HTTP request redirected to HTTPS
**GIVEN** an Ingress resource with TLS configured for `homeassistant.dev.truxonline.com`
**WHEN** a client sends HTTP request to `http://homeassistant.dev.truxonline.com`
**THEN** Traefik SHALL respond with HTTP 301 Moved Permanently
**AND** Location header SHALL be `https://homeassistant.dev.truxonline.com`
**AND** client SHALL be redirected to HTTPS automatically

#### Scenario: HTTPS request processed normally
**GIVEN** an Ingress resource with HTTPS redirect enabled
**WHEN** a client sends HTTPS request to `https://homeassistant.dev.truxonline.com`
**THEN** request SHALL be processed normally without redirect
**AND** application SHALL receive the request
**AND** response SHALL be HTTP 200 OK (or appropriate status)

### Requirement: Traefik Middleware SHALL Provide Centralized Redirect Logic

A Traefik Middleware CRD SHALL exist in the traefik namespace to handle HTTP → HTTPS redirection for all Ingress resources.

#### Scenario: Middleware deployed in traefik namespace
**GIVEN** Traefik is operational in the cluster
**WHEN** deploying the redirect middleware
**THEN** Middleware CRD `redirect-https` SHALL exist in namespace `traefik`
**AND** middleware SHALL specify `redirectScheme.scheme: https`
**AND** middleware SHALL specify `redirectScheme.permanent: true` (HTTP 301)

#### Scenario: Ingress references middleware via annotation
**GIVEN** Middleware `redirect-https` exists in namespace `traefik`
**WHEN** Ingress resource includes annotation `traefik.ingress.kubernetes.io/router.middlewares: traefik-redirect-https@kubernetescrd`
**THEN** Traefik SHALL apply the middleware to that ingress route
**AND** HTTP requests SHALL be redirected to HTTPS
**AND** annotation SHALL reference middleware with format `<namespace>-<name>@kubernetescrd`

### Requirement: Ingress Annotation SHALL Be Consistent Across Applications

All Ingress resources SHALL use the same annotation pattern to enable HTTPS redirect, ensuring consistency and maintainability.

#### Scenario: Standard annotation on all ingresses
**GIVEN** applications homeassistant, mail-gateway, traefik-dashboard, argocd, whoami
**WHEN** examining their Ingress resources in base/
**THEN** all SHALL have annotation `traefik.ingress.kubernetes.io/router.middlewares: traefik-redirect-https@kubernetescrd`
**AND** annotation SHALL be identical across all applications
**AND** no application SHALL use separate redirect Ingress resources

#### Scenario: New application follows redirect pattern
**GIVEN** a new application being added to the cluster
**WHEN** creating its Ingress resource
**THEN** it SHALL include the standard redirect annotation
**AND** it SHALL NOT create separate HTTP redirect Ingress
**AND** pattern SHALL be documented in CLAUDE.md for consistency
