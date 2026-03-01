# 📊 Matrice des Standards de Maturité (ADR-022)

Ce document récapitule les exigences techniques pour chaque grade de maturité, leur source de configuration et les mécanismes de vérification associés.

| Grade            | Élément                         | Source de Configuration             | Script ? | Kyverno ?   |
| :--------------- | :------------------------------ | :---------------------------------- | :------: | :---------: |
| **🥉 1. Bronze** | Déploiement actif               | `base/deployment.yaml`              |    ✅    |      ❌      |
|                  | Pas de tag `:latest`            | `base/deployment.yaml`              |    ✅    |  ✅ (Audit)  |
|                  | Resources requests définis      | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | Ingress/Service configuré       | `base/service.yaml`                 |    ✅    |  ✅ (Audit)  |
|                  | Structure Kustomize correcte    | Dossiers `base/`, `overlays/`       |    ✅    |      ❌      |
| **🥈 2. Silver** | Resources limits définis        | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | Readiness Probe                 | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | TLS/HTTPS activé                | `base/ingress.yaml` (spec.tls)      |    ✅    |  ✅ (Audit)  |
|                  | Secrets via Infisical           | `overlays/prod/kustomization.yaml`  |    ✅    |  ✅ (Audit)  |
|                  | PVC Strategy (contextuel)       | `base/pvc.yaml`                     |    ✅    |  ⚠️ (Audit)  |
| **🥇 3. Gold**   | Liveness Probe                  | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | Goldilocks activé               | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | VPA Annotations                 | `base/deployment.yaml`              |    ✅    |  ⚠️ (Audit)  |
|                  | Métriques exposées              | `base/deployment.yaml`              |    ✅    |  ✅ (Audit)  |
|                  | ServiceMonitor (contextuel)     | `base/servicemonitor.yaml`          |    ✅    |  ✅ (Audit)  |
| **💎 4. Platinum**| PriorityClass assigné           | `base/deployment.yaml`              |    ✅    | ⚠️ (Enforce) |
|                  | QoS Class                       | `base/deployment.yaml`              |    ✅    |  ✅ (Audit)  |
|                  | revisionHistoryLimit: 3         | `base/deployment.yaml`              |    ✅    | ⚠️ (Enforce) |
|                  | Sync-wave configuré             | `argocd/base/apps/<app>.yaml`       |    ✅    |  ✅ (Audit)  |
|                  | PodDisruptionBudget             | `base/pdb.yaml`                     |    ✅    |  ✅ (Audit)  |
|                  | Shared Components utilisés      | `overlays/prod/kustomization.yaml`  |    ✅    |      ❌      |
| **🟢 5. Emerald** | Backup Profile défini           | `base/deployment.yaml` (labels)     |    ✅    |  ✅ (Audit)  |
|                  | Litestream (si SQLite)          | `base/deployment.yaml` (sidecar)    |    ✅    |  ✅ (Audit)  |
|                  | Config-Syncer sidecar           | `base/deployment.yaml` (sidecar)    |    ✅    |  ✅ (Audit)  |
|                  | Restore InitContainer           | `base/deployment.yaml`              |    ✅    |  ✅ (Audit)  |
|                  | Sidecar Resources (granulaire)  | `base/deployment.yaml` (labels)     |    ✅    | ✅ (Mutate)  |
| **💠 6. Diamond** | PSA Labels (enforce)            | `_shared/namespaces/<ns>.yaml`      |    ✅    |  ⚠️ (Audit)  |
|                  | NetworkPolicies (L3/L4)         | `base/networkpolicy.yaml`           |    ✅    |  ✅ (Audit)  |
|                  | SecurityContext durci           | `base/deployment.yaml`              |    ✅    |  ✅ (Audit)  |
|                  | Authentik SSO (contextuel)      | `base/ingress.yaml`                 |    ✅    |  ✅ (Audit)  |
|                  | Velero backup confirmé          | `velero` namespace                  |    ✅    |  ✅ (Audit)  |
| **🌟 7. Orichalcum**| 7 jours de stabilité            | Métriques (restarts)                |    ✅    |  ✅ (Audit)  |
|                  | Guaranteed QoS                  | `base/deployment.yaml` (Req == Lim) |    ✅    |  ✅ (Audit)  |
|                  | Runbooks documentés             | Annotations (vixens.io/runbook)     |    ✅    |  ✅ (Audit)  |
|                  | SLO/SLI définis                 | Alerts / Monitoring Git             |    ❌    |      ❌      |
|                  | Policy Kyverno compliant        | PolicyReports cluster               |    ✅    | ✅ (Mutate)  |

---

### 💡 Notes sur les mécanismes de vérification :

1.  **Kyverno (Garde-fou passif/actif)** :
    *   **Audit** : Signale les manquements sans bloquer (majorité des politiques).
    *   **Enforce** : Bloque la création si non-conforme (PriorityClass, revisionLimit).
    *   **Mutate** : Injecte automatiquement les valeurs (Sizing granulaire, sidecars).
    *   **God Mode** : Vérifie la stabilité réelle (restarts), les liens opérationnels (Runbooks) et la sécurité prouvée (Trivy Scans).

2.  **Script `evaluate_maturity.py` (Reporting actif)** :
    *   Vérifie l'état **réel** dans le cluster.
    *   Analyse la structure des fichiers **locaux** (Kustomize, Components).
    *   C'est l'outil de référence pour générer les rapports `CONFORMITY-xxx.md`.

3.  **Légende des symboles** :
    *   ✅ : Totalement automatisé et vérifié.
    *   ⚠️ : Vérifié mais peut dépendre du mode (Audit/Enforce) ou vérifie un paramètre critique.
    *   ❌ : Non vérifié par cet outil (généralement des critères de processus métier).
    *   🚧 : En cours d'implémentation.
