# Lint & Quality Report

**Generated:** 2026-02-08 13:40:16
**Quality Score:** 0/100

**Status:** 🔴 Needs Improvement

---

## Summary

| Category            | Count | Status |
| ------------------- | ----- | ------ |
| Total YAML Files    | 1315  | ℹ️     |
| Files Passed        | 1268  | ✅     |
| Files Failed        | 47    | ❌     |
| Yamllint Errors     | 50    | ❌     |
| Yamllint Warnings   | 62    | ⚠️     |
| DRY Violations      | 44    | ❌     |
| Resource Violations | 51    | ❌     |

---

## Yamllint Errors

| File                                                                                         | Line | Message                                                            |
| -------------------------------------------------------------------------------------------- | ---- | ------------------------------------------------------------------ |
| apps/04-databases/postgresql-shared/base/cluster.yaml                                        | 104  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 102  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/04-databases/cloudnative-pg/base/charts/cloudnative-pg-0.27.0/cloudnative-pg/Chart.yaml | 4    | wrong indentation: expected 2 but found 0 (indentation)            |
| apps/04-databases/cloudnative-pg/base/charts/cloudnative-pg-0.27.0/cloudnative-pg/Chart.yaml | 13   | wrong indentation: expected 2 but found 0 (indentation)            |
| apps/04-databases/cloudnative-pg/base/charts/cloudnative-pg-0.27.0/cloudnative-pg/Chart.yaml | 19   | wrong indentation: expected 2 but found 0 (indentation)            |
| apps/04-databases/cloudnative-pg/base/charts/cloudnative-pg-0.27.0/cloudnative-pg/Chart.yaml | 23   | wrong indentation: expected 2 but found 0 (indentation)            |
| apps/10-home/homeassistant/overlays/prod/kustomization.yaml                                  | 32   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/01-storage/synology-csi/base/kustomization.yaml                                         | 20   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/vpa/overlays/prod/resources-patch.yaml                                         | 60   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/vpa/overlays/dev/resources-patch.yaml                                          | 60   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/kyverno/base/policies/require-litestream-monitoring.yaml                       | 41   | too many blank lines (1 > 0) (empty-lines)                         |
| apps/00-infra/kyverno/base/policies/add-default-labels.yaml                                  | 39   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/kyverno/base/policies/require-goldilocks.yaml                                  | 39   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/reloader/base/kustomization.yaml                                               | 31   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/traefik/values/prod.yaml                                                       | 46   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/00-infra/velero/overlays/prod/kustomization.yaml                                        | 9    | no new line character at the end of file (new-line-at-end-of-file) |
| apps/template-app/base/deployment.yaml                                                       | 198  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 204  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/base/dashboard.yaml                                                  | 59   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/base/management.yaml                                                 | 167  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/base/infra.yaml                                                      | 34   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/overlays/prod/patches.yaml                                           | 91   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/overlays/prod/ingress.yaml                                           | 106  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/overlays/dev/patches.yaml                                            | 61   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/netbird/overlays/dev/ingress.yaml                                            | 106  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/40-network/mail-gateway/base/kustomization.yaml                                         | 17   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/02-monitoring/prometheus/overlays/dev/kustomization.yaml                                | 13   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/02-monitoring/grafana/base/values.yaml                                                  | 89   | too many blank lines (1 > 0) (empty-lines)                         |
| apps/02-monitoring/grafana/overlays/prod/resources-patch.yaml                                | 26   | too many blank lines (1 > 0) (empty-lines)                         |
| apps/02-monitoring/loki/overlays/prod/kustomization.yaml                                     | 13   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/02-monitoring/loki/overlays/dev/kustomization.yaml                                      | 14   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/02-monitoring/promtail/overlays/prod/resources-patch.yaml                               | 17   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 89   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/60-services/firefly-iii-importer/overlays/dev/kustomization.yaml                        | 32   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 180  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/60-services/vaultwarden/overlays/prod/kustomization.yaml                                | 16   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/60-services/firefly-iii/base/ingress.yaml                                               | 28   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 176  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/70-tools/renovate/base/cronjob.yaml                                                     | 60   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/70-tools/stirling-pdf/overlays/prod/kustomization.yaml                                  | 14   | too many blank lines (1 > 0) (empty-lines)                         |
| apps/70-tools/radar/overlays/prod/kustomization.yaml                                         | 6    | no new line character at the end of file (new-line-at-end-of-file) |
| apps/70-tools/radar/overlays/dev/kustomization.yaml                                          | 6    | no new line character at the end of file (new-line-at-end-of-file) |
| apps/03-security/trivy/overlays/prod/resources-patch.yaml                                    | 25   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/03-security/trivy/overlays/dev/resources-patch.yaml                                     | 25   | no new line character at the end of file (new-line-at-end-of-file) |
| apps/20-media/hydrus-client/base/deployment.yaml                                             | 266  | no new line character at the end of file (new-line-at-end-of-file) |
| apps/20-media/sabnzbd/base/litestream-config.yaml                                            | 17   | no new line character at the end of file (new-line-at-end-of-file) |
| argocd/overlays/dev/kustomization.yaml                                                       | 97   | too many blank lines (1 > 0) (empty-lines)                         |
| argocd/overlays/dev/apps/policy-reporter.yaml                                                | 103  | too many blank lines (1 > 0) (empty-lines)                         |
| argocd/overlays/prod/apps/policy-reporter.yaml                                               | 103  | too many blank lines (1 > 0) (empty-lines)                         |
| argocd/overlays/prod/apps/nocodb.yaml                                                        | 22   | no new line character at the end of file (new-line-at-end-of-file) |

