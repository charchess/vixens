# Standards de Ressources et Priorités (QoS)

Ce document définit les standards de "Compute Quality of Service" pour le cluster Vixens.
L'objectif est de garantir la stabilité des services critiques tout en maximisant l'utilisation du matériel pour les tâches de fond.

## 1. Classes de Priorité (PriorityClasses)

Les applications doivent spécifier `priorityClassName` selon leur criticité.

| Nom | Valeur | Description | Exemples |
| :--- | :--- | :--- | :--- |
| **`vixens-critical`** | `100000` | **Infra Core.** Ne doit jamais être expulsé. | Ingress, Cert-Manager, CSI, Authentik |
| **`vixens-high`** | `50000` | **Vital Services.** Haute disponibilité requise. | Home Assistant, MQTT, Prometheus |
| **`vixens-medium`** | `10000` | **Standard Apps.** Applications utilisateur interactives. | Plex/Jellyfin, *arr apps, Vaultwarden |
| **`vixens-low`** | `0` | **Background.** Tâches de fond sacrifiables. | Downloaders (Sabnzbd), Scrapers, Transcodage |

## 2. Profils de Ressources (T-Shirt Sizing)

Utilisez ces profils comme base de départ pour `resources.requests` et `resources.limits`.
Ajustez ensuite selon les recommandations de **Goldilocks/VPA**.

| Taille | CPU (Req / Lim) | RAM (Req / Lim) | Usage Typique |
| :--- | :--- | :--- | :--- |
| **Micro** | `10m` / `100m` | `64Mi` / `128Mi` | Sidecars (Litestream, Reloader), Exporters |
| **Small** | `50m` / `500m` | `256Mi` / `512Mi` | Apps optimisées (Go/Rust), Outils statiques |
| **Medium** | `200m` / `1000m` | `512Mi` / `1Gi` | Web Apps standards (Python/Node), Monitoring léger |
| **Large** | `1000m` / `2000m` | `2Gi` / `4Gi` | Bases de données, Apps lourdes (Jellyfin, Gitlab) |
| **XLarge** | `2000m` / `4000m` | `4Gi` / `8Gi` | Traitement AI (Frigate), Gros Indexeurs |
| **Unknown**| `50m` / `2000m` | `256Mi` / `4Gi` | **Profil "Safety Net".** Pour les nouvelles apps au comportement inconnu. Permet le burst sans bloquer le scheduling. |

## 3. Règles d'Application

1.  **Requests = Garantie.** C'est ce que le scheduler "réserve". Ne pas sur-provisionner les requests, sinon les nodes seront "pleins" artificiellement.
2.  **Limits = Sécurité.** C'est le plafond avant que le process ne soit throttled (CPU) ou tué (RAM).
3.  **Burstable par défaut.** Nous privilégions le modèle `Burstable` (`requests < limits`) pour mutualiser les ressources CPU inactives.
4.  **Java/JVM.** Pour les apps Java, définissez toujours `RAM Request = RAM Limit` pour éviter que le Garbage Collector ne devienne fou.
5.  **Revision History Limit.** Pour éviter l'encombrement de l'API Kubernetes, toutes les ressources (Deployments, StatefulSets) doivent limiter leur historique à **3 révisions**. Ce standard est appliqué globalement via un composant Kustomize partagé (`apps/_shared/patches`).

## 4. Snippet YAML Standard

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      priorityClassName: vixens-medium  # <-- Choisir la priorité
      containers:
        - name: app
          resources: # <-- Profil "Small"
            requests:
              cpu: "50m"
              memory: "256Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
```
