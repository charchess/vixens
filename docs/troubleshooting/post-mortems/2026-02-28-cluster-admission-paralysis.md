# 📝 POST-MORTEM : Paralysie d'admission cluster et Corruption Home Assistant (PROD)

**Date :** 28 Février 2026
**Statut :** RÉSOLU
**Sévérité :** Critique (S1) - Interruption totale de service (DNS, ArgoCD, Domotique).

---

## 1. 📝 RÉSUMÉ EXÉCUTIF
Le 28 février 2026, une modification mineure sur Music Assistant a déclenché une cascade de défaillances. L'absence de labels de ressources sur Home Assistant a provoqué son crash immédiat, surchargeant le contrôleur d'admission Kyverno. En raison d'une saturation CPU du cluster et d'une configuration de sécurité trop stricte, le cluster est entré en état de "paralysie d'admission", bloquant toute création de pod (y compris ArgoCD et le DNS AdGuard). Le service a été rétabli en brisant manuellement les verrous d'admission et en stabilisant la configuration GitOps.

---

## 2. ⏳ CHRONOLOGIE DES ÉVÉNEMENTS
- **18:45** : Déploiement de Music Assistant (ouverture du port 3483). ArgoCD déclenche une resynchronisation globale.
- **18:50** : **Crash de Home Assistant** : L'application n'avait pas de label `vixens.io/sizing`. Kyverno a appliqué le profil `micro` par défaut (**128 Mo**). HA subit des OOMKilled répétés.
- **19:15** : **Saturation du cluster** : Le nœud `powder` atteint 99% de CPU. Les communications internes Kubernetes ralentissent.
- **19:30** : **Paralysie d'Admission** : Kyverno ne répond plus assez vite (timeouts). Le webhook étant en `failurePolicy: Fail`, Kubernetes bloque toute création de pod dans le cluster.
- **20:30** : Redémarrage des nœuds Talos. Le cluster se vide de ses pods. ArgoCD et AdGuard ne parviennent pas à remonter, bloqués par Kyverno.
- **21:00** : **Action d'Urgence** : Suppression manuelle du déploiement Kyverno et de ses webhooks pour lever le verrou réseau. ArgoCD et AdGuard redémarrent instantanément.
- **21:30** : **Stabilisation GitOps** : 
    - Réécriture de la politique Kyverno (`foreach`) pour stopper les panics internes.
    - Passage de Kyverno en `failurePolicy: Ignore` et timeout 30s.
    - Restauration des ressources Burstable et priorité `critical` pour HA et AdGuard.
- **22:15** : **Rétablissement Total** : Home Assistant et Music Assistant passent en `Running` avec les bons paramètres.

---

## 3. 🔍 ANALYSE DES CAUSES RACINES (Root Causes)

1.  **Dette Technique (Labels)** : Home Assistant n'avait pas de label de sizing explicite dans Git, le rendant vulnérable aux mutations par défaut de Kyverno.
2.  **Configuration "Fail-Closed"** : Le webhook Kyverno était configuré pour bloquer le cluster en cas d'indisponibilité (`failurePolicy: Fail`), ce qui est inadapté à un cluster saturé.
3.  **Conflit de Mutation (Bug Kyverno)** : Deux règles de mutation s'affrontaient sur la liste des containers, provoquant un dépassement d'index (`slice bounds out of range`) et des panics du contrôleur.
4.  **Saturation CPU** : La charge extrême du cluster (99% CPU) a transformé des micro-latences réseau en blocage total du plan de contrôle.

---

## 4. 🛡️ ACTIONS CORRECTIVES & PRÉVENTION

### Immédiat (Fait) :
*   **Résilience Kyverno** : Passage en `failurePolicy: Ignore` et boost des ressources (3 réplicas, 1 CPU).
*   **Bridage des Sidecars** : Nouvelle règle Kyverno limitant les sidecars (Litestream, config-syncer) à **128 Mo**, réduisant la demande de RAM de HA de 6 Go à 2.2 Go.
*   **Priorisation DNS/HA** : Passage d'AdGuard et Home Assistant en priorité `vixens-critical`.

### Recommandations à court terme (À faire) :
1.  **Audit de Sizing** : Vérifier que TOUTES les applications portent le label `vixens.io/sizing`.
2.  **Optimisation Media** : Activer le VPA en mode `Auto` sur la stack Media pour libérer du CPU sur les nœuds saturés.
3.  **Monitoring Kyverno** : Ajouter une alerte spécifique sur les erreurs de webhook admission.

---

## 5. 💡 LEÇONS APPRISES
*   **Le GitOps est la seule vérité** : Les interventions manuelles (`kubectl patch`) lors de l'incident ont créé plus de confusion. Il aurait fallu agir directement sur les `Application` ArgoCD.
*   **La mutation est une arme à double tranchant** : Une politique de mutation doit être testée contre les récursions et les conflits de listes avant d'être appliquée à la production.
*   **Fail-Open en Prod** : Pour les webhooks de mutation non-sécuritaires, le mode `Ignore` est impératif pour garantir la disponibilité du cluster.

---

**Fin du rapport.**
