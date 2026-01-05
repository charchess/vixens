# Modèle de Notation des Applications (Scoring Model)

Ce document définit les standards de qualité pour les applications du cluster Vixens.
Il se concentre sur les **capacités fonctionnelles** et la maturité opérationnelle.

**Note cible pour Production : 85/100**

## 1. Intégration GitOps & Standards (25 pts)

*L'application est-elle correctement gérée par notre usine logicielle ?*

*   **Standardisation (10 pts)** : Structure des dossiers, nommage des ressources et labels conformes aux conventions du projet (`apps/<type>/<nom>`).
*   **Isolation (5 pts)** : Namespace dédié ou partagé explicitement. Pas de pollution de ressources globales (ClusterRole) sans justification.
*   **Découplage Configuration (10 pts)** :
    *   La configuration métier est externalisée (ConfigMap/Secret).
    *   L'image conteneur est immuable (pas de `latest`, pas de modif interne).
    *   Les secrets ne sont jamais dans le code (référence à Infisical/ExternalSecrets).

## 2. Qualité de Service (QoS) & Stabilité (25 pts)

*L'application est-elle un "bon voisin" dans le cluster ?*

*   **Contrat de Ressources (10 pts)** :
    *   `Requests` définis (le minimum garanti).
    *   `Limits` définies (le plafond de sécurité).
    *   Recommandations VPA/Goldilocks consultées.
*   **Priorisation (5 pts)** : `PriorityClass` définie selon la criticité (`critical`, `important`, ou défaut).
*   **Observabilité de Santé (10 pts)** :
    *   L'application signale quand elle est vivante (`liveness`).
    *   L'application signale quand elle peut recevoir du trafic (`readiness`).
    *   L'application gère son démarrage lent si besoin (`startup`).

## 3. Sécurité & Accès (20 pts)

*L'application est-elle exposée de manière sécurisée ?*

*   **Accès Chiffré (10 pts)** :
    *   Terminaison TLS valide (Cert-Manager).
    *   Redirection HTTP -> HTTPS forcée.
*   **Contextualisation (10 pts)** :
    *   Les URLs s'adaptent automatiquement à l'environnement (`dev` vs `prod`).
    *   Pas d'IP ou de nom de domaine "en dur" dans les manifestes de base.

## 4. Continuité & Intégrité des Données (30 pts)

*L'application résiste-t-elle aux pannes et préserve-t-elle ses données ?*

*   **Stratégie de Persistance (10 pts)** :
    *   Les données éphémères (cache) sont distinguées des données persistantes.
    *   Le type de stockage (RWO/Block vs RWX/NFS) est adapté au besoin (Performance vs Partage).
*   **Protection des Données (10 pts)** :
    *   **DB SQLite** : Litestream Sidecar (Streaming Replication).
    *   **Fichiers Plats (Config)** : Config-Syncer Sidecar (Inotify + Rclone vers S3).
    *   **DB Externe** : Backup Operator/Dump vers S3.
*   **Auto-Guérison (10 pts)** :
    *   Capacité de restauration automatique au démarrage (si DB locale).
    *   Ou procédure de restauration documentée et testée (si DB externe).
    *   Rechargement automatique de la configuration (`Reloader`) sans redémarrage violent.

---

## Validation (Definition of Done)

Un score de 85+ ne suffit pas. Les tests suivants sont bloquants :

1.  **Test de "Zero-Touch"** : Le déploiement complet se fait sans aucune commande manuelle (`kubectl exec`, etc.).
2.  **Test d'Accès** : L'URL publique renvoie un code HTTP valide.
3.  **Test de Résilience** : Tuer un pod ne doit pas entraîner de perte de données ou d'indisponibilité prolongée au-delà du redémarrage.