---

## Yamllint Warnings

| File                                                                                         | Line | Message                                           |
| -------------------------------------------------------------------------------------------- | ---- | ------------------------------------------------- |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 85   | line too long (142 > 80 characters) (line-length) |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 87   | line too long (118 > 80 characters) (line-length) |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 89   | trailing spaces (trailing-spaces)                 |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 90   | line too long (88 > 80 characters) (line-length)  |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 91   | trailing spaces (trailing-spaces)                 |
| apps/04-databases/postgresql-shared/base/jobs/create-users-job.yaml                          | 94   | trailing spaces (trailing-spaces)                 |
| apps/04-databases/cloudnative-pg/base/charts/cloudnative-pg-0.27.0/cloudnative-pg/Chart.yaml | 1    | missing document start "---" (document-start)     |
| apps/00-infra/kyverno/base/policies/require-litestream-monitoring.yaml                       | 14   | line too long (81 > 80 characters) (line-length)  |
| apps/00-infra/kyverno/base/policies/require-litestream-monitoring.yaml                       | 29   | line too long (113 > 80 characters) (line-length) |
| apps/00-infra/kyverno/base/policies/require-litestream-monitoring.yaml                       | 33   | line too long (126 > 80 characters) (line-length) |
| apps/00-infra/velero/overlays/prod/kustomization.yaml                                        | 1    | missing document start "---" (document-start)     |
| apps/template-app/base/deployment.yaml                                                       | 25   | line too long (83 > 80 characters) (line-length)  |
| apps/template-app/base/deployment.yaml                                                       | 45   | line too long (85 > 80 characters) (line-length)  |
| apps/template-app/base/deployment.yaml                                                       | 163  | line too long (85 > 80 characters) (line-length)  |
| apps/template-app/base/deployment.yaml                                                       | 165  | trailing spaces (trailing-spaces)                 |
| apps/template-app/base/deployment.yaml                                                       | 170  | trailing spaces (trailing-spaces)                 |
| apps/template-app/base/deployment.yaml                                                       | 172  | trailing spaces (trailing-spaces)                 |
| apps/template-app/base/deployment.yaml                                                       | 173  | trailing spaces (trailing-spaces)                 |
| apps/template-app/base/deployment.yaml                                                       | 174  | trailing spaces (trailing-spaces)                 |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 50   | line too long (85 > 80 characters) (line-length)  |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 158  | line too long (85 > 80 characters) (line-length)  |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 160  | trailing spaces (trailing-spaces)                 |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 165  | trailing spaces (trailing-spaces)                 |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 167  | trailing spaces (trailing-spaces)                 |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 168  | trailing spaces (trailing-spaces)                 |
| apps/40-network/adguard-home/base/adguard-ss.yaml                                            | 169  | trailing spaces (trailing-spaces)                 |
| apps/40-network/netbird/base/management.yaml                                                 | 103  | line too long (112 > 80 characters) (line-length) |
| apps/40-network/netbird/base/management.yaml                                                 | 130  | line too long (163 > 80 characters) (line-length) |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 14   | missing document start "---" (document-start)     |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 41   | line too long (83 > 80 characters) (line-length)  |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 42   | line too long (91 > 80 characters) (line-length)  |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 70   | line too long (83 > 80 characters) (line-length)  |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 71   | line too long (91 > 80 characters) (line-length)  |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 73   | trailing spaces (trailing-spaces)                 |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 78   | trailing spaces (trailing-spaces)                 |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 80   | trailing spaces (trailing-spaces)                 |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 81   | trailing spaces (trailing-spaces)                 |
| apps/_shared/components/elite-syncer/kustomization.yaml                                      | 82   | trailing spaces (trailing-spaces)                 |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 33   | line too long (111 > 80 characters) (line-length) |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 50   | line too long (85 > 80 characters) (line-length)  |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 52   | line too long (178 > 80 characters) (line-length) |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 152  | line too long (85 > 80 characters) (line-length)  |
| apps/60-services/vaultwarden/base/deployment.yaml                                            | 155  | line too long (158 > 80 characters) (line-length) |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 44   | line too long (85 > 80 characters) (line-length)  |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 49   | line too long (102 > 80 characters) (line-length) |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 141  | line too long (85 > 80 characters) (line-length)  |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 146  | trailing spaces (trailing-spaces)                 |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 149  | trailing spaces (trailing-spaces)                 |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 150  | trailing spaces (trailing-spaces)                 |
| apps/60-services/firefly-iii/base/deployment.yaml                                            | 151  | trailing spaces (trailing-spaces)                 |
*... and 12 more warnings*


