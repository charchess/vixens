# Rapport d'Incident : Perte de Configuration Frigate (Production)
**Date :** 27 Février 2026
**Statut :** Investiguer (Solutions proposées)

## 📝 Résumé de l'incident
Lors du redémarrage du Pod Frigate (suite à une correction nécessaire des permissions système), la configuration active sur le volume permanent (PVC) a été écrasée par une version obsolète provenant du stockage S3. Ce comportement est dû à une faille dans le mécanisme de restauration automatique combinée à une panne silencieuse du système de sauvegarde.

---

## 🔍 Problèmes Identifiés

### 1. Restauration destructive (InitContainer)
L'initContainer `restore-config` effectue un `rclone copy` de S3 vers `/config` à chaque démarrage du Pod.
*   **Défaut :** Il n'y a aucune condition de vérification (ex: tester si le dossier est vide).
*   **Conséquence :** Si la version sur S3 est plus ancienne que celle sur le disque, S3 gagne et écrase tout.

### 2. Sidecar de Synchronisation cassé (Config-Syncer)
Le conteneur chargé de sauvegarder tes changements de l'UI vers S3 ne fonctionnait plus.
*   **Absence de dépendance :** L'image `python:3.14-alpine` utilisée n'inclut pas la bibliothèque `yaml`. La commande de validation `import yaml` plantait systématiquement.
*   **Logique de sécurité bloquante :** Le script est conçu pour ne pas synchroniser vers S3 si la validation YAML échoue. Comme Python ne trouvait pas le module YAML, il considérait la config comme invalide et **refusait de mettre à jour S3**.
*   **Pollution du backup :** `rclone` tentait de synchroniser le dossier `model_cache` contenant des liens symboliques, ce qui générait des erreurs de synchronisation.

### 3. Effet Domino
1.  Des modifications ont été faites via l'UI (PVC à jour, mais S3 obsolète car le synchro était cassé).
2.  Le Pod a redémarré pour appliquer les nouveaux droits root.
3.  `restore-config` a démarré en root, a pu lire S3, et a **écrasé** le PVC avec la vieille config.

---

## 💡 Solutions Proposées

### Solution A : Sécuriser la Restauration
Modifier l'initContainer `restore-config` pour qu'il soit **conditionnel**.
*   **Logique :** "Si `config.yml` existe déjà sur le PVC, ne fais rien. Sinon, télécharge depuis S3."
*   **Bénéfice :** Un redémarrage ne pourra plus jamais écraser tes données locales.

### Solution B : Réparer le Synchro (Sidecar)
1.  **Ajouter les dépendances :** Installer `py3-yaml` au démarrage du conteneur.
2.  **Optimiser le périmètre :** Exclure explicitement les dossiers inutiles (`model_cache/`, `lost+found/`, `.frigate.db-litestream/`) de la sauvegarde S3 pour éviter les erreurs de liens symboliques et réduire le volume de données.

### Solution C : Source de Vérité
Revoir si la configuration doit être pilotée par **Infisical** (GitOps) ou par l'**UI de Frigate** (Runtime). Actuellement, le système tente de faire les deux, ce qui crée des conflits de priorité.

---

## 🚀 Prochaines Étapes recommandées
1.  Appliquer le patch de "Restauration Conditionnelle".
2.  Réparer le conteneur de synchronisation (ajout de PyYAML).
3.  Ré-importer la bonne configuration via l'UI une fois le système sécurisé.
