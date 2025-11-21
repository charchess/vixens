# Applications Specification - Home Assistant

## ADDED Requirements

### Requirement: Home Assistant SHALL be deployed via Kubernetes

Home Assistant SHALL be deployed as a Kubernetes workload managed by ArgoCD.

#### Scenario: Home Assistant deployed in dev environment
- **WHEN** deploying Home Assistant to dev cluster
- **THEN** it SHALL be deployed via ArgoCD Application
- **AND** it SHALL run in namespace `homeassistant`
- **AND** it SHALL use image `ghcr.io/home-assistant/home-assistant:stable`
- **AND** it SHALL be accessible at https://homeassistant.dev.truxonline.com

#### Scenario: Home Assistant deployed in prod environment
- **WHEN** deploying Home Assistant to prod cluster
- **THEN** it SHALL be deployed via ArgoCD Application
- **AND** it SHALL run in namespace `homeassistant`
- **AND** it SHALL use pinned image version (not :stable tag)
- **AND** it SHALL be accessible at https://homeassistant.truxonline.com

### Requirement: Home Assistant SHALL use persistent storage

Home Assistant configuration SHALL be stored on persistent volume provided by Synology CSI.

#### Scenario: Configuration persists across pod restarts
- **GIVEN** Home Assistant pod is running with configuration
- **WHEN** pod is deleted or restarted
- **THEN** new pod SHALL mount same PVC
- **AND** all configuration SHALL be preserved
- **AND** automations SHALL continue to work

#### Scenario: Storage class for dev environment
- **WHEN** Home Assistant PVC is created in dev
- **THEN** it SHALL use storageClassName `synology-iscsi-storage`
- **AND** it SHALL request 10Gi capacity
- **AND** it SHALL be mounted at `/config` in container

#### Scenario: Storage class for prod environment
- **WHEN** Home Assistant PVC is created in prod
- **THEN** it SHALL use storageClassName `synology-iscsi-storage`
- **AND** it SHALL request 50Gi capacity
- **AND** it SHALL be mounted at `/config` in container

### Requirement: Home Assistant SHALL be accessible via Traefik ingress

Home Assistant SHALL be exposed via Traefik ingress with TLS termination.

#### Scenario: Ingress with TLS in dev
- **WHEN** accessing Home Assistant in dev
- **THEN** ingress SHALL route traffic to Home Assistant service
- **AND** TLS certificate SHALL be issued by Let's Encrypt
- **AND** hostname SHALL be `homeassistant.dev.truxonline.com`
- **AND** ingress SHALL support WebSocket connections

#### Scenario: Ingress with TLS in prod
- **WHEN** accessing Home Assistant in prod
- **THEN** ingress SHALL route traffic to Home Assistant service
- **AND** TLS certificate SHALL be issued by Let's Encrypt (production)
- **AND** hostname SHALL be `homeassistant.truxonline.com`
- **AND** ingress SHALL support WebSocket connections

### Requirement: Home Assistant SHALL have health checks configured

Liveness and readiness probes SHALL be configured to detect unhealthy pods.

#### Scenario: Liveness probe detects unresponsive pod
- **GIVEN** Home Assistant pod is running
- **WHEN** Home Assistant process hangs or crashes
- **THEN** liveness probe SHALL fail after startup delay
- **AND** Kubernetes SHALL restart the pod
- **AND** probe SHALL check HTTP endpoint http://localhost:8123/

#### Scenario: Readiness probe prevents premature traffic
- **GIVEN** Home Assistant pod is starting
- **WHEN** startup is not complete
- **THEN** readiness probe SHALL fail
- **AND** pod SHALL NOT receive traffic from service
- **AND** probe SHALL succeed only when HTTP endpoint is responsive

### Requirement: Home Assistant SHALL have appropriate resource limits

CPU and memory limits SHALL be configured to prevent resource exhaustion.

#### Scenario: Resource limits in dev environment
- **WHEN** Home Assistant runs in dev cluster
- **THEN** it SHALL have CPU request 500m
- **AND** it SHALL have memory request 1Gi
- **AND** limits SHALL be set to prevent node exhaustion

#### Scenario: Resource limits in prod environment
- **WHEN** Home Assistant runs in prod cluster
- **THEN** it SHALL have CPU request 1000m
- **AND** it SHALL have memory request 2Gi
- **AND** limits SHALL ensure QoS (Quality of Service)

### Requirement: Home Assistant SHALL deploy in correct sync wave

Home Assistant SHALL deploy after infrastructure and platform services are ready.

#### Scenario: Home Assistant waits for dependencies
- **GIVEN** Infrastructure apps (Traefik, cert-manager) with wave 1-2
- **AND** Home Assistant with wave 4
- **WHEN** ArgoCD syncs all applications
- **THEN** Home Assistant SHALL deploy after Traefik is healthy
- **AND** Ingress SHALL be immediately functional
- **AND** TLS certificate SHALL be issued without errors
