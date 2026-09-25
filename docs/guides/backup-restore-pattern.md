# Backup/restore pattern — historical

**Status: Superseded**

Ce document décrivait l'ancien pattern manuel `rclone/CronJob + Infisical`. Il ne doit plus servir de modèle de déploiement.

La stratégie actuelle est documentée dans :

- [`docs/reference/configuration-management-strategy.md`](../reference/configuration-management-strategy.md)
- [`docs/adr/013-layered-configuration-disaster-recovery.md`](../adr/013-layered-configuration-disaster-recovery.md)
- [`docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md`](../adr/014-litestream-backup-profiles-and-recovery-patterns.md)
- [`docs/guides/secret-management.md`](secret-management.md) pour les credentials OpenBao/ESO

## Principes encore valables

- distinguer configuration statique versionnable, configuration dynamique et données ;
- ne jamais stocker de secret en clair dans Git ;
- définir RPO/RTO selon la criticité de l'application ;
- tester le **restore**, pas uniquement la présence du backup ;
- garder le mécanisme de récupération déclaratif et reproductible ;
- utiliser les mécanismes spécialisés des bases de données lorsqu'ils existent plutôt qu'un backup de fichiers générique.

## Historique

Les anciennes révisions Git de ce fichier conservent le détail du pattern rclone/CronJob pour référence forensique. Ne pas copier ces anciens manifests dans de nouvelles applications.
