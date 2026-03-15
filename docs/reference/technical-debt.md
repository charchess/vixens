# Registre de la Dette Technique (Technical Debt Registry) 📒

## 🎯 Rôle du document
Ce registre n'est pas un simple inventaire de bugs, mais un journal de bord des **arbitrages d'ingénierie**. Il documente les moments où nous avons délibérément choisi de dévier de nos standards (Goldification, Sizing Kyverno, DRY) pour privilégier la **continuité de service** ou la **stabilité immédiate** du cluster.

L'objectif est de s'assurer que chaque "raccourci" est identifié, justifié et associé à une trajectoire de retour à la conformité via une tâche **Beads**.

---

## 💡 Pourquoi la dette technique apparaît-elle ?

Dans le cadre du projet **Vixens**, la dette technique est principalement générée par deux facteurs de stress :

1.  **Saturation des ressources (Le "Tetris" de la RAM) :** Le cluster Prod est physiquement saturé (allocation proche de 100% sur les Requests). Pour faire démarrer des applications critiques, nous devons parfois "tricher" sur les profils de sizing officiels (Mode Scout).
2.  **Limites du stockage réseau (Latence iSCSI) :** La sensibilité de SQLite aux verrous sur le NAS Synology a forcé des architectures hybrides (DB en RAM) qui s'écartent du déploiement "tout PVC" standard.
3.  **Complexité des Politiques (Kyverno) :** La rigueur des règles de sécurité (Diamond/Gold tier) crée parfois des obstacles au déploiement rapide de correctifs, nécessitant des configurations temporaires "en dur".

---

## 🔴 Dette Critique (Mise en conformité requise)

### 1. Mode Scout (V-scout) - Arbitrage Stabilité vs Standard
*   **ID Tâche :** `vixens-pgk8`
*   **Composants :** `booklore`, `lazylibrarian`
*   **Contexte :** Ces pods restaient en `Pending` car leurs profils standards demandaient trop de RAM garantie (`Requests`).
*   **Dérive :** Utilisation du label `V-scout` (inconnu de Kyverno) et définition manuelle des ressources (`requests: 16Mi/128Mi`, `limits: 4Gi/8Gi`).
*   **Motivation :** Forcer Kubernetes à accepter le pod sur des nœuds saturés tout en lui laissant une limite haute pour ne pas crasher (analyse de consommation réelle).
*   **Risque :** Masque la pression réelle sur le cluster aux yeux de l'ordonnanceur.

### 2. Ressources Sidecars Frigate - Arbitrage DRY vs Urgence
*   **ID Tâche :** `vixens-qkyq`
*   **Composant :** `frigate`
*   **Contexte :** Migration d'urgence de la DB SQLite vers la RAM pour stopper les verrous iSCSI.
*   **Dérive :** Ajout de conteneurs `restore-db` et `patch-config` avec des blocs `resources:` définis en dur dans le déploiement de base.
*   **Motivation :** Garantir que ces conteneurs utilitaires démarrent avec le minimum vital sans attendre une mise à jour complexe de la politique globale `sizing-v2-mutate` de Kyverno.
*   **Risque :** Si la politique Kyverno change, ces conteneurs pourraient devenir des exceptions incohérentes.

---

## 🟡 Dette Modérée (Optimisation requise)

### 3. Erreurs de logique Kyverno
*   **ID Tâche :** `vixens-mg4n`
*   **Composant :** `Kyverno Policies`
*   **Description :** Certaines règles (vulnerability-scan, velero-schedule) échouent avec des erreurs JMESPath (`Invalid type for: <nil>`).
*   **Origine :** Accroissement de la complexité des manifestes dépassant la robustesse actuelle des expressions régulières de la politique.
*   **Impact :** Pollution des logs du cluster et rapports de maturité imprécis.

---

## 🟢 Dette Légère (Maintenance différée)

### 4. NetworkPolicies manuelles
*   **ID Tâche :** `vixens-avrb`
*   **Composant :** `Cilium / NetworkPolicies`
*   **Dérive :** Écriture manuelle de règles de flux (ex: Whisparr -> Sabnzbd) dans chaque base d'application.
*   **Motivation :** Pas d'opérateur d'intention (type Otterize) déployé pour l'instant.
*   **Trajectoire :** Évaluer un contrôleur basé sur les identités Cilium pour automatiser ces flux et redevenir DRY.

---
*Dernière mise à jour : 2026-03-14*
