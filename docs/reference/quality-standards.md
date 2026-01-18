# Quality Standards (Goldification)

## Overview

This document defines the quality tiers for applications deployed in the Vixens infrastructure. Each tier represents a level of operational maturity and reliability.

## Quality Tiers

### Bronze (Basic Deployment)
**Status:** Initial deployment, minimal configuration

**Requirements:**
- ✅ Application deployed and accessible
- ✅ Basic resource requests defined
- ✅ Basic ingress/service configuration

**Not Required:**
- Resource limits
- Health checks
- Monitoring
- High availability

**Use Cases:**
- Testing new applications
- Non-critical services
- Development experiments

---

### Silver (Production-Ready)
**Status:** Suitable for non-critical production workloads

**Requirements (Bronze +):**
- ✅ Resource limits defined
- ✅ Basic readiness probe
- ✅ Persistent storage configured (if needed)
- ✅ TLS/HTTPS enabled

**Not Required:**
- Liveness probes
- Advanced monitoring
- High availability
- Backup strategy

**Use Cases:**
- Non-critical production services
- Internal tools
- Development/test environments

---

### Gold (Monitored Production)
**Status:** Production-grade with monitoring

**Requirements (Silver +):**
- ✅ Liveness probe configured
- ✅ Readiness probe configured
- ✅ Basic Prometheus metrics exposed
- ✅ ServiceMonitor configured
- ✅ Basic alerts defined
- ✅ Proper logging configuration

**Not Required:**
- Comprehensive dashboards
- Advanced alerts
- Backup validation
- SLO/SLI definitions

**Use Cases:**
- Standard production services
- Services requiring basic monitoring
- Most homelab applications

---

### Platinum (Highly Reliable)
**Status:** High reliability with comprehensive monitoring

**Requirements (Gold +):**
- ✅ Comprehensive Grafana dashboard
- ✅ Advanced alerting rules
- ✅ QoS class defined (Guaranteed/Burstable)
- ✅ Pod Disruption Budget (if HA)
- ✅ Backup strategy documented
- ✅ Resource recommendations from VPA/Goldilocks

**Not Required:**
- Automated backup validation
- Chaos engineering tests
- SLO compliance monitoring

**Use Cases:**
- Critical production services
- Services with uptime requirements
- Services with data persistence

---

### Elite (Mission-Critical)
**Status:** Maximum reliability and observability

**Requirements (Platinum +):**
- ✅ **Liveness probe** (mandatory for Elite)
- ✅ **Readiness probe** (mandatory for Elite)
- ✅ **Startup probe** (for slow-starting apps)
- ✅ Automated backup validation (if applicable)
- ✅ SLO/SLI definitions
- ✅ Chaos engineering tested
- ✅ Comprehensive runbooks
- ✅ Security scanning results documented

**Not Required:**
- Nothing - this is the highest tier

**Use Cases:**
- Infrastructure components (ArgoCD, Traefik, Cilium)
- Critical data services (databases)
- Core homelab services

---

### Diamond (Future/Reserved)
**Status:** Reserved for future enhancements

**Potential Requirements:**
- Multi-cluster deployment
- Advanced DR strategy
- Automated canary deployments
- Full observability stack

---

## Health Check Requirements

### Liveness Probe
**Purpose:** Determines if the container needs to be restarted

**When Required:**
- **Elite tier:** Mandatory
- Platinum tier: Highly recommended
- Gold tier and below: Optional

**Best Practices:**
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe
**Purpose:** Determines if the container is ready to serve traffic

**When Required:**
- **Elite tier:** Mandatory
- Platinum tier: Mandatory
- Gold tier: Mandatory
- Silver tier: Recommended
- Bronze tier: Optional

**Best Practices:**
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: http
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

### Startup Probe
**Purpose:** Protects slow-starting containers from being killed

**When Required:**
- Elite tier: For slow-starting apps (>30s)
- All tiers: When initialDelaySeconds would be >60s

**Best Practices:**
```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 0
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 30  # 300s total
```

---

## Goldification Process

1. **Audit Current State:** Review application configuration
2. **Identify Target Tier:** Based on criticality and requirements
3. **Implement Missing Requirements:** Work through tier requirements
4. **Validate:** Test health checks, monitoring, backups
5. **Document:** Update application documentation with tier status
6. **Track:** Create Beads task for goldification work

---

## Documentation Standards (ADR)

Toutes les décisions architecturales (ADR) doivent suivre le format standard Vixens pour garantir la traçabilité et l'interopérabilité entre les agents.

### ADR Header Template
```markdown
# ADR-XXX: [TITRE]

**Date:** YYYY-MM-DD
**Status:** [Accepted | Deprecated | Superseded by [ADR-YYY](Lien.md)]
**Deciders:** [Deciders List]
**Tags:** [tags]

---
```

---

## Related Documentation

- [ADR Template](../templates/adr-template.md)
- [Application Template](../templates/application-template.md)
- [Adding New Applications](../guides/adding-new-application.md)
- [Monitoring Setup](../guides/monitoring-setup.md)
- [Backup Strategy](../guides/backup-strategy.md)

---

**Last Updated:** 2026-01-17