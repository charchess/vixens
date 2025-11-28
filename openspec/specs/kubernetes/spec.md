# Kubernetes Specification - Vixens

## Purpose
Management des clusters Kubernetes via ArgoCD en pattern GitOps. Cette spec définit le déploiement des applications, la configuration réseau (Cilium), la gestion des secrets (Infisical), et les politiques de sécurité actuellement en vigueur sur tous les environnements. Projet homelab personnel en phase 2/4.

## Requirements

### Requirement: Cilium SHALL Use Correct IP Pool
Cilium SHALL use the IP pool defined in the infrastructure specification.

#### Scenario: Cilium config dev
- **WHEN** Cilium is deployed on dev
- **THEN** it SHALL use service range 192.168.208.0/24
- **AND** it SHALL allocate LB IPs according to infrastructure spec

#### Scenario: Cilium config test
- **WHEN** Cilium is deployed on test
- **THEN** it SHALL use service range 192.168.209.0/24
- **AND** it SHALL allocate LB IPs according to infrastructure spec

#### Scenario: Cilium config staging
- **WHEN** Cilium is deployed on staging
- **THEN** it SHALL use service range 192.168.210.0/24
- **AND** it SHALL allocate LB IPs according to infrastructure spec

#### Scenario: Cilium config prod
- **WHEN** Cilium is deployed on prod
- **THEN** it SHALL use service range 192.168.201.0/24
- **AND** it SHALL allocate LB IPs according to infrastructure spec

### Requirement: Applications SHALL be Deployed via ArgoCD
All applications SHALL use Multiple Sources Pattern avec GitOps.

#### Scenario: Whoami deployment dev
- **WHEN** whoami is deployed on dev
- **THEN** it SHALL use chart Helm + values from Git
- **AND** it SHALL be accessible at whoami.dev.truxonline.com
- **AND** it SHALL use insecure mode (no auth)

#### Scenario: HomeAssistant deployment
- **WHEN** HomeAssistant is deployed
- **THEN** it SHALL use ingress route homeassistant.{env}.truxonline.com
- **AND** it SHALL target local HomeAssistant instance (ex: 192.168.1.100)
- **AND** it SHALL use insecure mode

#### Scenario: Mail-gateway deployment
- **WHEN** mail-gateway is deployed
- **THEN** it SHALL use ingress route mail.{env}.truxonline.com
- **AND** it SHALL target local mail server (ex: 192.168.1.101)
- **AND** it SHALL use insecure mode

#### Scenario: ArgoCD self-management
- **WHEN** ArgoCD is deployed
- **THEN** it SHALL manage its own configuration via Git
- **AND** it SHALL be the only manual deployment allowed

### Requirement: Secrets Management SHALL use Infisical
Sensitive data SHALL be managed via Infisical Operator.

#### Scenario: CSI Synology credentials dev
- **WHEN** CSI Synology starts on dev
- **THEN** it SHALL retrieve credentials from Infisical path `/vixens/dev/synology-csi`

#### Scenario: CSI Synology credentials test
- **WHEN** CSI Synology starts on test
- **THEN** it SHALL retrieve credentials from Infisical path `/vixens/test/synology-csi`

#### Scenario: CSI Synology credentials staging
- **WHEN** CSI Synology starts on staging
- **THEN** it SHALL retrieve credentials from Infisical path `/vixens/stg/synology-csi`

#### Scenario: CSI Synology credentials prod
- **WHEN** CSI Synology starts on prod
- **THEN** it SHALL retrieve credentials from Infisical path `/vixens/prod/synology-csi`

#### Scenario: ArgoCD secrets
- **WHEN** ArgoCD needs secrets
- **THEN** it SHALL retrieve them from Infisical path `/vixens/{env}/argocd`

#### Scenario: Gandi API key
- **WHEN** cert-manager needs DNS challenge
- **THEN** it SHALL retrieve Gandi API key from Infisical path `/vixens/{env}/cert-manager/gandi`

### Requirement: TLS Certificates SHALL be Managed by cert-manager
All public endpoints SHALL have automatic TLS certificates via Gandi webhook.

#### Scenario: Certificate issuer configuration
- **GIVEN** cert-manager is deployed
- **WHEN** a Certificate resource is created
- **THEN** it SHALL use Gandi webhook for DNS-01 challenge
- **AND** certificates SHALL be stored as Kubernetes secrets
- **AND** ACME email shall be admin@truxonline.com

### Requirement: Persistent Storage SHALL use Synology CSI
Dynamic provisioning SHALL be provided by Synology CSI driver avec iSCSI.

#### Scenario: Storage class configuration
- **WHEN** an application needs persistent storage
- **THEN** it SHALL use StorageClass "synology-csi-{env}"
- **AND** it SHALL create iSCSI volumes on Synology DS1821+
- **AND** it SHALL NOT use NFS (déprécié)

#### Scenario: NFS storage legacy
- **WHEN** legacy NFS mount is needed
- **THEN** it SHALL use StorageClass "nfs-storage"
- **AND** it SHALL be marked as deprecated in values