---

## DRY Violations (Duplicated Configs)

*Files with identical content should be consolidated using shared resources.*

### Duplicate Group 1 (2 files)

- `apps/04-databases/mariadb-shared/overlays/prod/kustomization.yaml`
- `apps/04-databases/redis-shared/overlays/prod/kustomization.yaml`

### Duplicate Group 2 (3 files)

- `apps/04-databases/mariadb-shared/overlays/dev/kustomization.yaml`
- `apps/04-databases/postgresql-shared/overlays/dev/kustomization.yaml`
- `apps/04-databases/redis-shared/overlays/dev/kustomization.yaml`

### Duplicate Group 3 (2 files)

- `apps/04-databases/postgresql-shared/overlays/staging/patch-instances.yaml`
- `apps/04-databases/postgresql-shared/overlays/prod/patch-instances.yaml`

### Duplicate Group 4 (2 files)

- `apps/10-home/mosquitto/overlays/test/mosquitto-tcp-ingressroute.yaml`
- `apps/10-home/mosquitto/overlays/dev/mosquitto-tcp-ingressroute.yaml`

### Duplicate Group 5 (3 files)

- `apps/10-home/mosquitto/overlays/test/kustomization.yaml`
- `apps/10-home/mosquitto/overlays/staging/kustomization.yaml`
- `apps/10-home/mosquitto/overlays/prod/kustomization.yaml`

### Duplicate Group 6 (2 files)

- `apps/10-home/mosquitto/overlays/staging/mosquitto-tcp-ingressroute.yaml`
- `apps/10-home/mosquitto/overlays/prod/mosquitto-tcp-ingressroute.yaml`

### Duplicate Group 7 (2 files)

- `apps/10-home/homeassistant/overlays/test/kustomization.yaml`
- `apps/10-home/homeassistant/overlays/staging/kustomization.yaml`

### Duplicate Group 8 (3 files)

- `apps/10-home/homeassistant/overlays/test/ingressroute-udp.yaml`
- `apps/10-home/homeassistant/overlays/prod/ingressroute-udp.yaml`
- `apps/10-home/homeassistant/overlays/dev/ingressroute-udp.yaml`

### Duplicate Group 9 (3 files)

