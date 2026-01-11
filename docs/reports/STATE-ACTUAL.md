# Application State - Actual (Dev Reality)

**Last Updated:** 2026-01-11
**Environment:** Dev Cluster
**Data Source:** Live cluster state + GitOps manifests

---

## Production Application Inventory

| App                            | NS                        | CPU Req | CPU Lim | Mem Req | Mem Lim | VPA Target       | Profile | Priority                | Sync Wave | Backup Profile | Score |
| ------------------------------ | ------------------------- | ------- | ------- | ------- | ------- | ---------------- | ------- | ----------------------- | --------- | -------------- | ----- |
| **argocd**                     | argocd                    | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **argocd-image-updater**       | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 10        | None           | 0     |
| **authentik**                  | auth                      | N/A     | N/A     | N/A     | N/A     | 587m / 628694953 | Unknown | N/A                     | 10        | None           | 0     |
| **cert-manager**               | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **cert-manager-config**        | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 2         | None           | 0     |
| **cert-manager-secrets**       | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **cert-manager-webhook-gandi** | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 1         | None           | 0     |
| **cilium-lb**                  | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | -2        | None           | 0     |
| **cloudnative-pg**             | cnpg-system               | 50m     | 200m    | 128Mi   | 256Mi   | No VPA           | Small   | homelab-critical        | 3         | None           | 0     |
| **cloudnative-pg-crds**        | cnpg-system               | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 2         | None           | 0     |
| **descheduler**                | kube-system               | 50m     | 200m    | 64Mi    | 128Mi   | No VPA           | Micro   | system-cluster-critical | 10        | None           | 0     |
| **external-dns-unifi**         | networking                | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 5         | None           | 0     |
| **external-dns-unifi-secrets** | networking                | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 4         | None           | 0     |
| **farmos**                     | test                      | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **gitops-revision-controller** | tools                     | 50m     | 200m    | 128Mi   | 512Mi   | No VPA           | Small   | N/A                     | 10        | None           | 0     |
| **grafana**                    | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 6         | None           | 0     |
| **grafana-ingress**            | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 7         | None           | 0     |
| **headlamp**                   | tools                     | 100m    | 200m    | 128Mi   | 256Mi   | No VPA           | Small   | N/A                     | 10        | None           | 0     |
| **hubble-ui**                  | monitoring                | 100m    | 200m    | 128Mi   | 256Mi   | No VPA           | Small   | N/A                     | 10        | None           | 0     |
| **hydrus-client**              | media                     | 500m    | 2       | 1Gi     | 4Gi     | 23m / 548861636  | Large   | vixens-medium           | 10        | Standard       | 0     |
| **infisical-operator**         | infisical-operator-system | 100m    | 500m    | 128Mi   | 256Mi   | No VPA           | Small   | N/A                     | -3        | None           | 0     |
| **kubernetes-dashboard**       | kubernetes-dashboard      | 100m    | 250m    | 200Mi   | 400Mi   | No VPA           | Small   | N/A                     | 5         | None           | 0     |
| **mariadb-shared**             | databases                 | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 4         | None           | 0     |
| **mariadb-shared-config**      | databases                 | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 3         | None           | 0     |
| **media-namespace**            | media                     | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | -1        | None           | 0     |
| **metrics-server**             | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 10        | None           | 0     |
| **nfs-storage**                | media-stack               | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | -1        | None           | 0     |
| **postgresql-shared**          | databases                 | 100m    | 500m    | 256Mi   | 512Mi   | No VPA           | Small   | homelab-critical        | 4         | None           | 0     |
| **priority-classes**           | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | -4        | None           | 0     |
| **prometheus**                 | monitoring                | 100m    | N/A     | 128Mi   | 512Mi   | No VPA           | Small   | N/A                     | 5         | None           | 0     |
| **prometheus-ingress**         | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 6         | None           | 0     |
| **redis-shared**               | databases                 | N/A     | N/A     | N/A     | N/A     | 35m / 128Mi      | Unknown | N/A                     | -1        | None           | 0     |
| **reloader**                   | tools                     | 10m     | 100m    | 128Mi   | 256Mi   | No VPA           | Small   | N/A                     | 10        | None           | 0     |
| **renovate**                   | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 20        | None           | 0     |
| **synology-csi**               | synology-csi              | N/A     | N/A     | N/A     | N/A     | 11m / 63544758   | Unknown | N/A                     | 0         | None           | 0     |
| **synology-csi-secrets**       | synology-csi              | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | -1        | None           | 0     |
| **traefik**                    | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **traefik-dashboard**          | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 1         | None           | 0     |
| **traefik-middlewares**        | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 1         | None           | 0     |
| **vixens-app-of-apps**         | argocd                    | N/A     | N/A     | N/A     | N/A     | No VPA           | Unknown | N/A                     | 0         | None           | 0     |
| **vpa**                        | vpa                       | 50m     | 200m    | 100Mi   | 500Mi   | No VPA           | Small   | N/A                     | 0         | None           | 0     |