### Requirement: Ingress SHALL be Managed by Traefik
All HTTP/HTTPS traffic SHALL pass through Traefik.

#### Scenario: Traefik dashboard access
- **WHEN** Traefik dashboard is needed
- **THEN** it SHALL be accessible at traefik-dashboard.{env}.truxonline.com
- **AND** it SHALL be in insecure mode (no auth)

#### Scenario: Insecure mode all dashboards
- **WHEN** any dashboard is deployed (ArgoCD, Traefik, etc.)
- **THEN** it SHALL use insecure mode (no authentication)
- **AND** it SHALL be protected by NetworkPolicy until Authentik is deployed

### Requirement: Storage Configuration Errors SHALL be Fixed
La redondance synology-csi-talos SHALL be removed.

#### Scenario: Cleanup synology-csi-talos
- **WHEN** Phase 2 is completed
- **THEN** synology-csi-talos application SHALL be deleted
- **AND** only synology-csi SHALL remain

### Requirement: Git Workflow SHALL be Enforced
All changes SHALL follow strict PR workflow avec validation.

#### Scenario: PR validation
- **WHEN** a PR is created from dev to test
- **THEN** yamllint SHALL validate all YAML files
- **AND** OpenSpec validator SHALL validate spec changes
- **AND** merge SHALL be blocked if validation fails

#### Scenario: Branch progression
- **WHEN** dev is validated
- **THEN** PR SHALL be created to test
- **AND** test validation triggers PR to staging
- **AND** staging validation triggers PR to prod

### Requirement: No Monitoring Stack SHALL be Deployed Yet
Monitoring SHALL be part of Phase 3.

#### Scenario: Phase 2 scope
- **WHEN** Phase 2 is active
- **THEN** Prometheus, Grafana, Loki SHALL NOT be deployed
- **AND** only logging to stdout SHALL be used

### Requirement: No Security Scanning SHALL be Deployed Yet
Security scanning SHALL be part of Phase 3.

#### Scenario: Phase 2 security
- **WHEN** Phase 2 is active
- **THEN** Trivy, Falco, Kyverno SHALL NOT be deployed
- **AND** NetworkPolicies SHALL be permissive

### Requirement: No Policy as Code SHALL be Deployed Yet
OPA Gatekeeper SHALL NOT be deployed in Phase 2.

#### Scenario: Policy as code not deployed
- **WHEN** checking namespaces in Phase 2
- **THEN** gatekeeper-system SHALL NOT exist
- **AND** no Policy or ConstraintTemplate resources SHALL be found

### Requirement: Kustomize Overlays SHALL be Used
Environment differences SHALL be managed via Kustomize overlays.

#### Scenario: Base and overlays structure
- **WHEN** deploying to dev
- **THEN** it SHALL use `kustomize/overlays/{env}/kustomization.yaml`
- **AND** it SHALL reference `../../base/`
- **AND** changes SHALL be minimal and environment-specific

### Requirement: Applications SHALL Use Kustomize Base/Overlays Pattern
Chaque application SHALL avoir un dossier `base/` pour la config commune et des `overlays/{env}/` pour les différences.

#### Scenario: Base configuration
- **WHEN** creating an application
- **THEN** it SHALL have a `apps/{app}/base/` directory
- **AND** it SHALL contain resources communs (deployments, services, PVCs)

#### Scenario: Overlay structure
- **WHEN** deploying to an environment
- **THEN** it SHALL use `apps/{app}/overlays/{env}/kustomization.yaml`
- **AND** it SHALL reference `../../base/` in resources
- **AND** it SHALL contain patches pour valeurs spécifiques (IPs, noms, etc.)

#### Scenario: Environment-specific patches
- **WHEN** patching for dev
- **THEN** overlay SHALL patch l'IP de l'ingress pour `dev.truxonline.com`
- **AND** it SHALL patch le StorageClass pour `synology-csi-dev`

### Requirement: Terraform Backend SHALL be MinIO on NAS
State SHALL be stored in S3-compatible backend on Synology.

#### Scenario: MinIO backend configuration
- **WHEN** terraform init runs
- **THEN** it SHALL connect to nas.truxonline.com:9000
- **AND** it SHALL use bucket "terraform-state-vixens"
- **AND** it SHALL enable state locking via MinIO

### Requirement: Critical Applications SHALL have Pod Anti-Affinity
Applications requiring HA MUST avoid single node failure.

#### Scenario: HomeAssistant anti-affinity
- **WHEN** HomeAssistant is deployed
- **THEN** it SHALL have `podAntiAffinity` with `requiredDuringSchedulingIgnoredDuringExecution`
- **AND** it SHALL have `topologyKey: kubernetes.io/hostname`
- **AND** it SHALL have `replicas: 2` minimum

#### Scenario: Mail-gateway anti-affinity
- **WHEN** mail-gateway is deployed
- **THEN** it SHALL have `podAntiAffinity`
- **AND** it SHALL have `PodDisruptionBudget` with `minAvailable: 1`

