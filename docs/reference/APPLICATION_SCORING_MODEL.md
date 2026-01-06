# Modèle de Notation des Applications (Scoring Model)

Ce document définit les standards de qualité pour les applications du cluster Vixens.
Il se concentre sur les **capacités fonctionnelles** et la maturité opérationnelle.

**Note cible pour Production : 85/100**

## 1. Intégration GitOps & Standards (20 pts)

*L'application est-elle correctement gérée par notre usine logicielle ?*

*   **Standardisation (10 pts)** : Structure des dossiers, nommage et labels conformes aux conventions (`apps/<type>/<nom>`).
*   **Découplage Configuration (10 pts)** : Configuration métier externalisée (ConfigMap/Secret), images immuables (pas de `latest`) et secrets via Infisical.

## 2. Qualité de Service (QoS) & Stabilité (20 pts)

*L'application est-elle un "bon voisin" dans le cluster ?*

*   **Contrat de Ressources (10 pts)** : `Requests` et `Limits` définis de manière réaliste pour garantir la stabilité du node.
*   **Observabilité & Priorité (10 pts)** : Sondes de santé (`liveness`/`readiness`) configurées et `PriorityClass` adaptée à la criticité du service.

## 3. Sécurité & Accès (20 pts)

*L'application est-elle exposée de manière sécurisée ?*

*   **Accès Chiffré (10 pts)** : Terminaison TLS valide (Cert-Manager) et redirection HTTP -> HTTPS forcée (Middleware Traefik).
*   **Contextualisation (10 pts)** : Les URLs s'adaptent automatiquement à l'environnement (`dev` vs `prod`). Aucun "hardcoding" de domaine dans la `base`.

## 4. Parité Dev/Prod (20 pts)

*L'application est-elle ISO entre les environnements pour garantir la fiabilité des tests ?*

*   **Identité Architecturale (10 pts)** : Utilisation des mêmes briques logicielles en Dev et Prod (mêmes bases de données, mêmes sidecars de sauvegarde).
*   **Intégrité DRY (10 pts)** : Utilisation stricte d'une `base` commune. Seules les variables d'ajustement (replicas, hostnames, ressources) diffèrent dans les `overlays`.

## 5. Continuité & Intégrité des Données (20 pts)

*L'application résiste-t-elle aux pannes et préserve-t-elle ses données ?*

*   **Protection des Données (10 pts)** : Mécanisme de sauvegarde automatique (Litestream, Config-Syncer ou Operator) vers un stockage externe (S3).
*   **Auto-Guérison (10 pts)** : Restauration automatique au démarrage et rechargement de configuration sans interruption de service (`Reloader`).

---

## Validation (Definition of Done)

Un score de 85+ ne suffit pas. Les tests suivants sont bloquants :

1.  **Test de "Zero-Touch"** : Le déploiement complet se fait sans aucune commande manuelle (`kubectl exec`, etc.).
2.  **Test d'Accès** : L'URL publique renvoie un code HTTP valide.
3.  **Test de Résilience** : Tuer un pod ne doit pas entraîner de perte de données ou d'indisponibilité prolongée au-delà du redémarrage.
4.  **Validation Dev** : L'application doit être validée fonctionnellement en Dev avant toute promotion en Prod.