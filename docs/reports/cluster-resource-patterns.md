# Rapport : Dimensionnement et Réutilisation des Conteneurs (Cluster Prod) 📊

Ce rapport présente l'analyse exhaustive des requêtes de ressources (CPU/RAM) pour toutes les applications du cluster, ainsi qu'un état des lieux de la réutilisation des patterns de sidecars et initContainers.

---

## 1. Tableau des Requests et Recommandations VPA (Exhaustif)

Ce tableau liste les ressources garanties (**Requests**) réservées sur les nœuds du cluster pour chaque conteneur, comparées aux observations réelles du VPA.

| Namespace                 | Pod                                                             | Conteneur                            | Type | Request   | Mini (VPA) | Target  | Burst   |
| ------------------------- | --------------------------------------------------------------- | ------------------------------------ | ---- | --------- | ---------- | ------- | ------- |
| argocd                    | argocd-application-controller-0                                 | argocd-application-controller        | App  | -         | 932.3Mi    | 1.0Gi   | 1.4Gi   |
| argocd                    | argocd-applicationset-controller-669df69b77-6bsfs               | argocd-applicationset-controller     | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| argocd                    | argocd-dex-server-58954d4dd4-fmt4b                              | copyutil                             | Init | -         | -          | -       | -       |
| argocd                    | argocd-dex-server-58954d4dd4-fmt4b                              | dex                                  | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| argocd                    | argocd-notifications-controller-7c74c68fc5-x5bfn                | argocd-notifications-controller      | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| argocd                    | argocd-redis-7cbd6bb499-848zr                                   | secret-init                          | Init | -         | -          | -       | -       |
| argocd                    | argocd-redis-7cbd6bb499-848zr                                   | redis                                | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| argocd                    | argocd-repo-server-7975786d96-54xp4                             | copyutil                             | Init | -         | -          | -       | -       |
| argocd                    | argocd-repo-server-7975786d96-54xp4                             | argocd-repo-server                   | App  | -         | 640.1Mi    | 1.2Gi   | 1.7Gi   |
| argocd                    | argocd-server-74c5cd4598-p495c                                  | argocd-server                        | App  | 512Mi     | 137.8Mi    | 194.3Mi | 266.0Mi |
| auth                      | authentik-server-55686c8b9-lkx2h                                | restore-config                       | Init | 64Mi      | -          | -       | -       |
| auth                      | authentik-server-55686c8b9-lkx2h                                | authentik-server                     | App  | 1Gi       | 825.8Mi    | 878.1Mi | 908.6Mi |
| auth                      | authentik-server-55686c8b9-lkx2h                                | config-syncer                        | App  | 256Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| auth                      | authentik-worker-56d9864c7d-5crrb                               | authentik-worker                     | App  | 1Gi       | 1.5Gi      | 1.6Gi   | 1.7Gi   |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | generate-litestream-config           | Init | 16Mi      | -          | -       | -       |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | litestream-restore                   | Init | 32Mi      | -          | -       | -       |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | birdnet-go                           | App  | 512Mi     | 283.3Mi    | 362.6Mi | 434.1Mi |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | litestream                           | App  | 256Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| birdnet-go                | birdnet-go-66449c54b4-2mrx2                                     | data-syncer                          | App  | 256Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| cert-manager              | cert-manager-566fdcc9bb-7jffp                                   | cert-manager-controller              | App  | 128Mi     | -          | -       | -       |
| cert-manager              | cert-manager-566fdcc9bb-qfrvz                                   | cert-manager-controller              | App  | 128Mi     | -          | -       | -       |
| cert-manager              | cert-manager-cainjector-6fc9d76df9-nczkn                        | cert-manager-cainjector              | App  | 128Mi     | -          | -       | -       |
| cert-manager              | cert-manager-webhook-db5c59f79-xpw2z                            | cert-manager-webhook                 | App  | 128Mi     | -          | -       | -       |
| cert-manager              | cert-manager-webhook-gandi-6d465567f8-ct8hr                     | cert-manager-webhook-gandi           | App  | 128Mi     | -          | -       | -       |
| cert-manager              | cert-manager-webhook-gandi-6d465567f8-nptmc                     | cert-manager-webhook-gandi           | App  | 128Mi     | -          | -       | -       |
| cnpg-system               | cloudnative-pg-77c96f4cd5-qmgr5                                 | manager                              | App  | 128Mi     | -          | -       | -       |
| databases                 | mariadb-shared-0                                                | mariadb                              | App  | 512Mi     | 128.0Mi    | 128.0Mi | 143.6Mi |
| databases                 | postgresql-shared-1                                             | bootstrap-controller                 | Init | 512Mi     | -          | -       | -       |
| databases                 | postgresql-shared-1                                             | postgres                             | App  | 2Gi       | -          | -       | -       |
| databases                 | redis-shared-5879d7966c-6lb9j                                   | redis                                | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| downloads                 | amule-5cbd5d8c58-2kl7g                                          | configure-proxy                      | Init | 64Mi      | -          | -       | -       |
| downloads                 | amule-5cbd5d8c58-2kl7g                                          | amule                                | App  | 256Mi     | 128.0Mi    | 174.6Mi | 227.2Mi |
| downloads                 | pyload-79dfb6974b-gs9q4                                         | pyload                               | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| finance                   | firefly-iii-66c6cb7b5-rp8hr                                     | restore-config                       | Init | 64Mi      | -          | -       | -       |
| finance                   | firefly-iii-66c6cb7b5-rp8hr                                     | firefly-iii                          | App  | 512Mi     | 104.7Mi    | 104.7Mi | 123.3Mi |
| finance                   | firefly-iii-66c6cb7b5-rp8hr                                     | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| finance                   | firefly-iii-cron-29547540-hjsdt                                 | firefly-iii-cron                     | App  | 64Mi      | 128.0Mi    | 194.3Mi | 226.9Mi |
| finance                   | firefly-iii-importer-8d78799f5-qbv6d                            | importer                             | App  | 256Mi     | 128.0Mi    | 128.0Mi | 141.5Mi |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | restore-config                       | Init | 64Mi      | -          | -       | -       |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | restore-db                           | Init | 64Mi      | -          | -       | -       |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | fix-perms                            | Init | 64Mi      | -          | -       | -       |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | install-python-deps                  | Init | 64Mi      | -          | -       | -       |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | config-init                          | Init | 64Mi      | -          | -       | -       |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | homeassistant                        | App  | 2Gi       | 1.4Gi      | 1.4Gi   | 1.6Gi   |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | litestream                           | App  | 512Mi     | 599.5Mi    | 2.4Gi   | 2.4Gi   |
| homeassistant             | homeassistant-66bd698f79-zjh57                                  | config-syncer                        | App  | 256Mi     | 104.7Mi    | 120.9Mi | 124.7Mi |
| infisical-operator-system | infisical-opera-controller-manager-d9ff5d68f-dtn5s              | manager                              | App  | 128Mi     | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | config                               | Init | -         | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | apply-sysctl-overwrites              | Init | -         | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | mount-bpf-fs                         | Init | -         | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | clean-cilium-state                   | Init | -         | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | install-cni-binaries                 | Init | 10Mi      | -          | -       | -       |
| kube-system               | cilium-878rn                                                    | cilium-agent                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | config                               | Init | -         | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | apply-sysctl-overwrites              | Init | -         | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | mount-bpf-fs                         | Init | -         | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | clean-cilium-state                   | Init | -         | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | install-cni-binaries                 | Init | 10Mi      | -          | -       | -       |
| kube-system               | cilium-8phv2                                                    | cilium-agent                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-envoy-2nz4g                                              | cilium-envoy                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-envoy-45qnr                                              | cilium-envoy                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-envoy-hjgtv                                              | cilium-envoy                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-envoy-j2n9z                                              | cilium-envoy                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-envoy-jt2nx                                              | cilium-envoy                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | config                               | Init | -         | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | apply-sysctl-overwrites              | Init | -         | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | mount-bpf-fs                         | Init | -         | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | clean-cilium-state                   | Init | -         | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | install-cni-binaries                 | Init | 10Mi      | -          | -       | -       |
| kube-system               | cilium-nrxvl                                                    | cilium-agent                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-operator-6f778fbdc7-m29p6                                | cilium-operator                      | App  | -         | -          | -       | -       |
| kube-system               | cilium-operator-f6c85d9b8-xmjlq                                 | cilium-operator                      | App  | -         | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | config                               | Init | -         | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | apply-sysctl-overwrites              | Init | -         | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | mount-bpf-fs                         | Init | -         | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | clean-cilium-state                   | Init | -         | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | install-cni-binaries                 | Init | 10Mi      | -          | -       | -       |
| kube-system               | cilium-t5zs6                                                    | cilium-agent                         | App  | -         | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | config                               | Init | -         | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | apply-sysctl-overwrites              | Init | -         | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | mount-bpf-fs                         | Init | -         | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | clean-cilium-state                   | Init | -         | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | install-cni-binaries                 | Init | 10Mi      | -          | -       | -       |
| kube-system               | cilium-w9vlm                                                    | cilium-agent                         | App  | -         | -          | -       | -       |
| kube-system               | coredns-58b95c77f8-mb2g8                                        | coredns                              | App  | 70Mi      | -          | -       | -       |
| kube-system               | coredns-58b95c77f8-sr4hb                                        | coredns                              | App  | 70Mi      | -          | -       | -       |
| kube-system               | descheduler-29542080-kj7x9                                      | descheduler                          | App  | 64Mi      | -          | -       | -       |
| kube-system               | hubble-relay-7ccb867d9b-vvrxr                                   | hubble-relay                         | App  | -         | -          | -       | -       |
| kube-system               | kube-apiserver-phoebe                                           | kube-apiserver                       | App  | 512Mi     | -          | -       | -       |
| kube-system               | kube-apiserver-poison                                           | kube-apiserver                       | App  | 512Mi     | -          | -       | -       |
| kube-system               | kube-apiserver-powder                                           | kube-apiserver                       | App  | 512Mi     | -          | -       | -       |
| kube-system               | kube-controller-manager-phoebe                                  | kube-controller-manager              | App  | 256Mi     | -          | -       | -       |
| kube-system               | kube-controller-manager-poison                                  | kube-controller-manager              | App  | 256Mi     | -          | -       | -       |
| kube-system               | kube-controller-manager-powder                                  | kube-controller-manager              | App  | 256Mi     | -          | -       | -       |
| kube-system               | kube-scheduler-phoebe                                           | kube-scheduler                       | App  | 64Mi      | -          | -       | -       |
| kube-system               | kube-scheduler-poison                                           | kube-scheduler                       | App  | 64Mi      | -          | -       | -       |
| kube-system               | kube-scheduler-powder                                           | kube-scheduler                       | App  | 64Mi      | -          | -       | -       |
| kube-system               | metrics-server-778cf877c-r62gw                                  | metrics-server                       | App  | 200Mi     | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-4mwmq                   | kyverno-pre                          | Init | 64Mi      | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-4mwmq                   | kyverno                              | App  | 128Mi     | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-4xxjb                   | kyverno-pre                          | Init | 64Mi      | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-4xxjb                   | kyverno                              | App  | 128Mi     | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-qv8td                   | kyverno-pre                          | Init | 64Mi      | -          | -       | -       |
| kyverno                   | kyverno-admission-controller-69bff7b7cb-qv8td                   | kyverno                              | App  | 128Mi     | -          | -       | -       |
| kyverno                   | kyverno-background-controller-64b9f945f5-fq6hf                  | controller                           | App  | 128Mi     | -          | -       | -       |
| kyverno                   | kyverno-cleanup-controller-6679cc59c8-2pxgt                     | controller                           | App  | 128Mi     | -          | -       | -       |
| kyverno                   | kyverno-reports-controller-5fb4758854-4jtv5                     | controller                           | App  | 128Mi     | -          | -       | -       |
| kyverno                   | maturity-controller-29558895-98nwd                              | kubectl-copy                         | Init | -         | -          | -       | -       |
| kyverno                   | maturity-controller-29558895-98nwd                              | controller                           | App  | -         | -          | -       | -       |
| kyverno                   | maturity-controller-29558910-qmn88                              | kubectl-copy                         | Init | -         | -          | -       | -       |
| kyverno                   | maturity-controller-29558910-qmn88                              | controller                           | App  | -         | -          | -       | -       |
| kyverno                   | maturity-controller-29558925-d4szj                              | kubectl-copy                         | Init | -         | -          | -       | -       |
| kyverno                   | maturity-controller-29558925-d4szj                              | controller                           | App  | -         | -          | -       | -       |
| mealie                    | mealie-68cbcd45f6-f7wmb                                         | restore-config                       | Init | 64Mi      | -          | -       | -       |
| mealie                    | mealie-68cbcd45f6-f7wmb                                         | restore-db                           | Init | 64Mi      | -          | -       | -       |
| mealie                    | mealie-68cbcd45f6-f7wmb                                         | mealie                               | App  | 512Mi     | 362.5Mi    | 391.7Mi | 458.8Mi |
| mealie                    | mealie-68cbcd45f6-f7wmb                                         | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| mealie                    | mealie-68cbcd45f6-f7wmb                                         | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | booklore-7d84d9c6c6-j99tf                                       | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | booklore-7d84d9c6c6-j99tf                                       | config-syncer                        | App  | 64Mi      | 64.0Mi     | 259.5Mi | 311.6Mi |
| media                     | booklore-7d84d9c6c6-j99tf                                       | booklore                             | App  | 128Mi     | 990.1Mi    | 2.1Gi   | 2.6Gi   |
| media                     | booklore-mariadb-bdf4bd5fd-pqmv6                                | mariadb                              | App  | 256Mi     | 236.6Mi    | 259.5Mi | 466.5Mi |
| media                     | booklore-mariadb-bdf4bd5fd-pqmv6                                | db-backup                            | App  | 64Mi      | 74.6Mi     | 104.7Mi | 122.6Mi |
| media                     | frigate-5cf6967f67-6477t                                        | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | frigate-5cf6967f67-6477t                                        | validate-config                      | Init | 32Mi      | -          | -       | -       |
| media                     | frigate-5cf6967f67-6477t                                        | generate-litestream-config           | Init | 16Mi      | -          | -       | -       |
| media                     | frigate-5cf6967f67-6477t                                        | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | frigate-5cf6967f67-6477t                                        | patch-config                         | Init | -         | -          | -       | -       |
| media                     | frigate-5cf6967f67-6477t                                        | frigate                              | App  | 2Gi       | 4.9Gi      | 6.0Gi   | 6.1Gi   |
| media                     | frigate-5cf6967f67-6477t                                        | litestream                           | App  | 64Mi      | 236.6Mi    | 422.3Mi | 432.1Mi |
| media                     | frigate-5cf6967f67-6477t                                        | config-syncer                        | App  | 64Mi      | 104.7Mi    | 120.9Mi | 141.2Mi |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | check-integrity                      | Init | 512Mi     | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | restore-mappings                     | Init | 64Mi      | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | restore-master                       | Init | 64Mi      | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | restore-caches                       | Init | 64Mi      | -          | -       | -       |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | hydrus-client                        | App  | 1Gi       | 1.5Gi      | 2.8Gi   | 3.5Gi   |
| media                     | hydrus-client-76ff947f8d-p9zs8                                  | litestream                           | App  | 64Mi      | 64.0Mi     | 74.6Mi  | 87.7Mi  |
| media                     | jellyfin-fc6db5ccd-gnbdj                                        | jellyfin                             | App  | 512Mi     | 236.6Mi    | 334.9Mi | 396.1Mi |
| media                     | jellyseerr-694cf6fb9d-bjqnq                                     | jellyseerr                           | App  | 512Mi     | 283.3Mi    | 334.9Mi | 394.4Mi |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | lazylibrarian                        | App  | 225384266 | 155.8Mi    | 214.9Mi | 254.5Mi |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | lazylibrarian-7c58b85d68-mpb9d                                  | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | lidarr-6645d6f844-cxhq8                                         | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | lidarr-6645d6f844-cxhq8                                         | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | lidarr-6645d6f844-cxhq8                                         | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | lidarr-6645d6f844-cxhq8                                         | lidarr                               | App  | 512Mi     | 194.2Mi    | 259.5Mi | 304.9Mi |
| media                     | lidarr-6645d6f844-cxhq8                                         | litestream                           | App  | 64Mi      | 64.0Mi     | 74.6Mi  | 87.7Mi  |
| media                     | lidarr-6645d6f844-cxhq8                                         | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 71.2Mi  |
| media                     | music-assistant-7f66fcbcbb-gwctl                                | music-assistant                      | App  | 256Mi     | 334.8Mi    | 362.6Mi | 427.0Mi |
| media                     | mylar-68795ffd54-dpp4n                                          | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | mylar-68795ffd54-dpp4n                                          | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | mylar-68795ffd54-dpp4n                                          | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | mylar-68795ffd54-dpp4n                                          | mylar                                | App  | 512Mi     | 89.3Mi     | 104.7Mi | 122.6Mi |
| media                     | mylar-68795ffd54-dpp4n                                          | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | mylar-68795ffd54-dpp4n                                          | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | prowlarr-89457d948-wvg8z                                        | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | prowlarr-89457d948-wvg8z                                        | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | prowlarr-89457d948-wvg8z                                        | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | prowlarr-89457d948-wvg8z                                        | prowlarr                             | App  | 512Mi     | 120.9Mi    | 174.6Mi | 204.2Mi |
| media                     | prowlarr-89457d948-wvg8z                                        | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | prowlarr-89457d948-wvg8z                                        | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | qbittorrent-687b8d8585-mknb8                                    | qbittorrent                          | App  | 512Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| media                     | radarr-656c6674f7-gc4k4                                         | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | radarr-656c6674f7-gc4k4                                         | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | radarr-656c6674f7-gc4k4                                         | radarr                               | App  | 512Mi     | 259.4Mi    | 334.9Mi | 395.2Mi |
| media                     | radarr-656c6674f7-gc4k4                                         | litestream                           | App  | 64Mi      | 74.6Mi     | 174.6Mi | 204.5Mi |
| media                     | radarr-656c6674f7-gc4k4                                         | config-syncer                        | App  | 64Mi      | 89.3Mi     | 120.9Mi | 142.6Mi |
| media                     | sabnzbd-669b598b4-fptrb                                         | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | sabnzbd-669b598b4-fptrb                                         | fix-permissions                      | Init | -         | -          | -       | -       |
| media                     | sabnzbd-669b598b4-fptrb                                         | sabnzbd                              | App  | 512Mi     | 89.3Mi     | 104.7Mi | 142.3Mi |
| media                     | sabnzbd-669b598b4-fptrb                                         | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | sabnzbd-669b598b4-fptrb                                         | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| media                     | sonarr-66dd597c84-r5f2w                                         | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | sonarr-66dd597c84-r5f2w                                         | restore-db                           | Init | 64Mi      | -          | -       | -       |
| media                     | sonarr-66dd597c84-r5f2w                                         | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | sonarr-66dd597c84-r5f2w                                         | sonarr                               | App  | 256Mi     | 194.2Mi    | 334.9Mi | 405.3Mi |
| media                     | sonarr-66dd597c84-r5f2w                                         | litestream                           | App  | 64Mi      | 104.7Mi    | 137.9Mi | 161.7Mi |
| media                     | sonarr-66dd597c84-r5f2w                                         | config-syncer                        | App  | 64Mi      | 64.0Mi     | 104.7Mi | 122.8Mi |
| media                     | whisparr-7f4d575cd8-x5xl4                                       | restore-config                       | Init | 64Mi      | -          | -       | -       |
| media                     | whisparr-7f4d575cd8-x5xl4                                       | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| media                     | whisparr-7f4d575cd8-x5xl4                                       | whisparr                             | App  | 512Mi     | 174.5Mi    | 194.3Mi | 227.7Mi |
| media                     | whisparr-7f4d575cd8-x5xl4                                       | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 73.2Mi  |
| media                     | whisparr-7f4d575cd8-x5xl4                                       | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| monitoring                | goldilocks-controller-5c6d8dc5bd-5pnvj                          | goldilocks                           | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | goldilocks-dashboard-55bb7748c4-4slmk                           | goldilocks                           | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | goldilocks-dashboard-55bb7748c4-hqqjs                           | goldilocks                           | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | grafana-5fd5d566bd-69w6x                                        | init-chown-data                      | Init | 64Mi      | -          | -       | -       |
| monitoring                | grafana-5fd5d566bd-69w6x                                        | grafana-sc-dashboard                 | App  | 256Mi     | 422.3Mi    | 422.3Mi | 429.1Mi |
| monitoring                | grafana-5fd5d566bd-69w6x                                        | grafana                              | App  | 512Mi     | 174.6Mi    | 334.9Mi | 340.2Mi |
| monitoring                | loki-0                                                          | loki                                 | App  | 512Mi     | 137.9Mi    | 174.6Mi | 177.4Mi |
| monitoring                | prometheus-alertmanager-0                                       | alertmanager                         | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-kube-state-metrics-64bfb6fb74-clptp                  | kube-state-metrics                   | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-prometheus-node-exporter-4nj5n                       | node-exporter                        | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-prometheus-node-exporter-dk6jw                       | node-exporter                        | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-prometheus-node-exporter-fgnlz                       | node-exporter                        | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-prometheus-node-exporter-s6lv4                       | node-exporter                        | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-prometheus-node-exporter-x7m2d                       | node-exporter                        | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| monitoring                | prometheus-server-79dbc9786c-bfz6m                              | prometheus-server-configmap-reload   | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| monitoring                | prometheus-server-79dbc9786c-bfz6m                              | prometheus-server                    | App  | 2Gi       | 2.8Gi      | 2.8Gi   | 2.8Gi   |
| monitoring                | promtail-df4vw                                                  | promtail                             | App  | 256Mi     | 128.0Mi    | 137.9Mi | 140.0Mi |
| monitoring                | promtail-dsjpf                                                  | promtail                             | App  | 256Mi     | 128.0Mi    | 137.9Mi | 140.0Mi |
| monitoring                | promtail-jt6t9                                                  | promtail                             | App  | 256Mi     | 128.0Mi    | 137.9Mi | 140.0Mi |
| monitoring                | promtail-n9xtm                                                  | promtail                             | App  | 256Mi     | 128.0Mi    | 137.9Mi | 140.0Mi |
| monitoring                | promtail-rqcms                                                  | promtail                             | App  | 256Mi     | 128.0Mi    | 137.9Mi | 140.0Mi |
| mosquitto                 | mosquitto-0                                                     | restore-data                         | Init | 64Mi      | -          | -       | -       |
| mosquitto                 | mosquitto-0                                                     | create-mosquitto-users               | Init | 64Mi      | -          | -       | -       |
| mosquitto                 | mosquitto-0                                                     | mosquitto                            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| mosquitto                 | mosquitto-0                                                     | config-syncer                        | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| networking                | adguard-home-0                                                  | restore-config                       | Init | 64Mi      | -          | -       | -       |
| networking                | adguard-home-0                                                  | restore-db                           | Init | 64Mi      | -          | -       | -       |
| networking                | adguard-home-0                                                  | adguard-home                         | App  | 128Mi     | 89.3Mi     | 104.7Mi | 108.3Mi |
| networking                | adguard-home-0                                                  | litestream                           | App  | 64Mi      | 42.7Mi     | 42.7Mi  | 42.7Mi  |
| networking                | adguard-home-0                                                  | config-syncer                        | App  | 64Mi      | 42.7Mi     | 42.7Mi  | 42.7Mi  |
| networking                | external-dns-gandi-55cbb8b856-kpcm5                             | external-dns                         | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | external-dns-unifi-5cb59f8655-wcwrs                             | external-dns                         | App  | 128Mi     | 64.0Mi     | 120.9Mi | 122.8Mi |
| networking                | external-dns-unifi-5cb59f8655-wcwrs                             | unifi-webhook                        | App  | 128Mi     | 64.0Mi     | 74.6Mi  | 75.7Mi  |
| networking                | netbird-dashboard-fb866cbb8-9t6bl                               | fix-nginx-dirs                       | Init | -         | -          | -       | -       |
| networking                | netbird-dashboard-fb866cbb8-9t6bl                               | dashboard                            | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netbird-management-78c657c74b-dfr9h                             | init-config                          | Init | 64Mi      | -          | -       | -       |
| networking                | netbird-management-78c657c74b-dfr9h                             | management                           | App  | 512Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netbird-relay-88589948-tqtdg                                    | relay                                | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netbird-signal-547f8f8c5f-vrs4j                                 | signal                               | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netvisor-daemon-jnjdb                                           | daemon                               | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netvisor-daemon-nj8lp                                           | daemon                               | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netvisor-daemon-nz4cn                                           | daemon                               | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netvisor-daemon-pq78g                                           | daemon                               | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| networking                | netvisor-server-5d8b94d8c8-m8sw9                                | server                               | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| policy-reporter           | policy-reporter-565d967887-nzsz6                                | policy-reporter                      | App  | 128Mi     | -          | -       | -       |
| policy-reporter           | policy-reporter-kyverno-plugin-6fc7f9bb55-bv49m                 | kyverno-plugin                       | App  | 128Mi     | -          | -       | -       |
| policy-reporter           | policy-reporter-ui-7dd74dcc8d-rb95h                             | ui                                   | App  | 128Mi     | -          | -       | -       |
| security                  | node-collector-6776599f5-zr9wp                                  | node-collector                       | App  | 100M      | -          | -       | -       |
| security                  | node-collector-85cf877848-lzxzg                                 | node-collector                       | App  | 128Mi     | -          | -       | -       |
| security                  | scan-vulnerabilityreport-645fd7f6b6-ck4zg                       | a3d8a709-6f95-4a87-b3af-98848660305c | Init | 64Mi      | -          | -       | -       |
| security                  | scan-vulnerabilityreport-645fd7f6b6-ck4zg                       | redis                                | App  | 128Mi     | -          | -       | -       |
| security                  | scan-vulnerabilityreport-6696b8f868-qw6st                       | 0cf4830a-a1e4-4213-8790-f5c7889d9b9e | Init | 100M      | -          | -       | -       |
| security                  | scan-vulnerabilityreport-6696b8f868-qw6st                       | krr                                  | App  | 100M      | -          | -       | -       |
| security                  | scan-vulnerabilityreport-779f6b556b-7jrmg                       | 7a6614f5-3fbe-4734-b033-21ebb9496bf9 | Init | 100M      | -          | -       | -       |
| security                  | scan-vulnerabilityreport-779f6b556b-7jrmg                       | kube-scheduler                       | App  | 100M      | -          | -       | -       |
| security                  | trivy-trivy-operator-5b874d875b-q5rql                           | trivy-operator                       | App  | 1Gi       | -          | -       | -       |
| services                  | docspell-joex-0                                                 | joex                                 | App  | 1Gi       | 683.3Mi    | 683.5Mi | 806.7Mi |
| services                  | docspell-restserver-0                                           | restserver                           | App  | 512Mi     | 523.2Mi    | 523.4Mi | 622.1Mi |
| services                  | gluetun-59d46966fc-fvfsx                                        | gluetun                              | App  | 64Mi      | 128.0Mi    | 155.8Mi | 182.4Mi |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | restore-config                       | Init | 64Mi      | -          | -       | -       |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | setup-config                         | Init | 64Mi      | -          | -       | -       |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | install-tools                        | Init | 2Gi       | -          | -       | -       |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | install-gemini                       | Init | 512Mi     | -          | -       | -       |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | openclaw                             | App  | 2Gi       | 2.0Gi      | 2.6Gi   | 3.2Gi   |
| services                  | openclaw-7cd9c6fb99-5kskf                                       | data-syncer                          | App  | 64Mi      | 64.0Mi     | 74.6Mi  | 90.1Mi  |
| services                  | vaultwarden-7757cd48f7-5kf9l                                    | fix-permissions                      | Init | 64Mi      | -          | -       | -       |
| services                  | vaultwarden-7757cd48f7-5kf9l                                    | restore-config                       | Init | 64Mi      | -          | -       | -       |
| services                  | vaultwarden-7757cd48f7-5kf9l                                    | vaultwarden                          | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| services                  | vaultwarden-7757cd48f7-5kf9l                                    | litestream                           | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| services                  | vaultwarden-7757cd48f7-5kf9l                                    | config-syncer                        | App  | 64Mi      | 64.0Mi     | 89.3Mi  | 104.5Mi |
| synology-csi              | synology-csi-controller-0                                       | csi-provisioner                      | App  | 128Mi     | 47.3Mi     | 47.3Mi  | 48.0Mi  |
| synology-csi              | synology-csi-controller-0                                       | csi-attacher                         | App  | 128Mi     | 34.6Mi     | 34.6Mi  | 35.1Mi  |
| synology-csi              | synology-csi-controller-0                                       | csi-resizer                          | App  | 128Mi     | 60.6Mi     | 60.6Mi  | 61.5Mi  |
| synology-csi              | synology-csi-controller-0                                       | synology-csi-plugin                  | App  | 128Mi     | 32.0Mi     | 34.6Mi  | 35.1Mi  |
| synology-csi              | synology-csi-node-7qbjj                                         | iscsi-lock-cleanup                   | Init | -         | -          | -       | -       |
| synology-csi              | synology-csi-node-7qbjj                                         | csi-node-driver-registrar            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| synology-csi              | synology-csi-node-7qbjj                                         | synology-csi-plugin                  | App  | 128Mi     | 64.0Mi     | 120.9Mi | 158.2Mi |
| synology-csi              | synology-csi-node-d8rsm                                         | iscsi-lock-cleanup                   | Init | -         | -          | -       | -       |
| synology-csi              | synology-csi-node-d8rsm                                         | csi-node-driver-registrar            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| synology-csi              | synology-csi-node-d8rsm                                         | synology-csi-plugin                  | App  | 128Mi     | 64.0Mi     | 120.9Mi | 158.2Mi |
| synology-csi              | synology-csi-node-nrltn                                         | iscsi-lock-cleanup                   | Init | -         | -          | -       | -       |
| synology-csi              | synology-csi-node-nrltn                                         | csi-node-driver-registrar            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| synology-csi              | synology-csi-node-nrltn                                         | synology-csi-plugin                  | App  | 128Mi     | 64.0Mi     | 120.9Mi | 158.2Mi |
| synology-csi              | synology-csi-node-vfd2v                                         | iscsi-lock-cleanup                   | Init | -         | -          | -       | -       |
| synology-csi              | synology-csi-node-vfd2v                                         | csi-node-driver-registrar            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| synology-csi              | synology-csi-node-vfd2v                                         | synology-csi-plugin                  | App  | 128Mi     | 64.0Mi     | 120.9Mi | 158.2Mi |
| synology-csi              | synology-csi-node-x9mpk                                         | iscsi-lock-cleanup                   | Init | -         | -          | -       | -       |
| synology-csi              | synology-csi-node-x9mpk                                         | csi-node-driver-registrar            | App  | 128Mi     | 64.0Mi     | 64.0Mi  | 64.0Mi  |
| synology-csi              | synology-csi-node-x9mpk                                         | synology-csi-plugin                  | App  | 128Mi     | 64.0Mi     | 120.9Mi | 158.2Mi |
| tools                     | changedetection-c948f698d-r2b9v                                 | restore-config                       | Init | 64Mi      | -          | -       | -       |
| tools                     | changedetection-c948f698d-r2b9v                                 | changedetection                      | App  | 256Mi     | 236.6Mi    | 334.9Mi | 401.2Mi |
| tools                     | changedetection-c948f698d-r2b9v                                 | browserless                          | App  | 512Mi     | 391.6Mi    | 933.0Mi | 1.1Gi   |
| tools                     | changedetection-c948f698d-r2b9v                                 | config-syncer                        | App  | 64Mi      | 64.0Mi     | 64.0Mi  | 72.6Mi  |
| tools                     | headlamp-d76fb985-lbx6m                                         | headlamp                             | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| tools                     | homepage-65bcfd7498-gfc7z                                       | copy-initial-config                  | Init | 64Mi      | -          | -       | -       |
| tools                     | homepage-65bcfd7498-gfc7z                                       | homepage                             | App  | 256Mi     | 137.9Mi    | 137.9Mi | 161.4Mi |
| tools                     | it-tools-74b9cb47b4-skksk                                       | it-tools                             | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |
| tools                     | linkwarden-f787f478f-qvb8d                                      | linkwarden                           | App  | 512Mi     | 775.8Mi    | 878.1Mi | 1.0Gi   |
| tools                     | netbox-85454577c5-xbdc6                                         | netbox                               | App  | 512Mi     | 1.0Gi      | 1.0Gi   | 1.2Gi   |
| tools                     | nocodb-b99877dd9-clst8                                          | nocodb                               | App  | 256Mi     | 308.4Mi    | 308.5Mi | 364.2Mi |
| tools                     | penpot-backend-855dc47556-rng7f                                 | backend                              | App  | 1Gi       | 825.5Mi    | 825.8Mi | 979.6Mi |
| tools                     | penpot-exporter-794676f55-mznh4                                 | exporter                             | App  | 256Mi     | 128.0Mi    | 174.6Mi | 204.4Mi |
| tools                     | penpot-frontend-d4656b5d9-k2kd2                                 | frontend                             | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| tools                     | radar-85f67cf9f9-qxnwq                                          | radar                                | App  | 512Mi     | 391.6Mi    | 454.4Mi | 621.7Mi |
| tools                     | reloader-reloader-f4b6df755-tr5n2                               | reloader-reloader                    | App  | 128Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| tools                     | renovate-29540160-pm92h                                         | renovate                             | App  | 64Mi      | 128.0Mi    | 155.8Mi | 27.5Gi  |
| tools                     | trilium-7b8dfbc956-zmgjf                                        | trilium                              | App  | 256Mi     | 137.9Mi    | 155.8Mi | 182.7Mi |
| tools                     | vikunja-86965688d4-zxsn5                                        | vikunja                              | App  | 256Mi     | 128.0Mi    | 128.0Mi | 128.0Mi |
| traefik                   | traefik-7485ff4cb-pq869                                         | traefik                              | App  | 128Mi     | -          | -       | -       |
| traefik                   | traefik-7485ff4cb-xnj5v                                         | traefik                              | App  | 128Mi     | -          | -       | -       |
| velero                    | homeassistant-default-kopia-maintain-job-1773534873175-7w7tm    | velero-repo-maintenance-container    | App  | 128Mi     | -          | -       | -       |
| velero                    | homeassistant-default-kopia-maintain-job-1773535173176-sxpfp    | velero-repo-maintenance-container    | App  | 128Mi     | -          | -       | -       |
| velero                    | homeassistant-default-kopia-maintain-job-1773535473177-9mw8z    | velero-repo-maintenance-container    | App  | 128Mi     | -          | -       | -       |
| velero                    | node-agent-n4wdd                                                | node-agent                           | App  | 512Mi     | 64.0Mi     | 64.0Mi  | 101.5Mi |
| velero                    | node-agent-qlbs2                                                | node-agent                           | App  | 512Mi     | 64.0Mi     | 64.0Mi  | 101.5Mi |
| velero                    | node-agent-vhz7j                                                | node-agent                           | App  | 512Mi     | 64.0Mi     | 64.0Mi  | 101.5Mi |
| velero                    | node-agent-x68sn                                                | node-agent                           | App  | 512Mi     | 64.0Mi     | 64.0Mi  | 101.5Mi |
| velero                    | node-agent-x7n8t                                                | node-agent                           | App  | 512Mi     | 64.0Mi     | 64.0Mi  | 101.5Mi |
| velero                    | velero-65fb878966-nbsz7                                         | velero-plugin-for-aws                | Init | 64Mi      | -          | -       | -       |
| velero                    | velero-65fb878966-nbsz7                                         | velero                               | App  | 512Mi     | 104.7Mi    | 283.4Mi | 386.5Mi |
| vpa                       | vpa-vertical-pod-autoscaler-admission-controller-56585cdc4j6gz6 | admission-controller                 | App  | 128Mi     | -          | -       | -       |
| vpa                       | vpa-vertical-pod-autoscaler-recommender-7b7bc6c6bc-2wchj        | recommender                          | App  | 256Mi     | -          | -       | -       |
| vpa                       | vpa-vertical-pod-autoscaler-updater-5795b8fc65-2vpkc            | updater                              | App  | 256Mi     | -          | -       | -       |
| whoami                    | whoami-7746565486-hpvsr                                         | whoami                               | App  | 64Mi      | 128.0Mi    | 128.0Mi | 128.0Mi |

---

## 2. Patterns de Réutilisation (Fréquence d'usage)

Ce tableau identifie les conteneurs les plus fréquents sur le cluster.

| Pattern                 | Utilisations |
| ----------------------- | ------------ |
| restore-config          | 18           |
| config-syncer           | 18           |
| litestream              | 15           |
| fix-permissions         | 11           |
| restore-db              | 10           |
| controller              | 6            |
| synology-csi-plugin     | 6            |
| config                  | 5            |
| apply-sysctl-overwrites | 5            |
| mount-bpf-fs            | 5            |
| clean-cilium-state      | 5            |
| install-cni-binaries    | 5            |
| cilium-agent            | 5            |
| cilium-envoy            | 5            |
| node-exporter           | 5            |

---
*Rapport généré automatiquement le : 2026-03-14*