### Requirement: Cilium DaemonSet SHALL Tolerate All Nodes
Cilium MUST run on every node, including control plane nodes.

#### Scenario: Control plane toleration
- **WHEN** Cilium is deployed
- **THEN** it SHALL have toleration for `node-role.kubernetes.io/control-plane:NoSchedule`
- **AND** it SHALL have `priorityClassName: system-node-critical`
- **AND** it SHALL run in `hostNetwork: true`

### Requirement: Terraform SHALL Wait for Kubernetes API
Terraform SHALL validate cluster readiness before deploying apps.

#### Scenario: API health check
- **WHEN** Talos cluster is created
- **THEN** Terraform SHALL wait 90s initial delay
- **AND** it SHALL perform 60 attempts every 10s for /healthz endpoint

#### Scenario: Control plane stability
- **WHEN** API is responding
- **THEN** Terraform SHALL wait for 3 consecutive successful checks
- **AND** it SHALL validate kube-apiserver, controller-manager, scheduler pods

## Reference Data

### Requirement: Critical Applications SHALL be HA-Aware
Les applications critiques DOIVENT être tolérantes aux pannes de nœuds avec anti-affinity.

#### Scenario: HomeAssistant anti-affinity
- **WHEN** HomeAssistant est déployé
- **THEN** il SHALL avoir `podAntiAffinity` pour ne pas être sur le même nœud que Traefik
- **AND** il SHALL avoir `replicas: 2` minimum

#### Scenario: Mail-gateway anti-affinity
- **WHEN** mail-gateway est déployé
- **THEN** il SHALL avoir `podAntiAffinity` pour répartition sur zones de disponibilité
- **AND** il SHALL avoir `PodDisruptionBudget` avec `minAvailable: 1`

#### Scenario: ArgoCD HA
- **WHEN** ArgoCD est déployé
- **THEN** il SHALL avoir `replicas: 3` pour redis et server
- **AND** il SHALL tolérer la perte d'un nœud sans downtime

#### Scenario: Cilium tolerations
- **WHEN** Cilium est déployé
- **THEN** il SHALL avoir `tolerations` pour tourner sur tous les nœuds (même masters)
- **AND** il SHALL avoir `priorityClassName: system-node-critical`

### Kubernetes Versions

| Environment | Kubernetes Version | Talos Version | Cilium Version | Status |
|-------------|-------------------|---------------|----------------|--------|
| **dev** | v1.31.4 | v1.7.6 | 1.16.5 | Active |
| **test** | v1.31.4 | v1.7.6 | 1.16.5 | Active with issues |
| **staging** | v1.31.4 | v1.7.6 | 1.16.5 | Not deployed |
| **prod** | v1.31.4 | v1.7.6 | 1.16.5 | Not deployed |

### Applications Deployed by ArgoCD

| Application | Namespace | Chart Version | Source | Purpose | Phase | Environments |
|-------------|-----------|---------------|--------|---------|-------|--------------|
| **cilium** | kube-system | 1.16.5 | Helm repo | CNI | 1 | all |
| **argocd** | argocd | 7.6.12 | Helm repo | GitOps | 1 | all |
| **traefik** | traefik | 33.0.0 | Helm repo | Ingress | 2 | all |
| **traefik-dashboard** | traefik | 33.0.0 | Helm repo | Ingress UI | 2 | all |
| **cert-manager** | cert-manager | 1.16.1 | Helm repo | TLS automation | 2 | all |
| **cert-manager-webhook-gandi** | cert-manager | 0.8.0 | Helm repo | DNS challenge | 2 | all |
| **infisical-operator** | infisical | 0.8.1 | Helm repo | Secrets mgmt | 2 | all |
| **synology-csi** | synology-csi | 0.9.1 | Helm repo | iSCSI provisioner | 2 | all |
| **synology-csi-talos** | synology-csi | 0.9.1 | Helm repo | REDUNDANT - TO REMOVE | 2 | all |
| **nfs-storage** | nfs | 4.0.18 | Helm repo | NFS provisioner (legacy) | 2 | all |
| **homeassistant** | apps | 0.2.5 | Git repo | Local HA ingress | 2 | all |
| **mail-gateway** | apps | 0.1.0 | Git repo | Local mail ingress | 2 | all |
| **whoami** | apps | 4.0.0 | Helm repo | Test app | 1 | dev, test |

### How to Get Chart Versions

```bash
# Method 1: Check deployed versions via Helm
helm list -A --output yaml

# Method 2: Check via kubectl
kubectl get deployments -A -o yaml | grep 'chart='

# Method 3: Check ArgoCD UI
# Applications → Chart column

# Method 4: Check Git repo values files
# argocd/ applications YAML

# Method 5: Extract from running pods
kubectl get pods -n traefik -o yaml | grep 'image:' | head -1

# Method 6: Use kubectl describe
kubectl describe pod -n argocd argocd-server-xxx | grep 'Image:'