- `apps/01-storage/nfs-storage/overlays/staging/kustomization.yaml`
- `apps/01-storage/nfs-storage/overlays/prod/kustomization.yaml`
- `apps/01-storage/nfs-storage/overlays/dev/kustomization.yaml`

### Duplicate Group 10 (2 files)

- `apps/01-storage/synology-csi/infisical/overlays/test/kustomization.yaml`
- `apps/01-storage/synology-csi/infisical/overlays/staging/kustomization.yaml`

### Duplicate Group 11 (4 files)

- `apps/00-infra/traefik-dashboard/overlays/test/kustomization.yaml`
- `apps/00-infra/traefik-dashboard/overlays/staging/kustomization.yaml`
- `apps/00-infra/traefik-dashboard/overlays/prod/kustomization.yaml`
- `apps/00-infra/traefik-dashboard/overlays/dev/kustomization.yaml`

### Duplicate Group 12 (2 files)

- `apps/00-infra/cert-manager/overlays/test/kustomization.yaml`
- `apps/00-infra/cert-manager/overlays/dev/kustomization.yaml`

### Duplicate Group 13 (4 files)

- `apps/00-infra/cert-manager/overlays/test/cluster-issuer-prod.yaml`
- `apps/00-infra/cert-manager/overlays/staging/cluster-issuer-prod.yaml`
- `apps/00-infra/cert-manager/overlays/prod/cluster-issuer-prod.yaml`
- `apps/00-infra/cert-manager/overlays/dev/cluster-issuer-prod.yaml`

### Duplicate Group 14 (3 files)

- `apps/00-infra/cert-manager/overlays/test/cluster-issuer-staging.yaml`
- `apps/00-infra/cert-manager/overlays/staging/cluster-issuer-staging.yaml`
- `apps/00-infra/cert-manager/overlays/dev/cluster-issuer-staging.yaml`

### Duplicate Group 15 (4 files)

- `apps/00-infra/cilium-lb/overlays/test/kustomization.yaml`
- `apps/00-infra/cilium-lb/overlays/staging/kustomization.yaml`
- `apps/00-infra/cilium-lb/overlays/prod/kustomization.yaml`
- `apps/00-infra/cilium-lb/overlays/dev/kustomization.yaml`

### Duplicate Group 16 (4 files)

- `apps/00-infra/vpa/overlays/prod/kustomization.yaml`
- `apps/00-infra/vpa/overlays/dev/kustomization.yaml`
- `apps/03-security/trivy/overlays/prod/kustomization.yaml`
- `apps/03-security/trivy/overlays/dev/kustomization.yaml`

### Duplicate Group 17 (3 files)

- `apps/00-infra/reloader/overlays/prod/kustomization.yaml`
- `apps/00-infra/reloader/overlays/dev/kustomization.yaml`
- `apps/40-network/external-dns-unifi/overlays/prod/kustomization.yaml`

### Duplicate Group 18 (2 files)

- `apps/00-infra/argocd/base/namespace.yaml`
- `argocd/base/namespace.yaml`

### Duplicate Group 19 (14 files)

- `apps/00-infra/argocd/overlays/test/kustomization.yaml`
- `apps/00-infra/argocd/overlays/staging/kustomization.yaml`
- `apps/00-infra/argocd/overlays/dev/kustomization.yaml`
- `apps/40-network/contacts/overlays/prod/kustomization.yaml`
- `apps/40-network/adguard-home/overlays/test/kustomization.yaml`
- `apps/40-network/adguard-home/overlays/staging/kustomization.yaml`
- `apps/70-tools/headlamp/overlays/prod/kustomization.yaml`
- `apps/20-media/jellyseerr/overlays/test/kustomization.yaml`
- `apps/20-media/jellyseerr/overlays/staging/kustomization.yaml`
- `apps/20-media/jellyseerr/overlays/prod/kustomization.yaml`
- `apps/20-media/music-assistant/overlays/prod/kustomization.yaml`
- `apps/20-media/jellyfin/overlays/test/kustomization.yaml`
- `apps/20-media/jellyfin/overlays/staging/kustomization.yaml`
- `apps/20-media/jellyfin/overlays/prod/kustomization.yaml`

