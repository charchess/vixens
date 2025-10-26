# Principes Directeurs du Projet Vixens

Ce document est la constitution technique du projet. Toutes les décisions d'architecture, de code et de workflow doivent se conformer à ces principes.

## 1. Git comme Source de Vérité Unique

L'état désiré de toute l'infrastructure (décrite avec Terraform) et de toutes les applications (décrites en manifestes Kubernetes) est exclusivement défini dans ce dépôt Git. Aucune modification manuelle (`kubectl`, modification directe sur l'hyperviseur) n'est autorisée sur les environnements gérés. Le seul état canonique est celui déclaré dans la branche Git correspondant à l'environnement.

## 2. Idempotence des Opérations

Toutes les opérations automatisées doivent être idempotentes. Exécuter `terraform apply` ou une synchronisation ArgoCD plusieurs fois sur un état inchangé doit produire le même résultat final sans erreur ni modification.

## 3. Environnements Éphémères et Reproductibles

Les environnements (en particulier `dev` et `test`) sont considérés comme éphémères et entièrement remplaçables. Il n'y a pas de processus de mise à jour incrémentale de l'infrastructure socle (version de Talos, configuration des nœuds).

Les mises à jour fondamentales de l'infrastructure s'effectuent via un cycle complet de destruction et de recréation : `terraform destroy` suivi d'un `terraform apply`. Cette approche garantit qu'aucun état résiduel ou dérive de configuration ne peut exister.

## 4. Sécurité Intégrée et Pragmatic

L'objectif est de ne stocker aucun secret (mots de passe, tokens, clés d'API) en clair dans le dépôt Git. La solution privilégiée est le chiffrement des valeurs via un outil comme Mozilla SOPS.

Toutefois, la capacité à diagnostiquer rapidement les problèmes est une priorité. Si l'utilisation d'outils de chiffrement rend le débogage excessivement opaque durant les phases initiales de développement, le stockage temporaire de secrets en clair pour l'environnement `dev` peut être toléré. Cette exception doit être documentée et une tâche doit être créée dans le backlog pour la résorber avant de configurer l'environnement `staging`.