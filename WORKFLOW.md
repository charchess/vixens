# Processus de Travail (Beads + Just)

Adhérence stricte et totale requise. Pas de raccourcis. Ce workflow remplace l'ancien usage d'Archon pour la gestion des tâches.

## 1. Initialisation & Sélection (Beads)
*   **Commande :** Utiliser `just resume` pour identifier la tâche en cours ou reprendre le travail.
*   **Priorité 1 :** Tâches assignées à "coding-agent" avec le statut `review`.
*   **Priorité 2 :** Tâches assignées à "coding-agent" avec le statut `in_progress`.
*   **Priorité 3 :** Si aucune tâche active, lister les tâches ouvertes (`bd list --status open`) et **PROPOSER** la liste à l'utilisateur.

## 2. Exécution (Just Work)
Lancer la tâche avec `just work <task_id>`. Cela automatise les étapes suivantes :
*   **Phase 1 (Prérequis) :** Vérification automatique des spécificités techniques (ex: PVC RWO détecté -> ajout d'une note pour `strategy: Recreate`).
*   **Phase 2 (Documentation) :** Identification et invitation à consulter `docs/applications/<app>.md`.
*   **Phase 3 (Analyse & Développement) :** 
    *   Analyse du "Definition of Done" (DoD) basé sur la demande et les tests de non-régression.
    *   Utilisation de **Serena** pour l'accès au code et **Archon** pour la base de connaissances (RAG).
    *   Développement incrémental sur la branche `dev` uniquement.
    *   Mise à jour de `docs/applications/<app>.md` si l'infrastructure ou la validation évoluent.
    *   Application des changements dans l'overlay `prod` une fois le développement `dev` stabilisé.

## 3. Clôture & Validation
Une fois les modifications terminées :
*   **Commit & Push :** Branche `dev` uniquement !
*   **Validation Forcée :** `just _validate <task_id>` (exécute `scripts/validate.py`).
    *   Vérification de l'accès à l'application et conformité DoD.
    *   Utilisation de **Playwright** pour les interfaces web (validation visuelle).
*   **Promotion :** Si validation 100% OK en dev, la promotion vers `main`/prod se fait via GitHub Actions (`promote-prod.yaml`).
*   **Finalisation :** Si succès en prod, passer la tâche en statut `review` et changer l'assigné vers `user` (`bd update <task_id> --status review --assignee user`).

---

## Notes Importantes
*   **Controlplane :** Se rappeler des tolérations nécessaires.
*   **Storage :** Si PVC `ReadWriteOnce` (RWO) => `strategy: Recreate` obligatoire.
*   **Réseau :** Redirection HTTP vers HTTPS systématique.
*   **Certificats :** `letsencrypt-staging` en dev, `letsencrypt-prod` en prod.
*   **URLs Ingress :** `<app>.dev.truxonline.com` (dev) / `<app>.truxonline.com` (prod).
*   **Design :** Approche DRY, orientée maintenabilité et "state of the art".

## Workflow GitOps (Trunk-Based)
*   **Branches :** `dev` (développement) et `main` (production).
*   **Flux :** Feature branch -> PR vers `dev` -> Merge.
*   **Auto-Tag :** GitHub Action crée un tag `dev-vX.Y.Z` après merge dans `dev`.
*   **Promotion :** `gh workflow run promote-prod.yaml -f version=v1.2.3`.
*   **ArgoCD :** Synchronisation automatique basée sur les branches/tags. Voir ADR-008.

## Outils & Commandes
*   **Beads (bd) :** Gestionnaire de tâches (remplace Archon Task).
*   **Just :** Orchestrateur de workflow (`just resume`, `just work`).
*   **Serena :** Outil principal pour la lecture et modification de fichiers.
*   **Archon :** Outil de recherche documentaire (RAG).
*   **Playwright :** Validation des interfaces web (backup: curl).
*   **kubectl/talosctl :** Utiliser les configs dans `terraform/environments/dev/` ou `terraform/environments/prod/`.