### Duplicate Group 20 (2 files)

- `apps/00-infra/cert-manager-webhook-gandi/overlays/test/kustomization.yaml`
- `apps/00-infra/cert-manager-webhook-gandi/overlays/staging/kustomization.yaml`

*... and 24 more duplicate groups*

---

## Resource Standard Violations (ADR-008)

| Resource                            | Container                 | Issue                     | File                                                                        |
| ----------------------------------- | ------------------------- | ------------------------- | --------------------------------------------------------------------------- |
| StatefulSet/mariadb-shared          | mariadb                   | Missing resources section | apps/04-databases/mariadb-shared/base/statefulset.yaml                      |
| Deployment/redis-shared             | redis                     | Missing resources section | apps/04-databases/redis-shared/base/deployment.yaml                         |
| Deployment/homeassistant            | config-syncer             | Missing resources section | apps/10-home/homeassistant/overlays/prod/rclone-patch.yaml                  |
| StatefulSet/synology-csi-controller | csi-provisioner           | Missing resources section | apps/01-storage/synology-csi/base/controller.yaml                           |
| StatefulSet/synology-csi-controller | csi-attacher              | Missing resources section | apps/01-storage/synology-csi/base/controller.yaml                           |
| StatefulSet/synology-csi-controller | csi-resizer               | Missing resources section | apps/01-storage/synology-csi/base/controller.yaml                           |
| StatefulSet/synology-csi-controller | synology-csi-plugin       | Missing resources section | apps/01-storage/synology-csi/base/controller.yaml                           |
| DaemonSet/synology-csi-node         | csi-node-driver-registrar | Missing resources section | apps/01-storage/synology-csi/base/node.yaml                                 |
| DaemonSet/synology-csi-node         | synology-csi-plugin       | Missing resources section | apps/01-storage/synology-csi/base/node.yaml                                 |
| Deployment/argocd-redis             | redis                     | Missing resources section | apps/00-infra/argocd/overlays/prod/patches/redis-probes.yaml                |
| DaemonSet/node-agent                | node-agent                | Missing resources section | apps/00-infra/velero/overlays/prod/node-agent-probes.yaml                   |
| Deployment/netvisor-server          | server                    | Missing resources section | apps/40-network/netvisor/base/server-deployment.yaml                        |
| DaemonSet/netvisor-daemon           | daemon                    | Missing resources section | apps/40-network/netvisor/base/daemon-daemonset.yaml                         |
| Deployment/netvisor-server          | server                    | Missing resources section | apps/40-network/netvisor/overlays/test/server-url-patch.yaml                |
| Deployment/netvisor-server          | server                    | Missing resources section | apps/40-network/netvisor/overlays/staging/server-url-patch.yaml             |
| Deployment/netvisor-server          | server                    | Missing resources section | apps/40-network/netvisor/overlays/prod/server-url-patch.yaml                |
| Deployment/netbird-management       | management                | Missing resources section | apps/40-network/netbird/overlays/prod/patches.yaml                          |
| Deployment/netbird-relay            | relay                     | Missing resources section | apps/40-network/netbird/overlays/prod/patches.yaml                          |
| Deployment/netbird-dashboard        | dashboard                 | Missing resources section | apps/40-network/netbird/overlays/prod/patches.yaml                          |
| Deployment/netbird-management       | management                | Missing resources section | apps/40-network/netbird/overlays/dev/patches.yaml                           |
| Deployment/netbird-relay            | relay                     | Missing resources section | apps/40-network/netbird/overlays/dev/patches.yaml                           |
| Deployment/netbird-dashboard        | dashboard                 | Missing resources section | apps/40-network/netbird/overlays/dev/patches.yaml                           |
| StatefulSet/prometheus-alertmanager | alertmanager              | Missing resources section | apps/02-monitoring/prometheus/overlays/prod/patch-alertmanager-command.yaml |
| StatefulSet/prometheus-alertmanager | alertmanager              | Missing resources section | apps/02-monitoring/prometheus/overlays/dev/patch-alertmanager-command.yaml  |
| Deployment/robusta-holmes           | holmes                    | Missing resources section | apps/02-monitoring/robusta/base/holmes.yaml                                 |
| Deployment/gluetun                  | gluetun                   | Missing resources section | apps/60-services/gluetun/base/deployment.yaml                               |
| Deployment/vaultwarden              | vaultwarden               | Missing resources section | apps/60-services/vaultwarden/overlays/prod/deployment-patch.yaml            |
| Deployment/whoami                   | whoami                    | Missing resources section | apps/99-test/whoami/overlays/test/deployment.yaml                           |
| Deployment/whoami                   | whoami                    | Missing resources section | apps/99-test/whoami/overlays/staging/deployment.yaml                        |
| Deployment/tandoor                  | tandoor                   | Missing resources section | apps/99-test/tandoor/base/deployment.yaml                                   |
| Deployment/tandoor                  | db                        | Missing resources section | apps/99-test/tandoor/base/deployment.yaml                                   |
| Deployment/netbox                   | netbox                    | Missing resources section | apps/70-tools/netbox/base/deployment.yaml                                   |
| Deployment/netbox                   | netbox                    | Missing resources section | apps/70-tools/netbox/overlays/test/deployment-patch.yaml                    |
| Deployment/netbox                   | netbox                    | Missing resources section | apps/70-tools/netbox/overlays/staging/deployment-patch.yaml                 |
| Deployment/netbox                   | netbox                    | Missing resources section | apps/70-tools/netbox/overlays/prod/deployment-patch.yaml                    |
| Deployment/netbox                   | netbox                    | Missing resources section | apps/70-tools/netbox/overlays/dev/deployment-patch.yaml                     |
| Deployment/homepage                 | homepage                  | Missing resources section | apps/70-tools/homepage/base/deployment.yaml                                 |
| Deployment/homepage                 | homepage                  | Missing resources section | apps/70-tools/homepage/overlays/prod/deployment-patch.yaml                  |
| Deployment/linkwarden               | linkwarden                | Missing resources section | apps/70-tools/linkwarden/overlays/prod/patch-nextauth-url.yaml              |
| Deployment/authentik-server         | authentik-server          | Missing resources section | apps/03-security/authentik/overlays/prod/deployment-patch.yaml              |
| Deployment/authentik-worker         | authentik-worker          | Missing resources section | apps/03-security/authentik/overlays/prod/worker-patch.yaml                  |
| Deployment/trivy-trivy-operator     | trivy-operator            | Missing resources section | apps/03-security/trivy/base/manifests.yaml                                  |
| Deployment/lazylibrarian            | lazylibrarian             | Missing resources section | apps/20-media/lazylibrarian/base/deployment.yaml                            |
| Deployment/jellyseerr               | jellyseerr                | Missing resources section | apps/20-media/jellyseerr/base/deployment.yaml                               |
| Deployment/music-assistant          | music-assistant           | Missing resources section | apps/20-media/music-assistant/base/deployment.yaml                          |
| Deployment/whisparr                 | litestream                | Missing resources section | apps/20-media/whisparr/base/deployment.yaml                                 |
| Deployment/jellyfin                 | jellyfin                  | Missing resources section | apps/20-media/jellyfin/base/deployment.yaml                                 |
| Deployment/booklore                 | booklore                  | Missing resources section | apps/20-media/booklore/base/deployment.yaml                                 |
| Deployment/booklore-mariadb         | mariadb                   | Missing resources section | apps/20-media/booklore/base/mariadb-deployment.yaml                         |
| Deployment/sabnzbd                  | litestream                | Missing resources section | apps/20-media/sabnzbd/base/deployment.yaml                                  |
*... and 1 more violations*


---

## Recommendations

### 🔴 Critical: Fix Yamllint Errors
- 50 yamllint errors must be fixed
- Run: `just lint` to see all errors

### 🟡 High Priority: Consolidate Duplicates
- 44 duplicate configuration groups found
- Move shared configs to `apps/_shared/`
- Use Kustomize bases/components for reuse

### 🟠 Medium Priority: Add Resource Limits
- 51 containers missing resource specifications
- Follow ADR-008: All containers must have requests + limits
- Use VPA recommendations from Goldilocks

