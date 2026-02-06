# 📝 POST-MORTEM : Incident de perte de données et reset Home Assistant (PROD)

**Date :** 06 Février 2026
**Statut :** RÉSOLU (Restauration à J-6)
**Sévérité :** Critique (S1) - Interruption de service et corruption de données.

---

## 1. 📝 RÉSUMÉ EXÉCUTIF
Lors d'une opération de "Goldification" (durcissement de sécurité et optimisation), l'application Home Assistant a subi un crash en boucle suivi d'une réinitialisation de sa base d'utilisateurs et de ses registres d'entités. L'incident a été aggravé par un effet de bord du système de synchronisation S3 qui a propagé l'état "vide" vers les sauvegardes distantes. La restauration a été effectuée avec succès à partir d'un backup local fourni par l'utilisateur.

---

## 2. ⏳ CHRONOLOGIE DES ÉVÉNEMENTS
- **17:40** : Application du nouveau standard "Elite" sur Home Assistant (ajout de VPA, durcissement `securityContext`).
- **17:45** : **Déclenchement du crash** : Le container refuse de démarrer (`Permission denied` sur `/config/.ha_run.lock`).
- **17:50** : Problèmes de montage iSCSI sur le cluster (timeouts). Kubernetes tente un `mke2fs` sur le volume, croyant qu'il est vierge.
- **18:00 - 19:30** : Tentatives de correction du `securityContext`. Le pod finit par monter le volume mais Home Assistant affiche l'écran d'onboarding.
- **20:00** : **Découverte de la corruption S3** : On s'aperçoit que le container `config-syncer` (rclone) a synchronisé un dossier local vide vers Minio, effaçant ainsi le backup cloud.
- **21:30** : Analyse profonde du `/config` : Les fichiers YAML et la DB sont là, mais les fichiers d'authentification (`.storage/auth`) sont réinitialisés.
- **22:15** : L'utilisateur fournit un backup manuel datant du 31/01 (J-6).
- **22:45** : Lancement de la procédure de restauration manuelle via un pod `recovery`.
- **01:15** (J+1) : Fin de l'extraction, correction des permissions (`1000:1000`) et redémarrage.
- **01:30** : **Confirmation de rétablissement** : Le portail de login est à nouveau accessible.

---

## 3. 🔍 ANALYSE DES CAUSES RACINES (Root Causes)

1.  **Hardening inadapté** : L'application du standard `runAsNonRoot: true` sur une image non-native Kubernetes (`ghcr.io/home-assistant/home-assistant`) a bloqué les scripts d'initialisation (s6-overlay) qui nécessitent des privilèges root pour gérer les verrous et les permissions au démarrage.
2.  **Blind Sync (Le "Tueur de Backup")** : Le container de synchronisation `config-syncer` utilisait la commande `rclone sync`. Dans un état de défaillance iSCSI où le dossier source paraît vide, `rclone sync` a fidèlement reproduit cet état vide sur la destination (Minio), supprimant les sauvegardes valides.
3.  **Absence de Snapshot LUN** : Le NAS Synology n'avait pas de politique de snapshot active pour ce LUN iSCSI, empêchant un rollback instantané au niveau bloc.

---

## 4. 🛡️ ACTIONS CORRECTIVES & PRÉVENTION

### Immédiat (Fait) :
*   **Restauration fonctionnelle** : Retour au backup J-6.
*   **Fix Permissions** : Ajout d'un init-container permanent `fix-perms` pour garantir que l'UID 1000 possède toujours le volume, quel que soit le mode de démarrage du container principal.
*   **Rollback Sécurité** : Suppression des restrictions `securityContext` au niveau container pour cette application spécifique.

### Recommandations à court terme (À faire) :
1.  **Sécurisation du Sync S3** : Remplacer `rclone sync` par `rclone copy` ou ajouter le flag `--max-delete 0` pour empêcher la suppression de fichiers sur le backup cloud en cas d'anomalie locale.
2.  **Politique de Snapshots** : Activer les snapshots toutes les 4h sur le LUN iSCSI via Synology Snapshot Replication.
3.  **Monitoring des Inits** : Ajouter des alertes Robusta spécifiques si un container d'init reste en `PodInitializing` plus de 10 minutes.

---

## 5. 💡 LEÇONS APPRISES
*   **Elite n'est pas Universel** : Le standard "Elite" (Rootless) ne peut pas être appliqué par défaut sur des images "monolithiques" ou héritées du monde Docker-compose sans une phase de test approfondie sur les points de montage système.
*   **L'importance du Backup "Froid"** : Sans ton fichier `.tar` mis de côté, l'instance était totalement perdue suite à l'effacement auto du backup S3.
*   **iSCSI & K8s** : En cas de "Multi-Attach error", la solution la plus sûre est de scaler à 0, attendre le timeout iSCSI (environ 2 min), puis relancer, plutôt que de tenter des suppressions forcées.

---

**Fin du rapport.**
